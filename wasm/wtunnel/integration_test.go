package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync"
	"testing"
	"time"

	ietfsucks "github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/ietf-sucks"
	"github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/router"
	"github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/util"
	"github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/youdp"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

func readClientVIP(ctx context.Context, sess *webtransport.Session) (netip.Addr, error) {
	stream, err := sess.AcceptUniStream(ctx)
	if err != nil {
		return netip.Addr{}, err
	}
	var vip [4]byte
	if _, err := io.ReadFull(stream, vip[:]); err != nil {
		return netip.Addr{}, err
	}
	return netip.AddrFrom4(vip), nil
}

func TestWebTransportHybridRelayIntegration(t *testing.T) {
	mux := http.NewServeMux()
	cm := &ietfsucks.CertManager{}
	tlsConfig := &tls.Config{
		GetCertificate: cm.GetCert,
	}

	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP failed: %v", err)
	}
	defer udpConn.Close()
	port := udpConn.LocalAddr().(*net.UDPAddr).Port

	ht3Server := http3.Server{
		Handler:         mux,
		Addr:            fmt.Sprintf("127.0.0.1:%d", port),
		TLSConfig:       http3.ConfigureTLSConfig(tlsConfig),
		EnableDatagrams: true,
		QUICConfig: &quic.Config{
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}
	webtransport.ConfigureHTTP3Server(&ht3Server)

	wtServer := webtransport.Server{
		H3:          &ht3Server,
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	defer wtServer.Close()

	virtualRouter := router.NewRouter()
	mux.HandleFunc("/jamp", func(w http.ResponseWriter, r *http.Request) {
		sess, err := wtServer.Upgrade(w, r)
		if err != nil {
			return
		}
		handleWebTransportSession(sess, virtualRouter, defaultMaxDatagramSize, nil)
	})

	go func() {
		_ = wtServer.Serve(udpConn)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://127.0.0.1:%d/jamp", port)
	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		QUICConfig: &quic.Config{
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}

	_, sessA, err := dialer.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("client A dial failed: %v", err)
	}
	defer sessA.CloseWithError(0, "")

	_, sessB, err := dialer.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("client B dial failed: %v", err)
	}
	defer sessB.CloseWithError(0, "")

	vipA, err := readClientVIP(ctx, sessA)
	if err != nil {
		t.Fatalf("client A read VIP failed: %v", err)
	}
	vipB, err := readClientVIP(ctx, sessB)
	if err != nil {
		t.Fatalf("client B read VIP failed: %v", err)
	}
	if vipA == vipB {
		t.Fatalf("expected distinct VIPs, got %v for both", vipA)
	}

	// 1. Small packet (<= maxDatagramSize) via WebTransport Datagram
	smallPayload := []byte("small-openjk-packet")
	smallPkt := (&youdp.Packet{
		DestIp:   vipB,
		SrcIp:    vipA,
		DestPort: 27960,
		SrcPort:  27961,
		Data:     smallPayload,
	}).Serialize()

	if err := sessA.SendDatagram(smallPkt); err != nil {
		t.Fatalf("client A SendDatagram failed: %v", err)
	}

	gotDgram, err := sessB.ReceiveDatagram(ctx)
	if err != nil {
		t.Fatalf("client B ReceiveDatagram failed: %v", err)
	}
	decodedSmall, err := youdp.DeserializePacket(gotDgram)
	if err != nil {
		t.Fatalf("DeserializePacket(dgram) failed: %v", err)
	}
	if decodedSmall.SrcIp != vipA || decodedSmall.SrcPort != 27961 || decodedSmall.DestPort != 27960 || !bytes.Equal(decodedSmall.Data, smallPayload) {
		t.Fatalf("small packet mismatch: %+v", decodedSmall)
	}

	// 2. Large 1400-byte OpenJK packet (> maxDatagramSize) via framed Unidirectional Stream
	largePayload := bytes.Repeat([]byte{0xAB}, 1400)
	largePkt := (&youdp.Packet{
		DestIp:   vipB,
		SrcIp:    vipA,
		DestPort: 27960,
		SrcPort:  27961,
		Data:     largePayload,
	}).Serialize()

	outStreamA, err := sessA.OpenUniStreamSync(ctx)
	if err != nil {
		t.Fatalf("client A OpenUniStreamSync failed: %v", err)
	}
	frame := make([]byte, 2+len(largePkt))
	binary.BigEndian.PutUint16(frame[0:2], uint16(len(largePkt)))
	copy(frame[2:], largePkt)
	if _, err := outStreamA.Write(frame); err != nil {
		t.Fatalf("client A stream Write failed: %v", err)
	}

	inStreamB, err := sessB.AcceptUniStream(ctx)
	if err != nil {
		t.Fatalf("client B AcceptUniStream failed: %v", err)
	}
	var lenBuf [2]byte
	if _, err := io.ReadFull(inStreamB, lenBuf[:]); err != nil {
		t.Fatalf("client B read frame length failed: %v", err)
	}
	frameLen := int(binary.BigEndian.Uint16(lenBuf[:]))
	recvBuf := make([]byte, frameLen)
	if _, err := io.ReadFull(inStreamB, recvBuf); err != nil {
		t.Fatalf("client B read frame body failed: %v", err)
	}
	decodedLarge, err := youdp.DeserializePacket(recvBuf)
	if err != nil {
		t.Fatalf("DeserializePacket(stream) failed: %v", err)
	}
	if decodedLarge.SrcIp != vipA || decodedLarge.SrcPort != 27961 || decodedLarge.DestPort != 27960 || !bytes.Equal(decodedLarge.Data, largePayload) {
		t.Fatalf("large packet mismatch: len=%d src=%v:%d", len(decodedLarge.Data), decodedLarge.SrcIp, decodedLarge.SrcPort)
	}
}

