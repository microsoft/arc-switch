package transform

import "gnmi-collector/internal/gnmi"

func init() {
	Register("sonic-temperature", func() Transformer { return &SonicTemperatureTransformer{} })
	Register("sonic-psu", func() Transformer { return &SonicPsuTransformer{} })
	Register("sonic-fan", func() Transformer { return &SonicFanTransformer{} })
}

// SonicTemperatureTransformer extracts temperature sensor data from
// the SONiC native sonic-platform response.
type SonicTemperatureTransformer struct{}

func (t *SonicTemperatureTransformer) DataType() string { return dataTypeEnvTemp }

func (t *SonicTemperatureTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields
	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}
			if tempInfo := GetMap(vals, "TEMPERATURE_INFO"); tempInfo != nil {
				tempList := AsMapSlice(tempInfo["TEMPERATURE_INFO_LIST"])
				for _, sensor := range tempList {
					msg := map[string]interface{}{
						"sensor":                  GetString(sensor, "name"),
						"current_temp":             GetString(sensor, "temperature"),
						"high_threshold":            GetString(sensor, "high_threshold"),
						"critical_high_threshold":   GetString(sensor, "critical_high_threshold"),
						"low_threshold":             GetString(sensor, "low_threshold"),
						"warning_status":            GetString(sensor, "warning_status"),
						"timestamp":                 GetString(sensor, "timestamp"),
					}
					results = append(results, NewCommonFields(dataTypeEnvTemp, msg, n.Timestamp))
				}
			}
		}
	}
	return results, nil
}

// SonicPsuTransformer extracts power supply data from the SONiC
// native sonic-platform response.
type SonicPsuTransformer struct{}

func (t *SonicPsuTransformer) DataType() string { return "psu" }

func (t *SonicPsuTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields
	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}
			if psuInfo := GetMap(vals, "PSU_INFO"); psuInfo != nil {
				psuList := AsMapSlice(psuInfo["PSU_INFO_LIST"])
				for _, psu := range psuList {
					msg := map[string]interface{}{
						"name":           GetString(psu, "name"),
						"status":         GetString(psu, "status"),
						"model":          GetString(psu, "model"),
						"serial":         GetString(psu, "serial"),
						"temp":           GetString(psu, "temp"),
						"input_voltage":  GetString(psu, "input_voltage"),
						"input_current":  GetString(psu, "input_current"),
						"output_voltage": GetString(psu, "output_voltage"),
						"output_current": GetString(psu, "output_current"),
						"output_power":   GetString(psu, "output_power"),
					}
					results = append(results, NewCommonFields("psu", msg, n.Timestamp))
				}
			}
		}
	}
	return results, nil
}

// SonicFanTransformer extracts fan data from the SONiC native
// sonic-platform response.
type SonicFanTransformer struct{}

func (t *SonicFanTransformer) DataType() string { return "fan" }

func (t *SonicFanTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields
	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}
			if fanInfo := GetMap(vals, "FAN_INFO"); fanInfo != nil {
				fanList := AsMapSlice(fanInfo["FAN_INFO_LIST"])
				for _, fan := range fanList {
					msg := map[string]interface{}{
						"name":        GetString(fan, "name"),
						"speed":       GetString(fan, "speed"),
						"direction":   GetString(fan, "direction"),
						"model":       GetString(fan, "model"),
						"serial":      GetString(fan, "serial"),
						"status":      GetString(fan, "status"),
						"drawer_name": GetString(fan, "drawer_name"),
					}
					results = append(results, NewCommonFields("fan", msg, n.Timestamp))
				}
			}
		}
	}
	return results, nil
}
