package protocol

import "testing"

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
