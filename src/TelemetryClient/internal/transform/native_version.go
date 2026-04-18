package transform

import (
	"gnmi-collector/internal/gnmi"
)

const dataTypeVersion = "version"

// NativeVersionTransformer handles native Cisco NX-OS YANG version data
// from /System/showversion-items.
type NativeVersionTransformer struct{}

func init() {
	Register("nx-version", func() Transformer { return &NativeVersionTransformer{} })
}

func (t *NativeVersionTransformer) DataType() string { return dataTypeVersion }

func (t *NativeVersionTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	msg := map[string]interface{}{}
	var lastTS int64

	for _, n := range notifications {
		lastTS = n.Timestamp
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			msg["nxos_version"] = GetString(vals, "nxosVersion")
			msg["bios_version"] = GetString(vals, "biosVersion")
			msg["nxos_image_file"] = GetString(vals, "nxosImageFile")
			msg["nxos_compile_time"] = GetString(vals, "nxosCompileTime")
			msg["nxos_timestamp"] = GetString(vals, "nxosTimestamp")
			msg["last_reset_reason"] = GetString(vals, "lastResetReason")
			msg["kernel_uptime"] = GetString(vals, "kernelUptime")
			msg["chassis_id"] = GetString(vals, "chassisId")
			msg["cpu_name"] = GetString(vals, "cpuName")
			msg["memory_kb"] = GetInt64(vals, "mem")
			msg["device_name"] = GetString(vals, "hostName")
			msg["boot_mode"] = GetString(vals, "bootMode")
		}
	}

	result := NewCommonFields(dataTypeVersion, msg, lastTS)
	return []CommonFields{result}, nil
}
