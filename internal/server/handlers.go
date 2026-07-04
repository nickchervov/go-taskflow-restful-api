package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"restful-taskflow/internal/model"
	"restful-taskflow/internal/store/postgres"
	"restful-taskflow/pkg/httputil"
	"time"

	"github.com/google/uuid"
)

type Handler struct {
	repo *postgres.Repository
}

func NewHandler(repo *postgres.Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) SetRoutes(limit int) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/tasks", h.CreateTask)
	mux.HandleFunc("GET /api/v1/tasks/{id}", h.GetStatusOrResult)
	mux.HandleFunc("GET /api/v1/tasks", h.TasksList)
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", h.CancelTask)
	mux.HandleFunc("POST /api/v1/tasks/{id}/retry", h.ReloadTask)

	wrappedMux := MiddlewareRateLimiting(h.repo.Cache, limit)(mux)

	return wrappedMux
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	slog.Info("incoming CreateTask request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
	var task model.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		slog.Error("decoding request body", "error", err)
		httputil.JSONResponse(w, http.StatusBadRequest, map[string]string{"message": "decoding request body: " + err.Error()})
		return
	}
	if task.Type != "send_email" && task.Type != "resize_image" && task.Type != "generate_report" {
		slog.Error("invalid task type", "task type", task.Type)
		httputil.JSONResponse(w, http.StatusBadRequest, map[string]string{"message": "invalid task type"})
		return
	}
	payload, err := json.Marshal(task.Payload)
	if err != nil {
		slog.Error("encoding payload to json", "error", err)
		httputil.JSONResponse(w, http.StatusBadRequest, map[string]string{"message": "encoding payload to json: " + err.Error()})
		return
	}
	task.Payload = payload
	task.Status = "pending"
	task.CreatedAt = time.Now()

	id, err := h.repo.AddTask(r.Context(), task)
	if err != nil {
		slog.Error("adding task", "error", err)
		httputil.JSONResponse(w, http.StatusInternalServerError, map[string]string{"message": "adding task: " + err.Error()})
		return
	}

	task.ID = id
	err = h.repo.Cache.AddTaskToQueue(r.Context(), task)
	if err != nil {
		slog.Error("adding task to queue", "error", err)
		httputil.JSONResponse(w, http.StatusInternalServerError, map[string]string{"message": "adding task to queue: " + err.Error()})
		return
	}

	slog.Info("task created", "task_id", id)
	httputil.JSONResponse(w, http.StatusCreated, map[string]string{"message": "task created"})
}
func (h *Handler) GetStatusOrResult(w http.ResponseWriter, r *http.Request) {
	slog.Info("incoming GetStatusOrResult request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		slog.Error("invalid id", "error", err)
		httputil.JSONResponse(w, http.StatusBadRequest, map[string]string{"message": "invalid id: " + err.Error()})
		return
	}

	statusResult, err := h.repo.Cache.GetTaskStatusResultFromCache(r.Context(), id)
	if err == nil {
		slog.Info("OK get status or result from cache", "status", statusResult.Status, "result", statusResult.Result)
		httputil.JSONResponse(w, http.StatusOK, statusResult)
		return
	}

	taskStatusResult, err := h.repo.GetTaskStatusResult(r.Context(), id)
	if err != nil {
		slog.Error("getting task status and result", "error", err)
		httputil.JSONResponse(w, http.StatusInternalServerError, map[string]string{"message": "getting task status and result: " + err.Error()})
		return
	}
	slog.Info("OK get status or result from db", "status", taskStatusResult.Status, "result", taskStatusResult.Result)
	httputil.JSONResponse(w, http.StatusOK, taskStatusResult)
}
func (h *Handler) TasksList(w http.ResponseWriter, r *http.Request) {
	slog.Info("incoming TasksList request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
	tasks, err := h.repo.GetAllTasks(r.Context())
	if err != nil {
		slog.Error("getting tasks list", "error", err)
		httputil.JSONResponse(w, http.StatusInternalServerError, map[string]string{"message": "getting tasks list: " + err.Error()})
		return
	}
	slog.Info("OK get tasks list", "len", len(tasks))
	httputil.JSONResponse(w, http.StatusOK, tasks)
}
func (h *Handler) CancelTask(w http.ResponseWriter, r *http.Request) {
	slog.Info("incoming CancelTask request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		slog.Error("invalid id", "error", err)
		httputil.JSONResponse(w, http.StatusBadRequest, map[string]string{"message": "invalid id: " + err.Error()})
		return
	}
	if err := h.repo.CancelTask(r.Context(), id); err != nil {
		slog.Error("cancelling task", "error", err)
		httputil.JSONResponse(w, http.StatusInternalServerError, map[string]string{"message": "cancelling task: " + err.Error()})
		return
	}
	slog.Info("task cancelled", "id", id)
	httputil.JSONResponse(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("task %s cancelled", id)})
}
func (h *Handler) ReloadTask(w http.ResponseWriter, r *http.Request) {
	slog.Info("incoming ReloadTask request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		slog.Error("invalid id", "error", err)
		httputil.JSONResponse(w, http.StatusBadRequest, map[string]string{"message": "invalid id: " + err.Error()})
		return
	}
	if err := h.repo.ReloadTask(r.Context(), id); err != nil {
		slog.Error("reloading task", "error", err)
		httputil.JSONResponse(w, http.StatusInternalServerError, map[string]string{"message": "reloading task: " + err.Error()})
		return
	}
	slog.Info("task reloaded", "id", id)
	httputil.JSONResponse(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("task %s reloaded", id)})
}
