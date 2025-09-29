package ip_route_parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParsePrefix(t *testing.T) {
	tests := []struct {
		network      string
		expectedIP   string
		expectedLen  int
	}{
		{"192.168.1.0/24", "192.168.1.0", 24},
		{"0.0.0.0/0", "0.0.0.0", 0},
		{"192.168.1.1/32", "192.168.1.1", 32},
		{"172.16.0.1", "172.16.0.1", 32}, // No prefix defaults to /32
	}

	for _, test := range tests {
		ip, length := parsePrefix(test.network)
		if ip != test.expectedIP || length != test.expectedLen {
			t.Errorf("parsePrefix(%s) = (%s, %d); expected (%s, %d)",
				test.network, ip, length, test.expectedIP, test.expectedLen)
		}
	}
}

func TestParsePreferenceMetric(t *testing.T) {
	tests := []struct {
		input      string
		expectedP  int
		expectedM  int
	}{
		{"[20/0]", 20, 0},
		{"[200/0]", 200, 0},
		{"[0/0]", 0, 0},
		{"[110/10]", 110, 10},
		{"invalid", 0, 0},
	}

	for _, test := range tests {
		pref, metric := parsePreferenceMetric(test.input)
		if pref != test.expectedP || metric != test.expectedM {
			t.Errorf("parsePreferenceMetric(%s) = (%d, %d); expected (%d, %d)",
				test.input, pref, metric, test.expectedP, test.expectedM)
		}
	}
}

func TestDetermineRouteType(t *testing.T) {
	tests := []struct {
		protocol string
		attached bool
		expected string
	}{
		{"bgp-65238", false, "bgp"},
		{"direct", true, "attached"},
		{"direct", false, "direct"},
		{"local", false, "local"},
		{"hsrp", false, "hsrp"},
		{"unknown", false, "unknown"},
	}

	for _, test := range tests {
		result := determineRouteType(test.protocol, test.attached)
		if result != test.expected {
			t.Errorf("determineRouteType(%s, %v) = %s; expected %s",
				test.protocol, test.attached, result, test.expected)
		}
	}
}

