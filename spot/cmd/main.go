package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"github.com/urfave/cli/v3"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"

	"spot"
)

func loadClient(c *cli.Command) (*spotify.Client, error) {
	conf, err := spot.LoadConfig(c.String("config"))
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	client, err := spot.NewClient(conf,
		spotifyauth.ScopeUserFollowRead,
		spotifyauth.ScopeUserLibraryRead,
		spotifyauth.ScopePlaylistModifyPublic,
		spotifyauth.ScopePlaylistModifyPrivate,
	)
	if err != nil {
		return nil, fmt.Errorf("new client: %w", err)
	}
	return client, nil
}

func main() {
	cmd := &cli.Command{
		Name:  "spot",
		Usage: "A Spotify CLI utility",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Value: "config.toml",
				Usage: "Path to config file",
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "tracks",
				Usage: "View library track information",
				Action: func(ctx context.Context, c *cli.Command) error {
					client, err := loadClient(c)
					if err != nil {
						return err
					}
					tracks, err := spot.GetSavedTracks(ctx, client)
					if err != nil {
						return err
					}
					return json.NewEncoder(os.Stdout).Encode(tracks)
				},
			},
			{
				Name:  "artists",
				Usage: "View library artist information",
				Commands: []*cli.Command{
					{
						Name:  "followed",
						Usage: "View followed artists",
						Action: func(ctx context.Context, c *cli.Command) error {
							client, err := loadClient(c)
							if err != nil {
								return err
							}
							artists, err := spot.GetFollowedArtists(ctx, client)
							if err != nil {
								return err
							}
							return json.NewEncoder(os.Stdout).Encode(artists)
						},
					},
					{
						Name:  "unfollowed",
						Usage: "View unfollowed artists",
						Action: func(ctx context.Context, c *cli.Command) error {
							client, err := loadClient(c)
							if err != nil {
								return err
							}
							artists, err := spot.GetUnfollowedArtists(ctx, client)
							if err != nil {
								return err
							}
							return json.NewEncoder(os.Stdout).Encode(artists)
						},
					},
				},
			},
			{
				Name:  "shuffle",
				Usage: "Create a playlist of your shuffled library",
				Action: func(ctx context.Context, c *cli.Command) error {
					client, err := loadClient(c)
					if err != nil {
						return err
					}
					tracks, err := spot.GetSavedTracks(ctx, client)
					if err != nil {
						return err
					}
					trackIDs := make([]spotify.ID, len(tracks))
					for i := range tracks {
						trackIDs[i] = tracks[i].ID
					}
					// Fisher–Yates shuffle
					for i := range trackIDs {
						j := rand.Intn(i + 1)
						trackIDs[i], trackIDs[j] = trackIDs[j], trackIDs[i]
					}
					plID, err := spot.CreateNewPlaylist(ctx, client, trackIDs)
					if err != nil {
						return err
					}
					fmt.Println("Playlist created with ID", plID)
					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run: %s", err)
		os.Exit(1)
	}
}
