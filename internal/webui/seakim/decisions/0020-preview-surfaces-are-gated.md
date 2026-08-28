# 0020 — Preview surfaces are deliverables, and a render gate proves it

- **Status** Accepted
- **Date** 2026-08-16
- **Affects** `index.html` + the `*.card.html` gallery; `next/example`; `flutter/example`;
  `ds-shim.js`; a new `tool/preview-check.mjs`; `conformance.md`; CI; the gate list in
  `CLAUDE.md`

## Context

The four gates all pass on source that never renders. `version-check`, `build-tokens --check`,
`conformance-check`, and `flutter analyze && flutter test` read files and assert properties of
text — none of them draws a component and looks at the result. That gap has now cost twice, in
the same shape:

- **Lesson 14.** `Range` was built, exported, and specced, but its preview card mounted blank —
  `ds-shim.js` has a hand-maintained `FILES` registry, and a component missing from it renders
  nothing. Every gate was green; the card was empty.
- **Lesson 15.** The same `Range` was absent from the two demo apps (`/next`, `/flutter`)
  because each hand-picks the components it shows. "Built and exported" read as done; "a
  consumer can see it running" was the actual bar, and nothing enforced the difference.

Both are one root cause: **green never means visible.** The QA environment hosts exactly the
surfaces that would have caught these — the root gallery's card pages, `/next`, and `/flutter`
— but they are checked by a human remembering to look, which is how both slipped. Two incidents
in two consecutive components is not a two-incident problem; it is a 100% failure rate on the
one class of bug no gate covers.

## Decision

**The preview surfaces are deliverables, not demos, and a layered check — web-static,
Flutter-in-Dart, and a browser render — proves a component appears on them.**

### The canonical surfaces

Three, all already built and served by the QA `Dockerfile`:

1. **The root gallery** — `index.html` and the `*.card.html` pages (`core`, `forms`,
   `pickers`, `feedback`, `data`), which mount the `*.demo.jsx` projections through
   `ds-shim.js`.
2. **`/next`** — the Next.js example app.
3. **`/flutter`** — the Flutter example gallery.

These are the closest proxy for a shipping consumer — fantasy-hub on Next/React, triptogether
on Flutter — so a component absent from them is untested where it matters.

### The check, in three parts — web-static, Flutter-in-Dart, and render

**A fast static pre-filter for the web (every push).** `tool/preview-check.mjs` asserts each
`index.js` export is in `ds-shim.js`'s `FILES` registry and its category `*.demo.jsx`. A
millisecond, no browser; catches lesson 14's registry gap. It is also the graceful fallback
when CI's browser infrastructure is down.

**Flutter coverage lives in a Dart test, not this script.** The first draft derived a class
name from each barrel filename and grepped the example source for it — and that is exactly the
brittleness the rest of this ADR warns against: `sk_radio.dart` exports `SkRadioGroup`, not
`SkRadio`, and a grep cannot tell a real reference from the same word in a comment. So Flutter
coverage is a **compiler-checked** test instead: `flutter/example/test/preview_coverage_test.dart`
reads a canonical `skShowcase` list — one entry per widget file, each referencing the real
widget **type** — asserts it covers every barrel widget (a count against the barrel's stable
`export 'src/widgets/…'` lines) and that each entry builds without throwing. A renamed or
removed widget breaks compilation, not a string match. The gallery renders from the same
`skShowcase`, so the demo cannot drift from what the test verifies. This runs under
`flutter test` in CI.

**A render gate.** `tool/preview-check.mjs --render` serves the built
surfaces, loads them in headless Chrome (`--enable-unsafe-swiftshader` for the Flutter
CanvasKit surface, per lesson 13), and reads the result. Two assertions the static layer
cannot make:

- **Presence, not HTTP 200.** A page that loads with a blank card **fails** — the assertion is
  that the component's marker is in the DOM. This is the half a static check cannot do: `Range`
  was in every registry and still mounted blank (lesson 14), because the failure lived in the
  shim's transform, the demo's props, or a runtime error in the projection.
- **No error during load.** The same headless session listens for console errors and uncaught
  exceptions and fails on any. A component can render its marker and still be broken — a React
  error boundary swallows a child's throw, a Flutter widget paints an error box under its
  semantics label — and both pass a bare presence check. One listener, same session, closes
  that gap.

