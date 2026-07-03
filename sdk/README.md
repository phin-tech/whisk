# whiskd SDKs

Generated clients for the whiskd daemon's HTTP/JSON API. **One OpenAPI spec is
the single source of truth**, derived from the Go protocol/domain structs and
codegen'd into Python, headless TypeScript, and Go clients.

```
internal/protocol/*.go ─┐
internal/domain/**/*.go ├─► cmd/gen-openapi ─► sdk/openapi.json ─┬─► sdk/python (pydantic + httpx)
route table (routes.go) ┘                                       ├─► sdk/ts/whiskd.d.ts (+ openapi-fetch)
                         └─► cmd/gen-go-sdk ────────────────────└─► sdk/go/whiskd
```

The Wails desktop frontend is **not** a consumer here — it keeps its own
`wails3 generate bindings` path. These SDKs are for out-of-Wails consumers
(Python tools, `npx`/`bunx` scripts, integration tests) that hit the daemon
over HTTP directly.

## Regenerating

```sh
task sdk          # spec + Python + TypeScript + Go
task sdk:spec     # just sdk/openapi.json
task sdk:python   # spec + Python client
task sdk:ts       # spec + TypeScript types
task sdk:go       # Go client wrapper
task sdk:check    # CI guard: fails if openapi.json is stale vs the Go structs
```

Generated client outputs under `sdk/python`, `sdk/ts`, and `sdk/go` should not
be hand-edited — change the Go structs (or `cmd/gen-openapi/routes.go` for new
endpoints) and regenerate.

### Adding an endpoint

1. Add the handler + protocol struct in `internal/server` / `internal/protocol`.
2. Add one line to `routes` in `cmd/gen-openapi/routes.go`.
3. `task sdk` and commit.

## Using the clients

**Python** (`sdk/python/whiskd_client`, requires `httpx`, `attrs`, `python-dateutil`):

```python
from whiskd_client.api.workitems import list_work_items
from whiskd_client.local import local_client

client = local_client(base_url="http://127.0.0.1:8787")
items = list_work_items.sync(client=client, project_id="proj-1")
```

**TypeScript** (`sdk/ts/whiskd.d.ts`, types only — pair with `openapi-fetch`):

```ts
import createClient from "openapi-fetch";
import { whiskdClientOptions } from "./client";
import type { paths } from "./whiskd";

const d = createClient<paths>(whiskdClientOptions({ baseUrl: "http://127.0.0.1:8787" }));
const { data } = await d.GET("/v1/work-items", { params: { query: { projectId } } });
```

**Go** (`sdk/go/whiskd`, typed client + DTO aliases):

```go
package main

import (
	"context"
	"log"

	whiskd "github.com/phin-tech/whisk/sdk/go/whiskd"
)

func main() {
	client := whiskd.New("http://127.0.0.1:8787")
	items, err := client.ListWorkItems(context.Background(), "")
	if err != nil {
		log.Fatal(err)
	}
	_ = items
}
```

## Tests

Two layers, both run in CI:

```sh
task sdk:check        # fast, no daemon: route-parity guard + spec drift diff
task sdk:test         # live: boots a real whiskd, drives both clients end-to-end
task sdk:test:python  # just the Python suite (pytest, via uv)
task sdk:test:ts      # just the TypeScript suite (vitest + openapi-fetch)
task sdk:test:go      # just the Go suite
```

- **`sdk:check`** — Go-level guard. `cmd/gen-openapi/parity_test.go` asserts the
  route table and the real server router match in both directions, then the
  regenerated spec is diffed against the committed one. Catches drift without
  booting anything.
- **`sdk:test`** — boots `whiskd` on an ephemeral loopback port with an isolated
  XDG state dir (never touches real session state) and runs the generated Python,
  TS, and Go clients through a compat handshake + work-item round-trip. This is what
  verifies the spec matches real wire behavior — status codes, camelCase mapping,
  RFC3339 time parsing, query params, local control-token auth, and Go's
  `nil slice -> null` encoding. The suites skip themselves if `WHISKD_BIN` is
  unset (the tasks build it first).

## Notes / known wrinkles

- **Streaming is polling.** Live output/events use `GET /v1/ptys/{id}/output?from=`
  and `GET /v1/events/next?timeoutMs=` — drive them in a loop, tracking the
  returned offset. There is no websocket.
- **Local auth.** The daemon protects control routes with a per-user bearer token
  at `$XDG_STATE_HOME/whisk/control-token` or `~/.local/state/whisk/control-token`.
  Same-user Go, Python, TypeScript, CLI, and desktop clients read it automatically.
- `uint64` offsets are emitted as `integer/int64`.
- Empty-body endpoints (writes, resizes, deletes) return `204`.
