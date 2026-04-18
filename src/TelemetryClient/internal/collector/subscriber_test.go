package collector

import (
	"fmt"
	"testing"

	gnmiclient "gnmi-collector/internal/gnmi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDrillDownToSubscribedPath_SystemResources(t *testing.T) {
	// NX-OS subscribe returns the entire /System tree.
	// We need to drill down to /System/procsys-items/syscpusummary-items.
	notifs := []gnmiclient.Notification{{
		Timestamp: 1000,
		Updates: []gnmiclient.Update{{
			Path: "/System",
			Value: map[string]interface{}{
				"procsys-items": map[string]interface{}{
					"syscpusummary-items": map[string]interface{}{
						"idle":   "74.0",
						"kernel": "6.0",
						"user":   "18.0",
					},
				},
			},
		}},
	}}

	result := drillDownToSubscribedPath(notifs, "/System/procsys-items/syscpusummary-items")
	if len(result) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(result))
	}

	u := result[0].Updates[0]
	if u.Path != "/System/procsys-items/syscpusummary-items" {
		t.Errorf("path = %q, want /System/procsys-items/syscpusummary-items", u.Path)
	}

	vals, ok := u.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("value is not map, got %T", u.Value)
	}
	if vals["idle"] != "74.0" {
		t.Errorf("idle = %v, want 74.0", vals["idle"])
	}
}

func TestDrillDownToSubscribedPath_InterfaceCounters(t *testing.T) {
	// NX-OS subscribe returns the entire /interfaces tree as a list.
	notifs := []gnmiclient.Notification{{
		Timestamp: 1000,
		Updates: []gnmiclient.Update{{
			Path: "/interfaces",
			Value: map[string]interface{}{
				"interface": []interface{}{
					map[string]interface{}{
						"name": "eth1/1",
						"state": map[string]interface{}{
							"counters": map[string]interface{}{
								"in-octets":  int64(100),
								"out-octets": int64(200),
							},
						},
					},
					map[string]interface{}{
						"name": "eth1/2",
						"state": map[string]interface{}{
							"counters": map[string]interface{}{
								"in-octets":  int64(300),
								"out-octets": int64(400),
							},
						},
					},
				},
			},
		}},
	}}

	result := drillDownToSubscribedPath(notifs, "/openconfig-interfaces:interfaces/interface/state/counters")
	if len(result) != 2 {
		t.Fatalf("expected 2 notifications (one per interface), got %d", len(result))
	}

	// Check first interface
	u1 := result[0].Updates[0]
	if u1.Path != "/interfaces/interface[name=eth1/1]/state/counters" {
		t.Errorf("path[0] = %q, want /interfaces/interface[name=eth1/1]/state/counters", u1.Path)
	}
	v1, _ := u1.Value.(map[string]interface{})
	if v1["in-octets"] != int64(100) {
		t.Errorf("in-octets = %v, want 100", v1["in-octets"])
	}

	// Check second interface
	u2 := result[1].Updates[0]
	if u2.Path != "/interfaces/interface[name=eth1/2]/state/counters" {
		t.Errorf("path[1] = %q, want /interfaces/interface[name=eth1/2]/state/counters", u2.Path)
	}
}

func TestDrillDownToSubscribedPath_AlreadyAtLevel(t *testing.T) {
	// If notification is already at the subscribed path level, pass through.
	notifs := []gnmiclient.Notification{{
		Timestamp: 1000,
		Updates: []gnmiclient.Update{{
			Path: "/System/procsys-items/syscpusummary-items",
			Value: map[string]interface{}{
				"idle": "99.0",
			},
		}},
	}}

	result := drillDownToSubscribedPath(notifs, "/System/procsys-items/syscpusummary-items")
	if len(result) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(result))
	}
	if result[0].Updates[0].Path != "/System/procsys-items/syscpusummary-items" {
		t.Errorf("path = %q, want passthrough", result[0].Updates[0].Path)
	}
}

