package eventbus_test

import (
	"context"
	"errors"
	"testing"

	"blotting-consultancy/internal/eventbus"
)

func TestHookChainExecutesInOrder(t *testing.T) {
	chain := eventbus.NewHookChain()
	order := []int{}

	chain.Add("first", func(ctx context.Context, data interface{}) (interface{}, error) {
		order = append(order, 1)
		return data, nil
	})
	chain.Add("second", func(ctx context.Context, data interface{}) (interface{}, error) {
		order = append(order, 2)
		return data, nil
	})

	_, err := chain.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Errorf("expected [1,2], got %v", order)
	}
}

func TestHookChainAbortOnError(t *testing.T) {
	chain := eventbus.NewHookChain()

	chain.Add("fail", func(ctx context.Context, data interface{}) (interface{}, error) {
		return nil, errors.New("hook failed")
	})
	chain.Add("never", func(ctx context.Context, data interface{}) (interface{}, error) {
		t.Error("this hook should not be called")
		return data, nil
	})

	_, err := chain.Execute(context.Background(), nil)
	if err == nil {
		t.Error("expected error from failed hook")
	}
}

func TestHookChainDataTransform(t *testing.T) {
	chain := eventbus.NewHookChain()

	chain.Add("transform", func(ctx context.Context, data interface{}) (interface{}, error) {
		s := data.(string)
		return s + " modified", nil
	})

	result, err := chain.Execute(context.Background(), "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "input modified" {
		t.Errorf("expected 'input modified', got %v", result)
	}
}

func TestHookChainLen(t *testing.T) {
	chain := eventbus.NewHookChain()
	if chain.Len() != 0 {
		t.Errorf("expected empty chain, got len=%d", chain.Len())
	}

	chain.Add("a", func(ctx context.Context, data interface{}) (interface{}, error) {
		return data, nil
	})
	chain.Add("b", func(ctx context.Context, data interface{}) (interface{}, error) {
		return data, nil
	})

	if chain.Len() != 2 {
		t.Errorf("expected len=2, got %d", chain.Len())
	}
}

func TestHookChainEmptyExecute(t *testing.T) {
	chain := eventbus.NewHookChain()
	result, err := chain.Execute(context.Background(), "unchanged")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "unchanged" {
		t.Errorf("expected 'unchanged', got %v", result)
	}
}

func TestHookRegistryExecute(t *testing.T) {
	reg := eventbus.NewHookRegistry()

	reg.Register(eventbus.HookBeforePublish, "validate", func(ctx context.Context, data interface{}) (interface{}, error) {
		return data.(string) + " validated", nil
	})
	reg.Register(eventbus.HookBeforePublish, "transform", func(ctx context.Context, data interface{}) (interface{}, error) {
		return data.(string) + " transformed", nil
	})

	result, err := reg.Execute(context.Background(), eventbus.HookBeforePublish, "content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "content validated transformed" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestHookRegistryMissingHookPoint(t *testing.T) {
	reg := eventbus.NewHookRegistry()

	result, err := reg.Execute(context.Background(), "nonexistent", "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "data" {
		t.Errorf("expected original data for missing hook point, got %v", result)
	}
}

func TestHookPointConstants(t *testing.T) {
	constants := []string{
		eventbus.HookBeforePublish,
		eventbus.HookAfterPublish,
		eventbus.HookBeforeRender,
		eventbus.HookBeforeCreate,
		eventbus.HookAfterCreate,
		eventbus.HookBeforeDelete,
		eventbus.HookAfterDelete,
	}
	for _, c := range constants {
		if c == "" {
			t.Error("hook point constant should not be empty")
		}
	}
}
