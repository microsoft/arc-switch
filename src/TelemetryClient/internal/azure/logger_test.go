package azure

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildSignature(t *testing.T) {
	// Use a known key (base64 encoded 32 random bytes)
	key := base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	workspaceID := "test-workspace-id"
	date := "Mon, 18 Mar 2026 20:00:00 GMT"
	contentLength := 42

	sig, err := buildSignature(date, contentLength, key, workspaceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify format: "SharedKey workspace:base64sig"
	if !strings.HasPrefix(sig, "SharedKey test-workspace-id:") {
		t.Errorf("signature format wrong: %s", sig)
	}

	// Verify deterministic: same input → same output
	sig2, _ := buildSignature(date, contentLength, key, workspaceID)
	if sig != sig2 {
		t.Error("signature is not deterministic")
	}

	// Different content length → different signature
	sig3, _ := buildSignature(date, 99, key, workspaceID)
	if sig == sig3 {
		t.Error("different content length should produce different signature")
	}
}

func TestBuildSignatureInvalidKey(t *testing.T) {
	_, err := buildSignature("date", 10, "not-base64!!!", "ws")
	if err == nil {
		t.Error("expected error for invalid base64 key")
	}
}

func TestNewLoggerValidation(t *testing.T) {
	_, err := NewLogger("", "key", "", "cisco")
	if err == nil {
		t.Error("expected error for empty workspace ID")
	}

	_, err = NewLogger("ws", "", "", "cisco")
	if err == nil {
		t.Error("expected error for empty primary key")
	}

	l, err := NewLogger("ws", "key", "", "cisco")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.workspaceID != "ws" {
		t.Errorf("workspaceID = %q, want ws", l.workspaceID)
	}
}
