package spot

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"spot/internal/browserutil"
	"spot/internal/stringutil"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

type Config struct {
	RedirectURL   string `toml:"redirect_url"`
	SpotifyID     string `toml:"spotify_id"`
	SpotifySecret string `toml:"spotify_secret"`
}

func LoadConfig(path string) (Config, error) {
	var conf Config
	if _, err := toml.DecodeFile(path, &conf); err != nil {
		return Config{}, err
	}

	isLocalhost := strings.Contains(conf.RedirectURL, "localhost")
	is127 := strings.Contains(conf.RedirectURL, "127.0.0.1")
	if !isLocalhost && !is127 {
		return Config{}, fmt.Errorf("redirect_url must be to localhost")
	}
	if conf.SpotifyID == "" {
		return Config{}, fmt.Errorf("spotify_id is nil")
	}
	if conf.SpotifySecret == "" {
		return Config{}, fmt.Errorf("spotify_secret is nil")
	}

	return conf, nil
}

func NewClient(conf Config, scopes ...string) (*spotify.Client, error) {
	url, err := url.Parse(conf.RedirectURL)
	if err != nil {
		return nil, fmt.Errorf("parse auth url: %w", err)
	}
	port := url.Port()
	if port == "" {
		return nil, fmt.Errorf("port not specified in auth url")
	}

	auth := spotifyauth.New(
		spotifyauth.WithClientID(conf.SpotifyID),
		spotifyauth.WithClientSecret(conf.SpotifySecret),
		spotifyauth.WithRedirectURL(conf.RedirectURL),
		spotifyauth.WithScopes(scopes...),
	)
	state := stringutil.RandString(15)

	client := make(chan *spotify.Client)
	error := make(chan error)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Token(r.Context(), state, r)
		if err != nil {
			http.Error(w, "Couldn't get token", http.StatusBadRequest)
			error <- fmt.Errorf("parse token: %w", err)
			return
		}
		client <- spotify.New(auth.Client(r.Context(), token), spotify.WithRetry(true))
	})
	l, err := net.Listen("tcp", net.JoinHostPort("localhost", port))
	if err != nil {
		return nil, fmt.Errorf("listener: %w", err)
	}
	srv := &http.Server{Handler: mux}
	defer srv.Close()
	go srv.Serve(l)

	if err := browserutil.Open(auth.AuthURL(state)); err != nil {
		return nil, fmt.Errorf("open browser: %w", err)
	}

	select {
	case c := <-client:
		return c, nil
	case err := <-error:
		return nil, err
	}
}
