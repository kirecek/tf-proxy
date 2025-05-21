package proxy

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Provider represents a Terraform provider or backend type
type Provider string

const (
	// TypeProvider represents a Terraform provider
	TypeProvider Provider = "provider"
	// TypeBackend represents a Terraform backend
	TypeBackend Provider = "backend"
)

// String returns the string representation of the Provider
func (p Provider) String() string {
	return string(p)
}

// Validate checks if the provider type is valid
func (p Provider) Validate() error {
	switch p {
	case TypeProvider, TypeBackend:
		return nil
	default:
		return fmt.Errorf("invalid provider type: %s", p)
	}
}

// Config represents the configuration for proxy settings
type Config struct {
	// DefaultProxy is used when no specific proxy is defined for a provider
	DefaultProxy string `yaml:"default_proxy"`
	// ProviderProxies maps provider names to their specific proxy settings
	ProviderProxies map[string]ProviderConfig `yaml:"providers"`
}

// ProviderConfig represents configuration for a specific provider
type ProviderConfig struct {
	// Type specifies whether this is a "provider" or "backend" configuration
	// Defaults to "provider" if not specified
	Type Provider `yaml:"type,omitempty"`
	// Proxy is optional. If not set, the default proxy will be used
	Proxy *string `yaml:"proxy,omitempty"`
}

// LoadConfig loads the configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	// If no config path is provided, look for config in default locations
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(homeDir, ".tf-proxy.yaml")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return &Config{
				ProviderProxies: map[string]ProviderConfig{
					"s3":  {Type: TypeBackend},
					"aws": {Type: TypeProvider},
				},
			}, nil
		}
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Set default providers if none specified
	if len(config.ProviderProxies) == 0 {
		config.ProviderProxies = map[string]ProviderConfig{
			"s3":  {Type: TypeBackend},
			"aws": {Type: TypeProvider},
		}
	}

	// Set default type to "provider" for any configs that don't specify a type
	// and validate all provider types
	for name, cfg := range config.ProviderProxies {
		if cfg.Type == "" {
			cfg.Type = TypeProvider
		}
		if err := cfg.Type.Validate(); err != nil {
			return nil, fmt.Errorf("invalid type for provider %s: %w", name, err)
		}
		config.ProviderProxies[name] = cfg
	}

	return &config, nil
}

// GetProxyForProvider returns the proxy address for a specific provider
func (c *Config) GetProxyForProvider(provider string) string {
	if providerConfig, exists := c.ProviderProxies[provider]; exists {
		if providerConfig.Proxy != nil {
			return *providerConfig.Proxy
		}
	}
	return c.DefaultProxy
}

// GetProviders returns the list of providers to configure
func (c *Config) GetProviders() []string {
	providers := make([]string, 0, len(c.ProviderProxies))
	for name, cfg := range c.ProviderProxies {
		providers = append(providers, cfg.Type.String()+"/"+name)
	}
	return providers
}
