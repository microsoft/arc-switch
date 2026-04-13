package transform

import (
	"strconv"
	"strings"

	"gnmi-collector/internal/gnmi"
)

const dataTypeBgpSummary = "cisco_nexus_bgp_summary"

func init() {
	Register("bgp-neighbors", func() Transformer { return &BgpSummaryTransformer{} })
}

// BgpSummaryTransformer converts gNMI BGP neighbor state data (from the
// network-instance path) to the schema matching the current bgp-all-summary parser.
type BgpSummaryTransformer struct{}

func (t *BgpSummaryTransformer) DataType() string {
	return dataTypeBgpSummary
}

func (t *BgpSummaryTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			// When querying .../bgp/neighbors (keyed path), the response
			// contains a "neighbor" array with all BGP peers. When querying
			// .../neighbor/state (per-neighbor), vals IS the single neighbor.
			if neighborList := getSlice(vals, "neighbor"); neighborList != nil {
				vrfName := extractKey(u.Path, "name")
				if vrfName == "" {
					vrfName = "default"
				}
				for _, raw := range neighborList {
					nbr, ok := raw.(map[string]interface{})
					if !ok {
						continue
					}
					if entry := buildBgpNeighborEntry(nbr, u.Path, vrfName); entry != nil {
						results = append(results, *entry)
					}
				}
			} else {
				// Single-neighbor update (per-neighbor path or Subscribe mode)
				vrfName := extractKey(u.Path, "name")
				if vrfName == "" {
					vrfName = "default"
				}
				if entry := buildBgpNeighborEntry(vals, u.Path, vrfName); entry != nil {
					results = append(results, *entry)
				}
			}
		}
	}

	return results, nil
}

// buildBgpNeighborEntry extracts a single BGP neighbor entry from a
// neighbor state map. Returns nil if the neighbor address cannot be found.
func buildBgpNeighborEntry(vals map[string]interface{}, path, vrfName string) *CommonFields {
	neighborAddr := ExtractNeighborAddress(path)
	if neighborAddr == "" {
		neighborAddr = GetString(vals, "neighbor-address")
	}
	if neighborAddr == "" {
		return nil
	}

	// Check for nested "state" container (keyed path wraps state in a sub-map)
	stateVals := vals
	if state := GetMap(vals, "state"); state != nil {
		stateVals = state
	}

	// Total message counts
	var msgRecvd, msgSent int64
	var rcvdUpdates, rcvdNotifications string
	var sentUpdates, sentNotifications string

	if messages := GetMap(stateVals, "messages"); messages != nil {
		if rcv := GetMap(messages, "received"); rcv != nil {
			rcvdUpdates = GetString(rcv, "UPDATE")
			rcvdNotifications = GetString(rcv, "NOTIFICATION")
			for _, v := range rcv {
				if n, ok := toInt64(v); ok {
					msgRecvd += n
				}
			}
		}
		if snt := GetMap(messages, "sent"); snt != nil {
			sentUpdates = GetString(snt, "UPDATE")
			sentNotifications = GetString(snt, "NOTIFICATION")
			for _, v := range snt {
				if n, ok := toInt64(v); ok {
					msgSent += n
				}
			}
		}
	}

	sessionState := GetFirstString(stateVals, "session-state")
	if sessionState == "" {
		sessionState = GetFirstString(vals, "session-state")
	}

	// Extract prefix count from afi-safis if available
	prefixReceived := ""
	afiSafisMap := GetMap(vals, "afi-safis")
	if afiSafisMap == nil {
		afiSafisMap = GetMap(stateVals, "afi-safis")
	}
	if afiSafisMap != nil {
		if afiSafiList := getSlice(afiSafisMap, "afi-safi"); afiSafiList != nil {
			for _, raw := range afiSafiList {
				if as, ok := raw.(map[string]interface{}); ok {
					if asState := GetMap(as, "state"); asState != nil {
						if p := GetString(asState, "prefixes"); p != "" {
							if received := GetMap(asState, "prefixes"); received != nil {
								prefixReceived = GetString(received, "received")
							}
						}
						if prefixReceived == "" {
							prefixReceived = GetFirstString(asState, "received-pre-policy", "installed")
						}
					}
				}
			}
		}
	}

	msg := map[string]interface{}{
		"neighbor_id":      neighborAddr,
		"neighbor_address": neighborAddr,
		"vrf_name_out":     vrfName,
		"vrf_name":         vrfName,
		"neighbor_as":      GetFirstString(stateVals, "peer-as"),
		"peer_as":          GetFirstString(stateVals, "peer-as"),
		"peer_type":        GetFirstString(stateVals, "peer-type"),
		"description":      GetFirstString(stateVals, "description"),
		"state":            strings.ToLower(sessionState),
		"session_state":    strings.ToLower(sessionState),
		"enabled":          GetBool(stateVals, "enabled"),

		"msg_recvd": msgRecvd,
		"msg_sent":  msgSent,
		"messages_received_updates":       rcvdUpdates,
		"messages_received_notifications": rcvdNotifications,
		"messages_sent_updates":           sentUpdates,
		"messages_sent_notifications":     sentNotifications,

		"established_transitions": GetFirstString(stateVals, "established-transitions"),
		"last_established":        GetFirstString(stateVals, "last-established"),
		"prefix_received":         prefixReceived,

		"vrf_router_id": GetFirstString(stateVals, "router-id"),
		"vrf_local_as":  GetFirstString(stateVals, "local-as"),
	}

	entry := NewCommonFields(dataTypeBgpSummary, msg)
	return &entry
}

func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	case string:
		if n == "" {
			return 0, false
		}
		if i, err := strconv.ParseInt(n, 10, 64); err == nil {
			return i, true
		}
		return 0, false
	default:
		return 0, false
	}
}
