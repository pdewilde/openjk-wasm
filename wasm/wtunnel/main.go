package main

import (
	"crypto/subtle"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	stdpath "path"
	"strconv"
	"strings"

	"github.com/hashicorp/mdns"
	ietfsucks "github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/ietf-sucks"
	"github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/router"
	"github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/util"
	"github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/youdp"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
	"golang.org/x/crypto/acme/autocert"
)

const defaultMaxDatagramSize = 1150

type serverConfig struct {
	dir              string
	domain           string
	httpsAddr        string
	quicAddr         string
	authUser         string
	authPass         string
	corsOrigin       string
	maxDatagramSize  int
	udpAllow         *udpAllowlist
	maxSessionsPerIP int
	maxSessions      int
	rateLimitPerIP   float64
	rateBurstPerIP   float64
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return fallback
}

func envFloatOrDefault(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 {
			return f
		}
	}
	return fallback
}

func getLocalIP() net.IP {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP
			}
		}
	}
	return nil
}

func main() {
	dir := flag.String("dir", envOrDefault("WTUNNEL_DIR", "../../build-wasm/dist"), "directory to serve static files from")
	domain := flag.String("domain", envOrDefault("WTUNNEL_DOMAIN", ""), "domain name for Let's Encrypt (if empty, uses self-signed cert)")
	httpsAddr := flag.String("https-addr", envOrDefault("WTUNNEL_HTTPS_ADDR", ""), "HTTPS listen address (default 0.0.0.0:8080, or 0.0.0.0:443 with -domain)")
	quicAddr := flag.String("quic-addr", envOrDefault("WTUNNEL_QUIC_ADDR", ""), "QUIC/WebTransport listen address (default 0.0.0.0:8443, or 0.0.0.0:443 with -domain)")
	authUser := flag.String("auth-user", envOrDefault("WTUNNEL_AUTH_USER", ""), "optional HTTP basic auth username for hosted /base/ assets (disabled if empty)")
	authPass := flag.String("auth-pass", envOrDefault("WTUNNEL_AUTH_PASS", ""), "optional HTTP basic auth password for hosted /base/ assets")
	corsOrigin := flag.String("cors-origin", envOrDefault("WTUNNEL_CORS_ORIGIN", "*"), "allowed CORS / WebTransport origin (or * for any)")
	maxDgram := flag.Int("max-datagram-size", envIntOrDefault("WTUNNEL_MAX_DATAGRAM_SIZE", defaultMaxDatagramSize), "max serialized packet size sent via WebTransport datagram before falling back to framed uni stream")
	udpAllowSpec := flag.String("udp-allow", envOrDefault("WTUNNEL_UDP_ALLOW", ""), "comma-separated host:port or IPv4 CIDR allowlist for real UDP egress (empty disables egress)")
	maxSessionsPerIP := flag.Int("max-sessions-per-ip", envIntOrDefault("WTUNNEL_MAX_SESSIONS_PER_IP", defaultMaxSessionsPerIP), "max concurrent WebTransport sessions per client IP (0 disables)")
	maxSessions := flag.Int("max-sessions", envIntOrDefault("WTUNNEL_MAX_SESSIONS", defaultMaxSessions), "max total concurrent WebTransport sessions (0 disables)")
	rateLimitPerIP := flag.Float64("rate-limit-per-ip", envFloatOrDefault("WTUNNEL_RATE_LIMIT_PER_IP", defaultRateLimitPerIP), "max new WebTransport sessions per second per client IP (0 disables)")
	rateBurstPerIP := flag.Float64("rate-burst-per-ip", envFloatOrDefault("WTUNNEL_RATE_BURST_PER_IP", defaultRateBurstPerIP), "token bucket burst size for per-IP WebTransport connection rate limit")
	flag.Parse()

	udpAllow, err := parseUDPAllowlist(*udpAllowSpec)
	if err != nil {
		log.Fatalf("Invalid -udp-allow: %v", err)
	}

	cfg := serverConfig{
		dir:              *dir,
		domain:           *domain,
		httpsAddr:        *httpsAddr,
		quicAddr:         *quicAddr,
		authUser:         *authUser,
		authPass:         *authPass,
		corsOrigin:       *corsOrigin,
		maxDatagramSize:  *maxDgram,
		udpAllow:         udpAllow,
		maxSessionsPerIP: *maxSessionsPerIP,
		maxSessions:      *maxSessions,
		rateLimitPerIP:   *rateLimitPerIP,
		rateBurstPerIP:   *rateBurstPerIP,
	}
	if cfg.maxDatagramSize <= 0 {
		cfg.maxDatagramSize = defaultMaxDatagramSize
	}
	if cfg.httpsAddr == "" {
		if cfg.domain != "" {
			cfg.httpsAddr = "0.0.0.0:443"
		} else {
			cfg.httpsAddr = "0.0.0.0:8080"
		}
	}
	if cfg.quicAddr == "" {
		if cfg.domain != "" {
			cfg.quicAddr = "0.0.0.0:443"
		} else {
			cfg.quicAddr = "0.0.0.0:8443"
		}
	}

	serve(cfg)
}

