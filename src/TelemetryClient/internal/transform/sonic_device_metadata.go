package transform

import "gnmi-collector/internal/gnmi"

const dataTypeSonicDeviceMetadata = "device_metadata"

func init() {
	Register("sonic-device-metadata", func() Transformer { return &SonicDeviceMetadataTransformer{} })
}

type SonicDeviceMetadataTransformer struct{}

func (t *SonicDeviceMetadataTransformer) DataType() string { return dataTypeSonicDeviceMetadata }

func (t *SonicDeviceMetadataTransformer) Transform(notifications []gnmi.Notification) ([]CommonFields, error) {
	msg := map[string]interface{}{}
	var lastTS int64

	for _, n := range notifications {
		lastTS = n.Timestamp
		for _, u := range n.Updates {
			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			// Navigate to DEVICE_METADATA -> DEVICE_METADATA_LIST
			dm := GetMap(vals, "DEVICE_METADATA")
			if dm == nil {
				// Might be at root level in subscribe mode
				dm = vals
			}

			entries := AsMapSlice(dm["DEVICE_METADATA_LIST"])
			if len(entries) == 0 {
				// Try vals directly
				msg["hostname"] = GetString(vals, "hostname")
				msg["hwsku"] = GetString(vals, "hwsku")
				msg["platform"] = GetString(vals, "platform")
				msg["mac"] = GetString(vals, "mac")
				msg["type"] = GetString(vals, "type")
				continue
			}

			for _, entry := range entries {
				msg["hostname"] = GetString(entry, "hostname")
				msg["hwsku"] = GetString(entry, "hwsku")
				msg["platform"] = GetString(entry, "platform")
				msg["mac"] = GetString(entry, "mac")
				msg["type"] = GetString(entry, "type")
			}
		}
	}

	return []CommonFields{NewCommonFields(dataTypeSonicDeviceMetadata, msg, lastTS)}, nil
}
