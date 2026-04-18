package transform

import (
	"strings"

	"gnmi-collector/internal/gnmi"
)

// NativeArpTransformer handles native Cisco NX-OS YANG ARP table data
// from /System/arp-items/inst-items/dom-items/Dom-list/db-items/Db-list/adj-items/AdjEp-list.
type NativeArpTransformer struct{}

func init() {
	Register("nx-arp", func() Transformer { return &NativeArpTransformer{} })
}

func (t *NativeArpTransformer) DataType() string { return dataTypeArp }

func (t *NativeArpTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			// NX-OS returns arrays for YANG list nodes (AdjEp-list)
			entries := AsMapSlice(u.Value)
			if entries == nil {
				continue
			}

			for _, vals := range entries {
				ifId := GetString(vals, "ifId")
				flags := GetString(vals, "flags")

				msg := map[string]interface{}{
					"ip_address":          GetString(vals, "ip"),
					"mac_address":         GetString(vals, "mac"),
					"interface":           NormalizeInterfaceName(ifId),
					"interface_type":      InterfaceType(ifId),
					"physical_interface":  NormalizeInterfaceName(GetString(vals, "physIfId")),
					"age":                 GetString(vals, "upTS"),
					"flags_raw":           flags,
					"status":              GetString(vals, "operSt"),
				}

				// Parse individual flags from the combined flags string.
				lowerFlags := strings.ToLower(flags)
				if strings.Contains(lowerFlags, "syncedviacfsoe") {
					msg["cfsoe_sync"] = true
				}
				if strings.Contains(lowerFlags, "non-active-fhrp") {
					msg["non_active_fhrp"] = true
				}
				if strings.Contains(lowerFlags, "+l") || strings.Contains(lowerFlags, "l2rib") {
					msg["control_plane_l2rib"] = true
				}

				results = append(results, NewCommonFields(dataTypeArp, msg, n.Timestamp))
			}
		}
	}

	return results, nil
}
