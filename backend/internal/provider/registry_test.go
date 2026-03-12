package provider_test

import (
	"context"
	"testing"

	"blotting-consultancy/internal/provider"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := provider.NewRegistry()

	captcha := &provider.NoopCaptchaProvider{}
	reg.Register("captcha", captcha)

	got := reg.Get("captcha")
	if got == nil {
		t.Fatal("expected to get captcha provider")
	}
	if _, ok := got.(*provider.NoopCaptchaProvider); !ok {
		t.Error("expected NoopCaptchaProvider type")
	}
}

func TestRegistryReplaceOverwrites(t *testing.T) {
	reg := provider.NewRegistry()

	first := &mockNotifier{name: "first"}
	second := &mockNotifier{name: "second"}

	reg.Register("notifier", first)
	reg.Register("notifier", second) // should overwrite

	got := reg.Get("notifier")
	if got == nil {
		t.Fatal("expected to get notifier provider")
	}
	mn, ok := got.(*mockNotifier)
	if !ok {
		t.Fatal("expected mockNotifier type")
	}
	if mn.name != "second" {
		t.Errorf("expected second provider, got %q", mn.name)
	}
}

func TestRegistryGetMissing(t *testing.T) {
	reg := provider.NewRegistry()
	if got := reg.Get("nonexistent"); got != nil {
		t.Errorf("expected nil for missing provider, got %v", got)
	}
}

func TestRegistryList(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register("a", &provider.NoopCaptchaProvider{})
	reg.Register("b", &provider.NoopCaptchaProvider{})

	list := reg.List()
	if len(list) != 2 {
		t.Errorf("expected 2 providers, got %d", len(list))
	}
}

func TestRegistryMustGetPanics(t *testing.T) {
	reg := provider.NewRegistry()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet should panic on missing provider")
		}
	}()

	reg.MustGet("missing")
}

func TestRegistryMustGetSuccess(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register("captcha", &provider.NoopCaptchaProvider{})

	got := reg.MustGet("captcha")
	if got == nil {
		t.Error("MustGet should return the registered provider")
	}
}

type mockNotifier struct {
	name string
}

func (m *mockNotifier) Notify(ctx context.Context, event provider.NotifyEvent) error {
	return nil
}
