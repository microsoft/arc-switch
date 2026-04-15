package transform

import (
	"strings"

	"gnmi-collector/internal/gnmi"
)

const dataTypeNativeMac = "cisco_nexus_mac_table"

func init() {
	Register("nx-mac-table", func() Transformer { return &NativeMacTransformer{} })
}

type NativeMacTransformer struct{}

func (t *NativeMacTransformer) DataType() string {
	return dataTypeNativeMac
}

func (t *NativeMacTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			tableItems := GetMap(vals, "table-items")
			if tableItems == nil {
				continue
			}
			vlanItems := GetMap(tableItems, "vlan-items")
			if vlanItems == nil {
				continue
			}
			macList := getSlice(vlanItems, "MacAddressEntry-list")
			if macList == nil {
				continue
			}

			for _, raw := range macList {
				entry, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}

				vlanStr := GetString(entry, "vlan")
				if strings.HasPrefix(vlanStr, "vlan-") {
					vlanStr = strings.TrimPrefix(vlanStr, "vlan-")
				}

				msg := map[string]interface{}{
					"mac_address":   GetString(entry, "macAddress"),
					"vlan":          vlanStr,
					"type":          GetString(entry, "type"),
					"age":           GetString(entry, "age"),
					"port":          NormalizeInterfaceName(GetString(entry, "port")),
					"secure":        GetBool(entry, "secure"),
					"ntfy":          GetBool(entry, "ntfy"),
					"static":        GetBool(entry, "static"),
					"routed":        GetBool(entry, "routed"),
					"mac_info":      GetString(entry, "macInfo"),
					"primary_entry": GetString(entry, "type") == "primary",
					"routed_mac":    GetBool(entry, "routed"),
				}

				results = append(results, NewCommonFields(dataTypeNativeMac, msg, n.Timestamp))
			}
		}
	}

	return results, nil
}
