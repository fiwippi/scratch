package stringutil

import (
	"cmp"
	"slices"
	"strings"
)

func Deduplicate(s ...string) []string {
	slices.SortFunc(s, func(a, b string) int {
		// Same letter      => case-sensitive
		// Different letter => case-insensitive
		c, d := strings.ToLower(a), strings.ToLower(b)
		sameLetter := c == d
		if sameLetter {
			return cmp.Compare(a, b) * -1 // We want lowercase before uppercase
		} else {
			return cmp.Compare(c, d)
		}
	})
	return slices.Compact(s)
}

// Filter filters a slice of strings in-place.
func Filter(s []string, keep func(string) bool) []string {
	i := 0
	for _, str := range s {
		if keep(str) {
			s[i] = str
			i += 1
		}
	}
	return s[:i]
}
