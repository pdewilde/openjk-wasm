// Package youdp is the fake UDP layer we are building that gets put over WebTransport.
package youdp

import (
	"encoding/binary"
	"fmt"
	"net/netip"
)

type Packet struct {
	DestIp netip.Addr // IPv4 Only
	SrcIp netip.Addr // IPv4 Only
	DestPort uint16
	Data []byte
}

// DeserializePacket takes a bytestream and reconstructs it into a Packet Struct.
// Must be at least 10 bytes, 8 for ipv4 addresses and 2 for port.
// Dest Port and DestIp are both in big endian network order.
func DeserializePacket(data []byte) (*Packet, error) {
	// Must be at least 10 bytes for ip + ip + port
	// Data length of zero is valid.
	// Length and Checksum are irrelevant as WebTransprot garuntees those.
	if len(data) < 10 {
		return nil, fmt.Errorf("invalid udp packet, must be at least 10 bytes, but was %d bytes", len(data))
	}
	p := &Packet{
		DestIp:   netip.AddrFrom4([4]byte(data[0:4])),
		SrcIp:    netip.AddrFrom4([4]byte(data[4:8])),
		DestPort: binary.BigEndian.Uint16(data[8:10]),
		Data:     data[10:],
	}
	return p, nil
}

// Serialize takes a Packet and puts it into a byte stream.
// WILL PANIC if p.[DestIp,SrcIp].Is4() == false. I didn't want to add an 
// error to the signature here, so just don't do that.
// Dest Port and DestIp are both in big endian network order.
func (p *Packet) Serialize() []byte {
	buf := make([]byte, 10 + len(p.Data))
	ipBuf := p.DestIp.As4() // Panics if not ipv4
	copy(buf[0:4],  ipBuf[:])
	ipBuf = p.SrcIp.As4() // Panics if not ipv4
	copy(buf[4:8], ipBuf[:])
	binary.BigEndian.PutUint16(buf[8:10], p.DestPort)
	copy(buf[10:], p.Data)
	return buf
}