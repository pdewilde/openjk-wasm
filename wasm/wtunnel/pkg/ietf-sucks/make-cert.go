package ietfsucks

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"time"
)

// generateWebTransportCert creates a new, short-lived self-signed certificate
// and returns the tls.Certificate (for the server) and its SHA-256 hash (for the client).
func generateWebTransportCert() (*tls.Certificate, string, time.Time) {
	// 1. Generate ECDSA P-256 key pair (Strictly required by WebTransport)
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		err = fmt.Errorf("failed to generate private key: %w", err)
		panic(fmt.Sprintf("could not generate self signed cert: %s\n", err))
	}

	// 2. Set strict expiration windows (< 14 days)
	// Subtract 1 hour from notBefore to account for clock skew between server and client
	notBefore := time.Now().Add(-1 * time.Hour)
	notAfter := notBefore.Add(13 * 24 * time.Hour)

	// 3. Generate a secure random serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		err = fmt.Errorf("failed to generate serial number: %w", err)
		panic(fmt.Sprintf("could not generate self signed cert: %s\n", err))
	}

	// 4. Define the X.509 certificate template
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"In-Memory WebTransport Auto-Cert"},
			CommonName:   "localhost",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "openjk.local"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}

	// 5. Create the raw certificate (DER format)
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		err = fmt.Errorf("failed to create certificate: %w", err)
		panic(fmt.Sprintf("could not generate self signed cert: %s\n", err))
	}

	// 6. Calculate the SHA-256 hash of the DER bytes for the frontend
	hash := sha256.Sum256(derBytes)
	hashHex := hex.EncodeToString(hash[:])

	// 7. Package it into a tls.Certificate (No PEM encoding/decoding needed!)
	leaf, _ := x509.ParseCertificate(derBytes)
	tlsCert := &tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  privKey,
		Leaf:        leaf,
	}

	return tlsCert, hashHex, notAfter
}
