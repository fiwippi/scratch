package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	spotifyauth "github.com/zmb3/spotify/v2/auth"

	"spot"
)

func run(configPath string, category, subcommand string) error {
	conf, err := spot.LoadConfig(configPath)
	if err != nil {
		return err
	}
	client, err := spot.NewClient(conf,
		spotifyauth.ScopeUserFollowRead,
		spotifyauth.ScopeUserLibraryRead,
	)
	if err != nil {
		return err
	}

	var (
		v   any
		ctx = context.Background()
	)
	switch category {
	case "tracks":
		switch subcommand {
		case "saved":
			tracks, err := spot.GetSavedTracks(ctx, client)
			if err != nil {
				return err
			}
			v = tracks
		default:
			return fmt.Errorf("unsupported subcommand: %s", subcommand)
		}
	case "artists":
		switch subcommand {
		case "followed":
			artists, err := spot.GetFollowedArtists(ctx, client)
			if err != nil {
				return err
			}
			v = artists
		case "unfollowed":
			artists, err := spot.GetUnfollowedArtists(ctx, client)
			if err != nil {
				return err
			}
			v = artists
		default:
			return fmt.Errorf("unsupported subcommand: %s", subcommand)
		}
	default:
		return fmt.Errorf("unsupported category: %s", category)
	}
	return json.NewEncoder(os.Stdout).Encode(v)
}

func main() {
	configPath := flag.String("config", "config.toml", "Path to config file")
	flag.Parse()

	if err := run(*configPath, flag.Arg(0), flag.Arg(1)); err != nil {
		fmt.Println("Failed:", err)
		os.Exit(1)
	}
}
