package halo

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fiwippi/halo/internal/api"
)

type ctxKey uint

const (
	tagCtxKey ctxKey = iota
)

func getTag(r *http.Request) string {
	return r.Context().Value(tagCtxKey).(string)
}

const tagCookie = "tag-cookie"

func handleTagQuery(s *store) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var (
				tag string
				err error
			)

			if r.URL.Path == "/" {
				// We only set filters if we're on the index route since
				// they aren't specified for other routes, meaning they
				// get overriden to be empty
				tags := r.URL.Query()["tag"]

				// We only support querying for one tag
				if len(tags) > 0 {
					tag = tags[0]
				}
				// Ignore invalid tags
				if !s.HasTag(tag) {
					tag = ""
				}

				if err := api.StoreCookie(w, tagCookie, tag); err != nil {
					api.Error(w, fmt.Errorf("store tag: %w", err))
					return
				}
			} else {
				tag, err = api.LoadCookie(r, tagCookie)
				if err != nil {
					api.Error(w, fmt.Errorf("load tag: %w", err))
					return
				}
			}

			ctx := context.WithValue(r.Context(), tagCtxKey, tag)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
