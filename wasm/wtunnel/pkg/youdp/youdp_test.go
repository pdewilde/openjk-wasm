package youdp

import (
	"bytes"
	"net/netip"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	orig := &Packet{
		DestIp:   netip.MustParseAddr("100.64.0.2"),
		SrcIp:    netip.MustParseAddr("100.64.0.1"),
		DestPort: 27960,
		SrcPort:  27961,
		Data:     []byte("hello openjk"),
	}

	raw := orig.Serialize()
	if len(raw) != HeaderSize+len(orig.Data) {
		t.Fatalf("expected serialized length %d, got %d", HeaderSize+len(orig.Data), len(raw))
	}

	got, err := DeserializePacket(raw)
	if err != nil {
		t.Fatalf("DeserializePacket failed: %v", err)
	}
	if got.DestIp != orig.DestIp || got.SrcIp != orig.SrcIp {
		t.Fatalf("IP mismatch: got (%v, %v), want (%v, %v)", got.DestIp, got.SrcIp, orig.DestIp, orig.SrcIp)
	}
	if got.DestPort != orig.DestPort || got.SrcPort != orig.SrcPort {
		t.Fatalf("port mismatch: got (%d, %d), want (%d, %d)", got.DestPort, got.SrcPort, orig.DestPort, orig.SrcPort)
	}
	if !bytes.Equal(got.Data, orig.Data) {
		t.Fatalf("payload mismatch: got %q, want %q", got.Data, orig.Data)
	}
}

func TestEmptyPayload(t *testing.T) {
	orig := &Packet{
		DestIp:   netip.MustParseAddr("255.255.255.255"),
		SrcIp:    netip.MustParseAddr("100.64.0.5"),
		DestPort: 27960,
		SrcPort:  12345,
	}
	raw := orig.Serialize()
	if len(raw) != HeaderSize {
		t.Fatalf("expected %d bytes, got %d", HeaderSize, len(raw))
	}
	got, err := DeserializePacket(raw)
	if err != nil {
		t.Fatalf("DeserializePacket failed: %v", err)
	}
	if len(got.Data) != 0 || got.SrcPort != 12345 {
		t.Fatalf("unexpected decoded packet: %+v", got)
	}
}

func TestTooShort(t *testing.T) {
	for i := 0; i < HeaderSize; i++ {
		if _, err := DeserializePacket(make([]byte, i)); err == nil {
			t.Fatalf("expected error for %d-byte packet, got nil", i)
		}
	}
}
