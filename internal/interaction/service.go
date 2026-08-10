package interaction

import (
	"context"
	"errors"
	"xbs/internal/note"
	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/mq"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

type Service interface {
	Follow(ctx context.Context, me, target int64) error
	Unfollow(ctx context.Context, me, target int64) error
	FollowerIDs(ctx context.Context, authorID int64) ([]int64, error)

	//Like(ctx context.Context, userID, noteID int64) error
	//Unlike(ctx context.Context, userID, noteID int64) error
	//Collect(ctx context.Context, userID, noteID int64) error
	//Uncollect(ctx context.Context, userID, noteID int64) error
	//ApplyLikeEvent(ctx context.Context, ev mq.LikeEvent) error
	//
	//CreateComment(ctx context.Context, userID, noteID int64, content string) (*Comment, error)
	//ListComments(ctx context.Context, noteID, cursor int64, size int) ([]*Comment, error)
	//
	//RebuildCounts(ctx context.Context) error
}
type service struct {
	repos   *Repos
	rdb     *redis.Client
	m       *mq.MQ
	noteSvc note.Service
}

func NewService(repos *Repos, rdb *redis.Client, m *mq.MQ, noteSvc note.Service) Service {
	return &service{repos: repos, rdb: rdb, m: m, noteSvc: noteSvc}
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
