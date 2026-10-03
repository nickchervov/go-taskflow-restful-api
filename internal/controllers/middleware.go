package controllers

import (
	"net/http"
	"restful-taskflow/internal/usecase"
	"strconv"
)

func MiddlewareRateLimiting(cache usecase.Cache, limit int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isAllow, ttl := cache.CheckLimit(r.Context(), r.RemoteAddr, limit)
			if !isAllow {
				w.Header().Set("Retry-After", strconv.Itoa(int(ttl.Seconds())))
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(strconv.Itoa(int(ttl.Seconds()))))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
