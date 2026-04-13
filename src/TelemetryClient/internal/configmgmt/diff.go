package configmgmt

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// DiffEntry represents a single difference between desired and actual config.
type DiffEntry struct {
	Path     string      `json:"path"`
	Category string      `json:"category"`
	Name     string      `json:"name"`
	Status   DiffStatus  `json:"status"`
	Desired  interface{} `json:"desired,omitempty"`
	Actual   interface{} `json:"actual,omitempty"`
	Details  string      `json:"details,omitempty"`
}

// DiffStatus classifies a config difference.
type DiffStatus string

const (
	DiffMatch        DiffStatus = "match"         // desired == actual
	DiffMismatch     DiffStatus = "mismatch"      // desired != actual
	DiffMissing      DiffStatus = "missing"        // path exists in desired but not on device
	DiffExtra        DiffStatus = "extra"          // path exists on device but not in desired
	DiffFetchError   DiffStatus = "fetch_error"    // could not read from device
	DiffUnsupported  DiffStatus = "unsupported"    // vendor doesn't support this path
)

// DiffReport is the complete comparison between a desired config and an actual snapshot.
type DiffReport struct {
	Vendor  string      `json:"vendor"`
	Address string      `json:"address"`
	Entries []DiffEntry `json:"entries"`
	Summary DiffSummary `json:"summary"`
}

// DiffSummary counts entries by status.
type DiffSummary struct {
	Total       int `json:"total"`
	Match       int `json:"match"`
	Mismatch    int `json:"mismatch"`
	Missing     int `json:"missing"`
	Extra       int `json:"extra"`
	FetchError  int `json:"fetch_error"`
	Unsupported int `json:"unsupported"`
}

// DesiredConfig represents a user-supplied desired configuration to compare
// against the actual device config.
type DesiredConfig struct {
	// Paths maps category/name to desired values. The key format is
	// "category.name" (e.g., "interfaces.description").
	Paths map[string]interface{} `yaml:"paths"`
}

// ComputeDiff compares a desired config against an actual device snapshot.
// If desired is nil, the report simply lists all fetched paths with their
// values (useful for discovery / exploration).
func ComputeDiff(snapshot *ConfigSnapshot, desired *DesiredConfig) *DiffReport {
	report := &DiffReport{
		Vendor:  snapshot.Vendor,
		Address: snapshot.Address,
	}

	for _, result := range snapshot.Results {
		entry := DiffEntry{
			Path:     result.YANGPath,
			Category: result.Category,
			Name:     result.Name,
			Actual:   result.Value,
		}

		if result.Error != nil {
			entry.Status = DiffFetchError
			entry.Details = result.Error.Error()
		} else if desired == nil {
			// Discovery mode: just report what we found.
			entry.Status = DiffMatch
			entry.Details = fmt.Sprintf("fetched in %s", result.Duration)
		} else {
			key := result.Category + "." + result.Name
			desiredVal, exists := desired.Paths[key]
			if !exists {
				entry.Status = DiffExtra
				entry.Details = "present on device but not in desired config"
			} else {
				entry.Desired = desiredVal
				if jsonEqual(desiredVal, result.Value) {
					entry.Status = DiffMatch
				} else {
					entry.Status = DiffMismatch
					entry.Details = "actual differs from desired"
				}
			}
		}

		report.Entries = append(report.Entries, entry)
	}

	// Check for desired paths not present in snapshot
	if desired != nil {
		snapshotKeys := map[string]bool{}
		for _, r := range snapshot.Results {
			snapshotKeys[r.Category+"."+r.Name] = true
		}
		for key, val := range desired.Paths {
			if !snapshotKeys[key] {
				parts := strings.SplitN(key, ".", 2)
				cat, name := parts[0], ""
				if len(parts) == 2 {
					name = parts[1]
				}
				report.Entries = append(report.Entries, DiffEntry{
					Path:     key,
					Category: cat,
					Name:     name,
					Status:   DiffMissing,
					Desired:  val,
					Details:  "in desired config but not fetched from device",
				})
			}
		}
	}

	// Sort entries by category then name
	sort.Slice(report.Entries, func(i, j int) bool {
		if report.Entries[i].Category != report.Entries[j].Category {
			return report.Entries[i].Category < report.Entries[j].Category
		}
		return report.Entries[i].Name < report.Entries[j].Name
	})

	// Compute summary
	for _, e := range report.Entries {
		report.Summary.Total++
		switch e.Status {
		case DiffMatch:
			report.Summary.Match++
		case DiffMismatch:
			report.Summary.Mismatch++
		case DiffMissing:
			report.Summary.Missing++
		case DiffExtra:
			report.Summary.Extra++
		case DiffFetchError:
			report.Summary.FetchError++
		case DiffUnsupported:
			report.Summary.Unsupported++
		}
	}

	return report
}

// jsonEqual compares two interface{} values by their JSON representation.
// This handles type mismatches (e.g., float64 vs int) gracefully.
func jsonEqual(a, b interface{}) bool {
	ja, err1 := json.Marshal(a)
	jb, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(ja) == string(jb)
}

// FormatReport produces a human-readable text report of the diff.
func FormatReport(report *DiffReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("╔══════════════════════════════════════════════════════════╗\n"))
	sb.WriteString(fmt.Sprintf("║  gNMI Config Diff Report                                ║\n"))
	sb.WriteString(fmt.Sprintf("║  Vendor: %-47s ║\n", report.Vendor))
	sb.WriteString(fmt.Sprintf("║  Target: %-47s ║\n", report.Address))
	sb.WriteString(fmt.Sprintf("╚══════════════════════════════════════════════════════════╝\n\n"))

	currentCategory := ""
	for _, e := range report.Entries {
		if e.Category != currentCategory {
			currentCategory = e.Category
			sb.WriteString(fmt.Sprintf("── %s ──────────────────────────────────────\n", strings.ToUpper(currentCategory)))
		}

		icon := statusIcon(e.Status)
		sb.WriteString(fmt.Sprintf("  %s %-30s  %s\n", icon, e.Name, e.Status))

		if e.Status == DiffFetchError {
			sb.WriteString(fmt.Sprintf("      error: %s\n", e.Details))
		} else if e.Status == DiffMismatch {
			sb.WriteString(fmt.Sprintf("      desired: %s\n", truncateJSON(e.Desired, 120)))
			sb.WriteString(fmt.Sprintf("      actual:  %s\n", truncateJSON(e.Actual, 120)))
		} else if e.Actual != nil && e.Status == DiffMatch {
			sb.WriteString(fmt.Sprintf("      value: %s\n", truncateJSON(e.Actual, 120)))
		}
	}

	sb.WriteString(fmt.Sprintf("\n── Summary ────────────────────────────────────────────\n"))
	sb.WriteString(fmt.Sprintf("  Total: %d  |  ✅ Match: %d  |  ❌ Mismatch: %d  |  ⚠️  Error: %d  |  ❓ Missing: %d\n",
		report.Summary.Total, report.Summary.Match, report.Summary.Mismatch,
		report.Summary.FetchError, report.Summary.Missing))

	return sb.String()
}

func statusIcon(s DiffStatus) string {
	switch s {
	case DiffMatch:
		return "✅"
	case DiffMismatch:
		return "❌"
	case DiffMissing:
		return "❓"
	case DiffExtra:
		return "➕"
	case DiffFetchError:
		return "⚠️ "
	case DiffUnsupported:
		return "🚫"
	default:
		return "  "
	}
}

func truncateJSON(v interface{}, maxLen int) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	s := string(b)
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}
