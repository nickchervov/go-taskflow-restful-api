package connectors

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"restful-taskflow/internal/domain"
	"restful-taskflow/internal/dto"
	"restful-taskflow/internal/usecase"
	"restful-taskflow/pkg/httputil"

	"github.com/google/uuid"
)

type Handler struct {
	svc *usecase.TaskflowService
}

func NewHandler(svc *usecase.TaskflowService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	slog.Info("incoming CreateTask request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
	var input dto.CreateTaskInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		slog.Error("decoding request body", "error", err)
		httputil.JSONResponse(w, http.StatusBadRequest, map[string]string{"message": "decoding request body: " + err.Error()})
		return
	}
	output, err := h.svc.CreateTask(r.Context(), input)
	if err != nil {
		slog.Error("create task svc", "error", err)
		var targetErr *domain.TaskError
		if errors.As(err, &targetErr) {
			httputil.JSONResponse(w, targetErr.Code, targetErr)
			return
		}
		httputil.JSONResponse(w, http.StatusInternalServerError, map[string]string{"message": "create task internal error"})
		return
	}
	slog.Info("task created", "task_id", output.Id)
	httputil.JSONResponse(w, http.StatusCreated, output)
}
func (h *Handler) GetStatusOrResult(w http.ResponseWriter, r *http.Request) {
	slog.Info("incoming GetStatusOrResult request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		slog.Error("invalid id", "error", err)
		httputil.JSONResponse(w, http.StatusBadRequest, map[string]string{"message": "invalid id: " + err.Error()})
		return
	}

	input := dto.GetStatusResultInput{Id: id}
	output, err := h.svc.GetStatusOrResult(r.Context(), input)
	if err != nil {
		slog.Error("getting task status and result svc", "error", err)
		var targetErr *domain.TaskError
		if errors.As(err, &targetErr) {
			httputil.JSONResponse(w, targetErr.Code, targetErr)
			return
		}
		httputil.JSONResponse(w, http.StatusInternalServerError, map[string]string{"message": "getting status and result internal error"})
		return
	}

	slog.Info("OK get status or result from db", "status", output.Status, "result", output.Result)
	httputil.JSONResponse(w, http.StatusOK, output)
}
func (h *Handler) TasksList(w http.ResponseWriter, r *http.Request) {
	slog.Info("incoming TasksList request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
	output, err := h.svc.TasksList(r.Context())
	if err != nil {
		slog.Error("getting tasks list", "error", err)
		httputil.JSONResponse(w, http.StatusInternalServerError, map[string]string{"message": "getting tasks list internal error"})
		return
	}
	slog.Info("OK get tasks list", "len", len(output.Tasks))
	httputil.JSONResponse(w, http.StatusOK, output)
}
func (h *Handler) CancelTask(w http.ResponseWriter, r *http.Request) {
	slog.Info("incoming CancelTask request", "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		slog.Error("invalid id", "error", err)
		httputil.JSONResponse(w, http.StatusBadRequest, map[string]string{"message": "invalid id: " + err.Error()})
		return
	}
	input := dto.CancelTaskInput{Id: id}
	if err := h.svc.CancelTask(r.Context(), input); err != nil {
		slog.Error("cancelling task", "error", err)
		var targetErr *domain.TaskError
		if errors.As(err, &targetErr) {
			httputil.JSONResponse(w, targetErr.Code, targetErr)
			return
		}
		httputil.JSONResponse(w, http.StatusInternalServerError, map[string]string{"message": "cancelling task internal error"})
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
	input := dto.ReloadTaskInput{Id: id}
	if err := h.svc.ReloadTask(r.Context(), input); err != nil {
		slog.Error("reload task", "error", err)
		var targetErr *domain.TaskError
		if errors.As(err, &targetErr) {
			httputil.JSONResponse(w, targetErr.Code, targetErr)
			return
		}
		httputil.JSONResponse(w, http.StatusInternalServerError, map[string]string{"message": "reload task internal error"})
		return
	}
	slog.Info("task reloaded", "id", id)
	httputil.JSONResponse(w, http.StatusOK, map[string]string{"message": fmt.Sprintf("task %s reloaded", id)})
}
