// Package vendor provides vendor-specific gNMI config path providers.
// Each vendor registers itself via init() using the configmgmt registry.
//
// Architecture:
//
//	OpenConfigPaths()      ← shared across all vendors
//	    │
//	    ├── CiscoProvider  ← adds NX-OS native paths, overrides where needed
//	    ├── SonicProvider  ← adds SONiC-specific paths and normalization
//	    └── AristaProvider ← (future) adds EOS-specific paths
package vendor

import (
	"gnmi-collector/internal/configmgmt"
)

// OpenConfigPaths returns config paths based on OpenConfig YANG models that
// are expected to work across all vendors with gNMI support. These target
// /config subtrees (read-write) where possible, with /state fallbacks
// for verification.
//
// Vendors embed these and add their own native paths.
func OpenConfigPaths() []configmgmt.ConfigPath {
	return []configmgmt.ConfigPath{
		// --- Interfaces ---
		{
			Category:    "interfaces",
			Name:        "interface-config",
			YANGPath:    "/openconfig-interfaces:interfaces/interface/config",
			Description: "Interface configuration (enabled, description, mtu, type)",
		},
		{
			Category:    "interfaces",
			Name:        "interface-state",
			YANGPath:    "/openconfig-interfaces:interfaces/interface/state",
			Description: "Interface operational state (for verification against config)",
			ReadOnly:    true,
		},
		{
			Category:    "interfaces",
			Name:        "ethernet-config",
			YANGPath:    "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/config",
			Description: "Ethernet-specific config (auto-negotiate, speed, duplex)",
		},
		{
			Category:    "interfaces",
			Name:        "subinterface-config",
			YANGPath:    "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/config",
			Description: "Subinterface configuration (index, description, enabled)",
		},

		// --- BGP ---
		{
			Category:    "bgp",
			Name:        "bgp-global-config",
			YANGPath:    "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=BGP]/bgp/global/config",
			Description: "BGP global config (AS number, router-id)",
		},
		{
			Category:    "bgp",
			Name:        "bgp-neighbor-config",
			YANGPath:    "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=BGP]/bgp/neighbors/neighbor/config",
			Description: "BGP neighbor configuration (peer-as, description, enabled)",
		},
		{
			Category:    "bgp",
			Name:        "bgp-global-state",
			YANGPath:    "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=BGP]/bgp/global/state",
			Description: "BGP global operational state (for verification)",
			ReadOnly:    true,
		},

		// --- System ---
		{
			Category:    "system",
			Name:        "system-config",
			YANGPath:    "/openconfig-system:system/config",
			Description: "System configuration (hostname, domain-name, login-banner)",
		},
		{
			Category:    "system",
			Name:        "system-dns-config",
			YANGPath:    "/openconfig-system:system/dns/config",
			Description: "DNS resolver configuration",
		},
		{
			Category:    "system",
			Name:        "system-ntp-config",
			YANGPath:    "/openconfig-system:system/ntp/config",
			Description: "NTP configuration (enabled, source-address)",
		},
		{
			Category:    "system",
			Name:        "system-state",
			YANGPath:    "/openconfig-system:system/state",
			Description: "System operational state (hostname, uptime)",
			ReadOnly:    true,
		},

		// --- LLDP ---
		{
			Category:    "lldp",
			Name:        "lldp-config",
			YANGPath:    "/openconfig-lldp:lldp/config",
			Description: "LLDP global configuration (enabled, hello-timer, system-name)",
		},

		// --- Network Instance ---
		{
			Category:    "network-instance",
			Name:        "network-instance-config",
			YANGPath:    "/openconfig-network-instance:network-instances/network-instance/config",
			Description: "Network instance (VRF) configuration",
		},
	}
}
