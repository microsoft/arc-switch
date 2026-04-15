package transform

import (
	"gnmi-collector/internal/gnmi"
)

const dataTypeBgpGlobal = "cisco_nexus_bgp_global"

func init() {
	Register("bgp-global", func() Transformer { return &BgpGlobalTransformer{} })
}

type BgpGlobalTransformer struct{}

func (t *BgpGlobalTransformer) DataType() string {
	return dataTypeBgpGlobal
}

func (t *BgpGlobalTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			vrfName := extractKey(u.Path, "name")

			// SONiC wraps values under a "state" sub-container when
			// querying .../global instead of .../global/state.
			stateVals := vals
			if state := GetMap(vals, "state"); state != nil {
				stateVals = state
			}

			msg := map[string]interface{}{
				"vrf_name":       vrfName,
				"local_as":       GetString(stateVals, "as"),
				"router_id":      GetString(stateVals, "router-id"),
				"total_paths":    GetString(stateVals, "total-paths"),
				"total_prefixes": GetString(stateVals, "total-prefixes"),
			}

			results = append(results, NewCommonFields(dataTypeBgpGlobal, msg, n.Timestamp))
		}
	}

	return results, nil
}
