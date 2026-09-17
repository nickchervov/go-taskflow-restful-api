package redis

import (
	"context"
	"fmt"
	"time"
)

func (c *Cache) CheckLimit(ctx context.Context, ip string, limit int) (bool, time.Duration) {
	key := fmt.Sprintf("limiter:%s", ip)

	count, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return true, 0
	}
	if count == 1 {
		c.rdb.Expire(ctx, key, 30*time.Second)
	}
	if count > int64(limit) {
		return false, c.rdb.TTL(ctx, key).Val()
	}
	return true, 0
}
