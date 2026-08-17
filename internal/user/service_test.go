package user

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
	"xbs/internal/pkg/errs"
	"xbs/internal/pkg/snowflake"

	"gorm.io/gorm"
)

type fakeRepo struct {
	byName map[string]*User
	byID   map[int64]*User
	nextID int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byName: map[string]*User{}, byID: map[int64]*User{}, nextID: 1}
}
func (f *fakeRepo) Create(_ context.Context, u *User) error {
	u.ID = f.nextID
	f.nextID++
	f.byName[u.Username] = u
	f.byID[u.ID] = u
	return nil
}
func (f *fakeRepo) FindByUsername(_ context.Context, name string) (*User, error) {
	if u, ok := f.byName[name]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (f *fakeRepo) FindByID(_ context.Context, id int64) (*User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func TestRegisterDuplicateUsername(t *testing.T) {
	svc := NewService(newFakeRepo(), nil, "secret", time.Hour)
	if _, err := svc.Register(context.Background(), "alice", "123456", "Alice"); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Register(context.Background(), "alice", "654321", "Alice2")
	if !errors.Is(err, errs.ErrUsernameExists) {
		t.Fatalf("want ErrUsernameExists, got %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc := NewService(newFakeRepo(), nil, "secret", time.Hour)
	if _, err := svc.Register(context.Background(), "bob", "123456", "Bob"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(context.Background(), "bob", "wrong"); !errors.Is(err, errs.ErrWrongPassword) {
		t.Fatalf("want ErrWrongPassword, got %v", err)
	}
	token, err := svc.Login(context.Background(), "bob", "123456")
	if err != nil || token == "" {
		t.Fatalf("want token, got err=%v", err)
	}
}
func (f *fakeRepo) BatchFindByIDs(_ context.Context, ids []int64) ([]*User, error) {
	var out []*User
	for _, id := range ids {
		if u, ok := f.byID[id]; ok {
			out = append(out, u)
		}
	}
	return out, nil
}

func TestBatchByIDs(t *testing.T) {
	svc := NewService(newFakeRepo(), nil, "secret", time.Hour)
	u1, _ := svc.Register(context.Background(), "alice", "123456", "Alice")
	u2, _ := svc.Register(context.Background(), "bob", "123456", "Bob")
	got, err := svc.BatchByIDs(context.Background(), []int64{u1.ID, u2.ID, 999})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[u1.ID].Nickname != "Alice" {
		t.Fatalf("bad batch: %+v", got)
	}
	if m, err := svc.BatchByIDs(context.Background(), nil); err != nil || len(m) != 0 {
		t.Fatalf("empty batch err=%v len=%d", err, len(m))
	}
}

func (f *fakeRepo) Patch(_ context.Context, id int64, fields map[string]any) (*User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, errs.ErrUserNotFound
	}
	if v, ok := fields["nickname"]; ok {
		u.Nickname = v.(string)
	}
	if v, ok := fields["bio"]; ok {
		u.Bio = v.(string)
	}
	if v, ok := fields["avatar_url"]; ok {
		u.AvatarURL = v.(string)
	}
	return u, nil
}

type fakeStorage struct{}

func (fakeStorage) Upload(_ context.Context, _ io.Reader, _ int64, name, _ string) (string, error) {
	return "http://minio/" + name, nil
}
func TestUpdateProfile(t *testing.T) {
	svc := NewService(newFakeRepo(), fakeStorage{}, "secret", time.Hour)
	u, _ := svc.Register(context.Background(), "alice", "123456", "Alice")
	nick := "AliceNew"
	bio := "hello"
	got, err := svc.UpdateProfile(context.Background(), u.ID, &nick, &bio, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Nickname != "AliceNew" || got.Bio != "hello" || got.AvatarURL != "" {
		t.Fatalf("bad update: %+v", got)
	}
	if _, err := svc.UpdateProfile(context.Background(), u.ID, nil, nil, nil); !errors.Is(err, errs.ErrParam) {
		t.Fatalf("want ErrParam, got %v", err)
	}
	empty := ""
	got2, _ := svc.UpdateProfile(context.Background(), u.ID, &empty, nil, nil)
	if got2.Nickname != "" {
		t.Fatalf("nickname should be cleared, got %q", got2.Nickname)
	}
}

func TestUploadAvatar(t *testing.T) {
	_ = snowflake.Init(1)
	svc := NewService(newFakeRepo(), fakeStorage{}, "secret", time.Hour)
	url, err := svc.UploadAvatar(context.Background(), 7, strings.NewReader("x"), 3, "a.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "http://minio/avatars/7/") {
		t.Fatalf("bad avatar url: %s", url)
	}
	if _, err := svc.UploadAvatar(context.Background(), 7, strings.NewReader("x"), 11<<20, "a.jpg"); !errors.Is(err, errs.ErrParam) {
		t.Fatalf("want ErrParam for oversize, got %v", err)
	}
}
