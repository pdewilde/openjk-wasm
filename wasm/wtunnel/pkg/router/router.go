package router

import (
	"encoding/binary"
	"io"
	"log"
	"net/netip"
	"sync"
	"time"

	"github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/youdp"
)

var maxIP = binary.BigEndian.Uint32([]byte{100, 127, 255, 255})
var minIP = binary.BigEndian.Uint32([]byte{100, 64, 0, 0})
var ipv4Broadcast = netip.AddrFrom4([4]byte{255, 255, 255, 255})

// Router assigns ip addresses to WebTransport clients.
// Currently just assigns sequential IP addresses with
// no expiration. Rolls back to zero after exhausting block.
// I'm using the CGNAT ip block 100.64.0.0/10.
// This is ~4 million addresses from 100.64.0.0 to 100.127.255.255.
type Router struct {
	rLock    sync.RWMutex
	nextAddr uint32
	routes   map[netip.Addr]io.Writer
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[netip.Addr]io.Writer),
	}
}

func (r *Router) Send(sender netip.Addr, p *youdp.Packet) {
	if sender != p.SrcIp {
		log.Printf("%v is being naughty, prentending to be %v", sender, p.SrcIp)
		// Put them in timeout
		time.Sleep(100 * time.Millisecond)
		p.SrcIp = sender
	}
	r.rLock.RLock()
	defer r.rLock.RUnlock()
	dest := p.DestIp

	if ipv4Broadcast == dest {
		// Broadcast to all
		packetBuf := p.Serialize()
		for ip, w := range r.routes {
			if ip != sender {
				if _, err := w.Write(packetBuf); err != nil {
					// This should never run due to our async writer interface
					log.Printf("Error broadcasting to ip %v: %v", ip, err)
				}
			}
		}
	} else if w, ok := r.routes[dest]; ok {
		if _, err := w.Write(p.Serialize()); err != nil {
			// This should never run due to our async writer interface
			log.Printf("Error writing to ip %v: %v", dest, err)
		}
	} else {
		log.Printf("Dropped packet from %v to %v: dest had no route", sender, dest)
	}
}

func (r *Router) GetRoute(w io.Writer) netip.Addr {
	r.rLock.Lock()
	defer r.rLock.Unlock()
	ip := r.assignIP()
	addr := ipToAddr(ip)
	r.routes[addr] = w
	return addr
}

func (r *Router) RemoveRoute(ip netip.Addr) {
	r.rLock.Lock()
	defer r.rLock.Unlock()
	delete(r.routes, ip)
}

// TODO: make this less dumb
// MUST hold lock to call
func (r *Router) assignIP() uint32 {
	if r.rLock.TryLock() {
		r.rLock.Unlock()
		panic("assignIP called without holding lock")

	}
	if uint32(len(r.routes)) >= maxIP-minIP {
		// TODO don't panic, return error or something
		panic("IP address space exhausted")
	}
	for {
		if r.nextAddr < minIP {
			r.nextAddr = minIP
		}
		ip := r.nextAddr
		if r.nextAddr >= maxIP {
			log.Printf("WARNING: Rolling back to minIP: %d", minIP)
			r.nextAddr = minIP
		} else {
			r.nextAddr++
		}
		if _, ok := r.routes[ipToAddr(ip)]; !ok {
			return ip
		}
	}
}

func ipToAddr(ip uint32) netip.Addr {
	var ipBytes [4]byte
	binary.BigEndian.PutUint32(ipBytes[:], ip)
	return netip.AddrFrom4(ipBytes)
}
