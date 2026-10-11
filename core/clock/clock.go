package clock

import (
	"context"
	"time"
)

// Clock is the time now: time.Now, or a test's. The zero Clock is time.Now.
type Clock func() time.Time

// Now is the clock's time; a nil Clock reads time.Now.
func (c Clock) Now() time.Time {
	if c == nil {
		return time.Now()
	}
	return c()
}

// Sleep waits for d or ctx, whichever ends first, and reports whether d elapsed.
func Sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
