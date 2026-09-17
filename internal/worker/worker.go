package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"restful-taskflow/internal/domain"
	"restful-taskflow/internal/worker/tasks"
	"time"
)

type taskFunc func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error)

var handlers = map[string]taskFunc{
	"send_email":      tasks.SendEmail,
	"resize_image":    tasks.ResizeImage,
	"generate_report": tasks.GenerateReport,
}

func (w *WorkerPool) fail(task domain.Task, err error) {
	task.CompletedAt = time.Now()
	task.Error = err.Error()
	task.RetryCount++

	if task.RetryCount <= task.MaxRetries {
		task.Status = "pending"
		slog.Warn("requeue task", "task_id", task.ID, "retry", task.RetryCount)
		if err := w.Repo.UpdateTask(w.ctx, task.ID, task); err != nil {
			slog.Error("retry update task", "task_id", task.ID, "error", err)
			return
		}
		if err := w.Cache.AddTaskToQueue(w.ctx, task); err != nil {
			slog.Error("retry add task to queue", "task_id", task.ID, "error", err)
			return
		}
		return
	}

	task.Status = "failed"
	slog.Error("task failed permanently", "task_id", task.ID, "error", err)

	statusResult := domain.TaskStatusResult{
		ID:     task.ID,
		Status: task.Status,
	}
	if err := w.Repo.UpdateTask(w.ctx, task.ID, task); err != nil {
		slog.Error("updating task in db", "task_id", task.ID, "error", err)
		return
	}
	if err := w.Cache.CacheTaskStatusResult(w.ctx, statusResult); err != nil {
		slog.Error("add to cache task status", "task_id", task.ID, "error", err)
		return
	}
	w.Cache.IncrCounterProcessedTasks(w.ctx)
}

func (w *WorkerPool) Worker() {
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
		}

		task, err := w.Cache.BRPopTask(w.ctx)
		if errors.Is(err, domain.ErrNoTask) {
			continue
		}
		if err != nil {
			if w.ctx.Err() != nil {
				return
			}
			slog.Error("brpop task", "error", err)
			select {
			case <-w.ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		w.handle(task)
	}
}

func (w *WorkerPool) handle(task domain.Task) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in task", "task_id", task.ID, "panic", r)
			w.fail(task, fmt.Errorf("panic: %v", r))
		}
	}()
	slog.Info("worker picked up task", "task_id", task.ID, "task_type", task.Type)

	taskfunc, ok := handlers[task.Type]
	if !ok {
		slog.Error("unknown task type", "task_id", task.ID, "task_type", task.Type)
		w.fail(task, fmt.Errorf("unknown task type %q", task.Type))
		return
	}

	task.Status = "running"
	task.StartedAt = time.Now()

	ok, err := w.Repo.MarkRunning(w.ctx, task.ID)
	if err != nil {
		w.fail(task, fmt.Errorf("mark running %q: %w", task.ID, err))
		return
	}
	if !ok {
		slog.Warn("task no longer pending, skip", "task_id", task.ID)
		return
	}
	slog.Info("task status is running", "id", task.ID)

	statusResult := domain.TaskStatusResult{
		ID:     task.ID,
		Status: task.Status,
	}
	if err := w.Cache.CacheTaskStatusResult(w.ctx, statusResult); err != nil {
		slog.Error("set to cache task status and result", "error", err)
		w.fail(task, fmt.Errorf("set to cache task %q status and result: %w", task.ID, err))
		return
	}

	result, err := taskfunc(w.ctx, task.Payload)
	if err != nil {
		w.fail(task, err)
		return
	}

	task.Status = "completed"
	task.CompletedAt = time.Now()
	task.Result = result
	task.Error = ""

	if err := w.Repo.UpdateTask(w.ctx, task.ID, task); err != nil {
		slog.Error("updating task", "task_id", task.ID, "error", err)
		w.fail(task, fmt.Errorf("updating %q: %w", task.ID, err))
		return
	}

	statusResult.Status = task.Status
	statusResult.Result = task.Result
	if err := w.Cache.CacheTaskStatusResult(w.ctx, statusResult); err != nil {
		slog.Error("set to cache task status and result", "task_id", task.ID, "error", err)
		w.fail(task, fmt.Errorf("set to cache task %q status and result: %w", task.ID, err))
		return
	}
	w.Cache.IncrCounterProcessedTasks(w.ctx)
	slog.Info("worker complete task", "task_id", task.ID, "task_status", task.Status)
}
