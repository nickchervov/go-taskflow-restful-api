package worker

import (
	"context"
	"restful-taskflow/internal/store/postgres"
	"sync"
)

type WorkerPool struct {
	Repo    *postgres.Repository
	Workers int
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewWorkerPool(ctx context.Context, cancel context.CancelFunc, workersCount int, repo *postgres.Repository) *WorkerPool {
	return &WorkerPool{
		Repo:    repo,
		Workers: workersCount,
		ctx:     ctx,
		cancel:  cancel,
	}
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
