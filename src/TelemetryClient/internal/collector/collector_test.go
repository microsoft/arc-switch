package collector

import (
	"gnmi-collector/internal/transform"
	"testing"
)

func TestMergeByDataType(t *testing.T) {
	entries := []transform.CommonFields{
		{DataType: "interface_counters", Message: map[string]interface{}{"a": 1}},
		{DataType: "interface_counters", Message: map[string]interface{}{"b": 2}},
		{DataType: "bgp_summary", Message: map[string]interface{}{"c": 3}},
	}
	merged := mergeByDataType(entries)
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged groups, got %d", len(merged))
	}
	if merged[0].DataType != "interface_counters" {
		t.Errorf("first group data_type = %q, want interface_counters", merged[0].DataType)
	}
	if merged[1].DataType != "bgp_summary" {
		t.Errorf("second group data_type = %q, want bgp_summary", merged[1].DataType)
	}
}
