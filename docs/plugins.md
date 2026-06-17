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

## Context Resolution

External attachments are stored as `kind=external` with `provider` and `target`. If `includeInContext=true`, project context asks the trusted resolver for fresh content at context time.

Resolver commands receive the existing project attachment resolution request on stdin and return resolved context JSON on stdout. If no trusted resolver exists, the attachment row still renders, but context marks the item as skipped.
