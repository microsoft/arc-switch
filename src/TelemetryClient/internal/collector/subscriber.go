package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
		notifications, err := gnmiclient.DecodeSubscribeResponse(resp)
		if err != nil {
			log.Printf("WARN: decode subscribe response: %v", err)
			return nil // Don't kill stream on decode errors
		}
		if len(notifications) == 0 {
			return nil // Sync response, no data
		}

		// Route notifications to the correct transformer based on path prefix
		for _, sp := range subPaths {
			matching := filterNotificationsForPath(notifications, sp.YANGPath)
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
// match the given YANG path prefix.
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

	var matched []gnmiclient.Notification
	for _, n := range notifs {
		for _, u := range n.Updates {
			if strings.Contains(u.Path, lastElem) {
				matched = append(matched, n)
				break
			}
		}
	}

	// If no specific match, return all (single-path subscription responses
	// don't always include the full path prefix in updates)
	if len(matched) == 0 {
		return notifs
	}
	return matched
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
