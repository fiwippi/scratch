package halo

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/fiwippi/halo/internal/api"
)

type Config struct {
	Host        string           `toml:"host"`
	Port        uint16           `toml:"port"`
	DataDir     string           `toml:"data_dir"`
	Credentials []api.Credential `toml:"credentials"`
}

type Server struct {
	config Config
	store  *store
	mux    http.Handler
}

func NewServer(conf Config) (*Server, error) {
	if !strings.HasSuffix(conf.DataDir, "/") {
		conf.DataDir += "/"
	}
	_, err := os.Stat(conf.DataDir)
	if err != nil {
		return nil, fmt.Errorf("data dir does not exist: %w", err)
	}
	store, err := newStore(conf.DataDir + "store.db")
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", newStaticController(store))
	mux.Handle("/frag/", http.StripPrefix("/frag", newFragmentController(store)))

	return &Server{
		config: conf,
		store:  store,
		mux: api.ChainMiddleware(
			mux,
			handleTagFilters(store),
			api.BasicAuth(conf.Credentials, "Halo"),
			api.HttpLogger(),
		),
	}, nil
}

func (s *Server) ListenAndServe() error {
	defer s.store.Close()

	address := net.JoinHostPort(s.config.Host, fmt.Sprint(s.config.Port))
	slog.Info("Server listening", slog.String("address", address))
	return http.ListenAndServe(address, s.mux)
}
