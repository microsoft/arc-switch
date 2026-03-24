package transform

import (
	"gnmi-collector/internal/gnmi"
)

const dataTypeBgpGlobal = "cisco_nexus_bgp_global"

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

			msg := map[string]interface{}{
				"vrf_name":       vrfName,
				"local_as":       GetString(vals, "as"),
				"router_id":      GetString(vals, "router-id"),
				"total_paths":    GetString(vals, "total-paths"),
				"total_prefixes": GetString(vals, "total-prefixes"),
			}

			results = append(results, NewCommonFields(dataTypeBgpGlobal, msg))
		}
	}

	return results, nil
}
