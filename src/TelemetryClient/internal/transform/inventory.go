package transform

import (
	"gnmi-collector/internal/gnmi"
)

const dataTypeInventory = "cisco_nexus_inventory"

func init() {
	Register("platform-inventory", func() Transformer { return &InventoryTransformer{} })
}

type InventoryTransformer struct{}

func (t *InventoryTransformer) DataType() string { return dataTypeInventory }

func (t *InventoryTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			// Platform inventory updates can be arrays or objects
			components := normalizeComponentList(u.Value)

			for _, comp := range components {
				state := GetMap(comp, "state")
				if state == nil {
					continue
				}

				name := GetString(state, "name")
				if name == "" {
					name = extractKey(u.Path, "name")
				}

				msg := map[string]interface{}{
					"name":           name,
					"description":    GetString(state, "description"),
					"product_id":     GetString(state, "part-no"),
					"version_id":     GetString(state, "hardware-version"),
					"serial_number":  GetString(state, "serial-no"),
					"component_type": deriveComponentType(GetString(state, "type")),
				}

				results = append(results, NewCommonFields(dataTypeInventory, msg))
			}
		}
	}

	return results, nil
}

func normalizeComponentList(val interface{}) []map[string]interface{} {
	switch v := val.(type) {
	case []interface{}:
		var result []map[string]interface{}
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				result = append(result, m)
			}
		}
		return result
	case map[string]interface{}:
		return []map[string]interface{}{v}
	default:
		return nil
	}
}

func deriveComponentType(ocType string) string {
	switch ocType {
	case "CHASSIS":
		return "chassis"
	case "LINECARD":
		return "slot"
	case "POWER_SUPPLY":
		return "power_supply"
	case "FAN":
		return "fan"
	case "CPU":
		return "cpu"
	case "TRANSCEIVER":
		return "transceiver"
	default:
		return "unknown"
	}
}
