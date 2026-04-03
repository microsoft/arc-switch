package transform

import (
	"gnmi-collector/internal/gnmi"
)

const dataTypeArp = "cisco_nexus_arp_entry"

func init() {
	Register("arp-table", func() Transformer { return &ArpTransformer{} })
}

type ArpTransformer struct{}

func (t *ArpTransformer) DataType() string { return dataTypeArp }

func (t *ArpTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			neighbors := getSlice(vals, "neighbor")
			if neighbors == nil {
				// Try the value directly as a neighbor entry
				msg := extractArpEntry(vals, u.Path)
				if msg != nil {
					results = append(results, NewCommonFields(dataTypeArp, msg))
				}
				continue
			}

			for _, raw := range neighbors {
				nbr, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				msg := extractArpEntry(nbr, u.Path)
				if msg != nil {
					results = append(results, NewCommonFields(dataTypeArp, msg))
				}
			}
		}
	}

	return results, nil
}

func extractArpEntry(vals map[string]interface{}, path string) map[string]interface{} {
	state := GetMap(vals, "state")
	if state == nil {
		state = vals
	}

	ip := GetString(state, "ip")
	if ip == "" {
		return nil
	}

	ifName := ExtractInterfaceName(path)

	return map[string]interface{}{
		"ip_address":     ip,
		"mac_address":    GetString(state, "link-layer-address"),
		"interface":      NormalizeInterfaceName(ifName),
		"interface_type": InterfaceType(ifName),
		"age":            GetFirstString(state, "age", "expiry"),
	}
}
