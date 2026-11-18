// Package youdp is the fake UDP layer we are building that gets put over WebTransport.
package youdp

import (
	"encoding/binary"
	"fmt"
	"net/netip"
)

// HeaderSize is the size of the fixed youdp header in bytes:
// DestIp(4) + SrcIp(4) + DestPort(2) + SrcPort(2).
const HeaderSize = 12

type Packet struct {
	DestIp   netip.Addr // IPv4 Only
	SrcIp    netip.Addr // IPv4 Only
	DestPort uint16
	SrcPort  uint16
	Data     []byte
}

// DeserializePacket takes a bytestream and reconstructs it into a Packet struct.
// Must be at least 12 bytes: 4 for DestIp, 4 for SrcIp, 2 for DestPort, 2 for SrcPort.
// Ports and IPs are in big-endian network order.
func DeserializePacket(data []byte) (*Packet, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("invalid udp packet, must be at least %d bytes, but was %d bytes", HeaderSize, len(data))
	}
	p := &Packet{
		DestIp:   netip.AddrFrom4([4]byte(data[0:4])),
		SrcIp:    netip.AddrFrom4([4]byte(data[4:8])),
		DestPort: binary.BigEndian.Uint16(data[8:10]),
		SrcPort:  binary.BigEndian.Uint16(data[10:12]),
		Data:     data[HeaderSize:],
	}
	return p, nil
}

// Serialize encodes a Packet into a 12-byte header followed by Data.
// Panics if DestIp or SrcIp is not an IPv4 address.
func (p *Packet) Serialize() []byte {
	buf := make([]byte, HeaderSize+len(p.Data))
	ipBuf := p.DestIp.As4()
	copy(buf[0:4], ipBuf[:])
	ipBuf = p.SrcIp.As4()
	copy(buf[4:8], ipBuf[:])
	binary.BigEndian.PutUint16(buf[8:10], p.DestPort)
	binary.BigEndian.PutUint16(buf[10:12], p.SrcPort)
	copy(buf[HeaderSize:], p.Data)
	return buf
}
