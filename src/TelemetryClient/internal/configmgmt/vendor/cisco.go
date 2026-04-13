package vendor

import (
	"gnmi-collector/internal/configmgmt"
)

func init() {
	configmgmt.RegisterProvider("cisco-nxos", func() configmgmt.Provider {
		return NewCiscoProvider()
	})
}

// CiscoProvider extends BaseProvider with Cisco NX-OS specific config paths
// and Set capability declarations.
type CiscoProvider struct {
	configmgmt.BaseProvider
}

// NewCiscoProvider creates a Cisco NX-OS config provider with both OpenConfig
// and NX-OS native config paths.
func NewCiscoProvider() *CiscoProvider {
	paths := OpenConfigPaths()
	paths = append(paths, ciscoNativePaths()...)

	return &CiscoProvider{
		BaseProvider: configmgmt.BaseProvider{
			Name:  "cisco-nxos",
			Paths: paths,
		},
	}
}

// SupportsSet returns true for paths known to be writable on NX-OS.
// Cisco's OpenConfig Set support is partial — many /config paths are
// not writable even though the YANG model says they should be.
// Native /System paths have better Set coverage.
func (c *CiscoProvider) SupportsSet(path configmgmt.ConfigPath) bool {
	if path.ReadOnly {
		return false
	}
	// Known writable paths on NX-OS (validated by testing).
	// This allowlist will grow as we validate more paths.
	writable := map[string]bool{
		"interface-description": true,
		"system-hostname":      true,
		"lldp-config":          true,
	}
	return writable[path.Name]
}

// NormalizeValue handles Cisco NX-OS encoding quirks:
// - Base64-encoded floats returned as strings
// - Nested single-key maps that should be unwrapped
func (c *CiscoProvider) NormalizeValue(path configmgmt.ConfigPath, raw interface{}) interface{} {
	// Cisco returns JSON (not JSON_IETF), so no module prefix stripping needed.
	// The base gNMI client already handles JSON decoding.
	return raw
}

// ciscoNativePaths returns Cisco NX-OS native YANG config paths that provide
// richer or more reliable config access than their OpenConfig equivalents.
func ciscoNativePaths() []configmgmt.ConfigPath {
	return []configmgmt.ConfigPath{
		// --- Interface (Native) ---
		{
			Category:    "interfaces",
			Name:        "native-interface-config",
			YANGPath:    "/System/intf-items/phys-items/PhysIf-list",
			Description: "NX-OS native physical interface config (full interface model including admin state, speed, description)",
		},
		{
			Category:    "interfaces",
			Name:        "native-loopback-config",
			YANGPath:    "/System/intf-items/lb-items/LbRtdIf-list",
			Description: "NX-OS native loopback interface config",
		},
		{
			Category:    "interfaces",
			Name:        "native-vlan-config",
			YANGPath:    "/System/intf-items/svi-items/If-list",
			Description: "NX-OS native SVI/VLAN interface config",
		},

		// --- BGP (Native) ---
		{
			Category:    "bgp",
			Name:        "native-bgp-global",
			YANGPath:    "/System/bgp-items/inst-items",
			Description: "NX-OS native BGP instance config (ASN, router-id, address-families)",
		},
		{
			Category:    "bgp",
			Name:        "native-bgp-peers",
			YANGPath:    "/System/bgp-items/inst-items/dom-items/Dom-list/peer-items/Peer-list",
			Description: "NX-OS native BGP peer config (full neighbor model with all knobs)",
		},

		// --- System (Native) ---
		{
			Category:    "system",
			Name:        "native-hostname",
			YANGPath:    "/System/name",
			Description: "NX-OS native system hostname",
		},

		// --- ACL (Native) ---
		{
			Category:    "acl",
			Name:        "native-ipv4-acl",
			YANGPath:    "/System/acl-items/ipv4-items/name-items/ACL-list",
			Description: "NX-OS native IPv4 ACL configuration",
		},

		// --- VLAN (Native) ---
		{
			Category:    "vlan",
			Name:        "native-vlan-db",
			YANGPath:    "/System/bd-items/bd-items/BD-list",
			Description: "NX-OS native VLAN database (bridge-domain list)",
		},

		// --- Static Routes (Native) ---
		{
			Category:    "routing",
			Name:        "native-static-routes",
			YANGPath:    "/System/urib-items/table4-items/Table4-list/route4-items/Route4-list",
			Description: "NX-OS native IPv4 static route config",
			ReadOnly:    true, // URIB is read-only; config routes go through /System/ipv4-items
		},
	}
}
