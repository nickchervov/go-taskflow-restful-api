package server

import (
	"net/http"
	"restful-taskflow/internal/store/cache"
	"restful-taskflow/pkg/httputil"
	"strconv"
)

func MiddlewareRateLimiting(cache *cache.Cache, limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isAllow, ttl := cache.CheckLimit(r.Context(), r.RemoteAddr, limit)
			if !isAllow {
				w.Header().Set("Retry-After", strconv.Itoa(int(ttl.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				httputil.JSONResponse(w, http.StatusTooManyRequests, map[string]string{
					"message":     "rate_limited_exceeded",
					"retry-after": ttl.String(),
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
