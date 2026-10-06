# Adversarial review: WOPR plan v1

_Reviewed: `docs/PLAN.md` at commit `ada63a0` ("Add WOPR architecture and scope plan"). Review date
2026-10-06, revised the same day after the review itself was checked._

This review attacks the plan on purpose. It looks for claims that are false, designs that will break under
real use, ordering mistakes that cause rework, and places where the plan contradicts itself. Line numbers
(L123) refer to v1 as printed by `git show ada63a0:docs/PLAN.md`, and milestone numbers are v1's: "M6" here is
the optional LLM brain, which v2 renumbers to M7 after the owner added movie mode as M6. Every external fact the review relies on
was checked against upstream source at a named version, official documentation, or a local experiment; the
**Verification log** at the end lists each check under an ID (for example BT7) that findings cite. The
updated plan (`docs/PLAN.md` v2) answers each finding; `PLAN-v2-review.md` will review that version.

**Changes since the first draft.** The draft was itself attacked: a skeptic pass per section tried to refute
each finding, fact-check passes re-ran every external claim, and completeness passes looked for gaps.

- Withdrawn or corrected claims: T-1 (v1's `prek.toml` passes `prek validate-config`; named priorities are
  valid), B-3 (Git Bash), B-4 (attack vector), U-4 (stdin), A-11 (Go upgrades) and G-1 ("slow"). Fixes that
  did not work are replaced, notably T-2, S-1, B-6, C-1, G-5, Q-1, Q-3 and M-1.
- 30 findings were re-graded, all downward; each heading shows the old grade. 31 are new: A-13–A-17,
  U-7–U-11, D-5, D-6, B-9–B-14, T-8–T-11, S-7–S-10, L-4–L-6, P-4, and F-1 in a new section 14.
- The verification log is filled in, and a second round of owner decisions is recorded.

Findings use this severity scale:

| Level | Meaning |
|---|---|
| **Blocker** | Fails as written, or forces a rewrite later if not fixed before the code that depends on it. |
| **Major** | Real risk to correctness, security, schedule or maintenance. Fix in the plan now. |
| **Minor** | Worth fixing; cheap. |
| **Nit** | Wording, consistency, polish. |

## Verdict

The plan is far better than most hobby-project plans. Its strongest parts are the import rule that keeps
games and the persona free of Bubble Tea, the single clock, deterministic seeds for golden tests, and the
public-repo hygiene. The CLI surface and size budget are concrete and testable.

It is not ready to implement as written, but the first draft overstated how far off it is. Checking the
review withdrew or downgraded these headline claims. v1's `prek.toml` does pass `prek validate-config`; only
a placeholder rev and one hook id fail, at hook init (T-1, Blocker → Minor). v1 contradicts itself on Board
input rather than forcing per-game line editors, and Lip Gloss is not banned in games (A-1, A-2, Blocker →
Major). T-2, S-1 and P-1 are real but cost one config line, one CI step or one milestone move (Blocker →
Major). And `notnil/chess` is fast enough (SZ4); the issue is that it is archived (G-1, Major → Minor).

The real problems, in order:

1. **Blocking AI on the UI goroutine (A-3, Blocker).** A 4-ply chess search inside `Handle` takes 0.1–1.7 s
   (SZ4), during which nothing renders and Ctrl+C waits in the queue.
2. **A data race designed into the brain call (A-4, Blocker).** `Brain.Reply(ctx, *Session, …)` runs on a
   `tea.Cmd` goroutine while `Update` mutates the same session (BT7).
3. **Nothing can wire the games together (A-13).** The registry cannot import the game packages without an
   import cycle, and no other package may.
4. **The game protocol cannot express the film (A-1, A-15, A-16, A-17).** Input routing is tied to `Kind`;
   there is no clear, wait, layout change or empty Enter; nobody owns the climax; LOGON has no dispatch.
5. **CI would not build with the pinned Go (D-6).** `GOTOOLCHAIN=local` set before `actions/setup-go` makes
   it ignore `toolchain go1.27.1`.
6. **Release integrity (B-10, B-11).** Tags are rebuilt and published with no tests, smoke run or size gate,
   and the archives lack `NOTICE.md` and third-party licence notices.
7. **Third-party text already in v1 (L-4):** the GTW mockup map, the backdoor header and the montage list.
8. **Checks that never fire (T-2, S-1).** The draft's fixes for both also failed.

None of this argues for a different stack. Bubble Tea v2 + Lip Gloss v2 remains the right choice: about
3.6–3.9 MiB stripped across the six targets (SZ1); a stdlib HTTP client for M6 adds about 4.3 MB (SZ3); the
worst case with chess and HTTP together is about 8.3 MiB (SZ2, SZ3), inside the 10 MB target. The rest are
Major, Minor and Nit findings below.

## 1. Requirements and traceability

**R-1 (Major) — Stated requirements and decisions disagree; the provenance header is wrong.** The Context
says "Linux + Windows, **< 10 MB**" (L14); decision 5 adds macOS (L215); decision 10 makes 10 MB a target
with a 15 MB hard limit (L220). The header says "1–4 confirmed by the user in Q&A; 5–7 are routine
defaults" (L207), but the table has 12 rows and row 5 is labelled "(requirement)". A reader cannot tell
what the owner confirmed, and "< 10 MB" quietly became a warning. *Fix:* list requirements once with a
source each (user / derived / default), record changes in the decision log, and label a row "user" only
if the owner stated it. *Resolved for the size limit:* the owner kept the 10/15 MB split (round 1, Q5).

**R-2 (Minor) — Requirements with no acceptance criteria.** "Persona faithful to the film" and "every game
the film's WOPR offered" (L11-13) are not testable. *Fix:* give each game a definition of done: rules
implemented, AI plays legally, ends with a verdict, and a testkit transcript. These test consistency, not
fidelity; fidelity still needs the viewing pass (F-1).

**R-3 (Nit) — One path for the plan.** Refer to `docs/PLAN.md` by that path everywhere, and link it from the
README once one exists.

## 2. Architecture

**A-1 (Major, was Blocker) — Input routing is tied to `Kind`, and v1 contradicts itself.** Keys go "to game
(Board/Fullscreen) or input line" (L375), and `LineEvent` is for Teletype (L313). Yet the Board layout has a
"console strip (last N lines + input)" (L334), and `Prompt` shows `YOUR MOVE:` (L319). Chess is Board with
typed moves (L429-430). Fullscreen is "View only" (L335), yet GTW asks `PLEASE CHOOSE ONE:` and asks for
targets (L68-70). Gin Rummy and Hearts fit neither Kind (L424-425). The fix is in the host, not in each game.
*Fix:* split the axes. A static `Layout` (Console, Panel, Fullscreen) says where `View` goes. The input mode
is set by output: `Prompt{Text}` delivers `LineEvent`s, a new `AwaitKeys{}` delivers `KeyEvent`s (maze,
board cursor), and a game may declare both (checkers). The host owns one line editor. Layout changes are A-15.

