package llm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// AliasConfig represents the alias configuration file
type AliasConfig struct {
	Version        string        `yaml:"version"`
	FuzzyThreshold float64       `yaml:"fuzzy_threshold"`
	Aliases        []*ModelAlias `yaml:"aliases"`
}

// DefaultAliasConfig returns default configuration
func DefaultAliasConfig() *AliasConfig {
	return &AliasConfig{
		Version:        "1.0",
		FuzzyThreshold: 0.7,
		Aliases: []*ModelAlias{
			// OpenAI
			{Alias: "gpt4", TargetModel: "gpt-4", Provider: "openai", Description: "GPT-4 (latest)", Tags: []string{"openai", "gpt", "large"}},
			{Alias: "gpt4-turbo", TargetModel: "gpt-4-turbo-preview", Provider: "openai", Description: "GPT-4 Turbo", Tags: []string{"openai", "gpt", "fast"}},
			{Alias: "gpt3", TargetModel: "gpt-3.5-turbo", Provider: "openai", Description: "GPT-3.5 Turbo", Tags: []string{"openai", "gpt", "fast", "cheap"}},

			// Anthropic
			{Alias: "claude", TargetModel: "claude-3-opus-20240229", Provider: "anthropic", Description: "Claude 3 Opus", Tags: []string{"anthropic", "claude", "large"}},
			{Alias: "claude-opus", TargetModel: "claude-3-opus-20240229", Provider: "anthropic", Description: "Claude 3 Opus", Tags: []string{"anthropic", "claude", "large"}},
			{Alias: "claude-sonnet", TargetModel: "claude-3-sonnet-20240229", Provider: "anthropic", Description: "Claude 3 Sonnet", Tags: []string{"anthropic", "claude", "balanced"}},
			{Alias: "claude-haiku", TargetModel: "claude-3-haiku-20240307", Provider: "anthropic", Description: "Claude 3 Haiku", Tags: []string{"anthropic", "claude", "fast", "cheap"}},

			// Google
			{Alias: "gemini", TargetModel: "gemini-pro", Provider: "gemini", Description: "Gemini Pro", Tags: []string{"google", "gemini"}},
			{Alias: "gemini-pro", TargetModel: "gemini-pro", Provider: "gemini", Description: "Gemini Pro", Tags: []string{"google", "gemini"}},
			{Alias: "gemini-ultra", TargetModel: "gemini-ultra", Provider: "gemini", Description: "Gemini Ultra", Tags: []string{"google", "gemini", "large"}},

			// Local models
			{Alias: "llama", TargetModel: "llama-3-8b", Provider: "ollama", Description: "Llama 3 8B", Tags: []string{"local", "llama", "open-source"}},
			{Alias: "llama-70b", TargetModel: "llama-3-70b", Provider: "ollama", Description: "Llama 3 70B", Tags: []string{"local", "llama", "large"}},
			{Alias: "mistral", TargetModel: "mistral-7b", Provider: "ollama", Description: "Mistral 7B", Tags: []string{"local", "mistral", "open-source"}},
			{Alias: "codestral", TargetModel: "codestral-22b", Provider: "ollama", Description: "Codestral 22B (coding)", Tags: []string{"local", "coding", "mistral"}},

			// Qwen
			{Alias: "qwen", TargetModel: "qwen-turbo", Provider: "qwen", Description: "Qwen Turbo", Tags: []string{"qwen", "chinese"}},
			{Alias: "qwen-plus", TargetModel: "qwen-plus", Provider: "qwen", Description: "Qwen Plus", Tags: []string{"qwen", "chinese", "large"}},

			// Generic aliases
			{Alias: "fast", TargetModel: "gpt-3.5-turbo", Provider: "openai", Description: "Fast and cheap model", Tags: []string{"fast", "cheap"}},
			{Alias: "smart", TargetModel: "gpt-4", Provider: "openai", Description: "Most capable model", Tags: []string{"smart", "capable"}},
			{Alias: "local", TargetModel: "llama-3-8b", Provider: "ollama", Description: "Default local model", Tags: []string{"local", "privacy"}},
		},
	}
}

