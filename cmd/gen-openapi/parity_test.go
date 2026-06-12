package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/phin-tech/whisk/internal/adapters/pty/native"
	"github.com/phin-tech/whisk/internal/app"
	"github.com/phin-tech/whisk/internal/protocol"
	"github.com/phin-tech/whisk/internal/server"
)

// serverRoutePattern matches the `mux.HandleFunc("METHOD /path", ...)` lines in
// internal/server/http.go so the test can compare the real router against the
// generator's route table. Method is optional (the proxy catch-alls omit it).
var serverRoutePattern = regexp.MustCompile(`mux\.HandleFunc\("(?:([A-Z]+) )?(/v1/[^"]*)"`)

// routesNotInSDK are server endpoints intentionally excluded from the generated
// clients: liveness and the raw reverse-proxy passthroughs.
var routesNotInSDK = map[string]bool{
	"GET /v1/health":                        true,
	" /v1/http-forwards/{forwardID}/proxy":  true,
	" /v1/http-forwards/{forwardID}/proxy/": true,
}

// TestEverySDKRouteIsRegistered asserts that every route in the generator table
// resolves to a registered handler on the real server router. This is the
// contract guard: the route catalog is hand-maintained, so this catches a spec
// route that the server never serves. It matches on the router pattern (via
// ServeMux.Handler) rather than status codes, so handlers that legitimately
// return 404 (e.g. delete of a missing resource) don't produce false failures.
func TestEverySDKRouteIsRegistered(t *testing.T) {
	runtime := app.NewRuntime(app.RuntimeConfig{PTYBackend: native.NewBackend()})
	mux, ok := server.NewHTTP(runtime).(*http.ServeMux)
	if !ok {
		t.Fatal("server.NewHTTP no longer returns *http.ServeMux; update this test")
	}

	for _, rt := range protocol.APIRoutes {
		t.Run(rt.Method+" "+rt.Path, func(t *testing.T) {
			path := pathParamRe.ReplaceAllString(rt.Path, "x")
			req := httptest.NewRequest(rt.Method, path, nil)
			if _, pattern := mux.Handler(req); pattern == "" {
				t.Fatalf("route %s %s is not registered on the server", rt.Method, path)
			}
		})
	}
}

// TestEveryServerRouteIsInSDK is the reverse guard: scan the server's router and
// assert every /v1 endpoint (minus the documented exclusions) has a route-table
// entry, so a newly added handler can't silently miss the generated clients.
func TestEveryServerRouteIsInSDK(t *testing.T) {
	src, err := os.ReadFile("../../internal/server/http.go")
	if err != nil {
		t.Fatalf("read server source: %v", err)
	}

	inTable := map[string]bool{}
	for _, rt := range protocol.APIRoutes {
		inTable[rt.Method+" "+rt.Path] = true
	}

	for _, m := range serverRoutePattern.FindAllStringSubmatch(string(src), -1) {
		method, path := m[1], m[2]
		key := method + " " + path
		if routesNotInSDK[key] {
			continue
		}
		if !inTable[key] {
			t.Errorf("server route %q has no entry in protocol.APIRoutes", strings.TrimSpace(key))
		}
	}
}

var pathParamRe = regexp.MustCompile(`\{[^}]+\}`)
