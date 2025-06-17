package spot

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/zmb3/spotify/v2"
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
	tracks := make([]spotify.SavedTrack, 0)
	offset := 0
	for {
		t, err := c.CurrentUsersTracks(ctx, spotify.Limit(50), spotify.Offset(offset))
		if err != nil {
			return nil, fmt.Errorf("get user tracks: %w", err)
		}
		if len(t.Tracks) == 0 {
			return tracks, nil
		}
		tracks = append(tracks, t.Tracks...)
		offset += len(t.Tracks)
	}
}
