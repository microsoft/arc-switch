// Package configmgmt provides an extensible framework for reading and
// diffing switch configuration via gNMI. Each vendor registers a Provider
// that knows which YANG config paths are supported and how to normalize
// the responses into a comparable format.
package configmgmt

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	gnmiclient "gnmi-collector/internal/gnmi"
)

// ConfigPath describes a single gNMI config path that can be read (and
// eventually Set) on a target device.
type ConfigPath struct {
	// Category groups related paths (e.g., "interfaces", "bgp", "system").
	Category string
	// Name is a human-readable identifier within the category.
	Name string
	// YANGPath is the gNMI path to query. Must target /config subtrees
	// for read-write data, or /state for read-only verification.
	YANGPath string
	// Description explains what this path controls.
	Description string
	// ReadOnly marks paths that can be fetched but not Set.
	ReadOnly bool
}

// PathResult holds the outcome of fetching a single config path.
type PathResult struct {
	ConfigPath
	// Value is the decoded gNMI response (map, slice, or scalar).
	Value interface{}
	// Error is non-nil if the fetch failed.
	Error error
	// Duration is how long the Get took.
	Duration time.Duration
}

// ConfigSnapshot is the complete configuration state of a device, grouped
// by category.
type ConfigSnapshot struct {
	Vendor   string
	Address  string
	FetchedAt time.Time
	Results  []PathResult
}

// Categories returns the distinct sorted category names in the snapshot.
func (s *ConfigSnapshot) Categories() []string {
	seen := map[string]bool{}
	for _, r := range s.Results {
		seen[r.Category] = true
	}
	cats := make([]string, 0, len(seen))
	for c := range seen {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	return cats
}

// ByCategory returns all results for the given category.
func (s *ConfigSnapshot) ByCategory(category string) []PathResult {
	var out []PathResult
	for _, r := range s.Results {
		if r.Category == category {
			out = append(out, r)
		}
	}
	return out
}

// Provider is the interface every vendor must implement. It declares which
// config paths are available and can optionally customize how responses
// are normalized.
//
// Default implementations are provided by BaseProvider; vendors only need
// to override what differs.
type Provider interface {
	// VendorName returns a human-readable vendor identifier (e.g., "cisco-nxos").
	VendorName() string

	// ConfigPaths returns the full list of config paths this vendor supports.
	ConfigPaths() []ConfigPath

	// FetchConfig queries all config paths on the target and returns a snapshot.
	FetchConfig(ctx context.Context, client *gnmiclient.Client, timeout time.Duration) *ConfigSnapshot

	// NormalizeValue post-processes a raw gNMI response value. Vendors can
	// override this to handle encoding quirks (e.g., base64 floats on NX-OS,
	// module-prefixed keys on SONiC).
	NormalizeValue(path ConfigPath, raw interface{}) interface{}

	// SupportsSet returns true if the vendor supports gNMI Set for the given path.
	// This is a static declaration based on known vendor capabilities — it does
	// NOT attempt a live Set.
	SupportsSet(path ConfigPath) bool
}

// BaseProvider implements the Provider interface with sensible defaults.
// Vendor-specific providers embed this and override only what they need.
type BaseProvider struct {
	Name  string
	Paths []ConfigPath
}

func (b *BaseProvider) VendorName() string {
	return b.Name
}

func (b *BaseProvider) ConfigPaths() []ConfigPath {
	return b.Paths
}

func (b *BaseProvider) FetchConfig(ctx context.Context, client *gnmiclient.Client, timeout time.Duration) *ConfigSnapshot {
	snap := &ConfigSnapshot{
		Vendor:    b.Name,
		FetchedAt: time.Now(),
	}

	for _, cp := range b.Paths {
		start := time.Now()
		result := PathResult{
			ConfigPath: cp,
			Duration:   0,
		}

		pathCtx, cancel := context.WithTimeout(ctx, timeout)
		notifs, err := client.Get(pathCtx, cp.YANGPath)
		cancel()
		result.Duration = time.Since(start)

		if err != nil {
			result.Error = err
		} else {
			result.Value = b.mergeNotifications(notifs)
			result.Value = b.NormalizeValue(cp, result.Value)
		}

		snap.Results = append(snap.Results, result)
	}

	return snap
}

// mergeNotifications collapses multiple gNMI notifications into a single value.
func (b *BaseProvider) mergeNotifications(notifs []gnmiclient.Notification) interface{} {
	if len(notifs) == 0 {
		return nil
	}
	// If there's one notification with one update, return its value directly.
	if len(notifs) == 1 && len(notifs[0].Updates) == 1 {
		return notifs[0].Updates[0].Value
	}
	// Multiple updates: build a map keyed by the last path element.
	merged := map[string]interface{}{}
	for _, n := range notifs {
		for _, u := range n.Updates {
			key := lastPathElement(u.Path)
			if key == "" {
				key = u.Path
			}
			merged[key] = u.Value
		}
	}
	return merged
}

func (b *BaseProvider) NormalizeValue(_ ConfigPath, raw interface{}) interface{} {
	return raw // default: no normalization
}

func (b *BaseProvider) SupportsSet(_ ConfigPath) bool {
	return false // conservative default: assume Set not supported
}

// lastPathElement extracts the last segment of a gNMI path string.
// e.g., "interfaces/interface[name=eth1/1]/config/description" → "description"
func lastPathElement(path string) string {
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")
	if len(parts) == 0 {
		return path
	}
	last := parts[len(parts)-1]
	// Strip key selectors
	if idx := strings.Index(last, "["); idx != -1 {
		last = last[:idx]
	}
	return last
}

// --- Vendor Registry ---

var providers = map[string]func() Provider{}

// RegisterProvider registers a vendor provider factory. Called from init()
// in vendor-specific source files.
func RegisterProvider(name string, factory func() Provider) {
	if _, exists := providers[name]; exists {
		panic(fmt.Sprintf("configmgmt: duplicate provider registration for %q", name))
	}
	providers[name] = factory
}

// GetProvider returns a provider instance for the given vendor name.
// Returns nil if no provider is registered.
func GetProvider(name string) Provider {
	factory, ok := providers[name]
	if !ok {
		return nil
	}
	return factory()
}

// RegisteredProviders returns the names of all registered providers.
func RegisteredProviders() []string {
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
