package config

import (
	"fmt"
	"os"
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
	Enabled bool   `yaml:"enabled"`
	CAFile  string `yaml:"ca_file,omitempty"` // Optional: pin a specific CA cert file. When empty, TOFU is used.
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
	Name              string        `yaml:"name"`
	YANGPath          string        `yaml:"yang_path"`
	Table             string        `yaml:"table"`
	Enabled           bool          `yaml:"enabled"`
	Mode              string        `yaml:"mode,omitempty"`               // "sample" or "on_change" (subscribe); ignored in poll mode
	SampleInterval    time.Duration `yaml:"sample_interval,omitempty"`    // For sample mode subscriptions
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval,omitempty"` // Override server-side liveness interval (default: 2m for on_change)
	ResolvedLabel     string        `yaml:"-"`                            // Set by discovery; used for logging instead of Name when non-empty
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

	// TLS: when enabled, TOFU is used by default (fetch server cert on
	// first connect and verify against it). Optionally, a ca_file can
	// be provided to pin a specific certificate.
	// No additional config is required — TLS "just works" with TOFU.

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

// ValidatePathNames checks that every enabled path references a known
// transformer name. Call this after loading config and building the
// transformer registry so configuration errors are caught at startup.
func (c *Config) ValidatePathNames(validNames []string) error {
	valid := make(map[string]struct{}, len(validNames))
	for _, n := range validNames {
		valid[n] = struct{}{}
	}
	for _, p := range c.Paths {
		if !p.Enabled {
			continue
		}
		if _, ok := valid[p.Name]; !ok {
			return fmt.Errorf("path %q has no registered transformer (check the name in config; valid names: %v)", p.Name, validNames)
		}
	}
	return nil
}

// TargetAddr returns the target address in host:port format.
func (c *Config) TargetAddr() string {
	return fmt.Sprintf("%s:%d", c.Target.Address, c.Target.Port)
}

// LogLabel returns a display name for log messages: ResolvedLabel if set
// by discovery, otherwise the configured Name.
func (p *PathConfig) LogLabel() string {
	if p.ResolvedLabel != "" {
		return p.ResolvedLabel
	}
	return p.Name
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

