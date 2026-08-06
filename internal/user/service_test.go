package user

import (
	"context"
	"errors"
	"testing"
	"time"
	"xbs/internal/pkg/errs"

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
	svc := NewService(newFakeRepo(), "secret", time.Hour)
	if _, err := svc.Register(context.Background(), "alice", "123456", "Alice"); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Register(context.Background(), "alice", "654321", "Alice2")
	if !errors.Is(err, errs.ErrUsernameExists) {
		t.Fatalf("want ErrUsernameExists, got %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	svc := NewService(newFakeRepo(), "secret", time.Hour)
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
