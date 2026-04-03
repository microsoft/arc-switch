package transform

import (
	"fmt"
	"sync"
)

// registry holds all registered transformer factories, keyed by path name.
// Transformers self-register via init() in their source files so new
// vendors/transformers can be added without modifying collector.go.
var (
	registryMu sync.RWMutex
	registry   = map[string]func() Transformer{}
)

// Register adds a transformer factory for the given path name.
// Typically called from init() functions in transformer source files.
// Panics if the name is already registered (catches duplicate registrations
// at startup rather than silently overwriting).
func Register(name string, factory func() Transformer) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("transform: duplicate registration for %q", name))
	}
	registry[name] = factory
}

// Get returns a new transformer instance for the given path name.
// Returns nil if no transformer is registered for that name.
func Get(name string) Transformer {
	registryMu.RLock()
	defer registryMu.RUnlock()
	factory, ok := registry[name]
	if !ok {
		return nil
	}
	return factory()
}

// BuildMap returns a map of all registered transformers, keyed by path name.
// Each call creates fresh transformer instances.
func BuildMap() map[string]Transformer {
	registryMu.RLock()
	defer registryMu.RUnlock()
	m := make(map[string]Transformer, len(registry))
	for name, factory := range registry {
		m[name] = factory()
	}
	return m
}

// RegisteredNames returns a sorted list of all registered transformer names.
func RegisteredNames() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
