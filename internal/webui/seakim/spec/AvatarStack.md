# AvatarStack

More than one avatar in one place: overlapping marks, capped at a maximum, with the
remainder collapsed into a "+k" count. Per [0024](../decisions/0024-avatar-receipts-and-stack.md).

**Not for:** a single person, team, or account — that is an `Avatar`. The stack is the
primitive for two or more; below two it saves nothing and should not be reached for. Not a
"see all readers" control either — the count pill is a count, not a button; a caller that
needs a target wraps the stack.

## A row of overlapping marks with an overflow count

```
  ( A )( B )( C ) +17          <- three avatars, then the remainder as a pill
     \___\___\____ each slides under the one before by --avatar-overlap
  first-in-front: A sits on top of B sits on top of C
```

| Part | Treatment |
| --- | --- |
| Avatar | An `Avatar` at the stack's `size` (default `sm`). The single source of the circle, initials, and status dot — the stack only arranges instances of it. |
| Overlap | Each mark after the first slides under its neighbour by `--avatar-overlap`, one shared constant so every facepile in the system overlaps by the same amount. |
| Ring | Each mark carries a 2px `--surface-card` ring so it reads as separate from the one behind it. Drawn as an outline, so it never changes the diameter. |
| Overflow pill | Past `max` visible marks, the remainder becomes a `+k` count: `--radius-full`, mono figures (per the number-rendering rule), `--surface-sunken` fill, `--text-secondary` ink. Non-interactive. |

Values are tokens, not literals: the overlap is `--avatar-overlap`, the pill corner is
`--radius-full`, the ring is `--surface-card`. Sizes come from `Avatar` (`xs` 20 … `xl` 64).

## The overlap is one shared constant

Every facepile — read receipts, channel members, "who's on this trip" — takes the **same**
`--avatar-overlap`. Encoding it per call site is how two facepiles end up overlapping by
different amounts; it lives where the other spacing values live (`SkSpace.avatarOverlap` /
`--avatar-overlap`), not as a number retyped at each call.

## `max`, and the remainder rule

The stack shows up to `max` avatars. When more remain, the last slot becomes a `+k` pill
where `k` is the count still hidden.

The rule that gets missed: **when the remainder is exactly one, show the avatar, not "+1".**
A pill that replaces a single avatar saves no space and only adds a second shape — so
`max + 1` people render as `max + 1` avatars, with no pill at all. The pill appears only
when it hides two or more.

## Stacking order

`frontToBack` chooses which end sits on top — first-in-front by default. This is a visual
stacking choice, **not** a reading-order assumption: the system inherits no bidi/RTL
position, so the default must not quietly imply one.

## Responsive

At `sm` the stack does not restack — it is already a compact, fixed-height row. What changes
with a tight container is `max`: a facepile in a narrow column carries a smaller `max` (and
therefore a larger `+k`) so the row never wraps or overflows. The `sm` avatar size is the
floor; a stack does not shrink its marks below it, it drops visible marks into the count.

## Accessibility

- The stack announces itself as **one group**, labelled with the total: "N people" — or
  "N people, and k more" when the count overflows. The individual avatars are not exposed
  or focusable separately; the group's label carries the whole set.
- The `+k` pill is a count, not a control: not focusable, no hit area. A caller that wants
  "see all readers" wraps the stack in its own target and labels that.
- Passes the colour-alone check: overlap and count read structurally, never by hue.
