// Package youdp is the fake UDP layer we are building that gets put over WebTransport.
package youdp

import (
	"encoding/binary"
	"fmt"
	"net/netip"
)

type Packet struct {
	DestIp netip.Addr // IPv4 Only
	DestPort uint16
	Data []byte
}

// ParsePacket takes a bytestream and reconstructs it into a Packet Struct.
// Must be at least 6 bytes, 4 for ipv4 address and 2 for port.
// Dest Port and DestIp are both in big endian network order.
func DeseralizePacket(data []byte) (*Packet, error) {
	// Must be at least 6 bytes for ip + port
	// Data length of zero is valid.
	// Length and Checksum are irrelevant as WebTransprot garuntees those.
	if len(data) < 6 {
		return nil, fmt.Errorf("invalid udp packet, must be at least 6 bytes, but was %d bytes", len(data))
	}
	p := &Packet{
		DestIp:   netip.AddrFrom4([4]byte(data[0:4])),
		DestPort: binary.BigEndian.Uint16(data[4:6]),
		Data:     data[6:],
	}
	return p, nil
}

// Seralize takes a Packet and puts it into a byte stream.
// WILL PANIC if p.DestIp.Is4() == false. I didn't want to add an 
// error to the signature here, so just don't do that.
// Dest Port and DestIp are both in big endian network order.
func (p *Packet) Seralize() []byte {
	buf := make([]byte, 6 + len(p.Data))
	ipBuf := p.DestIp.As4() // Panics if not ipv4
	copy(buf[0:4],  ipBuf[:])
	binary.BigEndian.PutUint16(buf[4:6], p.DestPort)
	copy(buf[6:], p.Data)
	return buf
}