func TestParseIPRoutes(t *testing.T) {
	// Sample input data
	sampleInput := `CONTOSO-TOR1# show ip route |no
IP Route Table for VRF "default"
'*' denotes best ucast next-hop
'**' denotes best mcast next-hop
'[x/y]' denotes [preference/metric]
'%<string>' in via output denotes VRF <string>

0.0.0.0/0, ubest/mbest: 2/0
    *via 192.168.100.1, [20/0], 14w1d, bgp-65238, external, tag 64846
    *via 192.168.100.9, [20/0], 14w1d, bgp-65238, external, tag 64846
172.16.0.1/32, ubest/mbest: 1/0, attached
    *via 172.16.0.1, Tunnel1, [0/0], 37w6d, local
192.168.1.0/24, ubest/mbest: 1/0, attached
    *via 192.168.1.2, Vlan7, [0/0], 37w6d, direct
192.168.1.1/32, ubest/mbest: 1/0, attached
    *via 192.168.1.1, Vlan7, [0/0], 37w6d, hsrp
192.168.100.4/30, ubest/mbest: 1/0
    *via 192.168.100.18, [200/0], 14w1d, bgp-65238, internal, tag 65238
192.168.100.20/32, ubest/mbest: 2/0, attached
    *via 192.168.100.20, Lo0, [0/0], 37w6d, local
    *via 192.168.100.20, Lo0, [0/0], 37w6d, direct`

	// Parse the data
	routes := parseIPRoutes(sampleInput)

	// Should have 6 routes
	expectedCount := 6
	if len(routes) != expectedCount {
		t.Errorf("Expected %d routes, got %d", expectedCount, len(routes))
	}

	// Create map for easier testing
	routeMap := make(map[string]StandardizedEntry)
	for _, entry := range routes {
		routeMap[entry.Message.Network] = entry
	}

	// Test default route (0.0.0.0/0)
	if defaultEntry, exists := routeMap["0.0.0.0/0"]; exists {
		defaultRoute := defaultEntry.Message
		// Check standardized fields
		if defaultEntry.DataType != "cisco_nexus_ip_route" {
			t.Errorf("Default route data_type: expected 'cisco_nexus_ip_route', got '%s'", defaultEntry.DataType)
		}
		if defaultEntry.Timestamp == "" {
			t.Errorf("Default route timestamp should not be empty")
		}
		if defaultEntry.Date == "" {
			t.Errorf("Default route date should not be empty")
		}
		// Check message fields
		if defaultRoute.VRF != "default" {
			t.Errorf("Default route VRF: expected 'default', got '%s'", defaultRoute.VRF)
		}
		if defaultRoute.Prefix != "0.0.0.0" {
			t.Errorf("Default route prefix: expected '0.0.0.0', got '%s'", defaultRoute.Prefix)
		}
		if defaultRoute.PrefixLength != 0 {
			t.Errorf("Default route prefix length: expected 0, got %d", defaultRoute.PrefixLength)
		}
		if defaultRoute.UBest != 2 {
			t.Errorf("Default route ubest: expected 2, got %d", defaultRoute.UBest)
		}
		if defaultRoute.MBest != 0 {
			t.Errorf("Default route mbest: expected 0, got %d", defaultRoute.MBest)
		}
		if len(defaultRoute.NextHops) != 2 {
			t.Errorf("Default route: expected 2 next hops, got %d", len(defaultRoute.NextHops))
		} else {
			// Check first next hop
			nh1 := defaultRoute.NextHops[0]
			if nh1.Via != "192.168.100.1" {
				t.Errorf("Default route NH1 via: expected '192.168.100.1', got '%s'", nh1.Via)
			}
			if nh1.Preference != 20 {
				t.Errorf("Default route NH1 preference: expected 20, got %d", nh1.Preference)
			}
			if nh1.Protocol != "bgp-65238" {
				t.Errorf("Default route NH1 protocol: expected 'bgp-65238', got '%s'", nh1.Protocol)
			}
			// Check for external attribute
			hasExternal := false
			for _, attr := range nh1.Attributes {
				if attr == "external" {
					hasExternal = true
					break
				}
			}
			if !hasExternal {
				t.Error("Default route NH1 should have 'external' attribute")
			}
		}
	} else {
		t.Error("Default route (0.0.0.0/0) not found in parsed data")
	}

	// Test attached local route (172.16.0.1/32)
	if tunnelEntry, exists := routeMap["172.16.0.1/32"]; exists {
		tunnelRoute := tunnelEntry.Message
		if tunnelRoute.RouteType != "attached" {
			t.Errorf("Tunnel route type: expected 'attached', got '%s'", tunnelRoute.RouteType)
		}
		if len(tunnelRoute.NextHops) != 1 {
			t.Errorf("Tunnel route: expected 1 next hop, got %d", len(tunnelRoute.NextHops))
		} else {
			nh := tunnelRoute.NextHops[0]
			if nh.Interface != "Tunnel1" {
				t.Errorf("Tunnel route interface: expected 'Tunnel1', got '%s'", nh.Interface)
			}
			if nh.Protocol != "local" {
				t.Errorf("Tunnel route protocol: expected 'local', got '%s'", nh.Protocol)
			}
		}
	} else {
		t.Error("Tunnel route (172.16.0.1/32) not found in parsed data")
	}

	// Test HSRP route (192.168.1.1/32)
	if hsrpEntry, exists := routeMap["192.168.1.1/32"]; exists {
		hsrpRoute := hsrpEntry.Message
		if len(hsrpRoute.NextHops) != 1 {
			t.Errorf("HSRP route: expected 1 next hop, got %d", len(hsrpRoute.NextHops))
		} else {
			nh := hsrpRoute.NextHops[0]
			if nh.Protocol != "hsrp" {
				t.Errorf("HSRP route protocol: expected 'hsrp', got '%s'", nh.Protocol)
			}
		}
	} else {
		t.Error("HSRP route (192.168.1.1/32) not found in parsed data")
	}

	// Test route with multiple next hops (192.168.100.20/32)
	if loopbackEntry, exists := routeMap["192.168.100.20/32"]; exists {
		loopbackRoute := loopbackEntry.Message
		if loopbackRoute.UBest != 2 {
			t.Errorf("Loopback route ubest: expected 2, got %d", loopbackRoute.UBest)
		}
		if len(loopbackRoute.NextHops) != 2 {
			t.Errorf("Loopback route: expected 2 next hops, got %d", len(loopbackRoute.NextHops))
		}
	} else {
		t.Error("Loopback route (192.168.100.20/32) not found in parsed data")
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(routes)
	if err != nil {
		t.Errorf("Failed to marshal routes to JSON: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledRoutes []StandardizedEntry
	err = json.Unmarshal(jsonData, &unmarshaledRoutes)
	if err != nil {
		t.Errorf("Failed to unmarshal JSON: %v", err)
	}

	if len(unmarshaledRoutes) != len(routes) {
		t.Errorf("JSON round-trip failed: expected %d routes, got %d", 
			len(routes), len(unmarshaledRoutes))
	}
}

func TestLoadCommandsFromFile(t *testing.T) {
	// Create a temporary commands file
	tempFile := "test_commands.json"
	commandsData := `{
		"commands": [
			{
				"name": "ip-route",
				"command": "show ip route"
			},
			{
				"name": "test-command",
				"command": "show test"
			}
		]
	}`

	err := os.WriteFile(tempFile, []byte(commandsData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test commands file: %v", err)
	}
	defer os.Remove(tempFile)

	// Test loading commands
	config, err := loadCommandsFromFile(tempFile)
	if err != nil {
		t.Errorf("Failed to load commands from file: %v", err)
	}

	if len(config.Commands) != 2 {
		t.Errorf("Expected 2 commands, got %d", len(config.Commands))
	}

	// Test finding ip-route command
	command, err := findIPRouteCommand(config)
	if err != nil {
		t.Errorf("Failed to find ip-route command: %v", err)
	}

	expectedCommand := "show ip route"
	if command != expectedCommand {
		t.Errorf("Expected command '%s', got '%s'", expectedCommand, command)
	}
}