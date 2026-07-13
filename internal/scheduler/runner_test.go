package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/imposter-project/imposter-go/internal/config"
	"github.com/stretchr/testify/require"
)

func TestTriggerFunc(t *testing.T) {
	t.Run("every yields fixed interval", func(t *testing.T) {
		next, err := TriggerFunc(&config.Schedule{Every: "30s"})
		require.NoError(t, err)

		now := time.Now()
		require.Equal(t, now.Add(30*time.Second), next(now))
	})

	t.Run("cron yields next matching time", func(t *testing.T) {
		next, err := TriggerFunc(&config.Schedule{Cron: "0 * * * *"})
		require.NoError(t, err)

		now := time.Date(2026, 1, 1, 10, 30, 0, 0, time.UTC)
		require.Equal(t, time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC), next(now))
	})

	t.Run("invalid duration errors", func(t *testing.T) {
		_, err := TriggerFunc(&config.Schedule{Every: "bogus"})
		require.Error(t, err)
	})
}

func TestRunSchedule(t *testing.T) {
	t.Run("fires repeatedly until cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var count atomic.Int32
		fired := make(chan struct{}, 16)
		next := func(after time.Time) time.Time { return after.Add(10 * time.Millisecond) }

		done := make(chan struct{})
		go func() {
			defer close(done)
			RunSchedule(ctx, "test", next, 0, func() {
				count.Add(1)
				fired <- struct{}{}
			})
		}()

		// Wait for at least two firings
		for i := 0; i < 2; i++ {
			select {
			case <-fired:
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for schedule to fire")
			}
		}

		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("runner did not stop after cancellation")
		}

		require.GreaterOrEqual(t, count.Load(), int32(2))
	})

	t.Run("stops without firing when cancelled early", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		var count atomic.Int32
		next := func(after time.Time) time.Time { return after.Add(time.Hour) }

		done := make(chan struct{})
		go func() {
			defer close(done)
			RunSchedule(ctx, "test", next, 0, func() { count.Add(1) })
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("runner did not stop after cancellation")
		}
		require.Equal(t, int32(0), count.Load())
	})

	t.Run("stops after reaching the firing limit", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var count atomic.Int32
		next := func(after time.Time) time.Time { return after.Add(5 * time.Millisecond) }

		done := make(chan struct{})
		go func() {
			defer close(done)
			RunSchedule(ctx, "test", next, 3, func() { count.Add(1) })
		}()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("runner did not stop after reaching its limit")
		}
		require.Equal(t, int32(3), count.Load())
	})
}

func TestEffectiveLimit(t *testing.T) {
	t.Run("explicit limit wins", func(t *testing.T) {
		t.Setenv("IMPOSTER_SCHEDULE_LIMIT", "5")
		require.Equal(t, 10, EffectiveLimit(&config.Schedule{Limit: 10}))
	})

	t.Run("global default applies when unset", func(t *testing.T) {
		t.Setenv("IMPOSTER_SCHEDULE_LIMIT", "5")
		require.Equal(t, 5, EffectiveLimit(&config.Schedule{}))
	})

	t.Run("unlimited when neither is set", func(t *testing.T) {
		require.Equal(t, 0, EffectiveLimit(&config.Schedule{}))
	})

	t.Run("invalid global default is ignored", func(t *testing.T) {
		t.Setenv("IMPOSTER_SCHEDULE_LIMIT", "nonsense")
		require.Equal(t, 0, EffectiveLimit(&config.Schedule{}))
	})
}
