package collector

import (
	"testing"

	"gnmi-collector/internal/config"
)

func TestExtractKeyFromPath(t *testing.T) {
	tests := []struct {
		path    string
		element string
		key     string
		want    string
	}{
		{
			path:    "/network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=bgp]/bgp",
			element: "network-instance",
			key:     "name",
			want:    "default",
		},
		{
			path:    "/network-instances/network-instance[name=Vrf_mgmt]/state",
			element: "network-instance",
			key:     "name",
			want:    "Vrf_mgmt",
		},
		{
			path:    "/network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=bgp]",
			element: "protocol",
			key:     "identifier",
			want:    "BGP",
		},
		{
			path:    "/network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=bgp]",
			element: "protocol",
			key:     "name",
			want:    "bgp",
		},
		{
			path:    "/interfaces/interface/state",
			element: "network-instance",
			key:     "name",
			want:    "",
		},
	}

	for _, tc := range tests {
		got := extractKeyFromPath(tc.path, tc.element, tc.key)
		if got != tc.want {
			t.Errorf("extractKeyFromPath(%q, %q, %q) = %q, want %q", tc.path, tc.element, tc.key, got, tc.want)
		}
	}
}

func TestBuildBGPProbeBase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=bgp]/bgp/neighbors",
			want:  "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=bgp]/bgp",
		},
		{
			input: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=bgp]/bgp/global",
			want:  "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=bgp]/bgp",
		},
		{
			input: "/openconfig-network-instance:network-instances/network-instance[name=default]/fdb/mac-table",
			want:  "",
		},
	}

	for _, tc := range tests {
		got := buildBGPProbeBase(tc.input)
		if got != tc.want {
			t.Errorf("buildBGPProbeBase(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSplitKeys(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"[name=default]", []string{"name=default"}},
		{"[identifier=BGP][name=bgp]", []string{"identifier=BGP", "name=bgp"}},
		{"/state", nil},
		{"", nil},
	}

	for _, tc := range tests {
		got := splitKeys(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("splitKeys(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitKeys(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

func TestDiscoverAndExpandNoTemplates(t *testing.T) {
	// Paths without templates should be returned unchanged.
	paths := []config.PathConfig{
		{Name: "interface-counters", YANGPath: "/interfaces/interface/state/counters", Table: "T1", Enabled: true},
		{Name: "system-state", YANGPath: "/system/state", Table: "T2", Enabled: true},
	}

	// nil client is fine — no discovery should happen.
	result, err := DiscoverAndExpand(nil, paths)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != len(paths) {
		t.Fatalf("expected %d paths, got %d", len(paths), len(result))
	}
	for i, p := range result {
		if p.YANGPath != paths[i].YANGPath {
			t.Errorf("path[%d] = %q, want %q", i, p.YANGPath, paths[i].YANGPath)
		}
	}
}

func TestDiscoverAndExpandDisabledTemplate(t *testing.T) {
	// Disabled template paths should be passed through unchanged
	// and should NOT trigger discovery (no client needed).
	paths := []config.PathConfig{
		{Name: "bgp-neighbors", YANGPath: "/ni/network-instance[name={network_instance}]/bgp/neighbors", Table: "T1", Enabled: false},
		{Name: "system-state", YANGPath: "/system/state", Table: "T2", Enabled: true},
	}

	result, err := DiscoverAndExpand(nil, paths)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(result))
	}
	// Template path should be unchanged (disabled, not expanded)
	if result[0].YANGPath != paths[0].YANGPath {
		t.Errorf("disabled path changed: got %q, want %q", result[0].YANGPath, paths[0].YANGPath)
	}
}
