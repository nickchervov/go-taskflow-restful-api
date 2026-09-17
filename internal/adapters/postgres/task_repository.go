package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"restful-taskflow/internal/domain"

	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(ctx context.Context, dbURL string) (*Repository, error) {
	dbConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("creating db config: %w", err)
	}
	dbPool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		return nil, fmt.Errorf("connecting postgres db: %w", err)
	}

	err = createTables(ctx, dbPool)
	if err != nil {
		return nil, err
	}

	return &Repository{db: dbPool}, nil
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

func (r *Repository) AddTask(ctx context.Context, task domain.Task) (uuid.UUID, error) {
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

func (r *Repository) GetTaskStatusResult(ctx context.Context, id uuid.UUID) (domain.TaskStatusResult, error) {
	exists, err := r.checkExisting(ctx, "tasks", id)
	if err != nil {
		return domain.TaskStatusResult{}, fmt.Errorf("check existing row: %w", err)
	}
	if !exists {
		return domain.TaskStatusResult{}, domain.ErrTaskNotFound
	}

	var task domain.TaskStatusResult
	task.ID = id
	row := r.db.QueryRow(ctx, "SELECT status, result FROM tasks WHERE id = $1", id)
	if err := row.Scan(&task.Status, &task.Result); err != nil {
		return domain.TaskStatusResult{}, fmt.Errorf("scanning task row: %w", err)
	}
	return task, nil
}
func (r *Repository) GetAllTasks(ctx context.Context) ([]domain.Task, error) {
	taskList := make([]domain.Task, 0)
	rows, err := r.db.Query(ctx, "SELECT id, type, status, payload, result, error, priority, created_at, started_at, completed_at, retry_count, max_retries FROM tasks")
	if err != nil {
		return nil, fmt.Errorf("get tasks rows: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var task domain.Task
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
	tag, err := r.db.Exec(ctx, `UPDATE tasks SET status='cancelled' WHERE id=$1 AND status='pending'`, uuid)
	if err != nil {
		return fmt.Errorf("cancel task: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	exists, err := r.checkExisting(ctx, "tasks", uuid)
	if err != nil {
		return fmt.Errorf("check existing row: %w", err)
	}
	if !exists {
		return domain.ErrTaskNotFound
	}
	return domain.ErrTaskNotCancelable
}

func (r *Repository) UpdateTask(ctx context.Context, uuid uuid.UUID, task domain.Task) error {
	exists, err := r.checkExisting(ctx, "tasks", uuid)
	if err != nil {
		return fmt.Errorf("checking task existing: %w", err)
	}
	if !exists {
		return domain.ErrTaskNotFound
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
		return domain.ErrTaskNotFound
	}

	_, err = r.db.Exec(ctx, "UPDATE tasks SET status = $1 WHERE id = $2", status, uuid)
	if err != nil {
		return fmt.Errorf("updating task: %w", err)
	}
	return nil
}

func (r *Repository) ReloadTask(ctx context.Context, uuid uuid.UUID) (domain.Task, error) {
	exists, err := r.checkExisting(ctx, "tasks", uuid)
	if err != nil {
		return domain.Task{}, fmt.Errorf("checking task existing: %w", err)
	}
	if !exists {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	var status string
	var retriesCount int
	var maxRetries int
	if err = r.db.QueryRow(ctx, "SELECT status, retry_count, max_retries FROM tasks WHERE id = $1", uuid).Scan(&status, &retriesCount, &maxRetries); err != nil {
		return domain.Task{}, fmt.Errorf("get task status: %w", err)
	}
	if status == "pending" || status == "running" {
		return domain.Task{}, domain.ErrTaskAlreadyInWork
	}
	if retriesCount >= maxRetries {
		return domain.Task{}, domain.ErrTooManyRetries
	}

	_, err = r.db.Exec(ctx, "UPDATE tasks SET status = 'pending', retry_count = retry_count + 1 WHERE id = $1", uuid)
	if err != nil {
		return domain.Task{}, fmt.Errorf("reload task: %w", err)
	}

	task, err := r.GetTaskById(ctx, uuid)
	if err != nil {
		return domain.Task{}, fmt.Errorf("get task by id for adding to queue: %w", err)
	}
	task.Result = nil
	task.Error = ""
	task.CreatedAt = time.Now()

	return task, nil
}

func (r *Repository) GetTaskById(ctx context.Context, uuid uuid.UUID) (domain.Task, error) {
	var task domain.Task
	query := "SELECT id, type, status, payload, result, error, priority,created_at, started_at, completed_at, retry_count, max_retries FROM tasks WHERE id = $1"
	if err := r.db.QueryRow(ctx, query, uuid).Scan(&task.ID, &task.Type, &task.Status, &task.Payload, &task.Result, &task.Error, &task.Priority, &task.CreatedAt, &task.StartedAt, &task.CompletedAt, &task.RetryCount, &task.MaxRetries); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Task{}, domain.ErrTaskNotFound
		}
		return domain.Task{}, fmt.Errorf("get task by id: %w", err)
	}

	return task, nil
}

func (r *Repository) MarkRunning(ctx context.Context, id uuid.UUID) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE tasks SET status = 'running', started_at = now()
		WHERE id = $1 AND status = 'pending'`, id)
	if err != nil {
		return false, fmt.Errorf("mark running: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
