package service

import (
	"fmt"
	"strings"

	"github.com/yixian-huang/inkless/backend/internal/contentslots"
	"github.com/yixian-huang/inkless/backend/internal/model"
)

// ValidationError represents a field-level validation error
type ValidationError struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TranslationState represents the translation status of a field
type TranslationState string

const (
	TranslationStateDone    TranslationState = "done"
	TranslationStateMissing TranslationState = "missing"
	TranslationStateStale   TranslationState = "stale"
)

// ValidationResult represents the validation outcome
type ValidationResult struct {
	Valid             bool                        `json:"valid"`
	Errors            []ValidationError           `json:"errors"`
	TranslationStatus map[string]TranslationState `json:"translationStatus"`
	// SchemaKind is a discovery hint for agents: product-first | corporate | empty | unknown | global | …
	SchemaKind string `json:"schemaKind,omitempty"`
	// Theme contract discovery (when contentSlots apply).
	SchemaID     string `json:"schemaId,omitempty"`
	SchemaSource string `json:"schemaSource,omitempty"` // theme | host-fallback | none
}

// ValidationService provides content validation and translation state tracking
type ValidationService struct{}

// NewValidationService creates a new validation service
func NewValidationService() *ValidationService {
	return &ValidationService{}
}

// ValidateConfig validates a page configuration and calculates translation states.
// Always enforces MediaRef string leaves (url/alt/caption) across the tree.
func (vs *ValidationService) ValidateConfig(pageKey model.PageKey, config model.JSONMap) *ValidationResult {
	return vs.ValidateConfigWithSlot(pageKey, config, nil, "")
}

// ValidateConfigWithSlot is ValidateConfig plus optional theme contentSlots path rules.
// When slot is non-nil, host shape heuristics for home are skipped (theme is source of truth).
func (vs *ValidationService) ValidateConfigWithSlot(
	pageKey model.PageKey,
	config model.JSONMap,
	slot *contentslots.Slot,
	schemaSource string,
) *ValidationResult {
	result := &ValidationResult{
		Valid:             true,
		Errors:            []ValidationError{},
		TranslationStatus: make(map[string]TranslationState),
		SchemaSource:      schemaSource,
	}

	if config == nil {
		config = model.JSONMap{}
	}

	// Hard gate: MediaRef leaves must be plain strings (product-first + corporate).
	result.Errors = append(result.Errors, CollectMediaRefLeafErrors(config)...)

	if slot != nil {
		result.SchemaID = slot.SchemaID
		if result.SchemaSource == "" {
			result.SchemaSource = "theme"
		}
		// Prefer schemaId theme prefix as kind hint
		if slot.SchemaID != "" {
			if i := strings.IndexByte(slot.SchemaID, '/'); i > 0 {
				result.SchemaKind = slot.SchemaID[:i]
			} else {
				result.SchemaKind = slot.SchemaID
			}
		}
		for _, pe := range contentslots.ValidateConfigAgainstSlot(config, *slot) {
			result.Errors = append(result.Errors, ValidationError{
				Path: pe.Path, Code: pe.Code, Message: pe.Message,
			})
		}
		// Still allow light product-first structural checks when schema is product-first
		if pageKey == model.PageKeyHome && (result.SchemaKind == "product-first" || isProductFirstHomeConfig(config)) {
			vs.validateProductFirstHome(config, result)
		}
		result.Valid = len(result.Errors) == 0
		return result
	}

	if result.SchemaSource == "" {
		result.SchemaSource = "host-fallback"
	}

	switch pageKey {
	case model.PageKeyHome:
		vs.validateHomePage(config, result)
	case model.PageKeyAbout:
		result.SchemaKind = "corporate"
		vs.validateAboutPage(config, result)
	case model.PageKeyAdvantages:
		result.SchemaKind = "corporate"
		vs.validateAdvantagesPage(config, result)
	case model.PageKeyCoreServices:
		result.SchemaKind = "corporate"
		vs.validateCoreServicesPage(config, result)
	case model.PageKeyCases:
		result.SchemaKind = "corporate"
		vs.validateCasesPage(config, result)
	case model.PageKeyExperts:
		result.SchemaKind = "corporate"
		vs.validateExpertsPage(config, result)
	case model.PageKeyContact:
		result.SchemaKind = "contact"
		vs.validateContactPage(config, result)
	case model.PageKeyGlobal:
		result.SchemaKind = "global"
		vs.validateGlobalPage(config, result)
	case model.PageKeyTheme:
		result.SchemaKind = "theme"
	default:
		result.SchemaKind = "unknown"
		result.SchemaSource = "none"
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Path:    "pageKey",
			Code:    "INVALID_PAGE_KEY",
			Message: fmt.Sprintf("Invalid page key: %s", pageKey),
		})
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// CanPublish checks if a configuration can be published based on translation state
func (vs *ValidationService) CanPublish(validationResult *ValidationResult) bool {
	if !validationResult.Valid {
		return false
	}

	// Block publish if any required field is missing or stale
	for _, state := range validationResult.TranslationStatus {
		if state == TranslationStateMissing || state == TranslationStateStale {
			return false
		}
	}

	return true
}

