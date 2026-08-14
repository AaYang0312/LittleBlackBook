package note_test

import (
	"context"
	"errors"
	"io"
	"sort"
	"testing"
	"xbs/internal/pkg/cache"
	"xbs/internal/pkg/db"
	"xbs/internal/user"

	"github.com/alicebob/miniredis/v2"
	"gorm.io/gorm"

	"xbs/internal/note"
	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/snowflake"
)

type fakeRepo struct{ byID map[int64]*note.Note }

func newFakeRepo() *fakeRepo { return &fakeRepo{byID: map[int64]*note.Note{}} }

func (f *fakeRepo) Create(_ context.Context, n *note.Note) error {
	f.byID[n.ID] = n
	return nil
}
func (f *fakeRepo) FindByID(_ context.Context, id int64) (*note.Note, error) {
	if n, ok := f.byID[id]; ok {
		return n, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeRepo) BatchFindByIDs(_ context.Context, ids []int64) ([]*note.Note, error) {
	var out []*note.Note
	for _, id := range ids {
		if n, ok := f.byID[id]; ok {
			out = append(out, n)
		}
	}
	return out, nil
}
func (f *fakeRepo) ListLatest(_ context.Context, cursor int64, size int) ([]*note.Note, error) {
	var all []*note.Note
	for _, n := range f.byID {
		if n.Status == 0 && (cursor == 0 || n.ID < cursor) {
			all = append(all, n)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].ID > all[j].ID
	})
	if len(all) > size+1 {
		all = all[:size+1]
	}
	return all, nil
}
func (f *fakeRepo) SoftDelete(_ context.Context, id, userID int64) error {
	n, ok := f.byID[id]
	if !ok || n.UserID != userID {
		return errs.ErrForbidden
	}
	n.Status = 1
	return nil
}
func (f *fakeRepo) AddCountDelta(context.Context, int64, string, int) error     { return nil }
func (f *fakeRepo) ListAllIDs(context.Context) ([]int64, error)                 { return nil, nil }
func (f *fakeRepo) SetCounts(context.Context, int64, int64, int64, int64) error { return nil }

type fakeStorage struct{}

func (fakeStorage) Upload(_ context.Context, _ io.Reader, _ int64, name, _ string) (string, error) {
	return "http://minio/xhs-images/" + name, nil
}
func NewTestCache(t *testing.T) *cache.Cache {
	mr := miniredis.RunT(t)
	return cache.New(db.NewRedis(mr.Addr(), "", 0))
}

func TestPublishAndDetail(t *testing.T) {
	_ = snowflake.Init(1)
	repo := newFakeRepo()
	svc := note.NewService(repo, fakeStorage{}, nil, nil, NewTestCache(t), nil) // mq=nil 时跳过 fanout 发布；rdb=nil 时跳过实时计数覆盖
	n, err := svc.Publish(context.Background(), 7, "标题", "正文", []string{"http://minio/xhs-images/a.jpg"})
	if err != nil {
		t.Fatal(err)
	}
	if n.ID <= 0 || n.UserID != 7 || n.CoverURL != "http://minio/xhs-images/a.jpg" {
		t.Fatalf("bad note: %+v", n)
	}
	dto, err := svc.Detail(context.Background(), n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dto.Title != "标题" || len(dto.Images) != 1 {
		t.Fatalf("bad dto: %+v", dto)
	}
}

func TestDetailNotFound(t *testing.T) {
	_ = snowflake.Init(1)
	svc := note.NewService(newFakeRepo(), fakeStorage{}, nil, nil, NewTestCache(t), nil)
	_, err := svc.Detail(context.Background(), 999)
	if !errors.Is(err, errs.ErrNoteNotFound) {
		t.Fatalf("want ErrNoteNotFound, got %v", err)
	}
}

func TestPublishRequiresImage(t *testing.T) {
	_ = snowflake.Init(1)
	svc := note.NewService(newFakeRepo(), fakeStorage{}, nil, nil, NewTestCache(t), nil)
	if _, err := svc.Publish(context.Background(), 7, "t", "c", nil); !errors.Is(err, errs.ErrParam) {
		t.Fatalf("want ErrParam, got %v", err)
	}
}

func TestLatestPagination(t *testing.T) {
	_ = snowflake.Init(1)
	repo := newFakeRepo()
	svc := note.NewService(repo, fakeStorage{}, nil, nil, NewTestCache(t), nil)
	for i := 0; i < 25; i++ {
		if _, err := svc.Publish(context.Background(), 1, "title", "content", []string{"u"}); err != nil {
			t.Fatal(err)
		}
	}
	p1, err := svc.Latest(context.Background(), 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(p1.List) != 10 || !p1.HasMore {
		t.Fatalf("page1: len=%d, HasMore=%v", len(p1.List), p1.HasMore)
	}
	p2, err := svc.Latest(context.Background(), p1.NextCursor, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(p2.List) != 10 || p2.List[0].ID >= p1.NextCursor {
		t.Fatalf("page2 not descending from cursor: %+v", p2.List[0])
	}
	seen := map[int64]bool{}
	for _, d := range append(p1.List, p2.List...) {
		if seen[d.ID] {
			t.Fatalf("duplicate across pages: %d", d.ID)
		}
		seen[d.ID] = true
	}
}
func (f *fakeRepo) ListByUser(_ context.Context, userID, cursor int64, size int) ([]*note.Note, error) {
	var all []*note.Note
	for _, n := range f.byID {
		if n.UserID == userID && n.Status == 0 && (cursor == 0 || n.ID < cursor) {
			all = append(all, n)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].ID > all[j].ID
	})
	if len(all) > size+1 {
		all = all[:size+1]
	}
	return all, nil
}

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

func TestAuthorEnrichment(t *testing.T) {
	_ = snowflake.Init(1)
	repo := newFakeRepo()
	lookup := fakeUserLookup{byID: map[int64]*user.Author{7: {ID: 7, Nickname: "小红", AvatarURL: "http://a"}}}
	svc := note.NewService(repo, fakeStorage{}, nil, nil, NewTestCache(t), lookup)
	n, err := svc.Publish(context.Background(), 7, "标题", "", []string{"u"})
	if err != nil {
		t.Fatal(err)
	}
	dto, err := svc.Detail(context.Background(), n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dto.Author == nil || dto.Author.Nickname != "小红" {
		t.Fatalf("detail author not enriched: %+v", dto.Author)
	}
	p, _ := svc.Latest(context.Background(), 0, 10)
	if len(p.List) != 1 || p.List[0].Author == nil || p.List[0].Author.Nickname != "小红" {
		t.Fatalf("latest author not enriched: %+v", p.List)
	}
}

func TestListByUser(t *testing.T) {
	_ = snowflake.Init(1)
	repo := newFakeRepo()
	svc := note.NewService(repo, fakeStorage{}, nil, nil, NewTestCache(t), nil) // users=nil 跳过回填
	for i := 0; i < 3; i++ {
		if _, err := svc.Publish(context.Background(), 7, "t", "c", []string{"u"}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Publish(context.Background(), 9, "other", "", []string{"u"}); err != nil {
		t.Fatal(err)
	}
	p, err := svc.ListByUser(context.Background(), 7, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.List) != 3 {
		t.Fatalf("want 3 notes for user 7, got %d", len(p.List))
	}
	for _, d := range p.List {
		if d.UserID != 7 {
			t.Fatalf("leaked other user's note: %+v", d)
		}
	}
}
