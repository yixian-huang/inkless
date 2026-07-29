package themecatalog

import _ "embed"

// OfficialThemesJSON is the embedded fallback catalog used when the remote
// index is unset or unreachable (Phase A design).
//
//go:embed official_themes.json
var OfficialThemesJSON []byte
