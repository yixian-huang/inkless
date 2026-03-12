package plugin

import "fmt"

// Valid state transitions:
//
//	installed -> enabled
//	installed -> disabled  (skip enable, keep installed but off)
//	enabled   -> disabled
//	disabled  -> enabled
//	disabled  -> uninstalled (removed — handled by Delete, not a state)
//	installed -> uninstalled (removed — handled by Delete, not a state)
//	failed    -> disabled   (retry after fixing)
//	failed    -> uninstalled (removed — handled by Delete, not a state)
//	*         -> failed     (any state can transition to failed on error)
var validTransitions = map[PluginState][]PluginState{
	StateInstalled: {StateEnabled, StateDisabled, StateFailed},
	StateEnabled:   {StateDisabled, StateFailed},
	StateDisabled:  {StateEnabled, StateFailed},
	StateFailed:    {StateDisabled, StateFailed},
}

// CanTransition checks if a state transition is valid.
func CanTransition(from, to PluginState) bool {
	targets, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// Transition validates a state transition and returns an error if it is invalid.
func Transition(from, to PluginState) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid plugin state transition: %s -> %s", from, to)
	}
	return nil
}

// CanUninstall checks if a plugin in the given state can be uninstalled (deleted).
// Plugins can be uninstalled from installed, disabled, or failed states, but NOT from enabled.
func CanUninstall(state PluginState) bool {
	switch state {
	case StateInstalled, StateDisabled, StateFailed:
		return true
	default:
		return false
	}
}

// CheckDependenciesSatisfied checks whether all dependencies of a plugin are satisfied
// by the currently enabled plugins. enabledPlugins maps plugin IDs to their versions.
func CheckDependenciesSatisfied(meta *PluginMeta, enabledPlugins map[string]string) error {
	for _, dep := range meta.Dependencies {
		version, ok := enabledPlugins[dep.PluginID]
		if !ok {
			return fmt.Errorf("dependency %q is not installed or enabled", dep.PluginID)
		}
		if dep.MinVersion != "" {
			cmp, err := compareSemver(version, dep.MinVersion)
			if err != nil {
				return fmt.Errorf("failed to compare versions for dependency %q: %w", dep.PluginID, err)
			}
			if cmp < 0 {
				return fmt.Errorf("dependency %q requires version >= %s but found %s", dep.PluginID, dep.MinVersion, version)
			}
		}
	}
	return nil
}

// CheckDependentsAllowDisable checks whether disabling a plugin would break any
// dependent plugins. enabledPlugins maps plugin IDs to their PluginMeta.
func CheckDependentsAllowDisable(pluginID string, enabledPlugins map[string]*PluginMeta) error {
	for id, meta := range enabledPlugins {
		for _, dep := range meta.Dependencies {
			if dep.PluginID == pluginID {
				return fmt.Errorf("cannot disable %q: plugin %q depends on it", pluginID, id)
			}
		}
	}
	return nil
}

// compareSemver compares two semver strings (major.minor.patch only, ignoring pre-release).
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareSemver(a, b string) (int, error) {
	var aMajor, aMinor, aPatch int
	var bMajor, bMinor, bPatch int

	_, err := fmt.Sscanf(stripPreRelease(a), "%d.%d.%d", &aMajor, &aMinor, &aPatch)
	if err != nil {
		return 0, fmt.Errorf("invalid semver %q: %w", a, err)
	}
	_, err = fmt.Sscanf(stripPreRelease(b), "%d.%d.%d", &bMajor, &bMinor, &bPatch)
	if err != nil {
		return 0, fmt.Errorf("invalid semver %q: %w", b, err)
	}

	if aMajor != bMajor {
		if aMajor < bMajor {
			return -1, nil
		}
		return 1, nil
	}
	if aMinor != bMinor {
		if aMinor < bMinor {
			return -1, nil
		}
		return 1, nil
	}
	if aPatch != bPatch {
		if aPatch < bPatch {
			return -1, nil
		}
		return 1, nil
	}
	return 0, nil
}

// stripPreRelease removes pre-release and build metadata from a semver string.
func stripPreRelease(v string) string {
	for i, ch := range v {
		if ch == '-' || ch == '+' {
			return v[:i]
		}
	}
	return v
}
