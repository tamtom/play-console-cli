package output

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
)

// TableRenderer can render a specific type as table rows.
type TableRenderer struct {
	Headers  []string
	RenderFn func(data any) [][]string // returns rows
}

var (
	registry   = make(map[reflect.Type]*TableRenderer)
	registryMu sync.RWMutex
)

// RegisterType registers a table renderer for a specific type.
// The renderFn receives data of the registered type and returns rows.
func RegisterType(exemplar any, headers []string, renderFn func(data any) [][]string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	t := reflect.TypeOf(exemplar)
	registry[t] = &TableRenderer{Headers: headers, RenderFn: renderFn}
}

// RenderRegistered tries to render data using the type registry.
// Returns true if the type was found in the registry and rendered.
func RenderRegistered(w io.Writer, data any, format string) (bool, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	t := reflect.TypeOf(data)
	renderer, ok := registry[t]
	if !ok {
		return false, nil
	}

	rows := renderer.RenderFn(data)

	switch format {
	case "table":
		RenderTableTo(w, renderer.Headers, rows)
		return true, nil
	case "markdown", "md":
		return true, RenderMarkdownTable(w, renderer.Headers, rows)
	default:
		return false, fmt.Errorf("unsupported format for registry: %s", format)
	}
}

// RenderRegisteredToStdout is a convenience wrapper that renders to os.Stdout.
func RenderRegisteredToStdout(data any, format string) (bool, error) {
	return RenderRegistered(os.Stdout, data, format)
}
