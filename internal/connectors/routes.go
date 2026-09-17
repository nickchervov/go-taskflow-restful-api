package connectors

import (
	"fmt"
	"net/http"
	"restful-taskflow/internal/usecase"
	"strconv"
)

func SetRoutes(svc *usecase.TaskflowService, limit string) (http.Handler, error) {
	l, err := strconv.Atoi(limit)
	if err != nil {
		return nil, fmt.Errorf("wrong limit: %w", err)
	}
	h := NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/tasks", h.CreateTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}", h.GetStatusOrResult)
	mux.HandleFunc("GET /api/v1/tasks", h.TasksList)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", h.CancelTask)
	mux.HandleFunc("POST /api/v1/tasks/{id}/retry", h.ReloadTask)

	wrappedMux := MiddlewareRateLimiting(h.svc.Cache, l)(mux)

	return wrappedMux, nil
}