// Helper functions for validation

func (vs *ValidationService) validateHomePage(config model.JSONMap, result *ValidationResult) {
	if len(config) == 0 {
		result.SchemaKind = "empty"
		return
	}
	// product-first home schema (hero/showcase/features/…) — do not require corporate blocks.
	if isProductFirstHomeConfig(config) {
		result.SchemaKind = "product-first"
		vs.validateProductFirstHome(config, result)
		return
	}

	result.SchemaKind = "corporate"
	// Corporate / legacy home schema
	hero := getMapField(config, "hero")
	if hero == nil {
		addRequiredError(result, "hero", "Hero section is required")
	} else {
		validateLocalizedText(hero, "hero.title", result, true)
		validateLocalizedText(hero, "hero.subtitle", result, true)
		validateMediaRef(hero, "hero.backgroundImage", result, true)
	}

	about := getMapField(config, "about")
	if about == nil {
		addRequiredError(result, "about", "About section is required")
	} else {
		validateLocalizedText(about, "about.title", result, true)
		validateMediaRef(about, "about.image", result, true)
		validateCta(about, "about.cta", result, true)

		descriptions := getArrayField(about, "descriptions")
		if descriptions == nil || len(descriptions) == 0 {
			addRequiredError(result, "about.descriptions", "At least one description is required")
		} else {
			for i, desc := range descriptions {
				if descMap, ok := desc.(map[string]interface{}); ok {
					path := fmt.Sprintf("about.descriptions[%d]", i)
					validateLocalizedTextMap(descMap, path, result, true)
				}
			}
		}
	}

	advantages := getMapField(config, "advantages")
	if advantages == nil {
		addRequiredError(result, "advantages", "Advantages section is required")
	} else {
		validateLocalizedText(advantages, "advantages.title", result, true)
		cards := getArrayField(advantages, "cards")
		if cards == nil || len(cards) == 0 {
			addRequiredError(result, "advantages.cards", "At least one advantage card is required")
		} else {
			for i, card := range cards {
				if cardMap, ok := card.(map[string]interface{}); ok {
					basePath := fmt.Sprintf("advantages.cards[%d]", i)
					validateLocalizedText(cardMap, basePath+".title", result, true)
					validateLocalizedText(cardMap, basePath+".titleEn", result, true)
					validateLocalizedText(cardMap, basePath+".description", result, true)
					validateMediaRef(cardMap, basePath+".image", result, true)
				}
			}
		}
	}

	coreServices := getMapField(config, "coreServices")
	if coreServices == nil {
		addRequiredError(result, "coreServices", "Core services section is required")
	} else {
		validateLocalizedText(coreServices, "coreServices.title", result, true)
		items := getArrayField(coreServices, "items")
		if items == nil || len(items) == 0 {
			addRequiredError(result, "coreServices.items", "At least one service item is required")
		} else {
			for i, item := range items {
				if itemMap, ok := item.(map[string]interface{}); ok {
					basePath := fmt.Sprintf("coreServices.items[%d]", i)
					validateLocalizedText(itemMap, basePath+".title", result, true)
					validateLocalizedText(itemMap, basePath+".description", result, true)
					validateMediaRef(itemMap, basePath+".image", result, true)
					validateCta(itemMap, basePath+".cta", result, true)
				}
			}
		}
	}
}

