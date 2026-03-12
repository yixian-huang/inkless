// Package analytics provides an analytics code injection plugin for Impress CMS.
// It subscribes to the frontend injection mechanism, inserting tracking scripts
// (Google Analytics 4, Baidu Analytics, Umami) into rendered pages.
package analytics

import (
	"fmt"
	"html/template"
	"strings"

	"blotting-consultancy/internal/plugin"
)

// Manifest describes this plugin's metadata.
var Manifest = plugin.PluginMeta{
	ID:            "ana-analytics",
	Name:          "Analytics Injection Plugin",
	NameZh:        "统计代码注入插件",
	Version:       "1.0.0",
	Description:   "Injects GA4, Baidu Analytics, or Umami tracking scripts into page <head>.",
	Author:        "Impress CMS",
	License:       "MIT",
	MinAppVersion: "1.0.0",
	Permissions:   []plugin.Permission{plugin.PermFrontendInject},
}

// Provider identifies the analytics service.
type Provider string

const (
	// ProviderGA4 uses Google Analytics 4 (gtag.js).
	ProviderGA4 Provider = "ga4"

	// ProviderBaidu uses Baidu Analytics (百度统计).
	ProviderBaidu Provider = "baidu"

	// ProviderUmami uses Umami Analytics (self-hosted, privacy-friendly).
	ProviderUmami Provider = "umami"
)

// Config holds the analytics plugin configuration.
type Config struct {
	// Provider selects the analytics service (ga4, baidu, umami).
	Provider Provider

	// MeasurementID is the GA4 measurement ID (e.g. "G-XXXXXXXXXX"). Required for GA4.
	MeasurementID string

	// SiteID is the Baidu Analytics site ID or Umami website ID.
	SiteID string

	// UmamiScriptURL is the URL of the Umami script (e.g. "https://umami.example.com/script.js").
	// Required only for Umami.
	UmamiScriptURL string

	// AnonymizeIP controls IP anonymization in GA4 (default: true).
	AnonymizeIP bool
}

// Plugin generates analytics tracking script fragments for frontend injection.
type Plugin struct {
	config Config
}

// New creates a new Analytics plugin with the provided configuration.
func New(cfg Config) (*Plugin, error) {
	if cfg.Provider == "" {
		return nil, fmt.Errorf("analytics: provider is required (ga4, baidu, umami)")
	}

	switch cfg.Provider {
	case ProviderGA4:
		if cfg.MeasurementID == "" {
			return nil, fmt.Errorf("analytics: GA4 measurement_id is required")
		}
	case ProviderBaidu:
		if cfg.SiteID == "" {
			return nil, fmt.Errorf("analytics: Baidu site_id is required")
		}
	case ProviderUmami:
		if cfg.SiteID == "" {
			return nil, fmt.Errorf("analytics: Umami website_id (site_id) is required")
		}
		if cfg.UmamiScriptURL == "" {
			return nil, fmt.Errorf("analytics: Umami script_url is required")
		}
	default:
		return nil, fmt.Errorf("analytics: unknown provider %q (expected ga4, baidu, umami)", cfg.Provider)
	}

	return &Plugin{config: cfg}, nil
}

// NewFromSettings creates a Plugin from a string settings map (used by plugin manager).
func NewFromSettings(settings map[string]string) (*Plugin, error) {
	cfg := Config{
		Provider:       Provider(settings["provider"]),
		MeasurementID:  settings["measurement_id"],
		SiteID:         settings["site_id"],
		UmamiScriptURL: settings["script_url"],
		AnonymizeIP:    settings["anonymize_ip"] != "false", // default true
	}
	return New(cfg)
}

// HeadScript returns the HTML <script> fragment to inject into the page <head>.
// The returned string is safe for direct inclusion in HTML (no additional escaping needed).
func (p *Plugin) HeadScript() template.HTML {
	switch p.config.Provider {
	case ProviderGA4:
		return p.ga4Script()
	case ProviderBaidu:
		return p.baiduScript()
	case ProviderUmami:
		return p.umamiScript()
	default:
		return ""
	}
}

// ga4Script generates the Google Analytics 4 (gtag.js) script fragment.
func (p *Plugin) ga4Script() template.HTML {
	measurementID := template.HTMLEscapeString(p.config.MeasurementID)
	anonymize := "true"
	if !p.config.AnonymizeIP {
		anonymize = "false"
	}

	// Note: gtag URL uses the measurement ID directly.
	return template.HTML(fmt.Sprintf(`<!-- Google Analytics 4 -->
<script async src="https://www.googletagmanager.com/gtag/js?id=%s"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());
  gtag('config', '%s', { 'anonymize_ip': %s });
</script>`, measurementID, measurementID, anonymize))
}

// baiduScript generates the Baidu Analytics (百度统计) script fragment.
func (p *Plugin) baiduScript() template.HTML {
	siteID := template.HTMLEscapeString(p.config.SiteID)
	return template.HTML(fmt.Sprintf(`<!-- Baidu Analytics -->
<script>
var _hmt = _hmt || [];
(function() {
  var hm = document.createElement("script");
  hm.src = "https://hm.baidu.com/hm.js?%s";
  var s = document.getElementsByTagName("script")[0];
  s.parentNode.insertBefore(hm, s);
})();
</script>`, siteID))
}

// umamiScript generates the Umami Analytics script fragment.
func (p *Plugin) umamiScript() template.HTML {
	websiteID := template.HTMLEscapeString(p.config.SiteID)
	scriptURL := template.HTMLEscapeString(p.config.UmamiScriptURL)
	return template.HTML(fmt.Sprintf(
		`<!-- Umami Analytics -->
<script async defer src="%s" data-website-id="%s"></script>`,
		scriptURL, websiteID))
}

// Description returns a human-readable description of the current configuration.
func (p *Plugin) Description() string {
	switch p.config.Provider {
	case ProviderGA4:
		return fmt.Sprintf("Google Analytics 4 (Measurement ID: %s)", p.config.MeasurementID)
	case ProviderBaidu:
		return fmt.Sprintf("Baidu Analytics (Site ID: %s)", p.config.SiteID)
	case ProviderUmami:
		return fmt.Sprintf("Umami Analytics (Website ID: %s, Script: %s)",
			p.config.SiteID, p.config.UmamiScriptURL)
	default:
		return "Unknown analytics provider"
	}
}

// InjectIntoHTML inserts the tracking script into an HTML string just before </head>.
// Returns the modified HTML, or the original HTML if </head> is not found.
func (p *Plugin) InjectIntoHTML(htmlContent string) string {
	script := string(p.HeadScript())
	if script == "" {
		return htmlContent
	}
	headClose := "</head>"
	idx := strings.Index(strings.ToLower(htmlContent), strings.ToLower(headClose))
	if idx == -1 {
		return htmlContent
	}
	return htmlContent[:idx] + "\n" + script + "\n" + htmlContent[idx:]
}
