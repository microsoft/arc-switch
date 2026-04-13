// gnmi-config-diff explores and diffs gNMI configuration paths on network
// switches. It connects to a target device, queries all known /config paths
// for the detected vendor, and reports which paths are readable, what values
// they return, and (optionally) how they differ from a desired config.
//
// Usage:
//
//	gnmi-config-diff --config config.cisco.yaml                     # discovery mode
//	gnmi-config-diff --config config.sonic.yaml --desired desired.yaml  # diff mode
//	gnmi-config-diff --config config.cisco.yaml --json              # JSON output
//	gnmi-config-diff --config config.cisco.yaml --category interfaces   # filter
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"gnmi-collector/internal/config"
	"gnmi-collector/internal/configmgmt"
	gnmiclient "gnmi-collector/internal/gnmi"

	// Register vendor providers via init().
	_ "gnmi-collector/internal/configmgmt/vendor"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config.yaml", "Path to gnmi-collector config file (defines target, credentials, encoding)")
	desiredPath := flag.String("desired", "", "Path to desired config YAML (omit for discovery mode)")
	jsonOutput := flag.Bool("json", false, "Output report as JSON instead of human-readable text")
	category := flag.String("category", "", "Filter to a specific category (e.g., interfaces, bgp, system)")
	vendorOverride := flag.String("vendor", "", "Override vendor detection (cisco-nxos, sonic, arista-eos)")
	listVendors := flag.Bool("list-vendors", false, "List registered vendor providers and exit")
	listPaths := flag.Bool("list-paths", false, "List all config paths for the vendor and exit (no device connection)")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("gnmi-config-diff %s\n", version)
		os.Exit(0)
	}

	if *listVendors {
		fmt.Println("Registered config providers:")
		for _, name := range configmgmt.RegisteredProviders() {
			p := configmgmt.GetProvider(name)
			paths := p.ConfigPaths()
			writable := 0
			for _, cp := range paths {
				if p.SupportsSet(cp) {
					writable++
				}
			}
			fmt.Printf("  %-15s  %d paths (%d writable)\n", name, len(paths), writable)
		}
		os.Exit(0)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// --list-paths with --vendor doesn't need a config file or device connection
	if *listPaths && *vendorOverride != "" {
		providerName := mapDeviceTypeToProvider(*vendorOverride)
		provider := configmgmt.GetProvider(providerName)
		if provider == nil {
			log.Fatalf("FATAL: no config provider registered for vendor %q (available: %s)",
				providerName, strings.Join(configmgmt.RegisteredProviders(), ", "))
		}
		printPaths(provider, *category)
		os.Exit(0)
	}

	// Load collector config (reuses the same config format)
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("FATAL: load config: %v", err)
	}

	// Determine vendor
	vendorName := cfg.Azure.DeviceType
	if *vendorOverride != "" {
		vendorName = *vendorOverride
	}

	// Map device_type to provider name
	providerName := mapDeviceTypeToProvider(vendorName)
	provider := configmgmt.GetProvider(providerName)
	if provider == nil {
		log.Fatalf("FATAL: no config provider registered for vendor %q (available: %s)",
			providerName, strings.Join(configmgmt.RegisteredProviders(), ", "))
	}

	log.Printf("Using provider: %s (%d config paths)", provider.VendorName(), len(provider.ConfigPaths()))

	if *listPaths {
		printPaths(provider, *category)
		os.Exit(0)
	}

	// Connect to gNMI target
	log.Printf("Connecting to %s...", cfg.TargetAddr())
	client, err := gnmiclient.NewClient(cfg)
	if err != nil {
		log.Fatalf("FATAL: gNMI connect: %v", err)
	}
	defer client.Close()

	// Verify with Capabilities
	capCtx, capCancel := context.WithTimeout(context.Background(), cfg.Collection.Timeout)
	caps, err := client.Capabilities(capCtx)
	capCancel()
	if err != nil {
		log.Fatalf("FATAL: gNMI capabilities: %v", err)
	}
	log.Printf("Connected — gNMI %s, %d models", caps.GetGNMIVersion(), len(caps.GetSupportedModels()))

	// Fetch config snapshot
	log.Printf("Fetching config paths...")
	ctx := context.Background()
	snapshot := provider.FetchConfig(ctx, client, cfg.Collection.Timeout)
	snapshot.Address = cfg.TargetAddr()

	// Filter by category if requested
	if *category != "" {
		filtered := &configmgmt.ConfigSnapshot{
			Vendor:    snapshot.Vendor,
			Address:   snapshot.Address,
			FetchedAt: snapshot.FetchedAt,
		}
		for _, r := range snapshot.Results {
			if strings.EqualFold(r.Category, *category) {
				filtered.Results = append(filtered.Results, r)
			}
		}
		snapshot = filtered
		if len(snapshot.Results) == 0 {
			log.Fatalf("No paths matched category %q", *category)
		}
	}

	// Load desired config if provided
	var desired *configmgmt.DesiredConfig
	if *desiredPath != "" {
		desired, err = loadDesiredConfig(*desiredPath)
		if err != nil {
			log.Fatalf("FATAL: load desired config: %v", err)
		}
		log.Printf("Loaded desired config: %d paths", len(desired.Paths))
	}

	// Compute diff
	report := configmgmt.ComputeDiff(snapshot, desired)

	// Output
	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			log.Fatalf("FATAL: encode JSON: %v", err)
		}
	} else {
		fmt.Print(configmgmt.FormatReport(report))
	}

	// Log summary
	s := report.Summary
	log.Printf("Done — %d paths: %d ok, %d mismatch, %d errors, %d missing",
		s.Total, s.Match, s.Mismatch, s.FetchError, s.Missing)
}

// mapDeviceTypeToProvider maps config.yaml device_type values to provider names.
func mapDeviceTypeToProvider(deviceType string) string {
	switch strings.ToLower(deviceType) {
	case "cisco-nx-os", "cisco-nxos", "cisco":
		return "cisco-nxos"
	case "sonic", "dell-sonic", "dell":
		return "sonic"
	case "arista-eos", "arista":
		return "arista-eos"
	default:
		return deviceType
	}
}

func loadDesiredConfig(path string) (*configmgmt.DesiredConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	dc := &configmgmt.DesiredConfig{
		Paths: map[string]interface{}{},
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing desired config (JSON): %w", err)
	}
	if paths, ok := raw["paths"]; ok {
		if pm, ok := paths.(map[string]interface{}); ok {
			dc.Paths = pm
		}
	}
	return dc, nil
}

func printPaths(provider configmgmt.Provider, categoryFilter string) {
	fmt.Printf("Config paths for %s:\n\n", provider.VendorName())

	currentCategory := ""
	for _, cp := range provider.ConfigPaths() {
		if categoryFilter != "" && !strings.EqualFold(cp.Category, categoryFilter) {
			continue
		}
		if cp.Category != currentCategory {
			currentCategory = cp.Category
			fmt.Printf("── %s ──\n", strings.ToUpper(currentCategory))
		}

		setLabel := "read-only"
		if provider.SupportsSet(cp) {
			setLabel = "SET ✅"
		} else if !cp.ReadOnly {
			setLabel = "SET ❓ (untested)"
		}

		fmt.Printf("  %-30s  [%s]\n", cp.Name, setLabel)
		fmt.Printf("    %s\n", cp.YANGPath)
		fmt.Printf("    %s\n\n", cp.Description)
	}
}
