package gnmi

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const tofuProbeTimeout = 10 * time.Second

// FetchServerCert connects to the given address with TLS (skipping
// verification) and returns the server's leaf certificate in DER form.
// This is used for trust-on-first-use (TOFU) cert bootstrapping.
// The probe has a 10-second timeout to avoid hanging on unreachable targets.
func FetchServerCert(addr string) (*x509.Certificate, error) {
	dialer := &net.Dialer{Timeout: tofuProbeTimeout}
	// WORKAROUND: InsecureSkipVerify is required here because the TOFU probe
	// must connect before we have any trusted cert to verify against.
	// This is intended only for the current self-signed cert environment.
	// TODO: once a proper CA-signed certificate is installed on the switch
	// for gRPC, replace TOFU with standard certificate validation.
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		InsecureSkipVerify: true,
	})
	if err != nil {
		return nil, fmt.Errorf("TLS probe to %s: %w", addr, err)
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return nil, fmt.Errorf("server at %s presented no certificates", addr)
	}
	return certs[0], nil
}

// CertFingerprint returns the SHA-256 fingerprint of a certificate as a
// colon-separated hex string (e.g., "ab:cd:ef:12:...").
func CertFingerprint(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.Raw)
	parts := make([]string, len(hash))
	for i, b := range hash {
		parts[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(parts, ":")
}

// CertServerName extracts a usable ServerName from the certificate.
// Prefers the first DNS SAN; falls back to the first IP SAN as string;
// then falls back to the Subject CommonName.
func CertServerName(cert *x509.Certificate) string {
	if len(cert.DNSNames) > 0 {
		return cert.DNSNames[0]
	}
	if len(cert.IPAddresses) > 0 {
		return cert.IPAddresses[0].String()
	}
	return cert.Subject.CommonName
}

// TOFUCertPool performs trust-on-first-use: it probes the server,
// fetches its leaf certificate, and returns an in-memory CertPool
// along with the ServerName to use for hostname verification.
// The pool and server name are NOT persisted to disk.
func TOFUCertPool(addr string) (*x509.CertPool, string, error) {
	cert, err := FetchServerCert(addr)
	if err != nil {
		return nil, "", err
	}

	fp := CertFingerprint(cert)
	serverName := CertServerName(cert)
	log.Printf("TOFU: trusted server certificate from %s (ServerName=%s, SHA-256=%s)", addr, serverName, fp)

	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return pool, serverName, nil
}

// TOFURefetch re-probes the server and returns an updated CertPool
// if the certificate has changed since the given fingerprint.
// Returns the new pool, new fingerprint, and server name.
// Returns nil pool if the cert is unchanged.
func TOFURefetch(addr, oldFingerprint string) (*x509.CertPool, string, string, error) {
	cert, err := FetchServerCert(addr)
	if err != nil {
		return nil, "", "", fmt.Errorf("cert re-fetch probe failed: %w", err)
	}

	newFP := CertFingerprint(cert)
	if newFP == oldFingerprint {
		return nil, "", "", fmt.Errorf("server certificate unchanged (SHA-256: %s) — TLS error is not caused by cert rotation", newFP)
	}

	serverName := CertServerName(cert)
	log.Printf("WARN: server certificate changed — old fingerprint: %s, new fingerprint: %s", oldFingerprint, newFP)

	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return pool, newFP, serverName, nil
}

// SaveCertPEM writes a certificate to a PEM file atomically. It writes to
// a temporary file first and renames, preventing corruption on crash.
func SaveCertPEM(cert *x509.Certificate, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating cert directory %s: %w", dir, err)
	}

	block := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	data := pem.EncodeToMemory(block)

	// Write to temp file then rename for atomic update
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing temp cert file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // clean up on rename failure
		return fmt.Errorf("renaming cert file: %w", err)
	}
	return nil
}

// LoadCACertPool loads a PEM-encoded certificate file and returns an
// x509.CertPool containing it. Returns an error if the file cannot be
// read or contains no valid certificates.
func LoadCACertPool(path string) (*x509.CertPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading CA file %s: %w", path, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("CA file %s contains no valid certificates", path)
	}
	return pool, nil
}

