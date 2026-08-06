package note_test

import (
	"context"
	"errors"
	"io"
	"testing"

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
func (f *fakeRepo) ListLatest(context.Context, int64, int) ([]*note.Note, error) { return nil, nil }
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

func TestPublishAndDetail(t *testing.T) {
	_ = snowflake.Init(1)
	repo := newFakeRepo()
	svc := note.NewService(repo, fakeStorage{}, nil, nil) // mq=nil 时跳过 fanout 发布；rdb=nil 时跳过实时计数覆盖
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
	svc := note.NewService(newFakeRepo(), fakeStorage{}, nil, nil)
	_, err := svc.Detail(context.Background(), 999)
	if !errors.Is(err, errs.ErrNoteNotFound) {
		t.Fatalf("want ErrNoteNotFound, got %v", err)
	}
}

func TestPublishRequiresImage(t *testing.T) {
	_ = snowflake.Init(1)
	svc := note.NewService(newFakeRepo(), fakeStorage{}, nil, nil)
	if _, err := svc.Publish(context.Background(), 7, "t", "c", nil); !errors.Is(err, errs.ErrParam) {
		t.Fatalf("want ErrParam, got %v", err)
	}
}
