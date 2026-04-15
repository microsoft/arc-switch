package transform

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"gnmi-collector/internal/gnmi"
)

// CommonFields matches the existing parser output schema used by
// the syslog writer and Azure logger.
type CommonFields struct {
	DataType    string      `json:"data_type"`
	Timestamp   string      `json:"timestamp"`
	TimestampNs int64       `json:"timestamp_ns"` // Nanosecond-precision device timestamp for Kusto ordering
	Date        string      `json:"date"`
	Message     interface{} `json:"message"`
}

// NewCommonFields creates a CommonFields entry using the gNMI notification
// timestamp (nanoseconds since Unix epoch). If gnmiTimestampNs is zero
// (e.g., no notification available), falls back to the current wall clock.
func NewCommonFields(dataType string, message interface{}, gnmiTimestampNs int64) CommonFields {
	var ts time.Time
	if gnmiTimestampNs > 0 {
		ts = time.Unix(0, gnmiTimestampNs)
	} else {
		ts = time.Now()
	}
	return CommonFields{
		DataType:    dataType,
		TimestampNs: ts.UnixNano(),
		Timestamp:   ts.Format(time.RFC3339Nano),
		Date:        ts.Format("2006-01-02"),
		Message:     message,
	}
}

// Transformer converts gNMI notifications into CommonFields entries
// compatible with the existing Azure Log Analytics schema.
type Transformer interface {
	Transform(notifications []gnmi.Notification) ([]CommonFields, error)
	DataType() string
}

// DecodeBase64Float32 decodes a base64-encoded IEEE 754 float32 value.
// NX-OS encodes float values this way in JSON encoding mode.
func DecodeBase64Float32(s string) (float64, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return 0, fmt.Errorf("base64 decode: %w", err)
	}
	if len(data) != 4 {
		return 0, fmt.Errorf("expected 4 bytes for float32, got %d", len(data))
	}
	bits := binary.BigEndian.Uint32(data)
	return float64(math.Float32frombits(bits)), nil
}

// GetString safely extracts a string value from a map.
func GetString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

// GetFirstString tries multiple keys in order and returns the first non-empty value.
// Useful when NX-OS YANG field names differ from standard OpenConfig names.
func GetFirstString(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v := GetString(m, key); v != "" {
			return v
		}
	}
	return ""
}

// GetFloat safely extracts a numeric value from a map as float64.
func GetFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case int64:
			return float64(n)
		case string:
			// Try parsing as plain decimal first (e.g., "1.73" from NX-OS)
			if f, err := strconv.ParseFloat(n, 64); err == nil {
				return f
			}
			// Try parsing as base64 float
			if f, err := DecodeBase64Float32(n); err == nil {
				return f
			}
		}
	}
	return 0
}

// GetBool safely extracts a bool from a map.
func GetBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// GetInt64 safely extracts a numeric value from a map as int64.
// Handles float64, int, string (parseable), and int64 types.
// Returns 0 if key not found or not parseable.
func GetInt64(m map[string]interface{}, key string) int64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	return ToInt64(v)
}

// ToInt64 converts an interface value to int64.
func ToInt64(v interface{}) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	case string:
		i, _ := strconv.ParseInt(n, 10, 64)
		return i
	}
	return 0
}

// GetMap safely extracts a nested map from a map.
func GetMap(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if sub, ok := v.(map[string]interface{}); ok {
			return sub
		}
	}
	return nil
}

// AsMapSlice converts a gNMI update value that may be either a
// map[string]interface{} or []interface{} (array of maps) into a
// uniform []map[string]interface{}.  NX-OS returns arrays for YANG
// list nodes (e.g., PsuSlot-list, AdjEp-list, Peer-list) and maps
// for container nodes.
func AsMapSlice(v interface{}) []map[string]interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return []map[string]interface{}{val}
	case []interface{}:
		var out []map[string]interface{}
		for _, item := range val {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}

// ExtractInterfaceName extracts the interface name from a gNMI path.
// e.g., "/interfaces/interface[name=eth1/1]/state/counters" → "eth1/1"
func ExtractInterfaceName(path string) string {
	return extractKey(path, "name")
}

// ExtractNeighborAddress extracts the neighbor address from a gNMI path.
func ExtractNeighborAddress(path string) string {
	return extractKey(path, "neighbor-address")
}

func extractKey(path, key string) string {
	search := key + "="
	idx := strings.Index(path, search)
	if idx == -1 {
		return ""
	}
	start := idx + len(search)
	end := strings.Index(path[start:], "]")
	if end == -1 {
		return path[start:]
	}
	return path[start : start+end]
}

// NormalizeInterfaceName converts gNMI interface names to a canonical format.
// Handles both Cisco NX-OS names (eth1/1 → Eth1/1) and SONiC/Dell names
// (Ethernet0, PortChannel001 — returned as-is since they're already canonical).
func NormalizeInterfaceName(name string) string {
	// SONiC names are already in canonical format
	if strings.HasPrefix(name, "Ethernet") || strings.HasPrefix(name, "PortChannel") ||
		strings.HasPrefix(name, "Loopback") || strings.HasPrefix(name, "Management") {
		return name
	}
	// Cisco NX-OS names need normalization
	if strings.HasPrefix(name, "eth") {
		return "Eth" + name[3:]
	}
	if strings.HasPrefix(name, "mgmt") {
		return "mgmt" + name[4:]
	}
	if strings.HasPrefix(name, "lo") {
		return "lo" + name[2:]
	}
	if strings.HasPrefix(name, "vlan") {
		return "Vlan" + name[4:]
	}
	if strings.HasPrefix(name, "port-channel") {
		return "Po" + name[12:]
	}
	return name
}

// InterfaceType infers the interface type from its name.
func InterfaceType(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasPrefix(lower, "eth"):
		return "ethernet"
	case strings.HasPrefix(lower, "po") || strings.HasPrefix(lower, "port-channel"):
		return "port-channel"
	case strings.HasPrefix(lower, "vlan"):
		return "vlan"
	case strings.HasPrefix(lower, "mgmt"):
		return "management"
	case strings.HasPrefix(lower, "lo"):
		return "loopback"
	case strings.HasPrefix(lower, "tunnel"):
		return "tunnel"
	default:
		return "other"
	}
}
