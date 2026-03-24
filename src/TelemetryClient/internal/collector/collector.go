package collector

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gnmi-collector/internal/azure"
	"gnmi-collector/internal/config"
	gnmiclient "gnmi-collector/internal/gnmi"
	"gnmi-collector/internal/transform"
)

// Collector orchestrates the gNMI data collection, transformation, and
// Azure upload cycle.
type Collector struct {
	cfg          *config.Config
	client       *gnmiclient.Client
	logger       *azure.Logger
	transformers map[string]transform.Transformer
	dryRun       bool
	dumpDir      string
	outputDir    string // Write transformed JSON files for external sender
}

// New creates a Collector with all transformers registered.
func New(cfg *config.Config, client *gnmiclient.Client, logger *azure.Logger, dryRun bool, dumpDir, outputDir string) *Collector {
	transformers := map[string]transform.Transformer{
		// OpenConfig transformers
		"interface-counters": &transform.InterfaceCountersTransformer{},
		"interface-status":   &transform.InterfaceStatusTransformer{},
		"bgp-neighbors":     &transform.BgpSummaryTransformer{},
		"lldp-neighbors":    &transform.LldpNeighborTransformer{},
		"mac-table":         &transform.MacAddressTransformer{},
		"temperature":       &transform.EnvironmentTempTransformer{},
		"power-supply":      &transform.EnvironmentPowerTransformer{},
		"platform-inventory": &transform.InventoryTransformer{},
		"transceiver":       &transform.TransceiverTransformer{},
		"system-cpus":       &transform.SystemResourcesTransformer{},
		"system-memory":     &transform.SystemResourcesTransformer{},
		"system-state":      &transform.SystemUptimeTransformer{},
		"arp-table":         &transform.ArpTransformer{},

		// Supplementary OpenConfig transformers
		"if-ethernet":         &transform.InterfaceEthernetTransformer{},
		"bgp-global":          &transform.BgpGlobalTransformer{},
		"transceiver-channel": &transform.TransceiverChannelTransformer{},

		// Native Cisco YANG transformers (Cisco-NX-OS-device model)
		"nx-transceiver":  &transform.NativeTransceiverTransformer{},
		"nx-arp":          &transform.NativeArpTransformer{},
		"nx-bgp-peers":    &transform.NativeBgpTransformer{},
		"nx-env-sensor":   &transform.NativeEnvTempTransformer{},
		"nx-env-psu":      &transform.NativeEnvPowerTransformer{},
		"nx-sys-cpu":      &transform.NativeSystemCpuTransformer{},
		"nx-sys-memory":   &transform.NativeSystemMemoryTransformer{},
		"nx-mac-table":    &transform.NativeMacTransformer{},
		"nx-lldp":         &transform.NativeLldpTransformer{},
	}

	return &Collector{
		cfg:          cfg,
		client:       client,
		logger:       logger,
		transformers: transformers,
		dryRun:       dryRun,
		dumpDir:      dumpDir,
		outputDir:    outputDir,
	}
}

// RunOnce executes a single collection cycle for all enabled paths.
// Entries targeting the same table with the same data_type are merged
// into a single row (e.g., CPU + memory → one system_resources entry).
func (c *Collector) RunOnce() error {
	successCount := 0
	failureCount := 0
	start := time.Now()

	// Collect all entries first, grouped by table
	type tableEntry struct {
		table   string
		entries []transform.CommonFields
	}
	collected := map[string]*tableEntry{}

	for _, pathCfg := range c.cfg.Paths {
		if !pathCfg.Enabled {
			continue
		}

		entries, err := c.fetchAndTransform(pathCfg)
		if err != nil {
			log.Printf("ERROR [%s]: %v", pathCfg.Name, err)
			failureCount++
			continue
		}
		successCount++

		if len(entries) == 0 {
			continue
		}

		te, ok := collected[pathCfg.Table]
		if !ok {
			te = &tableEntry{table: pathCfg.Table}
			collected[pathCfg.Table] = te
		}
		te.entries = append(te.entries, entries...)
	}

	// Merge entries with the same (table, data_type) into single rows,
	// combining their message maps. This merges CPU + memory into one row.
	for _, te := range collected {
		te.entries = mergeByDataType(te.entries)
	}

	// Now send/print all merged entries
	for _, te := range collected {
		if c.dryRun {
			for _, e := range te.entries {
				data, _ := json.MarshalIndent(e, "", "  ")
				fmt.Printf("[%s] %s\n", te.table, string(data))
			}
			continue
		}
		if c.outputDir != "" {
			if err := c.writeTransformed(te.table, te.entries); err != nil {
				log.Printf("ERROR: write %s: %v", te.table, err)
			}
			continue
		}
		if c.logger != nil {
			batch := make([]map[string]interface{}, 0, len(te.entries))
			for _, e := range te.entries {
				raw, _ := json.Marshal(e)
				var m map[string]interface{}
				json.Unmarshal(raw, &m)
				batch = append(batch, m)
			}
			if err := c.logger.Send(te.table, batch); err != nil {
				log.Printf("ERROR: send %s: %v", te.table, err)
			} else {
				log.Printf("Sent %d entries to %s", len(batch), te.table)
			}
		}
	}

	elapsed := time.Since(start)
	log.Printf("Collection complete: %d success, %d failures in %s", successCount, failureCount, elapsed)

	if failureCount > 0 {
		return fmt.Errorf("%d/%d paths failed", failureCount, successCount+failureCount)
	}
	return nil
}

