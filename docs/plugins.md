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

These version 2 sections are catalog foundations only in this release. Whisk
does not dispatch plugin events, invoke blocking hooks, run usage resolver
commands, or run workflow gate/action commands yet. Usage resolvers are exposed
as plugin catalog metadata so clients can see which providers a plugin will
support in a later daemon-owned usage read model; they are not executed and do
not create a usage cache in this slice.

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
