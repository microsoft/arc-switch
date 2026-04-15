package transform

import (
	"gnmi-collector/internal/gnmi"
)

// NativeSystemCpuTransformer handles native Cisco NX-OS YANG CPU data
// from /System/procsys-items/syscpusummary-items.
type NativeSystemCpuTransformer struct{}

func init() {
	Register("nx-sys-cpu", func() Transformer { return &NativeSystemCpuTransformer{} })
	Register("nx-sys-memory", func() Transformer { return &NativeSystemMemoryTransformer{} })
	Register("nx-sys-load", func() Transformer { return &NativeSystemLoadTransformer{} })
}

func (t *NativeSystemCpuTransformer) DataType() string { return dataTypeSystemResources }

func (t *NativeSystemCpuTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	msg := map[string]interface{}{}
	var lastTS int64

	for _, n := range notifications {
		lastTS = n.Timestamp
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			msg["cpu_state_idle"] = GetString(vals, "idle")
			msg["cpu_state_kernel"] = GetString(vals, "kernel")
			msg["cpu_state_user"] = GetString(vals, "user")

			// Per-CPU data from syscpu-items/SysCpu-list
			if cpuItems := GetMap(vals, "syscpu-items"); cpuItems != nil {
				if cpuList := getSlice(cpuItems, "SysCpu-list"); cpuList != nil {
					var cpuUsages []map[string]interface{}
					for _, raw := range cpuList {
						cpu, ok := raw.(map[string]interface{})
						if !ok {
							continue
						}
						// NX-OS uses *-items sub-objects (idle-items, kernel-items, user-items)
						// each containing pct, avg, max, min fields
						cpuUsages = append(cpuUsages, map[string]interface{}{
							"cpuid":  GetString(cpu, "id"),
							"idle":   getItemsPct(cpu, "idle-items"),
							"kernel": getItemsPct(cpu, "kernel-items"),
							"user":   getItemsPct(cpu, "user-items"),
						})
					}
					msg["cpu_usage"] = cpuUsages
				}
			}

			// CPU usage history from syscpuhistory-items/SysCpuHistory-list
			if histItems := GetMap(vals, "syscpuhistory-items"); histItems != nil {
				if histList := getSlice(histItems, "SysCpuHistory-list"); histList != nil {
					var history []map[string]interface{}
					for _, raw := range histList {
						entry, ok := raw.(map[string]interface{})
						if !ok {
							continue
						}
						history = append(history, map[string]interface{}{
							"usage_avg": GetString(entry, "usageAvg"),
						})
					}
					msg["cpu_history"] = history
				}
			}
		}
	}

	result := NewCommonFields(dataTypeSystemResources, msg, lastTS)
	return []CommonFields{result}, nil
}

// NativeSystemLoadTransformer handles native Cisco NX-OS YANG load average data
// from /System/procsys-items/sysload-items.
type NativeSystemLoadTransformer struct{}

func (t *NativeSystemLoadTransformer) DataType() string { return dataTypeSystemResources }

func (t *NativeSystemLoadTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	msg := map[string]interface{}{}
	var lastTS int64

	for _, n := range notifications {
		lastTS = n.Timestamp
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			msg["load_avg_1min"] = GetFloat(vals, "loadAverage1m")
			msg["load_avg_5min"] = GetFloat(vals, "loadAverage5m")
			msg["load_avg_15min"] = GetFloat(vals, "loadAverage15m")
			msg["load_avg_5sec"] = GetFloat(vals, "loadAverage5sec")
			msg["processes_running"] = GetInt64(vals, "runProc")
			msg["processes_total"] = GetInt64(vals, "totalProc")
		}
	}

	result := NewCommonFields(dataTypeSystemResources, msg, lastTS)
	return []CommonFields{result}, nil
}

// getItemsPct extracts the "pct" (current percentage) from a native NX-OS
// *-items sub-object (e.g., idle-items, kernel-items, user-items).
func getItemsPct(m map[string]interface{}, key string) interface{} {
	if sub := GetMap(m, key); sub != nil {
		if v, ok := sub["pct"]; ok {
			return v
		}
		if v, ok := sub["avg"]; ok {
			return v
		}
	}
	return 0
}

// NativeSystemMemoryTransformer handles native Cisco NX-OS YANG memory data
// from /System/procsys-items/sysmem-items.
type NativeSystemMemoryTransformer struct{}

func (t *NativeSystemMemoryTransformer) DataType() string { return dataTypeSystemResources }

func (t *NativeSystemMemoryTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	msg := map[string]interface{}{}
	var lastTS int64

	for _, n := range notifications {
		lastTS = n.Timestamp
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			msg["memory_usage_total"] = ToInt64(vals["total"])
			msg["kernel_buffers"] = ToInt64(vals["buffers"])
			msg["kernel_cached"] = ToInt64(vals["cached"])
			msg["current_memory_status"] = GetString(vals, "memstatus")

			if freeItems := GetMap(vals, "sysmemfree-items"); freeItems != nil {
				msg["memory_usage_free"] = ToInt64(freeItems["curr"])
			}
			if usedItems := GetMap(vals, "sysmemused-items"); usedItems != nil {
				msg["memory_usage_used"] = ToInt64(usedItems["curr"])
			}
		}
	}

	result := NewCommonFields(dataTypeSystemResources, msg, lastTS)
	return []CommonFields{result}, nil
}
