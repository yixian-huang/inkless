package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"blotting-consultancy/internal/model"
	pb "blotting-consultancy/internal/plugin/proto"
	"blotting-consultancy/internal/provider"
)

// ManagerConfig holds configuration for the plugin manager.
type ManagerConfig struct {
	PluginDir string // directory where plugins are installed (default: ./plugins)
	DataDir   string // directory for plugin data storage (default: ./data/plugins)
}

// Manager orchestrates plugin discovery, lifecycle, and provider registration.
type Manager struct {
	config   ManagerConfig
	store    *Store
	registry *provider.Registry
	hosts    map[string]*GRPCHost // pluginID -> running host
	mu       sync.RWMutex

	// healthStop signals the health monitor to stop
	healthStop chan struct{}
}

// NewManager creates a new plugin manager.
func NewManager(cfg ManagerConfig, store *Store, registry *provider.Registry) *Manager {
	if cfg.PluginDir == "" {
		cfg.PluginDir = "./plugins"
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "./data/plugins"
	}
	return &Manager{
		config:     cfg,
		store:      store,
		registry:   registry,
		hosts:      make(map[string]*GRPCHost),
		healthStop: make(chan struct{}),
	}
}

// DiscoverPlugins scans PluginDir for directories containing valid plugin.yaml files.
func (m *Manager) DiscoverPlugins(_ context.Context) ([]PluginMeta, error) {
	entries, err := os.ReadDir(m.config.PluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no plugins directory is fine
		}
		return nil, fmt.Errorf("failed to read plugin directory %s: %w", m.config.PluginDir, err)
	}

	var discovered []PluginMeta
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(m.config.PluginDir, entry.Name())
		meta, err := LoadAndValidateManifest(dir)
		if err != nil {
			log.Printf("[PluginManager] skipping %s: %v", entry.Name(), err)
			continue
		}
		discovered = append(discovered, *meta)
	}
	return discovered, nil
}

// InstallPlugin installs a plugin from a directory path.
// It parses plugin.yaml, validates the manifest, checks for a binary, and
// creates a DB record with state=installed.
func (m *Manager) InstallPlugin(ctx context.Context, dir string) (*PluginMeta, error) {
	meta, err := LoadAndValidateManifest(dir)
	if err != nil {
		return nil, fmt.Errorf("invalid plugin manifest: %w", err)
	}

	// Check permissions are sufficient for declared capabilities
	if err := ValidateManifestPermissions(meta); err != nil {
		return nil, err
	}

	// Check if already installed
	exists, err := m.store.Exists(ctx, meta.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing plugin: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("plugin %q is already installed", meta.ID)
	}

	// Look for the plugin binary
	binaryPath := findBinary(dir, meta.ID)

	// Convert permissions to string slice for DB storage
	permStrings := make(model.JSONStringSlice, len(meta.Permissions))
	for i, p := range meta.Permissions {
		permStrings[i] = string(p)
	}

	// Create DB record
	plugin := &model.Plugin{
		PluginID:    meta.ID,
		Name:        meta.Name,
		NameZh:      meta.NameZh,
		Version:     meta.Version,
		Description: meta.Description,
		Author:      meta.Author,
		License:     meta.License,
		Homepage:    meta.Homepage,
		State:       string(StateInstalled),
		Source:      "local",
		BinaryPath:  binaryPath,
		Permissions: permStrings,
		Settings:    make(model.JSONMap),
	}

	if err := m.store.Create(ctx, plugin); err != nil {
		return nil, fmt.Errorf("failed to store plugin record: %w", err)
	}

	log.Printf("[PluginManager] installed plugin %s v%s", meta.ID, meta.Version)
	return meta, nil
}

