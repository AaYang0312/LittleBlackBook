package user

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"time"
	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/snowflake"
	"xbs/internal/pkg/storage"

	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service interface {
	Register(ctx context.Context, username, password, nickname string) (*User, error)
	Login(ctx context.Context, username, password string) (string, error)
	Profile(ctx context.Context, id int64) (*User, error)
	BatchByIDs(ctx context.Context, ids []int64) (map[int64]*Author, error)
	UpdateProfile(ctx context.Context, id int64, nickname, bio, avatarURL *string) (*User, error)
	UploadAvatar(ctx context.Context, UserID int64, reader io.Reader, size int64, filename string) (string, error)
}

type service struct {
	repo   Repository
	st     storage.Storage
	secret string
	expire time.Duration
}

func NewService(repo Repository, st storage.Storage, jwtSecret string, expire time.Duration) Service {
	return &service{
		repo:   repo,
		st:     st,
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
func (s *service) BatchByIDs(ctx context.Context, ids []int64) (map[int64]*Author, error) {
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
func (s *service) UpdateProfile(ctx context.Context, id int64, nickname, bio, avatarURL *string) (*User, error) {
	if nickname == nil && bio == nil && avatarURL == nil {
		return nil, errs.ErrParam
	}
	fields := map[string]any{}
	if nickname != nil {
		fields["nickname"] = *nickname
	}
	if bio != nil {
		fields["bio"] = *bio
	}
	if avatarURL != nil {
		fields["avatar_url"] = *avatarURL
	}
	return s.repo.Patch(ctx, id, fields)
}
func (s *service) UploadAvatar(ctx context.Context, userID int64, reader io.Reader, size int64, filename string) (string, error) {
	if s.st == nil {
		return "", errs.ErrInternal
	}
	if size <= 0 || size > 10<<20 {
		return "", errs.ErrParam
	}
	objectName := fmt.Sprintf("avatars/%d/%d%s", userID, snowflake.NextID(), path.Ext(filename))
	return s.st.Upload(ctx, reader, size, objectName, "image/jpeg")
}
