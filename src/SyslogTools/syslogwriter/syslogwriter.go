// Package syslogwriter provides a library for writing JSON entries to Linux syslog systems
package syslogwriter

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// CommonFields represents the required fields that all JSON entries must have
type CommonFields struct {
	DataType  string `json:"data_type"` // Can also serve as tag identifier for the entry
	Timestamp string `json:"timestamp"`
	Date      string `json:"date"`
}

// Config holds configuration for syslog writing
type Config struct {
	Priority     syslog.Priority
	Tag          string
	Facility     string
	UseSystemd   bool
	MaxEntrySize int
}

// Writer handles writing JSON entries to syslog
type Writer struct {
	config     *Config
	syslogWriter *syslog.Writer
	stats      *Statistics
}

// Statistics tracks processing statistics
type Statistics struct {
	TotalEntries     int
	SuccessEntries   int
	FailedEntries    int
	TruncatedEntries int
	StartTime        time.Time
}

// New creates a new syslog Writer instance
func New(config *Config) (*Writer, error) {
	// Initialize syslog writer
	writer, err := syslog.New(config.Priority, config.Tag)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize syslog: %w", err)
	}

	return &Writer{
		config:       config,
		syslogWriter: writer,
		stats: &Statistics{
			StartTime: time.Now(),
		},
	}, nil
}

// NewWithDefaults creates a new syslog Writer with default configuration
func NewWithDefaults(tag string) (*Writer, error) {
	config := &Config{
		Priority:     syslog.LOG_LOCAL0 | syslog.LOG_INFO,
		Tag:          tag,
		Facility:     "local0",
		UseSystemd:   DetectSystemLogger(),
		MaxEntrySize: 4096,
	}
	
	return New(config)
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
	return w.WriteEntryWithVerbose(jsonEntry, false)
}

// WriteEntryWithVerbose writes a single JSON entry to syslog with optional verbose output
func (w *Writer) WriteEntryWithVerbose(jsonEntry string, verbose bool) error {
	w.stats.TotalEntries++

	// Validate JSON and extract common fields
	if err := w.validateEntry(jsonEntry); err != nil {
		w.stats.FailedEntries++
		if verbose {
			log.Printf("Entry validation failed: %v", err)
		}
		return err
	}

	// Check entry size and truncate if necessary
	entry := w.handleEntrySize(jsonEntry, verbose)

	// Write to syslog
	if err := w.syslogWriter.Info(entry); err != nil {
		w.stats.FailedEntries++
		return fmt.Errorf("failed to write to syslog: %w", err)
	}

	w.stats.SuccessEntries++
	if verbose {
		fmt.Printf("Successfully logged entry: %s\n", entry[:min(100, len(entry))]+"...")
	}

	return nil
}

// WriteEntries writes multiple JSON entries to syslog
func (w *Writer) WriteEntries(jsonEntries []string) error {
	return w.WriteEntriesWithVerbose(jsonEntries, false)
}

// WriteEntriesWithVerbose writes multiple JSON entries to syslog with optional verbose output
func (w *Writer) WriteEntriesWithVerbose(jsonEntries []string, verbose bool) error {
	var lastError error
	
	for i, entry := range jsonEntries {
		if err := w.WriteEntryWithVerbose(entry, verbose); err != nil {
			lastError = err
			if verbose {
				log.Printf("Error processing entry %d: %v", i+1, err)
			}
			// Continue processing other entries even if one fails
		}
	}
	
	return lastError
}

// WriteFromReader processes JSON entries from an io.Reader
func (w *Writer) WriteFromReader(reader io.Reader) error {
	return w.WriteFromReaderWithVerbose(reader, false)
}

// WriteFromReaderWithVerbose processes JSON entries from an io.Reader with optional verbose output
func (w *Writer) WriteFromReaderWithVerbose(reader io.Reader, verbose bool) error {
	scanner := bufio.NewScanner(reader)
	lineNum := 0
	var lastError error

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		if err := w.WriteEntryWithVerbose(line, verbose); err != nil {
			lastError = err
			if verbose {
				log.Printf("Error processing line %d: %v", lineNum, err)
			}
			// Continue processing other entries even if one fails
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return lastError
}

// ValidateEntry validates that a JSON entry contains required common fields
func (w *Writer) ValidateEntry(jsonEntry string) error {
	return w.validateEntry(jsonEntry)
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

	return nil
}

// handleEntrySize checks entry size and truncates if necessary
func (w *Writer) handleEntrySize(entry string, verbose bool) string {
	if len(entry) <= w.config.MaxEntrySize {
		return entry
	}

	w.stats.TruncatedEntries++
	truncated := entry[:w.config.MaxEntrySize-3] + "..."

	if verbose {
		log.Printf("Entry truncated from %d to %d characters", len(entry), len(truncated))
	}

	return truncated
}

// GetStatistics returns current processing statistics
func (w *Writer) GetStatistics() *Statistics {
	return w.stats
}

// ResetStatistics resets the processing statistics
func (w *Writer) ResetStatistics() {
	w.stats = &Statistics{
		StartTime: time.Now(),
	}
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

// ParseSyslogPriority parses a string priority level to syslog.Priority
func ParseSyslogPriority(priority string) (syslog.Priority, error) {
	switch strings.ToLower(priority) {
	case "emerg", "emergency":
		return syslog.LOG_EMERG, nil
	case "alert":
		return syslog.LOG_ALERT, nil
	case "crit", "critical":
		return syslog.LOG_CRIT, nil
	case "err", "error":
		return syslog.LOG_ERR, nil
	case "warning", "warn":
		return syslog.LOG_WARNING, nil
	case "notice":
		return syslog.LOG_NOTICE, nil
	case "info":
		return syslog.LOG_INFO, nil
	case "debug":
		return syslog.LOG_DEBUG, nil
	default:
		return 0, fmt.Errorf("invalid priority level: %s", priority)
	}
}

// ParseSyslogFacility parses a string facility to syslog.Priority
func ParseSyslogFacility(facility string) (syslog.Priority, error) {
	switch strings.ToLower(facility) {
	case "local0":
		return syslog.LOG_LOCAL0, nil
	case "local1":
		return syslog.LOG_LOCAL1, nil
	case "local2":
		return syslog.LOG_LOCAL2, nil
	case "local3":
		return syslog.LOG_LOCAL3, nil
	case "local4":
		return syslog.LOG_LOCAL4, nil
	case "local5":
		return syslog.LOG_LOCAL5, nil
	case "local6":
		return syslog.LOG_LOCAL6, nil
	case "local7":
		return syslog.LOG_LOCAL7, nil
	case "user":
		return syslog.LOG_USER, nil
	case "mail":
		return syslog.LOG_MAIL, nil
	case "daemon":
		return syslog.LOG_DAEMON, nil
	case "auth":
		return syslog.LOG_AUTH, nil
	case "syslog":
		return syslog.LOG_SYSLOG, nil
	case "lpr":
		return syslog.LOG_LPR, nil
	case "news":
		return syslog.LOG_NEWS, nil
	case "uucp":
		return syslog.LOG_UUCP, nil
	case "cron":
		return syslog.LOG_CRON, nil
	case "authpriv":
		return syslog.LOG_AUTHPRIV, nil
	case "ftp":
		return syslog.LOG_FTP, nil
	default:
		return 0, fmt.Errorf("invalid facility: %s", facility)
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
