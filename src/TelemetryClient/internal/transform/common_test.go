package transform

import (
	"math"
	"testing"
)

func TestDecodeBase64Float32(t *testing.T) {
	// "RIl9cQ==" from the power-supply dump — should decode to a capacity value
	// Let's test with known values:
	// float32(100.0) = 0x42C80000 → base64 "QsgAAA=="
	f, err := DecodeBase64Float32("QsgAAA==")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(f-100.0) > 0.01 {
		t.Errorf("got %f, want 100.0", f)
	}

	// float32(3.14) = 0x4048F5C3 → base64 "QEj1ww=="
	f, err = DecodeBase64Float32("QEj1ww==")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if math.Abs(f-3.14) > 0.01 {
		t.Errorf("got %f, want ~3.14", f)
	}
}

func TestDecodeBase64Float32Invalid(t *testing.T) {
	_, err := DecodeBase64Float32("notbase64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}

	_, err = DecodeBase64Float32("AQID") // only 3 bytes
	if err == nil {
		t.Error("expected error for wrong length")
	}
}

func TestGetString(t *testing.T) {
	m := map[string]interface{}{
		"name":  "eth1/1",
		"count": 42,
	}
	if GetString(m, "name") != "eth1/1" {
		t.Error("expected eth1/1")
	}
	if GetString(m, "count") != "42" {
		t.Error("expected 42 as string")
	}
	if GetString(m, "missing") != "" {
		t.Error("expected empty for missing key")
	}
}

func TestExtractInterfaceName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/interfaces/interface[name=eth1/1]/state/counters", "eth1/1"},
		{"/interfaces/interface[name=vlan207]/state", "vlan207"},
		{"/interfaces/interface/state", ""},
	}
	for _, tt := range tests {
		got := ExtractInterfaceName(tt.path)
		if got != tt.want {
			t.Errorf("ExtractInterfaceName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestNormalizeInterfaceName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		// Cisco NX-OS names
		{"eth1/1", "Eth1/1"},
		{"eth1/36/4", "Eth1/36/4"},
		{"mgmt0", "mgmt0"},
		{"vlan207", "Vlan207"},
		{"port-channel50", "Po50"},
		{"lo0", "lo0"},
		// SONiC names (already canonical — returned as-is)
		{"Ethernet0", "Ethernet0"},
		{"Ethernet48", "Ethernet48"},
		{"PortChannel001", "PortChannel001"},
		{"Loopback0", "Loopback0"},
		{"Management0", "Management0"},
		{"Vlan100", "Vlan100"},
	}
	for _, tt := range tests {
		got := NormalizeInterfaceName(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeInterfaceName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestInterfaceType(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		// Cisco NX-OS names
		{"Eth1/1", "ethernet"},
		{"eth1/36/4", "ethernet"},
		{"Po50", "port-channel"},
		{"Vlan207", "vlan"},
		{"mgmt0", "management"},
		{"lo0", "loopback"},
		{"tunnel1", "tunnel"},
		{"nve1", "other"},
		// SONiC names
		{"Ethernet0", "ethernet"},
		{"Ethernet48", "ethernet"},
		{"PortChannel001", "port-channel"},
		{"Loopback0", "loopback"},
		{"Vlan100", "vlan"},
	}
	for _, tt := range tests {
		got := InterfaceType(tt.name)
		if got != tt.want {
			t.Errorf("InterfaceType(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
