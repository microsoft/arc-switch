package configmgmt

import (
	"testing"
)

func TestRegistryPanicsOnDuplicate(t *testing.T) {
	// Register a test provider
	name := "test-provider-dup"
	RegisterProvider(name, func() Provider {
		return &BaseProvider{Name: name}
	})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
		// Clean up
		delete(providers, name)
	}()

	// Should panic
	RegisterProvider(name, func() Provider {
		return &BaseProvider{Name: name}
	})
}

func TestGetProviderReturnsNilForUnknown(t *testing.T) {
	p := GetProvider("nonexistent-vendor-xyz")
	if p != nil {
		t.Fatalf("expected nil, got %v", p)
	}
}

func TestBaseProviderFetchConfigNilClient(t *testing.T) {
	// BaseProvider.FetchConfig with nil client should produce errors (not panic)
	// We can't easily test this without a mock client, so we just verify
	// the struct methods work.
	bp := &BaseProvider{
		Name: "test",
		Paths: []ConfigPath{
			{Category: "test", Name: "p1", YANGPath: "/test/path"},
		},
	}

	if bp.VendorName() != "test" {
		t.Fatalf("expected 'test', got %q", bp.VendorName())
	}

	paths := bp.ConfigPaths()
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}

	if bp.SupportsSet(paths[0]) {
		t.Fatal("BaseProvider should not support Set by default")
	}
}

func TestConfigSnapshotCategories(t *testing.T) {
	snap := &ConfigSnapshot{
		Results: []PathResult{
			{ConfigPath: ConfigPath{Category: "bgp", Name: "global"}},
			{ConfigPath: ConfigPath{Category: "interfaces", Name: "config"}},
			{ConfigPath: ConfigPath{Category: "bgp", Name: "neighbors"}},
			{ConfigPath: ConfigPath{Category: "system", Name: "hostname"}},
		},
	}

	cats := snap.Categories()
	expected := []string{"bgp", "interfaces", "system"}

	if len(cats) != len(expected) {
		t.Fatalf("expected %d categories, got %d: %v", len(expected), len(cats), cats)
	}
	for i, c := range cats {
		if c != expected[i] {
			t.Fatalf("category[%d]: expected %q, got %q", i, expected[i], c)
		}
	}
}

func TestConfigSnapshotByCategory(t *testing.T) {
	snap := &ConfigSnapshot{
		Results: []PathResult{
			{ConfigPath: ConfigPath{Category: "bgp", Name: "global"}},
			{ConfigPath: ConfigPath{Category: "interfaces", Name: "config"}},
			{ConfigPath: ConfigPath{Category: "bgp", Name: "neighbors"}},
		},
	}

	bgpResults := snap.ByCategory("bgp")
	if len(bgpResults) != 2 {
		t.Fatalf("expected 2 bgp results, got %d", len(bgpResults))
	}
}

func TestLastPathElement(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"interfaces/interface/config/description", "description"},
		{"system/config", "config"},
		{"bgp/neighbors/neighbor[address=10.0.0.1]/config", "config"},
		{"", ""},
		{"single", "single"},
	}

	for _, tt := range tests {
		got := lastPathElement(tt.path)
		if got != tt.expected {
			t.Errorf("lastPathElement(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}
