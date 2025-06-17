package halo

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fiwippi/halo/internal/api"
	"github.com/fiwippi/halo/internal/stringutil"
)

type ctxKey uint

const (
	tagsFilterCtxKey ctxKey = iota
)

func getFilters(r *http.Request) []string {
	return r.Context().Value(tagsFilterCtxKey).([]string)
}

const tagsFilterCookie = "tags-filter"

func handleTagFilters(s *store) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var (
				tags []string
				err  error
			)

			if r.URL.Path == "/" {
				// We only set filters if we're on the index route since
				// they aren't specified for other routes, meaning they
				// get overriden to be empty
				tags = r.URL.Query()["tag"]
				tags = stringutil.Deduplicate(tags...)
				tags = stringutil.Filter(tags, func(tag string) bool {
					return s.HasTag(tag)
				})
				if err := api.StoreCookieList(w, tagsFilterCookie, tags); err != nil {
					api.Error(w, fmt.Errorf("store tag filters: %w", err))
					return
				}
			} else {
				tags, err = api.LoadCookieList(r, tagsFilterCookie)
				if err != nil {
					api.Error(w, fmt.Errorf("load tag filters: %w", err))
					return
				}
			}

			// We want to avoid failed assertions even if no tags exist
			if tags == nil {
				tags = make([]string, 0)
			}
			ctx := context.WithValue(r.Context(), tagsFilterCtxKey, tags)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
