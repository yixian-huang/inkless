package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ManifestFileName is the expected name of the plugin manifest file.
const ManifestFileName = "plugin.yaml"

// LoadManifest reads and parses a plugin.yaml file from the given directory.
// It returns the parsed PluginMeta or an error if the file cannot be read or parsed.
func LoadManifest(dir string) (*PluginMeta, error) {
	path := filepath.Join(dir, ManifestFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest %s: %w", path, err)
	}

	return ParseManifest(data)
}

// ParseManifest parses raw YAML bytes into a PluginMeta struct.
func ParseManifest(data []byte) (*PluginMeta, error) {
	var meta PluginMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse manifest YAML: %w", err)
	}
	return &meta, nil
}

// LoadAndValidateManifest reads, parses, and validates a plugin.yaml from the given directory.
func LoadAndValidateManifest(dir string) (*PluginMeta, error) {
	meta, err := LoadManifest(dir)
	if err != nil {
		return nil, err
	}

	if err := meta.Validate(); err != nil {
		return nil, err
	}

	return meta, nil
}

// ValidateManifest validates a PluginMeta struct.
// This is a convenience wrapper around PluginMeta.Validate().
func ValidateManifest(meta *PluginMeta) error {
	return meta.Validate()
}
