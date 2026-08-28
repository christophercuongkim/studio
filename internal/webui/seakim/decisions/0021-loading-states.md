# 0021 — SeaKim shows loading with skeletons and a labeled fallback, never a bare spinner

- **Status** Accepted
- **Date** 2026-08-17
- **Affects** every binding; adds a skeleton primitive and a labeled loading
  treatment; new shimmer motion tokens (loop duration, loop easing, a per-theme
  highlight fill); `guidelines/voice-and-tone.md` — its "Empty, loading, error"
  section must point at these once they exist, not just state the rule;
  `conformance.md`; relates to [0002], [0005]
- **Proposed by** TripTogether (Flutter binding adopter), surfaced by the SeaKim
  rollout audit (TT-57)

## Context

SeaKim states a loading *rule* but ships no loading *component*.

Two places already say what loading should look like:

- `guidelines/voice-and-tone.md` ("Empty, loading, error"): *"Loading — name the
  thing being fetched if it takes over a second. **Never a bare spinner on a full
  page.**"* (example: "Checking 40 airlines…")
- `SkButton` doc: *"the control keeps its width. **There are no spinners in this
  system.**"* — an in-control submit shows an in-place `loadingLabel` ("Working…"),
  not a spinner.

So the intent is unambiguous: no spinners; name what's loading. But the binding
surface offers nothing for the two loading contexts that are not a button:

1. **Content-shaped regions** — a list, card grid, table, or a detail body whose
   eventual layout is known.
2. **Unknown outcome** — a route boot or a gate (e.g. a membership check) where
   there is no content layout yet to stand in for.

With no sanctioned treatment, adopters fall back to Material's
`CircularProgressIndicator` — a bare, unlabeled, off-token spinner that the
guideline explicitly forbids and that draws in the Material `ColorScheme`,
bypassing the token layer [0005] depends on. This is not hypothetical: the Flutter
binding (TripTogether) ships full-page bare `CircularProgressIndicator`s in five
screens (trips list, groups list, trip workspace gate, create-trip, verify)
because there is nothing else to reach for. **The rule cannot be satisfied because
the component that would satisfy it does not exist.**

## Decision

**SeaKim ships two loading treatments and no general spinner.**

### The seam: two contracts, not two amounts of shape

The two treatments are not one primitive at two fidelities — they make **different
promises.** A skeleton is a **spatial** claim: *content of this geometry is
arriving here.* A labeled state is a **temporal and semantic** claim: *we are
fetching this named thing.* A skeleton with no shape does not gracefully become a
caption; it is a spatial promise with nothing to place, and a skeleton shaped like
content that never arrives reintroduces the exact layout shift this ADR exists to
remove.

So the author's test is not "is the shape known?" but **"do you know *what* is
arriving, or only *that* something is?"** A membership gate knows neither the shape
nor the outcome — it may resolve to a workspace or a rejection — which is the clean
reason it can never be a skeleton.

### 1. A skeleton primitive — the default, when you know what is arriving

A placeholder that mirrors the geometry of the content that is coming: a rounded
block on `surface-sunken` with a shimmer sweep. Composable, so a list renders N
skeleton rows matching `SkCard` geometry, an `SkStat` renders a skeleton figure, a
table renders skeleton cells. This is the default **whenever the arriving content's
shape is known**, because it communicates both "content is coming" and "roughly
what shape," and it removes the layout shift a centered spinner causes when real
content replaces it.

**The shimmer is a token contract, not a per-binding value.** "Driven by the motion
tokens" is not enough — it leaves the highlight colour, its alpha, the sweep
duration, and the easing to each binding's judgement, which is precisely the
invent-at-point-of-use drift [0013] and [0014] forbid, and worse for being
animated: drift shows up as mismatched *rhythm*, not just a different shade. The
existing motion tokens cannot cover it — they are one-shot curves at 80–320 ms
tuned for discrete transitions, while a shimmer is a continuous loop wanting on the
order of 1200–1600 ms and a **linear or gently symmetric** ease the system does not
have (spring/pop overshoot is meaningless on a loop; `--dur-slow` + `--ease-out`
would strobe). So this adds tokens: a **loop duration**, a named **loop easing**,
and a **per-theme highlight fill** — per [0005] a lightening sweep over
`surface-sunken` in dark and a darkening one in light are not the same alpha, since
dark's sunken sits below the page and light's above white.

