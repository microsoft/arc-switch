package collector

import (
	"fmt"
	"log"
	"strings"

	"gnmi-collector/internal/config"
	gnmiclient "gnmi-collector/internal/gnmi"
)

// discoveryPlaceholder is the template token used in YANG paths that should
// be expanded at startup via gNMI discovery.
const discoveryPlaceholder = "{network_instance}"

// discoveryBasePath is the OpenConfig path queried to discover all
// network-instance names on the device.
const discoveryBasePath = "/openconfig-network-instance:network-instances/network-instance"

// DiscoverAndExpand resolves template placeholders in config paths by
// querying the device for the available network-instance names.
//
// Paths containing {network_instance} are expanded into one concrete path
// per discovered instance. For BGP paths the expansion also validates that
// the BGP container actually exists on each instance, skipping those where
// it does not to avoid "zero neighbors" false positives.
//
// Paths without templates are passed through unchanged.
func DiscoverAndExpand(client *gnmiclient.Client, paths []config.PathConfig) ([]config.PathConfig, error) {
	// Quick check: do any enabled paths actually use templates?
	hasTemplates := false
	for _, p := range paths {
		if p.Enabled && strings.Contains(p.YANGPath, discoveryPlaceholder) {
			hasTemplates = true
			break
		}
	}
	if !hasTemplates {
		return paths, nil
	}

	// Discover network-instance names from the device.
	names, err := discoverNetworkInstances(client)
	if err != nil {
		return nil, fmt.Errorf("network-instance discovery: %w", err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("network-instance discovery returned zero instances — check device configuration")
	}
	log.Printf("INFO Discovery: found %d network-instance(s): %v", len(names), names)

	// Expand templates.
	var expanded []config.PathConfig
	for _, p := range paths {
		if !strings.Contains(p.YANGPath, discoveryPlaceholder) {
			expanded = append(expanded, p)
			continue
		}
		if !p.Enabled {
			expanded = append(expanded, p)
			continue
		}

		// Determine if this is a BGP path that needs container validation.
		isBGPPath := strings.Contains(p.YANGPath, "/bgp/")

		expandedCount := 0
		for _, ni := range names {
			concretePath := strings.ReplaceAll(p.YANGPath, discoveryPlaceholder, ni)

			// For BGP paths, verify the BGP container exists on this
			// network-instance to avoid silent empty responses.
			if isBGPPath {
				bgpBase := buildBGPProbeBase(concretePath)
				if bgpBase != "" {
					exists, probeErr := probePath(client, bgpBase)
					if probeErr != nil {
						log.Printf("WARN Discovery: could not probe BGP on network-instance %q: %v", ni, probeErr)
						continue
					}
					if !exists {
						log.Printf("INFO Discovery: skipping network-instance %q for path %q — BGP not configured", ni, p.Name)
						continue
					}
				}
			}

			clone := p
			clone.YANGPath = concretePath
			// Attach a resolved label for logging/debugging.
			clone.ResolvedLabel = fmt.Sprintf("%s[ni=%s]", p.Name, ni)
			expanded = append(expanded, clone)
			expandedCount++
		}

		if expandedCount == 0 {
			log.Printf("WARN Discovery: path %q expanded to zero instances — BGP may not be configured on any network-instance", p.Name)
		} else {
			log.Printf("INFO Discovery: expanded %q into %d concrete path(s)", p.Name, expandedCount)
		}
	}

	// Fail if any unresolved templates remain (defensive).
	for _, p := range expanded {
		if strings.Contains(p.YANGPath, "{") && strings.Contains(p.YANGPath, "}") {
			return nil, fmt.Errorf("path %q still contains unresolved template: %s", p.Name, p.YANGPath)
		}
	}

	return expanded, nil
}

// discoverNetworkInstances queries the device for all network-instance
// names using Get with SubscribeOnce fallback (same strategy as normal
// collection — SONiC returns empty for bulk Get on list paths).
func discoverNetworkInstances(client *gnmiclient.Client) ([]string, error) {
	notifs, err := client.GetWithTimeout(discoveryBasePath)
	if err != nil {
		log.Printf("INFO Discovery: Get on %s failed (%v), trying Subscribe ONCE", discoveryBasePath, err)
		notifs, err = client.SubscribeOnceWithTimeout(discoveryBasePath)
		if err != nil {
			return nil, fmt.Errorf("Get and Subscribe ONCE both failed for %s: %w", discoveryBasePath, err)
		}
	}

	// Fallback if Get returned empty values
	if len(notifs) > 0 && !gnmiclient.HasNonEmptyValues(notifs) {
		log.Printf("INFO Discovery: Get returned empty values, trying Subscribe ONCE")
		subNotifs, subErr := client.SubscribeOnceWithTimeout(discoveryBasePath)
		if subErr == nil && len(subNotifs) > 0 {
			notifs = subNotifs
		}
	}

	// Extract network-instance names from update paths.
	// The response paths look like:
	//   /network-instances/network-instance[name=default]/...
	seen := make(map[string]bool)
	var names []string
	for _, n := range notifs {
		for _, u := range n.Updates {
			name := extractKeyFromPath(u.Path, "network-instance", "name")
			if name != "" && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}

	// Also check the notification path itself for top-level maps that
	// contain a "name" field (some devices return a flat list).
	if len(names) == 0 {
		for _, n := range notifs {
			for _, u := range n.Updates {
				if m, ok := u.Value.(map[string]interface{}); ok {
					if name, ok := m["name"].(string); ok && name != "" && !seen[name] {
						seen[name] = true
						names = append(names, name)
					}
				}
				// Array of instances
				if arr, ok := u.Value.([]interface{}); ok {
					for _, item := range arr {
						if m, ok := item.(map[string]interface{}); ok {
							if name, ok := m["name"].(string); ok && name != "" && !seen[name] {
								seen[name] = true
								names = append(names, name)
							}
						}
					}
				}
			}
		}
	}

	return names, nil
}

// extractKeyFromPath extracts a YANG list key value from a gNMI path string.
// For example, extractKeyFromPath("/network-instances/network-instance[name=default]/...", "network-instance", "name")
// returns "default".
func extractKeyFromPath(path, element, key string) string {
	// Find the element in the path
	idx := strings.Index(path, element+"[")
	if idx < 0 {
		return ""
	}
	// Extract the key-value portion after element[
	rest := path[idx+len(element):]
	// Find the specific key
	keyPrefix := key + "="
	for _, part := range splitKeys(rest) {
		if strings.HasPrefix(part, keyPrefix) {
			return part[len(keyPrefix):]
		}
	}
	return ""
}

// splitKeys extracts key=value pairs from a YANG path key section like
// "[name=default]" or "[identifier=BGP][name=bgp]".
func splitKeys(s string) []string {
	var keys []string
	for {
		start := strings.IndexByte(s, '[')
		if start < 0 {
			break
		}
		end := strings.IndexByte(s[start:], ']')
		if end < 0 {
			break
		}
		keys = append(keys, s[start+1:start+end])
		s = s[start+end+1:]
	}
	return keys
}

// buildBGPProbeBase extracts the BGP container base path from a fully
// qualified BGP path. For a path like:
//
//	.../protocol[identifier=BGP][name=bgp]/bgp/neighbors
//
// it returns:
//
//	.../protocol[identifier=BGP][name=bgp]/bgp
//
// This allows probing whether BGP is configured on a specific
// network-instance without querying the full neighbors list.
func buildBGPProbeBase(concretePath string) string {
	idx := strings.Index(concretePath, "/bgp/")
	if idx < 0 {
		return ""
	}
	return concretePath[:idx+4] // include "/bgp"
}

// probePath performs a lightweight gNMI Get to check whether data exists
// at the given path. Returns true if the path returns non-empty data.
func probePath(client *gnmiclient.Client, yangPath string) (bool, error) {
	notifs, err := client.GetWithTimeout(yangPath)
	if err != nil {
		// Try Subscribe ONCE as fallback
		notifs, err = client.SubscribeOnceWithTimeout(yangPath)
		if err != nil {
			return false, err
		}
	}
	if len(notifs) == 0 {
		return false, nil
	}
	return gnmiclient.HasNonEmptyValues(notifs), nil
}
