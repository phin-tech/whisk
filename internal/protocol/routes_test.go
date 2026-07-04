package protocol

import "testing"

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
