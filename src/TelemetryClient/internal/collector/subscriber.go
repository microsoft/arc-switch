package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	gnmiclient "gnmi-collector/internal/gnmi"
	"gnmi-collector/internal/transform"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

const (
	defaultFlushInterval = 30 * time.Second
	defaultBatchSize     = 200
	maxReconnectDelay    = 2 * time.Minute
	initialReconnectDelay = 2 * time.Second
)

// tableBatch accumulates entries for a single Azure table.
type tableBatch struct {
	mu      sync.Mutex
	table   string
	entries []transform.CommonFields
}

func (b *tableBatch) add(entries []transform.CommonFields) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = append(b.entries, entries...)
}

func (b *tableBatch) drain() []transform.CommonFields {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.entries
	b.entries = nil
	return out
}

func (b *tableBatch) size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.entries)
}

// RunStream starts the subscribe-mode streaming collector. It opens a
// persistent gNMI Subscribe stream, routes updates to the correct
// transformer, batches results, and flushes to Azure periodically.
// It reconnects automatically on stream failure with exponential backoff.
// Blocks until ctx is cancelled.
func (c *Collector) RunStream(ctx context.Context) error {
	// Build subscription paths and path→config lookup
	var subPaths []gnmiclient.SubscriptionPath
	pathLookup := map[string]pathMapping{}

	for _, p := range c.cfg.Paths {
		if !p.Enabled {
			continue
		}
		subPaths = append(subPaths, gnmiclient.SubscriptionPath{
			YANGPath:       p.YANGPath,
			Mode:           p.Mode,
			SampleInterval: p.SampleInterval,
			Name:           p.Name,
			Table:          p.Table,
		})

		t, ok := c.transformers[p.Name]
		if !ok {
			return fmt.Errorf("no transformer for %q", p.Name)
		}
		pathLookup[p.YANGPath] = pathMapping{
			name:        p.Name,
			table:       p.Table,
			transformer: t,
		}
	}

	if len(subPaths) == 0 {
		return fmt.Errorf("no paths enabled for subscription")
	}

	log.Printf("Subscribe mode: %d paths, flush every %s or %d entries",
		len(subPaths), defaultFlushInterval, defaultBatchSize)

	// Batches keyed by table name
	batches := map[string]*tableBatch{}
	for _, sp := range subPaths {
		if _, ok := batches[sp.Table]; !ok {
			batches[sp.Table] = &tableBatch{table: sp.Table}
		}
	}

	// Periodic flush goroutine
	flushDone := make(chan struct{})
	go func() {
		defer close(flushDone)
		ticker := time.NewTicker(defaultFlushInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.flushAll(batches)
			case <-ctx.Done():
				// Final flush on shutdown
				c.flushAll(batches)
				return
			}
		}
	}()

	// Reconnect loop
	delay := initialReconnectDelay
	for {
		err := c.subscribeOnce(ctx, subPaths, pathLookup, batches)
		if ctx.Err() != nil {
			// Context cancelled — graceful shutdown
			<-flushDone
			log.Printf("Subscribe stream stopped (context cancelled)")
			return nil
		}

		log.Printf("Subscribe stream error: %v — reconnecting in %s", err, delay)
		select {
		case <-time.After(delay):
			delay = delay * 2
			if delay > maxReconnectDelay {
				delay = maxReconnectDelay
			}
		case <-ctx.Done():
			<-flushDone
			return nil
		}
	}
}

type pathMapping struct {
	name        string
	table       string
	transformer transform.Transformer
}

// subscribeOnce runs a single subscribe session. Returns when the stream
// errors or the context is cancelled.
func (c *Collector) subscribeOnce(
	ctx context.Context,
	subPaths []gnmiclient.SubscriptionPath,
	pathLookup map[string]pathMapping,
	batches map[string]*tableBatch,
) error {
	updateCount := 0

	return c.client.Subscribe(ctx, subPaths, func(resp *gpb.SubscribeResponse) error {
		// Decode WITH prefix preservation so entity keys (e.g.,
		// [name=Ethernet0]) are included in the full update paths.
		notifications, err := gnmiclient.DecodeSubscribeResponseWithPrefix(resp)
		if err != nil {
			log.Printf("WARN: decode subscribe response: %v", err)
			return nil // Don't kill stream on decode errors
		}
		if len(notifications) == 0 {
			return nil // Sync response, no data
		}

		// Normalize leaf-level scalar updates into nested tree maps.
		// Subscribe responses send individual leaf values, but
		// transformers expect nested maps (same as Get responses).
		notifications = gnmiclient.NormalizeSubscribeNotifications(notifications)

		// Route notifications to the correct transformer based on path prefix.
		// Use drillDown to navigate nested subscribe-stream data to the
		// subscribed path level that transformers expect.
		for _, sp := range subPaths {
			matching := drillDownToSubscribedPath(notifications, sp.YANGPath)
			if len(matching) == 0 {
				continue
			}

			pm, ok := pathLookup[sp.YANGPath]
			if !ok {
				continue
			}

			entries, err := pm.transformer.Transform(matching)
			if err != nil {
				log.Printf("WARN [%s]: transform: %v", sp.Name, err)
				continue
			}
			if len(entries) == 0 {
				continue
			}

			// Apply vendor-specific data_type prefix (same as poll mode).
			applyDataTypePrefix(entries, c.cfg.DataTypePrefix())

			batch := batches[sp.Table]
			batch.add(entries)
			updateCount++

			// Flush if batch is large enough
			if batch.size() >= defaultBatchSize {
				c.flushBatch(batch)
			}
		}

		return nil
	})
}

