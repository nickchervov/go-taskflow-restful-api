package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"restful-taskflow/internal/model"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb *redis.Client
}

func NewCache(redisUrl string) *Cache {
	return &Cache{
		rdb: redis.NewClient(&redis.Options{
			Addr: redisUrl,
			DB:   0,
		}),
	}
}

func (c *Cache) CacheTaskStatusResult(ctx context.Context, taskResultStatus model.TaskStatusResult) error {
	key := fmt.Sprintf("task:%s:status-result", taskResultStatus.ID)
	data, err := json.Marshal(taskResultStatus)
	if err != nil {
		return fmt.Errorf("cache task status result serialization: %w", err)
	}
	if err := c.rdb.Set(ctx, key, data, 60*time.Second).Err(); err != nil {
		return fmt.Errorf("add task status result to cache: %w", err)
	}
	return nil
}

func (c *Cache) GetTaskStatusResultFromCache(ctx context.Context, uuid uuid.UUID) (model.TaskStatusResult, error) {
	key := fmt.Sprintf("task:%s:status-result", uuid)

	data, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return model.TaskStatusResult{}, fmt.Errorf("get task status result from cache: %w", err)
	}
	var statusResult model.TaskStatusResult
	if err := json.Unmarshal([]byte(data), &statusResult); err != nil {
		return model.TaskStatusResult{}, fmt.Errorf("deserializing status result from cache: %w", err)
	}
	return statusResult, nil
}

func (c *Cache) IncrCounterProcessedTasks(ctx context.Context) {
	c.rdb.Incr(ctx, "stats:processed")
}
