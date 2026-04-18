package transform

import "gnmi-collector/internal/gnmi"

const dataTypeRouteSummary = "cisco_nexus_route_summary"

// NativeRouteSummaryTransformer handles native Cisco NX-OS YANG route summary data
// from /System/urib-items/table4-items/Table4-list/sum-items.
type NativeRouteSummaryTransformer struct{}

func init() {
	Register("nx-route-summary", func() Transformer { return &NativeRouteSummaryTransformer{} })
}

func (t *NativeRouteSummaryTransformer) DataType() string { return dataTypeRouteSummary }

func (t *NativeRouteSummaryTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			vrf := extractKey(u.Path, "vrfName")

			msg := map[string]interface{}{
				"vrf":         vrf,
				"route_total": GetInt64(vals, "routeTotal"),
				"path_total":  GetInt64(vals, "pathTotal"),
				"mpath_total": GetInt64(vals, "mpathTotal"),
			}

			results = append(results, NewCommonFields(dataTypeRouteSummary, msg, n.Timestamp))
		}
	}

	if len(results) == 0 {
		return []CommonFields{NewCommonFields(dataTypeRouteSummary, map[string]interface{}{}, 0)}, nil
	}

	return results, nil
}
