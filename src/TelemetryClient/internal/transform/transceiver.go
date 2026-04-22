package transform

import (
	"gnmi-collector/internal/gnmi"
)

const dataTypeTransceiver = "transceiver"

func init() {
	Register("transceiver", func() Transformer { return &TransceiverTransformer{} })
}

type TransceiverTransformer struct{}

func (t *TransceiverTransformer) DataType() string { return dataTypeTransceiver }

func (t *TransceiverTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			ifName := extractKey(u.Path, "name")
			state := GetMap(vals, "state")
			if state == nil {
				state = vals
			}

			present := GetFirstString(state, "present", "empty")
			formFactor := GetFirstString(state, "form-factor", "form-factor-preconf", "ethernet-pmd-preconf")
			vendor := GetFirstString(state, "vendor", "mfg-name")
			vendorPart := GetFirstString(state, "vendor-part", "vendor-part-number")
			vendorRev := GetFirstString(state, "vendor-rev", "vendor-revision")
			serialNo := GetFirstString(state, "serial-no", "serial-number")

			// Check for DOM channel diagnostics (openconfig-platform-transceiver)
			domSupported := false
			var domData map[string]interface{}
			if channels := GetMap(vals, "physical-channels"); channels != nil {
				if chList := GetSlice(channels, "channel"); len(chList) > 0 {
					domSupported = true
					ch, _ := chList[0].(map[string]interface{})
					if ch != nil {
						if chState := GetMap(ch, "state"); chState != nil {
							domData = map[string]interface{}{
								"output_power": GetFirstString(chState, "output-power", "laser-bias-current"),
								"input_power":  GetString(chState, "input-power"),
							}
						}
					}
				}
			}

			msg := map[string]interface{}{
				"interface_name":      NormalizeInterfaceName(ifName),
				"transceiver_present": present == "PRESENT" || present == "true",
				"type":                formFactor,
				"manufacturer":        vendor,
				"part_number":         vendorPart,
				"revision":            vendorRev,
				"serial_number":       serialNo,
				"dom_supported":       domSupported,
				// Cisco-specific fields not available via OpenConfig YANG —
				// require native Cisco YANG paths or CLI fallback
				"nominal_bitrate":   GetString(state, "nominal-bitrate"),
				"link_length":       GetString(state, "link-length"),
				"cable_type":        GetString(state, "cable-type"),
				"cisco_id":          GetString(state, "cisco-id"),
				"cisco_extended_id":  GetString(state, "cisco-extended-id"),
				"cisco_part_number": GetString(state, "cisco-part-number"),
				"cisco_product_id":  GetString(state, "cisco-product-id"),
				"cisco_version_id":  GetString(state, "cisco-version-id"),
			}

			if domData != nil {
				msg["dom_data"] = domData
			}

			results = append(results, NewCommonFields(dataTypeTransceiver, msg, n.Timestamp))
		}
	}

	return results, nil
}
