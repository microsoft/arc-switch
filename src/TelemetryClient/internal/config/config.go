package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Target     TargetConfig     `yaml:"target"`
	Collection CollectionConfig `yaml:"collection"`
	Azure      AzureConfig      `yaml:"azure"`
	Paths      []PathConfig     `yaml:"paths"`
}

type TargetConfig struct {
	Address     string        `yaml:"address"`
	Port        int           `yaml:"port"`
	TLS         TLSConfig     `yaml:"tls"`
	Credentials CredConfig    `yaml:"credentials"`
}

type TLSConfig struct {
	Enabled    bool   `yaml:"enabled"`
	SkipVerify bool   `yaml:"skip_verify"`
	CAFile     string `yaml:"ca_file,omitempty"`
}

type CredConfig struct {
	UsernameEnv string `yaml:"username_env"`
	PasswordEnv string `yaml:"password_env"`
}

type CollectionConfig struct {
	Mode     string        `yaml:"mode"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Encoding string        `yaml:"encoding"`
}

type AzureConfig struct {
	WorkspaceIDEnv  string `yaml:"workspace_id_env"`
	PrimaryKeyEnv   string `yaml:"primary_key_env"`
	SecondaryKeyEnv string `yaml:"secondary_key_env"`
	DeviceType      string `yaml:"device_type"`
}

type PathConfig struct {
	Name           string        `yaml:"name"`
	YANGPath       string        `yaml:"yang_path"`
	Table          string        `yaml:"table"`
	Enabled        bool          `yaml:"enabled"`
	Mode           string        `yaml:"mode,omitempty"`            // "sample" or "on_change" (subscribe); ignored in poll mode
	SampleInterval time.Duration `yaml:"sample_interval,omitempty"` // For sample mode subscriptions
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	return Parse(data)
}

func Parse(data []byte) (*Config, error) {
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.Target.Address == "" {
		return fmt.Errorf("target.address is required")
	}
	if c.Target.Port <= 0 || c.Target.Port > 65535 {
		return fmt.Errorf("target.port must be 1-65535")
	}
	if c.Collection.Interval <= 0 {
		c.Collection.Interval = 5 * time.Minute
	}
	if c.Collection.Timeout <= 0 {
		c.Collection.Timeout = 30 * time.Second
	}
	if c.Collection.Encoding == "" {
		c.Collection.Encoding = "JSON"
	}
	if c.Collection.Mode == "" {
		c.Collection.Mode = "poll"
	}
	if c.Azure.DeviceType == "" {
		return fmt.Errorf("azure.device_type is required (supported: cisco-nx-os, sonic)")
	}

	enabledCount := 0
	for i, p := range c.Paths {
		if p.Enabled {
			if p.YANGPath == "" {
				return fmt.Errorf("path %q has empty yang_path", p.Name)
			}
			if p.Table == "" {
				return fmt.Errorf("path %q has empty table", p.Name)
			}
			if p.Mode == "" {
				c.Paths[i].Mode = "sample"
			}
			if p.SampleInterval <= 0 {
				c.Paths[i].SampleInterval = c.Collection.Interval
			}
			enabledCount++
		}
	}
	if enabledCount == 0 {
		return fmt.Errorf("at least one path must be enabled")
	}
	return nil
}

// TargetAddr returns the target address in host:port format.
func (c *Config) TargetAddr() string {
	return fmt.Sprintf("%s:%d", c.Target.Address, c.Target.Port)
}

// ResolveCredentials reads username and password from the environment variables
// specified in the config. Returns ("", "") if env vars are not set.
func (c *Config) ResolveCredentials() (username, password string) {
	if c.Target.Credentials.UsernameEnv != "" {
		username = os.Getenv(c.Target.Credentials.UsernameEnv)
	}
	if c.Target.Credentials.PasswordEnv != "" {
		password = os.Getenv(c.Target.Credentials.PasswordEnv)
	}
	return
}

// ResolveAzureKeys reads workspace ID and keys from environment variables.
func (c *Config) ResolveAzureKeys() (workspaceID, primaryKey, secondaryKey string) {
	if c.Azure.WorkspaceIDEnv != "" {
		workspaceID = os.Getenv(c.Azure.WorkspaceIDEnv)
	}
	if c.Azure.PrimaryKeyEnv != "" {
		primaryKey = os.Getenv(c.Azure.PrimaryKeyEnv)
	}
	if c.Azure.SecondaryKeyEnv != "" {
		secondaryKey = os.Getenv(c.Azure.SecondaryKeyEnv)
	}
	return
}

// DataTypePrefix returns the vendor-specific prefix used for data_type fields
// in telemetry output. Derived from the configured device_type.
// Examples: "cisco-nx-os" → "cisco_nexus", "sonic" → "sonic".
func (c *Config) DataTypePrefix() string {
	return DeviceTypeToPrefix(c.Azure.DeviceType)
}

// DeviceTypeToPrefix converts a device_type string to its data_type prefix.
func DeviceTypeToPrefix(deviceType string) string {
	switch deviceType {
	case "cisco-nx-os":
		return "cisco_nexus"
	case "sonic":
		return "sonic"
	default:
		return strings.ReplaceAll(deviceType, "-", "_")
	}
}
