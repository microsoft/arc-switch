package transform

import (
	"gnmi-collector/internal/gnmi"
)

const dataTypeTransceiverChannel = "transceiver_dom"

func init() {
	Register("transceiver-channel", func() Transformer { return &TransceiverChannelTransformer{} })
}

// TransceiverChannelTransformer extracts per-channel DOM diagnostics from
// the OpenConfig transceiver physical-channels path.
// Path: /openconfig-platform:components/component/transceiver/physical-channels
type TransceiverChannelTransformer struct{}

func (t *TransceiverChannelTransformer) DataType() string { return dataTypeTransceiverChannel }

func (t *TransceiverChannelTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	var results []CommonFields

	for _, n := range notifications {
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			ifName := extractKey(u.Path, "name")

			channelList := GetSlice(vals, "channel")
			if channelList == nil {
				continue
			}

			for _, raw := range channelList {
				ch, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}

				state := GetMap(ch, "state")
				if state == nil {
					continue
				}

				chIndex := GetString(state, "index")
				if chIndex == "" {
					chIndex = GetString(ch, "index")
				}

				inputPower := extractInstant(state, "input-power")
				outputPower := extractInstant(state, "output-power")
				laserBias := extractInstant(state, "laser-bias-current")

				msg := map[string]interface{}{
					"interface_name": NormalizeInterfaceName(ifName),
					"channel_index":  chIndex,
					"input_power":    inputPower,
					"output_power":   outputPower,
					"laser_bias_current": laserBias,
				}

				results = append(results, NewCommonFields(dataTypeTransceiverChannel, msg, n.Timestamp))
			}
		}
	}

	return results, nil
}

// extractInstant gets the "instant" value from a nested stats object.
func extractInstant(state map[string]interface{}, key string) string {
	if sub := GetMap(state, key); sub != nil {
		return GetString(sub, "instant")
	}
	return ""
}