// EnablePlugin starts a plugin process and registers its providers.
func (m *Manager) EnablePlugin(ctx context.Context, pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get plugin from DB
	p, err := m.store.GetByPluginID(ctx, pluginID)
	if err != nil {
		return err
	}

	// Validate state transition
	currentState := PluginState(p.State)
	if err := Transition(currentState, StateEnabled); err != nil {
		return err
	}

	// Check if binary exists
	if p.BinaryPath == "" {
		return fmt.Errorf("plugin %q has no binary path configured", pluginID)
	}

	// Load manifest from the plugin directory to get provider declarations
	pluginDir := filepath.Dir(p.BinaryPath)
	meta, err := LoadAndValidateManifest(pluginDir)
	if err != nil {
		// Try the plugin directory based on config
		pluginDir = filepath.Join(m.config.PluginDir, pluginID)
		meta, err = LoadAndValidateManifest(pluginDir)
		if err != nil {
			// Fall back to creating meta from DB record
			meta = &PluginMeta{
				ID:      p.PluginID,
				Name:    p.Name,
				Version: p.Version,
			}
		}
	}

	// Validate permissions for declared providers
	if err := ValidateManifestPermissions(meta); err != nil {
		_ = m.store.UpdateState(ctx, pluginID, StateFailed, err.Error())
		return err
	}

	// Convert settings
	settings := make(map[string]string)
	for k, v := range p.Settings {
		settings[k] = fmt.Sprintf("%v", v)
	}

	// Start plugin process
	host := NewGRPCHost(meta, p.BinaryPath)
	if err := host.Start(settings); err != nil {
		_ = m.store.UpdateState(ctx, pluginID, StateFailed, err.Error())
		return fmt.Errorf("failed to start plugin %s: %w", pluginID, err)
	}

	// Register providers
	for _, prov := range meta.Providers {
		switch prov.Type {
		case "storage":
			m.registry.Register(prov.Name, host.AsStorageProvider())
		case "search":
			m.registry.Register(prov.Name, host.AsSearchProvider())
		case "notifier":
			m.registry.Register(prov.Name, host.AsNotifierProvider())
		case "captcha":
			m.registry.Register(prov.Name, host.AsCaptchaProvider())
		}
	}

	m.hosts[pluginID] = host

	// Update DB state
	if err := m.store.UpdateState(ctx, pluginID, StateEnabled, ""); err != nil {
		host.Stop()
		return err
	}

	log.Printf("[PluginManager] enabled plugin %s", pluginID)
	return nil
}

// DisablePlugin stops a plugin process and unregisters its providers.
func (m *Manager) DisablePlugin(ctx context.Context, pluginID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get plugin from DB
	p, err := m.store.GetByPluginID(ctx, pluginID)
	if err != nil {
		return err
	}

	// Validate state transition
	currentState := PluginState(p.State)
	if err := Transition(currentState, StateDisabled); err != nil {
		return err
	}

	// Stop plugin process if running
	if host, ok := m.hosts[pluginID]; ok {
		if err := host.Stop(); err != nil {
			log.Printf("[PluginManager] warning: error stopping plugin %s: %v", pluginID, err)
		}
		delete(m.hosts, pluginID)
	}

	// Update DB state
	if err := m.store.UpdateState(ctx, pluginID, StateDisabled, ""); err != nil {
		return err
	}

	log.Printf("[PluginManager] disabled plugin %s", pluginID)
	return nil
}

// UninstallPlugin disables (if running) and removes a plugin from the system.
func (m *Manager) UninstallPlugin(ctx context.Context, pluginID string) error {
	// Get plugin from DB
	p, err := m.store.GetByPluginID(ctx, pluginID)
	if err != nil {
		return err
	}

	currentState := PluginState(p.State)

	// If enabled, disable first
	if currentState == StateEnabled {
		if err := m.DisablePlugin(ctx, pluginID); err != nil {
			return fmt.Errorf("failed to disable plugin before uninstall: %w", err)
		}
	}

	// Check if uninstall is allowed
	if currentState != StateEnabled && !CanUninstall(currentState) {
		return fmt.Errorf("cannot uninstall plugin in state %s", currentState)
	}

	// Delete from DB
	if err := m.store.Delete(ctx, pluginID); err != nil {
		return err
	}

	log.Printf("[PluginManager] uninstalled plugin %s", pluginID)
	return nil
}

// GetPlugin returns the current state of a plugin.
func (m *Manager) GetPlugin(ctx context.Context, pluginID string) (*model.Plugin, error) {
	return m.store.GetByPluginID(ctx, pluginID)
}

// ListPlugins returns all installed plugins.
func (m *Manager) ListPlugins(ctx context.Context) ([]model.Plugin, error) {
	return m.store.List(ctx)
}