func TestConfigHelpers(t *testing.T) {
	if !isOriginAllowed("*", "https://example.com") {
		t.Fatal("expected * to allow any origin")
	}
	if !isOriginAllowed("https://a.example, https://b.example", "https://b.example") {
		t.Fatal("expected origin in allowlist to pass")
	}
	if isOriginAllowed("https://a.example", "https://evil.example") {
		t.Fatal("expected unlisted origin to be rejected")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if !checkBasicAuth(req, "", "") {
		t.Fatal("expected empty auth config to disable basic auth")
	}
	if checkBasicAuth(req, "user", "pass") {
		t.Fatal("expected missing credentials to fail")
	}
	req.SetBasicAuth("user", "pass")
	if !checkBasicAuth(req, "user", "pass") {
		t.Fatal("expected valid credentials to pass")
	}
	req.SetBasicAuth("user", "wrong")
	if checkBasicAuth(req, "user", "pass") {
		t.Fatal("expected invalid password to fail")
	}

	ht3 := &http3.Server{}
	rec := httptest.NewRecorder()
	setAltSvcHeader(rec, req, ht3, "0.0.0.0:8443")
	if got := rec.Header().Get("Alt-Svc"); got != `h3=":8443"; ma=2592000` {
		t.Fatalf("Alt-Svc = %q, want %q", got, `h3=":8443"; ma=2592000`)
	}
}

func TestRealUDPAndWebTransportE2E(t *testing.T) {
	// 1. Start a real mock UDP OpenJK server on 127.0.0.1:0
	udpServerConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("mock UDP server ListenUDP failed: %v", err)
	}
	defer udpServerConn.Close()
	serverUDPAddr := udpServerConn.LocalAddr().(*net.UDPAddr)

	expectedOOBQuery := []byte("\xff\xff\xff\xffgetinfo e2etest")
	expectedOOBReply := []byte("\xff\xff\xff\xffinfoResponse\n\\hostname\\OpenJK-WASM-E2E\\mapname\\mp/ffa3")
	expectedLargeReply := make([]byte, 1400)
	copy(expectedLargeReply[:4], []byte{0x00, 0x00, 0x00, 0x01})
	for i := 4; i < len(expectedLargeReply); i++ {
		expectedLargeReply[i] = byte((i * 31) & 0xFF)
	}

	serverErrCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 2048)
		for {
			n, remoteAddr, err := udpServerConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			if !bytes.Equal(buf[:n], expectedOOBQuery) {
				serverErrCh <- fmt.Errorf("unexpected UDP query %q", buf[:n])
				return
			}
			if _, err := udpServerConn.WriteToUDP(expectedOOBReply, remoteAddr); err != nil {
				serverErrCh <- err
				return
			}
			if _, err := udpServerConn.WriteToUDP(expectedLargeReply, remoteAddr); err != nil {
				serverErrCh <- err
				return
			}
		}
	}()

	// 2. Start wtunnel WebTransport server
	mux := http.NewServeMux()
	cm := &ietfsucks.CertManager{}
	tlsConfig := &tls.Config{GetCertificate: cm.GetCert}

	wtUDPConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("wtunnel ListenUDP failed: %v", err)
	}
	defer wtUDPConn.Close()
	wtPort := wtUDPConn.LocalAddr().(*net.UDPAddr).Port

	ht3Server := http3.Server{
		Handler:         mux,
		Addr:            fmt.Sprintf("127.0.0.1:%d", wtPort),
		TLSConfig:       http3.ConfigureTLSConfig(tlsConfig),
		EnableDatagrams: true,
		QUICConfig: &quic.Config{
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}
	webtransport.ConfigureHTTP3Server(&ht3Server)

	wtServer := webtransport.Server{
		H3:          &ht3Server,
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	defer wtServer.Close()

	virtualRouter := router.NewRouter()
	mux.HandleFunc("/jamp", func(w http.ResponseWriter, r *http.Request) {
		sess, err := wtServer.Upgrade(w, r)
		if err != nil {
			return
		}
		handleWebTransportSession(sess, virtualRouter, defaultMaxDatagramSize, nil)
	})
	go func() {
		_ = wtServer.Serve(wtUDPConn)
	}()

	// 3. Connect a real UDP bridge route on virtualRouter <-> mock UDP server
	bridgeConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("bridge ListenUDP failed: %v", err)
	}
	defer bridgeConn.Close()

	var bridgeMu sync.Mutex
	var serverVIP netip.Addr
	var lastClientVIP netip.Addr
	var lastClientPort uint16
	bridgeWriter := util.WriterFunc(func(raw []byte) (int, error) {
		pkt, err := youdp.DeserializePacket(raw)
		if err != nil {
			return 0, err
		}
		if pkt.DestPort != uint16(serverUDPAddr.Port) {
			return 0, fmt.Errorf("unexpected DestPort %d, want %d", pkt.DestPort, serverUDPAddr.Port)
		}
		bridgeMu.Lock()
		lastClientVIP = pkt.SrcIp
		lastClientPort = pkt.SrcPort
		bridgeMu.Unlock()
		return bridgeConn.WriteToUDP(pkt.Data, serverUDPAddr)
	})
	serverVIP, err = virtualRouter.GetRoute(bridgeWriter)
	if err != nil {
		t.Fatalf("GetRoute for UDP bridge failed: %v", err)
	}
	defer virtualRouter.RemoveRoute(serverVIP)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, fromAddr, err := bridgeConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			bridgeMu.Lock()
			destVIP := lastClientVIP
			destPort := lastClientPort
			bridgeMu.Unlock()
			respPkt := &youdp.Packet{
				DestIp:   destVIP,
				SrcIp:    serverVIP,
				DestPort: destPort,
				SrcPort:  uint16(fromAddr.Port),
				Data:     append([]byte(nil), buf[:n]...),
			}
			virtualRouter.Send(serverVIP, respPkt)
		}
	}()

	// 4. Connect WebTransport client and send getinfo + verify hybrid responses
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		QUICConfig: &quic.Config{
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}
	_, sess, err := dialer.Dial(ctx, fmt.Sprintf("https://127.0.0.1:%d/jamp", wtPort), nil)
	if err != nil {
		t.Fatalf("WebTransport Dial failed: %v", err)
	}
	defer sess.CloseWithError(0, "")

	vipClient, err := readClientVIP(ctx, sess)
	if err != nil {
		t.Fatalf("readClientVIP failed: %v", err)
	}

	const clientSrcPort uint16 = 27961
	queryPkt := (&youdp.Packet{
		DestIp:   serverVIP,
		SrcIp:    vipClient,
		DestPort: uint16(serverUDPAddr.Port),
		SrcPort:  clientSrcPort,
		Data:     expectedOOBQuery,
	}).Serialize()
	if err := sess.SendDatagram(queryPkt); err != nil {
		t.Fatalf("SendDatagram(getinfo) failed: %v", err)
	}

	// Expect small OOB infoResponse via WebTransport Datagram
	gotDgram, err := sess.ReceiveDatagram(ctx)
	if err != nil {
		t.Fatalf("ReceiveDatagram(infoResponse) failed: %v", err)
	}
	oobResp, err := youdp.DeserializePacket(gotDgram)
	if err != nil {
		t.Fatalf("DeserializePacket(infoResponse) failed: %v", err)
	}
	if oobResp.SrcIp != serverVIP || oobResp.SrcPort != uint16(serverUDPAddr.Port) ||
		oobResp.DestIp != vipClient || oobResp.DestPort != clientSrcPort ||
		!bytes.Equal(oobResp.Data, expectedOOBReply) {
		t.Fatalf("OOB infoResponse mismatch: got src=%v:%d dst=%v:%d len=%d",
			oobResp.SrcIp, oobResp.SrcPort, oobResp.DestIp, oobResp.DestPort, len(oobResp.Data))
	}

	// Expect 1400-byte response via framed WebTransport Unidirectional Stream
	inStream, err := sess.AcceptUniStream(ctx)
	if err != nil {
		t.Fatalf("AcceptUniStream(1400B) failed: %v", err)
	}
	var lenBuf [2]byte
	if _, err := io.ReadFull(inStream, lenBuf[:]); err != nil {
		t.Fatalf("ReadFull(frameLen) failed: %v", err)
	}
	frameLen := int(binary.BigEndian.Uint16(lenBuf[:]))
	frameBuf := make([]byte, frameLen)
	if _, err := io.ReadFull(inStream, frameBuf); err != nil {
		t.Fatalf("ReadFull(frameBody) failed: %v", err)
	}
	largeResp, err := youdp.DeserializePacket(frameBuf)
	if err != nil {
		t.Fatalf("DeserializePacket(1400B) failed: %v", err)
	}
	if largeResp.SrcIp != serverVIP || largeResp.SrcPort != uint16(serverUDPAddr.Port) ||
		largeResp.DestIp != vipClient || largeResp.DestPort != clientSrcPort ||
		!bytes.Equal(largeResp.Data, expectedLargeReply) {
		t.Fatalf("1400B response mismatch: got src=%v:%d dst=%v:%d len=%d",
			largeResp.SrcIp, largeResp.SrcPort, largeResp.DestIp, largeResp.DestPort, len(largeResp.Data))
	}

	select {
	case err := <-serverErrCh:
		t.Fatalf("mock UDP server error: %v", err)
	default:
	}
}

