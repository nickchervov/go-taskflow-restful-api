package postgres

import (
	"context"
	"fmt"
	"restful-taskflow/internal/model"
	"restful-taskflow/internal/store/cache"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	Cache *cache.Cache
	db    *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool, cache *cache.Cache) *Repository {
	return &Repository{
		Cache: cache,
		db:    db,
	}
}

func generateUUID() (uuid.UUID, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate uuid: %w", err)
	}
	return id, nil
}

func (r *Repository) checkExisting(ctx context.Context, tablename string, uuid uuid.UUID) (bool, error) {
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)", tablename)
	if err := r.db.QueryRow(ctx, query, uuid).Scan(&exists); err != nil {
		return false, fmt.Errorf("check existing: %w", err)
	}
	return exists, nil
}

func (r *Repository) AddTask(ctx context.Context, task model.Task) (uuid.UUID, error) {
	id, err := generateUUID()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate task uuid: %w", err)
	}
	query := `INSERT INTO tasks(id,type,status,payload,result,error,priority,created_at,started_at,completed_at,retry_count)
	VALUES(@id, @type, @status, @payload, @result, @error, @priority, @created_at, @started_at, @completed_at, @retry_count)`
	_, err = r.db.Exec(ctx, query, pgx.NamedArgs{
		"id":           id,
		"type":         task.Type,
		"status":       task.Status,
		"payload":      task.Payload,
		"result":       task.Result,
		"error":        task.Error,
		"priority":     task.Priority,
		"created_at":   task.CreatedAt,
		"started_at":   task.StartedAt,
		"completed_at": task.CompletedAt,
		"retry_count":  task.RetryCount,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("add task: %w", err)
	}
	return id, nil
}

