package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"

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

// If you get this error:
// failed to sufficiently increase receive buffer size
// You must run the following commands:
// sudo sysctl -w net.core.rmem_max=7500000
// sudo sysctl -w net.core.wmem_max=7500000
//
// These do not persist across reboots.
// You must add them to /etc/sysctl.conf to make them permanent.

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
	dir := flag.String("dir", "../../build-wasm/dist", "directory to serve static files from")
	domain := flag.String("domain", "", "domain name for Let's Encrypt (if empty, uses self-signed cert)")
	flag.Parse()
	serve(*dir, *domain)
}

func serve(dir string, domain string) {
	mux := http.NewServeMux()

	var tlsConfig *tls.Config
	var getHashes func() []string

	httpsPort := ":8080"
	quicPort := ":8443"

	if domain != "" {
		httpsPort = ":443"
		quicPort = ":443"
		m := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(domain),
			Cache:      autocert.DirCache("certs"), // Cache certs in a local directory
		}
		tlsConfig = m.TLSConfig()
		getHashes = func() []string { return []string{} } // No hashes needed for real certs
	} else {
		cm := &ietfsucks.CertManager{}
		tlsConfig = &tls.Config{
			GetCertificate: cm.GetCert,
		}
		getHashes = cm.GetHashes
	}

	ht3Server := http3.Server{
		Handler:         mux,
		Addr:            quicPort,
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
			return true // Allow all frontend origins, this is a retro game, not a bank!
		},
	}

	httpServer := &http.Server{
		Addr:      httpsPort,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	virtualRouter := router.NewRouter()

	// Register mDNS service for openjk.local
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
				log.Printf("You can access the test page at: https://%v:8080/test_wt.html", localIP)
			}
		}
	} else {
		log.Printf("You can access the test page at: https://localhost:8080/test_wt.html")
	}

	// Serve static files
	log.Printf("Serving static files from: %s", dir)
	fs := http.FileServer(http.Dir(dir))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Gate bundled files behind basic auth
		path := r.URL.Path
		if path == "/openjk_bundled.html" || path == "/openjk_mp_bundled.html" ||
			path == "/openjk_bundled.js" || path == "/openjk_mp_bundled.js" ||
			path == "/openjk_bundled.wasm" || path == "/openjk_mp_bundled.wasm" ||
			path == "/openjk_bundled.data" || path == "/openjk_mp_bundled.data" {

			user, pass, ok := r.BasicAuth()
			if !ok || user != "jedi" || pass != "academy" {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		fs.ServeHTTP(w, r)
	}))

	// Tell clients with the active and next certificate hashes for WebTransport connections.
	mux.HandleFunc("GET /wt-server-certificate-hashes", func(w http.ResponseWriter, r *http.Request) {
		// Enable CORS so a web client running on a different port/domain can fetch it.
		// Not strictly necessary if we serve the game from this same server, but nice
		// to have.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		hashes := getHashes()
		json.NewEncoder(w).Encode(map[string][]string{"hashes": hashes})
	})

	// Handle WebTransport connections from wasm JKJA clients.
	mux.HandleFunc("/jamp", func(w http.ResponseWriter, r *http.Request) {
		// No need for CORS, WebTransport handles differently using CheckOrigin.
		sess, err := wtServer.Upgrade(w, r)
		if err != nil {
			log.Printf("WebTransport upgrade failed for %s: %v", r.RemoteAddr, err)
			http.Error(w, "failed to upgrade to webtransport connection", http.StatusUpgradeRequired)
			return
		}
		log.Printf("WebTransport session established for %s", r.RemoteAddr)

		// Give client their ip address
		// TODO: should I be using r.context here?
		stream, err := sess.OpenUniStreamSync(r.Context())
		if err != nil {
			log.Printf("Error opening stream to connect : %v", err)
			// TODO: do I need to close session or handle?
			return
		}

		datagramWriterFn := func(b []byte) (int, error) {
			err := sess.SendDatagram(b)
			if err != nil {
				return 0, err
			}
			return len(b), nil
		}

		datagramWriter := util.WriterFunc(util.AsyncDropOldPacketsWriter(datagramWriterFn, 16, false))

		vip := virtualRouter.GetRoute(datagramWriter)
		defer virtualRouter.RemoveRoute(vip)
		vipBytes := vip.As4()
		if _, err = stream.Write(vipBytes[:]); err != nil {
			log.Printf("Error writing vip to stream: %v", err)
			// TODO; do I need to close session or handle?
			return
		}
		stream.Close()

		for {
			content, err := sess.ReceiveDatagram(sess.Context())
			if err != nil {
				log.Printf("Error receiving datagram: %v", err)
				// TODO -- are there errors I want to continue for?
				break
			}
			packet, err := youdp.DeserializePacket(content)
			if err != nil {
				// TODO handle error
				log.Printf("Invalid packet: %v", err.Error())
				continue
			}
			// Forward packet on
			virtualRouter.Send(vip, packet)
		}
	})

	// Start WebTransport server in go routine
	go func() {
		log.Println("Starting UDP WebTransport :8443")
		if err := wtServer.ListenAndServe(); err != nil {
			log.Fatalf("WebTransport server failed: %v", err)
		}
	}()

	// Start normal http server in main thread
	log.Println("Starting HTTPS :8080")
	if err := httpServer.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("HTTPS server failed: %v", err)
	}
}

// Middleware to advertise http/3 upgrades in general
// Not currently used because would need a real cert (serverCertificateHash isn't supported outside of webTransport)
// func advertiseHttp3(next http.Handler, server *http3.Server) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.ProtoMajor < 3 {
// 			err := server.SetQUICHeaders((w.Header()))
// 			if err != nil {
// 				log.Printf(fmt.Errorf("failed to set QUIC headers: %w", err).Error())
// 			}
// 		}
// 		next.ServeHTTP(w, r)
// 	})
// }
