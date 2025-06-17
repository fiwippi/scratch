package halo

import (
	"image"
	"strings"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

var dummyImage image.Image = image.NewRGBA(image.Rect(0, 0, 10, 10)) // Black square

func TestGetImageIDs(t *testing.T) {
	s, err := newStore(inMemory)
	require.NoError(t, err)

	const (
		Cat      = "cat"
		Sloth    = "sloth"
		Platypus = "platypus"
	)

	require.NoError(t, s.AddTag(Cat))
	require.NoError(t, s.AddTag(Sloth))
	require.NoError(t, s.AddTag(Platypus))

	ids, err := s.AddImages([]image.Image{dummyImage, dummyImage, dummyImage}, "")
	require.NoError(t, err)
	id1 := ids[0]
	id2 := ids[1]
	id3 := ids[2]

	require.NoError(t, s.AddTagToImage(Cat, id1))
	require.NoError(t, s.AddTagToImage(Sloth, id1))
	require.NoError(t, s.AddTagToImage(Platypus, id1))
	require.NoError(t, s.AddTagToImage(Cat, id2))
	require.NoError(t, s.AddTagToImage(Platypus, id2))
	require.NoError(t, s.AddTagToImage(Sloth, id3))

	for _, subtest := range []struct {
		tags        []string
		expectedIDs []ulid.ULID
	}{
		{[]string{Cat}, []ulid.ULID{id1, id2}},
		{[]string{Sloth}, []ulid.ULID{id1, id3}},
		{[]string{Platypus}, []ulid.ULID{id1, id2}},
		{[]string{Cat, Sloth}, []ulid.ULID{id1}},
		{[]string{Cat, Platypus}, []ulid.ULID{id1, id2}},
		{[]string{Sloth, Platypus}, []ulid.ULID{id1}},
		{[]string{Cat, Sloth, Platypus}, []ulid.ULID{id1}},
		// If the tag doesn't exist, nothing should be returned,
		// unless no tags are suppled in which case everything
		// is returned
		{[]string{}, []ulid.ULID{id1, id2, id3}},
		{[]string{"N/A"}, []ulid.ULID(nil)},
	} {
		t.Run(strings.Join(subtest.tags, ","), func(t *testing.T) {
			ids, err := s.GetImageIDs(subtest.tags...)
			require.NoError(t, err)
			require.Equal(t, subtest.expectedIDs, ids)
		})
	}
}
