package protocol

import "testing"

func TestCompatibilityRouteUsesProtocolMetadataResponse(t *testing.T) {
	route, ok := routeByOperationID("getCompatibility")
	if !ok {
		t.Fatalf("getCompatibility route missing")
	}
	if route.Method != "GET" || route.Path != "/v1/compat" || route.Tag != "system" {
		t.Fatalf("route = %#v", route)
	}
	if route.Summary != "Daemon protocol compatibility and build metadata" {
		t.Fatalf("summary = %q", route.Summary)
	}
	if _, ok := route.Response.(CompatibilityResponse); !ok {
		t.Fatalf("getCompatibility response = %T, want CompatibilityResponse", route.Response)
	}
}

func TestNextEventRouteIncludesCursorAndResponseEnvelope(t *testing.T) {
	route, ok := routeByOperationID("nextEvent")
	if !ok {
		t.Fatalf("nextEvent route missing")
	}
	if _, ok := route.Response.(NextEventResponse); !ok {
		t.Fatalf("nextEvent response = %T, want NextEventResponse", route.Response)
	}
	if !hasQueryParam(route, "timeoutMs", "integer") {
		t.Fatalf("nextEvent route missing timeoutMs query: %#v", route.Query)
	}
	if !hasQueryParam(route, "afterSeq", "integer") {
		t.Fatalf("nextEvent route missing afterSeq query: %#v", route.Query)
	}
}

func TestDetectedAgentsRouteUsesAgentsTagAndReadModel(t *testing.T) {
	route, ok := routeByOperationID("listDetectedAgents")
	if !ok {
		t.Fatalf("listDetectedAgents route missing")
	}
	if route.Method != "GET" || route.Path != "/v1/agents/detected" || route.Tag != "agents" {
		t.Fatalf("route = %#v", route)
	}
	if _, ok := route.Response.([]DetectedAgent); !ok {
		t.Fatalf("listDetectedAgents response = %T, want []DetectedAgent", route.Response)
	}
}

func TestListUIContributionsRouteIncludesEntityScopeQuery(t *testing.T) {
	route, ok := routeByOperationID("listUIContributions")
	if !ok {
		t.Fatalf("listUIContributions route missing")
	}
	if route.Method != "GET" || route.Path != "/v1/ui-contributions" || route.Tag != "plugins" {
		t.Fatalf("route = %#v", route)
	}
	if _, ok := route.Response.(UIContributionsResponse); !ok {
		t.Fatalf("listUIContributions response = %T, want UIContributionsResponse", route.Response)
	}
	for _, name := range []string{"projectId", "workItemId", "runId", "sessionId", "paneId", "ptyId", "gateReportId", "phase"} {
		if !hasQueryParam(route, name, "string") {
			t.Fatalf("listUIContributions route missing %s query: %#v", name, route.Query)
		}
	}
}

func TestUsageResolverRoutesUsePluginTagAndReadModels(t *testing.T) {
	list, ok := routeByOperationID("listUsageResolvers")
	if !ok {
		t.Fatalf("listUsageResolvers route missing")
	}
	if list.Method != "GET" || list.Path != "/v1/usage-resolvers" || list.Tag != "plugins" {
		t.Fatalf("list route = %#v", list)
	}
	if _, ok := list.Response.([]UsageResolverReadModel); !ok {
		t.Fatalf("listUsageResolvers response = %T, want []UsageResolverReadModel", list.Response)
	}

	refresh, ok := routeByOperationID("refreshUsageResolver")
	if !ok {
		t.Fatalf("refreshUsageResolver route missing")
	}
	if refresh.Method != "POST" || refresh.Path != "/v1/plugins/{pluginID}/usage-resolvers/{resolverID}/refresh" || refresh.Tag != "plugins" {
		t.Fatalf("refresh route = %#v", refresh)
	}
	if _, ok := refresh.Request.(RefreshUsageResolverRequest); !ok {
		t.Fatalf("refreshUsageResolver request = %T, want RefreshUsageResolverRequest", refresh.Request)
	}
	if _, ok := refresh.Response.(UsageResolverReadModel); !ok {
		t.Fatalf("refreshUsageResolver response = %T, want UsageResolverReadModel", refresh.Response)
	}
}

func TestBrowserResourceRoutesUseBrowserTagAndReadModels(t *testing.T) {
	list, ok := routeByOperationID("listBrowserResources")
	if !ok {
		t.Fatalf("listBrowserResources route missing")
	}
	if list.Method != "GET" || list.Path != "/v1/browser-resources" || list.Tag != "browser" {
		t.Fatalf("list route = %#v", list)
	}
	if _, ok := list.Response.([]BrowserResource); !ok {
		t.Fatalf("listBrowserResources response = %T, want []BrowserResource", list.Response)
	}

	connect, ok := routeByOperationID("connectBrowserResource")
	if !ok {
		t.Fatalf("connectBrowserResource route missing")
	}
	if connect.Method != "POST" || connect.Path != "/v1/browser-resources" || connect.Status != 201 || connect.Tag != "browser" {
		t.Fatalf("connect route = %#v", connect)
	}
	if _, ok := connect.Request.(ConnectBrowserResourceRequest); !ok {
		t.Fatalf("connectBrowserResource request = %T, want ConnectBrowserResourceRequest", connect.Request)
	}
	if _, ok := connect.Response.(BrowserResource); !ok {
		t.Fatalf("connectBrowserResource response = %T, want BrowserResource", connect.Response)
	}

	targets, ok := routeByOperationID("listBrowserTargets")
	if !ok {
		t.Fatalf("listBrowserTargets route missing")
	}
	if targets.Method != "GET" || targets.Path != "/v1/browser-resources/{resourceID}/targets" || targets.Tag != "browser" {
		t.Fatalf("targets route = %#v", targets)
	}
	if _, ok := targets.Response.([]BrowserTarget); !ok {
		t.Fatalf("listBrowserTargets response = %T, want []BrowserTarget", targets.Response)
	}

	disconnect, ok := routeByOperationID("disconnectBrowserResource")
	if !ok {
		t.Fatalf("disconnectBrowserResource route missing")
	}
	if disconnect.Method != "DELETE" || disconnect.Path != "/v1/browser-resources/{resourceID}" || disconnect.Status != 204 || disconnect.Tag != "browser" {
		t.Fatalf("disconnect route = %#v", disconnect)
	}
}

func TestAPIRoutesDoNotExposePTYBookmarks(t *testing.T) {
	for _, route := range APIRoutes {
		if route.OperationID == "addPTYBookmark" || route.OperationID == "listPTYBookmarks" || route.OperationID == "removePTYBookmark" {
			t.Fatalf("bookmark route still registered: %#v", route)
		}
		if route.Path == "/v1/ptys/{ptyID}/bookmarks" || route.Path == "/v1/pty-bookmarks/{bookmarkID}" {
			t.Fatalf("bookmark path still registered: %#v", route)
		}
	}
}

func routeByOperationID(operationID string) (APIRoute, bool) {
	for _, route := range APIRoutes {
		if route.OperationID == operationID {
			return route, true
		}
	}
	return APIRoute{}, false
}

func hasQueryParam(route APIRoute, name string, typ string) bool {
	for _, param := range route.Query {
		if param.Name == name && param.Type == typ {
			return true
		}
	}
	return false
}
