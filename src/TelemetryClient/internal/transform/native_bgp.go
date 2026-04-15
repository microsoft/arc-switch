package transform

import (
	"strings"

	"gnmi-collector/internal/gnmi"
)

// NativeBgpTransformer converts native Cisco YANG BGP peer data from
// /System/bgp-items/inst-items/dom-items/Dom-list/peer-items/Peer-list
// to the schema matching the existing bgp-all-summary parser.
//
// Note: vrf_local_as is not available at the Peer-list level in NX-OS YANG.
// It is provided separately in the CiscoBgpGlobal_CL table via the bgp-global
// OpenConfig path, which includes local_as per VRF.
type NativeBgpTransformer struct{}

func init() {
	Register("nx-bgp-peers", func() Transformer { return &NativeBgpTransformer{} })
}

func (t *NativeBgpTransformer) DataType() string { return dataTypeBgpSummary }

func (t *NativeBgpTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			entries := AsMapSlice(u.Value)
			if entries == nil {
				continue
			}

			vrfName := extractKey(u.Path, "name")
			if vrfName == "" {
				vrfName = "default"
			}

			for _, vals := range entries {
				peerAddr := extractKey(u.Path, "addr")
				if peerAddr == "" {
					peerAddr = GetString(vals, "addr")
				}
				if peerAddr == "" || strings.Contains(peerAddr, "/") {
					// Skip entries that look like route prefixes (e.g. 100.71.182.128/25)
					continue
				}

				// The PeerEntry-list is inside ent-items
				entItems := GetMap(vals, "ent-items")
				var peerEntry map[string]interface{}
				if entItems != nil {
					if entryList := GetSlice(entItems, "PeerEntry-list"); len(entryList) > 0 {
						peerEntry, _ = entryList[0].(map[string]interface{})
					}
				}

				if peerEntry == nil {
					msg := map[string]interface{}{
						"neighbor_id":      peerAddr,
						"neighbor_address": peerAddr,
						"vrf_name_out":     vrfName,
						"vrf_name":         vrfName,
					}
					results = append(results, NewCommonFields(dataTypeBgpSummary, msg, n.Timestamp))
					continue
				}

				// Extract message stats from peerstats-items
				var msgRecvd, msgSent int64
				var updateRcvd, updateSent string
				if stats := GetMap(peerEntry, "peerstats-items"); stats != nil {
					msgRecvd = GetInt64(stats, "msgRcvd")
					msgSent = GetInt64(stats, "msgSent")
					updateRcvd = GetString(stats, "updateRcvd")
					updateSent = GetString(stats, "updateSent")
				}

				// Extract prefix info from af-items
				prefixReceived := ""
				if afItems := GetMap(peerEntry, "af-items"); afItems != nil {
					if afList := GetSlice(afItems, "PeerAfEntry-list"); len(afList) > 0 {
						if af, ok := afList[0].(map[string]interface{}); ok {
							prefixReceived = GetString(af, "acceptedPaths")
						}
					}
				}

				operSt := strings.ToLower(GetString(peerEntry, "operSt"))

				msg := map[string]interface{}{
					"neighbor_id":      peerAddr,
					"neighbor_address": peerAddr,
					"vrf_name_out":     vrfName,
					"vrf_name":         vrfName,
					"neighbor_as":      GetString(peerEntry, "operAsn"),
					"peer_as":          GetString(peerEntry, "operAsn"),
					"peer_type":        GetString(peerEntry, "type"),
					"state":            operSt,
					"session_state":    operSt,
					"vrf_router_id":    GetString(peerEntry, "rtrId"),
					"local_ip":         GetString(peerEntry, "localIp"),

					"msg_recvd":                 msgRecvd,
					"msg_sent":                  msgSent,
					"messages_received_updates": updateRcvd,
					"messages_sent_updates":     updateSent,

					"established_transitions": GetString(peerEntry, "connEst"),
					"last_established":        GetString(peerEntry, "lastFlapTs"),
					"prefix_received":         prefixReceived,

					"hold_interval":       GetString(peerEntry, "holdIntvl"),
					"keepalive_interval":  GetString(peerEntry, "kaIntvl"),
					"connection_attempts": GetString(peerEntry, "connAttempts"),
					"connection_drops":    GetString(peerEntry, "connDrop"),
					"flags":              GetString(peerEntry, "flags"),
					"shutdown_qualifier": GetString(peerEntry, "shutStQual"),
				}

				results = append(results, NewCommonFields(dataTypeBgpSummary, msg, n.Timestamp))
			}
		}
	}

	return results, nil
}
