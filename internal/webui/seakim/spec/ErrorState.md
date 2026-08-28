# ErrorState

A region or route that failed — says what broke, what it means, and the one way forward.
Per [0029](../decisions/0029-error-state.md).

**Not for:** *empty* (nothing is here yet → `EmptyState`), *loading* (something is coming
→ the loading states of [0021](../decisions/0021-loading-states.md)), or a failed
*incidental* action (a reaction that didn't save → `Toast`, which is transient).
`ErrorState` owns a persistent **region or route** failure the user returns to and acts on.

## The seam

Empty, loading, and error look alike and are semantically opposite — they make three
different promises, and a frame mid-transition can render two of them nearly identically.
The three must not read alike.

| State | Promise | Announce | Action | Border |
| --- | --- | --- | --- | --- |
| Empty | nothing is here; create it | static content | *create* the missing thing | the dashed border (empty's alone) |
| Loading | something is coming; wait | busy, polite | none | none |
| **Error** | it failed; here's the way forward | **assertive alert** | **recover** (retry) or **escape** | **none** |

## Anatomy

Reuses the centred frame — the same centring and title/description rhythm as `EmptyState`
and the labeled loading state — but takes **none of empty's identity**.

| Part | Treatment |
| --- | --- |
| Frame | Centred column on `--surface-sunken`, `--space-11`/`--space-7` padding (`--space-8`/`--space-6` compact). **No dashed border** — that belongs to empty alone. |
| Glyph | An error-toned glyph in `--text-danger` (a warning/danger hue, not the neutral glyph empty uses), so the state reads as *wrong*, not *absent*, before the copy is read. |
| Title | States the failure. `--type-subheading`, `--text-primary`. |
| Message | Cause → consequence → next step, one sentence each, max (`voice-and-tone`). `--text-secondary`, max ~44ch. |
| Action | The defining affordance. Default is a `Try again` retry (secondary button, refresh glyph). |

## Retryable vs terminal

- **Retryable** (a dropped connection, a timeout): the default `Try again` retry.
- **Terminal** (a 403, a not-found — retrying fails identically): replace retry with a
  **navigational escape** (go back / go home). Never leave a dead `Try again` that will
  fail the same way.

## States

| State | Treatment |
| --- | --- |
| Retryable | Glyph + copy + `Try again`. |
| Terminal | Glyph + copy + a navigational escape, no retry. |
| Compact | Tighter padding, for a panel or card rather than a full page. |

## Responsive

The centred frame fits its container at every breakpoint; on `sm` it fills the width like
the empty and loading states. Nothing is reachable only by hover.

## Accessibility

- **Announced assertively** (`role="alert"` / an assertive live region), so a user who
  just triggered the failed action hears *that it failed* without moving focus — the one
  announcement that matters. Not the polite-busy region loading uses, not the static
  content empty is.
- The recovery action is a real, focusable control; the retry/escape distinction is
  honoured so a keyboard user is never sent to a button that cannot help.
