package interaction

import (
	"context"
	"errors"
	"testing"
	"xbs/internal/pkg/errs"

	mysqlDriver "github.com/go-sql-driver/mysql"
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
func TestFollowTargetNotExists(t *testing.T) {
	repo := newFakeFollowRepo()
	repo.insertErr = &mysqlDriver.MySQLError{Number: 1452} // 外键 violation
	svc := NewService(&Repos{Follow: repo}, nil, nil, nil)
	err := svc.Follow(context.Background(), 1, 999)
	if !errors.Is(err, errs.ErrUserNotFound) {
		t.Fatalf("err=%v, want ErrUserNotFound", err)
	}
}

func TestFollowIdempotent(t *testing.T) {
	repo := newFakeFollowRepo()
	svc := NewService(&Repos{Follow: repo}, nil, nil, nil)
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
