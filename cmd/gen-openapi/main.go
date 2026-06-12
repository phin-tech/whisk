// Command gen-openapi emits an OpenAPI 3.0.3 spec for the whiskd daemon by
// reflecting over the protocol/domain structs and walking the protocol-owned
// route catalog. The spec is the single source of truth for the Python and
// headless-TypeScript clients; regenerate with `go run ./cmd/gen-openapi`.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/phin-tech/whisk/internal/protocol"
)

func main() {
	out := flag.String("o", "", "output file (default stdout)")
	flag.Parse()

	doc := build()
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal:", err)
		os.Exit(1)
	}
	data = append(data, '\n')

	if *out == "" {
		os.Stdout.Write(data)
		return
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d schemas, %d paths)\n", *out, len(doc.Components.Schemas), len(doc.Paths))
}

// --- OpenAPI document model (minimal subset, marshaled to JSON) ---

type openAPI struct {
	OpenAPI    string              `json:"openapi"`
	Info       info                `json:"info"`
	Servers    []serverURL         `json:"servers"`
	Paths      map[string]pathItem `json:"paths"`
	Components components          `json:"components"`
}

type info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

type serverURL struct {
	URL string `json:"url"`
}

type components struct {
	Schemas map[string]any `json:"schemas"`
}

// pathItem is keyed by lowercase HTTP method -> operation.
type pathItem map[string]operation