// isProductFirstHomeConfig detects product-first landing schema vs corporate home.
func isProductFirstHomeConfig(config model.JSONMap) bool {
	if config == nil {
		return false
	}
	// Strong product-first markers
	for _, key := range []string{"showcase", "howItWorks", "install", "bottomCta"} {
		if _, ok := config[key]; ok {
			return true
		}
	}
	// features grid without corporate about/coreServices
	if _, hasFeatures := config["features"]; hasFeatures {
		if _, hasAbout := config["about"]; !hasAbout {
			if _, hasCore := config["coreServices"]; !hasCore {
				return true
			}
		}
	}
	// hero.media (product) without corporate required siblings when only hero present
	if hero := getMapField(config, "hero"); hero != nil {
		if _, hasMedia := hero["media"]; hasMedia {
			if _, hasAbout := config["about"]; !hasAbout {
				return true
			}
		}
		// hero with localized title only and no backgroundImage/about → product-first agent draft
		if _, hasBG := hero["backgroundImage"]; !hasBG {
			if _, hasAbout := config["about"]; !hasAbout {
				if _, hasAdv := config["advantages"]; !hasAdv {
					return true
				}
			}
		}
	}
	return false
}

func (vs *ValidationService) validateProductFirstHome(config model.JSONMap, result *ValidationResult) {
	// Light structure checks — MediaRef leaves already collected globally.
	// Empty sections are allowed (theme placeholders); when present, check shapes.
	hero := getMapField(config, "hero")
	if hero != nil {
		if title, ok := hero["title"]; ok && title != nil {
			if m, isMap := title.(map[string]interface{}); isMap {
				validateLocalizedTextMap(m, "hero.title", result, false)
			}
		}
		if sub, ok := hero["subtitle"]; ok && sub != nil {
			if m, isMap := sub.(map[string]interface{}); isMap {
				validateLocalizedTextMap(m, "hero.subtitle", result, false)
			}
		}
		if media := getMapField(hero, "media"); media != nil {
			if _, hasURL := media["url"]; hasURL && strings.TrimSpace(getStringField(media, "url")) == "" {
				result.Errors = append(result.Errors, ValidationError{
					Path: "hero.media.url", Code: "REQUIRED", Message: "media url is empty",
				})
			}
		}
	}

	if showcase := getMapField(config, "showcase"); showcase != nil {
		items := getArrayField(showcase, "items")
		for i, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				result.Errors = append(result.Errors, ValidationError{
					Path:    fmt.Sprintf("showcase.items[%d]", i),
					Code:    "INVALID_TYPE",
					Message: "showcase item must be a MediaRef object",
				})
				continue
			}
			// Prefer objects with url when present; allow empty placeholder objects.
			if url, has := itemMap["url"]; has {
				if s, ok := url.(string); !ok {
					result.Errors = append(result.Errors, ValidationError{
						Path:    fmt.Sprintf("showcase.items[%d].url", i),
						Code:    "MEDIAREF_TYPE",
						Message: "url must be a string",
					})
				} else if strings.TrimSpace(s) == "" {
					result.Errors = append(result.Errors, ValidationError{
						Path:    fmt.Sprintf("showcase.items[%d].url", i),
						Code:    "REQUIRED",
						Message: "showcase item url is empty",
					})
				}
			}
		}
	}

	if features := getMapField(config, "features"); features != nil {
		items := getArrayField(features, "items")
		for i, item := range items {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				result.Errors = append(result.Errors, ValidationError{
					Path:    fmt.Sprintf("features.items[%d]", i),
					Code:    "INVALID_TYPE",
					Message: "feature item must be an object",
				})
				continue
			}
			base := fmt.Sprintf("features.items[%d]", i)
			if title, ok := itemMap["title"]; ok && title != nil {
				if m, isMap := title.(map[string]interface{}); isMap {
					validateLocalizedTextMap(m, base+".title", result, false)
				}
			}
			if desc, ok := itemMap["description"]; ok && desc != nil {
				if m, isMap := desc.(map[string]interface{}); isMap {
					validateLocalizedTextMap(m, base+".description", result, false)
				}
			}
		}
	}

	if install := getMapField(config, "install"); install != nil {
		if code, ok := install["code"]; ok && code != nil {
			if _, isStr := code.(string); !isStr {
				result.Errors = append(result.Errors, ValidationError{
					Path:    "install.code",
					Code:    "INVALID_TYPE",
					Message: "install.code must be a plain string (not LocalizedText)",
				})
			}
		}
	}
}

