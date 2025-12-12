package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

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
	log.SetFlags(0)

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
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "playlist",
						Value: "",
						Usage: "Create playlist with tracks with specified name",
					},
					&cli.BoolFlag{
						Name:  "shuffle",
						Value: false,
						Usage: "Shuffle tracks",
					},
					&cli.IntFlag{
						Name:  "year-added",
						Value: -1,
						Usage: "Filter tracks by year released",
					},
					&cli.IntFlag{
						Name:  "year-released",
						Value: -1,
						Usage: "Filter tracks by year saved to library",
					},
				},
				Action: func(ctx context.Context, c *cli.Command) error {
					client, err := loadClient(c)
					if err != nil {
						return err
					}
					tracks, err := spot.GetSavedTracks(ctx, client)
					if err != nil {
						return err
					}

					if year := c.Int("year-added"); year != -1 {
						i := 0
						for _, track := range tracks {
							timeAdded, err := time.Parse(spotify.TimestampLayout, track.AddedAt)
							if err != nil {
								return err
							}
							if timeAdded.Year() == year {
								tracks[i] = track
								i += 1
							}
						}
						tracks = tracks[:i]
					}
					if year := c.Int("year-released"); year != -1 {
						i := 0
						for _, track := range tracks {
							if track.Album.ReleaseDateTime().Year() == year {
								tracks[i] = track
								i += 1
							}
						}
						tracks = tracks[:i]
					}
					if c.Bool("shuffle") {
						// Fisher–Yates shuffle
						for i := range tracks {
							j := rand.Intn(i + 1)
							tracks[i], tracks[j] = tracks[j], tracks[i]
						}
					}
					if name := c.String("playlist"); name != "" {
						trackIDs := make([]spotify.ID, len(tracks))
						for i := range tracks {
							trackIDs[i] = tracks[i].ID
						}
						plID, err := spot.CreateNewPlaylist(ctx, name, client, trackIDs)
						if err != nil {
							return err
						}
						log.Println("playlist:", name, plID)
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
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run: %s", err)
		os.Exit(1)
	}
}
