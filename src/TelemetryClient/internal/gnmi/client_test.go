package gnmi

import (
	"encoding/json"
	"reflect"
	"testing"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

func TestParsePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantElem []string
	}{
		{
			name:     "simple path",
			input:    "/interfaces/interface/state/counters",
			wantLen:  4,
			wantElem: []string{"interfaces", "interface", "state", "counters"},
		},
		{
			name:     "with module prefix",
			input:    "/openconfig-interfaces:interfaces/interface/state/counters",
			wantLen:  4,
			wantElem: []string{"interfaces", "interface", "state", "counters"},
		},
		{
			name:     "with key selectors",
			input:    "/network-instances/network-instance[name=default]/protocols/protocol[name=bgp][identifier=BGP]",
			wantLen:  4,
			wantElem: []string{"network-instances", "network-instance", "protocols", "protocol"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := parsePath(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(path.Elem) != tt.wantLen {
				t.Fatalf("elem count = %d, want %d", len(path.Elem), tt.wantLen)
			}
			for i, want := range tt.wantElem {
				if path.Elem[i].Name != want {
					t.Errorf("elem[%d].Name = %q, want %q", i, path.Elem[i].Name, want)
				}
			}
		})
	}
}

func TestParsePathKeys(t *testing.T) {
	path, err := parsePath("/network-instances/network-instance[name=default]/protocols/protocol[name=bgp][identifier=BGP]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// network-instance[name=default]
	keys := path.Elem[1].Key
	if keys == nil {
		t.Fatal("expected keys on network-instance")
	}
	if keys["name"] != "default" {
		t.Errorf("key name = %q, want default", keys["name"])
	}

	// protocol[name=bgp][identifier=BGP]
	protoKeys := path.Elem[3].Key
	if protoKeys == nil {
		t.Fatal("expected keys on protocol")
	}
	if protoKeys["name"] != "bgp" {
		t.Errorf("key name = %q, want bgp", protoKeys["name"])
	}
	if protoKeys["identifier"] != "BGP" {
		t.Errorf("key identifier = %q, want BGP", protoKeys["identifier"])
	}
}

func TestParsePathEmpty(t *testing.T) {
	_, err := parsePath("")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestPathToString(t *testing.T) {
	path, _ := parsePath("/interfaces/interface/state/counters")
	s := pathToString(path)
	if s != "/interfaces/interface/state/counters" {
		t.Errorf("pathToString = %q, want /interfaces/interface/state/counters", s)
	}
}

func TestParseKeys(t *testing.T) {
	keys := parseKeys("[name=default][id=1]")
	if len(keys) != 2 {
		t.Fatalf("got %d keys, want 2", len(keys))
	}
	if keys[0] != "name=default" {
		t.Errorf("key[0] = %q, want name=default", keys[0])
	}
	if keys[1] != "id=1" {
		t.Errorf("key[1] = %q, want id=1", keys[1])
	}
}

func TestStripModulePrefixes(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  interface{}
	}{
		{
			name:  "nil value",
			input: nil,
			want:  nil,
		},
		{
			name:  "string value unchanged",
			input: "openconfig-bgp-types:IPV4_UNICAST",
			want:  "openconfig-bgp-types:IPV4_UNICAST",
		},
		{
			name:  "number value unchanged",
			input: float64(42),
			want:  float64(42),
		},
		{
			name:  "bool value unchanged",
			input: true,
			want:  true,
		},
		{
			name: "flat map with prefixed keys",
			input: map[string]interface{}{
				"openconfig-interfaces:counters": map[string]interface{}{
					"in-octets":  "123",
					"out-octets": "456",
				},
			},
			want: map[string]interface{}{
				"counters": map[string]interface{}{
					"in-octets":  "123",
					"out-octets": "456",
				},
			},
		},
		{
			name: "mixed prefixed and unprefixed keys",
			input: map[string]interface{}{
				"openconfig-system:state": map[string]interface{}{
					"hostname":                          "switch1",
					"boot-time":                         "1234567890",
					"openconfig-system-ext:auto-breakout": "DISABLE",
				},
			},
			want: map[string]interface{}{
				"state": map[string]interface{}{
					"hostname":     "switch1",
					"boot-time":    "1234567890",
					"auto-breakout": "DISABLE",
				},
			},
		},
		{
			name: "nested arrays with prefixed keys",
			input: map[string]interface{}{
				"openconfig-system:cpus": map[string]interface{}{
					"cpu": []interface{}{
						map[string]interface{}{
							"index": float64(0),
							"state": map[string]interface{}{
								"idle": map[string]interface{}{
									"avg":     float64(84),
									"instant": float64(84),
								},
							},
						},
					},
				},
			},
			want: map[string]interface{}{
				"cpus": map[string]interface{}{
					"cpu": []interface{}{
						map[string]interface{}{
							"index": float64(0),
							"state": map[string]interface{}{
								"idle": map[string]interface{}{
									"avg":     float64(84),
									"instant": float64(84),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "PSU cross-module augmentation keys",
			input: map[string]interface{}{
				"openconfig-platform:power-supply": map[string]interface{}{
					"state": map[string]interface{}{
						"openconfig-platform-psu:input-current":  "PpmZmg==",
						"openconfig-platform-psu:output-power":   "QiAAAA==",
						"openconfig-platform-psu:power-type":     "VOLT_AC",
					},
				},
			},
			want: map[string]interface{}{
				"power-supply": map[string]interface{}{
					"state": map[string]interface{}{
						"input-current":  "PpmZmg==",
						"output-power":   "QiAAAA==",
						"power-type":     "VOLT_AC",
					},
				},
			},
		},
		{
			name: "deeply nested with multiple module prefixes",
			input: map[string]interface{}{
				"openconfig-if-ethernet:state": map[string]interface{}{
					"auto-negotiate": false,
					"port-speed":     "openconfig-if-ethernet:SPEED_25GB",
					"counters": map[string]interface{}{
						"in-crc-errors": "0",
						"openconfig-if-ethernet-ext:in-distribution": map[string]interface{}{
							"in-frames-64-octets": "0",
						},
					},
					"openconfig-interfaces-ext:reason": "NO_TRANSCEIVER",
				},
			},
			want: map[string]interface{}{
				"state": map[string]interface{}{
					"auto-negotiate": false,
					"port-speed":     "openconfig-if-ethernet:SPEED_25GB",
					"counters": map[string]interface{}{
						"in-crc-errors": "0",
						"in-distribution": map[string]interface{}{
							"in-frames-64-octets": "0",
						},
					},
					"reason": "NO_TRANSCEIVER",
				},
			},
		},
		{
			name:  "empty map",
			input: map[string]interface{}{},
			want:  map[string]interface{}{},
		},
		{
			name:  "empty array",
			input: []interface{}{},
			want:  []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripModulePrefixes(tt.input)
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tt.want)
			if string(gotJSON) != string(wantJSON) {
				t.Errorf("stripModulePrefixes() =\n  %s\nwant:\n  %s", gotJSON, wantJSON)
			}
		})
	}
}

func TestResolveEncoding(t *testing.T) {
	tests := []struct {
		input string
		want  int32
	}{
		{"JSON", 0},       // gpb.Encoding_JSON = 0
		{"json", 0},
		{"", 0},
		{"JSON_IETF", 4},  // gpb.Encoding_JSON_IETF = 4
		{"json_ietf", 4},
		{"Json_Ietf", 4},
		{"PROTO", 2},      // gpb.Encoding_PROTO = 2
		{"proto", 2},
		{"unknown", 0},    // defaults to JSON
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveEncoding(tt.input)
			if int32(got) != tt.want {
				t.Errorf("resolveEncoding(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestDecodeTypedValue_JsonIetf_StripsModulePrefixes(t *testing.T) {
	// Simulate a JSON_IETF response from SONiC with module-prefixed keys.
	// The top-level single-key container should also be unwrapped.
	jsonIetf := []byte(`{
		"openconfig-interfaces:counters": {
			"in-octets": "12345",
			"out-octets": "67890"
		}
	}`)

	val := &gpb.TypedValue{
		Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: jsonIetf},
	}

	decoded, err := decodeTypedValue(val)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := decoded.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", decoded)
	}

	// After stripping + unwrapping, we should see the inner flat map directly
	if _, ok := m["in-octets"]; !ok {
		t.Errorf("expected key 'in-octets' (unwrapped), got keys: %v", keys(m))
	}
	if _, ok := m["counters"]; ok {
		t.Error("wrapper key 'counters' should have been unwrapped")
	}
	if m["in-octets"] != "12345" {
		t.Errorf("in-octets = %v, want 12345", m["in-octets"])
	}
}

func TestDecodeTypedValue_JsonIetf_NoUnwrapMultiKey(t *testing.T) {
	// If JSON_IETF has multiple top-level keys, don't unwrap
	jsonIetf := []byte(`{"key1": "val1", "key2": "val2"}`)

	val := &gpb.TypedValue{
		Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: jsonIetf},
	}

	decoded, err := decodeTypedValue(val)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := decoded.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", decoded)
	}
	if len(m) != 2 {
		t.Errorf("expected 2 keys, got %d: %v", len(m), keys(m))
	}
}

func TestDecodeTypedValue_JsonIetf_NestedPrefixes(t *testing.T) {
	// Verify cross-module augmentation keys are stripped at all levels
	jsonIetf := []byte(`{
		"openconfig-platform:power-supply": {
			"state": {
				"openconfig-platform-psu:input-current": "PpmZmg==",
				"openconfig-platform-psu:power-type": "VOLT_AC"
			}
		}
	}`)

	val := &gpb.TypedValue{
		Value: &gpb.TypedValue_JsonIetfVal{JsonIetfVal: jsonIetf},
	}

	decoded, err := decodeTypedValue(val)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// After strip + unwrap: {"state": {"input-current": "...", "power-type": "..."}}
	m, ok := decoded.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", decoded)
	}

	state, ok := m["state"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'state' map, got keys: %v", keys(m))
	}
	if _, ok := state["input-current"]; !ok {
		t.Errorf("expected 'input-current' (stripped), got keys: %v", keys(state))
	}
	if _, ok := state["openconfig-platform-psu:input-current"]; ok {
		t.Error("prefixed key should have been stripped")
	}
}

func TestDecodeTypedValue_Json_NoStripping(t *testing.T) {
	// JSON (non-IETF) should NOT strip prefixes (they shouldn't be there,
	// but if they are, it's not our job to strip them for JSON encoding)
	jsonVal := []byte(`{"counters": {"in-octets": "100"}}`)

	val := &gpb.TypedValue{
		Value: &gpb.TypedValue_JsonVal{JsonVal: jsonVal},
	}

	decoded, err := decodeTypedValue(val)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := decoded.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", decoded)
	}

	if _, ok := m["counters"]; !ok {
		t.Errorf("expected key 'counters', got keys: %v", keys(m))
	}
}

func keys(m map[string]interface{}) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func TestHasNonEmptyValues(t *testing.T) {
	tests := []struct {
		name   string
		notifs []Notification
		want   bool
	}{
		{
			name:   "nil notifications",
			notifs: nil,
			want:   false,
		},
		{
			name:   "empty slice",
			notifs: []Notification{},
			want:   false,
		},
		{
			name: "notification with empty map value",
			notifs: []Notification{
				{Updates: []Update{{Path: "/test", Value: map[string]interface{}{}}}},
			},
			want: false,
		},
		{
			name: "notification with non-empty map value",
			notifs: []Notification{
				{Updates: []Update{{Path: "/test", Value: map[string]interface{}{"key": "val"}}}},
			},
			want: true,
		},
		{
			name: "mixed empty and non-empty",
			notifs: []Notification{
				{Updates: []Update{{Path: "/a", Value: map[string]interface{}{}}}},
				{Updates: []Update{{Path: "/b", Value: map[string]interface{}{"key": "val"}}}},
			},
			want: true,
		},
		{
			name: "notification with nil value",
			notifs: []Notification{
				{Updates: []Update{{Path: "/test", Value: nil}}},
			},
			want: false,
		},
		{
			name: "notification with scalar value",
			notifs: []Notification{
				{Updates: []Update{{Path: "/test", Value: "hello"}}},
			},
			want: true,
		},
		{
			name: "notification with empty slice value",
			notifs: []Notification{
				{Updates: []Update{{Path: "/test", Value: []interface{}{}}}},
			},
			want: false,
		},
		{
			name: "notification with non-empty slice value",
			notifs: []Notification{
				{Updates: []Update{{Path: "/test", Value: []interface{}{"a"}}}},
			},
			want: true,
		},
		{
			name: "multiple updates all empty",
			notifs: []Notification{
				{Updates: []Update{
					{Path: "/a", Value: map[string]interface{}{}},
					{Path: "/b", Value: map[string]interface{}{}},
				}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasNonEmptyValues(tt.notifs)
			if got != tt.want {
				t.Errorf("HasNonEmptyValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitPathSegments(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"interfaces/interface/state", []string{"interfaces", "interface", "state"}},
		{"interfaces/interface[name=Ethernet0]/state", []string{"interfaces", "interface[name=Ethernet0]", "state"}},
		{"state/counters/in-octets", []string{"state", "counters", "in-octets"}},
		{"", nil},
		{"single", []string{"single"}},
	}
	for _, tt := range tests {
		got := splitPathSegments(tt.input)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitPathSegments(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestJoinPaths(t *testing.T) {
	tests := []struct {
		prefix, relative, want string
	}{
		{"/interfaces/interface[name=Ethernet0]", "/state/counters/in-octets", "/interfaces/interface[name=Ethernet0]/state/counters/in-octets"},
		{"/prefix", "", "/prefix"},
		{"", "/relative", "/relative"},
		{"", "", "/"},
		{"/a/b/", "/c/d", "/a/b/c/d"},
	}
	for _, tt := range tests {
		got := joinPaths(tt.prefix, tt.relative)
		if got != tt.want {
			t.Errorf("joinPaths(%q, %q) = %q, want %q", tt.prefix, tt.relative, got, tt.want)
		}
	}
}

func TestBuildTreeFromUpdates(t *testing.T) {
	updates := []Update{
		{Path: "/interfaces/interface[name=Ethernet0]/state/counters/in-octets", Value: int64(100)},
		{Path: "/interfaces/interface[name=Ethernet0]/state/counters/out-octets", Value: int64(200)},
		{Path: "/interfaces/interface[name=Ethernet0]/state/admin-status", Value: "UP"},
		{Path: "/interfaces/interface[name=Ethernet0]/state/oper-status", Value: "UP"},
	}

	tree := buildTreeFromUpdates(updates)

	// The common prefix is /interfaces/interface[name=Ethernet0]/state
	// So tree: {"counters": {"in-octets": 100, "out-octets": 200}, "admin-status": "UP", "oper-status": "UP"}
	counters, ok := tree["counters"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected counters map, got %T: %v", tree["counters"], tree)
	}
	if counters["in-octets"] != int64(100) {
		t.Errorf("in-octets = %v, want 100", counters["in-octets"])
	}
	if counters["out-octets"] != int64(200) {
		t.Errorf("out-octets = %v, want 200", counters["out-octets"])
	}
	if tree["admin-status"] != "UP" {
		t.Errorf("admin-status = %v, want UP", tree["admin-status"])
	}
}

func TestCommonPathPrefix(t *testing.T) {
	tests := []struct {
		name    string
		updates []Update
		want    string
	}{
		{
			name: "same entity different leaves",
			updates: []Update{
				{Path: "/iface[name=Eth0]/state/counters/in-octets"},
				{Path: "/iface[name=Eth0]/state/counters/out-octets"},
				{Path: "/iface[name=Eth0]/state/admin-status"},
			},
			want: "/iface[name=Eth0]/state",
		},
		{
			name: "single update",
			updates: []Update{
				{Path: "/a/b/c"},
			},
			want: "/a/b/c",
		},
		{
			name:    "empty",
			updates: []Update{},
			want:    "/",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commonPathPrefix(tt.updates)
			if got != tt.want {
				t.Errorf("commonPathPrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeSubscribeNotifications(t *testing.T) {
	// Simulate Subscribe ONCE leaf-level notifications for two interfaces
	notifs := []Notification{
		{
			Timestamp: 1000,
			Updates: []Update{
				{Path: "/interfaces/interface[name=Ethernet0]/state/admin-status", Value: "UP"},
				{Path: "/interfaces/interface[name=Ethernet0]/state/oper-status", Value: "UP"},
				{Path: "/interfaces/interface[name=Ethernet0]/state/mtu", Value: int64(9100)},
			},
		},
		{
			Timestamp: 2000,
			Updates: []Update{
				{Path: "/interfaces/interface[name=Ethernet4]/state/admin-status", Value: "DOWN"},
				{Path: "/interfaces/interface[name=Ethernet4]/state/oper-status", Value: "DOWN"},
			},
		},
	}

	result := NormalizeSubscribeNotifications(notifs)

	if len(result) != 2 {
		t.Fatalf("expected 2 normalized notifications, got %d", len(result))
	}

	// First notification should be for Ethernet0 with tree value
	n0 := result[0]
	if len(n0.Updates) != 1 {
		t.Fatalf("expected 1 merged update, got %d", len(n0.Updates))
	}
	if n0.Timestamp != 1000 {
		t.Errorf("timestamp = %d, want 1000", n0.Timestamp)
	}

	// Path should be the common prefix including entity key
	if n0.Updates[0].Path != "/interfaces/interface[name=Ethernet0]/state" {
		t.Errorf("path = %q, want /interfaces/interface[name=Ethernet0]/state", n0.Updates[0].Path)
	}

	// Value should be a tree map
	tree, ok := n0.Updates[0].Value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map value, got %T", n0.Updates[0].Value)
	}
	if tree["admin-status"] != "UP" {
		t.Errorf("admin-status = %v, want UP", tree["admin-status"])
	}
	if tree["mtu"] != int64(9100) {
		t.Errorf("mtu = %v, want 9100", tree["mtu"])
	}

	// Second notification for Ethernet4
	n1 := result[1]
	if n1.Updates[0].Path != "/interfaces/interface[name=Ethernet4]/state" {
		t.Errorf("path = %q, want /interfaces/interface[name=Ethernet4]/state", n1.Updates[0].Path)
	}
}

func TestNormalizeSubscribeNotifications_PassthroughTree(t *testing.T) {
	// If updates already contain map values (tree format), pass through
	notifs := []Notification{
		{
			Timestamp: 1000,
			Updates: []Update{
				{Path: "/system/state", Value: map[string]interface{}{"hostname": "switch1"}},
			},
		},
	}

	result := NormalizeSubscribeNotifications(notifs)
	if len(result) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(result))
	}
	if result[0].Updates[0].Path != "/system/state" {
		t.Errorf("path changed unexpectedly: %s", result[0].Updates[0].Path)
	}
}

func TestHasNonEmptyValues_EmptyString(t *testing.T) {
	// Empty string should be treated as empty (SONiC returns "" for some paths)
	notifs := []Notification{
		{Updates: []Update{{Path: "/test", Value: ""}}},
	}
	if HasNonEmptyValues(notifs) {
		t.Error("empty string should be considered empty")
	}

	// Non-empty string should be non-empty
	notifs2 := []Notification{
		{Updates: []Update{{Path: "/test", Value: "hello"}}},
	}
	if !HasNonEmptyValues(notifs2) {
		t.Error("non-empty string should be considered non-empty")
	}
}

func TestBuildTreeFromUpdates_ListSegments(t *testing.T) {
	// SONiC subscribe sends per-CPU leaf updates. After normalization,
	// list segments (cpu[index=X]) should produce arrays, not maps.
	updates := []Update{
		{Path: "/system/cpus/cpu[index=0]/state/idle/instant", Value: float64(73)},
		{Path: "/system/cpus/cpu[index=0]/state/kernel/instant", Value: float64(7)},
		{Path: "/system/cpus/cpu[index=0]/state/user/instant", Value: float64(18)},
		{Path: "/system/cpus/cpu[index=0]/index", Value: float64(0)},
		{Path: "/system/cpus/cpu[index=1]/state/idle/instant", Value: float64(80)},
		{Path: "/system/cpus/cpu[index=1]/state/kernel/instant", Value: float64(5)},
		{Path: "/system/cpus/cpu[index=1]/state/user/instant", Value: float64(12)},
		{Path: "/system/cpus/cpu[index=1]/index", Value: float64(1)},
	}
	tree := buildTreeFromUpdates(updates)

	// "cpu" should be an array with 2 elements
	cpuArr, ok := tree["cpu"].([]interface{})
	if !ok {
		t.Fatalf("cpu should be []interface{}, got %T: %v", tree["cpu"], tree["cpu"])
	}
	if len(cpuArr) != 2 {
		t.Fatalf("expected 2 CPU entries, got %d", len(cpuArr))
	}

	// Verify first CPU element has correct nested data
	cpu0, ok := cpuArr[0].(map[string]interface{})
	if !ok {
		t.Fatalf("cpu[0] should be map, got %T", cpuArr[0])
	}
	state0, ok := cpu0["state"].(map[string]interface{})
	if !ok {
		t.Fatalf("cpu[0].state should be map, got %T", cpu0["state"])
	}
	idle0, ok := state0["idle"].(map[string]interface{})
	if !ok {
		t.Fatalf("cpu[0].state.idle should be map, got %T", state0["idle"])
	}
	if idle0["instant"] != float64(73) {
		t.Errorf("cpu[0].state.idle.instant = %v, want 73", idle0["instant"])
	}
}

func TestParseListKeySelector(t *testing.T) {
	tests := []struct {
		input     string
		wantKey   string
		wantValue string
	}{
		{"[index=0]", "index", "0"},
		{"[name=Ethernet0]", "name", "Ethernet0"},
		{"[name=eth1/1]", "name", "eth1/1"},
		{"[]", "", ""},
	}
	for _, tc := range tests {
		gotK, gotV := parseListKeySelector(tc.input)
		if gotK != tc.wantKey || gotV != tc.wantValue {
			t.Errorf("parseListKeySelector(%q) = (%q, %q), want (%q, %q)",
				tc.input, gotK, gotV, tc.wantKey, tc.wantValue)
		}
	}
}