package transform

import (
	"gnmi-collector/internal/gnmi"
)

const dataTypeInterfaceStatus = "cisco_nexus_interface_status"

func init() {
	Register("interface-status", func() Transformer { return &InterfaceStatusTransformer{} })
}

// InterfaceStatusTransformer converts gNMI interface state data to the
// schema matching the existing cisco-parser interface-status output.
type InterfaceStatusTransformer struct{}

func (t *InterfaceStatusTransformer) DataType() string {
	return dataTypeInterfaceStatus
}

func (t *InterfaceStatusTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var entries []InterfaceStatusEntry

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			ifName := ExtractInterfaceName(u.Path)
			if ifName == "" {
				ifName = GetString(vals, "name")
			}
			if ifName == "" {
				continue
			}

			normalized := NormalizeInterfaceName(ifName)
			adminStatus := GetString(vals, "admin-status")
			operStatus := GetString(vals, "oper-status")

			entry := InterfaceStatusEntry{
				Port:   normalized,
				Name:   GetString(vals, "description"),
				Status: deriveStatus(adminStatus, operStatus),
				Vlan:   "",  // Not available in base interface state
				Duplex: "",  // Requires openconfig-if-ethernet path
				Speed:  "",  // Requires openconfig-if-ethernet path
				Type:   GetString(vals, "type"),
			}

			entries = append(entries, entry)
		}
	}

	// The current parser returns a single entry containing all interfaces
	msg := map[string]interface{}{
		"interfaces": entries,
	}
	result := NewCommonFields(dataTypeInterfaceStatus, msg)
	return []CommonFields{result}, nil
}

type InterfaceStatusEntry struct {
	Port   string `json:"port"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Vlan   string `json:"vlan"`
	Duplex string `json:"duplex"`
	Speed  string `json:"speed"`
	Type   string `json:"type"`
}

// deriveStatus maps OpenConfig admin/oper status to the status strings
// used by the current parser (connected, notconnec, disabled, etc.)
func deriveStatus(admin, oper string) string {
	admin = normalizeStatus(admin)
	oper = normalizeStatus(oper)

	if admin == "DOWN" {
		return "disabled"
	}
	if oper == "UP" {
		return "connected"
	}
	if oper == "DOWN" {
		return "notconnec"
	}
	return "down"
}

func normalizeStatus(s string) string {
	switch s {
	case "UP", "up":
		return "UP"
	case "DOWN", "down":
		return "DOWN"
	default:
		return s
	}
}
