// Package ietfsucks exists to tell the IETF that not everything needs extreme
// encryption. Sometimes you just want to make it easy for people to play
// a retro video game without worrying about certs. Sometimes you are doing
// ham radio and encryption is illegal. Get off your high horses.
package ietfsucks

import (
	"crypto/tls"
	"log"
	"sync"
	"time"
)

// CertManager manages short-lived self-signed certificates for WebTransport connections.
// It handles the creation and rotation of certificates.
type CertManager struct {
	mu sync.RWMutex

	activeCert *tls.Certificate
	activeHash string
	activeExp  time.Time

	nextCert *tls.Certificate
	nextHash string
	nextExp  time.Time
}

// Don't call unless you hold cm.mu.RLock
func (cm *CertManager) shouldUpdate() bool {
	if cm.activeCert == nil {
		return true
	}

	if cm.activeExp.Before(time.Now().Add(2*time.Hour)) && cm.nextCert == nil {
		return true
	}

	if cm.activeExp.Before(time.Now().Add(1*time.Hour)) && cm.nextCert != nil {
		return true
	}
	return false
}

func (cm *CertManager) update() {
	cm.mu.RLock()
	if !cm.shouldUpdate() {
		cm.mu.RUnlock()
		return
	}
	cm.mu.RUnlock()
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if !cm.shouldUpdate() {
		return
	}
	// Initial state
	if cm.activeCert == nil {
		cm.activeCert, cm.activeHash, cm.activeExp = generateWebTransportCert()
	}

	// Active cert is expiring soon. Lets create next cert.
	if cm.activeExp.Before(time.Now().Add(2*time.Hour)) && cm.nextCert == nil {
		cm.nextCert, cm.nextHash, cm.nextExp = generateWebTransportCert()
	}

	// Expiration is imminent, its time to swap certs. Only do so if we haven't already!
	if cm.activeExp.Before(time.Now().Add(1*time.Hour)) && cm.nextCert != nil {
		cm.activeCert = cm.nextCert
		cm.activeHash = cm.nextHash
		cm.activeExp = cm.nextExp
		cm.nextCert = nil
		cm.nextHash = ""
		cm.nextExp = time.Time{}
	}
}

// GetCert retuns active cert. Never returns error. Will update cert before returning
// if active cert is nil or expiring.
func (cm *CertManager) GetCert(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	log.Printf("TLS GetCert called for SNI: %s", hi.ServerName)
	cm.update()
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.activeCert, nil
}

// GetHashes retursn the active and next certificate hashes to give to WebTrasport
// client for serverCertificateHashes.
func (cm *CertManager) GetHashes() []string {
	cm.update()
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var hashes []string
	if cm.activeHash != "" {
		hashes = append(hashes, cm.activeHash)
	}
	if cm.nextHash != "" {
		hashes = append(hashes, cm.nextHash)
	}
	return hashes
}
