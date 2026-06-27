// Command gen-go-sdk emits the public Go SDK wrapper for whiskd.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"text/template"

	"github.com/phin-tech/whisk/internal/protocol"
)

type packageExport struct {
	ImportAlias string
	ImportPath  string
	Types       []string
	Consts      []string
}

type sdkTemplateData struct {
	Packages []packageExport
}

func main() {
	out := flag.String("o", "sdk/go/whiskd/client.go", "output file")
	flag.Parse()

	packages, err := sdkExports()
	if err != nil {
		fmt.Fprintln(os.Stderr, "collect:", err)
		os.Exit(1)
	}
	data, err := renderSDK(sdkTemplateData{Packages: packages})
	if err != nil {
		fmt.Fprintln(os.Stderr, "render:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", *out)
}

func sdkExports() ([]packageExport, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	specs := []struct {
		dir         string
		importAlias string
		importPath  string
	}{
		{dir: "internal/protocol", importAlias: "protocol", importPath: "github.com/phin-tech/whisk/internal/protocol"},
		{dir: "internal/domain/session", importAlias: "session", importPath: "github.com/phin-tech/whisk/internal/domain/session"},
		{dir: "internal/domain/workitem", importAlias: "workitem", importPath: "github.com/phin-tech/whisk/internal/domain/workitem"},
	}
	reachable := reachableAPITypes()
	seenTypes := map[string]bool{}
	seenConsts := map[string]bool{}
	var out []packageExport
	for _, spec := range specs {
		types, consts, err := collectPackageExports(filepath.Join(root, spec.dir))
		if err != nil {
			return nil, err
		}
		pkg := packageExport{ImportAlias: spec.importAlias, ImportPath: spec.importPath}
		for _, name := range types {
			if spec.importAlias != "protocol" && !reachable[spec.importPath][name] {
				continue
			}
			if seenTypes[name] {
				continue
			}
			seenTypes[name] = true
			pkg.Types = append(pkg.Types, name)
		}
		for _, name := range consts {
			if seenConsts[name] {
				continue
			}
			seenConsts[name] = true
			pkg.Consts = append(pkg.Consts, name)
		}
		if len(pkg.Types) > 0 || len(pkg.Consts) > 0 {
			out = append(out, pkg)
		}
	}
	return out, nil
}

func repoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot locate generator source")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", fmt.Errorf("go.mod not found above %s", file)
		}
		dir = next
	}
}

func reachableAPITypes() map[string]map[string]bool {
	out := map[string]map[string]bool{}
	seen := map[reflect.Type]bool{}
	for _, route := range protocol.APIRoutes {
		collectReachableType(reflect.TypeOf(route.Request), out, seen)
		collectReachableType(reflect.TypeOf(route.Response), out, seen)
	}
	collectReachableType(reflect.TypeOf(protocol.ErrorResponse{}), out, seen)
	return out
}

func collectReachableType(t reflect.Type, out map[string]map[string]bool, seen map[reflect.Type]bool) {
	if t == nil {
		return
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if seen[t] {
		return
	}
	seen[t] = true
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		collectReachableType(t.Elem(), out, seen)
		return
	case reflect.Map:
		collectReachableType(t.Key(), out, seen)
		collectReachableType(t.Elem(), out, seen)
		return
	}
	if t.PkgPath() != "" && ast.IsExported(t.Name()) {
		if out[t.PkgPath()] == nil {
			out[t.PkgPath()] = map[string]bool{}
		}
		out[t.PkgPath()][t.Name()] = true
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.IsExported() {
			collectReachableType(field.Type, out, seen)
		}
	}
}

func collectPackageExports(dir string) ([]string, []string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	typeSet := map[string]bool{}
	constSet := map[string]bool{}
	fset := token.NewFileSet()
	for _, file := range files {
		name := file.Name()
		if file.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			return nil, nil, err
		}
		for _, decl := range parsed.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			switch gen.Tok {
			case token.TYPE:
				for _, spec := range gen.Specs {
					typeSpec := spec.(*ast.TypeSpec)
					if typeSpec.Name.IsExported() {
						typeSet[typeSpec.Name.Name] = true
					}
				}
			case token.CONST:
				for _, spec := range gen.Specs {
					valueSpec := spec.(*ast.ValueSpec)
					for _, name := range valueSpec.Names {
						if name.IsExported() {
							constSet[name.Name] = true
						}
					}
				}
			}
		}
	}
	types := sortedKeys(typeSet)
	consts := sortedKeys(constSet)
	return types, consts, nil
}

func sortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for name := range values {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func renderSDK(data sdkTemplateData) ([]byte, error) {
	var buf bytes.Buffer
	if err := sdkTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("%w\n%s", err, buf.String())
	}
	return formatted, nil
}

var sdkTemplate = template.Must(template.New("sdk").Parse(`// Code generated by cmd/gen-go-sdk; DO NOT EDIT.

// Package whiskd is the public Go SDK for the whiskd HTTP API.
package whiskd

import (
	"net/http"

	daemonclient "github.com/phin-tech/whisk/internal/client"
{{- range .Packages }}
	{{ .ImportAlias }} "{{ .ImportPath }}"
{{- end }}
)

// Client is a typed HTTP client for a running whiskd daemon.
type Client = daemonclient.HTTPClient

// New returns a client that talks to baseURL, for example http://127.0.0.1:8787.
func New(baseURL string) *Client {
	return daemonclient.NewHTTP(baseURL, nil)
}

// NewWithHTTPClient returns a client using the provided HTTP client.
func NewWithHTTPClient(baseURL string, httpClient *http.Client) *Client {
	return daemonclient.NewHTTP(baseURL, httpClient)
}

{{ range .Packages -}}
{{- $alias := .ImportAlias }}
{{- range .Types }}
type {{ . }} = {{ $alias }}.{{ . }}
{{- end }}
{{ end -}}

const (
{{- range .Packages }}
{{- $alias := .ImportAlias }}
{{- range .Consts }}
	{{ . }} = {{ $alias }}.{{ . }}
{{- end }}
{{- end }}
)
`))
