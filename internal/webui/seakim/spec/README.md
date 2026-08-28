# Specs

One platform-neutral contract per component: anatomy, variants, sizes, states,
responsive behaviour, accessibility. **No code.** How to call a component is in that
binding's `.prompt.md`; what a component *is* lives here.

See [0001](../decisions/0001-platform-neutral-spec-layer.md) for why this folder exists,
and why it deliberately has no snippets in it.

## Rules for writing one

- **Reference tokens, never restate values.** Write `--control-h-md`, not `34px`. A spec
  with a literal pixel value that also lives in a token is a future contradiction with a
  date on it. Token names map predictably across bindings (`--control-h-md` becomes
  `SkControl.md`), so the CSS name is the canonical spelling.
- **State the judgement, not just the geometry.** "If two things on a screen are
  accent-coloured, one of them is wrong" is the most useful line in the colour docs. That
  register belongs here too.
- **Say what the component is not for**, and name the one that is. Most misuse is a wrong
  choice, not a wrong prop.
- **Cover `sm`.** A spec that only describes the wide layout is half a spec.
- **Cover both themes** where they differ structurally rather than only in value.

## Written on demand

Specs are not written in a batch. One exists when the component does not exist yet, or a
second binding is about to implement it, or two bindings disagree and someone has to be
wrong. A spec documenting a component that already works identically in two places
describes the present and then rots.

## Index

| Spec | Status | Bindings |
| --- | --- | --- |
| [Table](Table.md) | Built | React, Flutter — Bench's roster and standings both use it |
| [DatePicker](DatePicker.md) | Built | React, Flutter |
| [Slider](Slider.md) | Built | React, Flutter |
| [Range](Range.md) | Built | React — Bench projections; Flutter owed |
| [Popover](Popover.md) | Built | React, Flutter — anchored overlay species (0022) |
| [AvatarStack](AvatarStack.md) | Built | React, Flutter — facepile primitive (0024) |
| [Combobox](Combobox.md) | Built | React, Flutter — searchable long-list picker (0028) |
| [ErrorState](ErrorState.md) | Built | React, Flutter — first-class error treatment (0029) |

## Known debt

- **Opinions still live in `.prompt.md` files.** 0001 says cross-platform opinion moves
  here and gets deleted from the prompt. That edit has not been made for the 23 existing
  components, so some opinions currently sit in a React-shaped folder. Do it per component,
  when that component is next touched — rewriting 23 files in one pass is churn with no
  reader.
- **No spec covers the existing 23 components.** Correct per the on-demand rule, but it
  means a third binding still learns most of the library by reading React source. The first
  component a SwiftUI author disagrees with is the first spec to write.
