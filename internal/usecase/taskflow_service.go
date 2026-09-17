package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"restful-taskflow/internal/domain"
	"restful-taskflow/internal/dto"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Repository interface {
	AddTask(ctx context.Context, task domain.Task) (uuid.UUID, error)
	GetTaskStatusResult(ctx context.Context, id uuid.UUID) (domain.TaskStatusResult, error)
	GetAllTasks(ctx context.Context) ([]domain.Task, error)
	CancelTask(ctx context.Context, uuid uuid.UUID) error
	UpdateTask(ctx context.Context, uuid uuid.UUID, task domain.Task) error
	UpdateStatusTask(ctx context.Context, uuid uuid.UUID, status string) error
	ReloadTask(ctx context.Context, uuid uuid.UUID) (domain.Task, error)
	GetTaskById(ctx context.Context, uuid uuid.UUID) (domain.Task, error)
}

type Cache interface {
	CacheTaskStatusResult(ctx context.Context, taskResultStatus domain.TaskStatusResult) error
	GetTaskStatusResultFromCache(ctx context.Context, uuid uuid.UUID) (domain.TaskStatusResult, error)
	IncrCounterProcessedTasks(ctx context.Context)
	AddTaskToQueue(ctx context.Context, task domain.Task) error
	BRPopTask(ctx context.Context) (domain.Task, error)
	CheckLimit(ctx context.Context, ip string, limit int) (bool, time.Duration)
}

type TaskflowService struct {
	Repo  Repository
	Cache Cache
}

func NewTaskflowService(repo Repository, cache Cache) *TaskflowService {
	return &TaskflowService{Repo: repo, Cache: cache}
}
func (s *TaskflowService) CreateTask(ctx context.Context, input dto.CreateTaskInput) (dto.CreateTaskOutput, error) {
	if input.Type != "send_email" && input.Type != "resize_image" && input.Type != "generate_report" {
		slog.Error("invalid task type", "task type", input.Type)
		return dto.CreateTaskOutput{}, domain.ErrInvalidTaskType
	}
	payload, err := json.Marshal(input.Payload)
	if err != nil {
		slog.Error("encoding payload to json", "error", err)
		return dto.CreateTaskOutput{}, fmt.Errorf("encoding payload to json: %w", err)
	}
	task := domain.NewTask(input.Type, "pending", payload)

	id, err := s.Repo.AddTask(ctx, task)
	if err != nil {
		slog.Error("adding task", "error", err)
		return dto.CreateTaskOutput{}, fmt.Errorf("adding task to db: %w", err)
	}

	task.ID = id
	err = s.Cache.AddTaskToQueue(ctx, task)
	if err != nil {
		slog.Error("adding task to queue", "error", err)
		return dto.CreateTaskOutput{}, fmt.Errorf("adding task to cache: %w", err)
	}

	return dto.CreateTaskOutput{Id: id}, nil
}
func (s *TaskflowService) GetStatusOrResult(ctx context.Context, input dto.GetStatusResultInput) (dto.GetStatusResultOutput, error) {
	statusResult, err := s.Cache.GetTaskStatusResultFromCache(ctx, input.Id)
	if err == nil {
		slog.Info("OK get status or result from cache", "status", statusResult.Status, "result", statusResult.Result)
		return dto.GetStatusResultOutput{TaskStatusResult: statusResult}, nil
	}
	if !errors.Is(err, redis.Nil) {
		slog.Error("getting status result from cache", "error", err)
		return dto.GetStatusResultOutput{}, fmt.Errorf("getting status result from cache: %w", err)
	}

	taskStatusResult, err := s.Repo.GetTaskStatusResult(ctx, input.Id)
	if err != nil {
		slog.Error("getting task status and result", "error", err)
		return dto.GetStatusResultOutput{}, fmt.Errorf("getting status or result from db: %w", err)
	}
	return dto.GetStatusResultOutput{TaskStatusResult: taskStatusResult}, nil
}
func (s *TaskflowService) TasksList(ctx context.Context) (dto.TasksListsOutput, error) {
	tasks, err := s.Repo.GetAllTasks(ctx)
	if err != nil {
		slog.Error("getting tasks list", "error", err)
		return dto.TasksListsOutput{}, fmt.Errorf("getting all tasks: %w", err)
	}
	return dto.TasksListsOutput{Tasks: tasks}, nil
}
func (s *TaskflowService) CancelTask(ctx context.Context, input dto.CancelTaskInput) error {
	if err := s.Repo.CancelTask(ctx, input.Id); err != nil {
		slog.Error("cancelling task", "error", err)
		return fmt.Errorf("cancelling task: %w", err)
	}
	return nil
}
func (s *TaskflowService) ReloadTask(ctx context.Context, input dto.ReloadTaskInput) error {
	task, err := s.Repo.ReloadTask(ctx, input.Id)
	if err != nil {
		slog.Error("reloading task", "error", err)
		return fmt.Errorf("reload task: %w", err)
	}
	if err := s.Cache.AddTaskToQueue(ctx, task); err != nil {
		return fmt.Errorf("adding task to queue after retry: %w", err)
	}
	return nil
}
