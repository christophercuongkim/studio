# 0024 — Avatar as read-receipt, and an avatar stack primitive

- **Status** Accepted
- **Date** 2026-08-22
- **Affects** `spec/` (a read-receipt pattern; an `SkAvatarStack` component); every binding

## Context

A group conversation needs to answer "who has seen this?" The idiom the whole industry
settled on is the reader's avatar under the last message their read cursor covers.
SeaKim has no read-receipt pattern, so a binding consumer (chat, again the trigger)
invents one — correctly reaching for `SkAvatar` at its smallest size and reusing
`SkAvatarStatus` for presence, but with no guidance on placement or on what happens when
five people have read the same message.

The second half is the real gap: **there is no avatar stack.** Several avatars in a row
is a bare `Wrap`, which lays them out edge-to-edge and, past a handful, wraps to a second
line or overflows. The universal answer — overlap them, cap the count, show "+k" — has to
be hand-built, and every consumer that shows a facepile (receipts, channel members,
"who's on this trip") rebuilds it slightly differently.

Reusing `SkAvatar` was the right instinct and should be blessed, not re-derived; the
missing piece is the cluster behaviour above one avatar.

## Decision

The read-receipt half is not itself a contested decision — placing the reader's `xs`
avatar under the last message their cursor covers, on the message's side, with presence
via the existing `SkAvatarStatus`, is uncontested usage that composes `SkAvatar`. It is
recorded here as the **motivation**, not as a new rule: it is the surface that exposed
the actual gap. When this ADR lands it becomes a `spec/` note, not a component.

The decision is **`SkAvatarStack`** (React and Flutter), the primitive for more than one
avatar in one place, which receipts and every other facepile currently lack:

- avatars overlap by a fixed fraction of their diameter, held as a **shared constant**
  (an `SkSpace`/`SkRadius`-style value plus its CSS custom property) so the overlap is
  consistent everywhere. Dimensions in SeaKim are hand-authored constants today — there
  is no JSON dimension source yet (0007 phase 2) and 0013 governs alpha, not size — so
  this lives exactly where every other spacing value does, not in a pipeline that does
  not exist;
- a `max` visible count, after which the remainder collapses into a **"+k" count pill**
  (`SkRadius.pill`, mono figures, per the number-rendering rule). When the remainder is
  exactly one, show the avatar rather than "+1" — a pill that saves no space is noise;
- the `max` overflow pill is non-interactive by default (it is a count, not a control);
  a caller that wants "see all readers" wraps the stack, so the primitive stays free of a
  target it cannot always honour;
- last-in-front or first-in-front is a prop. It defaults to visual stacking order, **not**
  a reading-order assumption — the system has no bidi/RTL position to inherit, so the
  default must not quietly imply one.

Read receipts, channel-member clusters, and any "these people" facepile use
`SkAvatarStack`; none re-implements overlap-and-overflow.

## Consequences

- Receipts and facepiles share one behaviour and one overlap token; they stop drifting.
- A message read by twenty people shows three avatars and "+17", not twenty avatars
  wrapping down the screen.
- `SkAvatar` stays the single source of the circle, initials, and status dot; the stack
  only arranges instances of it.
- `SkAvatarStack` is a new component: per 0001 it owes a `spec/` file, and per 0020 a
  preview entry and manifest row before it ships.
- Versioning (0019): **Minor**. A component and a spec pattern are added; no rule
  changes.

## Rejected alternatives

- **Leave receipts to consumers.** The pattern is universal enough to standardise, and
  the bare-`Wrap` overflow is a real bug the first time a message is popular.
- **A receipts-specific component.** The hard part — overlap, cap, overflow count — is
  identical for any facepile. Solving it once as `SkAvatarStack` serves receipts and
  member clusters; a receipts-only widget would grow a member-cluster twin within a
  release.
- **Encode the overlap as a per-call constant.** That is how two facepiles end up with
  different overlaps. It is one shared constant, defined where `SkSpace`/`SkRadius` live,
  not a number re-typed at each call site.