// UpdateSettings updates a plugin's settings and re-initializes if enabled.
func (m *Manager) UpdateSettings(ctx context.Context, pluginID string, settings map[string]any) error {
	if err := m.store.UpdateSettings(ctx, pluginID, settings); err != nil {
		return err
	}

	m.mu.RLock()
	host, running := m.hosts[pluginID]
	m.mu.RUnlock()

	if running && host.IsRunning() {
		// Re-initialize with new settings
		strSettings := make(map[string]string)
		for k, v := range settings {
			strSettings[k] = fmt.Sprintf("%v", v)
		}
		resp, err := host.rpcClient.Initialize(ctx, &pb.InitRequest{
			Settings: strSettings,
			PluginID: pluginID,
		})
		if err != nil {
			return fmt.Errorf("failed to re-initialize plugin with new settings: %w", err)
		}
		if !resp.Success {
			return fmt.Errorf("plugin re-initialization failed: %s", resp.Error)
		}
	}

	return nil
}

// HandlePluginHTTP routes an HTTP request to the correct plugin.
func (m *Manager) HandlePluginHTTP(ctx context.Context, pluginID string, req *pb.HTTPRequest) (*pb.HTTPResponse, error) {
	m.mu.RLock()
	host, ok := m.hosts[pluginID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("plugin %q is not running", pluginID)
	}
	return host.HandleHTTP(ctx, req)
}

// StartEnabledPlugins starts all plugins that were previously enabled.
// Called during server startup.
func (m *Manager) StartEnabledPlugins(ctx context.Context) error {
	plugins, err := m.store.ListByState(ctx, StateEnabled)
	if err != nil {
		return fmt.Errorf("failed to list enabled plugins: %w", err)
	}

	for _, p := range plugins {
		// Reset to disabled first so EnablePlugin can transition properly
		_ = m.store.UpdateState(ctx, p.PluginID, StateDisabled, "")

		if err := m.EnablePlugin(ctx, p.PluginID); err != nil {
			log.Printf("[PluginManager] failed to start plugin %s: %v", p.PluginID, err)
			_ = m.store.UpdateState(ctx, p.PluginID, StateFailed, err.Error())
			continue
		}
	}

	return nil
}

// StopAll gracefully stops all running plugins.
// Called during server shutdown.
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop health monitor
	select {
	case <-m.healthStop:
		// already closed
	default:
		close(m.healthStop)
	}

	var errs []string
	for id, host := range m.hosts {
		if err := host.Stop(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", id, err))
		}
	}
	m.hosts = make(map[string]*GRPCHost)

	if len(errs) > 0 {
		return fmt.Errorf("errors stopping plugins: %s", join(errs, "; "))
	}
	return nil
}

// StartHealthMonitor starts a background goroutine that periodically checks
// plugin health and restarts crashed plugins.
func (m *Manager) StartHealthMonitor(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-m.healthStop:
				return
			case <-ticker.C:
				m.checkHealth()
			}
		}
	}()
}

// checkHealth inspects all running plugin hosts and restarts any that have crashed.
func (m *Manager) checkHealth() {
	m.mu.RLock()
	toRestart := make([]string, 0)
	for id, host := range m.hosts {
		if !host.IsRunning() {
			toRestart = append(toRestart, id)
		}
	}
	m.mu.RUnlock()

	for _, id := range toRestart {
		log.Printf("[PluginManager] plugin %s crashed, attempting restart", id)
		ctx := context.Background()

		// Remove old host
		m.mu.Lock()
		delete(m.hosts, id)
		m.mu.Unlock()

		// Reset to disabled so EnablePlugin can transition
		_ = m.store.UpdateState(ctx, id, StateDisabled, "")

		if err := m.EnablePlugin(ctx, id); err != nil {
			log.Printf("[PluginManager] failed to restart plugin %s: %v", id, err)
			_ = m.store.UpdateState(ctx, id, StateFailed, err.Error())
		} else {
			log.Printf("[PluginManager] plugin %s restarted successfully", id)
		}
	}
}

// findBinary locates the plugin binary in the given directory.
// It looks for a binary named after the plugin ID or a "plugin" binary.
func findBinary(dir, pluginID string) string {
	// Check for binary with plugin ID name
	candidate := filepath.Join(dir, pluginID)
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate
	}
	// Check for generic "plugin" binary
	candidate = filepath.Join(dir, "plugin")
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate
	}
	return ""
}

// join concatenates strings with a separator.
func join(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}

// settingsToStringMap converts a JSON map to a string map for gRPC.
func settingsToStringMap(settings model.JSONMap) map[string]string {
	result := make(map[string]string, len(settings))
	for k, v := range settings {
		switch val := v.(type) {
		case string:
			result[k] = val
		default:
			data, _ := json.Marshal(val)
			result[k] = string(data)
		}
	}
	return result
}
