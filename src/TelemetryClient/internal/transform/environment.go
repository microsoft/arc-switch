package transform

import (
"fmt"
"strings"

"gnmi-collector/internal/gnmi"
)

const dataTypeEnvTemp = "environment_temperature"

func init() {
Register("temperature", func() Transformer { return &EnvironmentTempTransformer{} })
Register("power-supply", func() Transformer { return &EnvironmentPowerTransformer{} })
}

type EnvironmentTempTransformer struct{}

func (t *EnvironmentTempTransformer) DataType() string { return dataTypeEnvTemp }

func (t *EnvironmentTempTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
var results []CommonFields

for _, n := range notifications {
for _, u := range n.Updates {
vals, ok := u.Value.(map[string]interface{})
if !ok {
continue
}

componentName := extractKey(u.Path, "name")

// Derive module and sensor from component name
// e.g., "Sensor 27" -> module "", sensor "Sensor 27"
// e.g., "Module1 Sensor FRONT" -> module "1", sensor "FRONT"
module, sensor := parseComponentSensor(componentName)

msg := map[string]interface{}{
"module":                  module,
"sensor":                  sensor,
"current_temp":            GetFirstString(vals, "instant", "avg", "max"),
"high_threshold":          GetFirstString(vals, "alarm-threshold", "critical-high"),
"critical_high_threshold": GetFirstString(vals, "critical-high", "alarm-threshold"),
"low_threshold":           GetFirstString(vals, "warning-threshold", "warning-high"),
"status":                  deriveTempStatus(GetBool(vals, "alarm-status")),
}

results = append(results, NewCommonFields(dataTypeEnvTemp, msg, n.Timestamp))
}
}

return results, nil
}

func parseComponentSensor(name string) (module, sensor string) {
// Try to parse patterns like "Module1 Sensor FRONT"
lower := strings.ToLower(name)
if strings.Contains(lower, "module") {
parts := strings.Fields(name)
for _, p := range parts {
pl := strings.ToLower(p)
if strings.HasPrefix(pl, "module") {
module = strings.TrimPrefix(pl, "module")
}
}
// Sensor is the last word (FRONT, BACK, CPU, etc.)
if len(parts) > 0 {
sensor = parts[len(parts)-1]
}
} else {
sensor = name
}
return module, sensor
}

func deriveTempStatus(alarm bool) string {
if alarm {
return "Alert"
}
return "Ok"
}

const dataTypeEnvPower = "environment_power"

type EnvironmentPowerTransformer struct{}

func (t *EnvironmentPowerTransformer) DataType() string { return dataTypeEnvPower }

func (t *EnvironmentPowerTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
var results []CommonFields

for _, n := range notifications {
for _, u := range n.Updates {
vals, ok := u.Value.(map[string]interface{})
if !ok {
continue
}

psName := extractKey(u.Path, "name")
state := GetMap(vals, "state")
if state == nil {
state = vals
}

msg := map[string]interface{}{
"ps_name": psName,
"status":  derivePowerStatus(GetBool(state, "enabled")),
"model":   GetFirstString(state, "description", "model", "part-no"),
}

// Decode base64-encoded float values
for _, field := range []struct{ src, dst string }{
{"capacity", "total_capacity"},
{"input-current", "input_current"},
{"input-voltage", "input_voltage"},
{"output-current", "output_current"},
{"output-power", "output_power"},
{"output-voltage", "output_voltage"},
} {
raw := GetString(state, field.src)
if raw != "" {
if f, err := DecodeBase64Float32(raw); err == nil {
msg[field.dst] = fmt.Sprintf("%.2f", f)
} else {
msg[field.dst] = raw
}
}
}

results = append(results, NewCommonFields(dataTypeEnvPower, msg, n.Timestamp))
}
}

return results, nil
}

func derivePowerStatus(enabled bool) string {
if enabled {
return "Ok"
}
return "Shutdown"
}