func setAltSvcHeader(w http.ResponseWriter, r *http.Request, ht3Server *http3.Server, quicAddr string) {
	if r.ProtoMajor >= 3 {
		return
	}
	if err := ht3Server.SetQUICHeaders(w.Header()); err != nil {
		if _, port, splitErr := net.SplitHostPort(quicAddr); splitErr == nil && port != "" {
			w.Header().Set("Alt-Svc", fmt.Sprintf(`h3=":%s"; ma=2592000`, port))
		}
	}
}

func isOriginAllowed(allowed, origin string) bool {
	if allowed == "" || allowed == "*" || origin == "" {
		return true
	}
	for _, candidate := range strings.Split(allowed, ",") {
		if strings.TrimSpace(candidate) == origin {
			return true
		}
	}
	return false
}

func checkBasicAuth(r *http.Request, expectedUser, expectedPass string) bool {
	if expectedUser == "" && expectedPass == "" {
		return true
	}
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	userMatch := subtle.ConstantTimeCompare([]byte(user), []byte(expectedUser)) == 1
	passMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(expectedPass)) == 1
	return userMatch && passMatch
}

func isProtectedAssetPath(rawPath string) bool {
	clean := stdpath.Clean("/" + strings.ReplaceAll(rawPath, "\\", "/"))
	if clean == "/base" || strings.HasPrefix(clean, "/base/") {
		return true
	}
	switch clean {
	case "/openjk_bundled.html", "/openjk_mp_bundled.html",
		"/openjk_bundled.js", "/openjk_mp_bundled.js",
		"/openjk_bundled.wasm", "/openjk_mp_bundled.wasm",
		"/openjk_bundled.data", "/openjk_mp_bundled.data":
		return true
	}
	return false
}

func sanitizeNextRedirect(next string) string {
	next = strings.TrimSpace(next)
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.ContainsAny(next, "\\\r\n") {
		return "/"
	}
	return next
}

const hostedLoggedOutCookie = "openjk_hosted_off"

func isHostedLoggedOut(r *http.Request) bool {
	c, err := r.Cookie(hostedLoggedOutCookie)
	return err == nil && c.Value == "1"
}

