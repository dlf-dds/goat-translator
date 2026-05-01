// Package adapter defines the contract every format adapter implements
// and the registry that maps format names to adapters.
//
// Every format the translator handles defines two functions: Decode (wire
// bytes → canonical entity) and Encode (canonical entity → wire bytes).
// Translation between two formats is the composition of two adapters via
// the canonical pivot.
//
// Adapters live in their own packages (open standards in this repo under
// adapters/, vendor SDKs in their respective private repos behind build
// tags). They register themselves at package init time via Register().
package adapter

import (
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/dlf-dds/goat-translator/internal/canonical"
)

// Adapter is the contract every format implements. Adapters are stateless
// and must be safe for concurrent use by multiple goroutines.
type Adapter interface {
	// Name returns the canonical, lowercase name of the format
	// ("cot", "lattice", "sapient", "picogrid", etc.). Used as the
	// CLI/HTTP identifier and as the Provenance.SourceFormat written
	// into canonical entities decoded by this adapter.
	Name() string

	// Description returns a short human-readable description for the
	// list-formats CLI surface.
	Description() string

	// Decode parses input bytes in this adapter's wire format into a
	// canonical Entity. Errors are wrapped with adapter context. The
	// returned Entity must satisfy canonical.Entity.Validate() — adapters
	// should call Validate before returning.
	Decode(input []byte) (canonical.Entity, error)

	// Encode serializes a canonical Entity into this adapter's wire
	// format. Errors are wrapped with adapter context.
	Encode(entity canonical.Entity) ([]byte, error)

	// Detect returns true if input bytes are likely in this adapter's
	// format. Used by the detect subcommand. Detection is best-effort;
	// a true return is not a guarantee that Decode will succeed.
	Detect(input []byte) bool
}

// Errors returned by the registry.
var (
	ErrUnknownFormat   = errors.New("adapter: unknown format")
	ErrAlreadyRegistered = errors.New("adapter: format already registered")
)

// registry holds the global adapter registry. Adapters register at package
// init time via Register; the registry is read-only after init.
type registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

var defaultRegistry = &registry{adapters: map[string]Adapter{}}

// Register adds an adapter to the default registry. Typically called from
// an adapter package's init() function. Panics if the format name is
// already registered — duplicate registration is always a programming
// error.
func Register(a Adapter) {
	if a == nil {
		panic("adapter.Register: nil adapter")
	}
	name := a.Name()
	if name == "" {
		panic("adapter.Register: empty format name")
	}
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	if _, exists := defaultRegistry.adapters[name]; exists {
		panic(fmt.Sprintf("adapter.Register: format %q already registered", name))
	}
	defaultRegistry.adapters[name] = a
}

// Get returns the adapter registered for the given format name, or
// ErrUnknownFormat if no adapter is registered.
func Get(name string) (Adapter, error) {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()
	a, ok := defaultRegistry.adapters[name]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownFormat, name)
	}
	return a, nil
}

// List returns the names of every registered adapter, sorted
// alphabetically. Used by the list-formats CLI surface.
func List() []string {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()
	names := make([]string, 0, len(defaultRegistry.adapters))
	for n := range defaultRegistry.adapters {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// All returns every registered adapter, in the same order as List.
func All() []Adapter {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()
	all := make([]Adapter, 0, len(defaultRegistry.adapters))
	for _, n := range List() {
		all = append(all, defaultRegistry.adapters[n])
	}
	return all
}

// ResetForTest clears the registry. Exported solely so adapter and
// pipeline package tests can isolate state between runs. Production code
// must not call this — register at init time and never reset.
func ResetForTest() {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.adapters = map[string]Adapter{}
}
