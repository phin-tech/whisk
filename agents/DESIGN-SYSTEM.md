# Whisk Frontend Design System

A practical reference for building and reshaping UI in the Whisk desktop app. It
captures the design language established by the Projects view and Work Item view
redesigns so new surfaces look like one app, not several.

Scope: `frontend/src/*.svelte`. This is **client-owned presentation only** — see
`AGENTS.md` for the runtime/ownership rules. Nothing here justifies moving state
or logic into the frontend.

---

## 1. Philosophy — what to do broadly

**Reference: Linear.** When unsure how a surface should look or behave, ask "how
would Linear do this?" Dense but calm; keyboard- and pointer-friendly; content
first, chrome last.

The five principles, in priority order:

1. **Tokens over raw values.** Never hand-pick a hex, a `white/12`, or an
   arbitrary size. Use the CSS-variable-backed utility classes (§2). If a value
   you need doesn't exist as a token, that's a signal to add a token, not to
   inline a one-off. Raw hex like `bg-[#0d0d10]` is a smell — the old work item
   modal was full of them and looked detached from the rest of the app.
2. **Hairlines over boxes.** Prefer `divide-y divide-hairline` rows and single
   thin separators to nested bordered cards. Boxes-within-boxes read as noise.
   One container border max; subdivide with hairlines, not more borders.
3. **Reveal on demand.** Don't render every control all the time. Editing
   controls live behind popovers / `<details>` / hover affordances and surface
   when relevant. The properties rail shows values; clicking a value opens the
   editor. Save/Cancel appear only when a field is dirty.
4. **Content column + properties rail.** The detail-view archetype is a wide
   main column (title, description, the "body" of the thing) beside a narrow,
   compact metadata rail. Actions that mutate metadata live in the rail; the
   primary "what's next" action is surfaced prominently above the fold.
5. **Logic stays pure and testable.** View-models and derivations live in plain
   `.ts` modules (`workView.ts`, `projectView.ts`, `navigation.ts`) with unit
   tests. Svelte components render read models and call callbacks. Keep the
   imperative/visual layer thin.