// mergeByDataType merges entries with the same DataType into a single entry
// by combining their Message maps. This is used to combine CPU + memory
// into one system_resources row matching the old CLI parser output.
func mergeByDataType(entries []transform.CommonFields) []transform.CommonFields {
	// Group by data_type
	groups := map[string][]transform.CommonFields{}
	var order []string
	for _, e := range entries {
		if _, seen := groups[e.DataType]; !seen {
			order = append(order, e.DataType)
		}
		groups[e.DataType] = append(groups[e.DataType], e)
	}

	var merged []transform.CommonFields
	for _, dt := range order {
		group := groups[dt]
		if len(group) == 1 {
			merged = append(merged, group...)
			continue
		}

		// Check if all entries have map messages (mergeable)
		allMaps := true
		for _, e := range group {
			if _, ok := e.Message.(map[string]interface{}); !ok {
				allMaps = false
				break
			}
		}

		if !allMaps {
			// Can't merge non-map messages — keep as separate entries
			merged = append(merged, group...)
			continue
		}

		// Merge all maps into the first entry's message
		base := group[0]
		baseMsg := base.Message.(map[string]interface{})
		for _, e := range group[1:] {
			for k, v := range e.Message.(map[string]interface{}) {
				baseMsg[k] = v
			}
		}
		base.Message = baseMsg
		merged = append(merged, base)
	}

	return merged
}

// fetchAndTransform fetches gNMI data for a path and transforms it,
// returning the entries without sending them.
func (c *Collector) fetchAndTransform(pathCfg config.PathConfig) ([]transform.CommonFields, error) {
	// Fetch gNMI data
	notifications, err := c.client.GetWithTimeout(pathCfg.YANGPath)
	if err != nil {
		return nil, fmt.Errorf("gNMI Get: %w", err)
	}

	if len(notifications) == 0 {
		log.Printf("WARN [%s]: no notifications returned", pathCfg.Name)
		return nil, nil
	}

	// Dump raw data if requested
	if c.dumpDir != "" {
		if err := c.dumpRaw(pathCfg.Name, notifications); err != nil {
			log.Printf("WARN [%s]: dump failed: %v", pathCfg.Name, err)
		}
	}

	// Debug: log raw notification structure when dry-run is enabled
	if c.dryRun {
		for i, n := range notifications {
			for j, u := range n.Updates {
				log.Printf("DEBUG [%s] notif[%d].update[%d] path=%s value_type=%T",
					pathCfg.Name, i, j, u.Path, u.Value)
				if m, ok := u.Value.(map[string]interface{}); ok {
					keys := make([]string, 0, len(m))
					for k := range m {
						keys = append(keys, k)
					}
					log.Printf("DEBUG [%s]   map keys (%d): %v", pathCfg.Name, len(keys), keys)
				}
			}
		}
	}

	// Transform
	t, ok := c.transformers[pathCfg.Name]
	if !ok {
		return nil, fmt.Errorf("no transformer for %q", pathCfg.Name)
	}

	entries, err := t.Transform(notifications)
	if err != nil {
		return nil, fmt.Errorf("transform: %w", err)
	}

	if len(entries) == 0 {
		log.Printf("WARN [%s]: transformer produced no entries", pathCfg.Name)
		return nil, nil
	}

	return entries, nil
}

func (c *Collector) dumpRaw(name string, notifications []gnmiclient.Notification) error {
	if err := os.MkdirAll(c.dumpDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(notifications, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(c.dumpDir, name+".json")
	return os.WriteFile(path, data, 0644)
}

// writeTransformed writes transformed entries as a JSON array to a file named
// <table>.json in the output directory. These files can be sent to Azure by
// the existing azure-logger script from the default VRF.
func (c *Collector) writeTransformed(table string, entries []transform.CommonFields) error {
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return err
	}

	batch := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		raw, _ := json.Marshal(e)
		var m map[string]interface{}
		json.Unmarshal(raw, &m)
		batch = append(batch, m)
	}

	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("marshaling transformed data: %w", err)
	}

	path := filepath.Join(c.outputDir, table+".json")
	return os.WriteFile(path, data, 0644)
}
