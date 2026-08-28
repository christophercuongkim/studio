# 0028 — A searchable long-list picker is its own component, not a hand-rolled trigger

- **Status** Accepted
- **Date** 2026-08-23
- **Affects** `spec/` (a new `Combobox` spec), the field-trigger conformance check (`conformance.md`); **every binding** — a native `<select>` cannot be searchable, so this is a fully custom widget in React *and* Flutter, not a Flutter-only concern

## Context

`SkSelect` is an anchored-popover (0022) picker for a **short, closed set**: it opens a
list under the trigger and you tap a row. It has no filter box, by design — a five-item
role menu does not need one, and a search field there is noise.

But the system has no answer for the **long** list — currency (~160 rows), timezone,
country, language. Scrolling 160 rows in an anchored popover is not a picker, it is a
punishment. So the consumer stops using `SkSelect` and hand-rolls the whole thing:

1. a **fake trigger** — an `SkPressable` wrapping a bordered `Container` that
   re-implements the select's closed-state chrome (hairline border, the field label, the
   caret, the 44px touch target) by eye, and
2. a **`showSkSheet` + `SkInput` + `ListView`** filter surface stitched together at the
   call site, with its own matching logic.

Both halves are a re-derivation of chrome the system already owns, and the re-derivation
is where it goes wrong. In a binding consumer (TripTogether's currency picker), the
hand-rolled trigger shipped **without the focus ring** — because a bare `SkPressable`
applies the press-scale and keyboard activation but not the focus ring, and nothing
reminded the author that a field trigger owes one.

**The reference binding has the same gap.** Flutter's `SkSelect` trigger applies
`borderFocus` only while the menu is *open* — there is no `SkFocusRing` and no
focused-state handling, so a keyboard-focused-but-closed `SkSelect` shows the plain
default border. That is a standing **Tier 0 #6 ("focus is always visible") violation** in
the primitive itself, not just in a consumer's copy of it — and it diverges from React's
`Select`, which rings correctly (`box-shadow: var(--focus-ring-inset)` on focus). So the
missing focus ring is not a consumer mistake; it is a missing primitive that every
long-list picker re-solves, each slightly differently. The focus-ring omission is the
first bug the gap produced; divergent match logic (substring here, fuzzy there) is the
second.

There is also no shared **matcher**. Each consumer invents its filter — TripTogether
wrote a scored-subsequence `fuzzyScore` for the currency list — so ranking and
empty-state behaviour drift picker to picker with nothing to point at as correct.

## Decision

**Add a searchable long-list picker as its own component — `SkCombobox` — not a
`searchable` mode on `SkSelect`.** `SkSelect` stays as-is for short closed sets.

**Two components, one shared trigger, because `searchable` is not a mode.** `size` and
`invalid` are modes — they change how a control looks. `searchable` changes what the
control *is*: the focus target on open, the keyboard model, the accessibility role, and
the presence of a text input. A `SkSelect` with the flag and one without share almost no
behaviour below the trigger. Bolting a filter, a keyboard model, an empty state, and a
managed-focus contract onto the control whose whole point is a *short closed set* is a
bimodal contract, not a variant.

The shared chrome *is* real and must not be re-derived — so extract it. A single internal
**field-trigger part** (the hairline border, the field label via the standard field
wrapper, the caret, the hover/focus border transition, **the focus ring**, and the 44px
touch floor per 0023) backs both `SkSelect` and `SkCombobox`. Fixing that part fixes the
Tier 0 focus-ring gap above in `SkSelect` at the same time. `SkCombobox` then owns, on top
of the shared trigger:

- **The overlay.** The anchored-popover species (0022) with a filter input pinned above a
  scrollable, keyboard-navigable option list, resolving on `sm` to the sheet species as
  the overlay-species rule already dictates.
- **Type-to-filter.** See the matcher contract below.
- **The a11y.** `role="combobox"`, `aria-expanded`/`aria-controls`/`aria-activedescendant`
  over a listbox in React; the Flutter equivalent. On open, focus moves into the filter
  input; arrow keys move the active option; Enter selects; Escape and outside-press close
  (0022); the selection is announced.

**This holds across bindings — which is why it must be two components, not one.** React's
`Select` is a **native `<select>`**: the browser owns the popup, the typeahead, the
keyboard model, and the a11y. A searchable combobox *cannot* be a native `<select>` — it
is a custom `role="combobox"` widget. A `searchable` flag would therefore make one React
component a native element in one mode and a hand-built widget in the other: different
DOM, different a11y tree, different focus behaviour, behind a boolean. Two components is
the coherent story: `Select` stays native in React and simple in Flutter; `SkCombobox` is
the *same custom widget in both bindings*. Per 0018's lesson (a rule that holds
differently across bindings is how one system becomes three), the parity is actually
better with two components, because the searchable primitive is fully ours everywhere.