func TestBuiltInUDPEgressAllowlistE2E(t *testing.T) {
	// 1. Real mock UDP OpenJK server on 127.0.0.1:0
	udpServerConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("mock UDP server ListenUDP failed: %v", err)
	}
	defer udpServerConn.Close()
	serverUDPAddr := udpServerConn.LocalAddr().(*net.UDPAddr)
	serverIP := netip.AddrFrom4([4]byte{127, 0, 0, 1})

	// Secondary disallowed UDP server to confirm SSRF allowlist blocks it
	blockedConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("blocked UDP server ListenUDP failed: %v", err)
	}
	defer blockedConn.Close()
	blockedPort := uint16(blockedConn.LocalAddr().(*net.UDPAddr).Port)
	blockedHit := make(chan struct{}, 1)
	go func() {
		buf := make([]byte, 256)
		if _, _, err := blockedConn.ReadFromUDP(buf); err == nil {
			select {
			case blockedHit <- struct{}{}:
			default:
			}
		}
	}()

	expectedOOBQuery := []byte("\xff\xff\xff\xffgetinfo builtin")
	expectedOOBReply := []byte("\xff\xff\xff\xffinfoResponse\n\\hostname\\Builtin-Egress-Test")
	expectedLargeReply := make([]byte, 1400)
	copy(expectedLargeReply[:4], []byte{0x00, 0x00, 0x00, 0x02})
	for i := 4; i < len(expectedLargeReply); i++ {
		expectedLargeReply[i] = byte((i * 17) & 0xFF)
	}

	go func() {
		buf := make([]byte, 2048)
		for {
			n, remoteAddr, err := udpServerConn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			if bytes.Equal(buf[:n], expectedOOBQuery) {
				_, _ = udpServerConn.WriteToUDP(expectedOOBReply, remoteAddr)
				_, _ = udpServerConn.WriteToUDP(expectedLargeReply, remoteAddr)
			}
		}
	}()

	// 2. Parse -udp-allow for only serverUDPAddr
	allow, err := parseUDPAllowlist(fmt.Sprintf("127.0.0.1:%d", serverUDPAddr.Port))
	if err != nil {
		t.Fatalf("parseUDPAllowlist failed: %v", err)
	}
	if !allow.Allows(serverIP, uint16(serverUDPAddr.Port)) {
		t.Fatalf("expected allowlist to permit 127.0.0.1:%d", serverUDPAddr.Port)
	}
	if allow.Allows(serverIP, blockedPort) {
		t.Fatalf("expected allowlist to block 127.0.0.1:%d", blockedPort)
	}

	// 3. Start wtunnel with built-in UDP egress
	mux := http.NewServeMux()
	cm := &ietfsucks.CertManager{}
	tlsConfig := &tls.Config{GetCertificate: cm.GetCert}

	wtUDPConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("wtunnel ListenUDP failed: %v", err)
	}
	defer wtUDPConn.Close()
	wtPort := wtUDPConn.LocalAddr().(*net.UDPAddr).Port

	ht3Server := http3.Server{
		Handler:         mux,
		Addr:            fmt.Sprintf("127.0.0.1:%d", wtPort),
		TLSConfig:       http3.ConfigureTLSConfig(tlsConfig),
		EnableDatagrams: true,
		QUICConfig: &quic.Config{
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}
	webtransport.ConfigureHTTP3Server(&ht3Server)

	wtServer := webtransport.Server{
		H3:          &ht3Server,
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	defer wtServer.Close()

	virtualRouter := router.NewRouter()
	mux.HandleFunc("/jamp", func(w http.ResponseWriter, r *http.Request) {
		sess, err := wtServer.Upgrade(w, r)
		if err != nil {
			return
		}
		handleWebTransportSession(sess, virtualRouter, defaultMaxDatagramSize, allow)
	})
	go func() {
		_ = wtServer.Serve(wtUDPConn)
	}()

	// 4. Connect WebTransport client directly to 127.0.0.1:serverUDPAddr.Port
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		QUICConfig: &quic.Config{
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}
	_, sess, err := dialer.Dial(ctx, fmt.Sprintf("https://127.0.0.1:%d/jamp", wtPort), nil)
	if err != nil {
		t.Fatalf("WebTransport Dial failed: %v", err)
	}
	defer sess.CloseWithError(0, "")

	vipClient, err := readClientVIP(ctx, sess)
	if err != nil {
		t.Fatalf("readClientVIP failed: %v", err)
	}

	const clientSrcPort uint16 = 27962
	// Send a blocked packet first to verify allowlist drops it
	blockedPkt := (&youdp.Packet{
		DestIp:   serverIP,
		SrcIp:    vipClient,
		DestPort: blockedPort,
		SrcPort:  clientSrcPort,
		Data:     []byte("should-be-blocked"),
	}).Serialize()
	if err := sess.SendDatagram(blockedPkt); err != nil {
		t.Fatalf("SendDatagram(blocked) failed: %v", err)
	}

	// Send allowed getinfo query
	queryPkt := (&youdp.Packet{
		DestIp:   serverIP,
		SrcIp:    vipClient,
		DestPort: uint16(serverUDPAddr.Port),
		SrcPort:  clientSrcPort,
		Data:     expectedOOBQuery,
	}).Serialize()
	if err := sess.SendDatagram(queryPkt); err != nil {
		t.Fatalf("SendDatagram(getinfo) failed: %v", err)
	}

	// Expect small OOB reply via Datagram
	gotDgram, err := sess.ReceiveDatagram(ctx)
	if err != nil {
		t.Fatalf("ReceiveDatagram failed: %v", err)
	}
	oobResp, err := youdp.DeserializePacket(gotDgram)
	if err != nil {
		t.Fatalf("DeserializePacket(oob) failed: %v", err)
	}
	if oobResp.SrcIp != serverIP || oobResp.SrcPort != uint16(serverUDPAddr.Port) ||
		oobResp.DestIp != vipClient || oobResp.DestPort != clientSrcPort ||
		!bytes.Equal(oobResp.Data, expectedOOBReply) {
		t.Fatalf("OOB response mismatch: got src=%v:%d dst=%v:%d", oobResp.SrcIp, oobResp.SrcPort, oobResp.DestIp, oobResp.DestPort)
	}

	// Expect 1400-byte reply via framed Uni Stream
	inStream, err := sess.AcceptUniStream(ctx)
	if err != nil {
		t.Fatalf("AcceptUniStream failed: %v", err)
	}
	var lenBuf [2]byte
	if _, err := io.ReadFull(inStream, lenBuf[:]); err != nil {
		t.Fatalf("ReadFull(len) failed: %v", err)
	}
	frameLen := int(binary.BigEndian.Uint16(lenBuf[:]))
	frameBuf := make([]byte, frameLen)
	if _, err := io.ReadFull(inStream, frameBuf); err != nil {
		t.Fatalf("ReadFull(body) failed: %v", err)
	}
	largeResp, err := youdp.DeserializePacket(frameBuf)
	if err != nil {
		t.Fatalf("DeserializePacket(1400B) failed: %v", err)
	}
	if largeResp.SrcIp != serverIP || largeResp.SrcPort != uint16(serverUDPAddr.Port) ||
		largeResp.DestIp != vipClient || largeResp.DestPort != clientSrcPort ||
		!bytes.Equal(largeResp.Data, expectedLargeReply) {
		t.Fatalf("1400B response mismatch: got src=%v:%d dst=%v:%d len=%d",
			largeResp.SrcIp, largeResp.SrcPort, largeResp.DestIp, largeResp.DestPort, len(largeResp.Data))
	}

	select {
	case <-blockedHit:
		t.Fatal("blocked UDP port received a packet despite -udp-allow")
	default:
	}
}
