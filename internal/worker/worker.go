package worker

import (
	"log"
	"log/slog"
	"restful-taskflow/internal/model"
	"restful-taskflow/internal/worker/tasks"
	"time"
)

func (w *WorkerPool) Worker() {
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			task, err := w.Repo.Cache.BRPopTask(w.ctx)
			if err != nil {
				log.Println(err)
				continue
			}
			slog.Info("worker picked up task", "task_id", task.ID, "task_type", task.Type)
			task.Status = "running"
			if err := w.Repo.UpdateStatusTask(w.ctx, task.ID, task.Status); err != nil {
				slog.Error("updating status task", "error", err)
			}
			statusResult := model.TaskStatusResult{
				ID:     task.ID,
				Status: task.Status,
			}
			w.Repo.Cache.CacheTaskStatusResult(w.ctx, statusResult)
			task.StartedAt = time.Now()
			switch task.Type {
			case "send_email":
				result, err := tasks.SendEmail(w.ctx, task.Payload)
				if err != nil {
					task.Status = "failed"
					task.CompletedAt = time.Now()
					task.Error = err.Error()
					if task.RetryCount < task.MaxRetries {
						result, err = tasks.SendEmail(w.ctx, task.Payload)
						if err != nil {
							statusResult.Status = "failed"
							w.Repo.Cache.CacheTaskStatusResult(w.ctx, statusResult)
							w.Repo.UpdateTask(w.ctx, task.ID, task)
							continue
						}
						task.RetryCount++
					} else {
						statusResult.Status = task.Status
						w.Repo.Cache.CacheTaskStatusResult(w.ctx, statusResult)
						w.Repo.UpdateTask(w.ctx, task.ID, task)
						continue
					}
				}
				task.Status = "completed"
				task.CompletedAt = time.Now()
				task.Result = result

				w.Repo.UpdateTask(w.ctx, task.ID, task)

				statusResult.Status = task.Status
				statusResult.Result = task.Result
				w.Repo.Cache.CacheTaskStatusResult(w.ctx, statusResult)
				w.Repo.Cache.IncrCounterProcessedTasks(w.ctx)
			case "resize_image":
				result, err := tasks.ResizeImage(w.ctx, task.Payload)
				if err != nil {
					task.Status = "failed"
					task.CompletedAt = time.Now()
					task.Error = err.Error()
					if task.RetryCount < task.MaxRetries {
						result, err = tasks.ResizeImage(w.ctx, task.Payload)
						if err != nil {
							statusResult.Status = "failed"
							w.Repo.Cache.CacheTaskStatusResult(w.ctx, statusResult)
							w.Repo.UpdateTask(w.ctx, task.ID, task)
							continue
						}
						task.RetryCount++
					} else {
						statusResult.Status = task.Status
						w.Repo.Cache.CacheTaskStatusResult(w.ctx, statusResult)
						w.Repo.UpdateTask(w.ctx, task.ID, task)
						continue
					}
				}
				task.Status = "completed"
				task.CompletedAt = time.Now()
				task.Result = result

				w.Repo.UpdateTask(w.ctx, task.ID, task)

				statusResult.Status = task.Status
				statusResult.Result = task.Result
				w.Repo.Cache.CacheTaskStatusResult(w.ctx, statusResult)
				w.Repo.Cache.IncrCounterProcessedTasks(w.ctx)
			case "generate_report":
				result, err := tasks.GenerateReport(w.ctx, task.Payload)
				if err != nil {
					task.Status = "failed"
					task.CompletedAt = time.Now()
					task.Error = err.Error()
					if task.RetryCount < task.MaxRetries {
						result, err = tasks.GenerateReport(w.ctx, task.Payload)
						if err != nil {
							statusResult.Status = "failed"
							w.Repo.Cache.CacheTaskStatusResult(w.ctx, statusResult)
							w.Repo.UpdateTask(w.ctx, task.ID, task)
							continue
						}
						task.RetryCount++
					} else {
						statusResult.Status = task.Status
						w.Repo.Cache.CacheTaskStatusResult(w.ctx, statusResult)
						w.Repo.UpdateTask(w.ctx, task.ID, task)
						continue
					}
				}
				task.Status = "completed"
				task.CompletedAt = time.Now()
				task.Result = result

				w.Repo.UpdateTask(w.ctx, task.ID, task)

				statusResult.Status = task.Status
				statusResult.Result = task.Result
				w.Repo.Cache.CacheTaskStatusResult(w.ctx, statusResult)
				w.Repo.Cache.IncrCounterProcessedTasks(w.ctx)
			}
		}
	}
}