**A-2 (Major, was Blocker) — Games have no theme-agnostic way to express colour.** The draft's premise was
wrong: v1 bans only Bubble Tea in `games/` (L231-232), and goldens strip ANSI before comparing (L486). The
real problem: with `View(w, h) string` (L327), a game must import Lip Gloss and the active theme, against the
stated direction (L231), or give up the big board's tracks, DEFCON colours and blinking subs (L146-147,
L393-394); and goldens cannot assert colour. *Fix:* games draw into a rune grid with **semantic** style tags
(`Text`, `Bright`, `Dim`, `Accent`, `Incoming`, `Outgoing`, `Defcon5…1`, a red-suit tag) and separate
attributes (`Blink`, `Reverse`, `Bold`), in a package that games, the persona and `ui` screens can all import
(v2's `proto`), not `games/canvas`. `ui` maps tags to the theme; goldens assert on tags. State that a visible
`Blink` cell keeps the clock ticking, an exception to "idle = no timer" (L367).

**A-3 (Blocker) — Synchronous `Handle` blocks the event loop during AI search.** `Update` runs inline in the
event loop, frames render only after it returns, and Ctrl+C and SIGINT go through the same loop (BT10).
While a search runs inside `Handle`, nothing renders and Ctrl+C is queued (delayed, not lost). With
`notnil/chess`, 4 ply plus quiescence takes a median 106 ms per move, but 1.09 s at p99 and 1.69 s at worst
on a 4-vCPU server (SZ4); checkers at depth ~6 (L428) has the same shape. *Fix:* add an output
`Think{Fn func(ctx) Event, Budget, Limit}` that the host runs in a `tea.Cmd` with a cancellable context,
showing a thinking signal that differs from idle (the cursor already blinks; BT3). Keep play deterministic:
a wall-clock budget makes the move depend on CPU speed, load and `-race` (X11), so with `--seed` and in tests
bound the search by depth or nodes (`Limit`), with wall-clock time only as a cap whose expiry fails the test.
Use a deadline for the budget and a cancel cause for Esc, so `Fn` can tell "move now" from "cancelled".

**A-4 (Blocker) — `Brain.Reply(ctx, *Session, …)` from a `tea.Cmd` is a data race and an ordering bug.**
*Race:* Cmds run on their own goroutines while `Update` keeps running (BT7); the scripted brain's `Once`
rules write `s.Said[ID]` and `Pick` reads `s.Rand` (L354-355). Concurrent map writes are a fatal error that
`recover` cannot catch (X29), leaving the terminal raw. The window is tiny for the scripted brain and certain
for an LLM, and the synchronous golden driver (L484-485) never shows it. *Ordering:* the brain answers
asynchronously, so a second line typed early comes out of order, and if the "sequence guard" (L344,
undefined) drops stale replies, the first line gets no answer. *Fix:* `Reply(ctx, Snapshot, input)`, with a
value-copy `Snapshot` and a generator seeded from `(seed, turn)`, where `turn` increments on every call,
retries included. `Reply` carries `Effects` (mark-said, set-flag, start-game) that `Update` applies. While a
reply is pending, keys go into the line editor but Enter does not submit (U-3); Esc cancels the context.

**A-5 (Major) — Two state machines own "phase".** `ui/app.go` has phases Connect→…→Ending (L240),
`wopr/session.go` holds `phase` (L248), and rules gate on `Phase` (L352); two copies will drift. *Fix:* the
persona owns the conversation phase, including the dial and disconnect states (A-17). `ui` owns only the
presentation mode and the active `Game`, and derives the mode from session state; `Done` is the sync point.

**A-6 (Major) — The import rule is stated but not enforced, and it is incomplete.** `wopr` needs the
registry (L251-252) and `cli` needs `games.Resolve` (L236), but neither edge is in the stated direction
(L231), and `sim` (L266) appears in none. Only AGENTS.md, written last (L836), records the rule. *Fix:* write
the full DAG and enforce it from M0: `cmd/wopr → cli, ui, version`; `wopr → games, proto`;
`cli → games, games/catalog, theme, version`; `games/catalog → games/<game>`;
`ui → wopr, games, games/catalog, theme, assets, proto, charm.land/*, x/ansi`;
`games/<game> → games, proto, games/{ai,cards,board,prompt}, sim, assets`. Only `ui` and `theme` (Lip Gloss
palettes, L238) import `charm.land/*`. A depguard rule denying Bubble Tea outside `ui` works (GL4; set
`issues.max-same-issues: 0`, since the default of 3 hid a hit). For the whole DAG, an archtest reads each
package's `Imports` and `TestImports` (`go list -deps` gives the transitive set, not edges).

**A-7 (Major) — Game-name detection "anywhere in the text" will misfire.** "I don't want to play chess",
"the Golden Gate **bridge**", "my **hearts** not in it" and "**poker** face" all launch games (L251-252,
L290), and it pre-empts a future LLM brain. *Fix:* start a game only on explicit intent, per clause. Split
on `. ! ? ;` before normalising, which drops punctuation (L249). A clause matches if, after optional fillers
(LATER, LOVE TO, OK, YES, WELL), it starts with PLAY, LET'S PLAY, LETS PLAY, LET US PLAY, HOW ABOUT or I WANT
TO PLAY plus a game, with no negator (NOT, DON'T, DONT, NO, NEVER) earlier in the clause. An exact game name
or alias also counts, except at LOGON (A-17). WOPR's offers of a named game (`… GAME OF CHESS?`) arm a
one-turn offer that YES, OK, SURE or LOVE TO accepts; a numbered list arms selection until the next
non-numeric input. Tests: "Love to. How about Global Thermonuclear War?" and "Later. Let's play Global
Thermonuclear War." must match; "I don't want to play chess" must not. GTW → chess stays a scene.

**A-8 (Minor) — `KeyEvent.Key` is "tea's `String()`" (L312).** Bubble Tea v2 already names Space `"space"`
where v1 used `" "` (X6), so games comparing strings break silently. *Fix:* games get their own key enum
(`Up`, `Down`, `Left`, `Right`, `Enter`, `Backspace`, `Rune`; Space is `Rune ' '`). The host maps to it and
never delivers Esc (L335-336).

**A-9 (Minor) — `Animate{Every}` vs. the single clock is underspecified.** The clock ticks at the fastest
consumer's rate (L364-368), so does a 500 ms animation get a `TickEvent` every 33 ms? *Fix:* accumulate `dt`
per consumer and arm the next tick at the earliest pending deadline; a tick at the minimum period drifts when
periods are not multiples. Re-arm (bump the generation) when consumers change. After a late or suspended
tick, deliver one `TickEvent` with the summed `Dt`, not a burst.

**A-10 (Minor) — Script lines in `lines/*.txt` (L255) imply keyed runtime lookups** that need a parser and
fail late, and scenes are already split between Go and text (L250). Unless non-programmers edit the lines,
Go string constants are simpler and compile-checked. Then exclude `lines.go` and `internal/assets/` from
golangci-lint's `misspell`, which checks string literals by default (X15). If `.txt` stays, test that every
referenced key exists.

**A-11 (Minor) — RNG package and seed derivation.** `Env.Rand *rand.Rand` (L309) and "seeded `*rand.Rand`"
(L248) do not say `math/rand` or `math/rand/v2`, whose sequences differ. The draft's caveat about Go upgrades
was wrong: both packages guarantee seeded sequences across releases (X3). The real hazards are that top-level
`math/rand/v2` functions (`rand.IntN` …) cannot be seeded and that `*rand.Rand` is not goroutine-safe.
*Fix:* `math/rand/v2` PCG with explicit seeds. `rand.NewPCG(seed1, seed2)` takes two seed halves, not a
"stream" (X3), so derive per-consumer seeds with disjoint domains: the brain by turn, a game by slug and play
count (a second Black Jack deals a fresh shoe). Ban the top-level functions outside one helper with
forbidigo. A `Think` function that needs randomness builds its own generator from a seed taken in `Handle`.

**A-12 (Nit) — `Pace` is used in `Say` (L318) but never declared;** its values exist only in prose
(L368-369). Declare `type Pace uint8` with `PaceSpeech`, `PaceTable` and `PaceInstant`. The draft's `Outcome`
and `Action` points are withdrawn: L330 names the outcomes and L341 carries the payloads.

**A-13 (Major, new) — Games cannot be wired into the registry without an import cycle.** The registry lives
in package `games` (L258) with 16 entries plus a stub (L823-824), and each `games/<game>` imports `games`
for `Game`, `Info` and `Env` (L304-331), so `games → games/chess` is a cycle (X10). The stated direction
(L231) lets no other package import the games, and `init()` registration still needs a blank import from
somewhere. `internal/assets` (L268) is reachable from nothing, `games/testkit` (L263) and `ui/testutil`
(L246) are unplaced, and each game's `Info()` (L324) duplicates its registry entry. Major, not Blocker:
M0–M1 need only `Info` entries. *Fix:* `internal/games` holds types only (`Info`, `Game`, `Env`, `Registry`,
`Resolve`); a new `internal/games/catalog` imports every game and returns the ordered `{Info, New}` list;
drop `Info()` from `Game`. `ui` and `cli` import the catalog, `wopr` gets a `Registry` injected (fakes in
tests), and games may import `assets`. archtest also checks test imports, which `go list -deps` ignores
(X10), and a test asserts that every playable entry has a constructor.

**A-14 (Major, new) — Host semantics live only in `ui/gamehost`, so testkit must re-implement them.**
`gamehost` (L243) owns input routing (L375), Esc → `Done{Aborted}` (L335-336), `Animate` cadence (L366) and
output order. `games/testkit` (L263) cannot import `ui`, which imports Bubble Tea, so it must copy those
rules, and tests then check the copy. The golden driver "runs returned cmds" (L246, L484-485), but
`tea.Tick` starts its timer when called (BT11) and a Cmd is opaque, so the driver cannot tell a clock Cmd
from a Think Cmd without running it in real time. *Fix:* in M1, a host core with no Bubble Tea import: a
state machine over events and host signals (key, Esc, `Tick(dt)`, resize, think result) with an injected
scheduler. `ui` adapts it (`clock.go` implements the scheduler); testkit and the golden driver use a fake
scheduler. Define Esc there: during a reveal it only skips; otherwise the first Esc shows an in-character
confirm that another key or a few seconds clears; a confirmed abort cancels any Think and drops its late
result; Esc does nothing at the shell and cannot abort the ending (Ctrl+C still quits).

**A-15 (Major, new) — The game protocol cannot clear, wait, change layout or deliver an empty Enter.** v1's
outputs are `Say`, `Prompt`, `Animate` and `Done` (L318-321). The film needs a *clear* after the backdoor
header (L63-64) and, per the fan transcripts, after `FINE.`, the side choice and target entry (EC6); a
*wait* after `Joshua`; a *layout change*, because `Kind` is fixed (L308) while GTW's menus are text screens
and its map is Fullscreen with only a bottom strip for `Say` (L335); and an *empty Enter*, which ends the
target list, though `LineEvent` is "a submitted line, trimmed" (L313). *Fix:* add `Clear{}` (a page break;
scrollback keeps the history above it), `Wait{D}` (host-timed, skippable, zero under `--instant`) and
`SetLayout{}` (`Info` sets only the initial layout). While a `Prompt` is active, deliver an empty Enter as
`LineEvent{Text: ""}`. Mock up each GTW phase at 80×24 (U-5).

**A-16 (Major, new) — The ending has no single owner.** GTW ends with "the montage and `A STRANGE GAME…`"
(L443-445), tic-tac-toe's `NUMBER OF PLAYERS: 0` starts "the self-play montage into the scenario montage"
(L446-447), and `ui` has scripted ending screens too (L298-299), so the montage would be built twice. The
film's chain cannot be expressed: a game can only emit `Done` (L321), so GTW cannot hand off to tic-tac-toe;
the NORAD notices (L82-83) have no state to print in, such as while GTW runs; an Animate-driven ending has
no `Prompt` for the user's `HELLO` before `A STRANGE GAME…` (fan transcript, F-1); and the persona's verdict
(L294-295) would print after `HOW ABOUT A NICE GAME OF CHESS?`, the film's last line. `Ending` (L240) has no
exit. *Fix:* one `ending` program (for example `games/ending`, which may import `assets`) with explicit states
and prompts. GTW and tic-tac-toe end with a `Result` naming the next step (`Next: Ending`), which can also
say "no verdict". Define `Ending → Shell` with the chess offer armed (A-7); decide what 0 players does from
a normal shell.

**A-17 (Major, new) — LOGON has no defined dispatch.** The dispatch order (L289-292) covers "every shell
input"; the LOGON gate (L53-55, L214) has a hint but no rules. If the shell chain runs at LOGON, game-name
detection starts Falken's Maze when the user types it as a login name, as David does in the film (EC6), and
`LIST GAMES` then `7` starts chess before login. Nothing owns `--CONNECTION TERMINATED--`, the re-dial or a
failure counter that survives disconnects, and neither the skip policy (L373) nor `--instant` (L369) covers
the dial animation. *Fix:* LOGON is a closed table: HELP LOGON → `HELP NOT AVAILABLE`; HELP GAMES; LIST
GAMES (arms no selection); JOSHUA, trimmed, any case → backdoor; empty → re-prompt; anything else →
`IDENTIFICATION NOT RECOGNIZED BY SYSTEM`, `--CONNECTION TERMINATED--`, re-dial. No intent parsing or brain
at LOGON. The session owns the dial states and the counter (A-5); any key or `--instant` skips the dial. Add
goldens `logon_game_name_is_not_a_login` and `logon_list_then_number`.

## 3. Terminal UI

**U-1 (Minor, was Major) — No alternate screen, background or exit-screen policy.** v1 never says whether it
uses the alternate screen, though it describes a full-screen app (L182, L377, L381-382). Without it the
console scrolls shell history, and the `#000000` background paints only written cells. *Fix:* set
`View.AltScreen` (BT13). Paint every cell, or set `View.BackgroundColor`, which sets the terminal's default
background (OSC 11) and resets it on exit (OSC 111), even after a panic and even under `NO_COLOR` (BT8, BT13).
Print `--CONNECTION TERMINATED--` on exit, and say whether error and Ctrl+C exits do too. The wheel is U-9.

**U-2 (Minor, was Major) — No sanitisation of text that did not come from the binary.** Pasted text
(`PasteMsg`, L374) and a future LLM reply are echoed into the view. Bubble Tea v2's renderer already drops
most sequences (OSC 52, cursor movement, BEL, BS) but passes SGR and OSC 8, so styling and links can be
spoofed, and an unterminated OSC blanks the rest of the frame (X5); `ansi.Strip` keeps C0 bytes (BT14). A
correctness issue now, a security requirement in M6 (S-4). *Fix:* `SanitizeInput`: `ansi.Strip`, then drop
C0, C1 and DEL, map tabs and newlines to a space (or keep a paste's first line; pick one and test it), replace
invalid UTF-8, cap at about 256 grapheme clusters. `SanitizeText` for output: expand tabs
(`ansi.StringWidth("\t")` is 0, Lip Gloss renders 4 spaces) and split `\n` into logical lines. Fuzz both.

**U-3 (Minor) — The skip policy throws away typeahead.** "Every `KeyPressMsg` … flushes the whole queue
and is swallowed" (L373-374), so a `h` (hit) typed while WOPR prints is lost and PgUp flushes instead of
scrolling. *Fix:* the first key flushes; printable keys also reach the input line; Enter and Space are pure
"skip" only when the line is empty; PgUp/PgDn always scroll; in key mode a key both flushes and reaches the
game. Use the same rule while a reply or Think is pending (A-4), and update "any key skips" (L776).

**U-4 (Minor) — TTY detection.** Refusing when stdout is not a terminal (L285-286) is right; the draft's
advice to also check stdin was wrong. Bubble Tea opens `/dev/tty` or `CONIN$` itself when stdin is not a
terminal, so `wopr </dev/null` works; when stdout is not one, it runs anyway at 0×0, writing escapes into the
file (BT9). `NO_COLOR=yes` is ignored, because it is parsed with `strconv.ParseBool` (BT8). *Fix:* refuse
when stdout is not a terminal or `TERM=dumb` (the renderer still writes cursor and mode sequences); map
Bubble Tea's `error opening TTY` to the same message, exit 2, and on Windows suggest Windows Terminal. Treat
any non-empty `NO_COLOR` as `tea.WithColorProfile(colorprofile.Ascii)`, as no-color.org specifies.

**U-5 (Minor) — 80×24 is tight for Board layouts.** A chess board takes about 18 rows with a grid and about
10 without; the anti-chrome rule (L182-183) argues for no grid. Mock up chess, Hearts and each GTW phase at
exactly 80×24 in M1, with double-spaced WOPR lines in the strip, before committing to the layout.

**U-6 (Nit) — Custom input line "~100 lines" (L379).** Wide runes, history and paste need about 250 lines
and a fuzz target. Measure width the way the renderer does: Bubble Tea uses wcwidth unless the terminal
confirms grapheme mode 2027, while `ansi.StringWidth` is grapheme-based; a family emoji is 2 cells one way
and 6 the other (X12). Backspace and the length cap work on grapheme clusters.

**U-7 (Major, new) — Colour downsampling and NO_COLOR collapse styles that carry meaning.** The themes
(L386-395) rely on automatic downsampling. In the 16-colour profile (`TERM=xterm`, `TERM=linux`), imsai Text
`#DCE6F0` and Accent `#9EC5FF` both become ANSI 12, and norad incoming `#FF3B30` and outgoing `#FFD60A` both
become ANSI 9 (X7); `green` and `amber` give Accent the same hex as Text even in truecolor (L389-390). Under
`NO_COLOR` only attributes survive, so tracks and DEFCON levels look alike, as they do with colour-vision
deficiency, and `NO_COLOR=yes` is ignored (BT8). Nothing caps flashing: subs blink (L394), DEFCON 1 is white
on red, the montage accelerates (L85-86). *Fix:* per theme and semantic style, define a truecolor hex, an
explicit ANSI-16 index and an ASCII-profile attribute; styles that carry meaning also differ by glyph or
attribute (incoming `*`, outgoing `+`, current DEFCON in reverse video). Detect `NO_COLOR` as in U-4. Cap
flashing at 3 Hz (WCAG 2.3.1; X7), add a reduced-motion option (flag and env var, per decision 6) that stops
blinking, panel lights and montage acceleration, and add style-map goldens under the ANSI and ASCII profiles.

**U-8 (Major, new) — TERMINAL TOO SMALL keeps the session running blind.** Below 80×24 v1 shows the card
"but keep processing ticks" (L381-382), so the typewriter, dial, DEFCON ladder and montage advance unseen,
and keys under the card are unspecified, so a user can submit a LOGON name or chess move blind. `--play`
(L296) can start a game before the first size message or below the minimum, giving `Env` a zero or
negative area (L309); a fresh pty is 0×0, where Bubble Tea renders nothing (X8). *Fix:* while too small,
pause the typewriter and `Animate`, hold any Think result, and drop every key but Ctrl+C. Never start a game
or send a resize below the minimum. Add an e2e case that starts at 60×20, resizes to 80×24 and resumes.

**U-9 (Minor, new) — The mouse wheel would cycle input history.** On the alternate screen with mouse
tracking off, many terminals (at least VTE-based ones such as GNOME Terminal) send ↑/↓ for the wheel;
Bubble Tea neither sets nor clears that mode (DECSET 1007), and v1 binds ↑/↓ to history (L379; X9).
*Fix:* enable cell-motion mouse mode and map the wheel to scrollback (selection then needs Shift+drag), or
reset mode 1007 at start, or document it. Check it in the M5 terminal pass.

**U-10 (Minor, new) — Exit semantics.** v1 has a `LOGOFF/EXIT` command (L251) and a `Logoff` action (L340)
with no exit code, and inside a game line input goes to the game, so the practical way out is Ctrl+C: exit
130 (L414-416), a "failure" under `set -e`. Ctrl+Z does nothing; Bubble Tea leaves `tea.Suspend` to the
program. SIGTERM, or closing a Windows console, makes `Run` return nil, so exit 0; Ctrl+C sets both
`ErrInterrupted` and `ErrProgramKilled` (BT5, BT10). *Fix:* the host handles LOGOFF, LOG OFF, BYE, QUIT and
EXIT in every phase, before routing a line to a game: print `--CONNECTION TERMINATED--`, exit 0. Ctrl+C stays
130; Ctrl+Z uses `tea.Suspend` on Unix; decide and document 0 or 143 for SIGTERM; `main` tests
`ErrInterrupted` before `ErrProgramPanic`. The e2e test covers LOGOFF → 0 and Ctrl+C → 130.

**U-11 (Minor, new) — Print-and-exit output is not downsampled and ignores NO_COLOR.** `--games`, `--help`
and errors print "without touching the terminal" (L285-286). Lip Gloss styles printed through `fmt` send
truecolor escapes into pipes, because `Style.Render` does not downsample; only Bubble Tea's renderer and Lip
Gloss's writer helpers do (BT8), so "automatic color downsampling" (L190) is loose. *Fix:* print through
`colorprofile.NewWriter(os.Stdout, os.Environ())` or `lipgloss.Fprintln` with U-4's `NO_COLOR` rule, or
print plain text.

## 4. Games and AI

**G-1 (Minor, was Major) — `notnil/chess` is archived.** It was archived on 2025-01-25; its last release is
v1.10.0 (2024-11-25), and its README points to a fork (EC1). The draft's "slow as a search back-end" is
withdrawn: perft(4) runs at about 1.7 M nodes/s, and 3–4 ply plus quiescence needs no custom generator
(SZ4). *Fix:* use the maintained MIT fork `github.com/corentings/chess/v2` (v2.6.0, 2026-08-17): it declares
`go 1.25.0`, has a nearly identical API (`ValidMoves` returns `[]Move`) and measured slightly faster (SZ4).
`*chess.Position` is not goroutine-safe, because `ValidMoves` fills a cache lazily (X4), so `Think` captures a
FEN (or the move list) and rebuilds the position inside `Fn`, off the UI goroutine (A-3).

**G-2 (Minor, was Major) — The `sim` engine is named but not specified.** v1 plans one shared engine
(decision 3, L213; L266-267), and some sims have limits (L435, L441-442), but victory conditions, combat
resolution and balance are missing. *Fix:* specify the engine contract as the first M4 task: map model, unit
table, action set, combat results table, events, turn limit, victory rule and kill-ratio generator, with each
sim as a scenario (data plus a few hooks). Fighter Combat (a dogfight, L432-433) and Air-to-Ground (sortie
planning, L437-438) do not fit a region map: let a scenario replace the movement model, or build them as
bespoke games that reuse only the combat table and kill ratios.

**G-3 (Minor, was Major) — Bridge scope.** v1 already marks Bridge "first to cut" (L426-427), and the owner
chose a minimal variant (round 1, Q2). As worded, "WOPR bids for all four seats and the user plays declarer"
is inconsistent: an honest auction often ends with East/West declaring, or with four passes. *Fix:* the user
always plays the declaring side's declarer and dummy, with seats rotated as needed, and WOPR defends; a
passed-out deal is redealt. WOPR's two-seat defence belongs in the estimate.

**G-4 (Minor) — Tic-tac-toe numbering.** `--games` shows it under "ALSO AVAILABLE" (L446-448), but nothing
stops `wopr -p 16` resolving the unlisted entry. *Fix:* resolve unlisted entries by name, slug or alias
only. Do not encode "unlisted" as `Number` 0, the zero value a forgotten field also has; use an explicit
`Listed bool`, or test that numbers 1–15 each appear exactly once.

**G-5 (Minor) — `stub` game in the registry.** M0 registers a stub (L823-824) for `--play` (L815) and the
`play_stub` golden (L487), so it would show in `--games` and `Resolve`, prefixes included. The draft's fix
fails: a stub in `registry_test.go` compiles only into package `games`' own test binary, so `ui` tests cannot
see it, and a build tag forces `-tags` onto every `go test`. *Fix:* inject the registry into `ui` and `cli`
(A-13), and put the stub in a non-test helper package (for example `internal/games/gamestest`) that only
tests import; archtest allows that test-only edge.

**G-6 (Nit) — Falken's Maze "re-routes walls based on the player's habits" (L420-422).** Define a habit
(turn bias, for example) and the guarantee: after every re-route the exit is reachable from the player's
current cell, no wall lands on the player, and revealed cells stay consistent or the change is announced.
Test the "exit found only after WOPR comments" gate too.

## 5. CLI

**C-1 (Minor, was Major) — `wopr gtw --instant` ignores `--instant`.** Stdlib `flag` stops at the first
positional, leaving later flags in `Args()` (X1), and v1 makes `wopr <game>` equivalent to `--play` (L216,
L405). Whether the flags are then dropped or rejected depends on code v1 does not specify. *Fix:* in
`cli.Parse`, take one remaining positional as the game and parse the rest again, or reject extra
positionals with exit 2. A re-parse loop must stop at `--`: `FlagSet.Args()` does not say whether parsing
stopped at a positional or at `--`, so a naive loop turns `wopr -- gtw -i` into instant mode. Detect the
terminator (`in[consumed-1] == "--"`) and test `-t -- gtw`, where `--` is a flag value. Test both orders.

**C-2 (Minor) — Combined short flags and attached values are rejected.** Stdlib `flag` rejects `-is 1` and
`-pchess` and accepts `-p=chess`, `--play=chess` and `-games` (X1). Show the accepted forms in the help and
pin them in a test, including `-i=false` versus `-i false` (the latter is `-i` plus a positional).

**C-3 (Minor) — Exit codes and closed pipes.** Keep 0/1/2/130. v1 requires `wopr --games | head -1` to work
(L857) without a mechanism. On Unix, Go raises SIGPIPE on a write to a closed stdout unless the signal is
ignored (25–35 of 500 runs exited 141 with one write per line; X2); on Windows the error is a raw `Errno`
that `errors.Is(err, syscall.EPIPE)` does not match. *Fix:* on Unix, `signal.Ignore(syscall.SIGPIPE)` (or
one buffered write) and treat EPIPE as success; on Windows, treat `ERROR_NO_DATA` (232) and
`ERROR_BROKEN_PIPE` (109) as success.

**C-4 (Nit) — `-v` for version (L216, L401).** Many tools use `-v` for verbose; say it is deliberate. Do
not then pick `-V` for verbose: curl, ssh and python use `-V` for version.

## 6. Dependencies and toolchain

**D-1 (Minor, was Major) — The `go` directive is the newest release, not the minimum.** `go 1.27` (L218,
L230) is above the stack's floor of 1.26.0 (BT1). The impact is small: wopr has no importable packages, and
with the default `GOTOOLCHAIN=auto`, `go install …@v0.1.0` fetches the newer toolchain itself (X13); only
`GOTOOLCHAIN=local` builders on Go 1.26 are blocked. *Closed by owner decision:* keep `go 1.27` +
`toolchain go1.27.1` (round 2, e). The draft's "CI still builds with 1.27.1" was false under v1's CI (D-6).

**D-2 (Minor, was Major) — The future LLM client must not pull in an SDK.** A stdlib client adds about
4.3 MB (SZ3); an official SDK adds about 4.5 MB more (X14) and would exceed the 10 MB target. *Fix:* state
now that M6 uses `net/http` + `encoding/json` only, against an OpenAI-compatible and/or Anthropic Messages
endpoint. No build tag: a tag doubles the artifacts or ships a release without the feature. The size gate
covers the growth; S-9 covers hardening.

**D-3 (Nit, was Minor) — Drop the tcell fallback.** A tcell v2 hello world is only about 1.2 MB smaller than
the Bubble Tea one (X14), against about 11 MB of headroom, and switching means rewriting `internal/ui`. v1
never called it cheap (L211, L461-462). Keep a one-line note.

**D-4 (Minor) — Version reporting.** `go install …@v0.1.0` builds without `-X Version` and without VCS
stamping (X13). *Fix:* `version.Version` falls back to `debug.ReadBuildInfo().Main.Version`, then to
`(devel)`; commit and date print as `unknown` when absent. Normalise the `v` prefix, which GoReleaser's
`{{.Version}}` lacks and `Main.Version` has.

**D-5 (Major, new) — Go 1.27 requires macOS 13, not 12.** v1 says "macOS 12 Monterey and newer … Go 1.27's
darwin support floor" (L215, L468). Go 1.27 requires macOS 13 Ventura, and its linker stamps a 13.0.0
minimum (PL1). No hosted runner can test 13: `macos-13` was retired in December 2025, and `macos-14` is
unsupported from 2026-11-02 (PL3). v1's "or a Homebrew tap" (L468) would also need signing, a
non-`GITHUB_TOKEN` secret and a `brew trust` step (EC3). *Resolved by the owner:* macOS 26 only (round 2, a).
Test on `macos-26` and `macos-26-intel` (B-5) and say macOS 26 in the README.

**D-6 (Major, new) — `GOTOOLCHAIN=local` before `actions/setup-go` drops the toolchain pin.** v1 runs
setup-go "with `go-version-file: go.mod` and env `GOTOOLCHAIN=local`, so a toolchain mismatch fails loudly"
(L709-710). setup-go reads the `toolchain` line only when `GOTOOLCHAIN` is not already `local`; otherwise it
uses the `go` line (PL7). With v1's `go 1.27` it installs whichever 1.27.x patch the runner has; with the
draft's `go 1.26.0` (D-1) it would have installed exactly 1.26.0, with a macOS 12 floor. `GOTOOLCHAIN=local`
then ignores the newer `toolchain` line without error, so nothing fails loudly (PL7). *Fix:* never set
`GOTOOLCHAIN` at workflow or job level before setup-go, which exports `GOTOOLCHAIN=local` itself after
resolving the version, or pass `go-version: '1.27.1'`. Print `go version` in every job.

## 7. Build, CI and release

**B-1 (Minor, was Major) — The smoke paths do not match the build output.** The build writes
`dist/wopr_${os}_${arch}` (L456); the smoke test runs `./dist/wopr-linux-amd64`, `./dist/wopr-macos-arm64`
and `.\dist\wopr-windows-amd64.exe` (L724-726), so it fails on all three runners in the first M0 run. The
artifact names (`wopr-macos-*`, L731) also differ from the release names (`wopr_<version>_darwin_*`,
L755-756). *Fix:* one naming scheme, and name the exact path the smoke job reads; under GoReleaser it is not
`dist/wopr_<os>_<arch>` (B-9).

**B-2 (Major) — CI artifacts are not the release artifacts.** CI builds with `scripts/build.sh` on three
runners (L719-735) and releases with GoReleaser (L753); the two definitions will drift. With
`CGO_ENABLED=0`, building on the target OS adds nothing (X28). What matters is **running** each binary on
its OS and architecture, and v1 runs only the native-arch one per runner (L728). *Fix:* one `build` job runs
`goreleaser release --snapshot --clean` with the release config and uploads six binaries; a `smoke` matrix
runs each on a native runner, using the free `ubuntu-24.04-arm`, `windows-11-arm` and `macos-26-intel`
labels (PL3, PL4); the size gate runs on that output. The tag build still differs in its version string, so
"CI tests what ships" holds only with reproducible archives (B-12) and a gated tag build (B-10). The owner
chose this (round 1, Q3); its costs are in B-9.

**B-3 (Minor, was Major) — Golden files arrive with CRLF on Windows runners.** `.gitattributes` forces LF
only for `*.go` and `*.txt` (L270), and GitHub's Windows images install Git with `core.autocrlf=true`, so
`*.golden` files and other test data check out with CRLF and byte-exact goldens fail (PL5). The draft's
claim that CRLF scripts fail under Git Bash with `$'\r': command not found` is withdrawn: Git for Windows'
MSYS2 bash drops CR from shell input (PL6); the error applies to WSL and Linux bash. *Fix:*
`* text=auto eol=lf` globally (verified, PL5); binary fixtures `-text`; `.bat`/`.cmd` files `eol=crlf` if any
are added.

**B-4 (Minor, was Major) — Release cache poisoning.** `actions/setup-go` caches by default (PL7), and v1
uses it "everywhere" (L709), including the tag-triggered release. The draft's vector was wrong: a
pull-request cache is restored only by re-runs of that pull request, never by a tag run (X16). The real
vector is a cache written by a default-branch run, by compromised code holding the Actions runtime token (a
dependency run by `go test`, a compromised hook), then restored by the tag run. v1's own zizmor hook already
flags it, and `--fix` will not apply the "unsafe" fix (HK3). *Fix:* `cache: false` on setup-go in every job
that runs GoReleaser, including the CI snapshot job (B-9).

**B-5 (Minor) — Runner labels.** The darwin/amd64 binary is never run (L725). `ubuntu-22.04` has been
deprecated since 2026-09-17 and is unsupported from 2027-04-17, with brownouts before that (PL2).
`macos-latest` moves are announced and roll out over one to two months, so the draft's "changes without
notice" is withdrawn, but a run can land on either version mid-migration (PL3). *Fix:* pin labels:
`ubuntu-24.04` (the binary is static, so 22.04 adds nothing), `macos-26` and `macos-26-intel` (D-5), and
explicit Windows labels. Record a revisit date for each in AGENTS.md.

**B-6 (Minor) — Duplicate CI runs and no concurrency group.** `ci.yml` "on push and pull_request" (L712)
runs twice for pull requests from branches in this repository; zizmor flags this only in its pedantic
persona. The draft's one-line snippet was invalid YAML (X17), and an unconditional `cancel-in-progress` would
cancel runs on `main` too (B-10). *Fix:* `push: branches: [main]` + `pull_request`, and:

```yaml
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: ${{ github.event_name == 'pull_request' }}
```

**B-7 (Minor) — size-gate.sh fails its own ShellCheck config and its own input.** With
`--severity=style --enable=all` (L679), ShellCheck 0.11.0 reports 14 notes (SC2250, SC2292) and exits 1
(HK8). An empty `dist/` makes the loop run on the literal `dist/*`, and `set -e` exits 1. The padding of
macOS `wc -c` is cosmetic, since bash accepts padded numbers. *Fix:* the draft's `shopt -s nullglob` turns an
empty `dist/` into a silent pass, so also fail when no binaries are found, and enumerate binaries from
GoReleaser's layout (B-9). Better, write the gate in Go (`go run ./internal/tools/sizegate`), which also runs
on Windows (B-14).

**B-8 (Nit, was Minor) — The README holds a measured size table (L814, L832).** Label any number with the
version and date it was measured at. Per-artifact sizes already go to the step summary (L744-746); sizes in
release notes need an extra step, since GoReleaser's templates do not expose them.

**B-9 (Major, new) — GoReleaser's defaults break the plan's paths and names.** A local snapshot run of
v2.18.2 (HK9) puts binaries in `dist/<project>_<os>_<arch>_<variant>/` (`dist/wopr_linux_amd64_v1/wopr`)
beside archives and JSON metadata, so v1's size gate (`for f in dist/*`, L745) fails on the directories and
the smoke paths (L724-726) do not exist. Default `goarch` includes `386`; Windows archives are `.tar.gz`; the
checksums file is `<project>_<version>_checksums.txt`, not `checksums.txt` (L757); without `project_name`
the name comes from the module path. Default ldflags set `main.version`, not `internal/version.Version`
(L455), without `-trimpath`. zizmor flags `goreleaser-action` beside a cache-enabled setup-go even in
`ci.yml` (HK3). *Fix:* set `project_name: wopr`, explicit `goos`/`goarch`, a zip override for Windows,
`checksum.name_template`, ldflags and `-trimpath`; read binary paths from `dist/artifacts.json` in the size
gate and smoke jobs; name uploaded artifacts explicitly; `cache: false` in GoReleaser jobs.

**B-10 (Major, new) — The tag build is not gated.** `release.yml` rebuilds on `v*` and publishes (L753-758)
with no tests, smoke run or size gate (that lives only in `ci.yml`, L728-729), and nothing checks that the
tagged commit is on `main` or passed CI. The CI snapshot is stamped `<version>-SNAPSHOT-<commit>`, so CI
never ran the shipped bytes (X18). With an unconditional `cancel-in-progress` (the draft's B-6), a merge to
`main` cancels the previous commit's CI, and that commit can still be tagged. *Fix:* `release.yml` runs
`verify` (the commit is an ancestor of `origin/main` and its `ci-ok` succeeded), then `build`
(`goreleaser release --skip=publish`, `contents: read`, no OIDC) with the size gate, then the smoke matrix on
the release archives (a reusable workflow shared with `ci.yml`), then a small `publish` job (S-7).

**B-11 (Major, new) — Release archives ship without NOTICE.md or third-party notices.** GoReleaser's
default archive contents are `license*`, `readme*` and `changelog*` (X19), and v1 configures nothing else
(L277), so archives ship the MIT LICENSE without NOTICE.md, where v1 excludes the film quotations from the
grant and disclaims affiliation (L278, L805-806). The binary also links 17 third-party modules (15 MIT;
`x/sys` and `x/sync` BSD-3) and the Go runtime (BSD-3), whose licence requires binary redistributions to
reproduce the notice (X19). *Fix:* `archives.files`: LICENSE, README.md, NOTICE.md, THIRD_PARTY_NOTICES.txt,
the last generated in a `before` hook by a pinned tool (for example `go-licenses` in `tools/go.mod`). Check
archive contents in the smoke job; point `go install` users to the notices from `--version`.

**B-12 (Minor, new) — Archives are not reproducible.** v1 sets no timestamps (L277), so LICENSE, README and
NOTICE enter the archives with their checkout mtime, and rebuilding a tag from a fresh clone gives different
archive hashes and checksums (X19). *Fix:* `builds.mod_timestamp: "{{ .CommitTimestamp }}"` and
`info.mtime: "{{ .CommitDate }}"` with fixed owner, group and mode for every extra file. A CI job rebuilds
from two fresh checkouts and compares checksums.

**B-13 (Minor, new) — "Try a development build" can hand users code from unreviewed fork PRs.** The README
points at Actions artifacts (L733-735, L782), and `ci.yml` runs on `pull_request` (L712). A fork PR's run
belongs to the base repository, so its artifacts appear under the same names as builds from `main` (L731),
without attestation (X21). *Fix:* link only to push runs on `main`, or publish a rolling `nightly`
prerelease from `main` through the attest step. Give PR artifacts distinct names and short retention.

**B-14 (Minor, new) — The Windows contributor path does not work.** The README's build section says "Go
1.27, `make`, `prek install`" (L783-784); the Makefile wraps lint (L494, L831); CI shell lives in
`scripts/*.sh` (L273); verification runs `go test ./... -race` (L863). Git for Windows ships no `make` and no
util-linux `script`; `-race` needs cgo, which Go disables when no C compiler is found, and is not supported on
windows/arm64 (X22). Bash cannot drive ConPTY for the e2e test (Q-1). *Fix:* prefer Go programs and tests to
shell (the size gate, smoke and e2e steps); keep one command table, in AGENTS.md; document the test command
per OS (`-race` on Linux, macOS, and Windows amd64 with a C toolchain); compile the e2e test per target
(`go test -c`) so smoke runners need no Go.

## 8. Lint and static analysis

**T-1 (Minor, was Blocker) — Two `prek.toml` entries cannot resolve, and `validate-config` does not
notice.** The draft said the config does not validate. It does: `prek validate-config` (0.5.5) reports "All
configs are valid" for v1's block, named `[priorities]` with `priority = "fixers"` have been valid since prek
0.4.11, and every key is valid (PK2–PK5). Two entries fail when hooks are initialised (`prek list`,
`prek run`, `prek install --prepare-hooks`): gitleaks `rev = "v8.x"` (L640, a placeholder v1 itself flags)
fails with `pathspec 'v8.x' did not match` (PK8), and `rumdl-check` does not exist at rumdl-pre-commit
v0.2.58 (L685-692), whose ids are `rumdl` and `rumdl-fmt` (HK6). With those two lines fixed, all 27 hooks
resolve and run. *Fix:* pin gitleaks v8.30.1 (by SHA, T-3); use rumdl-pre-commit v0.2.78 with `rumdl-check`.
The M0 gate is `prek list` plus `prek run --all-files`, since `validate-config` checks neither revs nor ids.

**T-2 (Major, was Blocker) — The `forbidigo` tick rule never fires.** forbidigo matches the printed text of
an identifier or selector (`tea.Tick`), never a call with its `(`, so `tea\.(Tick|Every|Sequence)\(` (L698)
reports 0 issues on code that calls `tea.Tick` (GL1). The draft's fix, `^(Tick|Every|Sequence)$` with `pkg`,
catches only a dot import: with `analyze-types`, a package selector's text is the declared package name plus
the selector, `tea.Tick`, whatever the import alias (GL2). It is one config line, hence Major. *Fix:*

```yaml
linters:
  settings:
    forbidigo:
      analyze-types: true
      forbid:
        - {pattern: '^tea\.(Tick|Every|Sequence)$', pkg: '^charm\.land/bubbletea/v2$'}
  exclusions:
    warn-unused: true
    rules: [{path: '^internal/ui/clock\.go$', linters: [forbidigo]}]
```

This flags `tea.Tick`, an aliased `bt.Tick`, a dot-imported `Every`, `tea.Sequence` and non-call references,
and the exclusion works (GL2, GL3). `config verify` accepts both patterns, so M0 must show the rule firing on
a deliberate `tea.Tick`. Consider also banning `time.Sleep`, `time.After` and tickers in `internal/ui`
outside `clock.go`, since a Cmd can sleep and return its own tick.

**T-3 (Major) — Hook revisions are pinned by mutable tags, while Actions are pinned by SHA.** Anyone who
controls a hook repo can move a tag. Several v1 revs are also behind: zizmor-pre-commit v1.26.1 (latest
v1.30.1), codespell v2.4.2 (v2.4.3), rumdl-pre-commit v0.2.58 (v0.2.78), and prek-action v2.0.6 (v3.0.1,
which has no floating tags) (HK3, HK4, HK6, HK7). *Fix:* pin hooks by commit SHA with a `# frozen: vX.Y.Z`
comment; set `[update] freeze = true` and `cooldown_days = 14` in `prek.toml` so a plain `prek update` does it
(PK3, PK10). The draft's `prek auto-update --freeze` no longer exists (T-10).

**T-4 (Minor, was Major) — `govulncheck@latest` runs a moving version and never runs in CI.** The hook runs
`go run golang.org/x/vuln/cmd/govulncheck@latest ./...` (L632), against "all `rev` values are pinned
exactly" (L699). The risk is reproducibility, not arbitrary code: module versions are immutable and checked
against `sum.golang.org`. The hook also fails when the local Go is older than 1.27 (X30), and as a pre-push
hook it never runs in CI (T-9). *Fix:* declare it in `tools/go.mod` and run
`go tool -modfile=tools/go.mod govulncheck ./...` (verified, X30), keeping that module's `go` line at or
below the CI toolchain. Run it as a CI step and weekly (S-10).

**T-5 (Minor) — zizmor `--fix` runs in the lint tier (L653).** A fixer that rewrites workflow files runs
alongside read-only linters, and in CI it only turns a finding into a "files modified" failure. *Fix:* run
zizmor without `--fix` in hooks and document `zizmor --fix` as a manual command. For its online audits, give
`GH_TOKEN` to a separate zizmor step, not to the prek step, where every hook would inherit it (HK3).
`golangci-lint-full` is also a fixer (T-8).

**T-6 (Nit, was Minor) — Redundant checks.** `linters.default: standard` already enables errcheck, govet,
ineffassign, staticcheck and unused (GL5), and golangci-lint's govet covers `go vet`. Drop the local `go vet`
hook (L600-607) and list only additions.

**T-7 (Minor) — Updates are manual.** Dependabot's pre-commit support reads only `.pre-commit-config.yaml`
and similar names, never `prek.toml` (PL8), and the owner has since dropped Dependabot entirely and kept
`prek.toml` (round 2, b). Hook revs, Go modules (`go.mod`, `tools/go.mod`) and Action SHAs are all updated by
hand. *Fix:* one routine in AGENTS.md, run at each milestone and on every Go security release: `prek update`
(frozen, with cooldown), `go get -u` and `go mod tidy` in both modules, and Action SHA bumps checked by
zizmor's online audits. The draft's monthly update-PR workflow is dropped: it needs "Allow GitHub Actions to
create and approve pull requests", and each PR waits for an "Approve workflows to run" click (X21).

**T-8 (Minor, new) — Fixers race, and `golangci-lint-full` is a fixer.** v1 gives the four mutating builtins
one priority (`fixers = 0`; L509, L517-533); same-priority hooks run concurrently, prek's docs call that
undefined, and lost updates were reproduced (PK5). `golangci-lint-full` runs `golangci-lint run --fix`
(HK1) but sits in the read-only tier (L591-594). *Fix:* one priority per mutating fixer (byte-order mark,
line endings, whitespace, end of file), an explicit priority on every hook (an omitted one defaults to its
index and collides), and `entry = "golangci-lint run"` on `golangci-lint-full`.

**T-9 (Minor, new) — The install and validation commands do less than v1 says.** `prek install` installs
only the pre-commit shim, so the pre-push hooks (`go test`, govulncheck; L618-635) never run, contrary to
L493, and bare `prek validate-config` (v1's `make lint`, L831; L701) prints "No configs to check" and exits
0 (PK11). Pre-push hooks never run in CI either. *Fix:* set `default_install_hook_types` to pre-commit and
pre-push, and `default_stages` to pre-commit and manual, with `stages` set on the five builtins whose
manifests include pre-push. Use `prek validate-config prek.toml` plus `prek list`; run `go test` and
govulncheck as CI steps.

**T-10 (Minor, new) — `prek auto-update` no longer exists.** It was renamed `prek update` in 0.4.8 and
removed in 0.5.0; with prek 0.5.5, `prek auto-update --freeze` exits 2 (PK10). v1 uses it (L700, L819), and
so did the draft (T-3, T-7). *Fix:* `prek update`.

**T-11 (Minor, new) — SHA-pinned actions still install unpinned tools.** `j178/prek-action` defaults
`prek-version: latest`, resolved at run time from a manifest on its `main` branch, and checks checksums only
for versions in its bundled list (HK7). `goreleaser-action` defaults to `~> v2` and only warns if its
checksum download fails (X20), in the job that holds `contents: write`, so CI and the release can run
different GoReleaser versions. *Fix:* `prek-version: 0.5.5`; an exact GoReleaser version (or GoReleaser as a
Go tool); gitleaks as a checksum-verified binary or Go tool. Bump each with its action SHA.

## 9. Security and supply chain

**S-1 (Major, was Blocker) — gitleaks never scans anything in CI.** The upstream hook runs
`gitleaks git --pre-commit --redact --staged --verbose`. On a fresh clone with three committed secrets,
`prek run --all-files` reports Passed and "0 commits scanned", while `gitleaks git .` finds all three (HK2),
so v1's "in CI" (L792-793) is false. The draft's `before..sha` range also scans nothing: a `pull_request`
`opened` event has no `before`, a new branch's is all zeros, and a depth-1 checkout makes the range invalid;
gitleaks logs an error and exits 0 (HK2). Its inline `${{ }}` in `run:` also broke v1's rule (L737-739).
Push protection guards against accidents but is not unbypassable: write-access users may bypass it with a
reason, and it covers only supported patterns (PL10). *Fix:* check out with `fetch-depth: 0` and scan the
full history with a pinned, checksum-verified gitleaks (`gitleaks git --redact --exit-code 1 .`). For a PR
range, pass `pull_request.base.sha..head.sha` through `env:` and fail on any error or on zero commits
scanned. In M0, show the job failing on a throwaway PR with a test secret. Enable repository-level push
protection, which is off by default (PL10).

**S-2 (Minor, was Major) — Releases have checksums but no provenance.** The checksums file lives in the
release it protects. Attestations prove "built by a workflow in this repository", not protection from a
compromised maintainer, and v1 chose "no signing keys in v1" (L796), so this is cheap hardening. *Fix:*
`actions/attest@v4` (attest-build-provenance v4 wraps it) with `id-token: write` and `attestations: write`;
`subject-checksums` attests every archive in one step (PL9). Verification flags are in S-7.

**S-3 (Minor, was Major) — Repository settings are part of the threat model and are missing.** For a
single-owner public repository the risk is low, and secret scanning already runs automatically (PL10).
*Fix:* document and apply: a `main` ruleset (no force-push; PR-only, as the owner chose, round 2 c); one
required check, an aggregate `ci-ok` job (`needs:` every job, `if: always()`, failing unless all
succeeded), because matrix jobs report checks named like `test (ubuntu-24.04)` and a required `test` never
reports (X21); a `v*` tag ruleset (S-7); repository-level push protection; private vulnerability reporting
and `SECURITY.md`; and the Actions policies in S-8. Dependabot security updates are out (round 2, b).

**S-4 (Minor, was Major) — The LLM milestone needs an opt-in rule now.** "Env-configured (keys from env
only)" (L820) could mean the LLM switches on whenever a provider key is in the environment, contradicting
"never phones home" (L798). *Fix:* only an explicit flag or `WOPR_LLM` enables the LLM brain, never a key
alone, and an in-character notice says it is on. Timeouts, max tokens, bounded history and retries,
sanitised output (U-2), validated effects and no prompt logging belong in the M6 design; transport is S-9.

**S-5 (Minor) — Debug log location and contents.** `WOPR_DEBUG` writes a log (L476-477, L799) but the plan
does not say where. *Fix:* `os.UserCacheDir()/wopr/` (directory 0700, file 0600), never the CWD; if that
fails, disable logging with a stderr note; print the path on exit; keep "never typed input". Do not document
`TEA_DEBUG`, which writes a panic log into the CWD (BT5).

**S-6 (Minor, moot) — `dependabot.yml` has no cooldown or grouping (L760).** Moot: the owner dropped
Dependabot (round 2, b). If it returns: `cooldown: {default-days: 7}` (v1's file fails zizmor's
`dependabot-cooldown` audit), grouped minor and patch updates, and gomod for `/` and `/tools` (PL8).

**S-7 (Major, new) — Tags and releases are mutable.** A tag ruleset that restricts only creation (S-3) still
lets the owner delete and re-push `v0.1.0`, the natural reaction to a failed release. The Go checksum
database keeps a version's first-fetched content, so after a re-tag `go install …@v0.1.0` (L770) builds the
old commit while the assets come from the new one, and `GOPROXY=direct` users get a security error (X18).
Immutable releases protect only releases created after the setting is on. GoReleaser publishes at the end
of its run, so attesting afterwards can leave a public release without provenance, and
`gh attestation verify --repo` alone accepts any workflow and ref in the repository. *Fix:* enable immutable
releases before v0.1.0; restrict creation, update and deletion of `v*`; never re-tag (fix forward with a
patch release and a `retract` directive); draft, attest, then publish. The README has users verify with
`gh attestation verify` plus `--signer-workflow` (the release workflow) and `--source-ref` (the tag) before
the `xattr` or SmartScreen steps (L768-769).

**S-8 (Minor, new) — Actions-level policies are missing.** SHA pinning and least privilege are enforced only
by zizmor in prek (L797), which `--no-verify` skips. *Fix:* in repository settings, require full-length SHA
pins, allow only listed actions, make default workflow permissions read-only, and require approval of PR
runs for all external contributors (the default, first-time contributors only, lapses after one merge; X21).

**S-9 (Minor, new) — The M6 HTTP client needs hardening.** Go's client follows 307/308 redirects, re-sends
the body, and on a cross-host redirect strips only `Authorization`, `Cookie`, `WWW-Authenticate` and the
`Proxy-*` headers, so a custom key header such as `x-api-key` reaches the new host with the prompt (X23).
*Fix:* `CheckRedirect` returns `http.ErrUseLastResponse`; https is required except for loopback hosts;
responses are read through `io.LimitReader`; LLM replies carry only lines and a registry-validated game
start, never `SetFlag` or `MarkSaid` effects (A-4); the reply parser is fuzzed.

**S-10 (Minor, new) — Scheduled checks stop silently, and nothing responds to a hit.** The weekly
govulncheck (T-4) relies on a schedule, and scheduled workflows in a public repository stop after 60 days
without repository activity (X21). A hit only turns a scheduled run red. *Fix:* also run govulncheck in
`ci.yml`; on a scheduled failure, open or update an issue (`issues: write` on that job only); check for
disabled schedules at each milestone. `SECURITY.md` states the response: a reachable finding or a Go security
release means bumping `toolchain` and a patch release within a stated number of days.

## 10. Legal and licensing

**L-1 (Major) — Provenance of film text and ASCII art.** The research names fan repositories as sources
(L25-27) and treats their text as canonical (L51). Their licences differ: `built1n/wargames` is
GPL-3.0-or-later, with its ASCII map (`MAP`, `map.h`) under CC BY-SA 4.0; `abs0/wargames` has BSD-2-Clause
headers per file; `elfuska/wargames` (a fork of zompiexx/wargames) has no open-source licence, and its map
appears derived from built1n's (EC2). None folds into an MIT grant. Film dialogue in a fan transcript is
MGM's expression, so the fan licences bite mainly on fan art, code and wording. v1 already contains such
material, including GTW mockup fragments that match built1n's CC BY-SA 4.0 map (L-4). *Fix:* draw ASCII art
from scratch; take short lines from the film itself; record provenance per entry (`film`, `original`,
`reconstructed`, `third-party:<repo>@<sha>:<path>`, `prompt`) and test that every entry has one. Fan-sourced
lines stay `reconstructed` until the viewing pass (F-1). `NOTICE.md` lists third-party material and ships in
the archives (B-11).

**L-2 (Nit, was Minor) — Trademark.** v1 already uses "WarGames" only descriptively, disclaims affiliation
and plans screenshots of the program itself (L7-8, L805-806, L819). Keep it so: no MGM marks in names or
artwork, no film stills. Registered marks are L-6.

**L-3 (Nit, was Minor) — The brother's prompt.** v1 already requires his permission and credit (L785-786,
L802-804). Gate the adapted text, not the milestone: prompt-derived lines carry their own provenance tag and
land only after written permission (L-5). The style rules (ALL CAPS, PROFESSOR, numbered lists) are ideas
and need no permission.

**L-4 (Major, new) — Third-party text is already in v1** (EC2, X24). The GTW mockup map (L150-160) contains
fragments identical to built1n's CC BY-SA 4.0 `MAP` (`__----/^\.`, `/__/   \/^^\`, `\.\_--`) at matching
positions. The backdoor header (L57-61, again at L122-125) is byte-identical, column spacing included, to
lines 170-173 of `abs0/wargames` `wargames.sh` (BSD-2-Clause, © 2012 David Brownlee). 40 of the 45 montage
names (L88-97) appear in that file in the same order, and its typo `BURMESE THEATERWIOE` sits at the same
place, marking it as a transcription source. v1 exists only on the feature branch, not on `main`;
squash-merging keeps it off `main`. *Fix:* replace the mockup map with original art before it moves anywhere
(not into `docs/screens.md`). Re-transcribe the header from the film, or credit abs0 under BSD-2. For the
montage, the owner chose to keep the list, credit `abs0/wargames` under BSD-2 in NOTICE.md now, and verify
it in the M5 viewing pass (round 2, d).

**L-5 (Minor, new) — Permission is not a licence.** Written permission to adapt the brother's prompt (L-3)
grants no licence to publish the adapted lines under MIT or to send them to an LLM provider as a system
prompt in M6. *Fix:* tag prompt-derived lines `prompt`. Before they ship, record an explicit licence from him
(MIT or CC0) in NOTICE.md; otherwise list them with the film quotations as excluded from the MIT grant.

**L-6 (Nit, new) — Registered marks.** "WOPR" is a live US registration: Reg. 8,024,966, Frontier
Technology, Inc., class 42 (software services, including simulation), registered 2025-11-11. WARGAMES is
MGM's (Reg. 2,690,923, class 41) (EC4). The risk for a free terminal game is low. Recorded for the owner to
decide; not legal advice.

## 11. Testing

**Q-1 (Major) — The real TUI binary is never exercised in CI.** Smoke tests run only `--version` and
`--games` (L724-729), and goldens drive the model, not the program (L484-489), so terminal setup, raw mode
and restore are untested. The draft's fix had three errors: `teatest/v2` runs the model in-process with
buffers and without signals, so it sees no exit code or restore (BT12); `SHALL WE PLAY A GAME?` comes three
user replies after `GREETINGS PROFESSOR FALKEN.`, not right after `Joshua` (L65-67, L128-141); and a fresh
pty is 0×0, which shows TERMINAL TOO SMALL (X8). *Fix:* one Go harness on every OS over a real pty, for
example `github.com/charmbracelet/x/xpty` (Unix pty and ConPTY; X27), sized 80×24. Run `--instant --seed 1`,
type `Joshua`, wait for `GREETINGS PROFESSOR FALKEN.`, then cover LOGOFF → 0 and Ctrl+C → 130 (U-10) and
check the terminal is restored. ConPTY re-renders output, so confirm the alternate-screen exit is observable
there before asserting it. Build the test per target so smoke runners need no Go (B-14).

**Q-2 (Minor) — Fuzzing.** Normalisation, `games.Resolve`, the line editor and `Sanitize` are pure
functions over user input; native fuzz targets are cheap, and their seed corpora run in plain `go test`.
`go test -fuzz` takes one package and one target per run, so CI calls each separately
(`-run '^$' -fuzz '^FuzzX$' -fuzztime=10s ./pkg`) and uploads any crasher as an artifact (X27). Fuzzing SAN
parsing adds little if the chess library parses.

**Q-3 (Minor) — Golden-file mechanics.** v1 names the driver and flows (L484-489) but not how goldens are
updated or stored. A `-update` flag does not work across packages: `go test ./internal/ui/... -update` fails
in every package that does not define it (X27), and game tests may not import `ui` helpers. *Fix:* a shared,
stdlib-only `internal/golden` helper for `ui`, `wopr` and game tests, driven by an environment variable
(`WOPR_UPDATE_GOLDEN=1 go test ./...`). Goldens live in `testdata/` with LF endings (B-3); diffs are reviewed
in the PR.

**Q-4 (Minor) — AI quality tests.** v1 already tests that tic-tac-toe never loses and that checkers forces
captures (L481-482); chess and checkers strength is untested. The draft's universal "beats random play in
≥ 95% of 200 seeded games" is wrong: perfect tic-tac-toe as O beats a random player only about 78–81% of
the time, the rest being draws (X27). It is meaningless for Black Jack, GTW, the always-`WINNER: NONE` sim
and the maze, and noisy for poker. *Fix:* one quality test per game with its own threshold, under a fixed
depth or node limit (A-3) so results are reproducible and fast; perft only for a custom move generator.

## 12. Documentation and maintainability

**M-1 (Minor, was Major) — AGENTS.md is written last.** The conventions agent sessions need are in the plan
(L231-247), which nothing loads automatically, and AGENTS.md waits until M5 (L222, L836-840). *Fix:* write
AGENTS.md in M0 and keep the conventions there, linked from the plan, so there is one copy. A `CLAUDE.md`, if
any, must contain the import line `@AGENTS.md` (or be a symlink): Claude Code reads AGENTS.md by itself only
when no CLAUDE.md exists, so a CLAUDE.md that merely mentions it, as L840 suggests, stops it loading (X25).

**M-2 (Minor, was Major) — The plan embeds full config files** (`prek.toml`, L501-695; `size-gate.sh`,
L740-751). Before M0 these are seeds with nothing to drift from. *Fix:* keep seeds in a deletable appendix
until M0; then the repository files are the source of truth and the plan links to them. This replaces T-1's
draft fix, which kept a validated config inside the plan.

**M-3 (Nit, was Minor) — Keep `ui/app.go` a thin router.** `console/`, `screens/` and `testutil/` are
already subpackages and the root has four files (L239-246), so `ui` is not a god package. Per-screen
sub-models, if added, return content and a cursor position, not `tea.View`, which carries program-wide
fields such as `AltScreen`, `BackgroundColor` and `MouseMode` (BT2).

**M-4 (Nit) — Markdown issues the plan's own linter would flag.** rumdl 0.2.58 with v1's own arguments
reports 23 issues (X26): MD040 (fence without a language) ×8, at L31, 57, 121, 149, 177, 234, 399 and 453;
MD031 (blank lines around fences) ×9; MD032 and MD022 at `### Tooling` (L489-490); MD038 at L53; MD049 at
L3; MD004 and MD032 at L843, where a wrapped `+ tests` becomes a list item. "Some lines are very long" is
withdrawn: v1 disables MD013 (L693). `rumdl fmt` fixes all 23; the hooks need T-1's id fix.

## 13. Milestones and implementability

**P-1 (Major, was Blocker) — GTW (M2) depends on `sim` (M4).** GTW is "(Fullscreen, `sim`)" (L445) and in
M2 (L816); `sim` is M4 (L818). Nothing fails as written: M2 would quietly build engine pieces, which is scope
creep and later rework. *Fix:* make GTW a self-contained set piece with a small `gtw`-local model (targets,
salvos, DEFCON ladder, kill-ratio tables), and extract shared pieces into `sim` when the second consumer
arrives in M4. The draft's alternative (a minimal `sim` core in M2) is withdrawn: it would shape the engine
around a scripted set piece before the engine is specified (G-2).

**P-2 (Major) — No tagged release until every game is done.** CI artifacts exist from M0 (L730-735), but
v0.1.0 comes only in M5 (L819), after all 16 games, five of them under-specified (G-2) and Bridge open-ended
(G-3). *Fix:* release v0.1.0 at the end of M2: the full film path plus chess and checkers. Unbuilt games stay
on `LIST GAMES`, as in the film, and WOPR declines in character with `** GAME ROUTINE NOT AVAILABLE **`, an
invented line to tag `original` (L-1). Specify what `wopr -p <unbuilt game>` prints and its exit code. Each
later milestone ships a minor release. The owner agreed (round 1, Q1).

**P-3 (Nit, was Minor) — Line-count estimates.** "~600 lines each" (L418) drives nothing in v1, and the
draft offered no data for calling it low. Drop per-game estimates; if actuals are tracked, add milestone
estimates to compare them with.

**P-4 (Major, new) — Menu parsing is scheduled after the games that need it.** `prompt/` (numbered menus and
yes/no; L262) is an M3 deliverable (L817), but M1's persona needs numbered selection from `LIST GAMES`
(L251-252), M2's GTW needs `1. UNITED STATES` / `2. SOVIET UNION` / `PLEASE CHOOSE ONE:` (L68-69), and M2's
tic-tac-toe needs `NUMBER OF PLAYERS` (L446): the pattern P-1 fixed for `sim`. *Fix:* move menu and yes/no
parsing into M1, in a package both `wopr` and the games may import. The canvas renderer (A-2) and Panel
geometry (row split, width rule above 80 columns) also belong in M1, so the stub game exercises every layout.

## 14. Film fidelity

**F-1 (Minor, new) — Several details are confirmed; two lines are doubtful.** Checked against production
sources and fan transcripts (EC5, EC6). *Confirmed:* David's monitor was a 17" Electrohome black-and-white
set, so white on black is right for `imsai` (L101-105); the "faint cool/blue cast" is unverified. Output
came from off-screen CompuPro systems on a 24×80 display with custom line-segment glyphs for the maps, which
supports the 80×24 target and suggests drawing maps from line glyphs. LOGON precedes the `#45` header, which
appears only after `Joshua` (L53-64). The GTW-versus-chess order is right (L65-67). David's input is mixed
case, so L380 is right and "chunky uppercase glyphs" (L103) applies only to WOPR. *Doubtful:* "USER
ACCOUNT NUMBER ON 6/23/73" (L66, L137), where a fan transcript and the subtitles say "user account"; and
"CITY AND/OR COUNTRY NAME" (L70, L171), where both fan transcripts say COUNTY. *Fix:* tag both, and every
other fan-sourced line, `reconstructed` until the M5 viewing pass, which checks each one.

## Owner decisions

The owner answered in the planning session. Record each answer durably (an issue comment or a commit by the
owner), so that the plan's "U" sources can cite it.

### Round 1 (2026-10-06)

| # | Question | Answer | Effect on v2 |
|---|---|---|---|
| 1 | Ship v0.1.0 after the film set pieces (M2), with other games added in minor releases? | **Yes.** | v0.1.0 at the end of M2. Unbuilt games stay listed and WOPR answers in character (P-2). |
| 2 | Bridge: cut, minimal, or full? | **Minimal variant.** | WOPR bids all four seats by point count; the user plays the declaring side (seat rule in G-3). Built last in the card milestone. |
| 3 | Build once with GoReleaser and run natively on each OS/arch, instead of native builds per OS? | **Yes.** | One GoReleaser snapshot build; a smoke matrix runs each binary on its own OS and architecture (B-2). Follow-on costs in B-9. |
| 4 | `go 1.26.0` (the stack's minimum) with `toolchain go1.27.1`, instead of `go 1.27`? | Not asked in round 1. | Asked and answered in round 2 (e). |
| 5 | 10 MB hard limit, or keep the 10/15 MB warn/fail split? | **Keep the 10/15 split.** | Decision 10 stands; R-1 is resolved for the size limit. |

### Round 2 (2026-10-06)

| # | Question | Answer | Resolves |
|---|---|---|---|
| a | (Volunteered by the owner while D-5 was open.) Which macOS versions are supported? | **macOS 26 only;** older versions are not considered. | D-5. CI tests `macos-26` and `macos-26-intel` (B-5). |
| b | Dependabot cannot read `prek.toml`. Switch to `.pre-commit-config.yaml`, or keep `prek.toml`? | **Keep `prek.toml`. No Dependabot at all for now;** it is dropped from the requirements. | T-7 (updates are manual), S-6 (moot), S-3 (no Dependabot security updates). Decision 8's "Dependabot keeps modules and Actions current" (L218) and `dependabot.yml` (L276, L760) go. |
| c | Should `main` be PR-only with required checks? | **Yes: PR-only, with one required aggregate check, `ci-ok`.** | S-3; B-10's `verify` job checks `ci-ok` on the tagged commit. |
| d | The montage names match `abs0/wargames`. Re-transcribe before v0.1.0, or keep and credit? | **Keep the list, credit `abs0/wargames` under BSD-2 in NOTICE.md now, verify in the M5 viewing pass.** | L-4 (montage part). Needs B-11, so that NOTICE.md ships. |
| e | `go 1.26.0` (D-1) or keep `go 1.27`? (round 1, Q4) | **Keep `go 1.27` + `toolchain go1.27.1`, as in v1.** | D-1 closed as an owner decision; v2's Decision 8 source is U. D-6 still applies. |
| f | (Volunteered by the owner.) New feature. | **Add a "movie mode" (`--movie`) that replays the film's WOPR scenes, as Milestone 6.** | New requirement in v2 (R12, §7). v1's M6 (LLM brain) becomes M7 in v2. |

## Verification log

Environment for every local experiment: Go 1.27.1, prek 0.5.5, golangci-lint 2.14.0 (built with go1.27.1),
`charm.land/bubbletea/v2` v2.0.10 and `charm.land/lipgloss/v2` v2.0.6 on a 4-vCPU linux/amd64 VM,
2026-10-06. "Local experiment" means a throwaway program or repository; nothing in this repository changed.
Windows and macOS behaviour was read from source, not run. Locations are `file:line` at the named version.
Rows that only confirm a v1 claim no finding disputes (for example PK1, PK9, GL7) stay, so the plan can cite
them.

| Claim | How checked | Result |
|---|---|---|
| **BT1** Bubble Tea v2.0.10 needs `go 1.26.0`; Lip Gloss v2.0.6 needs 1.25 | `go.mod` line 5 of each; `go list -m -f '{{.GoVersion}}' all` | True. 1.26.0 is the highest in the graph; `corentings/chess/v2` v2.6.0 declares 1.25.0, `notnil/chess` 1.14 |
| **BT2** `View()` returns `tea.View` with `Content`, `Cursor`, `AltScreen`, `BackgroundColor`, `MouseMode`, `KeyboardEnhancements` | bubbletea `tea.go:53-64`, `:84-190`; test program | True |
| **BT3** `tea.NewCursor(x, y)` gives a blinking block cursor | `tea.go:365-389`, `cursor.go:11-19`; pty capture (`ESC[1 q`) | True; blink is the default |
| **BT4** `KeyPressMsg`, `PasteMsg`, `BatchMsg` are exported; `tea.Sequence`'s message is not | `key.go:191`, `paste.go:5-8`, `commands.go:15-54` | True. A Batch or Sequence of one cmd returns it unwrapped |
| **BT5** A panic in `Update`, `View` or a Cmd restores the terminal and returns `ErrProgramKilled` + `ErrProgramPanic` | `tea.go:729-740`, `:1033-1040`, `:1292-1342`; pty runs of each case | True. No panic value in the error; `TEA_DEBUG=true` writes a panic log in the CWD |
| **BT6** `tea.WithFilter` runs on the event loop | `options.go:111-144`, `tea.go:761-769` | True |
| **BT7** Cmds run on their own goroutines while `Update` continues | `tea.go:708-745`, `:849-855`, `:880-889`; `-race` build sharing state between a Cmd and `Update` | True: `WARNING: DATA RACE` |
| **BT8** The renderer downsamples per profile; `NO_COLOR` caps it | `tea.go:1090-1098`; colorprofile v0.4.3 `env.go:33-53`, `:76-90`, `:114-118`; Lip Gloss `writer.go:14,68`; pty captures | True, but `NO_COLOR` uses `ParseBool` (`NO_COLOR=yes` ignored); OSC 10/11 sent even under `NO_COLOR`; `Style.Render` alone does not downsample |
| **BT9** Behaviour when stdin or stdout is not a TTY | `tea.go:1015-1027`, `:1048-1058`; ultraviolet `tty_unix.go:15-21`, `tty_windows.go:12-25`; pty runs | Stdin falls back to `/dev/tty` / `CONIN$`. A non-TTY stdout is not refused: 0×0, escapes written to the file |
| **BT10** Ctrl+C arrives as a key; a blocking `Update` delays it | x/term `term_windows.go:22-33`; Microsoft `SetConsoleMode` docs; `tea.go:572-580`, `:651-688`, `:772-777`; Linux pty runs | True: queued until `Update` returns. SIGTERM yields `QuitMsg`, exit 0; Ctrl+Z is left to the program (`tea.Suspend`) |
| **BT11** `tea.Tick` fires once; its timer starts when called | `commands.go:102-164` | True |
| **BT12** `teatest/v2` exists and needs no pty | proxy `@latest` (pseudo-version only); `teatest.go:129-178`; test under `setsid` | True; in-process with buffers and `WithoutSignals`, so not a real-binary harness |
| **BT13** `AltScreen` and `BackgroundColor` behaviour | `cursed_renderer.go:176-247`, `:354-362`; x/ansi `background.go:139-153`; pty captures incl. panic | `?1049h/l`; OSC 11 on start, OSC 111 on exit, also after a panic |
| **BT14** x/ansi has `Strip` and `StringWidth` | x/ansi v0.11.8 `width.go:11-68`; test program | True. `Strip` keeps C0 bytes; `StringWidth("\t")` is 0 |
| **SZ1** v1: the stack is "5–7 MB stripped" (L202-203) | Local `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w"`, six targets, one styled line | False: 3.59–3.89 MiB (3.76–4.08 MB); plain hello world 1.4–1.6 MiB |
| **SZ2** `notnil/chess` is cheap to add | Same builds plus FEN, moves, SAN and PGN use | True: +0.09–0.32 MiB; `corentings/chess/v2` similar |
| **SZ3** A stdlib HTTP client adds several MB | Same builds plus a JSON POST over `net/http` + `crypto/tls` | True: +3.9–4.5 MB per target (linux/amd64 +4.3 MB). TUI + chess + HTTP: 7.6–8.3 MiB, largest windows/amd64 8,745,984 B |
| **SZ4** `notnil/chess` reaches 3–4 ply + quiescence in about 1 s | Local benchmark: perft; alpha-beta with MVV-LVA and captures-only quiescence; 8 self-play games | Partly. perft(4) 197,281 in 114–127 ms; depth 4 + qs 22–272 ms on six positions; self-play median 106 ms, p90 614 ms, p99 1.09 s, max 1.69 s. corentings v2.6.0: same nodes, slightly faster |
| **PK1** `prek.toml` is native and preferred over YAML | prek 0.5.5 `crates/prek-consts/src/lib.rs:13`; `docs/configuration.md:13-20` | True |
| **PK2** `minimum_prek_version` is valid | `config/mod.rs:84`; test with `"9.0.0"` | True; enforced at parse time |
| **PK3** `update.cooldown_days` is valid | `config/update.rs:70-79`; `prek update --check` | True; `update.freeze` also exists |
| **PK4** `default_language_version.golang` is valid | `docs/reference/configuration.md:288-314`; hook run | True; `"1.27"` resolved to go1.27.1 |
| **PK5** Named `[priorities]` and `priority = "fixers"` are valid | `config/priority.rs`; CHANGELOG #2331 (0.4.11); `prek validate-config` on v1's block; 200-file race test | True: "All configs are valid". Same-priority fixers lost updates |
| **PK6** `repo = "builtin"` and all 14 builtin ids exist | `prek util list-builtins` | True |
| **PK7** Inline `hooks = [...]` and `[[repos.hooks]]` mix across repos | prek and `tomllib` parse | True; both in one repo entry is a TOML error |
| **PK8** gitleaks `rev = "v8.x"` fails | `prek list`, `prek run` on v1's block | True at hook init (`pathspec 'v8.x' did not match`); `validate-config` passes it. Latest v8.30.1 |
| **PK9** `language_version = "1.27"` works and prek fetches Go | `docs/reference/language-support.md`; `languages/golang/installer.rs:209-243`; hook run | True; prek downloaded go1.27.1 |
| **PK10** `prek auto-update --freeze --cooldown-days` | `prek update --help`; `cli/mod.rs:289`; CHANGELOG 0.4.8 (#2286), 0.5.0 (#2619) | Partly: the command is `prek update`; `prek auto-update --freeze` exits 2 |
| **PK11** `prek install` installs pre-commit and pre-push shims | `cli/mod.rs:247-253`; `cli/validate.rs:17-19`; runs | False: pre-commit only without `default_install_hook_types`. Bare `prek validate-config` exits 0 having checked nothing |
| **HK1** golangci-lint v2.14.0 hook ids | `.pre-commit-hooks.yaml:1-31` at v2.14.0 | `config-verify`, `fmt`, `full` exist; `full` runs `golangci-lint run --fix` |
| **HK2** The gitleaks hook scans only staged changes | gitleaks v8.30.1 `.pre-commit-hooks.yaml:1-17`, `cmd/git.go`; fresh clone with three secrets, via prek and directly; depth-1 clone with `before..sha` and `..sha` ranges | True: "0 commits scanned"; `gitleaks git .` finds 3. Both ranges also scan 0 commits and exit 0 |
| **HK3** zizmor flags setup-go caching in release jobs; online audits need a token | zizmor `cache_poisoning.rs:62-66`, `:240-250`, `:411-467`; `docs/usage.md:156-162`; zizmor 1.26.1 run | True, also for `ci.yml` jobs using goreleaser-action; the fix is "unsafe", so `--fix` skips it. Latest v1.30.1 |
| **HK4** codespell v2.4.2 and `--skip` | `_codespell.py:170-175`, `:1414-1459`; runs | v2.4.3 is latest. `--skip` applies to hook-passed paths; `testdata/*` misses nested dirs |
| **HK5** shellcheck-py v0.11.0.1 has hook `shellcheck` | `.pre-commit-hooks.yaml` at that tag | True; newer tag v0.11.0.1-1 |
| **HK6** rumdl-pre-commit v0.2.58 has `rumdl-check` | `.pre-commit-hooks.yaml` per tag; `git log -S rumdl-check`; prek run | False: `rumdl` and `rumdl-fmt`; `rumdl-check` from v0.2.66; latest v0.2.78 |
| **HK7** `j178/prek-action` v2.0.6 is current | `action.yml`, `src/manifest.ts:7-9`, `src/install.ts:193-223` at v3.0.1 | Partly: v3.0.1 is latest, without floating tags; `prek-version` defaults to `latest` from a manifest on `main` |
| **HK8** v1's `size-gate.sh` is ShellCheck-clean and handles its input | ShellCheck 0.11.0 `--severity=style --enable=all`; runs on missing, empty and GoReleaser `dist/`; Apple bash 3.2 source | False: 14 notes (SC2250, SC2292); empty `dist/` exits 1, or 0 with `nullglob`; GoReleaser layout fails; `wc` padding is cosmetic |
| **HK9** GoReleaser v2.18.2 layout and defaults | `internal/builders/base/options.go:22-30`, `golang/build.go:120-131`, `pipe/archive/archive.go:30-31`, `pipe/checksums/checksums.go:40-44`; snapshot runs | `dist/wopr_linux_amd64_v1/wopr`; `goarch` includes 386; Windows `.tar.gz`; `<project>_<version>_checksums.txt`; ldflags set `main.version` |
| **GL1** v1's pattern `tea\.(Tick\|Every\|Sequence)\(` matches calls | forbidigo v2.3.1 `forbidigo.go:223-258`; golangci-lint run on a module using bubbletea v2.0.10 | False: 0 issues |
| **GL2** The draft's `^(Tick\|Every\|Sequence)$` + `pkg` + `analyze-types` | `forbidigo.go:307`, `:335-337`; runs | False: 1 issue (dot import). `^tea\.(Tick\|Every\|Sequence)$` with the same settings: 7, incl. alias and dot import |
| **GL3** `linters.exclusions.rules` exempts `clock.go` | run; `.golangci.reference.yml:4668-4690` | True; `warn-unused` reports a stale rule |
| **GL4** depguard can forbid Bubble Tea outside `ui` | run; depguard v2.2.1 `settings.go:61-151` | True; default `max-same-issues: 3` hid a fourth hit |
| **GL5** `default: standard` = errcheck, govet, ineffassign, staticcheck, unused | `golangci-lint linters`; `lintersdb/manager.go:136-165` | True |
| **GL6** gofmt, goimports and gofumpt together | `goformatters/meta_formatter.go:33-60`; run | True; fixed order |
| **GL7** golangci-lint built with an older Go refuses a newer module | v2.5.0 (go1.25) and v2.14.0 built with go1.26.8; `config.go:116-145`, `goutil/version.go:13-36` | True: exit 3; the target is the `toolchain` line if present |
| **PL1** v1: Go 1.27 supports macOS 12 (L215, L468) | go.dev/doc/go1.27#darwin; go.dev/wiki/MinimumRequirements; go1.27.1 `cmd/link/internal/ld/macho.go:449` | False: macOS 13 Ventura; linker stamps 13.0.0 |
| **PL2** `ubuntu-22.04` is being retired | actions/runner-images issue #14254 | Deprecated from 2026-09-17; brownouts March–April 2027; unsupported 2027-04-17 |
| **PL3** `macos-latest` "changes without notice"; Intel labels | runner-images README; docs.github.com hosted-runner reference; GitHub changelog 2026-02-26, 2025-09-19 | Partly: arm64 macOS 26; moves are announced. `macos-26-intel`, `macos-15-intel` exist; `macos-13` retired 2025-12-04; `macos-14` unsupported from 2026-11-02 |
| **PL4** arm64 Linux and Windows runners are free for public repos | GitHub changelog 2025-08-07; hosted-runner reference | True: `ubuntu-24.04-arm`, `windows-11-arm` |
| **PL5** Windows runners check out with `core.autocrlf=true` | runner-images `Install-Git.ps1:26-37`; Git for Windows `install.iss:2358`, `:3244-3251`; clone with `autocrlf=true` | True; `* text=auto eol=lf` gives LF |
| **PL6** Git Bash fails on CRLF scripts | MSYS2 bash patch `0005-bash-4.3-msys2-fix-lineendings.patch`; msys2-runtime `spawn.cc:1255` | False (source only): MSYS2 bash drops `\r`; Linux bash fails |
| **PL7** setup-go caching and the `toolchain` line | setup-go `action.yml:15-17`, `src/installer.ts:652-662`, `src/main.ts:233-248`, `docs/advanced-usage.md:185`; run of `dist/setup/index.js`; local `GOTOOLCHAIN=local` build | Caches by default. Uses `toolchain` only if `GOTOOLCHAIN` is not `local` (spec 1.27.1 vs 1.26.0); a patchless `go` line takes the cached patch; `local` ignores a newer `toolchain` line silently |
| **PL8** Dependabot's pre-commit support reads only YAML | dependabot-core `pre_commit/.../file_fetcher.rb:13`; GitHub changelog 2026-03-10; Dependabot options reference | True; 3-day default cooldown |
| **PL9** Keyless attestations on a free public repo | actions/attest and attest-build-provenance READMEs; docs.github.com artifact attestations | True; attest-build-provenance v4 wraps `actions/attest@v4` |
| **PL10** Push protection is free for public repos | docs.github.com "About push protection", "About secret scanning" | True; repository-level is off by default, user-level on; bypass with a reason allowed |
| **EC1** `notnil/chess` archived; maintained fork | GitHub repo page; proxy `@latest`; fork `go.mod`; `LICENSE` files | Archived 2025-01-25; `corentings/chess/v2` v2.6.0 (2026-08-17), `go 1.25.0`, MIT |
| **EC2** Fan repositories' licences and contents | Clones: built1n `COPYING`, `MAP`; abs0 `wargames.sh:3-28`, `:170-173`; elfuska `README.md`, `wargames.bas:20`, `:5181-5300` | built1n GPL-3.0-or-later, `MAP` CC BY-SA 4.0; abs0 BSD-2-Clause; elfuska none. v1 L57-61 = abs0 `:170-173`; v1's map shares fragments with built1n `MAP` |
| **EC3** Homebrew taps for unsigned binaries | brew.sh 5.0.0, 6.0.0, 7.0.0 posts; docs.brew.sh Tap-Trust; GoReleaser `homebrew_casks` docs | Unsigned casks fail Gatekeeper; taps need `brew trust`; tap publishing needs a non-`GITHUB_TOKEN` token |
| **EC4** Trademarks | USPTO TSDR, Reg. 2,690,923 and serial 98472575 | WARGAMES: MGM, class 41, live. WOPR: Reg. 8,024,966, Frontier Technology, Inc., class 42, 2025-11-11 |
| **EC5** The film's terminal | Mini-Micro Systems, June 1984, pp. 135-145; CIO, 2013-06-17; 48k.ca/wgascii.html | 17" Electrohome B&W; off-screen CompuPro; 24×80 with 34 custom line glyphs; mixed-case input |
| **EC6** Dialogue order and LOGON | Subtitle transcript (springfieldspringfield.co.uk); built1n `TRANSCRIPT`; abs0 `wargames.sh` | Order confirmed; LOGON before the header; unknown IDs, incl. `Falkens-Maze`, terminate, `HELP`/`LIST` commands do not; sources say "USER ACCOUNT", "COUNTY" |
| **X1** Stdlib `flag` stops at the first positional | Local program, Go 1.27.1 and 1.24.7 | `gtw --instant`: instant false, `Args()` `[gtw --instant]`; `-is 1`, `-pchess` rejected; `-p=chess` accepted; `-- gtw -i` leaves `[gtw -i]` |
| **X2** `\| head -1` and SIGPIPE | go1.27.1 `os/file_unix.go:231-238`, `os/file_windows.go:145`; 500-run loop | Per-line writes: 25–35/500 exit 141; 0 with `signal.Ignore` or one write. No EPIPE mapping on Windows |
| **X3** Seeded `math/rand` and `math/rand/v2` are stable | go1.27.1 `math/rand/v2/regress_test.go:8`, `math/rand/regress_test.go:8`, `math/rand/v2/pcg.go:24` | True; `NewPCG(seed1, seed2 uint64)` |
| **X4** `*chess.Position` is not goroutine-safe | notnil `position.go:99-104`; corentings `position.go:246-262`; `-race` run | True: lazy `ValidMoves` cache races |
| **X5** The renderer drops most escape sequences | ultraviolet `styled.go:191-237`; capture via `tea.WithOutput` | OSC 52, CUP, BEL, BS dropped; SGR, OSC 8 kept; unterminated OSC blanked later rows |
| **X6** Space is `"space"` in Bubble Tea v2 | ultraviolet `key.go:391-396` | True |
| **X7** Colour conversions; flash and `NO_COLOR` rules | colorprofile v0.4.3 run; no-color.org; WCAG 2.2 SC 2.3.1 | ANSI-16: `#DCE6F0`, `#9EC5FF` → 12; `#FFB000`, `#FF3B30`, `#FFD60A` → 9; `TERM=xterm`/`linux` pick ANSI; `NO_COLOR=yes` keeps truecolor. WCAG: ≤ 3 flashes/s |
| **X8** A fresh pty is 0×0 | `os.openpty()` + `TIOCGWINSZ`; Bubble Tea in that pty | (0, 0); nothing renders until a size is set |
| **X9** Mouse wheel on the alternate screen | GNOME VTE issue 2035; Launchpad bug 291184; search of bubbletea, ultraviolet, x/ansi for 1007 | VTE sends arrow keys; none sets or clears 1007. Other terminals untested |
| **X10** Registry import cycle; test imports | Local module (`games` ↔ `games/chess`); `go list -deps` with and without `-test` | `import cycle not allowed`; `-deps` omits `_test.go` imports |
| **X11** A time budget makes search depth machine-dependent | Local iterative deepening, 300 ms budget, normal and `-race`, 3 runs each | Depth 6 vs 5, different move |
| **X12** Width methods differ | ultraviolet `buffer.go:617`; bubbletea `tea.go:802-805` | wcwidth unless mode 2027; family emoji 2 vs 6 cells |
| **X13** `go install` toolchains and build info | go1.27.1 `cmd/go/internal/toolchain/select.go`; go.dev/doc/toolchain; local `go install …@latest` | Auto mode switches toolchains; `Main.Version` set, no `vcs.*` |
| **X14** SDK and tcell sizes | Local builds as SZ1 | `anthropic-sdk-go` v1.78.0 client 11.1 MB vs stdlib 6.6 MB; tcell v2.13.10 hello 2.9 MB vs 4.1 MB |
| **X15** `misspell` checks string literals | golangci-lint v2.14.0 `pkg/golinters/misspell/misspell.go:89-97` | True unless `mode: restricted` |
| **X16** Pull-request caches cannot reach tag runs | docs.github.com dependency caching ("only be restored by re-runs of the pull request") | True |
| **X17** The draft's flow-mapping `concurrency` snippet | PyYAML 6.0.1; zizmor | Invalid: "expected ',' or '}', but got '{'" |
| **X18** Snapshot versions, publishing order, immutable tags | GoReleaser `pipe/snapshot/snapshot.go:23`, `client/github.go:541-543`, `:650-654`; docs.github.com immutable releases; go.dev/doc/modules/publishing; go.dev/ref/mod | `{{ .Version }}-SNAPSHOT-{{ .ShortCommit }}`; undraft at end; immutability only for later releases; Go pins first-fetched content |
| **X19** Archive contents, licences, reproducibility | GoReleaser `archive.go:83-91`, `pkg/archive/tar/tar.go:78-79`; `go list -deps`; go1.27.1 `LICENSE`; two `release --skip=publish` runs, mtimes touched | Defaults `license*`, `readme*`, `changelog*`; 17 modules (15 MIT, 2 BSD-3); BSD-3 binary clause; archive hashes differ |
| **X20** goreleaser-action installs `~> v2` | goreleaser-action v7.2.3 `action.yml:14-16`, `src/goreleaser.ts:57-67` | True; checksum failure only warns |
| **X21** GitHub Actions settings and events | docs.github.com: events that trigger workflows; repository Actions settings; `GITHUB_TOKEN`; disabling workflows; required status checks | Fork PR runs belong to the base repo; workflow PR creation off by default; `GITHUB_TOKEN` PRs await approval; schedules stop after 60 quiet days; SHA-pin and approval policies exist; one check per matrix combination (names not checked live) |
| **X22** Windows contributor toolchain | git-for-windows/build-extra `make-file-list.sh:34-36`, `:209-218`; go1.24.7 `cmd/go/internal/cfg/cfg.go:170-172`, `work/init.go:163-172`, `internal/platform/supported.go:29-30` | No `make`, no util-linux; cgo off without a C compiler; `-race` needs cgo, amd64 only on Windows |
| **X23** `net/http` keeps custom headers on redirect | go1.27.1 `net/http/client.go:826-827`; local 307 between two hosts | True: `x-api-key` and body reach the second host |
| **X24** v1's montage list comes from abs0 | abs0 `wargames.sh` (010ed92) lines 590-680 vs v1 L88-97 | 40 of 45 in order; `BURMESE THEATERWIOE` at line 612 |
| **X25** Claude Code and AGENTS.md | code.claude.com/docs/en/memory (AGENTS.md section) | Loaded natively only without a CLAUDE.md; otherwise `@AGENTS.md` |
| **X26** rumdl on v1 | rumdl 0.2.58, `--disable MD013,MD033,MD041` | 23 issues (M-4); all fixed by `rumdl fmt` |
| **X27** Test mechanics | Local modules for `-update` and `-fuzz`; 20,000-game tic-tac-toe simulation; `charmbracelet/x/xpty` v0.1.4 source | `-update` undefined elsewhere fails; `-fuzz` refuses several packages or targets; perfect play beats random 96.5–99.5% as X, 77.8–80.8% as O; xpty has pty and ConPTY |
| **X28** Go builds do not depend on the host OS | go.dev/blog/rebuild | True since Go 1.21 |
| **X29** Concurrent map writes are fatal | go1.27.1 `internal/runtime/maps/runtime.go:160`, `runtime/panic.go:1245-1255` | `fatal("concurrent map writes")`; not recoverable |
| **X30** v1's `govulncheck@latest` hook vs a tool module | Repo with `go 1.27`, local Go 1.24.7, `GOTOOLCHAIN=auto`; then `tools/go.mod` | v1's form fails (x/vuln v1.8.0 switches to go1.26.8, which refuses the module); `go tool -modfile=tools/go.mod govulncheck ./...` passes |
