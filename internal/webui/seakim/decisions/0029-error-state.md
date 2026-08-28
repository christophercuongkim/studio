# 0029 — The error state is a first-class treatment, not an SkEmptyState variant

- **Status** Accepted
- **Date** 2026-08-23
- **Affects** every binding; adds an error-state treatment (React + Flutter);
  `guidelines/voice-and-tone.md`'s "Empty, loading, error" section — its third
  state finally points at a component; `conformance.md`; relates to [0021]
  (the sibling loading states) and [0002]/[0005]
- **Proposed by** TripTogether (Flutter binding adopter), surfaced by the
  Milestone-5 spec audit (TT-36 tray, TT-37 states)

## Context

`guidelines/voice-and-tone.md` treats **empty, loading, and error** as three
distinct states. Two now have components: `SkEmptyState` owns *empty* ("nothing is
here, do the thing that creates it" — and the system's only dashed border, "here
and nowhere else"), and [0021] shipped the two *loading* treatments (a skeleton and
a labeled fallback). **Error has no component.**

So the rule for the third state — *"error: say what failed, what it means, and the
one next step"* — is stated but unsatisfiable the same way [0021]'s loading rule
was: there is nothing sanctioned to reach for. And the failure mode is worse than
loading's, because an error is not a momentary frame — it *persists* until the user
acts, so a wrong treatment sits on screen.

This is not hypothetical. The Flutter adopter (TripTogether) hand-rolled a
`RetryableError` widget and repeats it across the trips-settings, group, places,
money, and workspace-gate surfaces — the same gap-driven improvisation [0021] cited
for the bare `CircularProgressIndicator`. Its Milestone-5 "consistent empty /
loading / error across five tabs" work (TT-37) cannot be *consistent* by
construction: with no primitive, five surfaces will each grow their own error
markup, copy, and affordance — exactly the drift [0010] and [0013] exist to
prevent, now on the highest-stakes state.

The tempting shortcut is to fold error onto `SkEmptyState` — it already has a
centred frame, a glyph, a title, a message, and an action slot. That is the wrong
instinct, and for the same reasons [0021] refused to fold *loading* onto it.

## Decision

**SeaKim ships an error-state treatment, distinct from `SkEmptyState`.**

### The seam: three promises, not three amounts of chrome

Empty, loading, and error look alike — a centred glyph, a line or two, maybe a
button — and are semantically opposite. They make three different promises:

- **Empty** — *nothing is here; create it.* Static content; the action **makes**
  the missing thing; owns the dashed border.
- **Loading** — *something is coming; wait.* Announces **busy**, then completion;
  offers no action.
- **Error** — *it failed; here's the one way forward.* Announces **assertively**;
  the action **recovers** (retry) or **escapes** (go back), it does not create.

A frame mid-transition can render two of these nearly identically — the worst
confusion at the worst moment (a failed load that reads as "empty, nothing here" tells
the user to create what already exists but didn't arrive). So the error state
**reuses the centred frame's layout tokens** — the same centring and
title/description rhythm as `SkEmptyState` and [0021]'s labeled loading — as a
**separate treatment**, and takes none of `SkEmptyState`'s identity: **no dashed
border** (that belongs to empty alone, per [0021]), and it is **not an
`SkEmptyState` variant**.

### What the treatment is

`SkErrorState`: a centred frame carrying

- a **title** stating the failure and a **message** following the voice-and-tone
  shape — *cause → consequence → next step* ("Couldn't load places. Check your
  connection and try again.");
- an **error-toned glyph** drawn from the token layer (a `danger`/`warning` hue,
  not the neutral glyph an empty state uses) — so the state reads as *wrong*, not
  *absent*, before the copy is read;
- a **recovery action** as the default and defining affordance. The distinction
  from empty is the verb: empty offers *create*, error offers **Try again**. Where
  the error is **terminal** (a 403, a not-found — retrying cannot help), the retry
  is replaced by a **navigational escape** (go back / go home), never left as a
  dead "try again" that will fail identically.

### Accessibility — an alert, not static content

The three states differ in role, and this is not cosmetic: `SkEmptyState` is static
content, [0021]'s loading state announces **busy**/**polite**, and an error must
announce **assertively** (`role="alert"` / an assertive live region) so a user who
just triggered the failed action hears *that it failed* without moving focus. A
binding that renders the error into the same polite region as loading swallows the
one announcement that matters.

### Not a toast

`SkToast` is transient acknowledgement ("Saved", "Copied") and self-dismisses. A
failed *page or region load* needs a **persistent** state the user returns to and
acts on — a toast that fades leaves a blank region behind. Toast remains right for a
failed *incidental action* (a reaction that didn't save); `SkErrorState` owns the
*region/route* failure. Bindings should not substitute one for the other.

## Consequences

- Additive: one treatment added, nothing removed or renamed. **MINOR** per [0011]
  and [0019], for the rules layer and each binding.
- Closes the loop [0021] left half-open: `voice-and-tone.md`'s "Empty, loading,
  error" section now points at a real component for all three states instead of
  stating an unsatisfiable rule for the third.
- **Both bindings land it together** (React + Flutter), like [0021]'s treatments —
  a rules-layer state that exists in one binding is the parity gap [0019] flagged.
- The Flutter adopter replaces its hand-rolled `RetryableError` with `SkErrorState`
  and standardizes the five Milestone-5 surfaces on it; its "consistent error
  state" ticket (TT-37) becomes satisfiable. It stays blocked until this lands
  upstream and the mirror re-syncs.
- Per [0001] it owes a `spec/` file, and per [0020] a preview-surface entry and
  manifest row before it can ship.

## Enforceability

Per [0012], "does this screen have all three states?" and "is this the right one of
the three?" are judgements a static check cannot make — like [0021]'s "is the
skeleton shaped like its content?" What a check *can* assert once the primitive
exists: that a caught async error renders through `SkErrorState` rather than raw
error text with an ad-hoc button, the same shape as the loading check ("no
indefinite rotation outside the sanctioned treatments"). The retryable-vs-terminal
choice stays a manual pass.

## Rejected alternatives

- **Fold error onto `SkEmptyState`.** The most tempting wrong answer, refused on the
  same grounds [0021] refused it for loading: it overloads "nothing is here, create
  it" with "it broke, recover" — opposite copy obligations, an assertive-alert role
  vs. static content, a recovery verb vs. a create verb, and the dashed border that
  must not appear on an error. Reuse the frame's layout tokens, not the identity.
- **Leave it to adopters (hand-rolled error + button).** The status quo, and it
  already produced a bespoke `RetryableError` repeated across five surfaces with no
  shared copy or a11y contract — the drift [0010]/[0013] exist to prevent, on the
  highest-stakes state.
- **A toast for every error.** Transient acknowledgement can't hold a persistent
  region/route failure; the user returns to a blank region with nothing to act on.
  Toast keeps failed *incidental actions*; it does not cover a failed load.
- **Reuse [0021]'s labeled-loading treatment for errors too.** They make opposite
  promises — loading is "wait," error is "it failed, act now" — and share an
  indicator vocabulary that would blur the two at the exact moment (a stalled load
  becoming a failure) they must read differently.
- **One combined `SkStatusState` covering empty + loading + error.** Collapses three
  distinct promises, a11y roles, and border rules into one prop-configured
  component — the semantic-reuse trap [0021] named; a component that is sometimes an
  alert and sometimes static content will be wired wrong.
