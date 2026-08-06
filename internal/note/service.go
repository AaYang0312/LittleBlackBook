package note

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path"
	"time"
	"xbs/internal/pkg/counter"
	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/mq"
	"xbs/internal/pkg/snowflake"
	"xbs/internal/pkg/storage"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Service interface {
	Publish(ctx context.Context, userID int64, title, content string, images []string) (*Note, error)
	Detail(ctx context.Context, id int64) (*NoteDTO, error)
	BatchByIDs(ctx context.Context, ids []int64) ([]*NoteDTO, error)
	Latest(ctx context.Context, cursor int64, size int) (*Page, error)
	UploadImage(ctx context.Context, userID int64, reader io.Reader, size int64, filename string) (string, error)
	Delete(ctx context.Context, id, userID int64) error
	AddCountDelta(ctx context.Context, id int64, field string, delta int) error
	ListAllIDs(ctx context.Context) ([]int64, error)
	SetCounts(ctx context.Context, id int64, like, collect, comment int64) error
}
type service struct {
	repo Repository
	st   storage.Storage
	m    *mq.MQ
	rdb  *redis.Client
}

func NewService(repo Repository, st storage.Storage, m *mq.MQ, rdb *redis.Client) Service {
	return &service{repo: repo, st: st, m: m, rdb: rdb}
}
func (s *service) Publish(ctx context.Context, userID int64, title, content string, images []string) (*Note, error) {
	if title == "" || len(images) == 0 {
		return nil, errs.ErrParam
	}
	imgJSON, _ := json.Marshal(images)
	n := &Note{
		ID: snowflake.NextID(), UserID: userID, Title: title, Content: content,
		CoverURL: images[0], Images: string(imgJSON),
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	if s.m != nil {
		ev := mq.FanoutEvent{NoteID: n.ID, AuthorID: userID, Ts: time.Now().UnixMilli()}
		if err := s.m.Publish(ctx, mq.QueueFeedFanout, ev); err != nil {
			slog.Error("fanout publish failed", "note_id", n.ID, "err", err) // 不阻塞主流程
		}
	}
	return n, nil
}
func toDTO(n *Note) *NoteDTO {
	var imgs []string
	_ = json.Unmarshal([]byte(n.Images), &imgs)
	return &NoteDTO{
		ID: n.ID, UserID: n.UserID, Title: n.Title, Content: n.Content, CoverURL: n.CoverURL,
		Images: imgs, LikeCount: n.LikeCount, CollectCount: n.CollectCount,
		CommentCount: n.CommentCount, CreatedAt: n.CreatedAt,
	}
}

// overlayCounts 用 Redis 实时计数覆盖 DB 旧值；Redis 未初始化（-1）保留 DB 值。
func (s *service) overlayCounts(ctx context.Context, dto *NoteDTO) {
	if s.rdb == nil {
		return
	}
	if v, err := counter.Get(ctx, s.rdb, counter.KindLike, dto.ID); err == nil && v >= 0 {
		dto.LikeCount = v
	}
	if v, err := counter.Get(ctx, s.rdb, counter.KindCollect, dto.ID); err == nil && v >= 0 {
		dto.CollectCount = v
	}
	if v, err := counter.Get(ctx, s.rdb, counter.KindComment, dto.ID); err == nil && v >= 0 {
		dto.CommentCount = v
	}
}
func (s *service) Detail(ctx context.Context, id int64) (*NoteDTO, error) {
	n, err := s.repo.FindByID(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errs.ErrNoteNotFound
	}
	if err != nil {
		return nil, err
	}
	dto := toDTO(n)
	s.overlayCounts(ctx, dto)
	return dto, nil
}
func (s *service) BatchByIDs(ctx context.Context, ids []int64) ([]*NoteDTO, error) {
	ns, err := s.repo.BatchFindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]*NoteDTO, 0, len(ns))
	for _, n := range ns {
		dto := toDTO(n)
		s.overlayCounts(ctx, dto)
		out = append(out, dto)
	}
	return out, nil
}
func (s *service) Latest(ctx context.Context, cursor int64, size int) (*Page, error) {
	if size <= 0 || size > 50 {
		size = 20
	}
	ns, err := s.repo.ListLatest(ctx, cursor, size)
	if err != nil {
		return nil, err
	}
	p := &Page{}
	if len(ns) > size {
		p.HasMore = true
		ns = ns[:size]
	}
	for _, n := range ns {
		dto := toDTO(n)
		s.overlayCounts(ctx, dto)
		p.List = append(p.List, dto)
	}
	if len(p.List) > 0 {
		p.NextCursor = p.List[len(p.List)-1].ID
	}
	return p, nil
}
func (s *service) UploadImage(ctx context.Context, userID int64, reader io.Reader, size int64, filename string) (string, error) {
	if size <= 0 || size > 10<<20 {
		return "", errs.ErrParam
	}
	objectName := fmt.Sprintf("%d/%d%s", userID, snowflake.NextID(), path.Ext(filename))
	return s.st.Upload(ctx, reader, size, objectName, "image/jpeg")
}
func (s *service) Delete(ctx context.Context, id, userID int64) error {
	if err := s.repo.SoftDelete(ctx, id, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.ErrForbidden
		}
		return err
	}
	return nil
}
func (s *service) AddCountDelta(ctx context.Context, id int64, field string, delta int) error {
	return s.repo.AddCountDelta(ctx, id, field, delta)
}
func (s *service) ListAllIDs(ctx context.Context) ([]int64, error) { return s.repo.ListAllIDs(ctx) }
func (s *service) SetCounts(ctx context.Context, id int64, like, collect, comment int64) error {
	return s.repo.SetCounts(ctx, id, like, collect, comment)
}
