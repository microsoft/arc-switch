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
Register("nx-fan", func() Transformer { return &NativeEnvFanTransformer{} })
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

// NativeEnvFanTransformer handles native Cisco NX-OS YANG fan data
// from /System/ch-items/ftslot-items/FtSlot-list.
type NativeEnvFanTransformer struct{}

func (t *NativeEnvFanTransformer) DataType() string { return "fan" }

func (t *NativeEnvFanTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
var results []CommonFields

for _, n := range notifications {
for _, u := range n.Updates {
entries := AsMapSlice(u.Value)
if entries == nil {
continue
}

for _, vals := range entries {
id := GetString(vals, "id")
if id == "" {
id = extractKey(u.Path, "id")
}

// NX-OS structure: FtSlot-list -> ft-items -> fan-items -> Fan-list
ft := GetMap(vals, "ft-items")
if ft == nil {
// Try direct fan-items at slot level
ft = vals
}

fanList := GetSlice(ft, "Fan-list")
if fanList == nil {
// Flat structure: extract from ft-items directly
msg := map[string]interface{}{
"name":      id,
"model":     GetString(ft, "model"),
"direction": GetString(ft, "dir"),
"status":    GetString(ft, "operSt"),
"serial":    GetString(ft, "ser"),
}
results = append(results, NewCommonFields("fan", msg, n.Timestamp))
continue
}

// Nested structure: iterate over Fan-list entries
for _, raw := range fanList {
fan, ok := raw.(map[string]interface{})
if !ok {
continue
}
fanID := GetString(fan, "id")
name := id
if fanID != "" {
name = id + "/" + fanID
}
msg := map[string]interface{}{
"name":      name,
"model":     GetString(ft, "model"),
"direction": GetString(fan, "dir"),
"status":    GetString(fan, "operSt"),
"serial":    GetString(ft, "ser"),
}
results = append(results, NewCommonFields("fan", msg, n.Timestamp))
}
}
}
}

return results, nil
}

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