### The component set is derived, not restated

Each side **enumerates from the existing source of truth** — `index.js` exports for web, the
barrel's widget-file lines for Flutter (counted, in the Dart test) — and consults a small list
only for each component's *surface* and *marker*. So neither list can be *missing* an entry
(the derived set flags it), only wrong about one — which halves the drift surface that sank
`ds-shim.js`'s `FILES` in the first place. Fixing a registry-drift bug by adding a registry
only works if the new list cannot silently omit.

The marker is **the component's accessible name by default** — an `aria-label` or accessible
name that `guidelines/accessibility.md` already requires and the a11y rules already assert, so
it cannot drift without also breaking accessibility (the equivalent already exists for `Range`'s
three glyphs). A `data-testid` is the **documented exception**, for a non-interactive mark with
no accessible name of its own — never the default, because a testid is an identifier maintained
only for the gate, which is the classic thing that rots.

### What it deliberately does not do

It is a **presence** check, not a visual-regression check. It asserts the component rendered
into the DOM without erroring, not that it looks pixel-correct — screenshot diffing is flaky,
slow, and needs golden images the design side cannot regenerate (it cannot run code). Whether a
component *looks right* stays the manual review `conformance.md` already owns; this gate only
kills the class of bug where it does not render at all, or renders broken.

## Consequences

- **A fifth gate, and it needs a browser.** CI must build `/next` and `/flutter` and run
  headless Chrome. This is real cost, but the QA `Dockerfile` already builds both apps to
  deploy them — the gate renders what is shipping anyway, rather than adding a new build. The
  static pre-filter runs regardless, so a browser outage degrades coverage, it does not remove
  it.
- **`conformance.md` lists `preview-check.mjs` as a repo gate, not a binding obligation.** A
  contributed SwiftUI binding owes SeaKim its Tier 0 conformance, not three preview surfaces —
  requiring those would contradict [0010]. The line keeps a binding author from reading the
  fifth gate as a fifth thing they owe. It is a repo gate, not a tier.
- **Adding a component gains one required step:** a manifest entry naming its surfaces and (if
  non-interactive) its marker. This **narrows** lessons 14 and 15 to the residue a check cannot
  reach — a forgotten manifest entry for a component the author *also* forgot to export is still
  invisible, because the derived set never saw it. What was automatable is now automated; what
  remains is smaller and named, not "remember everything."
- **The gate list in `CLAUDE.md` grows to five**, split by cost across the hooks. The **static
  pre-filter runs pre-commit** — it is instant and has no false positives, so it belongs at the
  earliest point, the moment a component is added without its registry entry. The **render gate
  runs pre-push (on component/demo diffs) and in CI** — it is slow and needs a browser, so it
  cannot live pre-commit without being bypassed. Hooks are per-developer ergonomics; CI is the
  real gate.
- **`ds-shim.js`'s `FILES` registry stops being a silent single point of failure.** A gap in it
  now surfaces as a blank-card render failure — or a static-layer failure — with a component
  name attached, not a mystery empty box.

## Rejected alternatives

- **Keep it as lessons 14 and 15.** Already tried; it is the status quo, and it failed twice in
  two consecutive components. A discipline that depends on remembering, applied to the one thing
  no other gate covers, is exactly where a check earns its cost.
- **The static check alone, no browser.** Registry-membership is cheap and catches lesson 15,
  so it is worth having — as the pre-filter, not the whole gate. It cannot catch lesson 14:
  `Range` was in every registry and still mounted blank. Presence in a *list* is not presence in
  the *DOM*, and only the latter is what regressed.
- **Visual-regression / screenshot diffing.** Catches the blank-card case but drowns it in false
  positives on every deliberate visual change, and needs a golden-image store the design side
  cannot regenerate. Presence-without-error is the property that actually regressed; assert
  that, not appearance.
- **A unit test per binding that imports each component.** Proves it *constructs*, which the
  Flutter tests already do — and `Range` constructed fine while its card stayed blank. The bug
  lives in the wiring between the component and the surface, which only a render of the real
  surface exercises.
- **Trust the QA deploy and eyeball it.** A human looking at three surfaces after every change
  is the check this replaces; it is what let both lessons happen.
