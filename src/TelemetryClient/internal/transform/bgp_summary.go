package transform

import (
	"strings"

	"gnmi-collector/internal/gnmi"
)

const dataTypeBgpSummary = "cisco_nexus_bgp_summary"

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

			neighborAddr := ExtractNeighborAddress(u.Path)
			if neighborAddr == "" {
				neighborAddr = GetString(vals, "neighbor-address")
			}
			if neighborAddr == "" {
				continue
			}

			vrfName := extractKey(u.Path, "name")
			if vrfName == "" {
				vrfName = "default"
			}

			// Total message counts
			var msgRecvd, msgSent int64
			var rcvdUpdates, rcvdNotifications string
			var sentUpdates, sentNotifications string

			if messages := GetMap(vals, "messages"); messages != nil {
				if rcv := GetMap(messages, "received"); rcv != nil {
					rcvdUpdates = GetString(rcv, "UPDATE")
					rcvdNotifications = GetString(rcv, "NOTIFICATION")
					// Sum all received message types for total
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

			sessionState := GetString(vals, "session-state")

			// Extract prefix count from afi-safis if available
			prefixReceived := ""
			if afiSafis := GetMap(vals, "afi-safis"); afiSafis != nil {
				if afiSafiList := getSlice(afiSafis, "afi-safi"); afiSafiList != nil {
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
				"neighbor_as":      GetString(vals, "peer-as"),
				"peer_as":          GetString(vals, "peer-as"),
				"peer_type":        GetString(vals, "peer-type"),
				"description":      GetString(vals, "description"),
				"state":            strings.ToLower(sessionState),
				"session_state":    strings.ToLower(sessionState),
				"enabled":          GetBool(vals, "enabled"),

				"msg_recvd": msgRecvd,
				"msg_sent":  msgSent,
				"messages_received_updates":       rcvdUpdates,
				"messages_received_notifications": rcvdNotifications,
				"messages_sent_updates":           sentUpdates,
				"messages_sent_notifications":     sentNotifications,

				"established_transitions": GetString(vals, "established-transitions"),
				"last_established":        GetString(vals, "last-established"),
				"prefix_received":         prefixReceived,

				// Router ID and local AS from path context (may need global BGP path)
				"vrf_router_id": GetString(vals, "router-id"),
				"vrf_local_as":  GetString(vals, "local-as"),
			}

			results = append(results, NewCommonFields(dataTypeBgpSummary, msg))
		}
	}

	return results, nil
}

func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	default:
		return 0, false
	}
}
