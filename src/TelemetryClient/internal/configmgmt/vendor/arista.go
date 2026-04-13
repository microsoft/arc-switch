package vendor

import (
	"gnmi-collector/internal/configmgmt"
)

func init() {
	configmgmt.RegisterProvider("arista-eos", func() configmgmt.Provider {
		return NewAristaProvider()
	})
}

// AristaProvider extends BaseProvider for Arista EOS switches.
// Arista has the most mature gNMI Set implementation in the industry,
// supporting candidate datastores, full OpenConfig, and CLI-origin Set.
//
// This is a placeholder — paths will be populated when Arista testing begins.
type AristaProvider struct {
	configmgmt.BaseProvider
}

// NewAristaProvider creates an Arista EOS config provider.
func NewAristaProvider() *AristaProvider {
	paths := OpenConfigPaths()
	paths = append(paths, aristaNativePaths()...)

	return &AristaProvider{
		BaseProvider: configmgmt.BaseProvider{
			Name:  "arista-eos",
			Paths: paths,
		},
	}
}

// SupportsSet — Arista supports Set on nearly all OpenConfig config paths.
func (a *AristaProvider) SupportsSet(path configmgmt.ConfigPath) bool {
	// Arista has broad Set support. Only read-only state paths are excluded.
	return !path.ReadOnly
}

// aristaNativePaths returns Arista EOS-specific paths.
// Arista supports an "eos_native:" origin for full CLI-equivalent config.
func aristaNativePaths() []configmgmt.ConfigPath {
	return []configmgmt.ConfigPath{
		// Placeholder paths — to be validated on an actual Arista switch.
		{
			Category:    "system",
			Name:        "arista-system-config",
			YANGPath:    "/arista-system:system/config",
			Description: "Arista EOS system configuration (placeholder)",
		},
	}
}
