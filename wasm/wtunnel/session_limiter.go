package main

import (
	"net"
	"net/netip"
	"sync"
	"time"
)

const (
	defaultMaxSessionsPerIP = 128
	defaultMaxSessions      = 2048
	defaultRateLimitPerIP   = 10.0 // new WebTransport sessions per second per IP
	defaultRateBurstPerIP   = 40.0 // burst allowance for LAN / simultaneous joins
	staleBucketAge          = 5 * time.Minute
	sweepThreshold          = 4096
)

type ipState struct {
	active     int
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

// sessionLimiter enforces concurrent session limits (global and per client IP)
// and a per-IP token-bucket connection rate limit before WebTransport upgrade.
type sessionLimiter struct {
	mu           sync.Mutex
	maxPerIP     int
	maxGlobal    int
	ratePerSec   float64
	burst        float64
	activeGlobal int
	byIP         map[netip.Addr]*ipState
	nowFn        func() time.Time
}

func newSessionLimiter(maxPerIP, maxGlobal int, ratePerSec, burst float64) *sessionLimiter {
	if ratePerSec > 0 && burst < 1 {
		burst = ratePerSec
		if burst < 1 {
			burst = 1
		}
	}
	return &sessionLimiter{
		maxPerIP:   maxPerIP,
		maxGlobal:  maxGlobal,
		ratePerSec: ratePerSec,
		burst:      burst,
		byIP:       make(map[netip.Addr]*ipState),
		nowFn:      time.Now,
	}
}

func extractClientIP(remoteAddr string) (netip.Addr, bool) {
	if ap, err := netip.ParseAddrPort(remoteAddr); err == nil {
		return ap.Addr().Unmap(), true
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	if addr, err := netip.ParseAddr(host); err == nil {
		return addr.Unmap(), true
	}
	return netip.Addr{}, false
}

// Acquire checks rate and concurrency limits for ip. On success it returns a
// release callback that must be called once when the session terminates.
func (l *sessionLimiter) Acquire(ip netip.Addr) (release func(), reason string, ok bool) {
	if l == nil {
		return func() {}, "", true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.nowFn()
	if len(l.byIP) >= sweepThreshold {
		l.sweepLocked(now)
	}

	if l.maxGlobal > 0 && l.activeGlobal >= l.maxGlobal {
		return nil, "global session limit reached", false
	}

	st, exists := l.byIP[ip]
	if !exists {
		st = &ipState{
			tokens:     l.burst,
			lastRefill: now,
			lastSeen:   now,
		}
		l.byIP[ip] = st
	}

	if l.ratePerSec > 0 {
		elapsed := now.Sub(st.lastRefill).Seconds()
		if elapsed > 0 {
			st.tokens += elapsed * l.ratePerSec
			if st.tokens > l.burst {
				st.tokens = l.burst
			}
			st.lastRefill = now
		}
		if st.tokens < 1.0 {
			st.lastSeen = now
			return nil, "per-IP connection rate limit exceeded", false
		}
	}

	if l.maxPerIP > 0 && st.active >= l.maxPerIP {
		st.lastSeen = now
		return nil, "per-IP concurrent session limit reached", false
	}

	if l.ratePerSec > 0 {
		st.tokens -= 1.0
	}
	st.active++
	st.lastSeen = now
	l.activeGlobal++

	var once sync.Once
	return func() {
		once.Do(func() {
			l.release(ip)
		})
	}, "", true
}

func (l *sessionLimiter) release(ip netip.Addr) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.activeGlobal > 0 {
		l.activeGlobal--
	}
	st, ok := l.byIP[ip]
	if !ok {
		return
	}
	if st.active > 0 {
		st.active--
	}
	now := l.nowFn()
	st.lastSeen = now
	// Clean up immediately if no active sessions and rate limiting is disabled.
	if st.active == 0 && l.ratePerSec <= 0 {
		delete(l.byIP, ip)
	}
}

func (l *sessionLimiter) sweepLocked(now time.Time) {
	for ip, st := range l.byIP {
		if st.active == 0 && now.Sub(st.lastSeen) > staleBucketAge {
			delete(l.byIP, ip)
		}
	}
}
