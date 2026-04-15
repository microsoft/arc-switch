package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseValidConfig(t *testing.T) {
	yaml := []byte(`
target:
  address: 127.0.0.1
  port: 50051
  tls:
    enabled: true
    skip_verify: true
  credentials:
    username_env: GNMI_USER
    password_env: GNMI_PASS
collection:
  mode: poll
  interval: 300s
  timeout: 30s
  encoding: JSON
azure:
  workspace_id_env: WORKSPACE_ID
  primary_key_env: PRIMARY_KEY
  secondary_key_env: SECONDARY_KEY
  device_type: cisco-nx-os
paths:
  - name: interface-counters
    yang_path: /openconfig-interfaces:interfaces/interface/state/counters
    table: CiscoInterfaceCounter
    enabled: true
  - name: bgp-neighbors
    yang_path: /openconfig-network-instance:network-instances/network-instance/protocols/protocol/bgp/neighbors/neighbor/state
    table: CiscoBgpSummary
    enabled: true
`)
	cfg, err := Parse(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Target.Address != "127.0.0.1" {
		t.Errorf("address = %q, want 127.0.0.1", cfg.Target.Address)
	}
	if cfg.Target.Port != 50051 {
		t.Errorf("port = %d, want 50051", cfg.Target.Port)
	}
	if !cfg.Target.TLS.Enabled {
		t.Error("tls.enabled should be true")
	}
	if !cfg.Target.TLS.SkipVerify {
		t.Error("tls.skip_verify should be true")
	}
	if cfg.Collection.Interval != 300*time.Second {
		t.Errorf("interval = %v, want 5m0s", cfg.Collection.Interval)
	}
	if cfg.Collection.Timeout != 30*time.Second {
		t.Errorf("timeout = %v, want 30s", cfg.Collection.Timeout)
	}
	if cfg.TargetAddr() != "127.0.0.1:50051" {
		t.Errorf("TargetAddr() = %q, want 127.0.0.1:50051", cfg.TargetAddr())
	}
	if len(cfg.Paths) != 2 {
		t.Fatalf("paths count = %d, want 2", len(cfg.Paths))
	}
	if cfg.Paths[0].Name != "interface-counters" {
		t.Errorf("paths[0].name = %q, want interface-counters", cfg.Paths[0].Name)
	}
}

func TestParseDefaults(t *testing.T) {
	yaml := []byte(`
target:
  address: 10.0.0.1
  port: 50051
azure:
  device_type: cisco-nx-os
paths:
  - name: test
    yang_path: /test/path
    table: TestTable
    enabled: true
`)
	cfg, err := Parse(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Collection.Interval != 5*time.Minute {
		t.Errorf("default interval = %v, want 5m", cfg.Collection.Interval)
	}
	if cfg.Collection.Timeout != 30*time.Second {
		t.Errorf("default timeout = %v, want 30s", cfg.Collection.Timeout)
	}
	if cfg.Collection.Encoding != "JSON" {
		t.Errorf("default encoding = %q, want JSON", cfg.Collection.Encoding)
	}
	if cfg.Collection.Mode != "poll" {
		t.Errorf("default mode = %q, want poll", cfg.Collection.Mode)
	}
}

func TestTLSValidation(t *testing.T) {
	// ca_file without skip_verify should pass validation
	caFileYaml := []byte(`
target:
  address: 127.0.0.1
  port: 50051
  tls:
    enabled: true
    ca_file: /etc/gnmi/server.pem
azure:
  device_type: cisco-nx-os
paths:
  - name: test
    yang_path: /test
    table: T
    enabled: true
`)
	_, err := Parse(caFileYaml)
	if err != nil {
		t.Errorf("ca_file config should be valid, got: %v", err)
	}

	// skip_verify without ca_file should pass validation
	skipYaml := []byte(`
target:
  address: 127.0.0.1
  port: 50051
  tls:
    enabled: true
    skip_verify: true
azure:
  device_type: cisco-nx-os
paths:
  - name: test
    yang_path: /test
    table: T
    enabled: true
`)
	_, err = Parse(skipYaml)
	if err != nil {
		t.Errorf("skip_verify config should be valid, got: %v", err)
	}

	// TLS disabled should not require either
	disabledYaml := []byte(`
target:
  address: 127.0.0.1
  port: 50051
  tls:
    enabled: false
azure:
  device_type: cisco-nx-os
paths:
  - name: test
    yang_path: /test
    table: T
    enabled: true
`)
	_, err = Parse(disabledYaml)
	if err != nil {
		t.Errorf("TLS disabled config should be valid, got: %v", err)
	}
}

func TestParseMissingDeviceType(t *testing.T) {
	yaml := []byte(`
target:
  address: 10.0.0.1
  port: 50051
paths:
  - name: test
    yang_path: /test/path
    table: TestTable
    enabled: true
`)
	_, err := Parse(yaml)
	if err == nil {
		t.Fatal("expected error for missing device_type, got nil")
	}
	if !strings.Contains(err.Error(), "device_type") {
		t.Errorf("error should mention device_type, got: %v", err)
	}
}

func TestParseInvalidConfigs(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{"missing address", `
target:
  port: 50051
azure:
  device_type: cisco-nx-os
paths:
  - name: test
    yang_path: /test
    table: T
    enabled: true`},
		{"invalid port", `
target:
  address: 127.0.0.1
  port: 0
azure:
  device_type: cisco-nx-os
paths:
  - name: test
    yang_path: /test
    table: T
    enabled: true`},
		{"no enabled paths", `
target:
  address: 127.0.0.1
  port: 50051
azure:
  device_type: cisco-nx-os
paths:
  - name: test
    yang_path: /test
    table: T
    enabled: false`},
		{"empty yang_path", `
target:
  address: 127.0.0.1
  port: 50051
azure:
  device_type: cisco-nx-os
paths:
  - name: test
    yang_path: ""
    table: T
    enabled: true`},
		{"empty table", `
target:
  address: 127.0.0.1
  port: 50051
azure:
  device_type: cisco-nx-os
paths:
  - name: test
    yang_path: /test
    table: ""
    enabled: true`},
		{"tls enabled without skip_verify or ca_file", `
target:
  address: 127.0.0.1
  port: 50051
  tls:
    enabled: true
azure:
  device_type: cisco-nx-os
paths:
  - name: test
    yang_path: /test
    table: T
    enabled: true`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.yaml))
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestResolveCredentials(t *testing.T) {
	os.Setenv("TEST_USER", "admin")
	os.Setenv("TEST_PASS", "secret123")
	defer os.Unsetenv("TEST_USER")
	defer os.Unsetenv("TEST_PASS")

	cfg := &Config{
		Target: TargetConfig{
			Credentials: CredConfig{
				UsernameEnv: "TEST_USER",
				PasswordEnv: "TEST_PASS",
			},
		},
	}
	user, pass := cfg.ResolveCredentials()
	if user != "admin" {
		t.Errorf("username = %q, want admin", user)
	}
	if pass != "secret123" {
		t.Errorf("password = %q, want secret123", pass)
	}
}

func TestResolveAzureKeys(t *testing.T) {
	os.Setenv("TEST_WS", "ws-123")
	os.Setenv("TEST_PK", "primary-key")
	os.Setenv("TEST_SK", "secondary-key")
	defer os.Unsetenv("TEST_WS")
	defer os.Unsetenv("TEST_PK")
	defer os.Unsetenv("TEST_SK")

	cfg := &Config{
		Azure: AzureConfig{
			WorkspaceIDEnv:  "TEST_WS",
			PrimaryKeyEnv:   "TEST_PK",
			SecondaryKeyEnv: "TEST_SK",
		},
	}
	ws, pk, sk := cfg.ResolveAzureKeys()
	if ws != "ws-123" {
		t.Errorf("workspace = %q, want ws-123", ws)
	}
	if pk != "primary-key" {
		t.Errorf("primary = %q, want primary-key", pk)
	}
	if sk != "secondary-key" {
		t.Errorf("secondary = %q, want secondary-key", sk)
	}
}

func TestDataTypePrefix(t *testing.T) {
	tests := []struct {
		deviceType string
		want       string
	}{
		{"cisco-nx-os", "cisco_nexus"},
		{"sonic", "sonic"},
		{"arista-eos", "arista_eos"},       // fallback: hyphen→underscore
		{"juniper-junos", "juniper_junos"}, // fallback: hyphen→underscore
	}
	for _, tt := range tests {
		t.Run(tt.deviceType, func(t *testing.T) {
			cfg := &Config{Azure: AzureConfig{DeviceType: tt.deviceType}}
			got := cfg.DataTypePrefix()
			if got != tt.want {
				t.Errorf("DataTypePrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeviceTypeToPrefix(t *testing.T) {
	if got := DeviceTypeToPrefix("cisco-nx-os"); got != "cisco_nexus" {
		t.Errorf("got %q, want cisco_nexus", got)
	}
	if got := DeviceTypeToPrefix("sonic"); got != "sonic" {
		t.Errorf("got %q, want sonic", got)
	}
	// Fallback for unknown types
	if got := DeviceTypeToPrefix("some-new-vendor"); got != "some_new_vendor" {
		t.Errorf("got %q, want some_new_vendor", got)
	}
}

func TestValidatePathNames(t *testing.T) {
	cfg := &Config{
		Paths: []PathConfig{
			{Name: "interface-counters", YANGPath: "/ifc", Table: "T1", Enabled: true},
			{Name: "bgp-neighbors", YANGPath: "/bgp", Table: "T2", Enabled: true},
			{Name: "disabled-path", YANGPath: "/dis", Table: "T3", Enabled: false},
		},
	}
	validNames := []string{"interface-counters", "bgp-neighbors", "system-resources"}

	// All enabled paths are valid
	if err := cfg.ValidatePathNames(validNames); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Disabled paths with unknown names should not cause errors
	cfg.Paths[2].Name = "totally-unknown"
	if err := cfg.ValidatePathNames(validNames); err != nil {
		t.Fatalf("disabled unknown path should not error: %v", err)
	}

	// Enabled path with unknown name should fail
	cfg.Paths[0].Name = "typo-in-name"
	err := cfg.ValidatePathNames(validNames)
	if err == nil {
		t.Fatal("expected error for unknown enabled path, got nil")
	}
	if !strings.Contains(err.Error(), "typo-in-name") {
		t.Errorf("error should mention bad name, got: %v", err)
	}
}
