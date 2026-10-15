package database

import (
	"testing"
	"time"
)

func TestMaintenanceWaitsForActiveReaders(t *testing.T) {
	release := AcquireReadLease()
	acquired := make(chan struct{})
	done := make(chan struct{})
	go func() {
		unlock := AcquireMaintenance()
		close(acquired)
		<-done
		unlock()
	}()
	select {
	case <-acquired:
		t.Fatal("maintenance bypassed active request")
	case <-time.After(20 * time.Millisecond):
	}
	release()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("maintenance did not proceed after request finished")
	}
	close(done)
}
