package server

import "net/http"

func NewServer(router http.Handler) *http.Server {
	return &http.Server{
		Addr:    ":8080",
		Handler: router,
	}
}
