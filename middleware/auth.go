package middleware

import (
	"net/http"

	"github.com/spf13/viper"
)

// CheckSharedSecret makes sure that the shard secret is set locally and sent in the HTTP request
func CheckSharedSecret(next http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if viper.GetString("windlass.secret") == "" {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			if r.Header.Get("X-Auth-Token") == "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			if r.Header.Get("X-Auth-Token") != viper.GetString("windlass.secret") {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
