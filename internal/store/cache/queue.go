package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"restful-taskflow/internal/model"
)

func (c *Cache) AddTaskToQueue(ctx context.Context, task model.Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("serializing task to queue: %w", err)
	}
	if err := c.rdb.LPush(ctx, "queue:pending", data).Err(); err != nil {
		return fmt.Errorf("add to queue: %w", err)
	}
	return nil
}

func (c *Cache) BRPopTask(ctx context.Context) (model.Task, error) {
	result, err := c.rdb.BRPop(ctx, 0, "queue:pending").Result()
	if err != nil {
		return model.Task{}, fmt.Errorf("brpop task by type: %w", err)
	}

	var task model.Task
	if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
		return model.Task{}, fmt.Errorf("deserializing task from queue: %w", err)
	}
	return task, nil
}