func newStaticHandler(cfg serverConfig, ht3Server *http3.Server) http.Handler {
	fs := http.FileServer(http.Dir(cfg.dir))
	authEnabled := cfg.authUser != "" || cfg.authPass != ""
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := stdpath.Clean("/" + strings.ReplaceAll(r.URL.Path, "\\", "/"))
		if cleanPath == "/base/logout" {
			http.SetCookie(w, &http.Cookie{
				Name:     hostedLoggedOutCookie,
				Value:    "1",
				Path:     "/",
				SameSite: http.SameSiteLaxMode,
			})
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			if r.Header.Get("X-No-Auth-Prompt") == "1" {
				http.Error(w, "Logged out", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, sanitizeNextRedirect(r.URL.Query().Get("next")), http.StatusSeeOther)
			return
		}

		if cleanPath == "/base/login" {
			http.SetCookie(w, &http.Cookie{
				Name:     hostedLoggedOutCookie,
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				SameSite: http.SameSiteLaxMode,
			})
			if authEnabled && !checkBasicAuth(r, cfg.authUser, cfg.authPass) {
				w.Header().Set("WWW-Authenticate", `Basic realm="OpenJK Hosted Assets"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			http.Redirect(w, r, sanitizeNextRedirect(r.URL.Query().Get("next")), http.StatusSeeOther)
			return
		}

		if authEnabled && isProtectedAssetPath(r.URL.Path) {
			if isHostedLoggedOut(r) || !checkBasicAuth(r, cfg.authUser, cfg.authPass) {
				if r.Header.Get("X-No-Auth-Prompt") != "1" {
					w.Header().Set("WWW-Authenticate", `Basic realm="OpenJK Hosted Assets"`)
				}
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("X-Hosted-Auth", "enabled")
		}

		if ht3Server != nil {
			setAltSvcHeader(w, r, ht3Server, cfg.quicAddr)
		}

		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		fs.ServeHTTP(w, r)
	})
}

func serve(cfg serverConfig) {
	mux := http.NewServeMux()

	var tlsConfig *tls.Config
	var getHashes func() []string

	if cfg.domain != "" {
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(cfg.domain),
			Cache:      autocert.DirCache("certs"),
		}
		tlsConfig = m.TLSConfig()
		getHashes = func() []string { return []string{} }
	} else {
		cm := &ietfsucks.CertManager{}
		tlsConfig = &tls.Config{
			GetCertificate: cm.GetCert,
		}
		getHashes = cm.GetHashes
	}

	ht3Server := http3.Server{
		Handler:         mux,
		Addr:            cfg.quicAddr,
		TLSConfig:       http3.ConfigureTLSConfig(tlsConfig),
		EnableDatagrams: true,
		QUICConfig: &quic.Config{
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}
	webtransport.ConfigureHTTP3Server(&ht3Server)

	wtServer := webtransport.Server{
		H3: &ht3Server,
		CheckOrigin: func(r *http.Request) bool {
			return isOriginAllowed(cfg.corsOrigin, r.Header.Get("Origin"))
		},
	}

	httpServer := &http.Server{
		Addr:      cfg.httpsAddr,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	virtualRouter := router.NewRouter()
	limiter := newSessionLimiter(cfg.maxSessionsPerIP, cfg.maxSessions, cfg.rateLimitPerIP, cfg.rateBurstPerIP)

	localIP := getLocalIP()
	if localIP != nil {
		host, _ := os.Hostname()
		info := []string{"OpenJK WebTransport Relay"}
		service, err := mdns.NewMDNSService(host, "_http._tcp", "local.", "openjk.local.", 8080, []net.IP{localIP}, info)
		if err != nil {
			log.Printf("Failed to create mDNS service: %v", err)
		} else {
			server, err := mdns.NewServer(&mdns.Config{Zone: service})
			if err != nil {
				log.Printf("Failed to start mDNS server: %v", err)
			} else {
				defer server.Shutdown()
				log.Printf("Registered mDNS service openjk.local at %v", localIP)
			}
		}
	}

	log.Printf("Serving static files from: %s", cfg.dir)
	mux.Handle("/", newStaticHandler(cfg, &ht3Server))

	mux.HandleFunc("GET /wt-server-certificate-hashes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", cfg.corsOrigin)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		setAltSvcHeader(w, r, &ht3Server, cfg.quicAddr)

		hashes := getHashes()
		resp := struct {
			Hashes   []string `json:"hashes"`
			QuicPort string   `json:"quicPort,omitempty"`
		}{
			Hashes: hashes,
		}
		if _, port, err := net.SplitHostPort(cfg.quicAddr); err == nil {
			resp.QuicPort = port
		}
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/jamp", func(w http.ResponseWriter, r *http.Request) {
		clientIP, ok := extractClientIP(r.RemoteAddr)
		if !ok {
			http.Error(w, "invalid client address", http.StatusBadRequest)
			return
		}
		release, reason, allowed := limiter.Acquire(clientIP)
		if !allowed {
			log.Printf("Rejected WebTransport session from %s: %s", clientIP, reason)
			w.Header().Set("Retry-After", "1")
			http.Error(w, reason, http.StatusTooManyRequests)
			return
		}
		defer release()

		sess, err := wtServer.Upgrade(w, r)
		if err != nil {
			log.Printf("WebTransport upgrade failed for %s: %v", r.RemoteAddr, err)
			http.Error(w, "failed to upgrade to webtransport connection", http.StatusUpgradeRequired)
			return
		}
		log.Printf("WebTransport session established for %s", r.RemoteAddr)
		handleWebTransportSession(sess, virtualRouter, cfg.maxDatagramSize, cfg.udpAllow)
	})

	go func() {
		log.Printf("Starting UDP WebTransport %s\n", cfg.quicAddr)
		udpAddr, err := net.ResolveUDPAddr("udp4", cfg.quicAddr)
		if err != nil {
			log.Fatalf("Failed to resolve UDP address: %v", err)
		}
		conn, err := net.ListenUDP("udp4", udpAddr)
		if err != nil {
			log.Fatalf("Failed to listen UDP: %v", err)
		}
		if err := wtServer.Serve(conn); err != nil {
			log.Fatalf("WebTransport server failed: %v", err)
		}
	}()

	log.Printf("Starting HTTPS %s\n", cfg.httpsAddr)
	if err := httpServer.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("HTTPS server failed: %v", err)
	}
}

func handleWebTransportSession(sess *webtransport.Session, virtualRouter *router.Router, maxDatagramSize int, udpAllow *udpAllowlist) {
	defer sess.CloseWithError(0, "session closed")

	// outStream is owned exclusively by the single AsyncWriter worker goroutine.
	var outStream *webtransport.SendStream
	defer func() {
		if outStream != nil {
			_ = outStream.Close()
		}
	}()

	packetWriterFn := func(b []byte) (int, error) {
		if len(b) <= maxDatagramSize {
			if err := sess.SendDatagram(b); err != nil {
				return 0, err
			}
			return len(b), nil
		}
		if len(b) > 65535 {
			return 0, fmt.Errorf("packet too large for 2-byte framing: %d", len(b))
		}
		if outStream == nil {
			s, err := sess.OpenUniStreamSync(sess.Context())
			if err != nil {
				return 0, err
			}
			outStream = s
		}
		frame := make([]byte, 2+len(b))
		binary.BigEndian.PutUint16(frame[0:2], uint16(len(b)))
		copy(frame[2:], b)
		if _, err := outStream.Write(frame); err != nil {
			_ = outStream.Close()
			outStream = nil
			return 0, err
		}
		return len(b), nil
	}

	packetWriter := util.NewAsyncWriter(packetWriterFn, 64, func(err error) {
		log.Printf("Outbound write error for %s: %v", sess.RemoteAddr(), err)
	})
	defer packetWriter.Close()

	vip, err := virtualRouter.GetRoute(packetWriter)
	if err != nil {
		log.Printf("Failed to assign virtual IP for %s: %v", sess.RemoteAddr(), err)
		sess.CloseWithError(1, "no virtual IPs available")
		return
	}
	defer virtualRouter.RemoveRoute(vip)

	vipStream, err := sess.OpenUniStreamSync(sess.Context())
	if err != nil {
		log.Printf("Error opening VIP stream for %s: %v", sess.RemoteAddr(), err)
		sess.CloseWithError(1, "failed to open VIP stream")
		return
	}
	vipBytes := vip.As4()
	if _, err = vipStream.Write(vipBytes[:]); err != nil {
		_ = vipStream.Close()
		log.Printf("Error writing VIP to stream for %s: %v", sess.RemoteAddr(), err)
		sess.CloseWithError(1, "failed to send VIP")
		return
	}
	_ = vipStream.Close()

	egress := newUDPEgress(vip, udpAllow, packetWriter)
	defer egress.Close()

	forwardPacket := func(packet *youdp.Packet) {
		if cgnatPrefix.Contains(packet.DestIp) || packet.DestIp == netip.AddrFrom4([4]byte{255, 255, 255, 255}) {
			virtualRouter.Send(vip, packet)
			return
		}
		egress.Send(packet)
	}

	// Accept framed unidirectional streams for packets > maxDatagramSize.
	go func() {
		for {
			s, err := sess.AcceptUniStream(sess.Context())
			if err != nil {
				return
			}
			go readFramedUniStream(s, vip, forwardPacket)
		}
	}()

	for {
		content, err := sess.ReceiveDatagram(sess.Context())
		if err != nil {
			if !errors.Is(err, io.EOF) {
				log.Printf("Datagram receive ended for %s (%v): %v", sess.RemoteAddr(), vip, err)
			}
			break
		}
		packet, err := youdp.DeserializePacket(content)
		if err != nil {
			log.Printf("Invalid datagram packet from %v: %v", vip, err)
			continue
		}
		forwardPacket(packet)
	}
}

func readFramedUniStream(s *webtransport.ReceiveStream, vip netip.Addr, forward func(*youdp.Packet)) {
	var lenBuf [2]byte
	for {
		if _, err := io.ReadFull(s, lenBuf[:]); err != nil {
			return
		}
		frameLen := int(binary.BigEndian.Uint16(lenBuf[:]))
		if frameLen < youdp.HeaderSize {
			log.Printf("Invalid framed packet length %d from %v", frameLen, vip)
			return
		}
		buf := make([]byte, frameLen)
		if _, err := io.ReadFull(s, buf); err != nil {
			return
		}
		packet, err := youdp.DeserializePacket(buf)
		if err != nil {
			log.Printf("Invalid framed packet from %v: %v", vip, err)
			continue
		}
		forward(packet)
	}
}