func (vs *ValidationService) validateAboutPage(config model.JSONMap, result *ValidationResult) {
	// Validate hero
	hero := getMapField(config, "hero")
	if hero == nil {
		addRequiredError(result, "hero", "Hero section is required")
	} else {
		validateLocalizedText(hero, "hero.label", result, true)
		validateLocalizedText(hero, "hero.title", result, true)
		validateMediaRef(hero, "hero.image", result, true)
	}

	// Validate companyProfile
	profile := getMapField(config, "companyProfile")
	if profile == nil {
		addRequiredError(result, "companyProfile", "Company profile section is required")
	} else {
		validateLocalizedText(profile, "companyProfile.title", result, true)
		validateLocalizedText(profile, "companyProfile.description", result, true)
	}

	// Validate blocks
	blocks := getArrayField(config, "blocks")
	if blocks == nil {
		addRequiredError(result, "blocks", "Blocks section is required")
	} else {
		for i, block := range blocks {
			if blockMap, ok := block.(map[string]interface{}); ok {
				basePath := fmt.Sprintf("blocks[%d]", i)
				validateLocalizedText(blockMap, basePath+".title", result, false)
				validateLocalizedText(blockMap, basePath+".description", result, true)
				validateMediaRef(blockMap, basePath+".image", result, true)
			}
		}
	}
}

func (vs *ValidationService) validateAdvantagesPage(config model.JSONMap, result *ValidationResult) {
	// Validate hero
	hero := getMapField(config, "hero")
	if hero == nil {
		addRequiredError(result, "hero", "Hero section is required")
	} else {
		validateLocalizedText(hero, "hero.label", result, true)
		validateLocalizedText(hero, "hero.title", result, true)
		validateMediaRef(hero, "hero.image", result, true)
	}

	// Validate blocks
	blocks := getArrayField(config, "blocks")
	if blocks == nil || len(blocks) == 0 {
		addRequiredError(result, "blocks", "At least one advantage block is required")
	} else {
		for i, block := range blocks {
			if blockMap, ok := block.(map[string]interface{}); ok {
				basePath := fmt.Sprintf("blocks[%d]", i)
				validateLocalizedText(blockMap, basePath+".title", result, true)
				validateLocalizedText(blockMap, basePath+".description", result, true)
				validateMediaRef(blockMap, basePath+".image", result, true)
			}
		}
	}
}

func (vs *ValidationService) validateCoreServicesPage(config model.JSONMap, result *ValidationResult) {
	// Validate hero
	hero := getMapField(config, "hero")
	if hero == nil {
		addRequiredError(result, "hero", "Hero section is required")
	} else {
		validateLocalizedText(hero, "hero.label", result, true)
		validateLocalizedText(hero, "hero.title", result, true)
		validateMediaRef(hero, "hero.image", result, true)
	}

	// Validate services
	services := getArrayField(config, "services")
	if services == nil || len(services) == 0 {
		addRequiredError(result, "services", "At least one service is required")
	} else {
		for i, service := range services {
			if serviceMap, ok := service.(map[string]interface{}); ok {
				basePath := fmt.Sprintf("services[%d]", i)
				validateLocalizedText(serviceMap, basePath+".title", result, true)
				validateLocalizedText(serviceMap, basePath+".description", result, true)
				validateMediaRef(serviceMap, basePath+".image", result, true)
			}
		}
	}
}

func (vs *ValidationService) validateCasesPage(config model.JSONMap, result *ValidationResult) {
	// Validate hero
	hero := getMapField(config, "hero")
	if hero == nil {
		addRequiredError(result, "hero", "Hero section is required")
	} else {
		validateLocalizedText(hero, "hero.label", result, true)
		validateLocalizedText(hero, "hero.title", result, true)
		validateMediaRef(hero, "hero.image", result, true)
	}

	// Validate cases
	cases := getArrayField(config, "cases")
	if cases == nil || len(cases) == 0 {
		addRequiredError(result, "cases", "At least one case is required")
	} else {
		for i, caseItem := range cases {
			if caseMap, ok := caseItem.(map[string]interface{}); ok {
				basePath := fmt.Sprintf("cases[%d]", i)
				validateLocalizedText(caseMap, basePath+".title", result, true)

				items := getArrayField(caseMap, "items")
				if items == nil || len(items) == 0 {
					addRequiredError(result, basePath+".items", "At least one case item is required")
				} else {
					for j, item := range items {
						if itemMap, ok := item.(map[string]interface{}); ok {
							itemPath := fmt.Sprintf("%s.items[%d]", basePath, j)
							validateLocalizedTextMap(itemMap, itemPath, result, true)
						}
					}
				}
			}
		}
	}
}

