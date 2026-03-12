package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanTransition_ValidTransitions(t *testing.T) {
	validCases := []struct {
		from PluginState
		to   PluginState
	}{
		{StateInstalled, StateEnabled},
		{StateInstalled, StateDisabled},
		{StateInstalled, StateFailed},
		{StateEnabled, StateDisabled},
		{StateEnabled, StateFailed},
		{StateDisabled, StateEnabled},
		{StateDisabled, StateFailed},
		{StateFailed, StateDisabled},
		{StateFailed, StateFailed},
	}

	for _, tc := range validCases {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			assert.True(t, CanTransition(tc.from, tc.to))
		})
	}
}

func TestCanTransition_InvalidTransitions(t *testing.T) {
	invalidCases := []struct {
		from PluginState
		to   PluginState
	}{
		{StateEnabled, StateInstalled},
		{StateDisabled, StateInstalled},
		{StateFailed, StateInstalled},
		{StateFailed, StateEnabled},
		{StateEnabled, StateEnabled},
		{StateInstalled, StateInstalled},
		{StateDisabled, StateDisabled},
	}

	for _, tc := range invalidCases {
		t.Run(string(tc.from)+"->"+string(tc.to), func(t *testing.T) {
			assert.False(t, CanTransition(tc.from, tc.to))
		})
	}
}

func TestCanTransition_UnknownState(t *testing.T) {
	assert.False(t, CanTransition(PluginState("unknown"), StateEnabled))
}

func TestTransition_Valid(t *testing.T) {
	err := Transition(StateInstalled, StateEnabled)
	assert.NoError(t, err)
}

func TestTransition_Invalid(t *testing.T) {
	err := Transition(StateEnabled, StateInstalled)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid plugin state transition")
	assert.Contains(t, err.Error(), "enabled -> installed")
}

func TestCanUninstall(t *testing.T) {
	assert.True(t, CanUninstall(StateInstalled))
	assert.True(t, CanUninstall(StateDisabled))
	assert.True(t, CanUninstall(StateFailed))
	assert.False(t, CanUninstall(StateEnabled))
}

func TestCheckDependenciesSatisfied(t *testing.T) {
	t.Run("no dependencies", func(t *testing.T) {
		meta := &PluginMeta{
			ID:      "test-plugin",
			Name:    "Test",
			Version: "1.0.0",
		}
		err := CheckDependenciesSatisfied(meta, map[string]string{})
		assert.NoError(t, err)
	})

	t.Run("dependency satisfied", func(t *testing.T) {
		meta := &PluginMeta{
			ID:      "test-plugin",
			Name:    "Test",
			Version: "1.0.0",
			Dependencies: []Dependency{
				{PluginID: "base-plugin", MinVersion: "1.0.0"},
			},
		}
		enabled := map[string]string{
			"base-plugin": "1.2.0",
		}
		err := CheckDependenciesSatisfied(meta, enabled)
		assert.NoError(t, err)
	})

	t.Run("dependency exact version match", func(t *testing.T) {
		meta := &PluginMeta{
			ID:      "test-plugin",
			Name:    "Test",
			Version: "1.0.0",
			Dependencies: []Dependency{
				{PluginID: "base-plugin", MinVersion: "1.0.0"},
			},
		}
		enabled := map[string]string{
			"base-plugin": "1.0.0",
		}
		err := CheckDependenciesSatisfied(meta, enabled)
		assert.NoError(t, err)
	})

	t.Run("dependency not installed", func(t *testing.T) {
		meta := &PluginMeta{
			ID:      "test-plugin",
			Name:    "Test",
			Version: "1.0.0",
			Dependencies: []Dependency{
				{PluginID: "missing-plugin", MinVersion: "1.0.0"},
			},
		}
		err := CheckDependenciesSatisfied(meta, map[string]string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not installed or enabled")
	})

	t.Run("dependency version too low", func(t *testing.T) {
		meta := &PluginMeta{
			ID:      "test-plugin",
			Name:    "Test",
			Version: "1.0.0",
			Dependencies: []Dependency{
				{PluginID: "base-plugin", MinVersion: "2.0.0"},
			},
		}
		enabled := map[string]string{
			"base-plugin": "1.5.0",
		}
		err := CheckDependenciesSatisfied(meta, enabled)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "requires version >= 2.0.0")
	})

	t.Run("dependency without version constraint", func(t *testing.T) {
		meta := &PluginMeta{
			ID:      "test-plugin",
			Name:    "Test",
			Version: "1.0.0",
			Dependencies: []Dependency{
				{PluginID: "base-plugin"},
			},
		}
		enabled := map[string]string{
			"base-plugin": "0.1.0",
		}
		err := CheckDependenciesSatisfied(meta, enabled)
		assert.NoError(t, err)
	})
}

func TestCheckDependentsAllowDisable(t *testing.T) {
	t.Run("no dependents", func(t *testing.T) {
		enabled := map[string]*PluginMeta{
			"other-plugin": {
				ID:      "other-plugin",
				Name:    "Other",
				Version: "1.0.0",
			},
		}
		err := CheckDependentsAllowDisable("base-plugin", enabled)
		assert.NoError(t, err)
	})

	t.Run("has dependent", func(t *testing.T) {
		enabled := map[string]*PluginMeta{
			"child-plugin": {
				ID:      "child-plugin",
				Name:    "Child",
				Version: "1.0.0",
				Dependencies: []Dependency{
					{PluginID: "base-plugin", MinVersion: "1.0.0"},
				},
			},
		}
		err := CheckDependentsAllowDisable("base-plugin", enabled)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot disable")
		assert.Contains(t, err.Error(), "child-plugin")
	})

	t.Run("empty enabled map", func(t *testing.T) {
		err := CheckDependentsAllowDisable("any-plugin", map[string]*PluginMeta{})
		assert.NoError(t, err)
	})
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.9.9", "2.0.0", -1},
		{"1.1.0", "1.0.9", 1},
		{"0.1.0", "0.0.99", 1},
		{"1.0.0-beta.1", "1.0.0", 0}, // pre-release stripped, compared equal
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got, err := compareSemver(tt.a, tt.b)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCompareSemver_Invalid(t *testing.T) {
	_, err := compareSemver("bad", "1.0.0")
	assert.Error(t, err)

	_, err = compareSemver("1.0.0", "bad")
	assert.Error(t, err)
}
