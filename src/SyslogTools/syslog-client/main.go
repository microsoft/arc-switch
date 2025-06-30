package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/arc-switch/syslogwriter"
)

// CommonFields represents the required fields that all JSON entries must have
type CommonFields struct {
	DataType  string `json:"data_type"` // Can also serve as tag identifier for the entry
	Timestamp string `json:"timestamp"`
	Date      string `json:"date"`
}

// SyslogConfig holds configuration for syslog writing
type SyslogConfig struct {
	Priority     syslog.Priority
	Tag          string
	Facility     string
	UseSystemd   bool
	MaxEntrySize int
	Writer       *syslog.Writer
}

// SyslogWriter handles writing JSON entries to syslog
type SyslogWriter struct {
	config *SyslogConfig
	stats  *Statistics
}

// Statistics tracks processing statistics
type Statistics struct {
	TotalEntries    int
	SuccessEntries  int
	FailedEntries   int
	TruncatedEntries int
	StartTime       time.Time
}

// NewSyslogWriter creates a new SyslogWriter instance
func NewSyslogWriter(config *SyslogConfig) (*SyslogWriter, error) {
	// Initialize syslog writer
	writer, err := syslog.New(config.Priority, config.Tag)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize syslog: %w", err)
	}
	
	config.Writer = writer
	
	return &SyslogWriter{
		config: config,
		stats: &Statistics{
			StartTime: time.Now(),
		},
	}, nil
}

// Close closes the syslog writer
func (sw *SyslogWriter) Close() error {
	if sw.config.Writer != nil {
		return sw.config.Writer.Close()
	}
	return nil
}

// WriteEntry writes a single JSON entry to syslog
func (sw *SyslogWriter) WriteEntry(jsonEntry string, verbose bool) error {
	sw.stats.TotalEntries++
	
	// Validate JSON and extract common fields
	if err := sw.validateEntry(jsonEntry); err != nil {
		sw.stats.FailedEntries++
		if verbose {
			log.Printf("Entry validation failed: %v", err)
		}
		return err
	}
	
	// Check entry size and truncate if necessary
	entry := sw.handleEntrySize(jsonEntry, verbose)
	
	// Write the JSON entry as-is to syslog
	if err := sw.config.Writer.Info(entry); err != nil {
		sw.stats.FailedEntries++
		return fmt.Errorf("failed to write to syslog: %w", err)
	}
	
	sw.stats.SuccessEntries++
	if verbose {
		fmt.Printf("Successfully logged JSON entry: %s\n", entry[:min(100, len(entry))]+"...")
	}
	
	return nil
}

// validateEntry validates that the JSON entry contains required common fields
func (sw *SyslogWriter) validateEntry(jsonEntry string) error {
	var commonFields CommonFields
	if err := json.Unmarshal([]byte(jsonEntry), &commonFields); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	
	if commonFields.DataType == "" {
		return fmt.Errorf("missing required field: data_type")
	}
	
	if commonFields.Timestamp == "" {
		return fmt.Errorf("missing required field: timestamp")
	}
	
	if commonFields.Date == "" {
		return fmt.Errorf("missing required field: date")
	}
	
	return nil
}

// handleEntrySize checks entry size and truncates if necessary
func (sw *SyslogWriter) handleEntrySize(entry string, verbose bool) string {
	if len(entry) <= sw.config.MaxEntrySize {
		return entry
	}
	
	sw.stats.TruncatedEntries++
	truncated := entry[:sw.config.MaxEntrySize-3] + "..."
	
	if verbose {
		log.Printf("Entry truncated from %d to %d characters", len(entry), len(truncated))
	}
	
	return truncated
}

// GetStatistics returns current processing statistics
func (sw *SyslogWriter) GetStatistics() *Statistics {
	return sw.stats
}

// processFile processes a file containing JSON entries (one per line)
func processFile(filename string, writer *SyslogWriter, verbose bool) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()
	
	return processReader(file, writer, verbose)
}

// processReader processes JSON entries from an io.Reader
func processReader(reader io.Reader, writer *SyslogWriter, verbose bool) error {
	scanner := bufio.NewScanner(reader)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		if err := writer.WriteEntry(line, verbose); err != nil {
			if verbose {
				log.Printf("Error processing line %d: %v", lineNum, err)
			}
			// Continue processing other entries even if one fails
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}
	
	return nil
}

