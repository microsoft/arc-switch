package transform

import (
	"gnmi-collector/internal/gnmi"
)

const dataTypeInterfaceCounters = "cisco_nexus_interface_counters"

func init() {
	Register("interface-counters", func() Transformer { return &InterfaceCountersTransformer{} })
}

// InterfaceCountersTransformer converts gNMI interface counter data
// to the schema matching the existing cisco-parser interface-counters output.
type InterfaceCountersTransformer struct{}

func (t *InterfaceCountersTransformer) DataType() string {
	return dataTypeInterfaceCounters
}

func (t *InterfaceCountersTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			ifName := ExtractInterfaceName(u.Path)
			if ifName == "" {
				continue
			}

			normalized := NormalizeInterfaceName(ifName)
			msg := map[string]interface{}{
				"interface_name": normalized,
				"interface_type": InterfaceType(normalized),

				"in_octets":     GetInt64(vals, "in-octets"),
				"in_ucast_pkts": GetInt64(vals, "in-unicast-pkts"),
				"in_mcast_pkts": GetInt64(vals, "in-multicast-pkts"),
				"in_bcast_pkts": GetInt64(vals, "in-broadcast-pkts"),

				"out_octets":     GetInt64(vals, "out-octets"),
				"out_ucast_pkts": GetInt64(vals, "out-unicast-pkts"),
				"out_mcast_pkts": GetInt64(vals, "out-multicast-pkts"),
				"out_bcast_pkts": GetInt64(vals, "out-broadcast-pkts"),

				// Extra fields from gNMI not in original parser
				"in_errors":   GetInt64(vals, "in-errors"),
				"in_discards": GetInt64(vals, "in-discards"),
				"out_errors":  GetInt64(vals, "out-errors"),
				"out_discards": GetInt64(vals, "out-discards"),

				"has_ingress_data": GetString(vals, "in-octets") != "",
				"has_egress_data":  GetString(vals, "out-octets") != "",
			}

			results = append(results, NewCommonFields(dataTypeInterfaceCounters, msg))
		}
	}

	return results, nil
}
