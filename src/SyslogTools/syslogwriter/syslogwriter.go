// Package syslogwriter provides a library for writing JSON entries to Linux syslog systems
package syslogwriter

import (
	"encoding/json"
	"fmt"
	"log"
	"log/syslog"
	"os/exec"
	"runtime"
)

// CommonFields represents the required fields that all JSON entries must have
type CommonFields struct {
	DataType  string      `json:"data_type"` // Can also serve as tag identifier for the entry
	Timestamp string      `json:"timestamp"`
	Date      string      `json:"date"`
	Message   interface{} `json:"message"` // Required field containing parser-specific data
}

// Config holds configuration for syslog writing
type Config struct {
	Tag          string
	MaxEntrySize int
	Verbose      bool
}

// Writer handles writing JSON entries to syslog
type Writer struct {
	config       *Config
	syslogWriter *syslog.Writer
}

// New creates a new syslog Writer instance
func New(tag string, maxEntrySize int, verbose bool) (*Writer, error) {
	// Initialize with hardcoded priority and facility (LOG_LOCAL0 | LOG_INFO)
	writer, err := syslog.New(syslog.LOG_LOCAL0|syslog.LOG_INFO, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize syslog: %w", err)
	}

	return &Writer{
		config: &Config{
			Tag:          tag,
			MaxEntrySize: maxEntrySize,
			Verbose:      verbose,
		},
		syslogWriter: writer,
	}, nil
}

// NewWithDefaults creates a new syslog Writer with default configuration
func NewWithDefaults(tag string) (*Writer, error) {
	return New(tag, 4096, false)
}

// Close closes the syslog writer
func (w *Writer) Close() error {
	if w.syslogWriter != nil {
		return w.syslogWriter.Close()
	}
	return nil
}

// WriteEntry writes a single JSON entry to syslog
func (w *Writer) WriteEntry(jsonEntry string) error {
	// Validate JSON and extract common fields
	if err := w.validateEntry(jsonEntry); err != nil {
		if w.config.Verbose {
			log.Printf("Entry validation failed: %v", err)
		}
		return fmt.Errorf("validation error: %w", err)
	}

	// Check entry size and truncate if necessary
	entry := w.handleEntrySize(jsonEntry)

	// Write to syslog
	if err := w.syslogWriter.Info(entry); err != nil {
		return fmt.Errorf("syslog write error: %w", err)
	}

	if w.config.Verbose {
		display := entry
		if len(entry) > 100 {
			display = entry[:100] + "..."
		}
		log.Printf("Logged entry: %s", display)
	}

	return nil
}

// handleEntrySize checks entry size and truncates if necessary
func (w *Writer) handleEntrySize(entry string) string {
	if len(entry) <= w.config.MaxEntrySize {
		return entry
	}

	truncated := entry[:w.config.MaxEntrySize-3] + "..."

	if w.config.Verbose {
		log.Printf("Entry truncated from %d to %d characters", len(entry), len(truncated))
	}

	return truncated
}

// validateEntry validates that the JSON entry contains required common fields
func (w *Writer) validateEntry(jsonEntry string) error {
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

	if commonFields.Message == nil {
		return fmt.Errorf("missing required field: message")
	}

	return nil
}

// DetectSystemLogger detects if systemd-journald is available
func DetectSystemLogger() bool {
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


