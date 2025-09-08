package api

import (
	"cmder/internal/config"
	"net/http"
	"time"
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

// IpCheck 验证ip白名单
func IpCheck(provider config.WhiteListProvider, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !iPInWhiteList(provider, r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// TimeRestricted 接口时间段控制访问
func TimeRestricted(provider config.TimeRestrictedProvider, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		currentTime := now.Sub(todayStart)
		if currentTime < provider.GetAcStartTime() || currentTime > provider.GetAcEndTime() {
			http.Error(w, "服务暂时不可用", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}
