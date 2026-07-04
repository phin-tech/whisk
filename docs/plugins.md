# Plugins

Whisk plugins are daemon-loaded command plugins. The desktop never runs plugin code and plugins do not ship Svelte or browser JavaScript.

## Discovery

`whiskd` discovers plugin directories from:

- `WHISK_PLUGIN_DIRS`, split with the platform path-list separator
- `$XDG_CONFIG_HOME/whisk/plugins/*`
- `~/.config/whisk/plugins/*` when `XDG_CONFIG_HOME` is unset

Each plugin directory must contain `plugin.json`.

## Trust

Plugins are listed before they are trusted. Untrusted plugins are not executed and their resolvers/templates are not active.

CLI:

```sh
whisk plugin list
whisk plugin list -json
whisk plugin contributions -work-item wi_01 -phase review -json
whisk plugin rescan
whisk plugin trust github
whisk plugin untrust github
```

Trust changes are applied live inside `whiskd`; the daemon is not restarted and active PTYs are preserved.

## Manifest

`manifestVersion` is optional and defaults to `1`. Version 1 keeps the original
permissive manifest behavior for existing plugins. Version 2 is stricter: Whisk
rejects unsupported future versions and unknown top-level fields so new
contribution kinds do not get silently ignored.

```json
{
  "id": "github",
  "name": "GitHub Issues",
  "version": "0.1.0",
  "resolvers": [
    {
      "provider": "github",
      "kinds": ["external"],
      "command": "node ./resolve.mjs"
    }
  ],
  "ui": {
    "projectAttachments": [
      {
        "id": "github.issue.attach",
        "label": "GitHub Issue",
        "provider": "github",
        "kind": "external",
        "command": "node ./attach-issue.mjs",
        "fields": [
          {
            "id": "url",
            "label": "Issue URL",
            "type": "text",
            "placeholder": "https://github.com/owner/repo/issues/123",
            "required": true
          }
        ]
      }
    ]
  }
}
```

Manifest version 2 also parses daemon-owned foundations for future plugin
events, hooks, usage resolvers, workflow gates/actions, and permission
disclosures:

```json
{
  "manifestVersion": 2,
  "id": "linear",
  "name": "Linear",
  "version": "0.2.0",
  "events": [
    {
      "id": "linear.sync-work",
      "subjects": ["workitem.stage.changed"],
      "command": "node ./on-event.mjs",
      "timeoutMs": 10000
    }
  ],
  "hooks": [
    {
      "id": "linear.approval-policy",
      "point": "approval.evaluate",
      "command": "node ./policy.mjs",
      "timeoutMs": 3000
    }
  ],
  "usageResolvers": [
    {
      "id": "linear.usage",
      "provider": "linear",
      "label": "Linear",
      "profiles": ["linear-agent"],
      "command": "node ./usage.mjs",
      "timeoutMs": 10000,
      "outputCapBytes": 262144,
      "minRefreshMs": 300000,
      "staleAfterMs": 1800000
    }
  ],
  "permissions": {
    "ptyOutput": false,
    "envPrefixes": ["LINEAR_"],
    "network": ["api.linear.app"]
  }
}
```

These version 2 sections are catalog foundations unless a daemon route explicitly
executes them. Whisk does not dispatch plugin events, invoke blocking hooks, or
run workflow gate/action commands yet. Usage resolvers are the first executable
foundation: trusted manifest v2 resolver commands can be refreshed through the
daemon, producing a daemon-owned usage read model keyed by plugin, resolver,
provider, and profile. Untrusted plugins are never executed, and command strings
stay out of public plugin status and usage result metadata.

Usage resolver commands receive JSON on stdin:

```json
{
  "pluginId": "linear",
  "resolverId": "linear.usage",
  "provider": "linear",
  "profile": "linear-agent"
}
```

They return normalized usage/rate-limit JSON on stdout:

```json
{
  "summary": "75% daily API budget remaining",
  "metrics": [
    {
      "id": "api.requests",
      "kind": "rateLimit",
      "label": "API requests",
      "unit": "requests",
      "used": 2500,
      "limit": 10000,
      "remaining": 7500,
      "resetAt": "2026-07-04T20:00:00Z"
    }
  ],
  "meta": {
    "workspace": "acme"
  }
}
```

Supported metric kinds are `usage` and `rateLimit`. Each metric must include an
`id`, a supported `kind`, and at least one of `used`, `limit`, or `remaining`;
numeric values must be finite and non-negative. The daemon validates successful
command output, records command failures as usage result status `error`, and
marks cached results stale when `staleAfterMs` has elapsed.

CLI:

```sh
whisk plugin usage
whisk plugin usage -json
whisk plugin usage refresh linear linear.usage -profile linear-agent
whisk plugin usage refresh linear linear.usage -profile linear-agent -json
```

Manifest version 2 also catalogs plugin UI contributions in the `ui` section.
`ui.panels`, `ui.commands`, and `ui.reviewActions` are returned through
`PluginStatus` for both trusted and untrusted plugins so clients can show what a
plugin would contribute before trust. These fields are catalog data only in this
release: Whisk does not execute panel read commands, palette commands, panel
actions, or review-action submit commands from these declarations yet.

