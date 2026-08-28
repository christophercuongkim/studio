# Layout

Every screen so far reinvented its own composition. This is the shared answer: how a
page is divided, how wide content is allowed to get, and what changes at each
breakpoint.

Breakpoints themselves are in [`readme.md`](../readme.md#responsive) — measured
**container** width, `sm` under 640, `md` 640–1023, `lg` 1024 and up.

---

## The chassis

Both products use the same shell, and it swaps rather than shrinks.

```
lg  +--------+---------------------------------+
    | side   | top bar                52px     |
    | nav    +---------------------------------+
    | 232px  | main (scrolls)                  |
    |        |                                 |
    +--------+---------------------------------+

md  +----+--------------------------------------+
    | 56 | top bar                              |    side nav collapses to icons
    +----+--------------------------------------+
    |    | main (scrolls)                       |
    +----+--------------------------------------+

sm  +-------------------------------------------+
    | status bar                           44px |
    | screen header                             |
    | main (scrolls)                            |
    | sticky footer action           (optional) |
    | tab bar                              56px |
    +-------------------------------------------+
```

| Element | Token | Notes |
| --- | --- | --- |
| Top bar | `--topbar-h` 52 | Sticky. `--shadow-raised` only when content scrolls under it. |
| Sub bar | `--subbar-h` 44 | Filters, sort, result count. Optional. |
| Side nav | `--sidebar-w` 232 | `--sidebar-w-collapsed` 56 at `md`. Gone at `sm`. |
| Tab bar | `--tabbar-h` 56 | `sm` only. 3–5 destinations. |
| Status bar | 44 | `sm` only, and only in the kits — a real app gets the OS one. |

**Navigation swaps, it does not shrink.** Side nav from `md` up, tab bar at `sm`. A screen
never knows which is showing — that is the shell's job, not the screen's.

## Page padding

| Breakpoint | Horizontal | Top | Bottom |
| --- | --- | --- | --- |
| `sm` | `--space-5` (16) | `--space-5` | `--space-9` (40) |
| `md`, `lg` | `--space-6` (20) | `--space-7` (24) | `--space-11` (64) |

Bottom padding is deliberately larger than top on every screen — the last row of a list
needs air under it, and at `sm` it has to clear the tab bar.

Both kits expose this as `pagePad(bp)` rather than repeating the values.

## Content width

Wide is not free. Three ceilings, by content type:

| Content | Max | Why |
| --- | --- | --- |
| Prose | **68ch** | Beyond that the eye loses the line return. |
| Forms | **620px** | A 1200px-wide text input looks broken and scans badly. |
| Tables, lists, dashboards | full width | Comparison wants every column visible. |

So a checkout screen caps its form column at 620 even on a 2560px monitor, while a roster
table uses everything it is given. The container tokens (`--container-sm` through
`--container-xl`) exist for pages that need a hard outer bound — marketing, docs, the
overview page.

## Composition patterns

Four, and between them they cover every screen in both kits.

### 1. Stat strip + sections

The dashboard shape. A bordered figure strip, then titled sections below.

```
lg   [ stat | stat | stat | stat ]        4 up
md   [ stat | stat ]                      2x2
     [ stat | stat ]
sm   [ stat | stat | stat ]               3 up, drop the fourth, drop hints
```

At `sm` the strip goes edge to edge — it loses its left and right borders rather than
sitting inset, so it reads as part of the page rather than a floating card.

### 2. Content + rail

Detail screens. Primary content beside a 300–320px rail of secondary panels.

| Breakpoint | Layout |
| --- | --- |
| `lg` | `minmax(0,1fr) 300px` grid |
| `md`, `sm` | Rail moves **below** the content, full width |

The rail never becomes a drawer or a tab. It is secondary information, so at narrow
widths it goes where secondary information goes: after the primary content.

**Where the rail held the primary action** — Checkout's pay button, Trip detail's pay
balance — that action moves to a **sticky footer** below `lg`, because a page-bottom
button after a long form is not an action, it is a scavenger hunt.

### 3. Filter rail + results

Search screens. A 248px filter column beside a result list.

| Breakpoint | Layout |
| --- | --- |
| `lg` | Persistent 248px rail |
| `md`, `sm` | Chip row showing active filters, plus a Filters button that expands the same panel inline |

The chip row matters: collapsing filters behind a button hides *that filters are applied*,
which is the one thing the user needs to know. The chips carry that; the button carries
the rest.

### 4. Full-bleed header

Trip detail. Imagery spans the content width edge to edge, with the title block below it
rather than over it.

Text over an image needs a solid capsule, never a protection gradient — and since no real
photography has been supplied, every image area in the kits is a labelled `Placeholder`.

## Grids

There is no 12-column grid, and adding one would be a mistake. Nothing here needs to
reference "columns 3 through 7" — the layouts are a shell plus one of four patterns, all
expressed in flex and grid with `gap`.

Card grids use `repeat(auto-fill, minmax(260px, 1fr))` and let the browser decide the
count. At `sm` they collapse to one column and go edge to edge.

**Always `gap`, never margins between siblings.** Gap spacing survives reordering,
deletion, and duplication; a margin-bottom on every child but the last does not.

## Section rhythm

Density 7/10 means **compact controls, generous section gaps**, and that contrast is the
whole trick — it is what lets a dense screen stay legible.

| Gap | Token | Between |
| --- | --- | --- |
| Within a group | `--space-4` / `--space-5` | Form fields, list items, chips |
| Between sections | `--space-7` / `--space-8` | Titled blocks on a page |
| Around a page | `--space-9` / `--space-11` | Top and bottom of the scroll |

A section is a `--type-eyebrow` label, optional meta and trailing action, then content.
Both kits expose this as `PageSection` / `SectionLabel` — the label is uppercase mono at
`--text-tertiary`, never a heading-sized title, because the screen already has one `h1`.

## Sticky elements

Four things may stick, and nothing else:

| Element | Where | Shadow |
| --- | --- | --- |
| Top bar | Top of main | `--shadow-raised` when content is under it |
| Table header | Top of a scrolling table | none — the hairline does it |
| Selection or action bar | Bottom of the scroll region | `--shadow-raised` |
| Tab bar | Bottom, `sm` only | none — the top border does it |

This is the one sanctioned exception to "shadows only on things that float": a sticky bar
genuinely does have content passing beneath it, so the shadow is describing something
true.

## Safe areas

At `sm`, the tab bar adds the bottom safe-area inset **below** the bar rather than inside
it, so targets stay 44px on a device with a home indicator. A sticky footer above the tab
bar inherits that spacing rather than adding its own.

## What a binding owes

Tier 1 — the mechanism is yours, the outcome is not:

- Measure **container** width, not viewport. `ResizeObserver`, `LayoutBuilder`,
  `GeometryReader` — whichever your platform gives you.
- Swap navigation at `sm`; do not shrink it.
- Move rails below content rather than into a drawer.
- Move rail-held primary actions to a sticky footer below `lg`.
- Keep the three content ceilings.
