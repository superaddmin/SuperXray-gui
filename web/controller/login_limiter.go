package controller

import (
	"crypto/sha256"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	loginFailureWindow = 5 * time.Minute
	loginUserLimit     = 5
	loginIPLimit       = 25
	loginMaxIPs        = 10000
)

// An attempt occupies its slot until authentication succeeds or a failure expires.
type loginAttempt struct {
	username [sha256.Size]byte
	expires  time.Time
}

type loginLimiter struct {
	mu        sync.Mutex
	byIP      map[string][]*loginAttempt
	nextSweep time.Time
}

func (l *loginLimiter) reserve(ip, username string, now time.Time) *loginAttempt {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.byIP == nil {
		l.byIP = make(map[string][]*loginAttempt)
	}
	prune := func(entries []*loginAttempt) []*loginAttempt {
		return slices.DeleteFunc(entries, func(e *loginAttempt) bool {
			return !e.expires.IsZero() && !now.Before(e.expires)
		})
	}
	if !now.Before(l.nextSweep) {
		for key, entries := range l.byIP {
			if entries = prune(entries); len(entries) == 0 {
				delete(l.byIP, key)
			} else {
				l.byIP[key] = entries
			}
		}
		l.nextSweep = now.Add(time.Minute)
	}
	entries, exists := l.byIP[ip]
	if !exists && len(l.byIP) >= loginMaxIPs {
		return nil
	}
	entries = prune(entries)
	l.byIP[ip] = entries
	if len(entries) >= loginIPLimit {
		return nil
	}
	usernameKey := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(username))))
	count := 0
	for _, entry := range entries {
		if entry.username == usernameKey {
			count++
		}
	}
	if count >= loginUserLimit {
		return nil
	}
	attempt := &loginAttempt{username: usernameKey}
	l.byIP[ip] = append(entries, attempt)
	return attempt
}

func (l *loginLimiter) finish(ip string, attempt *loginAttempt, success bool, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !success {
		attempt.expires = now.Add(loginFailureWindow)
		return
	}
	entries := slices.DeleteFunc(l.byIP[ip], func(e *loginAttempt) bool { return e == attempt })
	if len(entries) == 0 {
		delete(l.byIP, ip)
	} else {
		l.byIP[ip] = entries
	}
}
