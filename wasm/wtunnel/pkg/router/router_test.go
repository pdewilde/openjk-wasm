package router

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/youdp"
)

func TestGetRouteExhaustionAndReuse(t *testing.T) {
	minIP := binary.BigEndian.Uint32([]byte{100, 64, 0, 1})
	maxIP := binary.BigEndian.Uint32([]byte{100, 64, 0, 2})
	r := NewRouterWithRange(minIP, maxIP)

	var b1, b2, b3 bytes.Buffer
	ip1, err := r.GetRoute(&b1)
	if err != nil {
		t.Fatalf("first GetRoute failed: %v", err)
	}
	ip2, err := r.GetRoute(&b2)
	if err != nil {
		t.Fatalf("second GetRoute failed: %v", err)
	}
	if ip1 == ip2 {
		t.Fatalf("expected distinct IPs, got %v twice", ip1)
	}

	if _, err := r.GetRoute(&b3); !errors.Is(err, ErrIPExhausted) {
		t.Fatalf("expected ErrIPExhausted, got %v", err)
	}

	r.RemoveRoute(ip1)
	ip3, err := r.GetRoute(&b3)
	if err != nil {
		t.Fatalf("GetRoute after RemoveRoute failed: %v", err)
	}
	if ip3 != ip1 {
		t.Fatalf("expected reused IP %v, got %v", ip1, ip3)
	}
}

func TestSendEnforcesSrcIPNoSleepAndTracksSrcPort(t *testing.T) {
	r := NewRouter()
	var buf1, buf2 bytes.Buffer
	ip1, err := r.GetRoute(&buf1)
	if err != nil {
		t.Fatal(err)
	}
	ip2, err := r.GetRoute(&buf2)
	if err != nil {
		t.Fatal(err)
	}

	spoofed := &youdp.Packet{
		DestIp:   ip2,
		SrcIp:    netip.MustParseAddr("1.2.3.4"),
		DestPort: 27960,
		SrcPort:  27965,
		Data:     []byte("ping"),
	}

	start := time.Now()
	r.Send(ip1, spoofed)
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("Send took %v, expected no 100ms sleep", elapsed)
	}

	got, err := youdp.DeserializePacket(buf2.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if got.SrcIp != ip1 {
		t.Fatalf("expected SrcIp %v, got %v", ip1, got.SrcIp)
	}
	if got.SrcPort != 27965 || got.DestPort != 27960 {
		t.Fatalf("unexpected ports: src=%d dst=%d", got.SrcPort, got.DestPort)
	}

	port, ok := r.GetSrcPort(ip1)
	if !ok || port != 27965 {
		t.Fatalf("GetSrcPort(%v) = (%d, %v), want (27965, true)", ip1, port, ok)
	}
}

func TestBroadcastSkipsSender(t *testing.T) {
	r := NewRouter()
	var buf1, buf2, buf3 bytes.Buffer
	ip1, _ := r.GetRoute(&buf1)
	_, _ = r.GetRoute(&buf2)
	_, _ = r.GetRoute(&buf3)

	r.Send(ip1, &youdp.Packet{
		DestIp:   netip.MustParseAddr("255.255.255.255"),
		SrcIp:    ip1,
		DestPort: 27960,
		SrcPort:  27961,
		Data:     []byte("bcast"),
	})

	if buf1.Len() != 0 {
		t.Fatalf("sender should not receive broadcast, got %d bytes", buf1.Len())
	}
	if buf2.Len() == 0 || buf3.Len() == 0 {
		t.Fatalf("expected all other peers to receive broadcast (buf2=%d, buf3=%d)", buf2.Len(), buf3.Len())
	}
}