// LoadAliasConfig loads alias configuration from file
func LoadAliasConfig(configPath string) (*AliasConfig, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return DefaultAliasConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read alias config: %w", err)
	}

	var config AliasConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse alias config: %w", err)
	}

	// Set defaults if not specified
	if config.Version == "" {
		config.Version = "1.0"
	}
	if config.FuzzyThreshold == 0 {
		config.FuzzyThreshold = 0.7
	}

	return &config, nil
}

// SaveAliasConfig saves alias configuration to file
func SaveAliasConfig(config *AliasConfig, configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal alias config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write alias config: %w", err)
	}

	return nil
}

// LoadAliasManagerFromConfig loads an AliasManager from config file
func LoadAliasManagerFromConfig(configPath string) (*AliasManager, error) {
	config, err := LoadAliasConfig(configPath)
	if err != nil {
		return nil, err
	}

	manager := NewAliasManager(config.FuzzyThreshold)

	for _, alias := range config.Aliases {
		if err := manager.AddAlias(alias); err != nil {
			// Log error but continue loading other aliases
			fmt.Printf("Warning: failed to add alias '%s': %v\n", alias.Alias, err)
		}
	}

	return manager, nil
}

// SaveAliasManagerToConfig saves an AliasManager to config file
func SaveAliasManagerToConfig(manager *AliasManager, configPath string) error {
	config := &AliasConfig{
		Version:        "1.0",
		FuzzyThreshold: manager.GetFuzzyThreshold(),
		Aliases:        manager.ExportAliases(),
	}

	return SaveAliasConfig(config, configPath)
}

// MergeAliasConfigs merges multiple alias configurations
func MergeAliasConfigs(configs ...*AliasConfig) *AliasConfig {
	if len(configs) == 0 {
		return DefaultAliasConfig()
	}

	merged := &AliasConfig{
		Version:        "1.0",
		FuzzyThreshold: 0.7,
		Aliases:        make([]*ModelAlias, 0),
	}

	// Use first config's settings as base
	if len(configs) > 0 && configs[0] != nil {
		merged.FuzzyThreshold = configs[0].FuzzyThreshold
	}

	// Merge aliases (later configs override earlier ones with same alias name)
	aliasMap := make(map[string]*ModelAlias)

	for _, config := range configs {
		if config == nil {
			continue
		}

		for _, alias := range config.Aliases {
			// Normalize key for comparison
			key := strings.ToLower(strings.TrimSpace(alias.Alias))
			aliasMap[key] = alias
		}
	}

	// Convert map back to slice
	for _, alias := range aliasMap {
		merged.Aliases = append(merged.Aliases, alias)
	}

	return merged
}

// GetConfigPaths returns standard config file paths
func GetConfigPaths() []string {
	homeDir, _ := os.UserHomeDir()

	return []string{
		".helix/model-aliases.yaml",                                    // Workspace
		filepath.Join(homeDir, ".config/helixcode/model-aliases.yaml"), // User
		"/etc/helixcode/model-aliases.yaml",                            // System (Linux)
	}
}

// LoadAliasManagerFromStandardPaths loads from standard config locations
func LoadAliasManagerFromStandardPaths() (*AliasManager, error) {
	paths := GetConfigPaths()
	configs := make([]*AliasConfig, 0)

	// Load from all existing config files
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			config, err := LoadAliasConfig(path)
			if err != nil {
				fmt.Printf("Warning: failed to load config from %s: %v\n", path, err)
				continue
			}
			configs = append(configs, config)
		}
	}

	// If no configs found, use default
	if len(configs) == 0 {
		configs = append(configs, DefaultAliasConfig())
	}

	// Merge all configs
	merged := MergeAliasConfigs(configs...)

	// Create manager from merged config
	manager := NewAliasManager(merged.FuzzyThreshold)
	for _, alias := range merged.Aliases {
		if err := manager.AddAlias(alias); err != nil {
			fmt.Printf("Warning: failed to add alias '%s': %v\n", alias.Alias, err)
		}
	}

	return manager, nil
}
