package interaction

import (
	"context"
	"errors"
	"xbs/internal/pkg/counter"
	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/mq"
	"xbs/internal/pkg/snowflake"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

type Service interface {
	Follow(ctx context.Context, me, target int64) error
	Unfollow(ctx context.Context, me, target int64) error
	FollowerIDs(ctx context.Context, authorID int64) ([]int64, error)

	Like(ctx context.Context, userID, noteID int64) error
	Unlike(ctx context.Context, userID, noteID int64) error
	Collect(ctx context.Context, userID, noteID int64) error
	Uncollect(ctx context.Context, userID, noteID int64) error

	ApplyLikeEvent(ctx context.Context, ev mq.LikeEvent) error
	//
	CreateComment(ctx context.Context, userID, noteID int64, content string) (*Comment, error)
	ListComments(ctx context.Context, noteID, cursor int64, size int) ([]*Comment, error)
	//
	//RebuildCounts(ctx context.Context) error
}
type service struct {
	repos   *Repos
	rdb     *redis.Client
	m       *mq.MQ
	noteSvc noteCounter
	publish func(ctx context.Context, ev mq.LikeEvent) error
}
type noteCounter interface {
	AddCountDelta(ctx context.Context, id int64, field string, delta int) error
	ListAllIDs(ctx context.Context) ([]int64, error)
	SetCounts(ctx context.Context, id int64, like, collect, comment int64) error
}

func NewService(repos *Repos, rdb *redis.Client, m *mq.MQ, noteSvc noteCounter) Service {
	svc := &service{repos: repos, rdb: rdb, m: m, noteSvc: noteSvc}
	svc.publish = func(ctx context.Context, ev mq.LikeEvent) error {
		if m == nil {
			return nil
		}
		return m.Publish(ctx, mq.QueueLikeEvent, ev)
	}
	return svc
}

// NewServiceForTest 注入 publish 便于单测
func NewServiceForTest(repos *Repos, rdb *redis.Client, publish func(ctx context.Context, ev mq.LikeEvent) error, noteSvc noteCounter) Service {
	return &service{repos: repos, rdb: rdb, publish: publish, noteSvc: noteSvc}
}
func (s *service) Follow(ctx context.Context, me, target int64) error {
	if me == target || me == 0 || target == 0 {
		return errs.ErrParam
	}
	if err := s.repos.Follow.InsertIgnore(ctx, me, target); err != nil {
		// 外键 violation:目标用户(或自己)不存在
		var sqlErr *mysqlDriver.MySQLError
		if errors.As(err, &sqlErr) && sqlErr.Number == 1452 {
			return errs.ErrUserNotFound
		}
		return err
	}
	return nil
}
func (s *service) Unfollow(ctx context.Context, me, target int64) error {
	return s.repos.Follow.Delete(ctx, me, target)
}

func (s *service) FollowerIDs(ctx context.Context, authorID int64) ([]int64, error) {
	return s.repos.Follow.FollowerIDs(ctx, authorID)
}

func (s *service) Like(ctx context.Context, userID, noteID int64) error {
	return s.react(ctx, counter.KindLike, userID, noteID, 1)
}
func (s *service) Unlike(ctx context.Context, userID, noteID int64) error {
	return s.react(ctx, counter.KindLike, userID, noteID, -1)
}
func (s *service) Collect(ctx context.Context, userID, noteID int64) error {
	return s.react(ctx, counter.KindCollect, userID, noteID, 1)
}
func (s *service) Uncollect(ctx context.Context, userID, noteID int64) error {
	return s.react(ctx, counter.KindCollect, userID, noteID, -1)
}

// react：SADD/SREM 判重（第一层幂等）→ INCR/DECR 实时计数 → 发 MQ 异步落库。
func (s *service) react(ctx context.Context, kind string, userID, noteID int64, delta int) error {
	var changed int64
	var err error
	if delta > 0 {
		changed, err = s.rdb.SAdd(ctx, counter.UsersKey(kind, noteID), userID).Result()
	} else {
		changed, err = s.rdb.SRem(ctx, counter.UsersKey(kind, noteID), userID).Result()
	}
	if err != nil {
		return err
	}
	if changed == 0 {
		return nil // 幂等：重复操作直接成功
	}
	if err := s.rdb.IncrBy(ctx, counter.CountKey(kind, noteID), int64(delta)).Err(); err != nil {
		return err
	}
	return s.publish(ctx, mq.LikeEvent{Kind: kind, NoteID: noteID, UserID: userID, Delta: delta})
}
func (s *service) ApplyLikeEvent(c context.Context, ev mq.LikeEvent) error {
	var repo interface {
		InsertIgnore(ctx context.Context, userID, noteID int64) (bool, error)
		Delete(ctx context.Context, userID, noteID int64) (bool, error)
	}
	var field string
	switch ev.Kind {
	case counter.KindLike:
		repo, field = s.repos.Like, "like_count"
	case counter.KindCollect:
		repo, field = s.repos.Collect, "collect_count"
	default:
		return errs.ErrParam
	}
	if ev.Delta > 0 {
		created, err := repo.InsertIgnore(c, ev.UserID, ev.NoteID)
		if err != nil {
			return err
		}
		if !created {
			return nil //重复事件，跳过
		}
		return s.noteSvc.AddCountDelta(c, ev.NoteID, field, 1)
	}
	deleted, err := repo.Delete(c, ev.UserID, ev.NoteID)
	if err != nil {
		return err
	}
	if !deleted {
		return nil
	}
	return s.noteSvc.AddCountDelta(c, ev.NoteID, field, -1)
}
func (s *service) CreateComment(ctx context.Context, userID, noteID int64, content string) (*Comment, error) {
	if content == "" {
		return nil, errs.ErrParam
	}
	c := &Comment{
		ID:      snowflake.NextID(),
		NoteID:  noteID,
		UserID:  userID,
		Content: content,
	}
	// 写为低频操作直接入库
	if err := s.repos.Comment.Create(ctx, c); err != nil {
		return nil, err
	}
	// 原子更新 db 计数
	if err := s.noteSvc.AddCountDelta(ctx, noteID, "comment_count", 1); err != nil {
		return nil, err
	}
	// 更新 redis
	if s.rdb != nil {
		_ = s.rdb.Incr(ctx, counter.CountKey(counter.KindComment, noteID)).Err()
	}
	return c, nil
}

func (s *service) ListComments(ctx context.Context, noteID, cursor int64, size int) ([]*Comment, error) {
	if size <= 0 || size > 50 {
		size = 20
	}
	return s.repos.Comment.ListByNote(ctx, noteID, cursor, size)
}
