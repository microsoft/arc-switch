package azure

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	apiVersion = "2016-04-01"
	logTypeURL = "https://%s.ods.opinsights.azure.com/api/logs?api-version=%s"
)

// Logger sends JSON telemetry data to Azure Log Analytics via the
// HTTP Data Collector API with HMAC-SHA256 authentication.
type Logger struct {
	workspaceID  string
	primaryKey   string
	secondaryKey string
	hostname     string
	deviceType   string
	httpClient   *http.Client
	verbose      bool
}

// NewLogger creates a Logger from the provided credentials.
func NewLogger(workspaceID, primaryKey, secondaryKey, deviceType string) (*Logger, error) {
	if workspaceID == "" {
		return nil, fmt.Errorf("workspace ID is required")
	}
	if primaryKey == "" {
		return nil, fmt.Errorf("primary key is required")
	}

	hostname, _ := os.Hostname()

	return &Logger{
		workspaceID:  workspaceID,
		primaryKey:   primaryKey,
		secondaryKey: secondaryKey,
		hostname:     hostname,
		deviceType:   deviceType,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// SetVerbose enables printing the exact JSON payload sent to Azure.
func (l *Logger) SetVerbose(v bool) {
	l.verbose = v
}

// Send posts a batch of JSON entries to a Log Analytics custom table.
// Each entry should be a map with the telemetry data. The logger adds
// hostname and device_type metadata automatically.
func (l *Logger) Send(tableName string, entries []map[string]interface{}) error {
	if len(entries) == 0 {
		return nil
	}

	// Inject metadata into each entry
	for i := range entries {
		entries[i]["hostname"] = l.hostname
		entries[i]["device_type"] = l.deviceType
	}

	body, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("marshaling entries: %w", err)
	}

	// Print the exact JSON being sent to Azure for debugging
	if l.verbose {
		pretty, _ := json.MarshalIndent(entries, "", "  ")
		fmt.Printf("\n=== AZURE POST [%s] (%d entries, %d bytes) ===\n%s\n=== END ===\n\n",
			tableName, len(entries), len(body), string(pretty))
	}

	// Try primary key first
	err = l.post(tableName, body, l.primaryKey)
	if err == nil {
		return nil
	}

	// Failover to secondary key
	if l.secondaryKey != "" {
		log.Printf("WARN: primary key failed for %s, trying secondary: %v", tableName, err)
		return l.post(tableName, body, l.secondaryKey)
	}

	return err
}

func (l *Logger) post(tableName string, body []byte, sharedKey string) error {
	date := time.Now().UTC().Format(time.RFC1123)
	// RFC1123 uses "UTC", but Azure expects "GMT"
	date = strings.Replace(date, "UTC", "GMT", 1)

	sig, err := buildSignature(date, len(body), sharedKey, l.workspaceID)
	if err != nil {
		return fmt.Errorf("building signature: %w", err)
	}

	url := fmt.Sprintf(logTypeURL, l.workspaceID, apiVersion)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Log-Type", tableName)
	req.Header.Set("x-ms-date", date)
	req.Header.Set("Authorization", sig)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP POST to %s: %w", tableName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Azure returned %d for %s: %s", resp.StatusCode, tableName, string(respBody))
	}

	return nil
}

// buildSignature generates the HMAC-SHA256 authorization header value
// matching the Azure Log Analytics Data Collector API specification.
func buildSignature(date string, contentLength int, sharedKey, workspaceID string) (string, error) {
	stringToSign := fmt.Sprintf("POST\n%d\napplication/json\nx-ms-date:%s\n/api/logs", contentLength, date)

	decodedKey, err := base64.StdEncoding.DecodeString(sharedKey)
	if err != nil {
		return "", fmt.Errorf("decoding shared key: %w", err)
	}

	mac := hmac.New(sha256.New, decodedKey)
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("SharedKey %s:%s", workspaceID, signature), nil
}
