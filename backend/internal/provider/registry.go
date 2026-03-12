package provider

import (
	"log"
	"sync"
)

// Registry is a centralized store for Provider instances.
// Same-type providers follow last-registration-wins semantics.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]interface{}
}

// NewRegistry creates a new Provider Registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]interface{}),
	}
}

// Register adds or replaces a provider by name.
// If a provider with the same name already exists, it is replaced and a log entry is emitted.
func (r *Registry) Register(name string, provider interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing, ok := r.providers[name]; ok {
		log.Printf("[Registry] replacing provider %q: %T -> %T", name, existing, provider)
	}
	r.providers[name] = provider
	log.Printf("[Registry] registered provider %q (%T)", name, provider)
}

// Get retrieves a provider by name. Returns nil if not found.
func (r *Registry) Get(name string) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.providers[name]
}

// List returns all registered provider names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// MustGet retrieves a provider by name and panics if not found.
// Use in startup code where missing providers are fatal.
func (r *Registry) MustGet(name string) interface{} {
	p := r.Get(name)
	if p == nil {
		panic("required provider not registered: " + name)
	}
	return p
}
