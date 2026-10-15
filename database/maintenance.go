package database

import "sync"

// Access leases let request and scheduled-job owners finish before a live
// database is replaced. Lock order is database lease, then core maintenance.
var accessLease sync.RWMutex

func AcquireReadLease() func() {
	accessLease.RLock()
	return accessLease.RUnlock
}

func AcquireMaintenance() func() {
	accessLease.Lock()
	return accessLease.Unlock
}
