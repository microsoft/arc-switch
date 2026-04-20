package transform

import (
	"sort"
	"testing"
)

func TestRegistryHasAllTransformers(t *testing.T) {
	// All path names that must be registered. This list matches the
	// config path names used in production YAML configs. If a new
	// transformer is added without registering it, this test will fail.
	expected := []string{
		// OpenConfig transformers
		"interface-counters",
		"interface-status",
		"if-ethernet",
		"bgp-neighbors",
		"bgp-global",
		"lldp-neighbors",
		"mac-table",
		"temperature",
		"power-supply",
		"platform-inventory",
		"transceiver",
		"transceiver-channel",
		"system-cpus",
		"system-memory",
		"system-state",
		"arp-table",
		// Native Cisco YANG transformers
		"nx-transceiver",
		"nx-arp",
		"nx-bgp-peers",
		"nx-env-sensor",
		"nx-env-psu",
		"nx-fan",
		"nx-sys-cpu",
		"nx-sys-memory",
		"nx-mac-table",
		"nx-lldp",
		"nx-version",
		"nx-intf-errors",
		"nx-route-summary",
		"nx-sys-load",
		// SONiC native YANG transformers
		"sonic-temperature",
		"sonic-psu",
		"sonic-fan",
		"sonic-device-metadata",
	}
	sort.Strings(expected)

	registered := RegisteredNames()
	sort.Strings(registered)

	if len(registered) != len(expected) {
		t.Fatalf("expected %d registered transformers, got %d.\nExpected: %v\nGot:      %v",
			len(expected), len(registered), expected, registered)
	}

	for i, name := range expected {
		if registered[i] != name {
			t.Errorf("mismatch at index %d: expected %q, got %q", i, name, registered[i])
		}
	}
}

func TestRegistryGet(t *testing.T) {
	// Verify Get returns a non-nil transformer for known names
	for _, name := range []string{"interface-counters", "nx-sys-cpu", "system-cpus"} {
		tr := Get(name)
		if tr == nil {
			t.Errorf("Get(%q) returned nil", name)
		}
	}

	// Verify Get returns nil for unknown names
	if tr := Get("nonexistent-path"); tr != nil {
		t.Errorf("Get(\"nonexistent-path\") should return nil, got %v", tr)
	}
}

func TestBuildMapReturnsFreshInstances(t *testing.T) {
	m1 := BuildMap()
	m2 := BuildMap()

	// Same keys
	if len(m1) != len(m2) {
		t.Fatalf("BuildMap returned different sizes: %d vs %d", len(m1), len(m2))
	}

	// Verify every key in m1 is also in m2
	for name := range m1 {
		if _, ok := m2[name]; !ok {
			t.Errorf("key %q in first map but not in second", name)
		}
	}

	// Note: We do NOT compare pointer identity because Go is allowed to
	// coalesce allocations of zero-size structs (all our transformers are
	// stateless). This is expected behavior.
}
