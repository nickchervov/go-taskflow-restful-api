package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"restful-taskflow/internal/server"
	"restful-taskflow/internal/store/cache"
	"restful-taskflow/internal/store/postgres"
	"restful-taskflow/internal/worker"
	"restful-taskflow/pkg/config"
	"restful-taskflow/pkg/logger"
	"syscall"
	"time"
)

func main() {
	config := config.NewConfig()
	if err := config.Load(); err != nil {
		log.Fatalf("load config: %v", err)
	}

	logger, err := logger.New(config.LOG_LEVEL, config.LOG_FORMAT)
	if err != nil {
		log.Fatalf("create logger")
	}
	slog.SetDefault(logger)

	db, err := postgres.InitDb(context.Background(), config.DATABASE_URL)
	if err != nil {
		slog.Error("connect db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	stopCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cache := cache.NewCache(config.REDIS_URL)
	repository := postgres.NewRepository(db, cache)

	pool := worker.NewWorkerPool(stopCtx, cancel, config.WORKER_COUNT, repository)
	pool.Start()

	handler := server.NewHandler(repository)
	router := handler.SetRoutes(config.RATE_LIMIT)
	server := server.NewServer(router)

	slog.Info("starting server", "port", server.Addr)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server issue", "error", err)
			os.Exit(1)
		}
	}()

	<-stopCtx.Done()
	slog.Info("starting graceful shutdown")

	pool.Stop()

	timeoutContext, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()

	if err := server.Shutdown(timeoutContext); err != nil {
		slog.Error("graceful shutdown issue", "error", err)
		os.Exit(1)
	}
	slog.Info("server is stopped")
}
