package transform

import (
	"gnmi-collector/internal/gnmi"
)

const dataTypeLldpNeighbor = "cisco_nexus_lldp_neighbor"

func init() {
	Register("lldp-neighbors", func() Transformer { return &LldpNeighborTransformer{} })
}

// LldpNeighborTransformer converts gNMI LLDP neighbor data to the schema
// matching the existing lldp-neighbor parser.
type LldpNeighborTransformer struct{}

func (t *LldpNeighborTransformer) DataType() string {
	return dataTypeLldpNeighbor
}

func (t *LldpNeighborTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			localPort := ExtractInterfaceName(u.Path)

			// Two response formats:
			// 1) Bulk: vals = {"neighbor": [...]}, the full neighbors list
			// 2) Per-neighbor (Subscribe ONCE): vals IS a single neighbor map
			//    with "state", "capabilities", etc. directly
			neighbors := GetSlice(vals, "neighbor")
			if neighbors == nil {
				// Single-neighbor: check if vals has a "state" sub-map
				if GetMap(vals, "state") != nil || GetString(vals, "id") != "" {
					neighbors = []interface{}{vals}
				}
			}
			for _, raw := range neighbors {
				nbr, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}

				state := GetMap(nbr, "state")
				if state == nil {
					continue
				}

				// Extract capabilities
				var sysCaps, enabledCaps []string
				if capsObj := GetMap(nbr, "capabilities"); capsObj != nil {
					if capList := GetSlice(capsObj, "capability"); capList != nil {
						for _, raw := range capList {
							cap, ok := raw.(map[string]interface{})
							if !ok {
								continue
							}
							capName := GetString(cap, "name")
							sysCaps = append(sysCaps, capName)
							if capState := GetMap(cap, "state"); capState != nil {
								if GetBool(capState, "enabled") {
									enabledCaps = append(enabledCaps, capName)
								}
							}
						}
					}
				}

				msg := map[string]interface{}{
					"chassis_id":           GetString(state, "chassis-id"),
					"port_id":              GetString(state, "port-id"),
					"local_port_id":        NormalizeInterfaceName(localPort),
					"port_description":     GetString(state, "port-description"),
					"system_name":          GetString(state, "system-name"),
					"system_description":   GetString(state, "system-description"),
					"management_address":   GetFirstString(state, "management-address", "mgmt-ip"),
					"management_address_ipv6": GetFirstString(state, "management-address-ipv6", "mgmt-ipv6"),
					"time_remaining":       GetString(state, "ttl"),
					"max_frame_size":       GetString(state, "max-frame-size"),
					"vlan_id":              GetString(state, "vlan-id"),
					"system_capabilities":  sysCaps,
					"enabled_capabilities": enabledCaps,
				}

				results = append(results, NewCommonFields(dataTypeLldpNeighbor, msg, n.Timestamp))
			}
		}
	}

	return results, nil
}