func (vs *ValidationService) validateExpertsPage(config model.JSONMap, result *ValidationResult) {
	// Validate hero
	hero := getMapField(config, "hero")
	if hero == nil {
		addRequiredError(result, "hero", "Hero section is required")
	} else {
		validateLocalizedText(hero, "hero.label", result, true)
		validateLocalizedText(hero, "hero.title", result, true)
		validateMediaRef(hero, "hero.image", result, true)
	}

	// Validate sectionTitle
	sectionTitle := getMapField(config, "sectionTitle")
	if sectionTitle == nil {
		addRequiredError(result, "sectionTitle", "Section title is required")
	} else {
		validateLocalizedTextMap(sectionTitle, "sectionTitle", result, true)
	}

	// Validate experts
	experts := getArrayField(config, "experts")
	if experts == nil || len(experts) == 0 {
		addRequiredError(result, "experts", "At least one expert is required")
	} else {
		for i, expert := range experts {
			if expertMap, ok := expert.(map[string]interface{}); ok {
				basePath := fmt.Sprintf("experts[%d]", i)

				// id is required but not bilingual
				if id := getStringField(expertMap, "id"); id == "" {
					addRequiredError(result, basePath+".id", "Expert ID is required")
				}

				validateLocalizedTextZhRequired(expertMap, basePath+".name", result)
				validateLocalizedTextZhRequired(expertMap, basePath+".title", result)
				validateMediaRef(expertMap, basePath+".avatar", result, true)

				bioParagraphs := getArrayField(expertMap, "bioParagraphs")
				if bioParagraphs == nil || len(bioParagraphs) == 0 {
					addRequiredError(result, basePath+".bioParagraphs", "At least one bio paragraph is required")
				} else {
					for j, para := range bioParagraphs {
						if paraMap, ok := para.(map[string]interface{}); ok {
							paraPath := fmt.Sprintf("%s.bioParagraphs[%d]", basePath, j)
							validateLocalizedTextMap(paraMap, paraPath, result, true)
						}
					}
				}
			}
		}
	}
}

func (vs *ValidationService) validateContactPage(config model.JSONMap, result *ValidationResult) {
	// Validate hero
	hero := getMapField(config, "hero")
	if hero == nil {
		addRequiredError(result, "hero", "Hero section is required")
	} else {
		validateLocalizedText(hero, "hero.title", result, true)
		validateLocalizedText(hero, "hero.subtitle", result, true)
		// backgroundColor is optional
	}

	// Validate form
	form := getMapField(config, "form")
	if form == nil {
		addRequiredError(result, "form", "Form section is required")
	} else {
		validateLocalizedText(form, "form.title", result, true)
		validateLocalizedText(form, "form.subtitle", result, true)
		validateLocalizedText(form, "form.submitLabel", result, true)
	}

	// Validate contactInfo
	contactInfo := getMapField(config, "contactInfo")
	if contactInfo == nil {
		addRequiredError(result, "contactInfo", "Contact info section is required")
	} else {
		validateLocalizedText(contactInfo, "contactInfo.phone", result, true)
		validateLocalizedText(contactInfo, "contactInfo.address", result, true)
	}
}

func (vs *ValidationService) validateGlobalPage(config model.JSONMap, result *ValidationResult) {
	// Validate branding
	branding := getMapField(config, "branding")
	if branding == nil {
		addRequiredError(result, "branding", "Branding section is required")
	} else {
		validateMediaRef(branding, "branding.logo", result, true)
		validateLocalizedText(branding, "branding.companyName", result, true)
	}

	// Validate nav
	nav := getMapField(config, "nav")
	if nav == nil {
		addRequiredError(result, "nav", "Navigation section is required")
	}

	// Validate footer
	footer := getMapField(config, "footer")
	if footer == nil {
		addRequiredError(result, "footer", "Footer section is required")
	} else {
		validateLocalizedText(footer, "footer.address", result, false)
		validateLocalizedText(footer, "footer.phone", result, false)
		validateLocalizedText(footer, "footer.copyright", result, false)
	}
}

// Field extraction helpers

func getMapField(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if mapVal, ok := v.(map[string]interface{}); ok {
			return mapVal
		}
	}
	return nil
}

func getArrayField(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key]; ok {
		if arrVal, ok := v.([]interface{}); ok {
			return arrVal
		}
	}
	return nil
}

func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if strVal, ok := v.(string); ok {
			return strVal
		}
	}
	return ""
}

// Validation helpers

func validateLocalizedText(parent map[string]interface{}, fullPath string, result *ValidationResult, required bool) {
	// Extract the field name from fullPath (last segment after last dot)
	parts := strings.Split(fullPath, ".")
	fieldName := parts[len(parts)-1]

	field := getMapField(parent, fieldName)
	if field == nil {
		if required {
			addRequiredError(result, fullPath, "Field is required")
		}
		return
	}

	validateLocalizedTextMap(field, fullPath, result, required)
}

