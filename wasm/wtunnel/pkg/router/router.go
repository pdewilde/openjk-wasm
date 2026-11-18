package router

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net/netip"
	"sync"

	"github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/youdp"
)

var (
	defaultMaxIP  = binary.BigEndian.Uint32([]byte{100, 127, 255, 255})
	defaultMinIP  = binary.BigEndian.Uint32([]byte{100, 64, 0, 0})
	ipv4Broadcast = netip.AddrFrom4([4]byte{255, 255, 255, 255})

	// ErrIPExhausted is returned by GetRoute when no virtual IPs remain.
	ErrIPExhausted = errors.New("IP address space exhausted")
)

type routeEntry struct {
	writer  io.Writer
	srcPort uint16
}

// Router assigns virtual IPv4 addresses to WebTransport clients and forwards
// youdp packets between them. By default it uses CGNAT 100.64.0.0/10.
type Router struct {
	rLock    sync.RWMutex
	minIP    uint32
	maxIP    uint32
	nextAddr uint32
	routes   map[netip.Addr]*routeEntry
}

func NewRouter() *Router {
	return NewRouterWithRange(defaultMinIP, defaultMaxIP)
}

func NewRouterWithRange(minIP, maxIP uint32) *Router {
	return &Router{
		minIP:    minIP,
		maxIP:    maxIP,
		nextAddr: minIP,
		routes:   make(map[netip.Addr]*routeEntry),
	}
}

func (r *Router) Send(sender netip.Addr, p *youdp.Packet) {
	if sender != p.SrcIp {
		p.SrcIp = sender
	}

	r.rLock.RLock()
	if entry, ok := r.routes[sender]; ok && p.SrcPort != 0 && entry.srcPort != p.SrcPort {
		// Upgrade to write lock only when SrcPort changes.
		r.rLock.RUnlock()
		r.rLock.Lock()
		if entry, ok := r.routes[sender]; ok {
			entry.srcPort = p.SrcPort
		}
		r.rLock.Unlock()
		r.rLock.RLock()
	}
	defer r.rLock.RUnlock()

	dest := p.DestIp
	packetBuf := p.Serialize()
	if ipv4Broadcast == dest {
		for ip, entry := range r.routes {
			if ip != sender {
				if _, err := entry.writer.Write(packetBuf); err != nil {
					log.Printf("Error broadcasting to ip %v: %v", ip, err)
				}
			}
		}
	} else if entry, ok := r.routes[dest]; ok {
		if _, err := entry.writer.Write(packetBuf); err != nil {
			log.Printf("Error writing to ip %v: %v", dest, err)
		}
	}
}

func (r *Router) GetRoute(w io.Writer) (netip.Addr, error) {
	r.rLock.Lock()
	defer r.rLock.Unlock()
	ip, err := r.assignIP()
	if err != nil {
		return netip.Addr{}, err
	}
	addr := ipToAddr(ip)
	r.routes[addr] = &routeEntry{writer: w}
	return addr, nil
}

func (r *Router) RemoveRoute(ip netip.Addr) {
	r.rLock.Lock()
	defer r.rLock.Unlock()
	delete(r.routes, ip)
}

// GetSrcPort returns the most recently observed source port for ip.
func (r *Router) GetSrcPort(ip netip.Addr) (uint16, bool) {
	r.rLock.RLock()
	defer r.rLock.RUnlock()
	entry, ok := r.routes[ip]
	if !ok {
		return 0, false
	}
	return entry.srcPort, true
}

// assignIP finds the next available IP. Caller must hold r.rLock.
func (r *Router) assignIP() (uint32, error) {
	if r.maxIP < r.minIP {
		return 0, ErrIPExhausted
	}
	total := uint64(r.maxIP-r.minIP) + 1
	if uint64(len(r.routes)) >= total {
		return 0, ErrIPExhausted
	}
	for checked := uint64(0); checked < total; checked++ {
		if r.nextAddr < r.minIP || r.nextAddr > r.maxIP {
			r.nextAddr = r.minIP
		}
		ip := r.nextAddr
		if r.nextAddr == r.maxIP {
			r.nextAddr = r.minIP
		} else {
			r.nextAddr++
		}
		if _, ok := r.routes[ipToAddr(ip)]; !ok {
			return ip, nil
		}
	}
	return 0, ErrIPExhausted
}

func ipToAddr(ip uint32) netip.Addr {
	var ipBytes [4]byte
	binary.BigEndian.PutUint32(ipBytes[:], ip)
	return netip.AddrFrom4(ipBytes)
}
