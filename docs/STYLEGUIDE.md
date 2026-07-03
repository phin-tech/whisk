# Whisk UI Style Guide

This is the UI style guide for Whisk: color token roles, token resolution, and
component usage rules. It is not an architecture document; daemon/runtime
ownership rules live in `AGENTS.md` and the Go packages.

Whisk is a daemon-owned agent workspace. The UI should feel quiet and
work-focused: neutral chrome frames terminals, project state, work items, and
agent activity without competing with them. Color is reserved for selection,
attention, status, and destructive actions.

When in doubt:

- Reach for a role token before reaching for a palette token.
- Reach for `frontend/src/styles.css` before hardcoding a color.
- Match the nearest local UI primitive before writing custom control styling.

## Source of truth

| Concern | Canonical location |
| --- | --- |
| Color tokens and Tailwind theme tokens | `frontend/src/styles.css` (`@theme`) |
| App typography, scrollbars, and global chrome | `frontend/src/styles.css` |
| Local UI primitives | `frontend/src/ui/` |
| Renderer boundary tests | `frontend/src/designSystemAdoption.test.ts` |

`frontend/src/styles.css` is canonical. Component code must not hardcode hex
colors. If a new role is needed, add it to `styles.css`, document it here, and
then consume it through Tailwind utilities or CSS variables.

## Color roles

Tokens come in pairs: a surface and the foreground intended to sit on it. Use
the pair together so contrast and visual hierarchy remain predictable.

| Role pair | Use it for | Don't use it for |
| --- | --- | --- |
| `--color-app-surface` / `--color-app-foreground` | The full app canvas and default page text | Cards, dialogs, sidebar, terminal panes |
| `--color-panel-surface` / `--color-panel-foreground` | Standard panels, cards, panel-like repeated items | The app canvas or terminal content |
| `--color-panel-elevated` / `--color-panel-foreground` | Menus, tooltips, popovers, floating overlays | Inline list rows or permanent chrome |
| `--color-sidebar-surface` / `--color-sidebar-foreground` | Sidebar dock background and normal sidebar text | Main editor/terminal panes |
| `--color-sidebar-active` / `--color-sidebar-active-foreground` | Active ActivityRail icon buttons and selected sidebar chrome | Global selected rows outside the sidebar |
| `--color-terminal-surface` / `--color-terminal-foreground` | xterm pane background and foreground | General panels, cards, or popovers |
| `--color-accent-dim` / `--color-text-primary` | Focus rings, primary affordance borders, low-volume action emphasis | Large filled surfaces or body copy |
| `--color-red` / `--color-text-primary` | Destructive actions, errors, unread notification badges | Cancel buttons or neutral warning copy |

Palette tokens such as `--color-bg-deep`, `--color-bg-surface`,
`--color-text-muted`, and `--color-border-subtle` remain available for legacy
surfaces and one-off local details. New or touched component chrome should first
look for a role token in the table above.

## Sidebar family

Use the sidebar token family inside the ActivityRail and SidebarDock surfaces:

| Token | Role |
| --- | --- |
| `--color-sidebar-rail` | Narrow ActivityRail strip |
| `--color-sidebar-surface` | Resizable SidebarDock panel |
| `--color-sidebar-foreground` | Default rail/dock icon and label text |
| `--color-sidebar-muted` | De-emphasized sidebar metadata |
| `--color-sidebar-hover` | Hover wash for rail and dock controls |
| `--color-sidebar-active` | Persistent active rail/dock item |
| `--color-sidebar-active-foreground` | Text/icon foreground on active sidebar items |
| `--color-sidebar-border` | Rail and dock dividers |
| `--color-sidebar-ring` | Keyboard focus ring inside sidebar chrome |

Do not reuse sidebar tokens for main content panels. Sidebar tokens exist so the
rail and dock can evolve as one family without forcing unrelated surfaces to
change.

## Terminal surface

`--color-terminal-surface` is Whisk's editor-surface equivalent. Use it for
terminal panes and terminal-adjacent embedded surfaces where matching xterm is
more important than matching app chrome.

The xterm theme should resolve through:

| Token | xterm role |
| --- | --- |
| `--color-terminal-surface` | Background |
| `--color-terminal-foreground` | Foreground |
| `--color-terminal-cursor` | Cursor |
| `--color-terminal-selection` | Selection background |

Do not use terminal tokens for forms, dialogs, sidebar panels, or work item
cards. Those surfaces should use app, panel, or sidebar roles.

## Git decoration colors

Whisk does not yet have a canonical diff surface. When one lands, add git
decoration tokens to `frontend/src/styles.css`, document them here, and use them
only for git status decoration:

| Future token | State |
| --- | --- |
| `--color-git-decoration-added` | Added / new |
| `--color-git-decoration-modified` | Modified |
| `--color-git-decoration-deleted` | Deleted |
| `--color-git-decoration-renamed` | Renamed |
| `--color-git-decoration-untracked` | Untracked |
| `--color-git-decoration-copied` | Copied |
| `--color-git-decoration-ignored` | Ignored by git |

Do not borrow git colors for unrelated success, warning, or destructive states.

## Resolution order

Use this order when styling a component:

1. Use an existing role token documented here.
2. Use an existing palette token from `frontend/src/styles.css` only for a local
   detail that does not deserve a reusable role.
3. If the role is missing, add it to `frontend/src/styles.css`, document it here,
   and update the adoption test when useful.
4. For third-party APIs that require computed colors, read CSS variables at
   runtime and provide non-hex fallbacks.
5. Do not add raw hex colors in component code. Hex belongs in the canonical
   token file.

## Typography

- Sans: `var(--font-sans)` from `frontend/src/styles.css`.
- Mono: `var(--font-mono)` for terminals, paths, IDs, code, and literal values.
- Whisk uses explicit pixel sizes in Tailwind arbitrary utilities
  (`text-[13px]`, `text-[11px]`) instead of the default Tailwind type scale.
- Uppercase metadata labels use small type, medium/semibold weight, and modest
  tracking. Body copy and controls should keep letter spacing at the default.

## Radius and elevation

Prefer the local primitive defaults. Cards and permanent panels stay flat with a
token border. Floating surfaces such as dialogs, menus, and popovers may use the
existing shadow utilities already present in the repo, but avoid introducing new
shadow tiers for ordinary panels.

## Components

Use local primitives in `frontend/src/ui/` before adding one-off controls:

| Need | Reach for |
| --- | --- |
| Standard command button | `Button.svelte` |
| Icon-only command | `IconButton.svelte` |
| Modal shell | `ModalShell.svelte` |
| Text input or textarea | `TextField.svelte`, `TextArea.svelte` |
| Select/menu choice | `SelectField.svelte`, `MenuItem.svelte` |
| Status or count label | `Badge.svelte`, `StatusDot.svelte` |
| Sidebar resize affordance | `ResizeHandle.svelte` |

Icons come from `@lucide/svelte`. Let icons inherit text color from the parent
unless the component state requires a documented status token.

## UX rules

- UI copy must not overclaim. Do not imply daemon work completed unless the UI
  has real daemon state proving it.
- Runtime state belongs to the daemon. UI styling changes must not introduce
  desktop-local runtime ownership.
- Keep dense workflow screens scannable: align rows and columns, avoid decorative
  color, and prefer explicit labels over clever phrasing.
- Do not expose placeholder shortcuts or future actions in visible UI.
