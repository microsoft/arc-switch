package transform

import (
	"fmt"
	"strconv"
	"time"

	"gnmi-collector/internal/gnmi"
)

const dataTypeSystemResources = "system_resources"

func init() {
	Register("system-cpus", func() Transformer { return &SystemResourcesTransformer{} })
	Register("system-memory", func() Transformer { return &SystemResourcesTransformer{} })
	Register("system-state", func() Transformer { return &SystemUptimeTransformer{} })
}

// SystemResourcesTransformer combines CPU and memory gNMI data into the
// schema matching the current system-resources parser.
type SystemResourcesTransformer struct{}

func (t *SystemResourcesTransformer) DataType() string { return dataTypeSystemResources }

func (t *SystemResourcesTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	msg := map[string]interface{}{}
	var lastTS int64

	for _, n := range notifications {
		lastTS = n.Timestamp
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			// CPU data (from /system/cpus)
			if cpuList := GetSlice(vals, "cpu"); cpuList != nil {
				var cpuUsages []map[string]interface{}
				for _, raw := range cpuList {
					cpu, ok := raw.(map[string]interface{})
					if !ok {
						continue
					}
					state := GetMap(cpu, "state")
					if state == nil {
						continue
					}

					cpuEntry := map[string]interface{}{
						// In poll mode, index is inside state; in subscribe
						// mode, array-aware normalization puts the list key
						// ("index") at the CPU entry level.
						"cpuid":  getCpuId(cpu, state),
						"user":   getStatInstant(state, "user"),
						"kernel": getStatInstant(state, "kernel"),
						"idle":   getStatInstant(state, "idle"),
					}
					cpuUsages = append(cpuUsages, cpuEntry)
				}
				msg["cpu_usage"] = cpuUsages

				// Derive aggregate CPU stats from average across all cores
				if len(cpuUsages) > 0 {
					var totalUser, totalKernel, totalIdle float64
					for _, c := range cpuUsages {
						totalUser += toFloat(c["user"])
						totalKernel += toFloat(c["kernel"])
						totalIdle += toFloat(c["idle"])
					}
					count := float64(len(cpuUsages))
					msg["cpu_state_user"] = fmt.Sprintf("%.1f", totalUser/count)
					msg["cpu_state_kernel"] = fmt.Sprintf("%.1f", totalKernel/count)
					msg["cpu_state_idle"] = fmt.Sprintf("%.1f", totalIdle/count)
				}
			}

			// Memory data (from /system/memory)
			if memState := GetMap(vals, "state"); memState != nil {
				physical := GetString(memState, "physical")
				reserved := GetString(memState, "reserved")

				physBytes, _ := strconv.ParseInt(physical, 10, 64)
				resBytes, _ := strconv.ParseInt(reserved, 10, 64)

				msg["memory_usage_total"] = physBytes / 1024
				msg["memory_usage_reserved"] = resBytes / 1024

				if physBytes > 0 {
					msg["memory_usage_used"] = resBytes / 1024
					msg["memory_usage_free"] = (physBytes - resBytes) / 1024
				}
			}

			// Fields not available via OpenConfig YANG — require CLI or native YANG
			// load_avg_1min, load_avg_5min, load_avg_15min
			// processes_total, processes_running
			// kernel_vmalloc_total, kernel_vmalloc_free
			// kernel_buffers, kernel_cached
		}
	}

	result := NewCommonFields(dataTypeSystemResources, msg, lastTS)
	return []CommonFields{result}, nil
}

// getCpuId extracts the CPU identifier, checking the entry level first
// (subscribe mode puts the list key at the entry level) then state level
// (poll mode has index inside the state container).
func getCpuId(cpu, state map[string]interface{}) string {
	if id := GetString(cpu, "index"); id != "" {
		return id
	}
	return GetString(state, "index")
}

// getStatInstant extracts the instantaneous value for a CPU stat.
// In poll mode (OpenConfig Get), the structure is nested:
//
//	state["idle"] = {"instant": 72.0}
//
// In subscribe mode, the structure may already be flat if the gNMI
// server sends the value directly at the leaf level:
//
//	state["idle"] = 72
func getStatInstant(state map[string]interface{}, key string) interface{} {
	if sub := GetMap(state, key); sub != nil {
		return sub["instant"]
	}
	// Fallback: the value may be at this level directly (subscribe mode)
	if v, ok := state[key]; ok {
		return v
	}
	return 0
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case uint64:
		return float64(n)
	case int32:
		return float64(n)
	case uint32:
		return float64(n)
	default:
		return 0
	}
}

const dataTypeSystemUptime = "system_uptime"

// SystemUptimeTransformer converts gNMI system state data to the schema
// matching the current system-uptime parser.
type SystemUptimeTransformer struct{}

func (t *SystemUptimeTransformer) DataType() string { return dataTypeSystemUptime }

func (t *SystemUptimeTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	msg := map[string]interface{}{}
	var lastTS int64

	for _, n := range notifications {
		lastTS = n.Timestamp
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			msg["hostname"] = GetString(vals, "hostname")
			msg["domain_name"] = GetString(vals, "domain-name")

			// Derive uptime from boot-time (nanoseconds since epoch)
			bootTimeStr := GetString(vals, "boot-time")
			if bootTimeStr != "" {
				bootNanos, err := strconv.ParseInt(bootTimeStr, 10, 64)
				if err == nil {
					bootTime := time.Unix(0, bootNanos)
					uptime := time.Since(bootTime)

					days := int(uptime.Hours()) / 24
					hours := int(uptime.Hours()) % 24
					minutes := int(uptime.Minutes()) % 60
					seconds := int(uptime.Seconds()) % 60

					msg["system_uptime_days"] = days
					msg["system_uptime_hours"] = hours
					msg["system_uptime_minutes"] = minutes
					msg["system_uptime_seconds"] = seconds
					msg["system_uptime_total"] = fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
					msg["system_start_time"] = bootTime.Format(time.RFC3339)

					// Kernel uptime mirrors system uptime (single boot-time source in YANG)
					msg["kernel_uptime_days"] = days
					msg["kernel_uptime_hours"] = hours
					msg["kernel_uptime_minutes"] = minutes
					msg["kernel_uptime_seconds"] = seconds
					msg["kernel_uptime_total"] = fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
				}
			}

			if dt := GetString(vals, "current-datetime"); dt != "" {
				msg["current_datetime"] = dt
			}
		}
	}

	result := NewCommonFields(dataTypeSystemUptime, msg, lastTS)
	return []CommonFields{result}, nil
}