type operation struct {
	OperationID string         `json:"operationId"`
	Summary     string         `json:"summary,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Parameters  []parameter    `json:"parameters,omitempty"`
	RequestBody *requestBody   `json:"requestBody,omitempty"`
	Responses   map[string]any `json:"responses"`
}

type parameter struct {
	Name     string         `json:"name"`
	In       string         `json:"in"`
	Required bool           `json:"required"`
	Schema   map[string]any `json:"schema"`
}

type requestBody struct {
	Required bool                 `json:"required"`
	Content  map[string]mediaType `json:"content"`
}

type mediaType struct {
	Schema map[string]any `json:"schema"`
}

// --- assembly ---

func build() *openAPI {
	reg := newRegistry()

	paths := map[string]pathItem{}
	for _, rt := range protocol.APIRoutes {
		item, ok := paths[rt.Path]
		if !ok {
			item = pathItem{}
			paths[rt.Path] = item
		}
		item[strings.ToLower(rt.Method)] = operationForRoute(rt, reg)
	}

	// Always expose the uniform error envelope.
	reg.register(reflect.TypeOf(protocol.ErrorResponse{}))

	return &openAPI{
		OpenAPI: "3.0.3",
		Info: info{
			Title:       "whiskd daemon API",
			Version:     fmt.Sprintf("%d", protocol.DaemonAPIVersion),
			Description: "HTTP/JSON API exposed by the whiskd daemon on loopback. Generated from Go structs; do not edit by hand.",
		},
		Servers:    []serverURL{{URL: "http://127.0.0.1:8787"}},
		Paths:      paths,
		Components: components{Schemas: reg.schemas},
	}
}

func operationForRoute(rt protocol.APIRoute, reg *registry) operation {
	op := operation{
		OperationID: rt.OperationID,
		Summary:     rt.Summary,
		Tags:        []string{rt.Tag},
		Responses:   map[string]any{},
	}

	for _, name := range pathParams(rt.Path) {
		op.Parameters = append(op.Parameters, parameter{
			Name: name, In: "path", Required: true,
			Schema: map[string]any{"type": "string"},
		})
	}
	for _, q := range rt.Query {
		op.Parameters = append(op.Parameters, parameter{
			Name: q.Name, In: "query", Required: q.Required,
			Schema: map[string]any{"type": q.Type},
		})
	}

	if rt.Request != nil {
		op.RequestBody = &requestBody{
			Required: true,
			Content: map[string]mediaType{
				"application/json": {Schema: reg.schemaFor(reflect.TypeOf(rt.Request))},
			},
		}
	}

	status := rt.Status
	if status == 0 {
		status = 200
	}
	if rt.Response == nil {
		op.Responses[fmt.Sprintf("%d", status)] = map[string]any{"description": "no content"}
	} else {
		op.Responses[fmt.Sprintf("%d", status)] = map[string]any{
			"description": "success",
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": reg.schemaFor(reflect.TypeOf(rt.Response)),
				},
			},
		}
	}
	op.Responses["default"] = map[string]any{
		"description": "error",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"$ref": "#/components/schemas/ErrorResponse"},
			},
		},
	}
	return op
}

func pathParams(path string) []string {
	var out []string
	for _, seg := range strings.Split(path, "/") {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			out = append(out, seg[1:len(seg)-1])
		}
	}
	return out
}

// --- reflection -> JSON Schema (OpenAPI 3.0.3 dialect) ---

type registry struct {
	schemas map[string]any
}

func newRegistry() *registry {
	return &registry{schemas: map[string]any{}}
}

func (r *registry) register(t reflect.Type) string {
	name := t.Name()
	if _, done := r.schemas[name]; done {
		return name
	}
	// Reserve the name first so recursive types (e.g. LayoutNode) resolve to a $ref.
	r.schemas[name] = map[string]any{}
	r.schemas[name] = r.structSchema(t)
	return name
}

func (r *registry) structSchema(t reflect.Type) map[string]any {
	props := map[string]any{}
	var required []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, opts := jsonName(f)
		if name == "-" {
			continue
		}
		prop := r.schemaFor(f.Type)
		// A nil slice/map without omitempty marshals to JSON `null` (not `[]`/`{}`),
		// so the field is present-but-nullable. With omitempty a nil collection is
		// omitted entirely, so it stays non-nullable and simply isn't required.
		if !opts.omitempty && isNullableCollection(f.Type) {
			prop["nullable"] = true
		}
		props[name] = prop
		if !opts.omitempty && f.Type.Kind() != reflect.Pointer {
			required = append(required, name)
		}
	}
	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func (r *registry) schemaFor(t reflect.Type) map[string]any {
	// time.Time is a struct but serializes as an RFC3339 string.
	if t == reflect.TypeOf(time.Time{}) {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	switch t.Kind() {
	case reflect.Pointer:
		s := r.schemaFor(t.Elem())
		// In OpenAPI 3.0.3 a `nullable` sibling to `$ref` is ignored, so wrap
		// referenced types in allOf to make the null actually take effect.
		if _, isRef := s["$ref"]; isRef {
			return map[string]any{"allOf": []any{s}, "nullable": true}
		}
		s["nullable"] = true
		return s
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		s := map[string]any{"type": "integer"}
		if t.Bits() == 64 {
			s["format"] = "int64"
		}
		return s
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 { // []byte -> base64 string
			return map[string]any{"type": "string", "format": "byte"}
		}
		return map[string]any{"type": "array", "items": r.schemaFor(t.Elem())}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": r.schemaFor(t.Elem()),
		}
	case reflect.Struct:
		name := r.register(t)
		return map[string]any{"$ref": "#/components/schemas/" + name}
	default:
		// interface{}/any and friends: unconstrained.
		return map[string]any{}
	}
}

// isNullableCollection reports whether a nil value of t marshals to JSON null —
// true for slices (except []byte, which becomes a base64 string) and maps.
func isNullableCollection(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Map:
		return true
	case reflect.Slice:
		return t.Elem().Kind() != reflect.Uint8
	default:
		return false
	}
}

type jsonOpts struct{ omitempty bool }

func jsonName(f reflect.StructField) (string, jsonOpts) {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name, jsonOpts{}
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		name = f.Name
	}
	var opts jsonOpts
	for _, p := range parts[1:] {
		if p == "omitempty" {
			opts.omitempty = true
		}
	}
	return name, opts
}
