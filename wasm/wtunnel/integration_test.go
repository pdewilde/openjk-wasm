package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	ietfsucks "github.com/pdewilde/openjk-wasm/wasm/wtunnel/pkg/ietf-sucks"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

func TestWebTransportIntegration(t *testing.T) {
	// 1. Setup Server
	mux := http.NewServeMux()
	cm := &ietfsucks.CertManager{}
	tlsConfig := &tls.Config{
		GetCertificate: cm.GetCert,
	}

	// Use port 0 for a random available port
	serverAddr := "127.0.0.1:0"

	ht3Server := http3.Server{
		Handler:         mux,
		Addr:            serverAddr,
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

	mux.HandleFunc("/jamp", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Server: Received upgrade request")
		sess, err := wtServer.Upgrade(w, r)
		if err != nil {
			fmt.Printf("Server: Upgrade failed: %v\n", err)
			return
		}
		fmt.Println("Server: Upgrade successful")

		stream, err := sess.OpenUniStreamSync(r.Context())
		if err != nil {
			fmt.Printf("Server: Failed to open uni stream: %v\n", err)
			return
		}
		fmt.Println("Server: Opened uni stream")
		stream.Write([]byte{100, 64, 0, 1})
		stream.Close()
		fmt.Println("Server: Sent VIP and closed stream")
	})

	// Start server
	// We need to use a listener to get the actual port
	// But http3.Server.ListenAndServe doesn't make it easy.
	// We'll use a trick: start it and then find the port.
	// Actually, quic-go's http3 server can take a PacketConn.

	// For simplicity in this test, we'll just use a fixed high port that's likely free.
	testPort := 48443
	ht3Server.Addr = fmt.Sprintf("127.0.0.1:%d", testPort)

	go func() {
		wtServer.ListenAndServe()
	}()
	defer wtServer.Close()

	// Give it a moment to start
	time.Sleep(500 * time.Millisecond)

	// 2. Setup Client
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://127.0.0.1:%d/jamp", testPort)

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		QUICConfig: &quic.Config{
			EnableDatagrams:                  true,
			EnableStreamResetPartialDelivery: true,
		},
	}

	_, sess, err := dialer.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer sess.CloseWithError(0, "")

	// 3. Verify Connection
	stream, err := sess.AcceptUniStream(ctx)
	if err != nil {
		t.Fatalf("Failed to accept stream: %v", err)
	}

	vip := make([]byte, 4)
	_, err = io.ReadFull(stream, vip)
	if err != nil {
		t.Fatalf("Failed to read VIP: %v", err)
	}

	expectedVip := []byte{100, 64, 0, 1}
	for i := range vip {
		if vip[i] != expectedVip[i] {
			t.Fatalf("Expected VIP %v, got %v", expectedVip, vip)
		}
	}

	t.Logf("Integration test passed! VIP: %v", vip)
}