// filterNotificationsForPath returns notifications whose update paths
// match the given YANG path prefix. After normalization, update paths
// contain full entity paths (with prefix), so substring matching on the
// last path element is reliable.
func filterNotificationsForPath(notifs []gnmiclient.Notification, yangPath string) []gnmiclient.Notification {
	// Normalize the path for comparison
	yangPath = strings.TrimPrefix(yangPath, "/")
	parts := strings.Split(yangPath, "/")
	if len(parts) == 0 {
		return nil
	}
	// Use the last significant element as a substring match
	// e.g., "counters" from ".../interface/state/counters"
	lastElem := parts[len(parts)-1]
	if idx := strings.Index(lastElem, ":"); idx != -1 {
		lastElem = lastElem[idx+1:]
	}
	// Strip key selectors for matching (e.g., "component[name=X]" → "component")
	if idx := strings.Index(lastElem, "["); idx != -1 {
		lastElem = lastElem[:idx]
	}

	var matched []gnmiclient.Notification
	for _, n := range notifs {
		for _, u := range n.Updates {
			if strings.Contains(u.Path, lastElem) {
				matched = append(matched, n)
				break
			}
		}
	}

	// If no specific match and only subscribed to a single path, the
	// response is likely for that path (some servers omit path prefix).
	if len(matched) == 0 {
		return notifs
	}
	return matched
}

// drillDownToSubscribedPath takes subscribe-stream notifications (which
// may arrive at a root-level path with deeply nested values) and drills
// into the nested map to reach the subscribed YANG path level.
//
// NX-OS subscribe STREAM returns entire subtrees, e.g.:
//
//	path=/System, value={procsys-items:{syscpusummary-items:{idle:74.0, ...}}}
//
// But transformers expect data at the subscribed sub-path level, e.g.:
//
//	path=/System/procsys-items/syscpusummary-items, value={idle:74.0, ...}
//
// For list paths (arrays), this creates separate notifications per element:
//
//	path=/interfaces, value={interface:[{name:eth1/1, state:{counters:{...}}}]}
//	→ path=/interfaces/interface[name=eth1/1]/state/counters, value={in-octets:...}
func drillDownToSubscribedPath(notifs []gnmiclient.Notification, yangPath string) []gnmiclient.Notification {
	cleanYang := stripPathModulePrefixes(yangPath)

	var result []gnmiclient.Notification
	for _, n := range notifs {
		for _, u := range n.Updates {
			cleanUpdatePath := stripPathModulePrefixes(u.Path)

			// Strip key selectors for comparison so that
			// /interfaces/interface[name=Ethernet0]/state/counters
			// matches the YANG path /interfaces/interface/state/counters.
			// Keep the original cleanUpdatePath (with keys) for the
			// notification output so transformers can extract entity names.
			cleanUpdatePathNoKeys := stripKeySelectors(cleanUpdatePath)

			// Already at subscribed path — pass through as-is
			if cleanUpdatePathNoKeys == cleanYang {
				result = append(result, gnmiclient.Notification{
					Timestamp: n.Timestamp,
					Updates:   []gnmiclient.Update{u},
				})
				continue
			}

			// Below subscribed path — wrap value in remaining path
			// segments so transformers see the same structure as poll mode.
			// E.g., subscribe sends path=/system/memory/state, value={physical:X}
			// but transformer expects path=/system/memory, value={state:{physical:X}}
			if strings.HasPrefix(cleanUpdatePathNoKeys, cleanYang+"/") {
				remaining := strings.TrimPrefix(cleanUpdatePathNoKeys, cleanYang+"/")
				wrapped := wrapValueInPath(u.Value, remaining)
				result = append(result, gnmiclient.Notification{
					Timestamp: n.Timestamp,
					Updates: []gnmiclient.Update{{
						Path:  yangPath,
						Value: wrapped,
					}},
				})
				continue
			}

			// Check if update path is a prefix of yang path — need to drill down
			if !strings.HasPrefix(cleanYang, cleanUpdatePathNoKeys) {
				continue
			}

			vals, ok := u.Value.(map[string]interface{})
			if !ok {
				continue
			}

			remaining := strings.TrimPrefix(cleanYang, cleanUpdatePathNoKeys)
			remaining = strings.TrimPrefix(remaining, "/")
			segments := strings.Split(remaining, "/")

			drilled := drillIntoMap(vals, segments, cleanUpdatePath)
			for _, d := range drilled {
				result = append(result, gnmiclient.Notification{
					Timestamp: n.Timestamp,
					Updates:   []gnmiclient.Update{d},
				})
			}
		}
	}
	return result
}

