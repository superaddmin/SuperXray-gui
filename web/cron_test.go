package web

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/superaddmin/SuperXray-gui/v2/database"
)

func TestCronPanicReleasesDatabaseLeaseAndAllowsNextRun(t *testing.T) {
	c := newCronScheduler(time.Local)
	runs := 0
	id, err := c.AddFunc("@every 1s", func() {
		runs++
		if runs == 1 {
			panic("fixture job panic")
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	job := c.Entry(id).WrappedJob
	job.Run()
	job.Run()
	if runs != 2 {
		t.Fatal("panic permanently consumed the non-overlap token")
	}
	done := make(chan struct{})
	go func() {
		release := database.AcquireMaintenance()
		release()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("panic leaked the database read lease")
	}
}

func TestNewCronSchedulerSkipsOverlappingRuns(t *testing.T) {
	c := newCronScheduler(time.Local)
	var runs int32
	var running int32
	var maxRunning int32

	if _, err := c.AddFunc("@every 1s", func() {
		current := atomic.AddInt32(&running, 1)
		for {
			max := atomic.LoadInt32(&maxRunning)
			if current <= max || atomic.CompareAndSwapInt32(&maxRunning, max, current) {
				break
			}
		}
		defer atomic.AddInt32(&running, -1)

		atomic.AddInt32(&runs, 1)
		time.Sleep(1500 * time.Millisecond)
	}); err != nil {
		t.Fatalf("AddFunc returned error: %v", err)
	}

	c.Start()
	time.Sleep(3600 * time.Millisecond)
	stopCtx := c.Stop()

	select {
	case <-stopCtx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("cron did not stop in time")
	}

	if got := atomic.LoadInt32(&runs); got < 2 {
		t.Fatalf("cron runs = %d, want at least 2", got)
	}
	if got := atomic.LoadInt32(&maxRunning); got != 1 {
		t.Fatalf("max concurrent cron runs = %d, want 1", got)
	}
}
