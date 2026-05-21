package global_config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"blotting-consultancy/internal/model"
)

func jsonStdMarshal(v any) ([]byte, error)      { return json.Marshal(v) }
func jsonStdUnmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

// Indirect through package-level vars so tests can inject if needed.
var (
	jsonMarshal   = jsonStdMarshal
	jsonUnmarshal = jsonStdUnmarshal
)

type LocaleMode string

const (
	LocaleModeMonoZh    LocaleMode = "mono-zh"
	LocaleModeMonoEn    LocaleMode = "mono-en"
	LocaleModeBilingual LocaleMode = "bilingual"
)

type LocalizedString struct {
	Zh string `json:"zh,omitempty"`
	En string `json:"en,omitempty"`
}

type Identity struct {
	Name          LocalizedString `json:"name"`
	Tagline       *LocalizedString `json:"tagline,omitempty"`
	LocaleMode    LocaleMode      `json:"localeMode"`
	DefaultLocale string          `json:"defaultLocale"`
}

type LogoRef struct {
	Light string `json:"light"`
	Dark  string `json:"dark,omitempty"`
}

type Brand struct {
	Logo         LogoRef `json:"logo"`
	Favicon      string  `json:"favicon"`
	OgImage      string  `json:"ogImage"`
	PrimaryColor string  `json:"primaryColor"`
	AccentColor  string  `json:"accentColor,omitempty"`
}

type Social struct {
	Kind  string `json:"kind"`
	URL   string `json:"url"`
	Label string `json:"label,omitempty"`
}

type Author struct {
	Name     string          `json:"name"`
	Avatar   string          `json:"avatar,omitempty"`
	Bio      *LocalizedString `json:"bio,omitempty"`
	Location string          `json:"location,omitempty"`
	Socials  []Social        `json:"socials"`
}

type ExtraLink struct {
	Label LocalizedString `json:"label"`
	URL   string          `json:"url"`
}

type Footer struct {
	Copyright  *LocalizedString `json:"copyright,omitempty"`
	ICP        string          `json:"icp,omitempty"`
	ExtraLinks []ExtraLink     `json:"extraLinks,omitempty"`
}

type SEO struct {
	DefaultTitle       *LocalizedString `json:"defaultTitle,omitempty"`
	TitleTemplate      string          `json:"titleTemplate,omitempty"`
	DefaultDescription *LocalizedString `json:"defaultDescription,omitempty"`
	TwitterHandle      string          `json:"twitterHandle,omitempty"`
}

type SiteConfigGlobal struct {
	Identity Identity `json:"identity"`
	Brand    Brand    `json:"brand"`
	Author   Author   `json:"author"`
	Footer   Footer   `json:"footer"`
	SEO      SEO      `json:"seo"`
}

// validateGlobalConfig converts a JSONMap to SiteConfigGlobal and validates.
// Returns nil error on success; concrete error describing the offending field on failure.
func validateGlobalConfig(raw model.JSONMap) (*SiteConfigGlobal, error) {
	bytes, err := jsonMarshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal raw config: %w", err)
	}
	var cfg SiteConfigGlobal
	if err := jsonUnmarshal(bytes, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal to SiteConfigGlobal: %w", err)
	}
	if cfg.Identity.Name.Zh == "" && cfg.Identity.Name.En == "" {
		return nil, errors.New("identity.name: at least one locale must be non-empty")
	}
	switch cfg.Identity.LocaleMode {
	case LocaleModeMonoZh, LocaleModeMonoEn, LocaleModeBilingual:
	default:
		return nil, fmt.Errorf("identity.localeMode: must be one of mono-zh, mono-en, bilingual (got %q)", cfg.Identity.LocaleMode)
	}
	if cfg.Identity.DefaultLocale != "zh" && cfg.Identity.DefaultLocale != "en" {
		return nil, fmt.Errorf("identity.defaultLocale: must be zh or en (got %q)", cfg.Identity.DefaultLocale)
	}
	switch cfg.Identity.LocaleMode {
	case LocaleModeMonoZh:
		if cfg.Identity.DefaultLocale != "zh" {
			return nil, errors.New("identity.defaultLocale: must equal zh when localeMode=mono-zh")
		}
	case LocaleModeMonoEn:
		if cfg.Identity.DefaultLocale != "en" {
			return nil, errors.New("identity.defaultLocale: must equal en when localeMode=mono-en")
		}
	}
	if len(cfg.Footer.ICP) > 100 {
		return nil, errors.New("footer.icp: max length 100")
	}
	for i, s := range cfg.Author.Socials {
		if strings.TrimSpace(s.URL) == "" {
			return nil, fmt.Errorf("author.socials[%d].url: required", i)
		}
	}
	return &cfg, nil
}
