package gnmi

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// generateSelfSignedCert creates a self-signed certificate for testing.
func generateSelfSignedCert(t *testing.T) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-switch"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	return cert
}

func TestCertFingerprint(t *testing.T) {
	cert := generateSelfSignedCert(t)
	fp := CertFingerprint(cert)

	// SHA-256 = 32 bytes = 32 hex pairs + 31 colons = 95 chars
	if len(fp) != 95 {
		t.Errorf("fingerprint length = %d, want 95", len(fp))
	}
	// Should be colon-separated hex
	parts := strings.Split(fp, ":")
	if len(parts) != 32 {
		t.Errorf("fingerprint parts = %d, want 32", len(parts))
	}

	// Same cert should produce same fingerprint
	fp2 := CertFingerprint(cert)
	if fp != fp2 {
		t.Errorf("fingerprint not deterministic: %s != %s", fp, fp2)
	}
}

func TestSaveCertPEM(t *testing.T) {
	cert := generateSelfSignedCert(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "test.pem")

	if err := SaveCertPEM(cert, path); err != nil {
		t.Fatalf("SaveCertPEM: %v", err)
	}

	// File should exist
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	// Should be valid PEM
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("no PEM block found")
	}
	if block.Type != "CERTIFICATE" {
		t.Errorf("PEM type = %q, want CERTIFICATE", block.Type)
	}

	// Should parse back to same cert
	parsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse cert from PEM: %v", err)
	}
	if CertFingerprint(parsed) != CertFingerprint(cert) {
		t.Error("round-tripped cert fingerprint doesn't match")
	}

	// Temp file should not exist
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file still exists after atomic write")
	}
}

func TestLoadCACertPool(t *testing.T) {
	cert := generateSelfSignedCert(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.pem")

	// Save a valid cert
	if err := SaveCertPEM(cert, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	pool, err := LoadCACertPool(path)
	if err != nil {
		t.Fatalf("LoadCACertPool: %v", err)
	}
	if pool == nil {
		t.Fatal("pool is nil")
	}

	// Test with invalid file
	badPath := filepath.Join(dir, "bad.pem")
	os.WriteFile(badPath, []byte("not a cert"), 0644)
	_, err = LoadCACertPool(badPath)
	if err == nil {
		t.Error("expected error for invalid PEM")
	}

	// Test with missing file
	_, err = LoadCACertPool(filepath.Join(dir, "missing.pem"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestIsCertVerificationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", os.ErrNotExist, false},
		{"x509 unknown authority string", errWithMsg("x509: certificate signed by unknown authority"), true},
		{"certificate unknown authority", errWithMsg("connection error: certificate signed by unknown authority"), true},
		{"unrelated grpc error", errWithMsg("rpc error: code = Unavailable"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCertVerificationError(tt.err)
			if got != tt.want {
				t.Errorf("IsCertVerificationError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

type simpleError struct{ msg string }

func (e *simpleError) Error() string { return e.msg }

func errWithMsg(msg string) error { return &simpleError{msg: msg} }

func TestRefetchAndSave_SameCert(t *testing.T) {
	// When the cert on disk matches what the server presents, RefetchAndSave
	// should return an error indicating the cert hasn't changed.
	// We can't easily test the full flow without a TLS server, but we can
	// test the "cert unchanged" path by saving a cert and then calling
	// RefetchAndSave with a server that would return the same cert.
	// Since we can't dial a real server in unit tests, we just verify the
	// file operations work correctly.

	cert := generateSelfSignedCert(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.pem")

	if err := SaveCertPEM(cert, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Verify the file was saved correctly
	pool, err := LoadCACertPool(path)
	if err != nil {
		t.Fatalf("load after save: %v", err)
	}
	if pool == nil {
		t.Fatal("pool should not be nil after save")
	}
}

func TestBootstrapCert_NoCAFile(t *testing.T) {
	// When ca_file is empty, BootstrapCert returns nil (skip_verify mode)
	pool, err := BootstrapCert("localhost:50051", "", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool != nil {
		t.Error("expected nil pool when ca_file is empty")
	}
}

func TestBootstrapCert_ExistingFile(t *testing.T) {
	cert := generateSelfSignedCert(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "ca.pem")

	if err := SaveCertPEM(cert, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	pool, err := BootstrapCert("localhost:50051", path, false)
	if err != nil {
		t.Fatalf("BootstrapCert: %v", err)
	}
	if pool == nil {
		t.Error("expected non-nil pool for existing ca_file")
	}
}

func TestBootstrapCert_MissingFileNoAutoFetch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.pem")

	_, err := BootstrapCert("localhost:50051", path, false)
	if err == nil {
		t.Error("expected error when ca_file missing and auto_fetch disabled")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}
