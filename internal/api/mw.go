package api

import (
	"cmder/internal/config"
	"net/http"
)

func Key(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Security-Key")
		if key != config.GetProxy().XSecurityKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
