package spot

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"math"
	"slices"
	"sync/atomic"

	"github.com/zmb3/spotify/v2"
	"golang.org/x/sync/errgroup"
)

func GetFollowedArtists(ctx context.Context, c *spotify.Client) ([]spotify.FullArtist, error) {
	artists := make([]spotify.FullArtist, 0)
	after := ""
	for {
		a, err := c.CurrentUsersFollowedArtists(ctx, spotify.After(after))
		if err != nil {
			return nil, fmt.Errorf("get followed artists: %w", err)
		}
		if len(a.Artists) == 0 {
			slices.SortFunc(artists, func(a, b spotify.FullArtist) int {
				return cmp.Compare(a.Name, b.Name)
			})
			return artists, nil
		}
		artists = append(artists, a.Artists...)
		after = artists[len(artists)-1].ID.String()
	}
}

func GetUnfollowedArtists(ctx context.Context, c *spotify.Client) ([]spotify.SimpleArtist, error) {
	artists, err := GetFollowedArtists(ctx, c)
	if err != nil {
		return nil, err
	}
	artistsIDs := make(map[spotify.ID]struct{}, len(artists))
	for _, a := range artists {
		artistsIDs[a.ID] = struct{}{}
	}
	tracks, err := GetSavedTracks(ctx, c)
	if err != nil {
		return nil, err
	}

	unfollowedArtists := make([]spotify.SimpleArtist, 0)
	for _, track := range tracks {
		for _, artist := range track.Artists {
			if _, found := artistsIDs[artist.ID]; !found {
				artistsIDs[artist.ID] = struct{}{}
				unfollowedArtists = append(unfollowedArtists, artist)
			}
		}
	}
	slices.SortStableFunc(unfollowedArtists, func(a, b spotify.SimpleArtist) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return unfollowedArtists, nil
}

func GetSavedTracks(ctx context.Context, c *spotify.Client) ([]spotify.SavedTrack, error) {
	const pageSize = 50
	trackInfo, err := c.CurrentUsersTracks(ctx, spotify.Limit(pageSize), spotify.Offset(0))
	if err != nil {
		return nil, fmt.Errorf("current user tracks: %w", err)
	}

	// No need to request tracks in parallel,
	// if there are no more tracks to request
	if trackInfo.Next == "" {
		log.Println("downloading: 1/1")
		return trackInfo.Tracks, nil
	}

	total := int(trackInfo.Total)
	pages := int(math.Ceil(float64(total) / float64(pageSize)))
	tracks := make([][]spotify.SavedTrack, pages)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(8)

	dlCounter := atomic.Int32{}
	dlCounter.Store(pageSize)
	log.Printf("downloading: %d/%d\n", dlCounter.Load(), pages*pageSize)

	tracks[0] = trackInfo.Tracks
	for i := 1; i < len(tracks); i++ {
		g.Go(func() error {
			info, err := c.CurrentUsersTracks(ctx, spotify.Limit(pageSize), spotify.Offset(i*pageSize))
			if err != nil {
				return fmt.Errorf("current user tracks: %w", err)
			}
			tracks[i] = info.Tracks

			counter := dlCounter.Add(pageSize)
			log.Printf("downloading: %d/%d\n", counter, pages*pageSize)

			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return slices.Concat(tracks...), nil
}

func CreateNewPlaylist(
	ctx context.Context,
	name string,
	c *spotify.Client,
	trackIDs []spotify.ID,
) (spotify.ID, error) {
	user, err := c.CurrentUser(ctx)
	if err != nil {
		return "", fmt.Errorf("current user: %w", err)
	}
	pl, err := c.CreatePlaylistForUser(ctx, user.ID, name, "", true, false)
	if err != nil {
		return "", fmt.Errorf("create playlist: %w", err)
	}
	for chunk := range slices.Chunk(trackIDs, 100) {
		_, err = c.AddTracksToPlaylist(ctx, pl.ID, chunk...)
		if err != nil {
			return "", fmt.Errorf("add tracks: %w", err)
		}
	}
	return pl.ID, nil
}
