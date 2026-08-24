// Package apischema exposes the reviewed Google Play discovery surface without
// requiring credentials or a network connection.
package apischema

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

//go:embed schema-index.json
var indexData []byte

// Index is the generated, compact union of the official Play discovery
// documents supported by gplay.
type Index struct {
	SchemaVersion int        `json:"schema_version"`
	APIs          []API      `json:"apis"`
	Endpoints     []Endpoint `json:"endpoints"`
	Types         []Type     `json:"types"`
}

// API describes one embedded official discovery document.
type API struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Revision     string `json:"revision"`
	DiscoveryURL string `json:"discovery_url"`
	BaseURL      string `json:"base_url,omitempty"`
	ServicePath  string `json:"service_path,omitempty"`
}

// Endpoint is the transport and type contract for an official method.
type Endpoint struct {
	API            string       `json:"api"`
	Version        string       `json:"version"`
	ID             string       `json:"id"`
	HTTPMethod     string       `json:"http_method"`
	Path           string       `json:"path"`
	Description    string       `json:"description,omitempty"`
	Parameters     []Parameter  `json:"parameters,omitempty"`
	ParameterOrder []string     `json:"parameter_order,omitempty"`
	RequestType    string       `json:"request_type,omitempty"`
	ResponseType   string       `json:"response_type,omitempty"`
	Scopes         []string     `json:"scopes,omitempty"`
	MediaUpload    *MediaUpload `json:"media_upload,omitempty"`
}

// Parameter describes a path or query parameter accepted by an endpoint.
type Parameter struct {
	Name             string   `json:"name"`
	Location         string   `json:"location"`
	Type             string   `json:"type"`
	Format           string   `json:"format,omitempty"`
	Description      string   `json:"description,omitempty"`
	Required         bool     `json:"required,omitempty"`
	Repeated         bool     `json:"repeated,omitempty"`
	Deprecated       bool     `json:"deprecated,omitempty"`
	Enum             []string `json:"enum,omitempty"`
	EnumDescriptions []string `json:"enum_descriptions,omitempty"`
	Default          string   `json:"default,omitempty"`
	Minimum          string   `json:"minimum,omitempty"`
	Maximum          string   `json:"maximum,omitempty"`
	Pattern          string   `json:"pattern,omitempty"`
}

// MediaUpload describes optional simple and resumable upload transports.
type MediaUpload struct {
	Accept        []string `json:"accept,omitempty"`
	MaxSize       string   `json:"max_size,omitempty"`
	SimplePath    string   `json:"simple_path,omitempty"`
	ResumablePath string   `json:"resumable_path,omitempty"`
}

// Type is an official discovery schema definition. Definition is preserved as
// Google publishes it so agents can inspect nested fields, refs, enum values,
// formats, and constraints without a lossy local translation.
type Type struct {
	API        string          `json:"api"`
	Name       string          `json:"name"`
	Definition json.RawMessage `json:"definition"`
}

// Filter selects endpoints or types in the embedded index.
type Filter struct {
	API    string
	Method string
	Query  string
}

var defaultIndex struct {
	once  sync.Once
	value *Index
	err   error
}

// Load parses the embedded index once on first use.
func Load() (*Index, error) {
	defaultIndex.once.Do(func() {
		var index Index
		if err := json.Unmarshal(indexData, &index); err != nil {
			defaultIndex.err = fmt.Errorf("decode embedded API schema index: %w", err)
			return
		}
		if index.SchemaVersion != 1 {
			defaultIndex.err = fmt.Errorf("unsupported embedded API schema version %d", index.SchemaVersion)
			return
		}
		defaultIndex.value = &index
	})
	return defaultIndex.value, defaultIndex.err
}

// FindEndpoints returns matching endpoints in stable method-ID order.
func (i *Index) FindEndpoints(filter Filter) ([]Endpoint, error) {
	api, method, query, err := i.normalizeFilter(filter)
	if err != nil {
		return nil, err
	}
	results := make([]Endpoint, 0)
	for _, endpoint := range i.Endpoints {
		if api != "" && strings.ToLower(endpoint.API) != api {
			continue
		}
		if method != "" && strings.ToUpper(endpoint.HTTPMethod) != method {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(strings.Join([]string{
			endpoint.API,
			endpoint.ID,
			endpoint.HTTPMethod + " " + endpoint.Path,
			endpoint.RequestType,
			endpoint.ResponseType,
		}, " ")), query) {
			continue
		}
		results = append(results, endpoint)
	}
	sort.Slice(results, func(left, right int) bool { return results[left].ID < results[right].ID })
	return results, nil
}

// FindTypes returns matching discovery types in stable API/name order.
func (i *Index) FindTypes(filter Filter) ([]Type, error) {
	api, _, query, err := i.normalizeFilter(Filter{API: filter.API, Query: filter.Query})
	if err != nil {
		return nil, err
	}
	results := make([]Type, 0)
	for _, item := range i.Types {
		if api != "" && strings.ToLower(item.API) != api {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(item.Name), query) {
			continue
		}
		results = append(results, item)
	}
	sort.Slice(results, func(left, right int) bool {
		if results[left].API != results[right].API {
			return results[left].API < results[right].API
		}
		return results[left].Name < results[right].Name
	})
	return results, nil
}

func (i *Index) normalizeFilter(filter Filter) (string, string, string, error) {
	api := strings.ToLower(strings.TrimSpace(filter.API))
	method := strings.ToUpper(strings.TrimSpace(filter.Method))
	query := strings.ToLower(strings.TrimSpace(filter.Query))
	if api != "" {
		found := false
		for _, item := range i.APIs {
			if strings.ToLower(item.Name) == api {
				found = true
				break
			}
		}
		if !found {
			return "", "", "", fmt.Errorf("unknown API %q", filter.API)
		}
	}
	if method != "" {
		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE":
		default:
			return "", "", "", fmt.Errorf("invalid HTTP method %q", filter.Method)
		}
	}
	return api, method, query, nil
}
