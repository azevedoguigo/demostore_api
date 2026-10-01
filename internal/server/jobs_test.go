package server

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunEvery_RunsUntilCancelled(t *testing.T) {
	var runs atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		runEvery(ctx, 5*time.Millisecond, func() { runs.Add(1) })
		close(done)
	}()

	assert.Eventually(t, func() bool { return runs.Load() >= 2 }, time.Second, time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runEvery did not stop after cancellation")
	}
}

type fakeExpirer struct{ calls atomic.Int32 }

func (f *fakeExpirer) ExpirePendingOrders(time.Time) (int, error) {
	f.calls.Add(1)
	return 1, nil
}

func TestBackgroundJobs_StopReturns(t *testing.T) {
	jobs := startBackgroundJobs(&fakeExpirer{})

	stopped := make(chan struct{})
	go func() {
		jobs.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return")
	}
}

func TestExpireOrders_CallsService(t *testing.T) {
	expirer := &fakeExpirer{}

	expireOrders(expirer)

	assert.Equal(t, int32(1), expirer.calls.Load())
}
