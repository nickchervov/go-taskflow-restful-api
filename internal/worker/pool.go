package worker

import (
	"context"
	"fmt"
	"restful-taskflow/internal/domain"
	"strconv"

	"sync"

	"github.com/google/uuid"
)

type Storage interface {
	UpdateTask(ctx context.Context, uuid uuid.UUID, task domain.Task) error
	UpdateStatusTask(ctx context.Context, uuid uuid.UUID, status string) error
	MarkRunning(ctx context.Context, id uuid.UUID) (bool, error)
}

type Cache interface {
	AddTaskToQueue(ctx context.Context, task domain.Task) error
	BRPopTask(ctx context.Context) (domain.Task, error)
	CacheTaskStatusResult(ctx context.Context, taskResultStatus domain.TaskStatusResult) error
	IncrCounterProcessedTasks(ctx context.Context)
}

type WorkerPool struct {
	Repo    Storage
	Cache   Cache
	Workers int
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewWorkerPool(ctx context.Context, cancel context.CancelFunc, workersCount string, repo Storage, cache Cache) (*WorkerPool, error) {
	count, err := strconv.Atoi(workersCount)
	if err != nil {
		return nil, fmt.Errorf("wrong worker count: %w", err)
	}
	return &WorkerPool{Repo: repo, Cache: cache, Workers: count, ctx: ctx, cancel: cancel}, nil
}

func (w *WorkerPool) Start() {
	for range w.Workers {
		w.wg.Go(func() {
			w.Worker()
		})
	}
}

func (w *WorkerPool) Stop() {
	w.cancel()
	w.wg.Wait()
}