// processSingleEntry processes a single JSON entry from command line
func processSingleEntry(entry string, writer *SyslogWriter, verbose bool) error {
	return writer.WriteEntry(entry, verbose)
}

// detectSystemLogger detects if systemd-journald is available
func detectSystemLogger() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	
	// Check if systemctl is available
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}
	
	// Check if systemd is running
	cmd := exec.Command("systemctl", "is-active", "systemd-journald")
	err := cmd.Run()
	return err == nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// printUsage prints usage information
func printUsage() {
	fmt.Fprintf(os.Stderr, `Syslog Client

This tool sends JSON entries to the Linux syslog system. All JSON entries must contain
the required common fields: data_type, timestamp, and date.

Usage:
  %s [OPTIONS]

Options:
  -file string
        Path to file containing JSON entries (one per line)
  -entry string
        Single JSON entry to log
  -tag string
        Syslog tag to use (default "syslog-client")
  -facility string
        Syslog facility (default "local0")
  -priority string
        Syslog priority level (default "info")
  -max-size int
        Maximum entry size in bytes (default 4096)
  -verbose
        Enable verbose output
  -test
        Test mode - validate entries but don't write to syslog
  -version
        Show version information

Examples:
  # Process a file with multiple JSON entries
  %s -file /path/to/data.json -tag "network-data" -verbose

  # Process a single JSON entry
  %s -entry '{"data_type":"test","timestamp":"06/23/2025 05:05:01 PM","date":"06/23/2025","message":"hello"}'

  # Test mode to validate entries without logging
  %s -file /path/to/data.json -test -verbose

Input Format:
  All JSON entries must contain these required fields:
  - data_type: String identifying the type of data (can also serve as tag identifier)
  - timestamp: Full timestamp with date and time
  - date: Date in MM/DD/YYYY format

  Additional fields are preserved and logged as-is.

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func main() {
	// Define command line flags
	var (
		filename    = flag.String("file", "", "Path to file containing JSON entries (one per line)")
		entry       = flag.String("entry", "", "Single JSON entry to log")
		tag         = flag.String("tag", "syslog-client", "Syslog tag to use")
		facility    = flag.String("facility", "local0", "Syslog facility")
		priority    = flag.String("priority", "info", "Syslog priority level")
		maxSize     = flag.Int("max-size", 4096, "Maximum entry size in bytes")
		verbose     = flag.Bool("verbose", false, "Enable verbose output")
		testMode    = flag.Bool("test", false, "Test mode - validate entries but don't write to syslog")
		showVersion = flag.Bool("version", false, "Show version information")
		showHelp    = flag.Bool("help", false, "Show help information")
	)
	
	// Custom usage function
	flag.Usage = printUsage
	flag.Parse()
	
	if *showHelp {
		printUsage()
		os.Exit(0)
	}
	
	if *showVersion {
		fmt.Println("Syslog Client v1.0.0")
		fmt.Println("Built for Debian and RPM-based distributions")
		os.Exit(0)
	}
	
	// Validate input arguments
	if *filename == "" && *entry == "" {
		fmt.Fprintf(os.Stderr, "Error: Either -file or -entry must be specified\n\n")
		printUsage()
		os.Exit(1)
	}
	
	if *filename != "" && *entry != "" {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify both -file and -entry\n\n")
		printUsage()
		os.Exit(1)
	}
	
	// Parse syslog priority using syslogwriter package
	sysPriority, err := syslogwriter.ParseSyslogPriority(*priority)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid priority level: %s\n", *priority)
		os.Exit(1)
	}

	// Parse syslog facility using syslogwriter package
	sysFacility, err := syslogwriter.ParseSyslogFacility(*facility)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid facility: %s\n", *facility)
		os.Exit(1)
	}
	
	if *verbose {
		fmt.Printf("Starting Syslog Client\n")
		fmt.Printf("Tag: %s, Facility: %s, Priority: %s\n", *tag, *facility, *priority)
		fmt.Printf("Max entry size: %d bytes\n", *maxSize)
		fmt.Printf("Test mode: %t\n", *testMode)
		if detectSystemLogger() {
			fmt.Println("Detected systemd-journald")
		} else {
			fmt.Println("Using traditional syslog")
		}
	}
	
	// Create syslog configuration
	config := &SyslogConfig{
		Priority:     sysFacility | sysPriority,
		Tag:          *tag,
		Facility:     *facility,
		UseSystemd:   detectSystemLogger(),
		MaxEntrySize: *maxSize,
	}
	
	// Create syslog writer (skip in test mode)
	var writer *SyslogWriter
	
	if !*testMode {
		writer, err = NewSyslogWriter(config)
		if err != nil {
			log.Fatalf("Failed to create syslog writer: %v", err)
		}
		defer writer.Close()
	} else {
		// In test mode, create a mock writer for validation
		writer = &SyslogWriter{
			config: config,
			stats: &Statistics{
				StartTime: time.Now(),
			},
		}
	}
	
	// Process input
	if *filename != "" {
		if *verbose {
			fmt.Printf("Processing file: %s\n", *filename)
		}
		
		if *testMode {
			// In test mode, just validate the file
			if err := validateFile(*filename, *verbose); err != nil {
				log.Fatalf("File validation failed: %v", err)
			}
		} else {
			if err := processFile(*filename, writer, *verbose); err != nil {
				log.Fatalf("Error processing file: %v", err)
			}
		}
	} else if *entry != "" {
		if *verbose {
			fmt.Printf("Processing single entry\n")
		}
		
		if *testMode {
			// In test mode, just validate the entry
			if err := validateSingleEntry(*entry); err != nil {
				log.Fatalf("Entry validation failed: %v", err)
			}
			fmt.Println("Entry validation successful")
		} else {
			if err := processSingleEntry(*entry, writer, *verbose); err != nil {
				log.Fatalf("Error processing entry: %v", err)
			}
		}
	}
	
	// Print statistics
	if !*testMode && writer != nil {
		stats := writer.GetStatistics()
		if *verbose || stats.FailedEntries > 0 || stats.TruncatedEntries > 0 {
			fmt.Printf("\nProcessing Statistics:\n")
			fmt.Printf("  Total entries: %d\n", stats.TotalEntries)
			fmt.Printf("  Successful: %d\n", stats.SuccessEntries)
			fmt.Printf("  Failed: %d\n", stats.FailedEntries)
			fmt.Printf("  Truncated: %d\n", stats.TruncatedEntries)
			fmt.Printf("  Processing time: %v\n", time.Since(stats.StartTime))
		}
		
		if stats.FailedEntries > 0 {
			os.Exit(1)
		}
	}
	
	if *verbose {
		fmt.Println("Processing completed successfully")
	}
}

// validateFile validates all entries in a file without writing to syslog
func validateFile(filename string, verbose bool) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	lineNum := 0
	validEntries := 0
	invalidEntries := 0
	
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		if err := validateSingleEntry(line); err != nil {
			invalidEntries++
			if verbose {
				log.Printf("Line %d validation failed: %v", lineNum, err)
			}
		} else {
			validEntries++
			if verbose {
				fmt.Printf("Line %d: Valid\n", lineNum)
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}
	
	fmt.Printf("Validation Results:\n")
	fmt.Printf("  Valid entries: %d\n", validEntries)
	fmt.Printf("  Invalid entries: %d\n", invalidEntries)
	fmt.Printf("  Total lines processed: %d\n", lineNum)
	
	if invalidEntries > 0 {
		return fmt.Errorf("found %d invalid entries", invalidEntries)
	}
	
	return nil
}

// validateSingleEntry validates a single JSON entry
func validateSingleEntry(entry string) error {
	var commonFields CommonFields
	if err := json.Unmarshal([]byte(entry), &commonFields); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	
	if commonFields.DataType == "" {
		return fmt.Errorf("missing required field: data_type")
	}
	
	if commonFields.Timestamp == "" {
		return fmt.Errorf("missing required field: timestamp")
	}
	
	if commonFields.Date == "" {
		return fmt.Errorf("missing required field: date")
	}
	
	return nil
}
