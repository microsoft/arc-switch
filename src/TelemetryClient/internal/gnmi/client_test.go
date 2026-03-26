package gnmi

import (
	"testing"
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
