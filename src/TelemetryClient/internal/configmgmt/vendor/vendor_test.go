package vendor

import (
	"testing"

	"gnmi-collector/internal/configmgmt"
)

func TestCiscoProviderRegistered(t *testing.T) {
	p := configmgmt.GetProvider("cisco-nxos")
	if p == nil {
		t.Fatal("cisco-nxos provider not registered")
	}
	if p.VendorName() != "cisco-nxos" {
		t.Fatalf("expected vendor name 'cisco-nxos', got %q", p.VendorName())
	}
}

func TestSonicProviderRegistered(t *testing.T) {
	p := configmgmt.GetProvider("sonic")
	if p == nil {
		t.Fatal("sonic provider not registered")
	}
	if p.VendorName() != "sonic" {
		t.Fatalf("expected vendor name 'sonic', got %q", p.VendorName())
	}
}

func TestAristaProviderRegistered(t *testing.T) {
	p := configmgmt.GetProvider("arista-eos")
	if p == nil {
		t.Fatal("arista-eos provider not registered")
	}
}

func TestAllProvidersRegistered(t *testing.T) {
	providers := configmgmt.RegisteredProviders()
	expected := map[string]bool{
		"cisco-nxos":  false,
		"sonic":       false,
		"arista-eos":  false,
	}
	for _, name := range providers {
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("provider %q not registered", name)
		}
	}
}

func TestCiscoHasOpenConfigAndNativePaths(t *testing.T) {
	p := configmgmt.GetProvider("cisco-nxos")
	paths := p.ConfigPaths()

	hasOC := false
	hasNative := false
	for _, cp := range paths {
		if len(cp.YANGPath) > 0 {
			if cp.YANGPath[0] == '/' && len(cp.YANGPath) > 1 {
				if cp.YANGPath[1] == 'o' { // /openconfig-*
					hasOC = true
				}
				if cp.YANGPath[1] == 'S' { // /System/*
					hasNative = true
				}
			}
		}
	}

	if !hasOC {
		t.Error("Cisco provider should have OpenConfig paths")
	}
	if !hasNative {
		t.Error("Cisco provider should have native /System paths")
	}
}

func TestSonicHasOpenConfigAndNativePaths(t *testing.T) {
	p := configmgmt.GetProvider("sonic")
	paths := p.ConfigPaths()

	hasOC := false
	hasSonic := false
	for _, cp := range paths {
		if len(cp.YANGPath) > 10 {
			if cp.YANGPath[:11] == "/openconfig" {
				hasOC = true
			}
			if cp.YANGPath[:7] == "/sonic-" {
				hasSonic = true
			}
		}
	}

	if !hasOC {
		t.Error("SONiC provider should have OpenConfig paths")
	}
	if !hasSonic {
		t.Error("SONiC provider should have SONiC-native paths")
	}
}

func TestCiscoSupportsSetConservative(t *testing.T) {
	p := configmgmt.GetProvider("cisco-nxos")

	// Read-only paths should never support Set
	readOnlyPath := configmgmt.ConfigPath{Name: "system-state", ReadOnly: true}
	if p.SupportsSet(readOnlyPath) {
		t.Error("Cisco should not support Set on read-only paths")
	}

	// Unknown paths should not support Set (conservative)
	unknownPath := configmgmt.ConfigPath{Name: "unknown-path"}
	if p.SupportsSet(unknownPath) {
		t.Error("Cisco should not support Set on unknown paths")
	}
}

func TestSonicSupportsSetBroader(t *testing.T) {
	p := configmgmt.GetProvider("sonic")

	// SONiC should support Set on interface config
	ifPath := configmgmt.ConfigPath{Name: "interface-config"}
	if !p.SupportsSet(ifPath) {
		t.Error("SONiC should support Set on interface-config")
	}

	// Read-only should still be blocked
	roPath := configmgmt.ConfigPath{Name: "system-state", ReadOnly: true}
	if p.SupportsSet(roPath) {
		t.Error("SONiC should not support Set on read-only paths")
	}
}

func TestAristaSupportsSetBroadly(t *testing.T) {
	p := configmgmt.GetProvider("arista-eos")

	// Arista should support Set on all non-read-only paths
	rwPath := configmgmt.ConfigPath{Name: "anything", ReadOnly: false}
	if !p.SupportsSet(rwPath) {
		t.Error("Arista should support Set on non-read-only paths")
	}

	roPath := configmgmt.ConfigPath{Name: "state", ReadOnly: true}
	if p.SupportsSet(roPath) {
		t.Error("Arista should not support Set on read-only paths")
	}
}

func TestOpenConfigPathsNonEmpty(t *testing.T) {
	paths := OpenConfigPaths()
	if len(paths) == 0 {
		t.Fatal("OpenConfigPaths returned no paths")
	}

	// All paths should have required fields
	for _, cp := range paths {
		if cp.Category == "" {
			t.Errorf("path %q has empty category", cp.Name)
		}
		if cp.Name == "" {
			t.Error("path has empty name")
		}
		if cp.YANGPath == "" {
			t.Errorf("path %q has empty YANG path", cp.Name)
		}
		if cp.Description == "" {
			t.Errorf("path %q has empty description", cp.Name)
		}
	}
}

func TestSonicNormalizePrefixStripping(t *testing.T) {
	p := NewSonicProvider()

	input := map[string]interface{}{
		"openconfig-interfaces:config": map[string]interface{}{
			"openconfig-interfaces:enabled": true,
			"openconfig-interfaces:mtu":     float64(9100),
		},
	}

	result := p.NormalizeValue(configmgmt.ConfigPath{}, input)

	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}

	// Verify prefixes were stripped
	if _, ok := m["config"]; !ok {
		t.Error("expected 'config' key after prefix stripping")
	}
	if _, ok := m["openconfig-interfaces:config"]; ok {
		t.Error("module prefix was not stripped")
	}
}
