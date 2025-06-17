package api

import "net/http"

type Credential struct {
	User string `toml:"user"`
	Pass string `toml:"pass"`
}

func BasicAuth(credentials []Credential, realm string) func(next http.Handler) http.Handler {
	accounts := make(map[string]string, len(credentials))
	for _, c := range credentials {
		accounts[c.User] = c.Pass
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			if accounts[user] != pass {
				w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