func (r *Repository) GetTaskStatusResult(ctx context.Context, id uuid.UUID) (model.TaskStatusResult, error) {
	exists, err := r.checkExisting(ctx, "tasks", id)
	if err != nil {
		return model.TaskStatusResult{}, fmt.Errorf("check existing row: %w", err)
	}
	if !exists {
		return model.TaskStatusResult{}, fmt.Errorf("no row with id = %s", id)
	}

	var task model.TaskStatusResult
	task.ID = id
	row := r.db.QueryRow(ctx, "SELECT status, result FROM tasks WHERE id = $1", id)
	if err := row.Scan(&task.Status, &task.Result); err != nil {
		return model.TaskStatusResult{}, fmt.Errorf("scanning task row: %w", err)
	}
	return task, nil
}
func (r *Repository) GetAllTasks(ctx context.Context) ([]model.Task, error) {
	var taskList []model.Task
	rows, err := r.db.Query(ctx, "SELECT id, type, status, payload, result, error, priority, created_at, started_at, completed_at, retry_count, max_retries FROM tasks")
	if err != nil {
		return nil, fmt.Errorf("get tasks rows: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var task model.Task
		if err := rows.Scan(
			&task.ID,
			&task.Type,
			&task.Status,
			&task.Payload,
			&task.Result,
			&task.Error,
			&task.Priority,
			&task.CreatedAt,
			&task.StartedAt,
			&task.CompletedAt,
			&task.RetryCount,
			&task.MaxRetries); err != nil {
			return nil, fmt.Errorf("scanning task rows: %w", err)
		}
		taskList = append(taskList, task)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("after scanning tasks rows: %w", err)
	}
	return taskList, nil
}

func (r *Repository) CancelTask(ctx context.Context, uuid uuid.UUID) error {
	exists, err := r.checkExisting(ctx, "tasks", uuid)
	if err != nil {
		return fmt.Errorf("checking task existing: %w", err)
	}
	if !exists {
		return fmt.Errorf("no task with id = %s", uuid)
	}

	var status string
	if err := r.db.QueryRow(ctx, "SELECT status FROM tasks WHERE id = $1", uuid).Scan(&status); err != nil {
		return fmt.Errorf("check task status before cancelling: %w", err)
	}
	if status != "pending" && status != "running" {
		return fmt.Errorf("task must be in pending or running status")
	}

	_, err = r.db.Exec(ctx, "UPDATE tasks SET status = 'cancelled' WHERE id = $1", uuid)
	if err != nil {
		return fmt.Errorf("cancelling task: %w", err)
	}
	return nil
}

func (r *Repository) UpdateTask(ctx context.Context, uuid uuid.UUID, task model.Task) error {
	exists, err := r.checkExisting(ctx, "tasks", uuid)
	if err != nil {
		return fmt.Errorf("checking task existing: %w", err)
	}
	if !exists {
		return fmt.Errorf("no task with id = %s", uuid)
	}

	_, err = r.db.Exec(ctx, "UPDATE tasks SET status = $1, result = $2, error = $3, started_at = $4, completed_at = $5, retry_count = $6 WHERE id = $7",
		task.Status, task.Result, task.Error, task.StartedAt, task.CompletedAt, task.RetryCount, uuid)
	if err != nil {
		return fmt.Errorf("updating task: %w", err)
	}
	return nil
}

func (r *Repository) UpdateStatusTask(ctx context.Context, uuid uuid.UUID, status string) error {
	exists, err := r.checkExisting(ctx, "tasks", uuid)
	if err != nil {
		return fmt.Errorf("checking task existing: %w", err)
	}
	if !exists {
		return fmt.Errorf("no task with id = %s", uuid)
	}

	_, err = r.db.Exec(ctx, "UPDATE tasks SET status = $1 WHERE id = $2", status, uuid)
	if err != nil {
		return fmt.Errorf("updating task: %w", err)
	}
	return nil
}

func (r *Repository) ReloadTask(ctx context.Context, uuid uuid.UUID) error {
	exists, err := r.checkExisting(ctx, "tasks", uuid)
	if err != nil {
		return fmt.Errorf("checking task existing: %w", err)
	}
	if !exists {
		return fmt.Errorf("no task with id = %s", uuid)
	}
	var status string
	var retriesCount int
	var maxRetries int
	if err = r.db.QueryRow(ctx, "SELECT status, retry_count, max_retries FROM tasks WHERE id = $1", uuid).Scan(&status, &retriesCount, &maxRetries); err != nil {
		return fmt.Errorf("get task status: %w", err)
	}
	if status == "pending" || status == "running" {
		return fmt.Errorf("task already in work")
	}
	if retriesCount >= maxRetries {
		return fmt.Errorf("too many retries, denied")
	}

	_, err = r.db.Exec(ctx, "UPDATE tasks SET status = 'pending', retry_count = retry_count + 1 WHERE id = $1", uuid)
	if err != nil {
		return fmt.Errorf("reload task: %w", err)
	}

	task, err := r.GetTaskById(ctx, uuid)
	if err != nil {
		return fmt.Errorf("get task by id for adding to queue: %w", err)
	}
	task.Result = nil
	task.Error = ""
	task.CreatedAt = time.Now()

	if err := r.Cache.AddTaskToQueue(ctx, task); err != nil {
		return fmt.Errorf("adding task to queue after retry: %w", err)
	}
	return nil
}

func (r *Repository) GetTaskById(ctx context.Context, uuid uuid.UUID) (model.Task, error) {
	var task model.Task
	query := "SELECT id, type, status, payload, result, error, priority,created_at, started_at, completed_at, retry_count, max_retries FROM tasks WHERE id = $1"
	if err := r.db.QueryRow(ctx, query, uuid).Scan(&task.ID, &task.Type, &task.Status, &task.Payload, &task.Result, &task.Error, &task.Priority, &task.CreatedAt, &task.StartedAt, &task.CompletedAt, &task.RetryCount, &task.MaxRetries); err != nil {
		return model.Task{}, fmt.Errorf("get task by id: %w", err)
	}

	return task, nil
}