func validateLocalizedTextMap(field map[string]interface{}, fullPath string, result *ValidationResult, required bool) {
	zh := getStringField(field, "zh")
	en := getStringField(field, "en")

	hasZh := strings.TrimSpace(zh) != ""
	hasEn := strings.TrimSpace(en) != ""

	if required {
		if !hasZh && !hasEn {
			result.TranslationStatus[fullPath] = TranslationStateMissing
			result.Errors = append(result.Errors, ValidationError{
				Path:    fullPath + ".zh",
				Code:    "REQUIRED",
				Message: "Chinese text is required",
			})
			result.Errors = append(result.Errors, ValidationError{
				Path:    fullPath + ".en",
				Code:    "REQUIRED",
				Message: "English text is required",
			})
		} else if !hasZh {
			result.TranslationStatus[fullPath] = TranslationStateMissing
			result.Errors = append(result.Errors, ValidationError{
				Path:    fullPath + ".zh",
				Code:    "REQUIRED",
				Message: "Chinese text is required",
			})
		} else if !hasEn {
			result.TranslationStatus[fullPath] = TranslationStateMissing
			result.Errors = append(result.Errors, ValidationError{
				Path:    fullPath + ".en",
				Code:    "REQUIRED",
				Message: "English text is required",
			})
		} else {
			result.TranslationStatus[fullPath] = TranslationStateDone
		}
	}
}

// validateLocalizedTextZhRequired validates a localized text field where Chinese is required but English is optional.
func validateLocalizedTextZhRequired(parent map[string]interface{}, fullPath string, result *ValidationResult) {
	parts := strings.Split(fullPath, ".")
	fieldName := parts[len(parts)-1]

	field := getMapField(parent, fieldName)
	if field == nil {
		addRequiredError(result, fullPath, "Field is required")
		return
	}

	zh := getStringField(field, "zh")
	hasZh := strings.TrimSpace(zh) != ""

	if !hasZh {
		result.TranslationStatus[fullPath] = TranslationStateMissing
		result.Errors = append(result.Errors, ValidationError{
			Path:    fullPath + ".zh",
			Code:    "REQUIRED",
			Message: "Chinese text is required",
		})
	} else {
		result.TranslationStatus[fullPath] = TranslationStateDone
	}
}

func validateMediaRef(parent map[string]interface{}, fullPath string, result *ValidationResult, required bool) {
	parts := strings.Split(fullPath, ".")
	fieldName := parts[len(parts)-1]

	field := getMapField(parent, fieldName)
	if field == nil {
		if required {
			addRequiredError(result, fullPath, "Media reference is required")
		}
		return
	}

	url := getStringField(field, "url")
	if required && strings.TrimSpace(url) == "" {
		result.Errors = append(result.Errors, ValidationError{
			Path:    fullPath + ".url",
			Code:    "REQUIRED",
			Message: "Media URL is required",
		})
	}

	// MediaRef.alt / caption must be plain strings (not LocalizedText bags).
	for _, leaf := range []string{"url", "alt", "caption"} {
		if val, ok := field[leaf]; ok && val != nil {
			if _, isStr := val.(string); !isStr {
				result.Errors = append(result.Errors, ValidationError{
					Path:    fullPath + "." + leaf,
					Code:    "MEDIAREF_TYPE",
					Message: leaf + " must be a string (not a bilingual object or other type)",
				})
			}
		}
	}
}

func validateCta(parent map[string]interface{}, fullPath string, result *ValidationResult, required bool) {
	parts := strings.Split(fullPath, ".")
	fieldName := parts[len(parts)-1]

	field := getMapField(parent, fieldName)
	if field == nil {
		if required {
			addRequiredError(result, fullPath, "CTA is required")
		}
		return
	}

	href := getStringField(field, "href")
	if required && strings.TrimSpace(href) == "" {
		result.Errors = append(result.Errors, ValidationError{
			Path:    fullPath + ".href",
			Code:    "REQUIRED",
			Message: "CTA href is required",
		})
	}

	// Validate label as LocalizedText
	label := getMapField(field, "label")
	if label != nil {
		validateLocalizedTextMap(label, fullPath+".label", result, required)
	} else if required {
		addRequiredError(result, fullPath+".label", "CTA label is required")
	}
}

func addRequiredError(result *ValidationResult, path string, message string) {
	result.Errors = append(result.Errors, ValidationError{
		Path:    path,
		Code:    "REQUIRED",
		Message: message,
	})
}