**When to reach for which — the axis is scannability, not row count.** A user reaches for
`SkSelect` when they can *scan* for their option and `SkCombobox` when they must *search*
for it. Scannability is driven by whether the set is open or closed, familiar or
unfamiliar, and meaningfully ordered: a closed 20-item list of workspace roles is
scannable; an 8-item list of unfamiliar timezone identifiers is not. The spec states that
principle first and a number only as a tiebreaker — "beyond roughly a dozen unfamiliar
rows, assume typing." A firm row-count rule on a soft axis gets gamed; a stated principle
with a rough number does not.

**The matcher: substring by default, fuzzy opt-in, ranking in the contract.** Fuzzy
matching is unpredictable — a subsequence match makes `usd` hit `United States Dollar` and
several things the user did not mean, with no mental model for why the top result is
wrong. Substring is boring and explicable: what you typed appears in what you got. So the
default is **prefix-then-substring**; a scored subsequence (the contributed
`skFuzzyScore`, now accent-folding) is opt-in per picker where the list's shape justifies
it. **Ranking is part of the contract, not the binding's choice** — otherwise the same
picker ranks differently in two apps, the exact drift this ADR opens with. The order is:
exact, then prefix, then substring, then original list order within each tier. **The
empty-result state is specified**, not merely named: per `spec/Table.md`'s filtered-to-
nothing precedent and `guidelines/voice-and-tone.md`, it names the filter and offers to
clear it (`No currencies match "xyz"` + a clear action), never a bare empty state implying
the list itself is empty.

## Consequences

- The hand-rolled trigger disappears, and with it the class of bug it produced — a field
  trigger that forgets the focus ring, the touch floor, or the border transition — because
  the consumer no longer builds a field trigger at all.
- The extraction fixes a **standing Tier 0 focus-ring violation** in Flutter `SkSelect` as
  a side effect, and closes the React/Flutter focus divergence.
- One matcher, one ranking order, one empty state, one keyboard model across every long
  list, instead of per-picker drift.
- Per 0012, "a field trigger renders the focus ring and meets the 44px floor" becomes
  statically checkable **at the named shared trigger part**, once — which is only possible
  because there is now a named thing to check (a `searchable` flag on `SkSelect` would
  leave nothing new to point at). What stays manual is "the consumer used the primitive
  rather than hand-rolling a lookalike"; no static check catches a bespoke
  `SkPressable`+`Container` that happens to resemble a field. Per the standing pattern
  (0012, 0017, 0018), the check asserts the outcome; the discipline stays judgement.
- New component surface: per 0001 `SkCombobox` owes a `spec/` file, and per 0020 a preview
  entry and manifest row in every gated binding before it ships.
- **Rollout is both bindings, not Flutter-first.** Because the React and Flutter
  `SkCombobox` are the same custom widget, they should land together; a primitive that
  exists in one binding and not the other is a divergence with no version number to expose
  it (0019). If React must lag, that lag is declared like any other, not silent.
- Versioning (0019): **Minor** — adds a component and a spec; no Tier 0 rule moves (the
  focus-ring fix restores an existing rule rather than changing one).

## Rejected alternatives

- **A `searchable` mode on `SkSelect`.** The tempting answer, and the wrong one. The flag
  changes the interaction model, the focus contract, and the a11y role — that is a
  different control, not a variant. Worse, it is incoherent in the reference binding: React
  `Select` is a native `<select>`, so `searchable: true` would swap a native element for a
  hand-built `role="combobox"` widget behind a boolean. Extract the shared trigger and ship
  two components instead.
- **Leave it to consumers.** The status quo, and it already shipped a trigger with no focus
  ring and N divergent matchers. An anchored field pattern that every long list re-derives
  is a spec gap, not a non-decision.
- **Make `SkSelect` always searchable.** A five-item role menu does not want a filter box;
  forcing one adds a focus target and a keyboard mode to a control that needed neither.
  Searchability is a property of the list, so it is a different component, not a default.
- **Widen `showSkSheet` into a "picker sheet" helper.** Standardises the *overlay* half but
  leaves every consumer still hand-building the trigger — the half the focus-ring bug lived
  in. Half a primitive re-opens the same gap.
- **Pull a third-party combobox.** Imports someone else's tokens, focus model, and overlay
  species; the whole point of 0022/0023 is that these are ours to define.
