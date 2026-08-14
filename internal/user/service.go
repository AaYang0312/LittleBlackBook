package user

import (
	"context"
	"errors"
	"time"
	"xbs/internal/pkg/errs"

	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service interface {
	Register(ctx context.Context, username, password, nickname string) (*User, error)
	Login(ctx context.Context, username, password string) (string, error)
	Profile(ctx context.Context, id int64) (*User, error)
	BatchFindByIDs(ctx context.Context, ids []int64) (map[int64]*Author, error)
}

type service struct {
	repo   Repository
	secret string
	expire time.Duration
}

func NewService(repo Repository, jwtSecret string, expire time.Duration) Service {
	return &service{
		repo:   repo,
		secret: jwtSecret,
		expire: expire,
	}
}

func (s *service) Register(ctx context.Context, username, password, nickname string) (*User, error) {
	if username == "" || len(password) < 6 {
		return nil, errs.ErrParam
	}
	if _, err := s.repo.FindByUsername(ctx, username); err == nil {
		return nil, errs.ErrUsernameExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &User{Username: username, PasswordHash: string(hash), Nickname: nickname}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}
func (s *service) Login(ctx context.Context, username, password string) (string, error) {
	u, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return "", errs.ErrWrongPassword
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", errs.ErrWrongPassword
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": u.ID,
		"exp": time.Now().Add(s.expire).Unix(),
	})
	return t.SignedString([]byte(s.secret))
}
func (s *service) Profile(ctx context.Context, id int64) (*User, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, errs.ErrUnauthorized
	}
	return u, nil
}
func (s *service) BatchFindByIDs(ctx context.Context, ids []int64) (map[int64]*Author, error) {
	out := make(map[int64]*Author)
	if len(ids) == 0 {
		return out, nil
	}
	usrs, err := s.repo.BatchFindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, u := range usrs {
		out[u.ID] = &Author{
			ID:        u.ID,
			Nickname:  u.Nickname,
			AvatarURL: u.AvatarURL,
		}
	}
	return out, nil
}
