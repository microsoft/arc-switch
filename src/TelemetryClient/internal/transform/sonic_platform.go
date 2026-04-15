package transform

import "gnmi-collector/internal/gnmi"

const (
	dataTypeSonicTemperature = "sonic_temperature"
	dataTypeSonicPsu         = "sonic_psu"
	dataTypeSonicFan         = "sonic_fan"
)

func init() {
	Register("sonic-platform", func() Transformer { return &SonicPlatformTransformer{} })
}

type SonicPlatformTransformer struct{}

func (t *SonicPlatformTransformer) DataType() string { return dataTypeSonicTemperature }

func (t *SonicPlatformTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			// Temperature sensors
			if tempInfo := GetMap(vals, "TEMPERATURE_INFO"); tempInfo != nil {
				tempList := AsMapSlice(tempInfo["TEMPERATURE_INFO_LIST"])
				for _, sensor := range tempList {
					msg := map[string]interface{}{
						"sensor":                 GetString(sensor, "name"),
						"current_temp":            GetString(sensor, "temperature"),
						"high_threshold":           GetString(sensor, "high_threshold"),
						"critical_high_threshold":  GetString(sensor, "critical_high_threshold"),
						"low_threshold":            GetString(sensor, "low_threshold"),
						"warning_status":           GetString(sensor, "warning_status"),
						"timestamp":                GetString(sensor, "timestamp"),
					}
					results = append(results, NewCommonFields(dataTypeSonicTemperature, msg, n.Timestamp))
				}
			}

			// PSU
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
					results = append(results, NewCommonFields(dataTypeSonicPsu, msg, n.Timestamp))
				}
			}

			// Fans
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
					results = append(results, NewCommonFields(dataTypeSonicFan, msg, n.Timestamp))
				}
			}
		}
	}

	return results, nil
}
