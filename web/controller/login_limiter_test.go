package controller

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestLoginLimiterIncludesConcurrentChecksAndExpiresFailures(t *testing.T) {
	var limiter loginLimiter
	now := time.Now()
	attempts := make(chan *loginAttempt, 50)
	var wg sync.WaitGroup
	for range cap(attempts) {
		wg.Go(func() {
			if attempt := limiter.reserve("peer", "admin", now); attempt != nil {
				attempts <- attempt
			}
		})
	}
	wg.Wait()
	close(attempts)
	if len(attempts) != loginUserLimit {
		t.Fatalf("admitted %d simultaneous checks", len(attempts))
	}
	for attempt := range attempts {
		limiter.finish("peer", attempt, false, now)
	}
	if limiter.reserve("peer", " Admin ", now) != nil {
		t.Fatal("normalized username evaded the per-account limit")
	}
	for i := loginUserLimit; i < loginIPLimit; i++ {
		attempt := limiter.reserve("peer", fmt.Sprint(i), now)
		if attempt == nil {
			t.Fatal("address limited prematurely")
		}
		limiter.finish("peer", attempt, false, now)
	}
	if limiter.reserve("peer", "other", now) != nil || limiter.reserve("other-peer", "admin", now) == nil {
		t.Fatal("address-wide limit must apply to this peer only")
	}
	for range 2 * loginIPLimit {
		later := now.Add(loginFailureWindow)
		attempt := limiter.reserve("peer", "admin", later)
		if attempt == nil {
			t.Fatal("expired failures or successful logins still count")
		}
		limiter.finish("peer", attempt, true, later)
	}
	if _, ok := limiter.byIP["peer"]; ok {
		t.Fatal("idle peer still tracked")
	}
}
