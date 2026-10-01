package server

import (
	"context"
	"log"
	"sync"
	"time"
)

const orderExpirationInterval = time.Minute

type orderExpirer interface {
	ExpirePendingOrders(now time.Time) (int, error)
}

// backgroundJobs runs periodic tasks until stopped by the graceful shutdown.
type backgroundJobs struct {
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func startBackgroundJobs(orders orderExpirer) *backgroundJobs {
	ctx, cancel := context.WithCancel(context.Background())
	jobs := &backgroundJobs{cancel: cancel}

	jobs.wg.Add(1)
	go func() {
		defer jobs.wg.Done()
		runEvery(ctx, orderExpirationInterval, func() { expireOrders(orders) })
	}()

	return jobs
}

func runEvery(ctx context.Context, interval time.Duration, task func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			task()
		}
	}
}

func expireOrders(orders orderExpirer) {
	expired, err := orders.ExpirePendingOrders(time.Now())
	if err != nil {
		log.Printf("order expiration: %v", err)
		return
	}

	if expired > 0 {
		log.Printf("order expiration: cancelled %d expired order(s)", expired)
	}
}

// Stop signals the jobs and waits for a run in progress to finish.
func (j *backgroundJobs) Stop() {
	j.cancel()
	j.wg.Wait()
}
