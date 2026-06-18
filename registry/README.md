# Whisk plugin registry

This directory is the seed for the **phin-tech/whisk-plugins** registry: the
catalog Whisk reads to discover and install plugins.

## Layout

```
registry.json            # the catalog: one entry per installable plugin
plugins/<id>/            # monorepo-hosted plugins ("path" source)
  plugin.json            # the plugin manifest (its id must match the catalog id)
```

## Catalog format (`registry.json`)

```jsonc
{
  "version": 1,
  "plugins": [
    {
      "id": "github",                 // must equal the plugin's plugin.json id
      "name": "GitHub Issues",
      "description": "…",
      "source": { "type": "path", "path": "plugins/github-issues" }
    }
  ]
}
```

### Source types

- **`path`** — a plugin that lives in this repo (monorepo style). `path` is
  relative to the repo root. Whisk fetches it from this repo's tarball.
- **`git`** — a plugin that lives in its own repository:

  ```jsonc
  { "type": "git", "repo": "phin-tech/whisk-plugin-linear", "subdir": "plugin", "ref": "v0.2.0" }
  ```

  `repo` accepts `owner/repo` or a GitHub URL; `subdir` and `ref` are optional.

## How install works

Whisk downloads a plugin's files via the GitHub **tarball** (no `git clone`),
verifies the bundle carries a matching `plugin.json`, records a SHA-256
fingerprint in `plugins.lock.json`, and writes the files into
`~/.config/whisk/plugins/<id>/`. Installed plugins start **untrusted**; the user
must trust a plugin before its commands run.

Point Whisk at a different registry with `WHISK_PLUGIN_REGISTRY` (a GitHub
`owner/repo[@ref]` or a local directory for development).