// drillIntoMap navigates into a nested map following path segments.
// When a segment resolves to a list ([]interface{}), it iterates each
// element and continues drilling, creating separate Updates per entity.
func drillIntoMap(vals map[string]interface{}, segments []string, currentPath string) []gnmiclient.Update {
	if len(segments) == 0 {
		return []gnmiclient.Update{{Path: currentPath, Value: vals}}
	}

	seg := segments[0]
	rest := segments[1:]

	v, ok := vals[seg]
	if !ok {
		return nil
	}

	newPath := currentPath + "/" + seg

	switch typedV := v.(type) {
	case map[string]interface{}:
		return drillIntoMap(typedV, rest, newPath)
	case []interface{}:
		// List — iterate each element and add entity key to path
		var results []gnmiclient.Update
		for _, item := range typedV {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			name := getEntityName(itemMap)
			elemPath := newPath
			if name != "" {
				elemPath = newPath + "[name=" + name + "]"
			}
			results = append(results, drillIntoMap(itemMap, rest, elemPath)...)
		}
		return results
	default:
		// Leaf value — can't drill further
		if len(rest) == 0 {
			return []gnmiclient.Update{{Path: newPath, Value: v}}
		}
		return nil
	}
}

// getEntityName tries common key fields to extract an entity identity.
func getEntityName(m map[string]interface{}) string {
	for _, key := range []string{"name", "id", "index"} {
		if v, ok := m[key]; ok {
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// stripPathModulePrefixes removes YANG module prefixes from path segments.
// e.g., "/openconfig-interfaces:interfaces/interface" → "/interfaces/interface"
func stripPathModulePrefixes(path string) string {
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if idx := strings.Index(p, ":"); idx != -1 {
			// Preserve key selectors: "interface[name=X]" stays unchanged
			if bracketIdx := strings.Index(p, "["); bracketIdx != -1 && bracketIdx < idx {
				continue
			}
			parts[i] = p[idx+1:]
		}
	}
	return strings.Join(parts, "/")
}

// stripKeySelectors removes YANG list key selectors from path segments.
// e.g., "/interfaces/interface[name=Ethernet0]/state/counters"
//
//	→ "/interfaces/interface/state/counters"
func stripKeySelectors(path string) string {
	// Use a simple regex to strip [...] from all path segments
	re := regexp.MustCompile(`\[[^\]]*\]`)
	return re.ReplaceAllString(path, "")
}

// wrapValueInPath wraps a value in nested maps following a relative path.
// E.g., wrapValueInPath({"physical": 100}, "state") → {"state": {"physical": 100}}
// E.g., wrapValueInPath(val, "a/b") → {"a": {"b": val}}
func wrapValueInPath(value interface{}, relativePath string) interface{} {
	segments := strings.Split(relativePath, "/")
	// Build inside-out: start from the deepest segment
	var current interface{} = value
	for i := len(segments) - 1; i >= 0; i-- {
		current = map[string]interface{}{segments[i]: current}
	}
	return current
}

func (c *Collector) flushAll(batches map[string]*tableBatch) {
	for _, batch := range batches {
		c.flushBatch(batch)
	}
}

func (c *Collector) flushBatch(batch *tableBatch) {
	entries := batch.drain()
	if len(entries) == 0 {
		return
	}

	// Merge complementary entries (e.g. CPU + memory → single system row),
	// same as poll mode does in RunOnce.
	entries = mergeByDataType(entries)

	if c.dryRun {
		for _, e := range entries {
			data, _ := json.MarshalIndent(e, "", "  ")
			fmt.Printf("[%s] %s\n", batch.table, string(data))
		}
		return
	}

	if c.logger == nil {
		log.Printf("WARN: cannot flush %d entries for %s — no Azure logger", len(entries), batch.table)
		return
	}

	maps := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		raw, _ := json.Marshal(e)
		var m map[string]interface{}
		json.Unmarshal(raw, &m)
		maps = append(maps, m)
	}

	if err := c.logger.Send(batch.table, maps); err != nil {
		log.Printf("ERROR: flush %d entries to %s: %v", len(entries), batch.table, err)
	} else {
		log.Printf("Flushed %d entries to %s", len(entries), batch.table)
	}
}
