# 0005 — Light mode is first-class, and enforced

- **Status** Accepted
- **Date** 2026-08-04
- **Affects** every binding; every specimen card and UI kit; review process

## Context

The readme says light is "a peer, not a filter of dark" and describes real design work
behind it — the light card is pure white on an off-white page, so cards read *lighter*
than the page, the inverse of dark's logic. `tokens/theme-light.css` and
`SkColors.light` both exist and are hand-tuned, not derived.

And yet: **every preview in the repo is dark.** Every specimen card, both UI kits, all
six slide templates, the overview page. Light mode has never been looked at in anger.
The two kits carry a theme toggle, which is the only reason it can be seen at all.

So the honest current state is: light is *specified* as first-class and *verified* as
nothing. This is a status decision before it is a visual one, because the answer
determines whether that gap is debt or a documented limit.

## Decision

**Light is first-class. Dark stays the default.** Best-effort is rejected.

The reasoning is not aesthetic, it is structural. Light being second-class would mean
the semantic token layer is only trustworthy in one theme — and the semantic layer is
the mechanism the entire system rests on. A binding author who cannot trust
`--surface-card` to be right in both themes will start reaching for stone steps
directly, and at that point per-app theming breaks too. Light mode is the test that
keeps the abstraction honest.

Three enforcement rules follow, and they are the substance of this ADR:

**1. Every new specimen card and screen is reviewed in both themes before it merges.**
A component that has only ever been seen in dark is not done. This is a checklist item
in 0008, not a good intention.

**2. Contrast is verified against both ends.** A new accent hue already has to clear
4.5:1 on `--stone-950` *and* `--stone-50`. Extend that to any new semantic token: if it
only works in one theme it is not a semantic token, it is a dark-mode value that needs a
light peer written alongside it.

**3. Neither theme is derived from the other.** No filters, no inversion, no
programmatic lightness flip. Where light needs a different *structure* — not just
different values — light wins its own treatment. The known cases:

| Concern | Dark | Light |
| --- | --- | --- |
| Surface direction | Cards step **up** from the page | Cards step **up to white**, page is off-white |
| Shadows | Heavy, cool-black, carry most of the lift | Light and warm; the hairline does more of the work |
| Status colours | `400` steps | `500` steps, separately tuned |
| Accent fill | `brand-400` | `brand-500`, because 400 on white fails contrast |

### OS preference is not followed

Dark is a **brand** decision, not an accessibility one. A first-time visitor gets dark
regardless of their system setting; a returning visitor gets whatever they last chose,
persisted. `prefers-color-scheme` is ignored for the initial theme.

This is the most arguable line in the ADR and is called out as such. It is defensible
because SeaKim products are consumer apps with a deliberate look, not documentation that
should disappear into the OS. Revisit if a real user tells us otherwise — that is worth
more than the principle.

`prefers-reduced-motion` is a different matter and **is** always honoured.

## Consequences

- Roughly doubles visual review time per component. This is the actual cost and there is
  no way to have first-class light without paying it.
- The existing repo is out of compliance the moment this is accepted: ~20 specimen cards,
  2 kits, and 6 slide templates have never been checked in light. Recorded as **known
  debt** rather than pretended away. Slides are exempt (they are projected, dark-only by
  design) — that leaves the cards and kits.
- Any future "just tint dark to make light" shortcut is now explicitly off the table,
  including for a SwiftUI binding that might get it cheaply from the platform.

## Rejected alternatives

- **Light is best-effort, dark is the product.** Honest about today, but it licenses the
  semantic layer to be half-trustworthy, and that layer is load-bearing for per-app
  theming as well as theming.
- **Light is the default.** Contradicts a decision the user already made deliberately.
- **Follow the OS.** Reasonable, and it means most users never see the brand's intended
  look. Rejected on brand grounds, flagged as the weakest call here.
- **Derive light from dark programmatically.** Would produce a grey-card-on-grey-page
  light theme — the thing that makes most dark-first systems look wrong in light.
