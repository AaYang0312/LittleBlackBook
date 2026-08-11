package interaction

import (
	"context"
	"errors"
	"testing"
	"xbs/internal/pkg/db"
	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/mq"
	"xbs/internal/pkg/snowflake"

	"github.com/alicebob/miniredis/v2"
	mysqlDriver "github.com/go-sql-driver/mysql"
)

type fakeFollowRepo struct {
	pairs     map[[2]int64]bool
	insertErr error
}

func newFakeFollowRepo() *fakeFollowRepo { return &fakeFollowRepo{pairs: make(map[[2]int64]bool)} }
func (f *fakeFollowRepo) InsertIgnore(_ context.Context, a, b int64) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	f.pairs[[2]int64{a, b}] = true
	return nil
}
func (f *fakeFollowRepo) Delete(_ context.Context, a, b int64) error {
	delete(f.pairs, [2]int64{a, b})
	return nil
}
func (f *fakeFollowRepo) IsFollowing(_ context.Context, a, b int64) (bool, error) {
	return f.pairs[[2]int64{a, b}], nil
}
func (f *fakeFollowRepo) FollowerIDs(_ context.Context, b int64) ([]int64, error) {
	var out []int64
	for k := range f.pairs {
		if k[1] == b {
			out = append(out, k[0])
		}
	}
	return out, nil
}
func TestLikeIdempotent(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := db.NewRedis(mr.Addr(), "", 0)
	var published []mq.LikeEvent
	svc := NewServiceForTest(&Repos{}, rdb,
		func(_ context.Context, ev mq.LikeEvent) error {
			published = append(published, ev)
			return nil
		}, nil)
	ctx := context.Background()
	if err := svc.Like(ctx, 1, 100); err != nil {
		t.Fatal(err)
	}
	if err := svc.Like(ctx, 1, 100); err != nil { // 双击 → 幂等
		t.Fatal(err)
	}
	if len(published) != 1 || published[0].Delta != 1 || published[0].Kind != "like" {
		t.Fatalf("published=%v", published)
	}
	cnt, _ := rdb.Get(ctx, "note:like:count:100").Int64()
	if cnt != 1 {
		t.Fatalf("count=%d", cnt)
	}
	if err := svc.Unlike(ctx, 1, 100); err != nil {
		t.Fatal(err)
	}
	if err := svc.Unlike(ctx, 1, 100); err != nil { // 重复取消 → 幂等
		t.Fatal(err)
	}
	if len(published) != 2 || published[1].Delta != -1 {
		t.Fatalf("published=%v", published)
	}
	cnt, _ = rdb.Get(ctx, "note:like:count:100").Int64()
	if cnt != 0 {
		t.Fatalf("count=%d", cnt)
	}
}

func TestFollowTargetNotExists(t *testing.T) {
	repo := newFakeFollowRepo()
	repo.insertErr = &mysqlDriver.MySQLError{Number: 1452} // 外键 violation
	svc := NewService(&Repos{Follow: repo}, nil, nil, nil)
	err := svc.Follow(context.Background(), 1, 999)
	if !errors.Is(err, errs.ErrUserNotFound) {
		t.Fatalf("err=%v, want ErrUserNotFound", err)
	}
}

