package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"restful-taskflow/internal/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

func (c *Cache) AddTaskToQueue(ctx context.Context, task domain.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("serializing task to queue: %w", err)
	}
	if err := c.rdb.LPush(ctx, "queue:pending", data).Err(); err != nil {
		return fmt.Errorf("add to queue: %w", err)
	}
	return nil
}

func (c *Cache) BRPopTask(ctx context.Context) (domain.Task, error) {
	result, err := c.rdb.BRPop(ctx, time.Second, "queue:pending").Result()
	if errors.Is(err, redis.Nil) {
		return domain.Task{}, domain.ErrNoTask
	}
	if err != nil {
		return domain.Task{}, fmt.Errorf("brpop task by type: %w", err)
	}

	var task domain.Task
	if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
		return domain.Task{}, fmt.Errorf("deserializing task from queue: %w", err)
	}
	return task, nil
}
