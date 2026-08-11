package feed_test

import (
	"context"
	"testing"
	"xbs/internal/feed"
	"xbs/internal/note"
	"xbs/internal/pkg/db"
	"xbs/internal/pkg/mq"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type fakeFollows struct {
	fans []int64
}

func (f fakeFollows) FollowerIDs(context.Context, int64) ([]int64, error) { return f.fans, nil }

type fakeNotes struct {
	m map[int64]*note.NoteDTO
}

func (f fakeNotes) BatchByIDs(_ context.Context, ids []int64) ([]*note.NoteDTO, error) {
	var out []*note.NoteDTO
	for _, id := range ids {
		if d, ok := f.m[id]; ok {
			out = append(out, d)
		}
	}
	return out, nil
}
func TestHandleFanoutWritesInboxes(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := db.NewRedis(mr.Addr(), "", 0)
	svc := feed.NewService(rdb, fakeNotes{}, fakeFollows{fans: []int64{11, 22, 33}}, 500)
	ev := mq.FanoutEvent{NoteID: 100, AuthorID: 1, Ts: 1700000000000}
	if err := svc.HandleFanout(context.Background(), ev); err != nil {
		t.Fatal(err)
	}
	for _, fan := range []int64{11, 22, 33} {
		ids, err := rdb.ZRangeArgs(context.Background(), redis.ZRangeArgs{
			Key:   feed.InboxKey(fan),
			Start: 0,
			Stop:  -1,
			Rev:   true,
		}).Result()
		if err != nil || len(ids) != 1 || ids[0] != "100" {
			t.Fatalf("fan %d inbox=%v err=%v", fan, ids, err)
		}
	}
}

func TestHandleFanoutTrimsInbox(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := db.NewRedis(mr.Addr(), "", 0)
	svc := feed.NewService(rdb, fakeNotes{}, fakeFollows{fans: []int64{11}}, 5) // 测试用小上限
	ctx := context.Background()
	for i := int64(1); i <= 8; i++ {
		if err := svc.HandleFanout(ctx, mq.FanoutEvent{NoteID: i, AuthorID: 1, Ts: i}); err != nil {
			t.Fatal(err)
		}
	}
	ids, _ := rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
		Key:   feed.InboxKey(11),
		Start: 0,
		Stop:  -1,
		Rev:   true,
	}).Result()
	if len(ids) != 5 || ids[0] != "8" || ids[4] != "4" {
		t.Fatalf("trimmed inbox=%v", ids)
	}
}
