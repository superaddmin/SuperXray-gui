package service

import (
	"strings"
	"testing"
	"time"
)

func TestPublicIPCacheRefreshesAndRetainsLastSuccess(t *testing.T) {
	cache := &publicIPCache{}
	now := time.Now()
	lookup := func(endpoint string) string {
		if strings.Contains(endpoint, "api6") {
			return "2001:db8::1"
		}
		return "192.0.2.1"
	}
	cache.refresh(now, lookup)
	if cache.ipv4 != "192.0.2.1" || cache.ipv6 != "2001:db8::1" || !cache.nextRefresh.Equal(now.Add(5*time.Minute)) {
		t.Fatal("successful lookup was not cached for five minutes")
	}
	cache.refresh(now.Add(5*time.Minute), func(string) string { return "N/A" })
	if cache.ipv4 != "192.0.2.1" || cache.ipv6 != "2001:db8::1" || !cache.nextRefresh.Equal(now.Add(6*time.Minute)) {
		t.Fatal("failed lookup discarded a valid address or disabled retry")
	}
	cache.refresh(now.Add(6*time.Minute), func(endpoint string) string {
		if strings.Contains(endpoint, "api6") {
			return "2001:db8::2"
		}
		return "192.0.2.2"
	})
	if cache.ipv4 != "192.0.2.2" || cache.ipv6 != "2001:db8::2" {
		t.Fatal("changed public addresses stayed stale")
	}
}

func TestPublicIPSnapshotDoesNotBlockOrDuplicateLookups(t *testing.T) {
	cache := &publicIPCache{}
	started, release := make(chan struct{}), make(chan struct{})
	finished := make(chan struct{})
	first := true
	lookup := func(string) string {
		if first {
			first = false
			close(started)
			<-release
		}
		return "N/A"
	}
	defer close(release)
	go func() {
		v4, v6 := cache.snapshot(time.Now(), lookup)
		if v4 != "N/A" || v6 != "N/A" {
			t.Error("cold snapshot should be unavailable")
		}
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("snapshot waited on network")
	}
	<-started
	cache.snapshot(time.Now(), func(string) string { t.Error("duplicate refresh"); return "N/A" })
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if !cache.refreshing {
		t.Fatal("active lookup not tracked")
	}
}

func TestPublicIPLookupRejectsWrongFamiliesAndNonAddresses(t *testing.T) {
	for _, value := range []string{"<html>error</html>", "2001:db8::1", "::ffff:192.0.2.1"} {
		if got := lookupPublicIP([]string{"fixture"}, false, func(string) string { return value }); got != "" {
			t.Fatalf("invalid IPv4 response accepted: %q", value)
		}
	}
}
