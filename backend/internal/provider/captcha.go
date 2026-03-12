package provider

import (
	"context"
)

// CaptchaProvider defines the contract for captcha verification.
// Default implementation is a noop that always passes.
// Plugins can replace with reCAPTCHA, hCaptcha, Turnstile, etc.
type CaptchaProvider interface {
	// Verify checks a captcha response token. Returns nil if valid.
	Verify(ctx context.Context, token string, remoteIP string) error
}

// NoopCaptchaProvider always passes captcha verification.
// Used as a default when no captcha service is configured.
type NoopCaptchaProvider struct{}

// Verify always returns nil (passes verification).
func (n *NoopCaptchaProvider) Verify(ctx context.Context, token string, remoteIP string) error {
	return nil
}
