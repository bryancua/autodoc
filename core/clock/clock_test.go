package clock_test

import (
	"context"
	"testing"
	"time"

	"github.com/yongjohnlee80/autodoc/core/clock"
)

func TestAClockReadsItsTime(t *testing.T) {
	at := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	if got := clock.Clock(func() time.Time { return at }).Now(); !got.Equal(at) {
		t.Errorf("a test's clock reads %v, want %v", got, at)
	}
	before := time.Now()
	if got := clock.Clock(nil).Now(); got.Before(before) || time.Since(got) > time.Minute {
		t.Errorf("the zero clock reads %v, want time.Now", got)
	}
}

func TestSleepEndsAtTheDurationOrTheContext(t *testing.T) {
	if !clock.Sleep(context.Background(), time.Millisecond) {
		t.Error("a sleep its context outlives reports that it elapsed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if clock.Sleep(ctx, time.Hour) {
		t.Error("a sleep whose context ended reports that it elapsed")
	}
	if time.Since(start) > 10*time.Second {
		t.Error("a sleep whose context ended still waited")
	}
}
