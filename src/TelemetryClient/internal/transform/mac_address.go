package transform

import (
	"gnmi-collector/internal/gnmi"
)

const dataTypeMacTable = "cisco_nexus_mac_table"

func init() {
	Register("mac-table", func() Transformer { return &MacAddressTransformer{} })
}

type MacAddressTransformer struct{}

func (t *MacAddressTransformer) DataType() string { return dataTypeMacTable }

func (t *MacAddressTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			entriesObj := GetMap(vals, "entries")
			if entriesObj == nil {
				continue
			}

			entryList := GetSlice(entriesObj, "entry")
			for _, raw := range entryList {
				entry, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}

				state := GetMap(entry, "state")
				if state == nil {
					state = entry
				}

				// Extract port from interface ref
				port := ""
				if iface := GetMap(entry, "interface"); iface != nil {
					if ref := GetMap(iface, "interface-ref"); ref != nil {
						if refState := GetMap(ref, "state"); refState != nil {
							port = NormalizeInterfaceName(GetString(refState, "interface"))
						}
					}
				}

				msg := map[string]interface{}{
					"mac_address": GetString(state, "mac-address"),
					"vlan":        GetString(state, "vlan"),
					"type":        GetString(state, "entry-type"),
					"age":         GetString(state, "age"),
					"port":        port,
					"secure":      GetString(state, "secure"),
					"ntfy":        GetString(state, "ntfy"),
				}

				results = append(results, NewCommonFields(dataTypeMacTable, msg, n.Timestamp))
			}
		}
	}

	return results, nil
}
