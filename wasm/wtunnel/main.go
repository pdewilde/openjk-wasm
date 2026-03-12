package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	ietfsucks "github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/ietf-sucks"
	"github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/youdp"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

// If you get this error:
// failed to sufficiently increase receive buffer size
// You must run the following commands:
// sudo sysctl -w net.core.rmem_max=7500000
// sudo sysctl -w net.core.wmem_max=7500000
//
// These do not persist across reboots.
// You must add them to /etc/sysctl.conf to make them permanent.

func main() {
	dir := flag.String("dir", "../../build-wasm/dist", "directory to serve static files from")
	flag.Parse()
	serve(*dir)
}

func serve(dir string) {
	mux := http.NewServeMux()
	cm := &ietfsucks.CertManager{}
	tlsConfig := &tls.Config{
		GetCertificate: cm.GetCert,
	}
	ht3Server := http3.Server{
		Handler: mux,
		Addr:    "0.0.0.0:8443",
		// TLSConfig: http3.ConfigureTLS TODO
		TLSConfig:  http3.ConfigureTLSConfig(tlsConfig),
		QUICConfig: &quic.Config{EnableDatagrams: true},
	}

	wtServer := webtransport.Server{
		H3: &ht3Server,
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all frontend origins, this is a retro game, not a bank!
		},
	}

	httpServer := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: mux,
	}

	// Serve static files
	fs := http.FileServer(http.Dir(dir))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		json.NewEncoder(w).Encode(map[string][]string{"hashes": cm.GetHashes()})
	})

	// Handle WebTransport connections from wasm JKJA clients.
	mux.HandleFunc("/jamp", func(w http.ResponseWriter, r *http.Request) {
		// No need for CORS, WebTransport handles differently using CheckOrigin.
		sess, err := wtServer.Upgrade(w, r)
		if err != nil {
			err := fmt.Errorf("failed to upgrade to webtransport connection: %w", err)
			log.Printf(err.Error())
			http.Error(w, err.Error(), http.StatusUpgradeRequired)
			return
		}
		log.Print(fmt.Sprintf("sess: %v\n", sess))
		
		// TODO: two goroutines, one to send one to receive

		// go func() {
			
		// }

		for {
			content, err := sess.ReceiveDatagram(sess.Context())
			if err != nil {
				log.Printf("Error receiving datagram: %v", err)
				// TODO -- are there errors I want to continue for?
				break
			}
			log.Printf("Received datagram: %v", content)
			packet, err := youdp.DeseralizePacket(content)
			if err != nil {
				// TODO handle error
				log.Printf("Invalid packet: %v", err.Error())
				continue
			}
			log.Printf("received packet: %#v", packet)
			
		}
	})

	// Start http3 server in go routine
	go func() {
		log.Println("Starting UDP WebTransport 0.0.0.0:8443")
		if err := ht3Server.ListenAndServe(); err != nil {
			log.Fatalf("QUIC server failed: %v", err)
		}
	}()

	// Start normal http server in main thread
	log.Println("Starting HTTP 0.0.0.0:8080")
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
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
