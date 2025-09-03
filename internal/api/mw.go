package api

import (
	"cmder/internal/config"
	"net/http"
)

// Key 中间件：校验 X-Security-Key
func Key(provider config.KeyProvider, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Security-Key")
		if key != provider.GetXSecurityKey() {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// IpCheck 验证i白名单
func IpCheck(provider config.WhiteListProvider, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !iPInWhiteList(provider, r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}
