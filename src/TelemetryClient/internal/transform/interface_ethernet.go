package transform

import (
	"strings"

	"gnmi-collector/internal/gnmi"
)

const dataTypeInterfaceEthernet = "interface_ethernet"

func init() {
	Register("if-ethernet", func() Transformer { return &InterfaceEthernetTransformer{} })
}

type InterfaceEthernetTransformer struct{}

func (t *InterfaceEthernetTransformer) DataType() string {
	return dataTypeInterfaceEthernet
}

func (t *InterfaceEthernetTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			ifName := extractKey(u.Path, "name")
			if ifName == "" {
				continue
			}

			speed := GetFirstString(vals, "negotiated-port-speed", "port-speed")
			speed = normalizeSpeed(speed)

			duplex := GetFirstString(vals, "negotiated-duplex-mode", "duplex-mode")

			msg := map[string]interface{}{
				"interface_name": NormalizeInterfaceName(ifName),
				"speed":          speed,
				"duplex":         strings.ToLower(duplex),
				"auto_negotiate": GetBool(vals, "auto-negotiate"),
				"mac_address":    GetString(vals, "mac-address"),
				"hw_mac_address": GetString(vals, "hw-mac-address"),
			}

			// Extract ethernet-specific counters if present (SONiC provides these)
			if counters := GetMap(vals, "counters"); counters != nil {
				msg["in_crc_errors"] = getInt64Multi(counters, "in-crc-errors", "openconfig-if-ethernet:in-crc-errors")
				msg["in_fragment_frames"] = getInt64Multi(counters, "in-fragment-frames", "openconfig-if-ethernet:in-fragment-frames")
				msg["in_jabber_frames"] = getInt64Multi(counters, "in-jabber-frames", "openconfig-if-ethernet:in-jabber-frames")
				msg["in_oversize_frames"] = getInt64Multi(counters, "in-oversize-frames", "openconfig-if-ethernet:in-oversize-frames")
				msg["in_undersize_frames"] = getInt64Multi(counters, "in-undersize-frames", "openconfig-if-ethernet:in-undersize-frames")
				msg["out_crc_errors"] = getInt64Multi(counters, "out-crc-errors", "openconfig-if-ethernet:out-crc-errors")
			}

			results = append(results, NewCommonFields(dataTypeInterfaceEthernet, msg, n.Timestamp))
		}
	}

	return results, nil
}

// getInt64Multi returns the first non-zero int64 value found for any of the given keys.
func getInt64Multi(m map[string]interface{}, keys ...string) int64 {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return ToInt64(v)
		}
	}
	return 0
}

// normalizeSpeed converts OpenConfig speed enums to short form.
func normalizeSpeed(speed string) string {
	switch speed {
	case "SPEED_10MB":
		return "10M"
	case "SPEED_100MB":
		return "100M"
	case "SPEED_1GB":
		return "1G"
	case "SPEED_10GB":
		return "10G"
	case "SPEED_25GB":
		return "25G"
	case "SPEED_40GB":
		return "40G"
	case "SPEED_50GB":
		return "50G"
	case "SPEED_100GB":
		return "100G"
	case "SPEED_200GB":
		return "200G"
	case "SPEED_400GB":
		return "400G"
	default:
		s := strings.TrimPrefix(speed, "SPEED_")
		s = strings.TrimSuffix(s, "B")
		return s
	}
}
