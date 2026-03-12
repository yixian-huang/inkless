// Package shared defines constants and configuration shared between the
// plugin host (CMS server) and plugin processes (external binaries).
package shared

import (
	goplugin "github.com/hashicorp/go-plugin"
)

// Handshake is the handshake config for Impress plugins.
// Changing this breaks all existing plugins — do so only on major versions.
var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "IMPRESS_PLUGIN",
	MagicCookieValue: "impress-cms-v1",
}

// ProviderPluginName is the plugin type name used in go-plugin's plugin map.
const ProviderPluginName = "provider"
