package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"blotting-consultancy/internal/model"

	"gorm.io/gorm"
)

// Store provides GORM-based persistence for installed plugins.
type Store struct {
	db *gorm.DB
}

// NewStore creates a new plugin store backed by the given GORM database.
func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// AutoMigrate creates or updates the plugins and plugin_settings tables.
func (s *Store) AutoMigrate() error {
	return s.db.AutoMigrate(&model.Plugin{}, &model.PluginSetting{})
}

// Create inserts a new plugin record into the database.
func (s *Store) Create(ctx context.Context, p *model.Plugin) error {
	return s.db.WithContext(ctx).Create(p).Error
}

// GetByPluginID retrieves a plugin by its unique plugin ID (not the auto-increment ID).
func (s *Store) GetByPluginID(ctx context.Context, pluginID string) (*model.Plugin, error) {
	var p model.Plugin
	err := s.db.WithContext(ctx).Where("plugin_id = ?", pluginID).First(&p).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("plugin %q not found", pluginID)
		}
		return nil, err
	}
	return &p, nil
}

// List returns all installed plugins.
func (s *Store) List(ctx context.Context) ([]model.Plugin, error) {
	var plugins []model.Plugin
	err := s.db.WithContext(ctx).Find(&plugins).Error
	return plugins, err
}

// ListByState returns all plugins matching the given state.
func (s *Store) ListByState(ctx context.Context, state PluginState) ([]model.Plugin, error) {
	var plugins []model.Plugin
	err := s.db.WithContext(ctx).Where("state = ?", string(state)).Find(&plugins).Error
	return plugins, err
}

// UpdateState updates a plugin's state and optionally its error message.
func (s *Store) UpdateState(ctx context.Context, pluginID string, state PluginState, errMsg string) error {
	result := s.db.WithContext(ctx).
		Model(&model.Plugin{}).
		Where("plugin_id = ?", pluginID).
		Updates(map[string]interface{}{
			"state":     string(state),
			"error_msg": errMsg,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("plugin %q not found", pluginID)
	}
	return nil
}

// UpdateSettings updates a plugin's settings JSON.
func (s *Store) UpdateSettings(ctx context.Context, pluginID string, settings map[string]any) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	result := s.db.WithContext(ctx).
		Model(&model.Plugin{}).
		Where("plugin_id = ?", pluginID).
		Update("settings", string(data))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("plugin %q not found", pluginID)
	}
	return nil
}

// Delete permanently removes a plugin record from the database (hard delete).
func (s *Store) Delete(ctx context.Context, pluginID string) error {
	result := s.db.WithContext(ctx).
		Unscoped().
		Where("plugin_id = ?", pluginID).
		Delete(&model.Plugin{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("plugin %q not found", pluginID)
	}
	return nil
}

// Exists checks if a plugin with the given plugin ID exists.
func (s *Store) Exists(ctx context.Context, pluginID string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&model.Plugin{}).
		Where("plugin_id = ?", pluginID).
		Count(&count).Error
	return count > 0, err
}