When a surface gets long or owns a lot of state, **extract it into its own
component** (e.g. `WorkItemDetail.svelte` was lifted out of `WorkBoard.svelte`).
The parent keeps orchestration (what's open); the child owns its local state.

---

## 2. Tokens

Defined in `frontend/src/styles.css` inside a Tailwind v4 `@theme {}` block. Each
`--color-*` var auto-generates `bg-*`, `text-*`, `border-*`, `divide-*` utilities.
**Note the `color-` prefix is dropped in class names**: `--color-text-muted` →
`text-text-muted`, `--color-bg-surface` → `bg-bg-surface`.

### Backgrounds (darkest → lightest)
| Token | Value | Use |
|---|---|---|
| `bg-bg-deep` | `#09090b` | App base, modal body, input fields |
| `bg-bg-base` | `#0f0f11` | Header/footer bars within a surface |
| `bg-bg-surface` | `#18181b` | Popover/menu panels, raised rows |
| `bg-bg-elevated` | `#27272a` | Higher elevation |
| `bg-bg-hover` | `#2e2e32` | Hover fill |
| `bg-bg-active` | `rgba(39,39,42,0.72)` | Active/pressed |

Use `/NN` opacity for tints: `bg-bg-surface/40`, `bg-bg-surface/60`.

### Borders & separators
| Token | Value | Use |
|---|---|---|
| `border-hairline` | `rgba(255,255,255,0.18)` | Section/row separators, `divide-hairline` |
| `border-border` | `rgba(113,113,122,0.4)` | Input borders |
| `border-border-subtle` | `rgba(82,82,91,0.25)` | Secondary buttons, light containers |

### Text
| Token | Value | Use |
|---|---|---|
| `text-text-primary` | `#fafafa` | Titles, primary content |
| `text-text-secondary` | `#d4d4d8` | Body, row values |
| `text-text-muted` | `#b5b5bd` | Labels, meta, timestamps, placeholders |

### Accent & status
| Token | Value | Use |
|---|---|---|
| `accent` | `#7dd3fc` | Hover/active text & borders, focus emphasis |
| `accent-dim` | `#0ea5e9` | **Primary button fill**, focus ring (`focus:border-accent-dim`) |
| `green` | `#4ade80` | running / awaiting / success / approved |
| `blue` | `#38bdf8` | queued / info |
| `amber` | `#fbbf24` | draft / warning |
| `red` | `#fb7185` | failed / cancelled / destructive |

### Type
`font-sans` (Geist/Inter) is the default. `font-mono` (JetBrains Mono) for IDs,
paths, branch names, statuses, counts.

### Custom classes (in `styles.css`, not tokens)
- `.app-scrollbar` — thin custom scrollbar; put on every scroll region.
- `.work-card-title` — 2-line `-webkit-line-clamp` clamp.
- `.writing-vertical` — vertical text (collapsed kanban columns).

---

## 3. Typography ladder

Use explicit bracket sizes, **not** Tailwind defaults (`text-sm` etc.). The
established steps:

- **Section header (eyebrow):** `text-[11px] font-semibold uppercase text-text-muted`
- **Sub-section label:** `text-[11px] font-medium uppercase tracking-wide text-text-muted`
- **Title (detail view):** `text-[18px] font-semibold` (inline-editable input, `bg-transparent`)
- **Row primary / value:** `text-[13px] font-medium text-text-primary` (+ `truncate`)
- **Body text:** `text-[13px]`–`text-[14px] leading-6 text-text-secondary`
- **Meta / timestamps:** `text-[11px] text-text-muted`
- **Mono IDs / paths / counts:** `font-mono text-[10px]`–`text-[11px] text-text-muted`

---

## 4. Component recipes

`bits-ui` is the behavior foundation for reusable controls. New reusable controls
live in `frontend/src/ui/`; that local UI layer is the only public component API
for feature code. Feature components must not import from `bits-ui` directly.

Style the local UI primitives with the same utility-class recipes below. Existing
feature-local markup may stay until its surface is migrated, but do not add a new
ad hoc styled `<button>`, `<input>`, `<textarea>`, dialog, popover, menu, tab, or
select when an equivalent `frontend/src/ui/` primitive exists.
Form controls use local wrappers (`TextField`, `TextArea`, `Switch`, `SelectField`)
from `frontend/src/ui/`; feature components should not style native form controls
directly once a wrapper exists.
Popover/menu controls use local wrappers (`Popover`, `Menu`, `MenuItem`) from
`frontend/src/ui/`; feature components should not build backdrop catchers or
positioned menu panels by hand once those wrappers fit.
Display primitives use local wrappers (`StatusDot`, `Badge`, `SectionHeader`,
`EmptyState`) from `frontend/src/ui/`; feature components should not carry their
own status-dot helpers, ad hoc rounded status chips, repeated section-eyebrow
markup, or local empty-state snippets once these wrappers fit.
Detail-view content/rail layouts use `DetailLayout` from `frontend/src/ui/`;
feature components should not repeat the `minmax(0,1fr)_280px` grid recipe once
the detail-view archetype applies.
Properties rails use `PropertyRow` from `frontend/src/ui/`; feature components
should not repeat the label/value row recipe once a rail row is a simple split
row.
Primary action bars use `NextActionBar` from `frontend/src/ui/`; feature
components should derive the action in a pure `.ts` view-model and pass the view
state plus callback into the primitive.

Icons are `@lucide/svelte/icons/*` at `size={12–16}`. In long components, a
temporary hoisted recipe string can keep legacy markup readable, but migrate it
to `frontend/src/ui/` once the recipe is stable and reused.

### Buttons
```
// Primary (accent fill)
inline-flex h-8 items-center gap-1 rounded border border-accent-dim bg-accent-dim
px-2.5 text-[12px] font-semibold text-text-primary transition-colors
hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-50

// Secondary / outline
inline-flex h-7 items-center gap-1 rounded border border-border-subtle bg-bg-surface/60
px-2 text-[11px] text-text-secondary transition-colors hover:border-accent hover:text-accent
disabled:cursor-not-allowed disabled:opacity-50

// Ghost icon button
inline-flex h-7 w-7 items-center justify-center rounded border border-transparent
text-text-muted transition-colors hover:border-accent/40 hover:bg-accent-dim/10
hover:text-accent disabled:cursor-not-allowed disabled:opacity-50

// Destructive ghost (delete) — same as ghost but:
hover:border-red/40 hover:bg-red/10 hover:text-red
```

### Inputs
```
h-8 w-full rounded border border-border bg-bg-deep px-2 text-[12px] text-text-primary
outline-none transition-colors placeholder:text-text-muted focus:border-accent-dim
disabled:opacity-60
// textarea: add  min-h-16 resize-y  (drop the fixed h-8)
```
**Seamless edit field** (description-style — looks like text until focused):
```
border border-transparent bg-transparent px-2 py-1.5 transition-colors
hover:border-border-subtle focus:border-accent-dim focus:bg-bg-deep
```

### Borderless list (the core list pattern)
```
// Container
divide-y divide-hairline   (optionally wrapped in: rounded border border-border-subtle bg-bg-surface/20)
// Clickable row
w-full min-w-0 px-3 py-2 text-left transition-colors hover:bg-bg-surface/40
// Multi-column row — use a grid template, e.g.
grid grid-cols-[72px_minmax(0,1fr)_120px_32px] items-center gap-3
// Empty state
px-3 py-3 text-[12px] text-text-muted
```

### Popover / overflow menu
```
<div class="relative">
  <button class={ghostIcon} on:click={() => toggleMenu("x")}> … </button>
  {#if openMenu === "x"}
    <!-- backdrop catcher closes on outside click -->
    <div class="fixed inset-0 z-10" on:click={() => (openMenu = "")}></div>
    <div class="absolute right-0 top-full z-20 mt-1 min-w-44 rounded border
                border-border-subtle bg-bg-surface py-1 shadow-lg">
      <!-- menu item -->
      <button class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-[12px]
                     text-text-secondary transition-colors hover:bg-bg-surface/80
                     hover:text-text-primary disabled:opacity-50"> … </button>
      <div class="my-1 border-t border-hairline"></div> <!-- divider -->
      <!-- destructive item: text-red hover:bg-red/10 -->
    </div>
  {/if}
</div>
```
Rules: **one popover open at a time** via a single `openMenu` string. The active
item shows `text-accent`. Escape closes the open popover first, then (if none
open) closes the modal — handle this locally so Escape doesn't blow past the menu.

### Status dot convention
Prefer a colored `●` + muted label over a filled badge.
```ts
function runStatusDot(status: string) {
  if (status === "running" || status === "awaiting_input") return "text-green";
  if (status === "queued") return "text-blue";
  if (status === "failed" || status === "cancelled") return "text-red";
  return "text-text-muted";
}
```
```svelte
<span class={runStatusDot(status)}>●</span><span class="text-text-muted">{status}</span>
```

### Properties row (rail)
```svelte
<div class="flex items-center justify-between gap-2 px-3 py-2">
  <span class="text-[11px] font-semibold uppercase text-text-muted">Label</span>
  <!-- value: read-only display, or a button opening a popover editor -->
  <button class="flex max-w-[60%] items-center gap-1 rounded border border-transparent
                 px-1.5 py-1 text-[12px] text-text-primary transition-colors
                 hover:border-border-subtle hover:bg-bg-surface/60">
    <span class="truncate">{value}</span>
    <ChevronDown size={12} class="shrink-0 text-text-muted" />
  </button>
</div>
```

---

## 5. Layout patterns

### Modal / detail-view shell
```
// Overlay
fixed inset-0 z-50 flex items-center justify-center bg-black/70 px-4 py-6 backdrop-blur-sm
  role="dialog" aria-modal="true" aria-label="…"
// Dialog card
flex max-h-[92vh] w-full max-w-[1180px] flex-col overflow-hidden rounded-md
border border-hairline bg-bg-deep shadow-[0_28px_90px_rgba(0,0,0,0.7)]
// Header / footer bars
shrink-0 border-b/border-t border-hairline bg-bg-base px-5 py-3
// Scroll body
app-scrollbar min-h-0 flex-1 overflow-y-auto px-5 py-5
```

### Content + properties grid
```
grid gap-6 xl:grid-cols-[minmax(0,1fr)_280px] xl:items-start
```
Left `<main>` = the body (title/description/plan/activity). Right `<aside>` =
properties rail. Collapses to single column below `xl`.

### Primary action surfacing
Derive a single "what's next" step (see `workView.ts#deriveNextStep`) and render
it as a full-width bar directly under the header: eyebrow `Next` + one-line
message + primary button. Don't bury the main action in a menu.

### Compact header bar (panel-style, see ProjectsView)
`border-b border-hairline bg-bg-deep px-4 py-2.5`; inline-editable name input
(transparent border, lights up on focus); a mono meta line with dot separators:
```svelte
<span class="font-mono">{n}</span><span>items</span>
<span class="opacity-40">·</span> …
```
plus `Info`(ⓘ) to toggle description and `Ellipsis`(⋯) for overflow.

### Tab bar (ProjectsView)
`border-b border-hairline px-5`; each tab `inline-flex h-10 items-center gap-1.5
border-b px-3 text-[12px] font-medium`; active = `border-accent text-text-primary`,
inactive = `border-transparent text-text-muted hover:text-text-primary`; trailing
count `font-mono text-[10px] text-text-muted`.

---

## 6. Navigation & deep-linking

Cross-view navigation uses a **navigation stack** so a deep-linked target can
return to its origin (`navigation.ts`: `navigateTo`, `navigateBack`,
`clearNavigationStack` over `NavigationState`). A card click calls
`onOpenWorkItem(id)` → `navigateTo("work", { openItemId: id })`; the work board
opens that item and `onDetailClose` pops back. Any user-initiated nav
(`selectSession`, `selectProject`, …) calls `clearNavigationStack()` so the
breadcrumb doesn't surprise the user. Keep deep-link wiring in `App.svelte`.

---

## 7. Accessibility & interaction

- Keep `role="dialog"`, `aria-modal="true"`, and `aria-label` on modals; label
  every icon-only button (`aria-label` + `title`).
- Provide `focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-accent-dim/50`
  on custom focusable controls.
- Escape: close the topmost layer (popover) before the modal.
- Destructive actions (`deleteWorkItem`) confirm via `window.confirm`.
- Every `disabled` state is meaningful — mirror the same guard the action uses.

---

## 8. Don't

- ❌ Raw hex / `white/NN` / `black/NN` for surfaces, borders, or text — use tokens.
- ❌ Nested bordered cards. One border, then hairlines.
- ❌ Always-on edit controls in a metadata rail — reveal via popover.
- ❌ Default Tailwind type scale (`text-sm`, `text-base`) — use the `[px]` ladder.
- ❌ Filled status badges where a dot + label reads cleaner.
- ❌ Business logic in `.svelte` — put pure logic in a tested `.ts` module.
- ❌ A `<style>` block — this app is Tailwind-utility-only.
- ❌ Frontend-owned runtime state (see `AGENTS.md`).

---

## 9. Reference files

- `frontend/src/styles.css` — tokens, `@theme`, custom classes.
- `frontend/src/ProjectsView.svelte` — header bar, tabs, list rows, overflow menu.
- `frontend/src/WorkItemDetail.svelte` — modal shell, content+rail grid, popover
  property editors, status dots, hoisted recipe consts.
- `frontend/src/WorkBoard.svelte` — kanban board + detail orchestration.
- `frontend/src/{workView,projectView,navigation}.ts` — pure view-models (+ tests).

## 10. Verify UI changes

```
cd frontend
npm run check   # svelte-check: 0 errors
npm run build   # production build
npm test        # vitest (raw-source + view-model unit tests)
```
Then run the app and eyeball the surface against ProjectsView for token/spacing
parity. `npm` checks prove types and logic, not pixels — always look.

---

## 11. Adoption punch list

Snapshot audit of `frontend/src/*.svelte` against this doc (2026-06-25). Tick
items off as surfaces are brought onto the system. Re-run the audit greps in §12
to refresh counts.

### On-system (reference) ✅
- [x] `ProjectsView.svelte` — 0 raw hex, 0 `white/NN`, 0 default type scale.
- [x] `WorkItemDetail.svelte` — clean (only `bg-black/70`, the documented scrim).
- [x] No `<style>` blocks anywhere — Tailwind-only rule holds app-wide.

### P1 — `WorkBoard.svelte` kanban board (highest value)
The board region predates both redesigns; the modal beside it is now modern, so
the contrast is visible. ~10 raw hex + ~38 `white/NN`·`black/NN`.
- [x] Map hex surfaces → tokens: `#050506`/`#080809`/`#0a0a0c`/`#0b0b0d` → `bg-bg-deep`/`bg-bg-base`; `#0d0d10`/`#101014`/`#111114`/`#151519` → `bg-bg-surface` (+ `/NN` tints).
- [x] `border-white/12|14|16` → `border-hairline` / `border-border-subtle`; `border-white/28` → `border-border`.
- [x] `bg-white/6|8|10` → `bg-bg-surface/NN`; `bg-black/20|35|40` → `bg-bg-deep/NN`.
- [x] Replace filled status badges with the `runStatusDot` ● + label convention (§4).
- [x] Reconsider bordered cards → hairline rows where the column layout allows.

### P2 — shared dialogs & sidebar (older surfaces)
- [x] `SidebarDock.svelte` — 6 `white/NN`·`black/NN`.
- [x] `ConfirmDialog.svelte` — 1 raw hex + 4 `white/NN`·`black/NN`.
- [x] `ActivityRail.svelte` — 4 `white/NN`·`black/NN`.
- [x] `NotificationsPanel.svelte` — 2 `white/NN`·`black/NN`.
- [x] `NewProjectDialog`, `NewSessionDialog`, `OnboardingPanel`, `SessionsPanel`,
      `CommandPalette`, `SettingsView` — 1 `white/NN`·`black/NN` each.

### P3 — type-scale drift (swap default scale → `[px]` ladder, §3)
- [x] `App.svelte` (3), `PtysPanel` (2), `SessionsPanel` (2).
- [x] `CommandPalette`, `NewProjectDialog`, `NewSessionDialog`, `NotificationsPanel`,
      `OnboardingPanel`, `SettingsView`, `WorkBoard`, `WorkItemsPanel` — 1 each.

> Note: `WorkBoard.svelte` and the `*Panel` files also hold non-runtime sidebar
> state; restyling is presentation-only and must not move runtime state per
> `AGENTS.md`.

## 12. Audit greps

```sh
cd frontend/src
# raw hex surfaces
for f in *.svelte; do c=$(grep -oE '\b(bg|text|border|shadow|ring)-\[#[0-9a-fA-F]{3,8}\]' "$f" | wc -l|tr -d ' '); [ "$c" != 0 ] && echo "$f: $c"; done
# white/NN · black/NN (bg-black/70 scrim is allowed)
for f in *.svelte; do c=$(grep -oE '\b(bg|border|text|divide)-(white|black)/[0-9]+' "$f" | wc -l|tr -d ' '); [ "$c" != 0 ] && echo "$f: $c"; done
# default Tailwind type scale
for f in *.svelte; do c=$(grep -oE '\btext-(sm|base|lg|xl|2xl)\b' "$f" | wc -l|tr -d ' '); [ "$c" != 0 ] && echo "$f: $c"; done
# style blocks (should be none)
grep -l '<style' *.svelte
```

---

## 13. Componentization roadmap

Today the system is **repeated utility-class recipes** (§4) inlined everywhere,
sometimes hoisted to `const` strings within a file (`WorkItemDetail.svelte`).
That was deliberate — it kept the redesigns fast and avoided premature
abstraction. The next maturity step is to promote the most-repeated recipes into
Bits-backed primitive components so a recipe change happens in one place, not 22
files.

### Principles for the split
- **Primitives are dumb and presentational.** They wrap `bits-ui` where behavior
  is needed, take props + snippets, call callback props, and own zero
  runtime/business state. New `frontend/src/ui/` primitives use Svelte 5 runes:
  `$props()`, `$derived`, `$bindable` for bindable values, and `{@render ...}`
  snippets instead of `export let`, `$:`, `<slot>`, or event forwarding. Logic
  stays in `.ts` view-models (§1.5).
- **Wrap a recipe only once it's load-bearing** — used in 3+ places and stable.
  Don't componentize a layout that exists once.
- **Props mirror the recipe's variants**, not arbitrary flexibility. e.g.
  `Button` exposes `variant`/`size`, not a freeform `class` escape hatch that
  re-opens the door to drift. Allow a narrow `class` passthrough for layout
  (margins/grid placement) only.
- **Keep them in `frontend/src/ui/`.** Feature components import local UI
  primitives only; direct `bits-ui` imports stay inside `frontend/src/ui/`.
  Co-locate or add a `*.test.ts` raw-source/render test per primitive group
  (matches the existing vitest pattern).

### Tier 1 — primitives (`frontend/src/ui/`)
| Component | Replaces recipe (§4) | Key props |
|---|---|---|
| `Button.svelte` | primary / outline buttons | Bits-backed; `variant: "primary"\|"outline"\|"ghost"\|"danger"\|"danger-ghost"`, `size`, `align`, `disabled`, snippet content |
| `IconButton.svelte` | ghost / destructive icon button | Bits-backed via `Button`; `label` (a11y), `tone: "default"\|"danger"`, `size`, `disabled` |
| `StatusDot.svelte` | ● status convention + `runStatusDot` | `status`, `showLabel` |
| `Badge.svelte` | stage chip / mono status pill | `tone`, `mono`, slot |
| `TextField.svelte` / `TextArea.svelte` | input / textarea / seamless field | `value` (bindable), `variant: "boxed"\|"seamless"`, `placeholder`, `disabled`, `aria-label` |
| `Switch.svelte` | binary toggle | Bits-backed; `checked` (bindable), `label`, `disabled` |
| `SelectField.svelte` | compact single select | Bits-backed; `value` (bindable), `label`, `options`, `disabled` |
| `Popover.svelte` | popover + backdrop catcher + one-open logic | `open` (bindable), `align: "left"\|"right"`; slots `trigger` + default; handles outside-click + Escape-closes-popover-first |
| `Menu.svelte` / `MenuItem.svelte` | overflow menu items + divider | `MenuItem`: `tone`, `active`, `disabled`, `onclick` |
| `SectionHeader.svelte` | `text-[11px] font-semibold uppercase` eyebrow | slot, optional trailing slot |
| `List.svelte` / `ListRow.svelte` | borderless `divide-y` row | `List` owns `divide-y divide-hairline`; `ListRow` exposes `as: "button"\|"div"`, `cols` (grid template), `onclick` |
| `EmptyState.svelte` | `px-3 py-3 text-[12px] text-text-muted` | slot |
| `ResizeHandle.svelte` | sidebar resize rail button | `dragging`, `label`, `onmousedown` |

### Tier 2 — layout shells
| Component | Replaces | Notes |
|---|---|---|
| `ModalShell.svelte` | §5 overlay + dialog card + header/footer/scroll body | slots `header` / default / `footer`; owns `role=dialog`, `aria-modal`, scrim, Escape→close; **WorkItemDetail and the dialogs all adopt this** |
| `DetailLayout.svelte` | §5 content + properties grid | slots `main` + `aside`; the `xl:grid-cols-[minmax(0,1fr)_280px]` archetype |
| `PropertyRow.svelte` | §4 properties row | `label` + value slot; pairs with `Popover` for the editor |
| `Tabs.svelte` | §5 tab bar | `tabs`, `active` (bindable), per-tab count |
| `PanelHeader.svelte` | §5 compact header bar | inline-edit name, mono meta strip, ⓘ/⋯ slots (generalize the existing `SidebarPanelHeader.svelte`) |
| `NextActionBar.svelte` | the "what's next" bar | takes a `NextStepView` + runs its action |

### Landed splits (2026-06-25)
- [x] `List.svelte` / `ListRow.svelte`, `Tabs.svelte`, and `PanelHeader.svelte`
      live in `frontend/src/ui/`.
- [x] `ProjectsView.svelte` is now a shell over `projects/ProjectOverview`,
      `ProjectAttachments`, `ProjectCards`, `ProjectSessions`, and `ProjectRuns`.
- [x] `WorkBoard.svelte` delegates board chrome/card rendering to
      `workboard/WorkBoardColumn.svelte` and `workboard/WorkItemCard.svelte`.
- [x] The extracted `ProjectsView` and `WorkBoard` feature components use local
      UI primitives for buttons, text fields, text areas, lists, rows, and tabs.
- [x] `WorkItemDetail.svelte` uses `NextActionBar` for the primary next action
      and `PropertyRow` for simple properties rail rows.
- [x] `SettingsView.svelte` is now a shell over `settings/GeneralSettings`,
      `settings/TerminalSettings`, `settings/PluginsSettings`, and
      `settings/IntegrationsSettings`.
- [x] `SessionsPanel`, `NotificationsPanel`, `SettingsView`, `DaemonSettings`,
      `KeybindingsPanel`, sidebar chrome, and `TerminalPane` now use local UI
      primitives or Svelte 5 event attributes at the feature boundary.

### Tier 3 — feature components (decompose the big views)
These compose Tier 1/2; they're not reusable across the app but make the giant
files legible and testable.

- **`App.svelte` (~2k lines)** — the priority decomposition. Pull out:
  `MainRouter` (the `activeMain` switch), `Sidebar` (dock + panels), and lift the
  inline `navigateTo/navigateBack` into the already-tested `navigation.ts`
  (App currently reimplements them — see the Projects audit note). Move toast /
  command-palette / settings wiring into their own already-existing panels.
- **`WorkBoard.svelte`** — split into `WorkBoardColumn.svelte` (stage column +
  collapse) and `WorkItemCard.svelte` (the kanban card, incl. attention rail +
  hover actions). `WorkItemDetail.svelte` is already extracted. Follow-up work:
  move more card/column derivations into tested `.ts` helpers only if the parent
  starts accumulating duplicated view logic again.
- **`WorkItemDetail.svelte`** — once Tier 1/2 exist, lift its sub-blocks:
  `WorkItemPlan.svelte`, `WorkItemActivity.svelte` (questions/feedback/gates/
  history), and `WorkItemProperties.svelte` (the rail). Replace the hoisted
  recipe `const`s with `<Button>`/`<IconButton>`/`<Popover>`.
- **`ProjectsView.svelte`** — split per tab: `ProjectOverview`, `ProjectCards`,
  `ProjectAttachments`, `ProjectSessions`, and `ProjectRuns`, sharing `List`/
  `ListRow`/`PanelHeader`/`Tabs`. Follow-up work: keep business derivations in
  `projectView.ts` as tab components get richer.
- **Dialogs** (`ConfirmDialog`, `NewProjectDialog`, `NewSessionDialog`, …) — all
  reskin onto `ModalShell` + `Button` + `TextField`; clears most of the P2 punch
  list as a side effect.

### Suggested sequencing
1. Land `Button`, `IconButton`, `StatusDot`, `TextField`/`TextArea`, `Popover`/
   `Menu` (Tier 1 core) with tests — highest reuse, immediately cuts drift.
2. Adopt them in `WorkItemDetail.svelte` first (smallest, already on-system) to
   prove the API, then `ProjectsView.svelte`.
3. `ModalShell` → migrate all dialogs (knocks out P2).
4. `WorkBoard` card/column extraction **+** P1 token migration in one pass.
5. `App.svelte` structural split last (largest blast radius; do it once
   primitives are stable so the diff is mostly mechanical).

Guardrail: this is a presentation refactor only. No primitive may own runtime
state, construct PTYs, or persist — see `AGENTS.md`. Keep view-models in `.ts`.