```json
{
  "manifestVersion": 2,
  "id": "linear",
  "name": "Linear",
  "version": "0.2.0",
  "permissions": {
    "network": ["api.linear.app"]
  },
  "ui": {
    "panels": [
      {
        "id": "linear.issue",
        "title": "Linear issue",
        "scope": "workItem",
        "kind": "view",
        "read": { "command": "node ./render-issue.mjs" },
        "actions": [
          {
            "id": "sync",
            "label": "Sync",
            "command": "node ./sync.mjs"
          }
        ]
      },
      {
        "id": "linear.board",
        "title": "Linear board",
        "scope": "project",
        "kind": "html",
        "entry": "./panel/"
      }
    ],
    "commands": [
      {
        "id": "linear.open-triage",
        "label": "Linear: Open triage",
        "scope": "global",
        "command": "node ./triage.mjs"
      }
    ],
    "reviewActions": [
      {
        "id": "linear.review",
        "label": "Linear review",
        "scope": "workItem",
        "urlTemplate": "https://linear.app/acme/issue/{{work_item.id.url}}",
        "submitCommand": "node ./fetch-review.mjs",
        "blocking": true
      }
    ]
  }
}
```

Supported UI scopes are `global`, `project`, `workItem`, `run`, and `gate`.
Panel kinds are `view` for future host-rendered view documents and `html` for a
future daemon-served iframe surface. The catalog summary intentionally omits raw
command strings while retaining IDs, labels, scopes, kind, URL templates,
permissions, and normalized timeout/output-cap metadata for later trust prompts
and UI placement.

Attachment templates are declarative UI hints. Whisk renders the form and sends the values to the daemon. The plugin command receives JSON on stdin:

```json
{
  "pluginId": "github",
  "templateId": "github.issue.attach",
  "projectId": "proj_01",
  "values": {
    "url": "https://github.com/owner/repo/issues/123"
  }
}
```

The command returns attachment JSON on stdout:

```json
{
  "kind": "external",
  "provider": "github",
  "target": "owner/repo#123",
  "url": "https://github.com/owner/repo/issues/123",
  "title": "Issue title",
  "includeInContext": true,
  "meta": {
    "github/type": { "type": "string", "string": "issue" },
    "github/repo": { "type": "string", "string": "owner/repo" },
    "github/number": { "type": "number", "number": 123 }
  }
}
```

Agents can run the same template through the CLI:

```sh
whisk plugin attach github github.issue.attach -project proj_01 -field url=https://github.com/owner/repo/issues/123
```

## Aggregated UI Contributions Read Model

`GET /v1/ui-contributions` returns a scope-filtered aggregation of trusted,
valid plugin UI contributions. The frontend uses this active render model to
determine what panels, commands, and review actions are available for the
currently viewed entity. Use `GET /v1/plugins` for catalog previews of
untrusted or invalid plugins.

Query parameters:

| Parameter      | Value                             |
|----------------|-----------------------------------|
| `projectId`    | Include project-scoped items      |
| `workItemId`   | Include work-item-scoped items    |
| `runId`        | Include run-scoped items          |
| `sessionId`    | Identify the current session      |
| `paneId`       | Identify the current pane         |
| `ptyId`        | Identify the current PTY          |
| `gateReportId` | Include gate-scoped items         |
| `phase`        | Identify the current workflow phase |

Global-scoped contributions are always included. When no entity query
parameters are provided, only global contributions are returned; an empty entity
value such as `workItemId=` is treated the same as an omitted entity. `phase`
is accepted and echoed as contextual metadata, but a phase-only request still
uses the global-only scope until phase-scoped contribution kinds exist. Current
manifest UI contributions match `global`, `project`, `workItem`, `run`, and
`gate`; the broader entity IDs are accepted and echoed so clients can make one
stable call from any surface as later contribution kinds are added.

Response shape:

```json
{
  "scope": { "workItemId": "wi_01" },
  "plugins": [
    {
      "pluginId": "linear",
      "name": "Linear",
      "version": "0.2.0",
      "trusted": true,
      "enabled": true,
      "resolvers": [{"provider": "linear", "kinds": ["external"]}],
      "permissions": {"network": ["api.linear.app"]},
      "panels": [
        {
          "id": "linear.issue",
          "title": "Linear issue",
          "scope": "workItem",
          "kind": "view",
          "read": {"timeoutMs": 10000, "outputCapBytes": 262144},
          "actions": [{"id": "sync", "label": "Sync", "timeoutMs": 10000, "outputCapBytes": 262144}]
        }
      ],
      "commands": [
        {"id": "linear.open-triage", "label": "Linear: Open triage", "scope": "global", "timeoutMs": 10000}
      ],
      "reviewActions": [
        {"id": "linear.review", "label": "Linear review", "scope": "workItem", "urlTemplate": "https://...", "hasSubmit": true, "blocking": true}
      ]
    }
  ]
}
```

Each plugin entry mirrors the UI catalog fields from `PluginStatus` but is
grouped by plugin with the scope applied per-contribution. Only trusted, valid
plugins are included in this active render model, and plugins that have no
matching contributions after filtering are omitted.

Agents can query the same read model through the CLI:

```sh
whisk plugin contributions -work-item wi_01 -phase review -json
```

## Command Execution

Trusted resolver and attachment template commands run inside `whiskd` with the
plugin directory as their working directory. Whisk invokes the command through
the platform shell (`sh -lc` on Unix-like systems, `cmd /c` on Windows), sends a
JSON request on stdin, and reads JSON from stdout.

Commands are bounded to protect the daemon: the default timeout is 10 seconds,
stdout is capped at 1 MiB, and stderr captured for error reporting is capped at
64 KiB. A timeout, non-zero exit, or stdout cap violation fails the command; when
a command exits non-zero, capped stderr is included in the error message.
Malformed stdout is reported by the caller as a JSON parse error.

## Context Resolution

External attachments are stored as `kind=external` with `provider` and `target`. If `includeInContext=true`, project context asks the trusted resolver for fresh content at context time.

Resolver commands receive the existing project attachment resolution request on stdin and return resolved context JSON on stdout. If no trusted resolver exists, the attachment row still renders, but context marks the item as skipped.
