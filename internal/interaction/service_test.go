package interaction

import (
	"context"
	"errors"
	"sort"
	"testing"
	"xbs/internal/pkg/db"
	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/mq"
	"xbs/internal/pkg/snowflake"
	"xbs/internal/user"

	"github.com/alicebob/miniredis/v2"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
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
		}, nil, nil)
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
	svc := NewService(&Repos{Follow: repo}, nil, nil, nil, nil)
	err := svc.Follow(context.Background(), 1, 999)
	if !errors.Is(err, errs.ErrUserNotFound) {
		t.Fatalf("err=%v, want ErrUserNotFound", err)
	}
}

func TestFollowIdempotent(t *testing.T) {
	repo := newFakeFollowRepo()
	svc := NewService(&Repos{Follow: repo}, nil, nil, nil, nil)
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

type fakeUserLookup struct{ byID map[int64]*user.Author }

func (f fakeUserLookup) BatchByIDs(_ context.Context, ids []int64) (map[int64]*user.Author, error) {
	out := map[int64]*user.Author{}
	for _, id := range ids {
		if a, ok := f.byID[id]; ok {
			out[id] = a
		}
	}
	return out, nil
}

func TestApplyLikeEventExactlyOnce(t *testing.T) {
	likes := newFakeLikeRepo()
	nc := &fakeNoteCounter{}
	svc := NewServiceForTest(&Repos{Like: likes}, nil, nil, nc, nil)
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
	byID map[int64]*Comment
}

func newFakeCommentRepo() *fakeCommentRepo { return &fakeCommentRepo{byID: map[int64]*Comment{}} }

func (f *fakeCommentRepo) Create(_ context.Context, c *Comment) error {
	f.list = append(f.list, c)
	f.byID[c.ID] = c
	return nil
}
func (f *fakeCommentRepo) ListTopLevelByNote(_ context.Context, noteID, cursor int64, size int) ([]*Comment, error) {
	var out []*Comment
	for _, c := range f.list {
		if c.NoteID == noteID && c.ParentID == 0 && (cursor == 0 || c.ID < cursor) {
			out = append(out, c)
		}
	}
	if len(out) > size {
		out = out[:size]
	}
	return out, nil
}
func (f *fakeCommentRepo) ListReplies(_ context.Context, parentID, cursor int64, size int) ([]*Comment, error) {
	var out []*Comment
	for _, c := range f.list {
		if c.ParentID == parentID && (cursor == 0 || c.ID > cursor) {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	if len(out) > size {
		out = out[:size]
	}
	return out, nil
}
func (f *fakeCommentRepo) FindByID(_ context.Context, id int64) (*Comment, error) {
	if c, ok := f.byID[id]; ok {
		return c, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeCommentRepo) IncrementReplyCount(_ context.Context, parentID int64, delta int) error {
	if c, ok := f.byID[parentID]; ok {
		c.ReplyCount += int64(delta)
	}
	return nil
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
	comments := newFakeCommentRepo()
	svc := NewServiceForTest(&Repos{Comment: comments}, rdb, nil, nc, nil)
	c, err := svc.CreateComment(context.Background(), 1, 100, "good", 0, 0)
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
	if _, err := svc.CreateComment(context.Background(), 1, 100, "", 0, 0); !errors.Is(err, errs.ErrParam) {
		t.Fatalf("empty content should fail")
	}
}

type fakeNoteCounterWithIDs struct {
	ids         []int64
	setCalls    int
	lastLike    int64
	lastComment int64
}

func (f *fakeNoteCounterWithIDs) AddCountDelta(context.Context, int64, string, int) error { return nil }
func (f *fakeNoteCounterWithIDs) ListAllIDs(context.Context) ([]int64, error)             { return f.ids, nil }
func (f *fakeNoteCounterWithIDs) SetCounts(_ context.Context, _ int64, like, _, comment int64) error {
	f.setCalls++
	f.lastLike = like
	f.lastComment = comment
	return nil
}

// CollectRepository 与 LikeRepository 形状相同，直接复用 fakeLikeRepo 不满足类型时包一层：
type fakeCollectRepo struct{ inner *fakeLikeRepo }

func newFakeLikeRepoAdapter() *fakeCollectRepo { return &fakeCollectRepo{inner: newFakeLikeRepo()} }
func (f *fakeCollectRepo) InsertIgnore(ctx context.Context, a, b int64) (bool, error) {
	return f.inner.InsertIgnore(ctx, a, b)
}
func (f *fakeCollectRepo) Delete(ctx context.Context, a, b int64) (bool, error) {
	return f.inner.Delete(ctx, a, b)
}
func (f *fakeCollectRepo) CountByNote(ctx context.Context, b int64) (int64, error) {
	return f.inner.CountByNote(ctx, b)
}
func TestRebuildCounts(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := db.NewRedis(mr.Addr(), "", 0)
	likes := newFakeLikeRepo()
	likes.pairs[[2]int64{1, 100}] = true
	likes.pairs[[2]int64{2, 100}] = true
	comments := &fakeCommentRepo{list: []*Comment{{NoteID: 100, Content: "x"}}}
	nc := &fakeNoteCounterWithIDs{ids: []int64{100}}
	svc := NewServiceForTest(&Repos{Like: likes, Comment: comments, Collect: newFakeLikeRepoAdapter()}, rdb, nil, nc, nil)
	if err := svc.RebuildCounts(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if v, _ := rdb.Get(ctx, "note:like:count:100").Int64(); v != 2 {
		t.Fatalf("like count=%d", v)
	}
	if v, _ := rdb.Get(ctx, "note:comment:count:100").Int64(); v != 1 {
		t.Fatalf("comment count=%d", v)
	}
	if nc.setCalls != 1 || nc.lastLike != 2 || nc.lastComment != 1 {
		t.Fatalf("SetCalls=%d like=%d comment=%d", nc.setCalls, nc.lastLike, nc.lastComment)
	}
}

func TestCreateReplyValidation(t *testing.T) {
	_ = snowflake.Init(1)
	mr := miniredis.RunT(t)
	rdb := db.NewRedis(mr.Addr(), "", 0)
	comments := newFakeCommentRepo()
	nc := &fakeNoteCounter{}
	svc := NewServiceForTest(&Repos{Comment: comments}, rdb, nil, nc, nil)
	ctx := context.Background()
	top, err := svc.CreateComment(ctx, 1, 100, "top", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateComment(ctx, 2, 100, "r", 999, 0); !errors.Is(err, errs.ErrCommentNotFound) {
		t.Fatalf("want ErrCommentNotFound, got %v", err)
	}
	if _, err := svc.CreateComment(ctx, 2, 200, "r", top.ID, 0); !errors.Is(err, errs.ErrCommentNotFound) {
		t.Fatalf("want ErrCommentNotFound for note mismatch, got %v", err)
	}
	r, err := svc.CreateComment(ctx, 2, 100, "reply", top.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if r.ParentID != top.ID {
		t.Fatalf("reply parent_id=%d", r.ParentID)
	}
	if len(nc.deltas) != 2 {
		t.Fatalf("note deltas=%v (want 2: top+reply)", nc.deltas)
	}
	parent, _ := comments.FindByID(ctx, top.ID)
	if parent.ReplyCount != 1 {
		t.Fatalf("reply_count=%d", parent.ReplyCount)
	}
	if _, err := svc.CreateComment(ctx, 3, 100, "r2", r.ID, 0); !errors.Is(err, errs.ErrParam) {
		t.Fatalf("want ErrParam for replying to a reply, got %v", err)
	}
}

func TestListCommentsTopLevelOnly(t *testing.T) {
	_ = snowflake.Init(1)
	comments := newFakeCommentRepo()
	nc := &fakeNoteCounter{}
	svc := NewServiceForTest(&Repos{Comment: comments}, nil, nil, nc, nil)
	ctx := context.Background()
	top, _ := svc.CreateComment(ctx, 1, 100, "top", 0, 0)
	svc.CreateComment(ctx, 2, 100, "reply", top.ID, 0)
	list, err := svc.ListComments(ctx, 100, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != top.ID || list[0].ReplyCount != 1 {
		t.Fatalf("top-level list=%+v", list)
	}
}

func TestListReplies(t *testing.T) {
	_ = snowflake.Init(1)
	comments := newFakeCommentRepo()
	nc := &fakeNoteCounter{}
	svc := NewServiceForTest(&Repos{Comment: comments}, nil, nil, nc, nil)
	ctx := context.Background()
	top, _ := svc.CreateComment(ctx, 1, 100, "top", 0, 0)
	r1, _ := svc.CreateComment(ctx, 2, 100, "r1", top.ID, 1)
	r2, _ := svc.CreateComment(ctx, 3, 100, "r2", top.ID, 2)
	if _, err := svc.ListReplies(ctx, 100, 999, 0, 20); !errors.Is(err, errs.ErrCommentNotFound) {
		t.Fatalf("want ErrCommentNotFound, got %v", err)
	}
	if _, err := svc.ListReplies(ctx, 200, top.ID, 0, 20); !errors.Is(err, errs.ErrCommentNotFound) {
		t.Fatalf("want ErrCommentNotFound for note mismatch, got %v", err)
	}
	reps, err := svc.ListReplies(ctx, 100, top.ID, 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(reps) != 2 || reps[0].ID != r1.ID || reps[1].ID != r2.ID {
		t.Fatalf("replies not ascending: %+v", reps)
	}
}

func TestCommentAuthorEnrichment(t *testing.T) {
	_ = snowflake.Init(1)
	comments := newFakeCommentRepo()
	nc := &fakeNoteCounter{}
	lookup := fakeUserLookup{byID: map[int64]*user.Author{1: {ID: 1, Nickname: "小红"}, 2: {ID: 2, Nickname: "小明"}}}
	svc := NewServiceForTest(&Repos{Comment: comments}, nil, nil, nc, lookup)
	ctx := context.Background()
	top, _ := svc.CreateComment(ctx, 1, 100, "top", 0, 0)
	if top.Author == nil || top.Author.Nickname != "小红" {
		t.Fatalf("top author not enriched: %+v", top.Author)
	}
	r, _ := svc.CreateComment(ctx, 2, 100, "reply", top.ID, 1)
	if r.Author == nil || r.Author.Nickname != "小明" || r.ReplyToAuthor == nil || r.ReplyToAuthor.Nickname != "小红" {
		t.Fatalf("reply authors not enriched: author=%+v replyTo=%+v", r.Author, r.ReplyToAuthor)
	}
}
