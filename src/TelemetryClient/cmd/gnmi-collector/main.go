package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gnmi-collector/internal/azure"
	"gnmi-collector/internal/collector"
	"gnmi-collector/internal/config"
	gnmiclient "gnmi-collector/internal/gnmi"
)

var version = "dev"

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	dryRun := flag.Bool("dry-run", false, "Fetch and transform but print to stdout instead of sending to Azure")
	once := flag.Bool("once", false, "Run a single collection cycle then exit")
	dump := flag.String("dump", "", "Directory to save raw gNMI JSON responses")
	output := flag.String("output", "", "Directory to write transformed JSON files (for external Azure sender)")
	verbose := flag.Bool("verbose", false, "Print the exact JSON payload sent to Azure")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("gnmi-collector %s\n", version)
		os.Exit(0)
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("FATAL: load config: %v", err)
	}

	enabledPaths := 0
	for _, p := range cfg.Paths {
		if p.Enabled {
			enabledPaths++
		}
	}
	log.Printf("Loaded config: target=%s, %d paths enabled, interval=%s",
		cfg.TargetAddr(), enabledPaths, cfg.Collection.Interval)

	// Connect gNMI
	log.Printf("Connecting to gNMI server at %s...", cfg.TargetAddr())
	client, err := gnmiclient.NewClient(cfg)
	if err != nil {
		log.Fatalf("FATAL: gNMI connect: %v", err)
	}
	defer client.Close()

	// Verify connectivity with Capabilities
	capCtx, capCancel := context.WithTimeout(context.Background(), cfg.Collection.Timeout)
	caps, err := client.Capabilities(capCtx)
	capCancel()
	if err != nil {
		log.Fatalf("FATAL: gNMI capabilities: %v", err)
	}
	log.Printf("Connected — gNMI version %s, %d models", caps.GetGNMIVersion(), len(caps.GetSupportedModels()))

	// Setup Azure logger (unless dry-run or output mode)
	var logger *azure.Logger
	var wsID string
	if !*dryRun && *output == "" {
		var pk, sk string
		wsID, pk, sk = cfg.ResolveAzureKeys()
		if wsID == "" || pk == "" {
			log.Printf("WARN: Azure credentials not set — running in dry-run mode")
			*dryRun = true
		} else {
			logger, err = azure.NewLogger(wsID, pk, sk, cfg.Azure.DeviceType)
			if err != nil {
				log.Fatalf("FATAL: Azure logger: %v", err)
			}
			if *verbose {
				logger.SetVerbose(true)
			}
		}
	}

	// Log operating mode
	switch {
	case *dryRun:
		log.Printf("Mode: dry-run (print to stdout, no Azure send)")
	case *output != "":
		log.Printf("Mode: file output → %s (for external Azure sender)", *output)
	default:
		displayID := wsID
		if len(wsID) > 8 {
			displayID = wsID[:8]
		}
		log.Printf("Mode: direct Azure POST (workspace %s...)", displayID)
	}

	// Create collector
	c, err := collector.New(cfg, client, logger, *dryRun, *dump, *output, *verbose)
	if err != nil {
		log.Fatalf("FATAL: create collector: %v", err)
	}

	if *once {
		if err := c.RunOnce(); err != nil {
			log.Printf("Collection completed with errors: %v", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Graceful shutdown context with force-exit deadline.
	// Context cancellation alone cannot interrupt a Recv blocked on a
	// half-open TCP connection; the keepalive timeout may take tens of
	// seconds to fire. A hard deadline guarantees the process exits promptly.
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received %s, shutting down...", sig)
		cancel()

		// Force exit if graceful shutdown takes too long
		time.AfterFunc(5*time.Second, func() {
			log.Printf("Graceful shutdown timed out, forcing exit")
			os.Exit(1)
		})
	}()

	if strings.EqualFold(cfg.Collection.Mode, "subscribe") {
		// Subscribe mode: persistent streaming connection
		log.Printf("Starting subscribe stream. Press Ctrl+C to stop.")
		if err := c.RunStream(ctx); err != nil {
			log.Fatalf("FATAL: subscribe stream: %v", err)
		}
	} else {
		// Poll mode: periodic Get requests
		ticker := time.NewTicker(cfg.Collection.Interval)
		defer ticker.Stop()

		log.Printf("Starting poll loop (interval=%s). Press Ctrl+C to stop.", cfg.Collection.Interval)

		// Run first collection immediately
		if err := c.RunOnce(); err != nil {
			log.Printf("Collection completed with errors: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := c.RunOnce(); err != nil {
					log.Printf("Collection completed with errors: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}
}