Because the shimmer is decorative motion on an indefinite loop, it must honour
`prefers-reduced-motion` (Tier 0 #9): reduced motion collapses it to a **static**
placeholder block, **not** a frozen mid-sweep gradient. A binding that implements
reduced motion as `duration: 0` on the animation ships a permanently half-lit
block — the wrong result — so this is called out explicitly.

### 2. A labeled loading state — when you know only that something is arriving

Where you know only that a fetch is in flight — a route boot, a membership gate —
show the sanctioned caption that **names the thing being fetched**. This is the
direct component form of the voice-and-tone rule.

**It reuses `SkEmptyState`'s frame, not its identity.** Folding this onto
`SkEmptyState` is the draft's instinct and it is wrong: `voice-and-tone.md` treats
empty, loading, and error as three *distinct* states, and `SkEmptyState` "always
names the thing that goes here and offers the action that creates it" and owns the
system's only dashed border, "here and nowhere else." Loading offers no action,
creates nothing, and must not read as absence — and the two states can look nearly
identical for a frame mid-transition, the worst confusion at the worst moment. They
also differ in accessibility: an empty state is static content, while a loading
state must announce **busy** and then announce **completion** (a live region) —
different roles, different live-region behaviour. So the labeled state **reuses the
centred frame's layout tokens** — the same centring and title/description rhythm —
as a **separate treatment, without the dashed border and without being an
`SkEmptyState` variant.** Frame reuse is cheap; semantic reuse is the trap.

**No rotating indicator.** The caption may stand alone or carry a **static or
sweeping linear** element — a determinate bar is a magnitude, not a rotation, and
is fine. An indeterminate spinner, even minimal and even token-coloured, is the
banned object at reduced size, so it is **not permitted**. An escape hatch here
becomes the default, which [0002] learned the hard way.

**In-control loading is unchanged.** `SkButton`'s in-place `loadingLabel` remains
the answer for submit/action buttons; neither treatment above applies there.

## Consequences

- Additive: two treatments added, nothing removed or renamed. **MINOR** per [0011]
  and [0019] — for the rules layer and for each binding.
- **Both bindings must land the new tokens together.** This adds shimmer tokens, so
  a Flutter skeleton shipping with tokens the CSS side lacks is exactly the "token
  added to one binding, not the other" parity gap [0019] flagged as uncovered by
  the version numbers. TT-57 creates real pressure to ship Dart first; that opens
  the gap the moment it lands, so the token set is a single cross-binding change.
- Bindings gain a sanctioned way to satisfy the existing "never a bare spinner"
  rule; the rule stops being unsatisfiable.
- `guidelines/voice-and-tone.md`'s "Empty, loading, error" section stops being an
  orphaned rule — once these treatments exist it points at them, closing the loop
  that let this gap persist.
- The Flutter adopter (TripTogether) can then replace its bare
  `CircularProgressIndicator`s: skeleton rows for the trips/groups lists, the
  labeled state for the workspace membership gate and the verify boot. Tracked as
  TT-57; it stays blocked until this lands upstream and the mirror re-syncs.

## Enforceability

Per [0012], a check can flag a spinner used as a loading affordance — but the
target is **an indefinite rotation, not a class name.** `CircularProgressIndicator`
is one framework's spelling; a check that matches only it binds one binding, and
per [0018] a rule that catches one binding and not the others is how one
interpretation becomes three. State the rule as **"no indefinite rotation as a
loading affordance,"** with each binding's grep shape, outside the two sanctioned
treatments' own source:

- **CSS/React** has no such primitive — the analogue is a hand-rolled infinite
  rotation: a `@keyframes` with `transform: rotate` and `infinite`, or a
  `border-radius: 50%` element spun the same way.
- **Flutter** is `CircularProgressIndicator` **and** a `RotationTransition` (or an
  `AnimationController.repeat()`) driving a rotation — the same bug without the word
  "spinner" in it.

"Is this skeleton shaped like the content it stands in for?" is judgement, like
[0017]'s shared-domain rule and 0012's "is this shadow justified?", and stays
unenforced.

## Rejected alternatives

- **One composable skeleton that degrades to a caption when it has no shape.** The
  most plausible wrong answer, and it fails because the two treatments differ in
  **what they promise**, not in how much shape they carry: a skeleton is a spatial
  claim, a labeled state a temporal/semantic one, and a shapeless skeleton is a
  spatial promise with nothing to place. Worse, a primitive that *sometimes* claims
  geometry will be handed shape wrongly, and a skeleton shaped like content that
  never arrives is the layout shift this ADR removes.
- **Fold the labeled state onto `SkEmptyState`.** Overloads a component whose
  semantics are "nothing is here, do something" with "something is coming, wait" —
  different copy obligations, different accessibility roles, and a dashed border
  that must not appear on a loading state. Reuse the frame's layout tokens, not the
  identity (see decision §2).
- **A general `SkSpinner`.** Contradicts "there are no spinners in this system,"
  communicates nothing about what is loading or its shape, and causes layout shift.
  The button already shows sanctioned in-control loading without one. This is the
  framing TT-57 originally proposed and is rejected on the design system's own
  stated grounds.
- **Do nothing (adopters keep Material `CircularProgressIndicator`).** The
  guideline forbids a bare full-page spinner but ships no alternative, so every
  binding either violates it or hand-rolls a skeleton — exactly the drift [0010]
  and [0013] exist to prevent.
- **Skeleton only, no labeled fallback.** Unknown-outcome boots (route/gate) have
  no layout to stand in for; they need the labeled state.
- **A labeled spinner everywhere (caption + Material spinner).** Better than bare,
  but still a rotation, still off-token, and still shifts layout where a skeleton
  would not. The labeled state's optional indicator is a static or linear element,
  never a rotation — see decision §2.
