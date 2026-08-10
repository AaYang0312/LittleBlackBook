package cache

import (
	"context"
	"errors"
	"math/rand"
	"time"
	"xbs/internal/pkg/errs"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

const notFoundTTL = 30 * time.Second

type Cache struct {
	rdb *redis.Client
	g   singleflight.Group
}

func New(rdb *redis.Client) *Cache {
	return &Cache{
		rdb: rdb,
	}
}

// GetOrLoad: 空值标记防穿透；singleflight 防击穿；TTL 加随机抖动防雪崩
func (c *Cache) GetOrLoad(ctx context.Context, key string, ttl time.Duration, load func(ctx2 context.Context) (string, error)) (string, error) {
	// 检查 redis
	if n, _ := c.rdb.Exists(ctx, key+":notfound").Result(); n > 0 {
		return "", errs.ErrNoteNotFound
	}
	if v, err := c.rdb.Get(ctx, key).Result(); err == nil {
		return v, nil
	}
	// redis 没查到，走 singleflight
	v, err, _ := c.g.Do(key, func() (any, error) {
		s, err := load(ctx) // 调 load 查数据库
		// db 没查到
		if errors.Is(err, errs.ErrNoteNotFound) {
			_ = c.rdb.Set(ctx, key+":notfound", "1", notFoundTTL).Err()
			return "", err
		}
		if err != nil {
			return "", err
		}
		// db 查到，加抖动写入 redis
		jitter := time.Duration(rand.Int63n(int64(ttl / 5)))
		_ = c.rdb.Set(ctx, key, s, ttl+jitter).Err() // 写 redis
		return s, nil
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}
