package configmgmt

import (
	"fmt"
	"testing"
)

func TestComputeDiffDiscoveryMode(t *testing.T) {
	snap := &ConfigSnapshot{
		Vendor:  "test",
		Address: "10.0.0.1:50051",
		Results: []PathResult{
			{
				ConfigPath: ConfigPath{Category: "interfaces", Name: "config", YANGPath: "/test"},
				Value:      map[string]interface{}{"enabled": true},
			},
			{
				ConfigPath: ConfigPath{Category: "bgp", Name: "global", YANGPath: "/test2"},
				Value:      map[string]interface{}{"as": float64(65000)},
			},
		},
	}

	report := ComputeDiff(snap, nil)

	if report.Summary.Total != 2 {
		t.Fatalf("expected 2 total, got %d", report.Summary.Total)
	}
	if report.Summary.Match != 2 {
		t.Fatalf("expected 2 match in discovery mode, got %d", report.Summary.Match)
	}
}

func TestComputeDiffWithDesired(t *testing.T) {
	snap := &ConfigSnapshot{
		Vendor:  "test",
		Address: "10.0.0.1:50051",
		Results: []PathResult{
			{
				ConfigPath: ConfigPath{Category: "interfaces", Name: "config"},
				Value:      map[string]interface{}{"enabled": true},
			},
			{
				ConfigPath: ConfigPath{Category: "bgp", Name: "global"},
				Value:      map[string]interface{}{"as": float64(65000)},
			},
		},
	}

	desired := &DesiredConfig{
		Paths: map[string]interface{}{
			"interfaces.config": map[string]interface{}{"enabled": true}, // match
			"bgp.global":        map[string]interface{}{"as": float64(65001)}, // mismatch
			"system.hostname":   "switch01",                             // missing from snapshot
		},
	}

	report := ComputeDiff(snap, desired)

	if report.Summary.Match != 1 {
		t.Errorf("expected 1 match, got %d", report.Summary.Match)
	}
	if report.Summary.Mismatch != 1 {
		t.Errorf("expected 1 mismatch, got %d", report.Summary.Mismatch)
	}
	if report.Summary.Missing != 1 {
		t.Errorf("expected 1 missing, got %d", report.Summary.Missing)
	}
}

func TestComputeDiffFetchError(t *testing.T) {
	snap := &ConfigSnapshot{
		Vendor: "test",
		Results: []PathResult{
			{
				ConfigPath: ConfigPath{Category: "bgp", Name: "global"},
				Error:      fmt.Errorf("gNMI Get failed: rpc error"),
			},
		},
	}

	report := ComputeDiff(snap, nil)

	if report.Summary.FetchError != 1 {
		t.Fatalf("expected 1 fetch_error, got %d", report.Summary.FetchError)
	}
}

func TestJsonEqual(t *testing.T) {
	tests := []struct {
		a, b     interface{}
		expected bool
	}{
		{true, true, true},
		{"hello", "hello", true},
		{float64(42), float64(42), true},
		{float64(42), float64(43), false},
		{map[string]interface{}{"a": float64(1)}, map[string]interface{}{"a": float64(1)}, true},
		{map[string]interface{}{"a": float64(1)}, map[string]interface{}{"a": float64(2)}, false},
		{nil, nil, true},
	}

	for i, tt := range tests {
		got := jsonEqual(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("test %d: jsonEqual(%v, %v) = %v, want %v", i, tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestFormatReport(t *testing.T) {
	report := &DiffReport{
		Vendor:  "cisco-nxos",
		Address: "10.0.0.1:50051",
		Entries: []DiffEntry{
			{Category: "interfaces", Name: "config", Status: DiffMatch, Actual: "up"},
			{Category: "bgp", Name: "global", Status: DiffFetchError, Details: "timeout"},
		},
		Summary: DiffSummary{Total: 2, Match: 1, FetchError: 1},
	}

	output := FormatReport(report)

	if output == "" {
		t.Fatal("FormatReport returned empty string")
	}
	// Verify key elements are present
	if !containsAll(output, "cisco-nxos", "INTERFACES", "BGP", "config", "timeout") {
		t.Errorf("FormatReport missing expected content:\n%s", output)
	}
}

func containsAll(s string, substrs ...string) bool {
	for _, sub := range substrs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
