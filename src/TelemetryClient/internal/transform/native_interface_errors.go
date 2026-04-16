package transform

import (
	"gnmi-collector/internal/gnmi"
)

const dataTypeInterfaceErrors = "interface_error_counters"

// NativeInterfaceErrorsTransformer handles native Cisco NX-OS YANG data
// from /System/intf-items/phys-items/PhysIf-list/dbgEtherStats-items.
type NativeInterfaceErrorsTransformer struct{}

func init() {
	Register("nx-intf-errors", func() Transformer { return &NativeInterfaceErrorsTransformer{} })
}

func (t *NativeInterfaceErrorsTransformer) DataType() string { return dataTypeInterfaceErrors }

func (t *NativeInterfaceErrorsTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			ifName := extractKey(u.Path, "id")

			// CRC field name varies between NX-OS versions
			crc := GetInt64(vals, "cRCAlignErrors")
			if crc == 0 {
				crc = GetInt64(vals, "CRCAlignErrors")
			}

			msg := map[string]interface{}{
				"interface_name":          NormalizeInterfaceName(ifName),
				"interface_type":          InterfaceType(ifName),
				"crc_align_errors":        crc,
				"collisions":              GetInt64(vals, "collisions"),
				"fragments":               GetInt64(vals, "fragments"),
				"jabbers":                 GetInt64(vals, "jabbers"),
				"overrun":                 GetInt64(vals, "overrun"),
				"pkts_64_octets":          GetInt64(vals, "pkts64Octets"),
				"pkts_65_to_127_octets":   GetInt64(vals, "pkts65to127Octets"),
				"pkts_128_to_255_octets":  GetInt64(vals, "pkts128to255Octets"),
				"pkts_256_to_511_octets":  GetInt64(vals, "pkts256to511Octets"),
				"pkts_512_to_1023_octets": GetInt64(vals, "pkts512to1023Octets"),
				"pkts_1024_to_1518_octets": GetInt64(vals, "pkts1024to1518Octets"),
				"broadcast_pkts":          GetInt64(vals, "broadcastPkts"),
				"multicast_pkts":          GetInt64(vals, "multicastPkts"),
			}

			results = append(results, NewCommonFields(dataTypeInterfaceErrors, msg, n.Timestamp))
		}
	}

	return results, nil
}
