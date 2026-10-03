package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"restful-taskflow/internal/adapters/postgres"
	"restful-taskflow/internal/adapters/redis"
	"restful-taskflow/internal/controllers"
	"restful-taskflow/internal/usecase"
	"restful-taskflow/internal/worker"
	"restful-taskflow/pkg/httpserver"
	"restful-taskflow/pkg/logger"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")

	logger, err := logger.New(os.Getenv("LOG_LEVEL"), os.Getenv("LOG_FORMAT"))
	if err != nil {
		log.Fatalf("create logger")
	}
	slog.SetDefault(logger)

	stopCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cache := redis.NewCache(os.Getenv("REDIS_URL"))
	repository, err := postgres.NewRepository(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("creating repository", "error", err)
		os.Exit(1)
	}

	poolCtx, poolCancel := context.WithCancel(context.Background())
	defer poolCancel()

	pool, err := worker.NewWorkerPool(poolCtx, poolCancel, os.Getenv("WORKER_COUNT"), repository, cache)
	if err != nil {
		slog.Error("creating worker pool", "error", err)
		os.Exit(1)
	}
	pool.Start()

	svc := usecase.NewTaskflowService(repository, cache)

	router, err := controllers.SetRoutes(svc, os.Getenv("RATE_LIMIT"))
	if err != nil {
		slog.Error("creating router", "error", err)
		os.Exit(1)
	}
	server := httpserver.NewServer(router)

	slog.Info("starting server", "port", server.Addr)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server issue", "error", err)
			os.Exit(1)
		}
	}()

	<-stopCtx.Done()
	slog.Info("starting graceful shutdown")

	timeoutContext, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()

	if err := server.Shutdown(timeoutContext); err != nil {
		slog.Error("graceful shutdown issue", "error", err)
		os.Exit(1)
	}
	slog.Info("server is stopped")

	pool.Stop()
}
