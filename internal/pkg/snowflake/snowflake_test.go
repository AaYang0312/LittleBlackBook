package snowflake

import (
	"testing"
)

func TestNextIDUnique(t *testing.T) {
	if err := Init(1); err != nil {
		t.Fatal(err)
	}
	seen := make(map[int64]struct{}, 10000)
	for i := 0; i < 10000; i++ {
		id := NextID()
		if id <= 0 {
			t.Fatalf("invalid id %d", id)
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = struct{}{}
	}
}
