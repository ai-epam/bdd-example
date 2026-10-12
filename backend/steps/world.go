// Package steps holds the godog step definitions. They are domain-level:
// they call authz.Authorize directly against an in-memory store.
package steps

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"authgate/internal/authz"
	"authgate/internal/store"
)

// epoch is the frozen "now" for every scenario, so time-based rules are deterministic.
var epoch = time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

// world is the per-scenario state, carried through context.Context.
type world struct {
	store  *store.Store
	last   authz.Result
	ran    bool
	before map[string]int64 // account balance just before the last authorization
}

type worldKey struct{}

func newWorld() *world {
	w := &world{store: store.New(), before: map[string]int64{}}
	w.store.SetClock(func() time.Time { return epoch })
	return w
}

func from(ctx context.Context) *world {
	w, ok := ctx.Value(worldKey{}).(*world)
	if !ok {
		panic("scenario world missing from context")
	}
	return w
}

// parseMoney converts "12.50" into 1250 cents without floats.
func parseMoney(s string) (int64, error) {
	whole, frac, _ := strings.Cut(strings.TrimSpace(s), ".")
	if len(frac) > 2 {
		return 0, fmt.Errorf("invalid amount %q: more than 2 decimals", s)
	}
	frac += strings.Repeat("0", 2-len(frac))
	w, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q", s)
	}
	f, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q", s)
	}
	return w*100 + f, nil
}
