package transform

import (
	"strings"
	"unicode"

	"gnmi-collector/internal/gnmi"
)

const dataTypeNativeLldp = "cisco_nexus_lldp_neighbor"

func init() {
	Register("nx-lldp", func() Transformer { return &NativeLldpTransformer{} })
}

type NativeLldpTransformer struct{}

func (t *NativeLldpTransformer) DataType() string {
	return dataTypeNativeLldp
}

func (t *NativeLldpTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			// NX-OS returns arrays for YANG list nodes (If-list)
			entries := AsMapSlice(u.Value)
			if entries == nil {
				continue
			}

			for _, vals := range entries {
				// Get local port from the map entry or from the path key
				localPort := GetString(vals, "id")
				if localPort == "" {
					localPort = extractKey(u.Path, "id")
				}

				adjItems := GetMap(vals, "adj-items")
				if adjItems == nil {
					continue
				}
				adjList := GetSlice(adjItems, "AdjEp-list")
				if adjList == nil {
					continue
				}

				for _, raw := range adjList {
					adj, ok := raw.(map[string]interface{})
					if !ok {
						continue
					}

					chassisId := GetString(adj, "chassisIdV")
					chassisId = strings.Map(func(r rune) rune {
						if !unicode.IsPrint(r) || r < 32 {
							return -1
						}
						return r
					}, chassisId)

					mgmtIp := GetString(adj, "mgmtIp")
					if mgmtIp == "unspecified" {
						mgmtIp = ""
					}

					// NX-OS returns capabilities as a comma/space-separated string
					// (e.g., "bridge,router"). Parse into []string to match the
					// OpenConfig lldp-neighbors schema.
					sysCaps := parseCapabilityString(GetString(adj, "capability"))
					enCaps := parseCapabilityString(GetString(adj, "enCap"))

					msg := map[string]interface{}{
						"chassis_id":                  chassisId,
						"port_id":                     GetString(adj, "portIdV"),
						"local_port_id":               NormalizeInterfaceName(localPort),
						"port_description":             GetString(adj, "portDesc"),
						"system_name":                 GetString(adj, "sysName"),
						"system_description":          GetString(adj, "sysDesc"),
						"management_address":          mgmtIp,
						"time_remaining":              GetString(adj, "ttl"),
						"vlan_id":                     GetString(adj, "portVlan"),
						"max_frame_size":              GetString(adj, "maxFramesize"),
						"link_aggregation_capability": GetString(adj, "linkAggCap"),
						"link_aggregation_id":         GetString(adj, "linkAggID"),
						"link_aggregation_status":     GetString(adj, "linkAggStatus"),
						"vlan_name":                   GetString(adj, "vlanName"),
						"system_capabilities":         sysCaps,
						"enabled_capabilities":        enCaps,
					}

					results = append(results, NewCommonFields(dataTypeNativeLldp, msg, n.Timestamp))
				}
			}
		}
	}

	return results, nil
}

// parseCapabilityString splits an NX-OS capability string (e.g., "bridge,router"
// or "bridge router") into a []string matching the OpenConfig schema.
// Returns nil for empty input so JSON serialization omits the field.
func parseCapabilityString(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	// NX-OS may use commas, spaces, or both as separators
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' '
	})
	var caps []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			caps = append(caps, p)
		}
	}
	if len(caps) == 0 {
		return nil
	}
	return caps
}
