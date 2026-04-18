package transform

import (
"gnmi-collector/internal/gnmi"
)

// NativeEnvTempTransformer handles native Cisco NX-OS YANG temperature sensor data
// from /System/ch-items/supslot-items/SupCSlot-list/sup-items/sensor-items.
type NativeEnvTempTransformer struct{}

func init() {
Register("nx-env-sensor", func() Transformer { return &NativeEnvTempTransformer{} })
Register("nx-env-psu", func() Transformer { return &NativeEnvPowerTransformer{} })
}

func (t *NativeEnvTempTransformer) DataType() string { return dataTypeEnvTemp }

func (t *NativeEnvTempTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
var results []CommonFields

for _, n := range notifications {
for _, u := range n.Updates {
vals, ok := u.Value.(map[string]interface{})
if !ok {
continue
}

module := extractKey(u.Path, "id")

sensors := GetSlice(vals, "Sensor-list")
if sensors == nil {
// Value may be a single sensor entry rather than a list.
msg := nativeTempMsg(module, vals)
if msg != nil {
results = append(results, NewCommonFields(dataTypeEnvTemp, msg, n.Timestamp))
}
continue
}

for _, raw := range sensors {
sensor, ok := raw.(map[string]interface{})
if !ok {
continue
}
msg := nativeTempMsg(module, sensor)
if msg != nil {
results = append(results, NewCommonFields(dataTypeEnvTemp, msg, n.Timestamp))
}
}
}
}

return results, nil
}

func nativeTempMsg(module string, sensor map[string]interface{}) map[string]interface{} {
return map[string]interface{}{
"module":          module,
"sensor":          GetString(sensor, "descr"),
"current_temp":    GetString(sensor, "tempValue"),
"high_threshold":  GetString(sensor, "majorThresh"),
"low_threshold":   GetString(sensor, "minorThresh"),
"status":          GetString(sensor, "operSt"),
}
}

// NativeEnvPowerTransformer handles native Cisco NX-OS YANG PSU data
// from /System/ch-items/psuslot-items/PsuSlot-list.
type NativeEnvPowerTransformer struct{}

func (t *NativeEnvPowerTransformer) DataType() string { return dataTypeEnvPower }

func (t *NativeEnvPowerTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
var results []CommonFields

for _, n := range notifications {
for _, u := range n.Updates {
// NX-OS returns arrays for YANG list nodes (PsuSlot-list)
entries := AsMapSlice(u.Value)
if entries == nil {
continue
}

for _, vals := range entries {
psu := GetMap(vals, "psu-items")
if psu == nil {
continue
}

psNumber := GetString(vals, "id")
if psNumber == "" {
psNumber = extractKey(u.Path, "id")
}

msg := map[string]interface{}{
"ps_name":         psNumber,
"model":           GetString(psu, "model"),
"serial":          GetString(psu, "ser"),
"vendor":          GetString(psu, "vendor"),
"status":          GetString(psu, "operSt"),
"total_capacity":  GetString(psu, "cap"),
"input_voltage":   GetString(psu, "vIn"),
"input_current":   GetString(psu, "iIn"),
"output_voltage":  GetString(psu, "vOut"),
"output_current":  GetString(psu, "iOut"),
"output_power":    GetString(psu, "pOut"),
"actual_input":    GetString(psu, "pIn"),
"actual_output":   GetString(psu, "pOut"),
"cord_status":     GetString(psu, "typeCordConnected"),
"software_alarm":  GetBool(psu, "softwareAlarm"),
"hardware_alarm":  GetString(psu, "hardwareAlarm"),
"fan_direction":   GetString(psu, "fanDirection"),
"fan_status":      GetString(psu, "fanOpSt"),
}

results = append(results, NewCommonFields(dataTypeEnvPower, msg, n.Timestamp))
}
}
}

return results, nil
}