package main

import (
	"net/netip"
	"testing"
	"time"
)

func TestSessionLimiterPerIPCapAndRelease(t *testing.T) {
	lim := newSessionLimiter(2, 10, 0, 0)
	ip := netip.MustParseAddr("192.0.2.10")

	rel1, _, ok := lim.Acquire(ip)
	if !ok {
		t.Fatal("first Acquire failed")
	}
	rel2, _, ok := lim.Acquire(ip)
	if !ok {
		t.Fatal("second Acquire failed")
	}
	if _, reason, ok := lim.Acquire(ip); ok || reason == "" {
		t.Fatalf("expected third Acquire to fail per-IP cap, got ok=%v reason=%q", ok, reason)
	}

	// Releasing one slot allows a new session (and double-release is a safe no-op).
	rel1()
	rel1()
	rel3, _, ok := lim.Acquire(ip)
	if !ok {
		t.Fatal("Acquire after release failed")
	}
	rel2()
	rel3()

	if lim.activeGlobal != 0 {
		t.Fatalf("expected activeGlobal=0, got %d", lim.activeGlobal)
	}
}

func TestSessionLimiterGlobalCap(t *testing.T) {
	lim := newSessionLimiter(128, 2, 0, 0)
	ip1 := netip.MustParseAddr("192.0.2.1")
	ip2 := netip.MustParseAddr("192.0.2.2")
	ip3 := netip.MustParseAddr("192.0.2.3")

	r1, _, ok1 := lim.Acquire(ip1)
	r2, _, ok2 := lim.Acquire(ip2)
	if !ok1 || !ok2 {
		t.Fatal("expected first two global acquires to succeed")
	}
	if _, _, ok3 := lim.Acquire(ip3); ok3 {
		t.Fatal("expected third acquire to fail global cap")
	}
	r1()
	r3, _, ok3 := lim.Acquire(ip3)
	if !ok3 {
		t.Fatal("expected acquire after global release to succeed")
	}
	r2()
	r3()
}

func TestSessionLimiterRateLimitAndRefill(t *testing.T) {
	now := time.Unix(1000, 0)
	lim := newSessionLimiter(128, 2048, 2.0, 3.0)
	lim.nowFn = func() time.Time { return now }
	ip := netip.MustParseAddr("198.51.100.7")

	// Burst of 3 succeeds immediately.
	for i := 0; i < 3; i++ {
		rel, _, ok := lim.Acquire(ip)
		if !ok {
			t.Fatalf("burst acquire %d failed", i)
		}
		rel()
	}

	// 4th immediate attempt is rate-limited.
	if _, _, ok := lim.Acquire(ip); ok {
		t.Fatal("expected 4th immediate acquire to be rate limited")
	}

	// Advance 0.5s -> 1 token refilled (rate 2/s).
	now = now.Add(500 * time.Millisecond)
	rel, _, ok := lim.Acquire(ip)
	if !ok {
		t.Fatal("expected acquire after 500ms refill to succeed")
	}
	rel()

	// Next immediate attempt is rate-limited again.
	if _, _, ok := lim.Acquire(ip); ok {
		t.Fatal("expected acquire right after refilled token to be rate limited")
	}
}

func TestExtractClientIPNormalizesMappedV6(t *testing.T) {
	ip, ok := extractClientIP("[::ffff:127.0.0.1]:54321")
	if !ok || !ip.Is4() || ip.String() != "127.0.0.1" {
		t.Fatalf("unexpected extracted IP: ok=%v ip=%v", ok, ip)
	}
}
