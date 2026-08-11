package feed

import (
	"context"
	"fmt"
	"xbs/internal/note"
	"xbs/internal/pkg/mq"

	"github.com/redis/go-redis/v9"
)

func InboxKey(userID int64) string { return fmt.Sprintf("feed:inbox:%d", userID) }

type FollowerLister interface {
	FollowerIDs(ctx context.Context, authorID int64) ([]int64, error)
}
type NoteLister interface {
	BatchByIDs(ctx context.Context, ids []int64) ([]*note.NoteDTO, error)
}
type Service interface {
	HandleFanout(ctx context.Context, ev mq.FanoutEvent) error
	//Inbox(ctx context.Context, userID int64, offset, size int) ([]*note.NoteDTO, error)
}

type service struct {
	rdb      *redis.Client
	notes    NoteLister
	follows  FollowerLister
	inboxMax int64
}

func NewService(rdb *redis.Client, notes NoteLister, follows FollowerLister, inboxMax int) Service {
	return &service{
		rdb:      rdb,
		notes:    notes,
		follows:  follows,
		inboxMax: int64(inboxMax),
	}
}

// HandleFanout 把笔记推入每个粉丝的 Inbox(ZSet)，并修剪最新 InboxMax 条
func (s *service) HandleFanout(ctx context.Context, ev mq.FanoutEvent) error {
	fans, err := s.follows.FollowerIDs(ctx, ev.AuthorID)
	if err != nil {
		return err
	}
	for _, fanID := range fans {
		key := InboxKey(fanID)
		if err := s.rdb.ZAdd(ctx, key, redis.Z{
			Score:  float64(ev.Ts),
			Member: ev.NoteID,
		}).Err(); err != nil {
			return err
		}
		if err := s.rdb.ZRemRangeByRank(ctx, key, 0, -(s.inboxMax + 1)).Err(); err != nil {
			return err
		}
	}
	return nil
}