func TestDrillDownToSubscribedPath_NoMatch(t *testing.T) {
	// Notification from a completely different path should not match.
	notifs := []gnmiclient.Notification{{
		Timestamp: 1000,
		Updates: []gnmiclient.Update{{
			Path:  "/System",
			Value: map[string]interface{}{"procsys-items": map[string]interface{}{}},
		}},
	}}

	result := drillDownToSubscribedPath(notifs, "/interfaces/interface/state/counters")
	if len(result) != 0 {
		t.Fatalf("expected 0 notifications for unrelated path, got %d", len(result))
	}
}

func TestStripPathModulePrefixes(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"/openconfig-interfaces:interfaces/interface/state", "/interfaces/interface/state"},
		{"/System/procsys-items", "/System/procsys-items"},
		{"/oc-platform:components/component", "/components/component"},
		{"", ""},
	}
	for _, tc := range tests {
		got := stripPathModulePrefixes(tc.input)
		if got != tc.want {
			t.Errorf("stripPathModulePrefixes(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestStripKeySelectors(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"/interfaces/interface[name=Ethernet0]/state/counters", "/interfaces/interface/state/counters"},
		{"/interfaces/interface/state/counters", "/interfaces/interface/state/counters"},
		{"/components/component[name=PSU1]/state", "/components/component/state"},
		{"/network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=BGP]", "/network-instances/network-instance/protocols/protocol"},
		{"", ""},
	}
	for _, tc := range tests {
		got := stripKeySelectors(tc.input)
		if got != tc.want {
			t.Errorf("stripKeySelectors(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDrillDown_SonicStylePathWithKeys(t *testing.T) {
	// SONiC subscribe sends data at the correct path level but with entity keys.
	// E.g., path="/openconfig-interfaces:interfaces/interface[name=Ethernet14]/state/counters"
	// YANG path="/openconfig-interfaces:interfaces/interface/state/counters"
	notifs := []gnmiclient.Notification{{
		Timestamp: 2000,
		Updates: []gnmiclient.Update{{
			Path: "/openconfig-interfaces:interfaces/interface[name=Ethernet14]/state/counters",
			Value: map[string]interface{}{
				"in-octets":  "12345",
				"out-octets": "67890",
			},
		}},
	}}

	result := drillDownToSubscribedPath(notifs, "/openconfig-interfaces:interfaces/interface/state/counters")
	if len(result) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(result))
	}

	u := result[0].Updates[0]
	// The original path (with keys) should be preserved
	want := "/openconfig-interfaces:interfaces/interface[name=Ethernet14]/state/counters"
	if u.Path != want {
		t.Errorf("path = %q, want %q", u.Path, want)
	}

	vals, ok := u.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("value is not map, got %T", u.Value)
	}
	if vals["in-octets"] != "12345" {
		t.Errorf("in-octets = %v, want 12345", vals["in-octets"])
	}
}

func TestIsPermanentSubscribeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "InvalidArgument is permanent",
			err:  status.Error(codes.InvalidArgument, "on_change not supported for path"),
			want: true,
		},
		{
			name: "wrapped InvalidArgument is permanent",
			err:  fmt.Errorf("subscribe recv: %w", status.Error(codes.InvalidArgument, "COUNTERS DB")),
			want: true,
		},
		{
			name: "Unavailable is transient",
			err:  status.Error(codes.Unavailable, "connection refused"),
			want: false,
		},
		{
			name: "Unknown is transient",
			err:  status.Error(codes.Unknown, "stream terminated"),
			want: false,
		},
		{
			name: "plain error is transient",
			err:  fmt.Errorf("connection reset by peer"),
			want: false,
		},
		{
			name: "nil is transient",
			err:  nil,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isPermanentSubscribeError(tc.err)
			if got != tc.want {
				t.Errorf("isPermanentSubscribeError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
