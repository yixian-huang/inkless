package eventbus

import (
	"context"
)

// Hook point names for request-level interception.
const (
	HookBeforePublish = "hook.before_publish"
	HookAfterPublish  = "hook.after_publish"
	HookBeforeRender  = "hook.before_render"
	HookBeforeCreate  = "hook.before_create"
	HookAfterCreate   = "hook.after_create"
	HookBeforeDelete  = "hook.before_delete"
	HookAfterDelete   = "hook.after_delete"
)

// HookFunc is a function that can inspect and transform data at a hook point.
// It receives context and the current data, and returns potentially modified data.
// Returning an error aborts the chain.
type HookFunc func(ctx context.Context, data interface{}) (interface{}, error)

// HookEntry pairs a name with a hook function for debugging/logging.
type HookEntry struct {
	Name string
	Fn   HookFunc
}

// HookChain manages an ordered list of hooks for a specific hook point.
type HookChain struct {
	hooks []HookEntry
}

// NewHookChain creates an empty hook chain.
func NewHookChain() *HookChain {
	return &HookChain{}
}

// Add appends a named hook to the chain.
func (c *HookChain) Add(name string, fn HookFunc) {
	c.hooks = append(c.hooks, HookEntry{Name: name, Fn: fn})
}

// Execute runs all hooks in order, threading data through each.
// Stops and returns on the first error.
func (c *HookChain) Execute(ctx context.Context, data interface{}) (interface{}, error) {
	var err error
	for _, h := range c.hooks {
		data, err = h.Fn(ctx, data)
		if err != nil {
			return data, err
		}
	}
	return data, nil
}

// Len returns the number of hooks in the chain.
func (c *HookChain) Len() int {
	return len(c.hooks)
}

// HookRegistry manages hook chains for multiple hook points.
type HookRegistry struct {
	chains map[string]*HookChain
}

// NewHookRegistry creates a new HookRegistry.
func NewHookRegistry() *HookRegistry {
	return &HookRegistry{chains: make(map[string]*HookChain)}
}

// Register adds a hook function to a named hook point.
func (r *HookRegistry) Register(hookPoint string, name string, fn HookFunc) {
	if _, ok := r.chains[hookPoint]; !ok {
		r.chains[hookPoint] = NewHookChain()
	}
	r.chains[hookPoint].Add(name, fn)
}

// Execute runs the hook chain for a given hook point.
// Returns the original data unchanged if no hooks are registered.
func (r *HookRegistry) Execute(ctx context.Context, hookPoint string, data interface{}) (interface{}, error) {
	chain, ok := r.chains[hookPoint]
	if !ok {
		return data, nil
	}
	return chain.Execute(ctx, data)
}
