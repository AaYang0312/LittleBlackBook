package cache_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"xbs/internal/pkg/db"
	"xbs/internal/pkg/errs"

	"github.com/alicebob/miniredis/v2"
	"github.com/rogpeppe/go-internal/cache"
)

func newMini(t *testing.T) *cache.Cache {
	mr := miniredis.RunT(t)
	return cache.New(db.NewRedis(mr.Addr(), "", 0))
}
func TestGetOrLoadSingleflight(t *testing.T) {
	c := newMini(t)
	var calls atomic.Int64
	load := func(context.Context) (string, error) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		return "v", nil
	}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := c.GetorLoad(context.Background(), "k1", load)
			if err != nil {
				t.Errorf("got %q, %v", v, err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("singleflight broken, load called %d times", calls.Load())
	}
}
func TestGetOrLoadNotFoundMarker(t *testing.T) {
	c := newMini(t)
	var calls atomic.Int64
	load := func(context.Context) (string, error) {
		calls.Add(1)
		return "", errs.ErrNoteNotFound
	}
	for i := 0; i < 3; i++ {
		_, err := c.GetOrLoad(context.Background(), "k2", time.Minute, load)
		if !errors.Is(err, errs.ErrNoteNotFound) {
			t.Fatalf("want ErrNoteNotFound, got %v", err)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("notfound marker broken, load called %d times", calls.Load())
	}
}
