package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/youdp"
)

var cgnatPrefix = netip.MustParsePrefix("100.64.0.0/10")

type udpAllowRule struct {
	prefix netip.Prefix
	port   uint16 // 0 means any port
}

type udpAllowlist struct {
	rules []udpAllowRule
}

func parseUDPAllowlist(spec string) (*udpAllowlist, error) {
	al := &udpAllowlist{}
	if strings.TrimSpace(spec) == "" {
		return al, nil
	}
	for _, raw := range strings.Split(spec, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			pfx, err := netip.ParsePrefix(entry)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q: %w", entry, err)
			}
			if !pfx.Addr().Is4() {
				return nil, fmt.Errorf("only IPv4 CIDR supported in -udp-allow: %q", entry)
			}
			al.rules = append(al.rules, udpAllowRule{prefix: pfx})
			continue
		}
		if addr, err := netip.ParseAddr(entry); err == nil {
			if !addr.Is4() {
				return nil, fmt.Errorf("only IPv4 address supported in -udp-allow: %q", entry)
			}
			al.rules = append(al.rules, udpAllowRule{prefix: netip.PrefixFrom(addr, 32)})
			continue
		}
		host, portStr, err := net.SplitHostPort(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid -udp-allow entry %q: %w", entry, err)
		}
		portNum, err := strconv.Atoi(portStr)
		if err != nil || portNum < 1 || portNum > 65535 {
			return nil, fmt.Errorf("invalid port in -udp-allow entry %q", entry)
		}
		if addr, err := netip.ParseAddr(host); err == nil {
			if !addr.Is4() {
				return nil, fmt.Errorf("only IPv4 address supported in -udp-allow: %q", entry)
			}
			al.rules = append(al.rules, udpAllowRule{
				prefix: netip.PrefixFrom(addr, 32),
				port:   uint16(portNum),
			})
			continue
		}
		ips, err := net.LookupIP(host)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve host %q in -udp-allow: %w", host, err)
		}
		added := false
		for _, ip := range ips {
			if ip4 := ip.To4(); ip4 != nil {
				addr := netip.AddrFrom4([4]byte(ip4))
				al.rules = append(al.rules, udpAllowRule{
					prefix: netip.PrefixFrom(addr, 32),
					port:   uint16(portNum),
				})
				added = true
			}
		}
		if !added {
			return nil, fmt.Errorf("no IPv4 address found for host %q in -udp-allow", host)
		}
	}
	return al, nil
}

func (al *udpAllowlist) Enabled() bool {
	return al != nil && len(al.rules) > 0
}

func (al *udpAllowlist) Allows(ip netip.Addr, port uint16) bool {
	if al == nil || !ip.Is4() {
		return false
	}
	for _, r := range al.rules {
		if r.prefix.Contains(ip) && (r.port == 0 || r.port == port) {
			return true
		}
	}
	return false
}

type udpEgress struct {
	vip       netip.Addr
	allowlist *udpAllowlist
	writer    io.Writer

	mu             sync.Mutex
	conn           *net.UDPConn
	closed         bool
	lastClientPort atomic.Uint32
	wg             sync.WaitGroup
}

func newUDPEgress(vip netip.Addr, allowlist *udpAllowlist, writer io.Writer) *udpEgress {
	return &udpEgress{
		vip:       vip,
		allowlist: allowlist,
		writer:    writer,
	}
}

func (e *udpEgress) Send(p *youdp.Packet) {
	if !e.allowlist.Allows(p.DestIp, p.DestPort) {
		return
	}
	if p.SrcPort != 0 {
		e.lastClientPort.Store(uint32(p.SrcPort))
	}

	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return
	}
	if e.conn == nil {
		conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			e.mu.Unlock()
			log.Printf("UDP egress socket open failed for %v: %v", e.vip, err)
			return
		}
		e.conn = conn
		e.wg.Add(1)
		go e.readLoop(conn)
	}
	conn := e.conn
	e.mu.Unlock()

	if _, err := conn.WriteToUDPAddrPort(p.Data, netip.AddrPortFrom(p.DestIp, p.DestPort)); err != nil {
		log.Printf("UDP egress write failed (%v -> %v:%d): %v", e.vip, p.DestIp, p.DestPort, err)
	}
}

func (e *udpEgress) readLoop(conn *net.UDPConn) {
	defer e.wg.Done()
	buf := make([]byte, 65535)
	for {
		n, from, err := conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			return
		}
		fromIP := from.Addr().Unmap()
		if !fromIP.Is4() || !e.allowlist.Allows(fromIP, from.Port()) {
			continue
		}
		pkt := &youdp.Packet{
			DestIp:   e.vip,
			SrcIp:    fromIP,
			DestPort: uint16(e.lastClientPort.Load()),
			SrcPort:  from.Port(),
			Data:     buf[:n],
		}
		if _, err := e.writer.Write(pkt.Serialize()); err != nil {
			return
		}
	}
}

func (e *udpEgress) Close() {
	e.mu.Lock()
	e.closed = true
	conn := e.conn
	e.conn = nil
	e.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	e.wg.Wait()
}
