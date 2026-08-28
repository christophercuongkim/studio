# Combobox

A searchable picker for a **long or unfamiliar** list — currency, timezone, country,
language. A field trigger opens an anchored surface with a search input over a filtered,
ranked, keyboard-navigable option list. Per
[0028](../decisions/0028-searchable-select.md).

**Not for:** a short, closed set you can scan at a glance — that is `Select`, which shows
every row with no filter. The axis is **scannability, not row count**: reach for `Select`
when a user can *find* their option by looking, and `Combobox` when they must *type* to
find it. A closed 20-item list of roles is scannable; an 8-item list of unfamiliar
timezone identifiers is not. Beyond roughly a dozen unfamiliar rows, assume typing.

## Anatomy

```
┌─────────────────────────────┐   trigger — the shared field-trigger part,
│ US Dollar               ⌄   │   identical to a closed Select
└─────────────────────────────┘
┌─────────────────────────────┐   overlay (anchored popover, 0022)
│ 🔍 us                       │   ← search input, pinned, focused on open
├─────────────────────────────┤
│ USD · US Dollar          ✓  │   ← ranked options, one active/highlighted
│ AUD · Australian Dollar     │
└─────────────────────────────┘
```

| Part | Treatment |
| --- | --- |
| Trigger | The shared field-trigger: `--surface-raised` fill, hairline border thickening to a 2px inset `--border-focus` on keyboard focus or while open, a trailing caret, the 44px touch floor. It is the *same part* a closed `Select` uses — a Combobox never rebuilds it. |
| Overlay | Anchored-popover species: `--surface-overlay`, hairline border, `--shadow-popover`. On `sm` it resolves to the sheet species, per the overlay-species rule. |
| Search input | Pinned above the list; receives focus when the overlay opens. Standard field chrome. |
| Option row | Minimum 44px, left-aligned label; `--surface-hover` on hover, `--surface-selected` + `--text-accent` + a check on the selected value; one row is the *active* (keyboard-highlighted) row. |

## Matching and ranking

The matcher is **shared code, not per-picker**, so every long list behaves the same.

- **Default: substring, tiered.** Rank by **exact, then prefix, then substring**, then no
  match; within a tier, keep the list's original order. Substring is the default because
  it is explicable — what you typed appears in what you got, and where.
- **Fuzzy is opt-in.** A scored subsequence (rewarding start-of-string, start-of-word, and
  contiguous runs) is available where the list's shape — short codes plus long names —
  justifies a scattered match. It is not the default, because fuzzy ranking is harder to
  predict.
- **Case- and accent-insensitive.** `zurich` finds `Zürich`; `koln` finds `Köln`.
- **Ranking is part of the contract**, not the binding's choice: the same picker must rank
  the same way in every app, or the drift this component exists to end reappears.

## States

| State | Treatment |
| --- | --- |
| Closed | The field trigger with the selected label, or the placeholder. |
| Open | Overlay with the search input focused and the full list ranked. |
| Filtered | Only matching rows, ranked; the active row highlighted. |
| Filtered to nothing | **Names the filter and offers to clear it** — `No options match "xyz"` with a clear action — never a bare empty box implying the list itself is empty (the `Table` filtered-to-nothing precedent). |
| Disabled | Field-trigger disabled tokens; the overlay does not open. |

## Responsive

On `sm` the overlay is the **sheet** species (the overlay-species rule), so the list and
its search field get the full width and thumb reach rather than a cramped anchored panel.
On `md`+ it is the anchored popover under the trigger.

## Accessibility

- The trigger is a combobox control (`role="combobox"`, expanded/controls/activedescendant
  in ARIA; the platform equivalent elsewhere), labelled by its field label.
- **Focus moves into the search input on open** and returns to the trigger on close.
- **Keyboard:** arrow keys move the active option (wrapping); Enter selects it; Escape and
  outside-press close.
- The selection is announced. The 44px floor holds on touch (0023).
