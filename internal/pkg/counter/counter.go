package counter

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	KindLike    = "like"
	KindCollect = "collect"
	KindComment = "comment"
)

func CountKey(kind string, noteID int64) string { return fmt.Sprintf("note:%s:count:%d", kind, noteID) }
func UsersKey(kind string, noteID int64) string { return fmt.Sprintf("note:%s:users:%d", kind, noteID) }

// Get 返回实时计数；Redis 无该 key 时返回 -1（调用方回落 DB 值）。
func Get(ctx context.Context, rdb *redis.Client, kind string, noteID int64) (int64, error) {
	v, err := rdb.Get(ctx, CountKey(kind, noteID)).Int64()
	if errors.Is(err, redis.Nil) {
		return -1, nil
	}
	return v, err
}
