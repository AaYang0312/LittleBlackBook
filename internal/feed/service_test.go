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
func TestInboxOrderAndSkipDeleted(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := db.NewRedis(mr.Addr(), "", 0)
	ctx := context.Background()
	// 模拟三条：101，102，103，其中 102 被删除，BatchByIDs 查不到
	rdb.ZAdd(ctx, feed.InboxKey(7), redis.Z{
		Score:  3,
		Member: 103,
	}, redis.Z{
		Score:  2,
		Member: 102,
	}, redis.Z{
		Score:  1,
		Member: 101,
	})
	notes := fakeNotes{m: map[int64]*note.NoteDTO{
		101: {ID: 101, Title: "a"},
		103: {ID: 103, Title: "c"},
	}}
	svc := feed.NewService(rdb, notes, fakeFollows{}, 500)
	out, err := svc.Inbox(ctx, 7, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].ID != 103 || out[1].ID != 101 {
		t.Fatalf("inbox order wrong: %+v", out)
	}
}
