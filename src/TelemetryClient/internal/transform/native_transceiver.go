package transform

import (
	"fmt"

	"gnmi-collector/internal/gnmi"
)

// NativeTransceiverTransformer handles native Cisco NX-OS YANG transceiver data
// from /System/intf-items/phys-items/PhysIf-list/phys-items.
type NativeTransceiverTransformer struct{}

func init() {
	Register("nx-transceiver", func() Transformer { return &NativeTransceiverTransformer{} })
}

func (t *NativeTransceiverTransformer) DataType() string { return dataTypeTransceiver }

func (t *NativeTransceiverTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			ifName := extractKey(u.Path, "id")

			msg := map[string]interface{}{
				"interface_name": NormalizeInterfaceName(ifName),
				"speed":          GetString(vals, "operSpeed"),
				"duplex":         GetString(vals, "operDuplex"),
				"description":    GetString(vals, "operDescr"),
			}

			fcot := GetMap(vals, "fcot-items")
			if fcot == nil || !GetBool(fcot, "isFcotPresent") {
				msg["transceiver_present"] = false
				results = append(results, NewCommonFields(dataTypeTransceiver, msg))
				continue
			}

			msg["transceiver_present"] = true
			msg["type"] = GetString(fcot, "typeName")
			msg["manufacturer"] = GetString(fcot, "vendorName")
			msg["part_number"] = GetString(fcot, "vendorPn")
			msg["revision"] = GetString(fcot, "vendorRev")
			msg["serial_number"] = GetString(fcot, "vendorSn")
			msg["nominal_bitrate"] = formatBitrate(fcot)
			msg["link_length"] = formatLinkLength(fcot)
			msg["cable_type"] = GetString(fcot, "transceiverType")
			msg["cisco_id"] = GetString(fcot, "xcvrId")
			msg["cisco_extended_id"] = GetString(fcot, "xcvrExtId")
			msg["cisco_part_number"] = GetString(fcot, "partNumber")
			msg["cisco_product_id"] = GetString(fcot, "typeName")
			msg["cisco_version_id"] = GetString(fcot, "versionId")
			msg["dom_supported"] = nativeIntVal(fcot, "diagMonType") != 0

			// Extract DOM sensor data when diagnostics monitoring is supported
			if nativeIntVal(fcot, "diagMonType") != 0 {
				laneItems := GetMap(fcot, "lane-items")
				if laneItems != nil {
					sensors := AsMapSlice(laneItems["FcotSensor-list"])

					domData := map[string]interface{}{}
					sensorNames := map[string]string{
						"1": "temperature", "2": "voltage", "3": "current",
						"4": "tx_power", "5": "rx_power",
					}

					for _, sensor := range sensors {
						sensorId := GetString(sensor, "sensorId")
						name, exists := sensorNames[sensorId]
						if !exists {
							continue
						}

						domData[name+"_instant"] = GetFloat(sensor, "value")
						domData[name+"_high_alarm"] = GetFloat(sensor, "highAlarm")
						domData[name+"_high_warn"] = GetFloat(sensor, "highWarn")
						domData[name+"_low_alarm"] = GetFloat(sensor, "lowAlarm")
						domData[name+"_low_warn"] = GetFloat(sensor, "lowWarn")
						domData[name+"_min"] = GetFloat(sensor, "min")
						domData[name+"_max"] = GetFloat(sensor, "max")
						domData[name+"_avg"] = GetFloat(sensor, "avg")
						domData[name+"_alert"] = GetString(sensor, "alert")
					}
					if len(domData) > 0 {
						msg["dom_data"] = domData
					}
				}
			}

			results = append(results, NewCommonFields(dataTypeTransceiver, msg))
		}
	}

	return results, nil
}

func formatBitrate(fcot map[string]interface{}) string {
	br := nativeIntVal(fcot, "brIn100MHz")
	if br == 0 {
		return ""
	}
	return fmt.Sprintf("%d MHz", br*100)
}

func formatLinkLength(fcot map[string]interface{}) string {
	if v := nativeIntVal(fcot, "distIn1mForCu"); v > 0 {
		return fmt.Sprintf("%d m (Cu)", v)
	}
	if v := nativeIntVal(fcot, "distIn10mFor50u"); v > 0 {
		return fmt.Sprintf("%d m (50um)", v*10)
	}
	if v := nativeIntVal(fcot, "distInKmFor9u"); v > 0 {
		return fmt.Sprintf("%d km (9um)", v)
	}
	return ""
}

// nativeIntVal extracts an integer value from a map, handling both int and float64 JSON types.
func nativeIntVal(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}