func TestFollowIdempotent(t *testing.T) {
	repo := newFakeFollowRepo()
	svc := NewService(&Repos{Follow: repo}, nil, nil, nil)
	ctx := context.Background()
	if err := svc.Follow(ctx, 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := svc.Follow(ctx, 1, 2); err != nil { // 重复关注不报错
		t.Fatal(err)
	}
	fans, _ := svc.FollowerIDs(ctx, 2)
	if len(fans) != 1 || fans[0] != 1 {
		t.Fatalf("fans=%v", fans)
	}
	if err := svc.Unfollow(ctx, 1, 2); err != nil {
		t.Fatal(err)
	}
	fans, _ = svc.FollowerIDs(ctx, 2)
	if len(fans) != 0 {
		t.Fatalf("after unfollow fans=%v", fans)
	}
}

type fakeLikeRepo struct {
	pairs map[[2]int64]bool
}

func newFakeLikeRepo() *fakeLikeRepo { return &fakeLikeRepo{pairs: make(map[[2]int64]bool)} }

func (f *fakeLikeRepo) InsertIgnore(_ context.Context, a, b int64) (bool, error) {
	k := [2]int64{a, b}
	if f.pairs[k] {
		return false, nil
	}
	f.pairs[k] = true
	return true, nil
}
func (f *fakeLikeRepo) Delete(_ context.Context, a, b int64) (bool, error) {
	k := [2]int64{a, b}
	if !f.pairs[k] {
		return false, nil
	}
	delete(f.pairs, k)
	return true, nil
}
func (f *fakeLikeRepo) CountByNote(_ context.Context, b int64) (int64, error) {
	var a int64
	for k := range f.pairs {
		if k[1] == b {
			a++
		}
	}
	return a, nil
}

type fakeNoteCounter struct {
	deltas []int
}

func (f *fakeNoteCounter) AddCountDelta(_ context.Context, _ int64, _ string, delta int) error {
	f.deltas = append(f.deltas, delta)
	return nil
}
func (f *fakeNoteCounter) ListAllIDs(context.Context) ([]int64, error)                 { return nil, nil }
func (f *fakeNoteCounter) SetCounts(context.Context, int64, int64, int64, int64) error { return nil }

func TestApplyLikeEventExactlyOnce(t *testing.T) {
	likes := newFakeLikeRepo()
	nc := &fakeNoteCounter{}
	svc := NewServiceForTest(&Repos{Like: likes}, nil, nil, nc)
	ctx := context.Background()
	ev := mq.LikeEvent{
		Kind:   "like",
		NoteID: 100,
		UserID: 1,
		Delta:  1,
	}
	if err := svc.ApplyLikeEvent(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if err := svc.ApplyLikeEvent(ctx, ev); err != nil {
		t.Fatal(err) // mq 重复投递
	}
	if len(nc.deltas) != 1 || nc.deltas[0] != 1 {
		t.Fatalf("delta=%v", nc.deltas)
	}
	ev.Delta = -1
	if err := svc.ApplyLikeEvent(ctx, ev); err != nil {
		t.Fatal(err)
	}
	if err := svc.ApplyLikeEvent(ctx, ev); err != nil { // 重复的取消事件
		t.Fatal(err)
	}
	if len(nc.deltas) != 2 || nc.deltas[1] != -1 {
		t.Fatalf("deltas=%v", nc.deltas)
	}
}

type fakeCommentRepo struct {
	list []*Comment
}

func (f *fakeCommentRepo) Create(_ context.Context, c *Comment) error {
	f.list = append(f.list, c)
	return nil
}
func (f *fakeCommentRepo) ListByNote(_ context.Context, noteID, _ int64, _ int) ([]*Comment, error) {
	var out []*Comment
	for _, c := range f.list {
		if c.NoteID == noteID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeCommentRepo) CountByNote(_ context.Context, noteID int64) (int64, error) {
	var n int64
	for _, c := range f.list {
		if c.NoteID == noteID {
			n++
		}
	}
	return n, nil
}
func TestCreateCommentIncrementsCount(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := db.NewRedis(mr.Addr(), "", 0)
	_ = snowflake.Init(1)
	nc := &fakeNoteCounter{}
	comments := &fakeCommentRepo{}
	svc := NewServiceForTest(&Repos{Comment: comments}, rdb, nil, nc)
	c, err := svc.CreateComment(context.Background(), 1, 100, "good")
	if err != nil {
		t.Fatal(err)
	}
	if c.ID <= 0 {
		t.Fatalf("bad comment id: %d", c.ID)
	}
	if len(nc.deltas) != 1 || nc.deltas[0] != 1 {
		t.Fatalf("note delta=%v", nc.deltas)
	}
	cnt, _ := rdb.Get(context.Background(), "note:comment:count:100").Int64()
	if cnt != 1 {
		t.Fatalf("redis comment count=%d", cnt)
	}
	if _, err := svc.CreateComment(context.Background(), 1, 100, ""); !errors.Is(err, errs.ErrParam) {
		t.Fatalf("empty content should fail")
	}
}
