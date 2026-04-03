package collector

import (
	"gnmi-collector/internal/transform"
	"testing"
)

func TestApplyDataTypePrefix(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		input    string
		expected string
	}{
		{
			name:     "cisco prefix is no-op",
			prefix:   "cisco_nexus",
			input:    "cisco_nexus_interface_counters",
			expected: "cisco_nexus_interface_counters",
		},
		{
			name:     "sonic prefix replaces cisco",
			prefix:   "sonic",
			input:    "cisco_nexus_interface_counters",
			expected: "sonic_interface_counters",
		},
		{
			name:     "sonic prefix for bgp_summary",
			prefix:   "sonic",
			input:    "cisco_nexus_bgp_summary",
			expected: "sonic_bgp_summary",
		},
		{
			name:     "sonic prefix for system_resources",
			prefix:   "sonic",
			input:    "cisco_nexus_system_resources",
			expected: "sonic_system_resources",
		},
		{
			name:     "unknown prefix for arbitrary vendor",
			prefix:   "arista_eos",
			input:    "cisco_nexus_lldp_neighbor",
			expected: "arista_eos_lldp_neighbor",
		},
		{
			name:     "no match if prefix missing",
			prefix:   "sonic",
			input:    "something_else",
			expected: "something_else",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := []transform.CommonFields{
				{DataType: tt.input},
			}
			applyDataTypePrefix(entries, tt.prefix)
			if entries[0].DataType != tt.expected {
				t.Errorf("got %q, want %q", entries[0].DataType, tt.expected)
			}
		})
	}
}

func TestApplyDataTypePrefixMultipleEntries(t *testing.T) {
	entries := []transform.CommonFields{
		{DataType: "cisco_nexus_interface_counters"},
		{DataType: "cisco_nexus_bgp_summary"},
		{DataType: "cisco_nexus_system_resources"},
	}
	applyDataTypePrefix(entries, "sonic")

	expected := []string{
		"sonic_interface_counters",
		"sonic_bgp_summary",
		"sonic_system_resources",
	}
	for i, e := range entries {
		if e.DataType != expected[i] {
			t.Errorf("entries[%d].DataType = %q, want %q", i, e.DataType, expected[i])
		}
	}
}
