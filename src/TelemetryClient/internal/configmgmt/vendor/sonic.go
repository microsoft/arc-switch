package vendor

import (
	"strings"

	"gnmi-collector/internal/configmgmt"
)

func init() {
	configmgmt.RegisterProvider("sonic", func() configmgmt.Provider {
		return NewSonicProvider()
	})
}

// SonicProvider extends BaseProvider with Dell Enterprise SONiC specific
// config paths and JSON_IETF normalization.
type SonicProvider struct {
	configmgmt.BaseProvider
}

// NewSonicProvider creates a SONiC config provider. SONiC uses OpenConfig
// paths extensively, plus SONiC-native /sonic-* paths that map directly
// to ConfigDB.
func NewSonicProvider() *SonicProvider {
	paths := sonicOpenConfigPaths()
	paths = append(paths, sonicNativePaths()...)

	return &SonicProvider{
		BaseProvider: configmgmt.BaseProvider{
			Name:  "sonic",
			Paths: paths,
		},
	}
}

// SupportsSet returns true for paths known to be writable on SONiC.
// SONiC has good gNMI Set support because its management framework
// maps OpenConfig paths to ConfigDB operations.
func (s *SonicProvider) SupportsSet(path configmgmt.ConfigPath) bool {
	if path.ReadOnly {
		return false
	}
	// SONiC supports Set on most OpenConfig config paths and all sonic-* paths.
	// Only /state paths and a few edge cases are read-only.
	// Start conservative; expand as we validate.
	writable := map[string]bool{
		"interface-config":       true,
		"ethernet-config":        true,
		"bgp-global-config":      true,
		"bgp-neighbor-config":    true,
		"system-config":          true,
		"lldp-config":            true,
		"sonic-interface-config":  true,
		"sonic-vlan-config":       true,
		"sonic-portchannel-config": true,
		"sonic-ntp-config":         true,
	}
	return writable[path.Name]
}

// NormalizeValue handles SONiC JSON_IETF quirks:
// - Module prefixes on map keys (already stripped by gNMI client)
// - Empty objects {} returned for missing config
func (s *SonicProvider) NormalizeValue(path configmgmt.ConfigPath, raw interface{}) interface{} {
	// Strip any remaining module prefixes not caught by the gNMI client
	return stripRemainingPrefixes(raw)
}

// sonicOpenConfigPaths returns OpenConfig paths adapted for SONiC.
// SONiC requires explicit VRF keys on network-instance paths and sometimes
// returns empty for bulk paths (requiring per-instance queries).
func sonicOpenConfigPaths() []configmgmt.ConfigPath {
	return []configmgmt.ConfigPath{
		// --- Interfaces ---
		{
			Category:    "interfaces",
			Name:        "interface-config",
			YANGPath:    "/openconfig-interfaces:interfaces/interface/config",
			Description: "Interface configuration (enabled, description, mtu)",
		},
		{
			Category:    "interfaces",
			Name:        "interface-state",
			YANGPath:    "/openconfig-interfaces:interfaces/interface/state",
			Description: "Interface operational state (verification)",
			ReadOnly:    true,
		},
		{
			Category:    "interfaces",
			Name:        "ethernet-config",
			YANGPath:    "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/config",
			Description: "Ethernet config (speed, auto-negotiate)",
		},

		// --- BGP (requires VRF key) ---
		{
			Category:    "bgp",
			Name:        "bgp-global-config",
			YANGPath:    "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=BGP]/bgp/global/config",
			Description: "BGP global config (AS, router-id)",
		},
		{
			Category:    "bgp",
			Name:        "bgp-neighbor-config",
			YANGPath:    "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=BGP]/bgp/neighbors/neighbor/config",
			Description: "BGP neighbor config (peer-as, enabled)",
		},
		{
			Category:    "bgp",
			Name:        "bgp-global-state",
			YANGPath:    "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=BGP][name=BGP]/bgp/global/state",
			Description: "BGP global state (verification)",
			ReadOnly:    true,
		},

		// --- System ---
		{
			Category:    "system",
			Name:        "system-config",
			YANGPath:    "/openconfig-system:system/config",
			Description: "System config (hostname, domain-name)",
		},
		{
			Category:    "system",
			Name:        "system-state",
			YANGPath:    "/openconfig-system:system/state",
			Description: "System state (verification)",
			ReadOnly:    true,
		},

		// --- LLDP ---
		{
			Category:    "lldp",
			Name:        "lldp-config",
			YANGPath:    "/openconfig-lldp:lldp/config",
			Description: "LLDP global config (enabled, hello-timer)",
		},
	}
}

// sonicNativePaths returns SONiC-native YANG paths that map directly to
// ConfigDB tables. These often provide more direct config access than OC.
func sonicNativePaths() []configmgmt.ConfigPath {
	return []configmgmt.ConfigPath{
		{
			Category:    "interfaces",
			Name:        "sonic-interface-config",
			YANGPath:    "/sonic-interface:sonic-interface/INTERFACE/INTERFACE_LIST",
			Description: "SONiC ConfigDB interface table (IP addresses, VRF binding)",
		},
		{
			Category:    "vlan",
			Name:        "sonic-vlan-config",
			YANGPath:    "/sonic-vlan:sonic-vlan/VLAN/VLAN_LIST",
			Description: "SONiC ConfigDB VLAN table",
		},
		{
			Category:    "vlan",
			Name:        "sonic-vlan-member-config",
			YANGPath:    "/sonic-vlan:sonic-vlan/VLAN_MEMBER/VLAN_MEMBER_LIST",
			Description: "SONiC ConfigDB VLAN member table (port-to-VLAN mapping)",
		},
		{
			Category:    "interfaces",
			Name:        "sonic-portchannel-config",
			YANGPath:    "/sonic-portchannel:sonic-portchannel/PORTCHANNEL/PORTCHANNEL_LIST",
			Description: "SONiC ConfigDB port-channel (LAG) table",
		},
		{
			Category:    "system",
			Name:        "sonic-device-metadata",
			YANGPath:    "/sonic-device-metadata:sonic-device-metadata/DEVICE_METADATA/DEVICE_METADATA_LIST",
			Description: "SONiC device metadata (hostname, platform, hwsku)",
			ReadOnly:    true,
		},
		{
			Category:    "system",
			Name:        "sonic-ntp-config",
			YANGPath:    "/sonic-ntp:sonic-ntp/NTP_SERVER/NTP_SERVER_LIST",
			Description: "SONiC ConfigDB NTP server table",
		},
		{
			Category:    "acl",
			Name:        "sonic-acl-config",
			YANGPath:    "/sonic-acl:sonic-acl/ACL_TABLE/ACL_TABLE_LIST",
			Description: "SONiC ConfigDB ACL table definitions",
		},
		{
			Category:    "acl",
			Name:        "sonic-acl-rule-config",
			YANGPath:    "/sonic-acl:sonic-acl/ACL_RULE/ACL_RULE_LIST",
			Description: "SONiC ConfigDB ACL rules",
		},
	}
}

// stripRemainingPrefixes handles any YANG module prefixes that the gNMI
// client's stripModulePrefixes might have missed (e.g., in nested sonic-* data).
func stripRemainingPrefixes(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, child := range val {
			newKey := k
			if idx := strings.Index(k, ":"); idx != -1 {
				newKey = k[idx+1:]
			}
			out[newKey] = stripRemainingPrefixes(child)
		}
		return out
	case []interface{}:
		for i, item := range val {
			val[i] = stripRemainingPrefixes(item)
		}
		return val
	default:
		return v
	}
}
