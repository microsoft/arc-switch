package transform

import (
	"strings"

	"gnmi-collector/internal/gnmi"
)

const dataTypeInterfaceEthernet = "cisco_nexus_interface_ethernet"

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

			results = append(results, NewCommonFields(dataTypeInterfaceEthernet, msg))
		}
	}

	return results, nil
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
