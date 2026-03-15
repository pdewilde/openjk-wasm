package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/quic-go/webtransport-go"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := "https://127.0.0.1:8443/jamp"
	fmt.Printf("Connecting to %s (InsecureSkipVerify=true)...\n", url)

	dialer := webtransport.Dialer{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	_, sess, err := dialer.Dial(ctx, url, nil)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer sess.CloseWithError(0, "test finished")

	fmt.Println("Connected successfully!")

	// Wait for VIP
	fmt.Println("Waiting for VIP stream...")
	stream, err := sess.AcceptUniStream(ctx)
	if err != nil {
		log.Fatalf("Failed to accept stream: %v", err)
	}

	vip := make([]byte, 4)
	_, err = io.ReadFull(stream, vip)
	if err != nil {
		log.Fatalf("Failed to read VIP: %v", err)
	}

	fmt.Printf("SUCCESS! Received Virtual IP: %d.%d.%d.%d\n", vip[0], vip[1], vip[2], vip[3])

	fmt.Println("Test complete.")
	os.Exit(0)
}