// BootstrapCert loads a pinned CA cert from ca_file. If the file doesn't
// exist, it probes the server via TOFU, saves the cert, and returns
// the loaded cert pool. Returns nil if ca_file is not configured
// (caller should use in-memory TOFU instead).
func BootstrapCert(addr, caFile string) (*x509.CertPool, error) {
	if caFile == "" {
		return nil, nil
	}

	// If file exists, load it
	if _, err := os.Stat(caFile); err == nil {
		pool, loadErr := LoadCACertPool(caFile)
		if loadErr != nil {
			return nil, loadErr
		}
		log.Printf("Loaded pinned server certificate from %s", caFile)
		return pool, nil
	}

	// File doesn't exist — auto-fetch via TOFU
	log.Printf("ca_file %s not found — fetching server certificate from %s (TOFU)", caFile, addr)
	cert, err := FetchServerCert(addr)
	if err != nil {
		return nil, err
	}

	fp := CertFingerprint(cert)
	log.Printf("WARN: auto-fetched server certificate from %s (SHA-256: %s)", addr, fp)

	if err := SaveCertPEM(cert, caFile); err != nil {
		return nil, err
	}
	log.Printf("Saved server certificate to %s", caFile)

	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return pool, nil
}

// IsCertVerificationError returns true if the error is caused by a TLS
// certificate verification failure (e.g., unknown authority, cert mismatch).
// These errors indicate the server's cert doesn't match the pinned CA.
func IsCertVerificationError(err error) bool {
	if err == nil {
		return false
	}
	// Check for specific x509 certificate errors
	var unknownAuth x509.UnknownAuthorityError
	var certInvalid x509.CertificateInvalidError
	var hostErr x509.HostnameError

	errStr := err.Error()

	// x509 errors may be wrapped inside gRPC transport errors.
	// Check both typed assertion and string patterns.
	if asErr(&unknownAuth, err) || asErr(&certInvalid, err) || asErr(&hostErr, err) {
		return true
	}

	// gRPC wraps transport errors; check for common TLS patterns
	return strings.Contains(errStr, "x509:") ||
		strings.Contains(errStr, "certificate") && strings.Contains(errStr, "unknown authority")
}

// asErr is a helper that checks if any error in the chain matches the target type.
// Uses the same unwrap logic as errors.As but works with value types.
func asErr[T error](target *T, err error) bool {
	for err != nil {
		if e, ok := err.(T); ok {
			*target = e
			return true
		}
		if u, ok := err.(interface{ Unwrap() error }); ok {
			err = u.Unwrap()
		} else {
			return false
		}
	}
	return false
}

// RefetchAndSave probes the server for its current certificate and
// compares it to the existing pinned cert. If the cert has changed,
// it saves the new one and returns the updated cert pool. If the cert
// is the same, it returns nil (the problem isn't a changed cert).
func RefetchAndSave(addr, caFile string) (*x509.CertPool, error) {
	newCert, err := FetchServerCert(addr)
	if err != nil {
		return nil, fmt.Errorf("cert re-fetch probe failed: %w", err)
	}
	newFP := CertFingerprint(newCert)

	// Load existing cert to compare fingerprints
	oldFP := "(none)"
	if data, err := os.ReadFile(caFile); err == nil {
		block, _ := pem.Decode(data)
		if block != nil {
			if oldCert, err := x509.ParseCertificate(block.Bytes); err == nil {
				oldFP = CertFingerprint(oldCert)
			}
		}
	}

	if oldFP == newFP {
		// Cert hasn't changed — the TLS error is something else
		return nil, fmt.Errorf("server certificate unchanged (SHA-256: %s) — TLS error is not caused by cert rotation", newFP)
	}

	log.Printf("WARN: server certificate changed — old fingerprint: %s, new fingerprint: %s", oldFP, newFP)

	if err := SaveCertPEM(newCert, caFile); err != nil {
		return nil, err
	}
	log.Printf("Saved updated server certificate to %s", caFile)

	pool := x509.NewCertPool()
	pool.AddCert(newCert)
	return pool, nil
}
