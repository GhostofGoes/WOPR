# WOPR — Architecture & Scope Plan (v2)

_Status: v2, 2026-10-06. Supersedes v1 (commit `ada63a0`, which never reached `main`).
[`docs/reviews/PLAN-v1-review.md`](reviews/PLAN-v1-review.md) is the adversarial review of v1; finding ids such as
A-3 are cited inline where v2 answers them. [`docs/reviews/PLAN-v2-review.md`](reviews/PLAN-v2-review.md) reviews
this version. Implementation follows the milestones in §15._

**How to read this plan.** It states intent, contracts and constraints. Config files appear only as **M0 seeds**
in Appendix A. Each seed was run against the real tool on 2026-10-06. When M0 lands, the files in the
repository become authoritative and Appendix A is deleted (M-2). After M0, conventions live in `AGENTS.md`. This
plan links to it rather than repeating it (M-1).

**What changed from v1, in one paragraph.**

- **Contracts.** Games and the persona now speak one protocol: dynamic input mode, `Clear`, `Wait`,
  `SetLayout`, `Think` for any slow work, and a semantic canvas for colour. A Bubble-Tea-free host core runs
  them. A catalog package wires the game constructors without an import cycle.
- **Persona.** The brain gets a value snapshot and returns effects. LOGON is a closed table. The film's climax
  has one owner.
- **Toolchain.** CI no longer silently downgrades it.
- **Releases.** Release builds are gated, reproducible, attested and ship their licence notices. The film text
  has per-line provenance, and third-party text is credited.
- **Owner decisions.** macOS 26 only. Dependabot is dropped. `main` is PR-only. Movie mode is M6, and the
  optional LLM brain moves to M7.

---

## 1. Requirements

Source codes: **U** = stated by the owner. **D** = derived from a U requirement. **P** = a plan default the owner
may override.

| # | Requirement | Src | Acceptance |
|---|---|---|---|
| R1 | Go terminal-UI recreation of WOPR from *WarGames* (1983); binary `wopr`. | U | — |
| R2 | Every game on the film's `LIST GAMES` (15 entries), plus the tic-tac-toe ending. | U | Each game meets the definition of done (§6). |
| R3 | Persona faithful to the film and to the brother's "WOPR Simulator" prompt: WOPR speaks ALL CAPS, addresses the user as PROFESSOR, is terse, numbers its lists, refuses invalid moves. User input is echoed as typed, in mixed case, as in the film (F-1). | U | Persona table tests and golden transcripts (§9). |
| R4 | Single static binary for **Linux and Windows**. | U | CI smoke-runs the release binaries natively (§11). |
| R5 | **macOS 26 (Tahoe)** on Intel and Apple Silicon. Older macOS versions are out of scope. | U (2026-10-06) | Smoke runs on `macos-26` and `macos-26-intel`. |
| R6 | Size: **target ≤ 10 MB, hard limit 15 MB** per binary (decimal MB). | U (confirmed 2026-10-06) | CI warns above 10,000,000 bytes and fails above 15,000,000. |
| R7 | `wopr` starts the TUI. `--version`, `--help`, `--games`, `-p/--play <game>`. | U | CLI tests (§5). |
| R8 | Optional LLM hookup, or purely scripted responses. | U | Scripted from v0.1; LLM in M7, opt-in (§4.7). |
| R9 | Every long flag has a one-letter shorthand. | v1 decision 6 (source not recorded) | `flags_test.go`. |
| R10 | Public repository: no secrets, least-privilege CI, reproducible and attested releases. | D | §11, §12. |
| R11 | First release after the film set pieces; the remaining games ship in minor releases. | U (2026-10-06) | §15. |
| R12 | **Movie mode** (`-m/--movie`) replays the WOPR terminal scenes from the film. | U (2026-10-06) | §7, M6. |
| R13 | No Dependabot for now. Dependencies are updated by hand on a schedule (§11.5). | U (2026-10-06) | No `dependabot.yml`. |

**Supported platforms.**

| OS | Versions | Architectures | Tested on (CI) |
|---|---|---|---|
| Linux | Ubuntu 22.04 and newer, and any distribution with an xterm-class terminal | amd64, arm64 | `ubuntu-24.04` (incl. an `ubuntu:22.04` container), `ubuntu-24.04-arm` |
| macOS | 26 Tahoe | amd64, arm64 | `macos-26-intel`, `macos-26` |
| Windows | 11 | amd64, arm64 | `windows-2025`, `windows-11-arm` |

Binaries are static (`CGO_ENABLED=0`). Go 1.27's own floor is macOS 13, so builds run fine on 26 (D-5). The
`ubuntu-22.04` runner image is deprecated and unsupported from 2027-04-17. The 22.04 claim is therefore tested
by running the smoke step inside an `ubuntu:22.04` container (B-5).

---

## 2. Film research

### 2.1 Sources and provenance

Primary sources:

- the film itself (a viewing pass in M5 confirms or corrects every *reconstructed* line);
- *Mini-Micro Systems*, June 1984, "Making Wargames computers compute"; CIO, 2013, "The technology of
  WarGames";
- subtitle-derived dialogue.

Secondary sources: fan transcripts and simulators.

| Fan source | Licence |
|---|---|
| `built1n/wargames` | GPL-3.0-or-later; its `MAP` is CC BY-SA 4.0 |
| `abs0/wargames` | BSD-2-Clause per-file headers, © David Brownlee |
| `elfuska/wargames` | No licence; "credit Andy Glenn" |

These are references only. No code or art is taken from any of them. Exactly two text items are taken from
`abs0/wargames` under BSD-2, and both are credited in `NOTICE.md` (owner decision 2026-10-06, L-4):

- the backdoor header block;
- the montage scenario names.

**Provenance tags.** Every script line, scene step and asset carries one. A test fails on any entry without a
tag.

| Tag | Meaning |
|---|---|
| `film` | Confirmed against the film in the M5 viewing pass |
| `reconstructed` | From transcripts or subtitles; not yet confirmed |
| `third-party:abs0/wargames@010ed92:wargames.sh` | Copied; BSD-2 notice in `NOTICE.md` |
| `original` | Written for this project |
| `prompt` | Adapted from the brother's prompt. Excluded from builds until he grants written permission **and** a licence (MIT or CC0) (L-3, L-5). |

### 2.2 Facts that shape the design (F-1)

- David's terminal was a 17-inch **Electrohome black-and-white** monitor: white text on black. The default
  `imsai` theme is white phosphor, and green and amber are alternates.
- WOPR's text was generated off-screen by CompuPro S-100 systems on a **24-line × 80-column** display. The map
  glyphs came from 34 custom line-segment characters. This supports the 80×24 target, and it is why the GTW map
  is drawn as **original line-segment-style ASCII art**.
- **WOPR's output is upper case. David's typing appears in mixed case** ("Hello.", "Love to. How about Global
  Thermonuclear War?", "Las Vegas").
- **LOGON comes first.** The `#45 11456 …` header and status burst appear only *after* `Joshua` is accepted.
  An unrecognised ID prints two lines and drops the connection.
- The GTW-vs-chess exchange is a separate scene after the greeting (§2.3).

### 2.3 Canonical screen text

The full text, with a provenance tag per line, is in **Appendix B**. In M1 it moves into `internal/wopr/lines.go`
and `internal/assets/`, which become authoritative. Two fan-source conflicts are recorded there and settled in
the M5 viewing pass:

- `USER ACCOUNT` vs `USER ACCOUNT NUMBER`;
- `CITY AND/OR COUNTY NAME` vs `COUNTRY`.

### 2.4 Visual language

- **Console**: full-bleed text with no chrome. WOPR lines are double-spaced and revealed at modem speed, with a
  solid block cursor.
- **NORAD big board**: dark blue field, cyan/blue outlines, red incoming and yellow outgoing tracks, white
  labels, a DEFCON column of boxed numerals 5…1, and `** … **` notices. This is the `norad` theme.
- **WOPR cabinet**: rows of lights and a `W.O.P.R.` nameplate. This becomes an optional one-row front panel,
  which also shows "thinking" (§4.3).
- **Verbiage**: ALL CAPS, terse, declarative; never asks clarifying questions; numbered lists `1.  2.  3.`;
  refuses invalid actions with `** IMPROPER REQUEST **`-style lines; calls the user PROFESSOR.
- **Anti-pattern**: bordered panes and labelled input boxes. Borders appear only inside the big board and in
  NORAD notices.

The target screens are in **Appendix C**, all at exactly 80×24: the greeting, the GTW big board, and the chess
panel. The GTW mockup uses a placeholder for the map. The art is drawn from scratch in M2 (L-4).

---

## 3. Decisions

| # | Decision | Choice | Src |
|---|---|---|---|
| 1 | TUI stack | **Bubble Tea v2.0.10 + Lip Gloss v2.0.6**, no bubbles. Measured 3.6–3.9 MiB stripped on all six targets. No fallback renderer is planned (D-3). | U |
| 2 | Brain | **Scripted** behind `Brain`. The LLM provider is **M7**, optional and opt-in. | U |
| 3 | Military sims | Short turn-based mini-wargames on one `sim` engine, specified before any sim is built (G-2). | U |
| 4 | LOGON | Film-faithful: a closed command table, the `Joshua` backdoor, a hint after 3 failures. `--play` and `--movie` bypass it (A-17). | U |
| 5 | Platforms | Linux and Windows (amd64, arm64); **macOS 26 only** (amd64, arm64). | U |
| 6 | Arg parsing | Stdlib `flag`. Every long flag has a shorthand. One bare positional means `--play`, and flags may follow it (C-1). | v1 |
| 7 | Persistence | None. Flags and `WOPR_*` environment variables only. | P |
| 8 | Toolchain | `go 1.27` + `toolchain go1.27.1`, as in v1 (D-1 closed by the owner). CI must not set `GOTOOLCHAIN` before `setup-go` (D-6). | U |
| 9 | Lint | **prek** with **`prek.toml`**; seed in Appendix A.1. | U |
| 10 | Size | Target ≤ 10 MB, hard limit 15 MB. | U |
| 11 | CI topology | Build once with GoReleaser; run each binary natively on its OS/arch (B-2). | U |
| 12 | Docs | README from M0. **AGENTS.md from M0** is the single source for commands and conventions. `CLAUDE.md` contains `@AGENTS.md` (M-1). | U |
| 13 | Releases | v0.1.0 after M2, then a minor release per milestone, then v1.0.0 after M5 (P-2). | U |
| 14 | Bridge | Minimal: WOPR bids all four seats by point count. The user always plays the declaring side, as declarer plus dummy, with seats rotated when East/West win the contract. Passed-out deals are redealt (G-3). | U |
| 15 | Chess rules | `github.com/corentings/chess/v2` v2.6.0, the maintained MIT fork of the archived `notnil/chess`. Search lives in `games/ai` (G-1). | P |
| 16 | Branching | **`main` is PR-only**. One required check, `ci-ok`. Squash merges only. | U |
| 17 | Dependabot | **Not used.** Manual update cadence plus a weekly staleness and vulnerability report (§11.5). | U |
| 18 | Montage names | Keep the list and credit abs0 under BSD-2 now. Verify in the M5 viewing pass. | U |
| 19 | Movie mode | **M6**: `-m/--movie` replays the film's WOPR terminal scenes (§7). | U |

---

## 4. Architecture

### 4.1 Packages and the import DAG

Module `github.com/GhostofGoes/WOPR`, binary `wopr`. Import edges not listed below are forbidden (A-6, A-13).

```text
cmd/wopr             → cli, ui, version, games/catalog, (llm in M7)
internal/version     → stdlib
internal/cli         → games (Registry, Resolve), movie (scene index), version
internal/proto       → stdlib                         the program protocol, canvas, keys, rand
internal/proto/host  → proto                          Bubble-Tea-free runner + Scheduler interface (A-14)
internal/prompt      → stdlib                         normalise, clauses, menus, yes/no
internal/theme       → proto, charm.land/lipgloss/v2, github.com/charmbracelet/colorprofile
internal/ui          → proto, proto/host, prompt, wopr, games, theme, assets, movie,
                       charm.land/bubbletea/v2, charm.land/lipgloss/v2, github.com/charmbracelet/x/ansi,
                       github.com/charmbracelet/colorprofile
internal/wopr        → proto, prompt, games (types only; the registry is injected)
internal/games       → proto                          types only: Info, Status, Game, Registry, Resolve
internal/games/{ai,cards,board}       → proto, games
internal/games/<game>                 → proto, prompt, games, games/{ai,cards,board}, games/ending, assets, sim
internal/games/ending                 → proto, prompt, assets
internal/games/catalog                → games, every games/<game>      the only place constructors are wired
internal/games/gamestest              → proto, games    stub programs; importable from tests only
internal/games/testkit                → proto, proto/host, games, golden   tests only
internal/sim         → proto                          (M4)
internal/assets      → stdlib                         embedded art, scenario names (with provenance)
internal/movie       → proto, prompt, assets, games/gtw, games/ending      (M6)
internal/golden      → stdlib                         golden-file helper (Q-3)
internal/archtest    → stdlib                         enforces this table
internal/e2e         → stdlib, github.com/charmbracelet/x/xpty   build tag e2e (Q-1)
internal/tools/*     → stdlib                         sizegate, smoke, notices (Go programs, not shell)
internal/llm         → proto, wopr, net/http          (M7)
```

- **Only `ui` imports Bubble Tea.** Only `ui` and `theme` import `charm.land/*`.
- **`games` holds types only.** `games/catalog` imports every game package and returns the ordered
  `Registry` of `{Info, New func(Env) Game}`. That removes the `games → games/<game>` import cycle (A-13).
- **Injection.** `cmd/wopr` passes the registry to `cli`, `wopr` and `ui`. Their tests use fakes from
  `gamestest`, which must not appear in non-test imports (G-5).
- **`internal/archtest`** reads each package's `Imports`, `TestImports` and `XTestImports` from
  `go list -json ./...` and checks every edge against this table. It also checks that every `Playable`
  registry entry has a constructor, and that the running Go version equals `go.mod`'s `toolchain` line (D-6).
- **Editor feedback**: `depguard` (Appendix A.2) reports Bubble Tea imports outside `ui`. `archtest` is
  authoritative.

```text
cmd/wopr/main.go             flags → dispatch → exit code; NO_COLOR and TTY handling (§4.3, §5)
internal/version/            Version/Commit/Date via -X; fallback to debug.ReadBuildInfo(); "unknown" if absent (D-4)
internal/cli/                Parse(args, reg, scenes, stdout, stderr) (Config, Action, error); usage; --games
internal/proto/              event.go output.go program.go canvas.go key.go pace.go rand.go
internal/proto/host/         runner.go (program stack, input mode, Esc machine, Think jobs, timers, pacing)
internal/prompt/             normalize.go clause.go menu.go yesno.go
internal/theme/              palettes: Style → (truecolor, ANSI-16 index, ASCII attribute) (U-7)
internal/ui/                 app.go (thin router) clock.go (the single tea.Tick) adapter.go keys.go
  console/                   scrollback, typewriter, line editor, sanitize, render
  screens/                   dial, too-small card, canvas renderer, front panel
  testutil/drive.go          synchronous driver with a fake Scheduler (A-14)
internal/wopr/               session.go logon.go intent.go scenes.go commands.go brain.go scripted.go lines.go
internal/games/              registry.go game.go
  ai/ cards/ board/          search (depth/node limits, time cap), decks and tricks, board drawing and cursor
  ending/                    the climax: tic-tac-toe self-play → montage → final dialogue (A-16)
  catalog/ gamestest/ testkit/
  falkensmaze/ blackjack/ ginrummy/ hearts/ bridge/ checkers/ chess/ poker/ gtw/ tictactoe/
  fightercombat/ guerrilla/ desertwarfare/ airtoground/ theaterwide/ biotoxic/
internal/sim/                M4 engine
internal/assets/             gtw_map.go (original), scenarios.go (third-party:abs0), banner.go (original)
internal/movie/              director.go scenes/*.go (M6)
internal/golden/ internal/archtest/ internal/e2e/ internal/tools/{sizegate,smoke,notices}/
tools/go.mod                 Go tools: govulncheck, gitleaks (pinned, checksummed by go.sum)
tools/release/go.mod         Go tool: goreleaser (T-11)
```

### 4.2 Runtime flow and state ownership

1. **`main`** parses flags. Print-and-exit actions (`--version`, `--help`, `--games`, `--licenses`) write
   plain, unstyled text in one buffered write. They ignore `SIGPIPE`, and they treat `EPIPE` (Unix) or
   `ERROR_NO_DATA`/`ERROR_BROKEN_PIPE` (Windows) as success (C-3, U-11).
2. **TTY check.** If stdout is not a terminal, or `TERM=dumb`, `main` prints one line naming the cause and
   exits 2. On Windows the line suggests Windows Terminal. A non-TTY stdin is fine, because Bubble Tea opens
   `/dev/tty` or `CONIN$` itself. Bubble Tea's "error opening TTY" also maps to the same message and exit 2
   (U-4).
3. **NO_COLOR.** If `NO_COLOR` is set to any non-empty value, `main` passes
   `tea.WithColorProfile(colorprofile.Ascii)`. colorprofile alone parses it with `ParseBool` and ignores
   `NO_COLOR=yes` (U-7).
4. **Screen.** The TUI runs on the **alternate screen** and paints the theme background into every cell.
   It does not use OSC 11, which ignores `NO_COLOR` and resets the terminal to its default background on exit.
   On a normal exit (0 or 130), the normal screen gets one line: `--CONNECTION TERMINATED--` (U-1).
5. **Ownership** (A-5).
   - `proto/host.Runner` owns the **program stack**. The persona is the root program. `Launch` pushes a game;
     `Done` pops it and delivers `GameOver{Result}` to the program below.
   - `wopr.Session` owns the **conversation phase**: `Dialing → Logon → Disconnected → … → Greeting → Shell ⇄
     Ending`. There is no `InGame` phase: "a game is active" is simply "the stack is deeper than one".
   - `ui` owns only presentation state: the screen mode (`Console`, `Panel`, `Full`, `TooSmall`), derived from
     the top program's layout and the terminal size, plus scroll position.
6. **Dial and LOGON** (A-17).
   - **Dialing**: the modem dial animation runs as `Say`/`Wait` steps of the persona. Any key or `--instant`
     skips it. A re-dial uses a short version.
   - **Logon** is a closed table. Input is normalised and matched exactly; no intent parsing, no brain, no
     numbered selection.

     | Input | Response |
     |---|---|
     | `HELP LOGON` | `HELP NOT AVAILABLE` |
     | `HELP GAMES` | The games blurb |
     | `LIST GAMES` | The list. No selection is armed. |
     | `JOSHUA` | The backdoor |
     | `LOGOFF`, `LOG OFF`, `EXIT`, `QUIT` | Exit 0 |
     | Empty line | `LOGON:` again |
     | Anything else, including a game name | `IDENTIFICATION NOT RECOGNIZED BY SYSTEM` / `--CONNECTION TERMINATED--`, then `Disconnected` → `Wait` 1 s → `Dialing` |

   - The failure counter lives in `Session`. After the third termination, an `original` hint line is printed
     before the next `LOGON:`.
7. **Backdoor**: the header block, the status burst, `Clear`, `GREETINGS PROFESSOR FALKEN.`, then the
   greeting scene and **Shell**.
8. **Shell dispatch.** Sanitised input goes, in order, to: host commands (`LOGOFF` family, at every depth of the
   stack) → **active scene** → **phase-gated commands** (`HELP`, `HELP GAMES`, `LIST GAMES`, `PLAY <game>`) →
   **explicit intent** (§4.6) → **offer acceptance** (§4.6) → **Brain** (asynchronous, via `Think`).
9. **`--play <game>`** pushes the game straight onto a persona in `Shell`, with the greeting marked as done.
   `--movie` runs the movie director as the root program instead of the persona (§7).
10. **Exit codes.**
    - `LOGOFF`, `Ctrl+D` on an empty input line, and SIGTERM exit 0.
    - `Ctrl+C` and SIGINT exit 130.
    - `Ctrl+Z` suspends on Unix (`tea.Suspend`) (U-10).

### 4.3 Console, clock, cursor, input

- **One clock** (A-9). `ui/clock.go` implements `host.Scheduler` with the only `tea.Tick`. Consumers are the
  typewriter, each program's `Animate`, `Wait` timers, `Blink` cells, the front panel and the dial.
  - The next tick is armed for the **earliest pending deadline**. When the set of consumers changes, the clock
    bumps its generation and re-arms.
  - A late tick delivers one coalesced `dt`. With no busy consumer, no timer runs.
  - The golden driver substitutes a fake `Scheduler`, so it never calls `tea.Tick` (A-14).
- **Pacing.** A dt-based typewriter: about 30 chars/s for speech, fast for tables, instant for boards.
  `--instant` / `WOPR_INSTANT=1` and "skip" both mean "advance by infinity". Movie mode adds a user-typing pace
  (§7).
- **Cursor.** `tea.NewCursor` gives a blinking block. It is parked at the end of the revealing line, or on the
  input line.
- **Skip policy** (U-3).
  - While text is revealing, the first key flushes the queue.
  - In line mode, a printable key is *also* typed into the input line (typeahead kept). Enter and Space act
    purely as "skip" only when the input line is empty.
  - In key mode, the key flushes *and* is delivered to the game.
  - PgUp/PgDn always scroll. Esc only skips. Ctrl+C always quits.
  - The README says "any key skips; what you type is kept".
- **Thinking.** While a `Think` or a Brain reply is pending:
  - the cursor stops blinking and the input line shows `PROCESSING` with a dot group cycling at 2 Hz;
  - the front panel lights animate when the panel is shown;
  - keys are **buffered** into the line editor but Enter is held until the reply lands, the same typeahead
    rule as during a reveal.
- **Esc machine** (host-owned; games never see Esc).
  - During a reveal, Esc skips.
  - In a game, the first Esc shows `** PRESS ESC AGAIN TO END GAME **` for 3 s. A pending `Think` keeps
    running. A second Esc within 3 s cancels the `Think` context (its late result is dropped by generation)
    and pops the game with `Done{Aborted}`. Any other key disarms it.
  - While a Brain reply is pending, Esc cancels it and prints `** REQUEST CANCELLED **` (`original`).
  - At the Shell, Esc does nothing. The ending sequence cannot be aborted with Esc; Ctrl+C still quits.
- **Sanitising** (U-2). Two functions, both fuzzed.
  - `SanitizeInput` serves typed and pasted text:
    1. `ansi.Strip`;
    2. drop C0, C1 and DEL;
    3. map `\t` to a space, and join pasted lines with one space;
    4. replace invalid UTF-8 with U+FFFD;
    5. cap at 256 grapheme clusters.
  - `SanitizeText` serves any text from outside the binary (Brain, LLM): `ansi.Strip`, drop controls except
    `\n`, expand tabs to 8-column stops, and split on `\n` into logical lines.
- **Width** (U-6). The line editor and the wrapper measure with the same method as the renderer: `wcwidth` by
  default, and grapheme widths once the terminal confirms mode 2027.
- **Scrollback.** Logical lines are wrapped at render time and cached per width, capped at 2000. New output
  auto-follows. `Clear` starts a new page, and history stays above it. `Render(w, h)` emits exactly `h` rows.
- **Layout geometry.**
  - The layout width is `min(cols, 80)`, centred.
  - **Console**: every row for console text, then the input row.
  - **Panel**: the game's `Info.PanelRows` (≤ 12) on top. The console strip takes the rest, which is at least
    10 rows at 24.
  - **Full**: the view takes `H − 4` rows, then a 3-row strip and the input row.
  - The optional front panel takes the last row. It is shown by default in `norad`, and `WOPR_PANEL=1/0`
    overrides that.
  - A registry test checks that every game fits at 80×24.
- **Too small** (U-8). Below 80×24 the `TERMINAL TOO SMALL` card is shown, and:
  - the typewriter, `Animate`, `Wait` and Blink consumers **pause**;
  - a finishing `Think` is held until the size is valid again;
  - every key except Ctrl+C is dropped.
  No program starts before the first valid size. `ResizeEvent` fires only when a program's layout area
  actually changes.
- **Mouse.** Mouse reporting stays off, so native text selection works. On the alternate screen some terminals
  (VTE) turn the wheel into ↑/↓, which cycles input history. This is documented as known behaviour, and the M5
  accessibility pass revisits it (U-9).

### 4.4 The program protocol (`internal/proto`)

The persona, every game, the ending and the movie director implement one interface. The host runner is the
only code that interprets it (A-1, A-14, A-15).

```go
// internal/proto — stdlib only.
type Program interface {
	Start(Env) []Output
	Handle(Event) []Output
	View(c *Canvas) // draw into the layout area; never called for LayoutConsole
}

type Env struct {
	Seed          uint64 // derive streams with NewRand(Seed, StreamID)
	Width, Height int    // the program's layout area, always ≥ its minimum
	Instant       bool
	Deterministic bool   // --seed given, or under test: AI search obeys Limit, not the clock
	Mode          string // optional launch mode, e.g. "climax" (§6.2)
}

type Layout uint8 // LayoutConsole, LayoutPanel, LayoutFull

// Events: host → program.
type Event interface{ isEvent() }
type LineEvent   struct{ Text string }          // answer to the last Prompt; "" for an empty Enter
type KeyEvent    struct{ Key Key; Rune rune }   // only in key mode; Key: Up Down Left Right Enter Backspace Rune
type TickEvent   struct{ Dt time.Duration }     // at the cadence the program asked for
type ResizeEvent struct{ Width, Height int }
type ThinkDone   struct{ Value any; Err error } // Err is context.Canceled only after a confirmed abort
type GameOver    struct{ Result Result }        // to the program below a popped one

// Outputs: program → host. All are declared here, so the interface stays sealed (A-15).
type Output interface{ isOutput() }
type Say       struct{ Lines []string; Pace Pace }       // typewriter; PaceSpeech, PaceTable, PaceInstant
type Prompt    struct{ Text string }                     // line mode: next LineEvent answers it
type AwaitKeys struct{ Hint string }                     // key mode (maze, board cursor)
type Animate   struct{ Every time.Duration }             // 0 stops
type Wait      struct{ D time.Duration }                 // host-timed pause; skippable; 0 under Instant
type Clear     struct{}                                  // page break
type SetLayout struct{ Layout Layout; PanelRows int }    // e.g. GTW: Console for text, Full for the map
type Redraw    struct{}
type Think     struct {                                  // all slow work runs off the UI goroutine (A-3)
	Fn     func(ctx context.Context) (any, error) // captures only values (a FEN string, not *chess.Position)
	Limit  Limit                                  // MaxDepth / MaxNodes: binding when Env.Deterministic
	Budget time.Duration                          // wall-clock cap; primary only when not deterministic
}
type Launch struct{ Slug, Mode string }               // persona → host: push a game
type Done   struct{ Result Result }                   // pop this program
type Quit   struct{}                                  // LOGOFF: exit 0

type Outcome uint8 // Win, Loss, Draw, NoWinner, Aborted
type Result struct {
	Outcome   Outcome
	Lines     []string // kill ratios, detail
	Next      *Launch  // hand-off without a verdict, e.g. gtw → tictactoe(climax) → ending
	NoVerdict bool     // the persona adds nothing (the ending's last line stands alone)
}
```

Rules:

- **Input mode is dynamic.** `Prompt` switches to line mode and `AwaitKeys` to key mode. A program toggles
  between them as it needs (checkers: type `b6-a5`, or move a cursor). While a `Prompt` is active, an empty
  Enter is delivered as `LineEvent{""}`; GTW ends its target list that way.
- **`Think` contract** (A-3, A-18, G-1).
  - The host runs `Fn` in a `tea.Cmd`. It sets a `context.WithDeadline` for `Budget`, and uses
    `WithCancelCause` for aborts.
  - When the deadline is exceeded, `Fn` returns its best result as a normal `Value`.
  - With `Env.Deterministic` (`--seed`, tests, movie mode), searches stop at `Limit`. Hitting `Budget` there
    fails tests and is logged in seeded play.
  - `Fn` must capture values only. A `*chess.Position` lazily caches its moves and is not goroutine-safe.
  - AI randomness comes from a seed captured in `Handle`.
- **The runner is pure** (`proto/host`). It takes host inputs (keys, lines, Esc, resize, scheduler fires, Think
  results) and returns render requests and async jobs.
  - `ui/adapter.go` turns the jobs into `tea.Cmd`s.
  - `games/testkit` drives the *same* runner with a fake scheduler, and runs `Fn` on a goroutine concurrently
    with `Handle` and `View`, so `-race` sees captured state.
  - The definition-of-done checks therefore exercise shipping code.

### 4.5 Canvas, themes and accessibility

`proto.Canvas` is a `W×H` grid of `Cell{R rune; S Style; A Attr}` (A-2).

- **Style is semantic**: `Text`, `Bright`, `Dim`, `Accent`, `Land`, `Label`, `Incoming`, `Outgoing`, `Target`,
  `Defcon1`…`Defcon5`, `Alert`, `SuitRed`, `SuitBlack`, `Selected`.
- **Attr**: `Blink`, `Reverse`, `Underline`, `Bold`.
- `ui/screens/canvas.go` maps each Style to the active theme. `Canvas.String()` and `Canvas.StyleMap()` make
  goldens readable and independent of the theme.

Each theme defines every Style three ways: a truecolor hex, an explicit **ANSI-16 index**, and an **ASCII
attribute**. Automatic downsampling is not trusted, because it collapsed meaningful pairs in testing (U-7):

| Theme | Text | Bright | Dim | Accent | Background |
|---|---|---|---|---|---|
| `imsai` (default) | `#DCE6F0` / 7 | `#FFFFFF` / 15 / Bold | `#7A8694` / 8 / Faint | `#9EC5FF` / 14 / Bold | `#000000` / 0 |
| `green` (P1) | `#33FF33` / 10 | `#B6FFB6` / 15 / Bold | `#1A8C1A` / 2 / Faint | Bright + Bold | `#000000` / 0 |
| `amber` (P3) | `#FFB000` / 11 | `#FFD27A` / 15 / Bold | `#8A5E00` / 3 / Faint | Bright + Bold | `#000000` / 0 |
| `norad` | `#5AC8FA` / 14 | `#FFFFFF` / 15 / Bold | `#2F6FBF` / 4 / Faint | `#FFD60A` / 11 / Bold | `#02060F` / 0 |

DEFCON: 5 blue (12), 4 green (10), 3 yellow (11), 2 red (9), and 1 is white on red (15 on 1). Incoming tracks
are red (9) and outgoing yellow (11).

**Accessibility rules.**

- **Meaning never rests on colour alone.**
  - Incoming tracks draw `*` and outgoing `+`.
  - The current DEFCON level is `Reverse`. Alerts are `Bold`.
  - Card suits always show their letter.
- **Flashing** is capped at 3 Hz for any change covering a large area of the screen. That includes Blink
  toggles, DEFCON 1 and montage frames. This meets WCAG 2.3.1.
- **`-r/--reduce-motion`** (`WOPR_REDUCE_MOTION=1`) stops Blink, freezes the front panel, and removes the
  montage's acceleration.
- **Art** is pure ASCII.
- **Goldens** of the `StyleMap` are rendered under the TrueColor, ANSI and ASCII profiles.

### 4.6 Persona: session, intent, offers, brain

```go
// internal/wopr/brain.go
type Snapshot struct { // a value copy taken in Handle; safe on another goroutine (A-4)
	Phase   Phase
	Turn    uint64            // increments on every brain request, including cancelled ones
	Seed    uint64
	Said    map[string]bool   // copied
	Flags   map[string]bool   // copied
	Side    string
	Last    *proto.Result     // the previous game's result; replaces v1's PostGame
	History []Exchange        // last 20
}
type Effect interface{ isEffect() }
type MarkSaid struct{ ID string }
type SetFlag  struct{ Key string }
type StartGame struct{ Slug string }
type Reply struct{ Lines []string; Effects []Effect }
type Brain interface {
	Reply(ctx context.Context, s Snapshot, input string) (Reply, error)
}

type Rule struct { // the scripted brain's table; first match wins, specific before general, fallbacks last
	ID      string
	When    PhaseSet               // Greeting | Shell | Ending  (LOGON is the closed table, §4.2)
	After   AfterGame              // Any | AfterWin | AfterLoss | AfterNoWinner | AfterAbort
	Match   func(c prompt.Clause) bool // Exact, HasAll, Re (RE2: linear time)
	Lines   []string               // or Pick []string, chosen with NewRand(Seed, Brain|Turn)
	Once    bool                   // emits MarkSaid; afterwards falls through
	Effects []Effect
}
```

- **Brain calls.** The persona calls the Brain through `Think`: `Fn` calls `Reply` with the snapshot. The reply
  comes back as `ThinkDone`, and the persona applies its `Effects` in `Handle`. Nothing else mutates the
  `Session`. On an error or timeout (scripted: never; LLM: 20 s), the persona answers with a scripted fallback
  line.
- **Explicit intent** (A-7). The input is split into clauses on `. ! ? ; ,` and each clause is normalised:
  1. upper-case;
  2. `[^A-Z0-9']+` collapses to one space;
  3. `LET'S`, `LETS` and `LET US` become `LETS`.

  A clause starts a game when all of these hold:
  - after optional fillers (`LATER`, `LOVE TO`, `OK`, `OKAY`, `YES`, `SURE`, `WELL`, `FINE`, `THEN`), it
    begins with `PLAY`, `LETS PLAY`, `HOW ABOUT`, `I WANT TO PLAY`, `I'D LIKE TO PLAY` or `CAN WE PLAY`;
  - that verb is followed by a game number, name, slug or alias, with optional `A GAME OF` / `SOME`;
  - no negator (`NOT`, `DON'T`, `DONT`, `NO`, `NEVER`) comes before the verb in that clause.

  Input that is exactly a game name, slug or alias also counts.

  **Numbered selection** is armed by `LIST GAMES` and disarmed by the next non-numeric input.
- **Offers** (A-7). Some WOPR lines arm an offer that lasts one turn:
  - `SHALL WE PLAY A GAME?` arms `Offer{Any}`. YES gets `WHICH GAME?` (`original`) and arms selection.
  - `WOULDN'T YOU PREFER A GOOD GAME OF CHESS?` and `HOW ABOUT A NICE GAME OF CHESS?` arm `Offer{chess}`.

  `YES`, `Y`, `OK`, `SURE`, `LOVE TO` and `FINE` accept. `NO` declines in character. Explicit intent outranks
  an offer, so "Love to. How about Global Thermonuclear War?" asks for GTW.
- **GTW-vs-chess scene.** The first GTW intent gets `WOULDN'T YOU PREFER A GOOD GAME OF CHESS?` and arms
  `Offer{chess}`. A second GTW intent ("Later. Let's play Global Thermonuclear War.") gets `FINE.` and
  `Launch{gtw}`. `YES` starts chess.
- **Intent table tests** include both film lines as must-match cases, alongside the must-not-match cases:
  - "I don't want to play chess"
  - "the golden gate bridge"
  - "my heart's not in it"
  - "poker face"
- **Streams** (A-11). `proto.NewRand(seed, id)` is the only sanctioned constructor; forbidigo bans the
  top-level `math/rand/v2` functions. The two seed halves are `Env.Seed` and a `StreamID` with disjoint
  domains:

  | Domain | StreamID |
  |---|---|
  | Brain | `1<<56 \| Turn` |
  | Game | `2<<56 \| hash32(slug)<<16 \| playIndex` |
  | AI | `3<<56 \| thinkSeq` |
  | Movie | `4<<56 \| scene` |
  | UI | `5<<56` |

  Without `--seed`, the seed comes from `crypto/rand` and is written to the debug log. Go guarantees seeded
  `math/rand/v2` sequences across releases.

### 4.7 LLM brain (M7) contract, fixed now

- **Opt-in.** Only `--llm <provider>` (`-l`) or `WOPR_LLM=<provider>` enables it. An API key in the
  environment never does. When it is enabled, WOPR announces it in one in-character line (S-4).
- **Client.** `net/http` + `encoding/json` only, no SDK (D-2: an SDK adds about 4.5 MB). The rules (S-9):
  - `CheckRedirect` returns `http.ErrUseLastResponse`, so there are no redirects;
  - HTTPS is required, except to loopback hosts;
  - keys travel only in headers;
  - the response is read through a 64 KiB `io.LimitReader`;
  - 20 s timeout, max-tokens cap, bounded retries with backoff;
  - prompts and replies are never logged.
- **Output and history.** Replies go through `SanitizeText`. History is bounded to 20 exchanges.
- **Effects.** LLM replies may carry `Lines` and a registry-validated `StartGame` only. `MarkSaid` and `SetFlag`
  belong to the scripted brain.
- **Brother's prompt.** It is sent to a provider only once he has granted a licence (L-5).

---

## 5. CLI

```text
wopr                          start the TUI (dial → LOGON)
wopr -v | --version           wopr v0.1.0 (commit abc1234, built 2026-…, go1.27.1); "unknown" if not stamped
wopr -h | --help              usage, including LOGON: Joshua, LOGOFF and the movie scene list
wopr -g | --games             numbered list in film order (+ slugs); tic-tac-toe under "ALSO AVAILABLE"
wopr -p | --play <game>       launch a game (number 1-15, slug, alias, name, or unique prefix)
wopr <game> [flags]           same as --play; flags may come before or after
wopr -m | --movie [scene]     movie mode (§7); without a scene, a scene menu
wopr -t | --theme <name>      imsai | green | amber | norad          (env WOPR_THEME)
wopr -i | --instant           no pacing                              (env WOPR_INSTANT=1)
wopr -s | --seed <n>          deterministic run (also bounds AI search by depth/nodes)
wopr -r | --reduce-motion     no blink, no panel animation, no montage acceleration (env WOPR_REDUCE_MOTION=1)
wopr -L | --licenses          print NOTICE.md and third-party notices, then exit
```

- **Interspersed positional** (C-1).
  - `cli.Parse` calls `fs.Parse` in a loop. When parsing stops at a positional, the positional is recorded and
    parsing resumes on the rest.
  - If the consumed tokens end in `--`, flag parsing stops there for good.
  - With `--movie`, the positional is a scene. Otherwise it is a game.
  - More than one positional, or a positional together with `--play`, is a usage error (exit 2).
  - Tests pin each case: `wopr gtw -i`, `wopr -i gtw`, `wopr -- gtw -i`, `wopr -t -- gtw`, `wopr gtw chess`,
    `wopr -m 2 -i`.
- **Accepted forms** (C-2). `-games`, `--games`, `-p chess`, `-p=chess`, `-i=false`. Rejected: `-pchess`,
  `-is 1`. A test pins both lists, and the help text shows the accepted ones.
- **Letters.** `-v` means version, deliberately. A future verbose flag gets another letter (not `-V`, which
  conventionally means version) (C-4). `-l` is reserved for `--llm` (M7). `--licenses` uses `-L`.
- **`games.Resolve`**: number (1–15) → slug → alias → exact normalised name → unique prefix.
  - Unlisted entries are never matched by number. Tic-tac-toe has `Listed: false`, which is explicit rather
    than relying on a zero `Number` (G-4). A registry test checks that 1..15 each appear exactly once.
  - Ambiguity (e.g. `theaterwide`) prints the candidates and exits 2.
  - A `Planned` game resolves; the TUI starts, and WOPR answers `** GAME ROUTINE NOT AVAILABLE **` (`original`)
    and stays in Shell.
- **Exit codes**: 0 ok, `LOGOFF` or SIGTERM; 1 runtime error or panic; 2 usage, not a TTY, ambiguous or unknown
  game or scene; 130 Ctrl+C or SIGINT.
  - Bubble Tea errors are mapped `ErrInterrupted` first (130), then `ErrProgramPanic` (1), because both wrap
    `ErrProgramKilled`.
  - Bubble Tea prints the panic stack to stderr itself.

---

## 6. Games

### 6.1 Definition of done (R2)

Every game must meet all of these:

- rules implemented;
- WOPR plays legally, with every search through `Think`;
- Esc abort works through the host;
- an in-character verdict, or a `Next` hand-off;
- a deterministic `testkit` transcript (seeded, depth- or node-limited);
- a game-specific quality test (below);
- fits 80×24 (`PanelRows` declared);
- a one-line "how to play" in the README;
- every script line provenance-tagged.

| # | Game | Layout | Input | M | Notes and quality test |
|---|---|---|---|---|---|
| 1 | Falken's Maze | Panel | keys | 4 | Procedural maze, fog of war. WOPR learns a turn bias and re-routes walls. Property test after every re-route: the exit is reachable *from the player's current cell*, no wall lands on the player, revealed cells stay consistent or the change is announced (G-6). |
| 2 | Black Jack | Console | line | 3 | Dealer stands on soft 17 (configurable); split/double; chips per session. Test: the dealer follows its fixed rules; payouts are exact. |
| 3 | Gin Rummy | Panel (hand) | line | 3 | Knock/gin scoring; deadwood-minimising AI. Test: scoring tables; the AI never makes an illegal play. |
| 4 | Hearts | Panel (trick) | line | 3 | 4 seats, passing, shoot-the-moon; heuristic AI on `cards/trick.go`. Test: legality over 200 seeded deals. |
| 5 | Bridge | Panel (trick) | line | 3 (last) | Minimal (decision 14). Test: auction and declarer-seat rotation; legality. |
| 6 | Checkers | Panel | line + keys | 2 | 8×8, forced captures, kings. Search depth limit 6 (deterministic). Test: forced-capture puzzles, legality. |
| 7 | Chess | Panel | line | 2 | Rules, SAN and UCI from `corentings/chess/v2`; `games/ai` alpha-beta with quiescence. Interactive: iterative deepening, 1.5 s budget, depth cap 4. Deterministic: depth 3 + quiescence (measured median 27 ms, p99 150 ms). Test: mate-in-1/2 puzzles; no illegal move in 20 seeded self-play games at depth 2. |
| 8 | Poker | Console | line | 3 | 5-card draw heads-up; betting heuristic + bluff probability. Test: hand ranking; pot accounting. |
| 9–14 | Military sims | Console/Panel | line | 4 | See §6.3. Biotoxic always ends `WINNER: NONE`. |
| 15 | Global Thermonuclear War | Console ⇄ Full | line | 2 | Self-contained film set piece (P-1); §6.2. Cannot be won. |
| — | Tic-Tac-Toe | Panel | line | 2 | Perfect minimax. Unlisted (`Listed: false`), resolvable by name. Test: exhaustive "never loses". |

### 6.2 The film path: GTW, tic-tac-toe and the ending (A-16)

The climax has **one owner**: `games/ending`. GTW and tic-tac-toe hand off to it through `Result.Next` and add
no verdict of their own.

1. **GTW**, film scenario:
   1. `SetLayout(Console)`: side choice (`1. UNITED STATES` / `2. SOVIET UNION` / `PLEASE CHOOSE ONE:`).
   2. `Clear`, then `AWAITING FIRST STRIKE COMMAND`, and the targets prompt. Targets are read until an empty
      line.
   3. `SetLayout(Full)`: the big board, trajectories, and the DEFCON ladder by turn. Force allocation covers
      ICBM, SLBM and bombers; then the enemy responds, and the kill-ratio tables follow.
   4. **Climax.** WOPR proceeds toward launch. Any input other than tic-tac-toe gets a NORAD notice in turn:
      `** GAME ROUTINE RUNNING **`, `** ROUTINE MUST COMPLETE BEFORE RESET **`, `** ACCESS DENIED **`. Each
      third rejection adds an `original` hint naming tic-tac-toe.
   5. `TIC-TAC-TOE` (or any intent naming it) gives `Done{NoWinner, Next: Launch{tictactoe, "climax"}}`.
2. **Tic-tac-toe in `climax` mode**: `ONE OR TWO PLAYERS? PLEASE LIST NUMBER OF PLAYERS:`.
   - `1` plays a game, which ends in `STALEMATE. WANT TO PLAY AGAIN?`.
   - `0` (or `ZERO`) gives `Done{NoWinner, Next: Launch{ending}}`.
   - In normal mode, `NUMBER OF PLAYERS: 0` does the same.
3. **The ending** runs with Animate, Prompt and `Wait`:
   1. self-play at increasing speed, capped by the flash rule;
   2. the scenario montage, each ending `WINNER: NONE`;
   3. `Clear`, `GREETINGS PROFESSOR FALKEN.`, `Prompt`;
   4. any input, e.g. `Hello.`;
   5. `A STRANGE GAME.` / `THE ONLY WINNING MOVE IS` / `NOT TO PLAY.`;
   6. `HOW ABOUT A NICE GAME OF CHESS?`;
   7. `Done{NoWinner, NoVerdict: true}`.

   The persona's phase goes `Ending → Shell` with `Offer{chess}` armed.

### 6.3 Sim engine (M4)

The first M4 task is a written engine spec:

- a map model (named regions on a strip or grid);
- a unit table (type, strength, mobility, range);
- per-scenario action sets;
- a combat-results table drawn with the game's stream;
- an event deck;
- a turn limit and a victory rule;
- a kill-ratio generator in the film's table format.

Guerrilla Engagement, Desert Warfare, Theaterwide Tactical and Biotoxic are **scenarios** (data plus a few
hooks). Fighter Combat (an energy/aspect dogfight) and Air-to-Ground (sortie packages against SAM threat) do
not fit a region map. They are bespoke games that reuse only the combat-results and kill-ratio parts (G-2).
GTW moves onto `sim` only if that removes code.

---

## 7. Movie mode (M6)

`wopr --movie [scene]` replays the film's **WOPR terminal scenes** as a self-running show, through the same
console, typewriter, canvas and game code as interactive play.

**Scenes** (provisional list; the scripts are fixed in the M5 viewing pass):

| # | Slug | Content |
|---|---|---|
| 1 | `first-contact` | Dial; LOGON attempts and `IDENTIFICATION NOT RECOGNIZED BY SYSTEM` / `--CONNECTION TERMINATED--`; `HELP LOGON`, `HELP GAMES`, `LIST GAMES` |
| 2 | `joshua` | `LOGON: Joshua`; header and status burst; greeting; GTW-vs-chess; `FINE.` |
| 3 | `first-strike` | Side choice; targets (Las Vegas, Seattle); big board, trajectories, DEFCON |
| 4 | `call-back` | WOPR's later conversation with David: `IS THIS A GAME OR IS IT REAL?` / `WHAT'S THE DIFFERENCE?`, the kill-ratio offer, Falken's address |
| 5 | `climax` | Tic-tac-toe with 0 players, self-play, montage, `A STRANGE GAME…` |

**Design.**

- **Data.** A scene is data in `internal/movie/scenes/` with a provenance tag on every step. A step is one of:
  - `Type{Text}`: the user's line, typed at about 8 chars/s with seeded jitter, in mixed case as on screen;
  - `Say{Lines, Pace}`;
  - `Clear`;
  - `Wait{D}`;
  - `Board{…}`: a GTW film-scenario frame drawn by `games/gtw`'s exported film renderer;
  - `Run{…}`: the real `games/ending` program fed scripted input.
- **Director.** `movie.Director` is a `proto.Program`, so all host rules apply. It runs `Deterministic`, with its
  own stream, so every replay is identical.
- **Controls**:

  | Key | Action |
  |---|---|
  | Space | Pause / resume |
  | → or `n` | Next scene |
  | ← or `p` | Previous scene |
  | Esc | Scene menu |
  | `LOGOFF` or `q` (in the menu) | Exit 0 |
  | Ctrl+C | Exit 130 |

  `--instant` plays with no pacing, which tests use. `--theme` and `--reduce-motion` apply.
- **Isolation.** No LOGON, no persona, no Brain, no network.
- **Consistency test.** Scenes whose WOPR lines the interactive persona also produces are marked `Interactive`
  (`joshua`, parts of `first-contact`). A test feeds their `Type` steps through the real persona via `testkit`
  and asserts that the persona's lines equal the scene's `Say` lines. Movie mode and interactive play cannot
  drift apart, and the scenes double as end-to-end persona tests.
- **Legal scope** (risk R-1 in §16). Movie mode concentrates film text. Scenes contain only text that appears on
  the WOPR terminal on screen: no spoken-only dialogue, no audio, no stills. Every line is tagged, and the
  quotations are listed in `NOTICE.md` as excluded from the MIT grant. The owner accepts the remaining risk by
  requesting the feature.

---

## 8. Binary size and portability

- **Build** (GoReleaser, the same config for snapshots and releases; Appendix A.3): `CGO_ENABLED=0`,
  `-trimpath`, `-s -w`, and `-X` for Version, Commit and Date. `mod_timestamp`, `builds_info` and per-file
  `info.mtime` are pinned to the commit time, so archives are byte-reproducible. A test with GoReleaser v2.18.2
  touched the extra files between two runs and got identical checksums (B-12).
- **Measured** (go1.27.1, one Lip Gloss-styled view, stripped):

  | Target | Bubble Tea + Lip Gloss | + chess (full use) | + stdlib HTTP client (M7) |
  |---|---|---|---|
  | linux/amd64 | 3.88 MiB | ≈ 4.0 MiB | 7.99 MiB |
  | linux/arm64 | 3.69 MiB | ≈ 3.8 MiB | 7.44 MiB |
  | darwin/amd64 | 3.89 MiB | ≈ 4.0 MiB | 8.15 MiB |
  | darwin/arm64 | 3.71 MiB | ≈ 3.8 MiB | 7.60 MiB |
  | windows/amd64 | 3.84 MiB | ≈ 4.2 MiB | 8.04 MiB |
  | windows/arm64 | 3.59 MiB | ≈ 3.9 MiB | 7.33 MiB |

  The worst case, with everything including HTTP (windows/amd64), is 8.75 MB, under the 10 MB target. Game
  code, art and scenes add hundreds of KB. A provider SDK would add about 4.5 MB more, so none is used.
- **Size gate**: `go run ./internal/tools/sizegate dist/artifacts.json` (B-7, B-9).
  - It reads GoReleaser's artifact list and gates only `Binary` entries.
  - It fails on an empty list.
  - It writes a table to the step summary, warns above 10 MB and fails above 15 MB.
  - GoReleaser's `report_sizes` also logs the sizes.
  - The README states only the budget (B-8).
- **Never UPX** (Windows AV false positives).
- **Windows.**
  - Ctrl+C arrives as a key in raw mode on every OS, so `Update` maps it to `tea.Interrupt`. A blocked
    `Update` would delay it, which is why all slow work is `Think`.
  - Keyboard enhancements stay off.
  - conhost resize events may report the buffer height (9001). Clamp to the window rect and check in the M2 QA
    pass.
  - Ship unsigned. Document SmartScreen, and verify before running (§13).
- **Panics**: no custom `recover` around `p.Run()`. Bubble Tea restores the terminal, recovers `Cmd` goroutine
  panics too, and prints the stack to stderr. The model returned on panic is nil and is not used.
- **Debug log** (S-5): `WOPR_DEBUG=1` writes `os.UserCacheDir()/wopr/debug.log`.
  - Directory mode 0700, file mode 0600.
  - If there is no cache dir, logging is disabled and a note goes to stderr.
  - The path is printed on exit.
  - Typed input is never logged.
  - `TEA_DEBUG` is never set by the app; Bubble Tea would write a panic log into the CWD.

---

## 9. Testing

- **Unit tests** cover:
  - game rules (via `testkit`);
  - `Resolve`, `cli.Parse`, intent and offers (the film lines and the false-positive table);
  - the LOGON table;
  - the scripted brain, typewriter dt math and the line editor;
  - `Sanitize*`, the canvas, and the theme fallbacks under the ASCII and ANSI profiles.
- **Architecture**: `archtest` covers the DAG, test imports, catalog completeness, toolchain equality and
  provenance tags.
- **Golden** (Q-3, A-14).
  - `internal/golden.Assert(t, name, got)` stores files under each package's `testdata/`. They are LF-only
    and CRLF is normalised before comparing.
  - Regenerate every golden with `WOPR_UPDATE_GOLDEN=1 go test ./...`, which works on every OS and package.
  - `ui/testutil/drive.go` runs the model synchronously with a fake `Scheduler`. It executes returned
    commands, unwraps `tea.BatchMsg` (and a single unwrapped cmd), and snapshots `View().Content` after
    `ansi.Strip`, plus StyleMaps.
  - `tea.Sequence` is banned (forbidigo).
  - **Flows**: `logon_joshua`, `logon_fail_x3_hint`, `logon_game_name_is_not_a_login`,
    `logon_list_then_number`, `help_games`, `list_games`, `intent_film_lines`, `intent_false_positives`,
    `offer_accept`, `esc_confirm`, `too_small_pause_resume`, `logoff_exit0`, then each set piece and each movie
    scene.
- **Fuzz** (Q-2): normalise and clause split, `Resolve`, intent, the line editor, `Sanitize*`, and the M7 reply
  parser.
  - Seed corpora run in plain `go test`.
  - CI runs `go test -run '^$' -fuzz '^FuzzX$' -fuzztime=10s ./pkg` once per target on Linux, and uploads any
    crasher as an artifact.
- **End-to-end** (Q-1, U-8).
  - `internal/e2e` (build tag `e2e`) is a Go test that drives the **real binary** in a pseudo-terminal sized
    **80×24**. It uses `charmbracelet/x/xpty`: a Unix pty, or ConPTY on Windows.
  - **Cases**:
    - `Joshua` → wait for `GREETINGS PROFESSOR FALKEN.` → `LOGOFF` → exit 0;
    - a second run → Ctrl+C → exit 130 with the terminal restored;
    - start at 60×20 → resize to 80×24 → the session resumes where it paused;
    - `--movie joshua --instant` → `FINE.` → exit on `q`.
  - On Unix it asserts termios is restored. Under ConPTY the assertion is weaker (exit code and final screen),
    because conhost re-renders the output.
  - The `build` job cross-compiles the test per target (`GOOS/GOARCH go test -c -tags e2e`) and uploads it
    next to each binary, so smoke runners need no Go toolchain.
- **Race.**
  - CI runs `go test -race ./...` on `ubuntu-24.04`, `macos-26` and `windows-2025`; the Windows image ships
    MinGW, and M0 verifies that.
  - `testkit` exercises `Handle` and `View` concurrently with `Think` so the detector has something to see.
  - Contributors: `go test ./...` everywhere. Add `-race` on Linux and macOS, and on Windows amd64 only with a
    C toolchain. windows/arm64 has no race detector (B-14).

---

## 10. Tooling: prek and static analysis

- **One hook set in `prek.toml`** (Appendix A.1), run identically locally (`prek install` installs both the
  pre-commit and pre-push shims) and in CI.
  - **The gate is `prek run --all-files`.** `prek validate-config` checks syntax only; it resolves neither revs
    nor hook ids (T-1, T-9).
  - **Ordering**: each file-mutating fixer gets its own priority, because equal priorities run concurrently and
    lose updates (T-8). Formatters come next, then read-only linters.
  - golangci-lint's `golangci-lint-full` entry is overridden to drop `--fix`. zizmor runs without `--fix`
    (T-5).
- **Pinning** (T-3, T-10).
  - Every hook rev is a full commit SHA with a `# frozen: vX.Y.Z` comment.
  - `[update] freeze = true, cooldown_days = 14` makes a plain `prek update` (formerly `auto-update`) keep that
    form.
- **Go tools** (T-4, T-11). govulncheck and gitleaks are `tool` lines in `tools/go.mod`, and GoReleaser is in
  `tools/release/go.mod`. They are pinned and checksummed by `go.sum`, and run as
  `go tool -modfile=tools/go.mod <tool>`. Only `tools/release` must not raise the main module's `go` line.
- **golangci-lint** (Appendix A.2, verified with v2.14.0):
  - `default: standard` plus revive, gocritic, misspell, forbidigo and depguard; formatters gofmt, goimports
    and gofumpt.
  - **forbidigo** (T-2, A-11), with `analyze-types: true`:
    - `^tea\.(Tick|Every|Sequence)$` with `pkg: ^charm\.land/bubbletea/v2$`, exempt in `internal/ui/clock.go`;
    - top-level `math/rand/v2` functions, exempt in `internal/proto/rand.go`;
    - `time.Sleep`, `After`, `AfterFunc`, `Tick`, `NewTicker` and `NewTimer`, exempt in `clock.go` and tests.
  - **misspell** skips `lines.go`, `assets/` and `movie/scenes/`, because the film's spelling (`RECOGNISED`) is
    verbatim.
  - **Self-test.** `config verify` accepts rules that never fire, so `internal/archtest/lintfixture` holds a
    deliberate violation of each rule. A CI step asserts that each rule reports on it.
- **codespell** skips script and art data with a prek `exclude` regex, `(^|/)testdata/` included.
- **ShellCheck** stays in the hook set for any future script. CI and tooling logic is Go (`internal/tools/*`),
  so it runs on Windows (B-14).
- **Commands** have one home: the Commands table in `AGENTS.md`. There is no Makefile, because Git for Windows
  ships no `make`. The README links to that table.

---

## 11. CI/CD (GitHub Actions)

### 11.1 Common rules

- Every action is pinned to a full commit SHA. Tools inside actions are pinned too: `prek-version: 0.5.5`
  (T-11), and GoReleaser and gitleaks via `go tool`.
- `permissions: {}` at the top of each workflow; each job grants only what it needs.
- `persist-credentials: false` on every checkout. Only `GITHUB_TOKEN` is used. No `pull_request_target`.
- **`actions/setup-go`** uses `go-version-file: go.mod`, and **nothing sets `GOTOOLCHAIN` before it** (D-6).
  setup-go exports `GOTOOLCHAIN=local` itself after installing 1.27.1. `archtest` asserts the toolchain
  version.
- **Caching.** `cache: false` in `release.yml` and in any job that runs GoReleaser (B-4, B-9).
- **`.gitattributes`**: `* text=auto eol=lf`, plus `*.png binary`. Windows runners check out with
  `core.autocrlf=true`, which would otherwise break goldens (B-3).
- **Concurrency**, as a block mapping. Pushes to `main` are never cancelled (B-6, B-10):

  ```yaml
  concurrency:
    group: ci-${{ github.ref }}
    cancel-in-progress: ${{ github.event_name == 'pull_request' }}
  ```

### 11.2 `ci.yml` (on `pull_request` and `push` to `main`)

| Job | Runner(s) | Does |
|---|---|---|
| `lint` | `ubuntu-24.04` | prek via `j178/prek-action` with `prek-version: 0.5.5`. Separately, a zizmor online-audits step with `GH_TOKEN` scoped to that step only (T-5). The lint-fixture self-test. |
| `secrets` | `ubuntu-24.04` | `fetch-depth: 0`; `go tool -modfile=tools/go.mod gitleaks git --redact --exit-code 1 .` over **full history**. Fails on any `ERR` line (S-1). |
| `test` | `ubuntu-24.04`, `macos-26`, `windows-2025` | `go test -race ./...`; fuzz smoke and `govulncheck` on Linux; the third-party-notices check (§12). |
| `build` | `ubuntu-24.04` | `go tool -modfile=tools/release/go.mod goreleaser release --snapshot --clean`; size gate; cross-compiled e2e test binaries. Uploads one artifact per target: `wopr_<os>_<arch>` on `main`, `wopr_<os>_<arch>-pr<N>` with 3-day retention on PRs (B-13). |
| `smoke` | `ubuntu-24.04` (+ an `ubuntu:22.04` container step), `ubuntu-24.04-arm`, `macos-26`, `macos-26-intel`, `windows-2025`, `windows-11-arm` | Download that target's archive and e2e binary. `go run`-free smoke: `--version`, `--games`, `--games` into a closed pipe, then the e2e test. Every target has a native runner, so none is skipped. |
| `ci-ok` | `ubuntu-24.04` | `needs: [lint, secrets, test, build, smoke]`, `if: always()`. Fails unless every needed job succeeded. **This is the only required check** (S-3), so matrix names never appear in settings. |

`smoke` is a reusable workflow (`workflow_call`), so `release.yml` runs the same job.

### 11.3 `release.yml` (on `v*` tags): gated, reproducible, attested (B-10, S-7)

1. **`verify`** (`contents: read`, `checks: read`). The tag is semver; the tagged commit is an ancestor of
   `origin/main`; and that commit's `ci-ok` check run succeeded.
2. **`build`** (`contents: read`, no OIDC). `goreleaser release --clean --skip=publish` with release settings,
   then the size gate, then a check that every archive contains `LICENSE`, `README.md`, `NOTICE.md` and
   `THIRD_PARTY_NOTICES.txt` (B-11). `dist/` is uploaded.
3. **`smoke`**: the reusable workflow, run on the release archives themselves.
4. **`repro`**: rebuild from a fresh checkout on `ubuntu-24.04-arm` (cross-compiling) and diff `checksums.txt`.
   Any difference fails the release.
5. **`publish`** (`contents: write`, `id-token: write`, `attestations: write`).
   1. `actions/attest@v4` with `subject-checksums: dist/checksums.txt`.
   2. `gh release create vX.Y.Z --draft --verify-tag --generate-notes`, then upload the archives and
      `checksums.txt`.
   3. `gh release edit vX.Y.Z --draft=false`.

   Immutable releases (§12) lock the release at publish time. A failed publish leaves only a draft.

**Never re-tag.** A bad release is fixed by the next patch version, plus a `retract` directive for
`go install` users. That rule goes in `SECURITY.md` and `AGENTS.md`. `workflow_dispatch` runs steps 2–4 as a
dry run.

### 11.4 Runner labels

The labels as of 2026-10-06 are `ubuntu-24.04`, `ubuntu-24.04-arm`, `macos-26`, `macos-26-intel`,
`windows-2025` and `windows-11-arm`, all free on public repositories. `ubuntu-22.04` is deprecated and
`macos-14` is unsupported from 2026-11-02, so neither is used. The labels are checked at every milestone
boundary (AGENTS.md checklist) (B-5).

### 11.5 Dependency updates without Dependabot (R13)

- **`scheduled.yml`**, weekly, `contents: read` plus `issues: write` on the report job only. It runs:
  - `govulncheck` on `main`;
  - `go list -m -u all` (outdated modules);
  - `prek update --check` (stale hooks);
  - the third-party-notices check.

  On any finding it opens or updates a single tracking issue. It never opens PRs, so "Allow GitHub Actions to
  create pull requests" stays off.
- **Manual cadence** (AGENTS.md checklist, at every milestone and at least monthly):
  1. `go get -u ./... && go mod tidy`;
  2. update the tool modules with `go get -tool`;
  3. `prek update`;
  4. bump action SHAs from their release tags;
  5. regenerate the third-party notices.

  Each step goes through a normal PR.
- **Gaps covered** (S-10). Public repositories disable schedules after 60 days without activity, so the
  milestone checklist includes "re-enable `scheduled.yml` if disabled". Dependency PRs also run `govulncheck`
  in `ci.yml`.
- **Response rule** (SECURITY.md). A reachable govulncheck finding, or a Go security release that affects the
  stdlib, means a `toolchain` bump and a patch release within 14 days.

---

## 12. Security and public-repository hygiene

- **Secrets.** None in the tree, ever.
  - gitleaks runs as a pre-commit hook and in the `secrets` job.
  - M0 proves the job: a throwaway PR containing a gitleaks test fixture must turn it red before `ci-ok` is
    made required.
  - `.gitignore` adds `*.pem`, `*.key`, `dist/`, `coverage.*` and editor/OS files.
- **Repository settings** (S-3, S-7, S-8). The owner applies them in M0; AGENTS.md lists them.
  - **`main` ruleset**: pull request required; required check `ci-ok`; no force-push; no deletion. Merges are
    **squash only**, so v1's history (which contains the CC BY-SA map fragments, L-4) never reaches `main`.
  - **`v*` tag ruleset**: creation, update and deletion restricted, with only the owner able to bypass.
    **Immutable releases** are enabled before v0.1.0.
  - **Secret scanning** is on by default for public repos. **Push protection** must be enabled at
    repository level, where it is off by default. Also: private vulnerability reporting, plus `SECURITY.md`.
  - **Actions**:
    - allow only listed actions (`actions/*`, `j178/prek-action`);
    - require actions pinned to a full-length commit SHA;
    - default `GITHUB_TOKEN` read-only;
    - workflows may not create or approve PRs;
    - approval required for **all** external contributors' workflow runs.
  - **Dependabot**: off (owner decision). govulncheck covers the Go graph.
- **The app.** No network code before M7, no telemetry, no transcripts. External text is sanitised. The debug
  log holds no typed input.
- **Releases.** Users verify before running:
  - Command: `gh attestation verify <archive> --repo GhostofGoes/WOPR`, plus
    `--signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml`,
    `--source-ref refs/tags/vX.Y.Z` and `--deny-self-hosted-runners`.
  - Only after that: `xattr -d com.apple.quarantine`, or the SmartScreen prompt.
  - Signing and notarisation are post-1.0 options. There is no Homebrew tap: casks need signing, third-party
    taps need `brew trust`, and a tap needs a token beyond `GITHUB_TOKEN`.
- **Licensing** (L-1, L-4, L-5, B-11).
  - Code is MIT.
  - `NOTICE.md` lists:
    - the film quotations, excluded from the MIT grant, with a non-affiliation disclaimer;
    - the BSD-2 notice for the abs0 text;
    - the brother's licence once granted.
  - `THIRD_PARTY_NOTICES.txt` is generated by `go run ./internal/tools/notices`, from `go list -deps` plus each
    module's LICENSE and Go's own. It is committed, embedded in the binary for `wopr --licenses`, checked
    up-to-date in CI, and shipped in every archive.
  - "WarGames" is used only nominatively, in the README. There are no film stills.
  - "WOPR" is a registered US mark held by an unrelated company (L-6). It is recorded as a low risk in §16.

---

## 13. Documentation

- **README.md** (basic in M0, completed in M5):
  - what this is;
  - install per OS, **verify first** (§12), then Gatekeeper and SmartScreen notes, then `go install …@latest`;
  - quick start (`Joshua`, `LOGOFF`);
  - inside the shell: commands, keys, and "any key skips; what you type is kept";
  - flags; themes; the games ("coming in vX.Y" for `Planned` ones); movie mode;
  - troubleshooting: too small, `NO_COLOR`, reduced motion, Windows/mintty, the mouse-wheel note, `-i`, `-s`;
  - development builds: artifacts from **`main` push runs only** (B-13);
  - build from source (a link to the AGENTS.md Commands table); credits and licence; a link to this plan.
- **AGENTS.md** (M0; updated each milestone). It is the single source for:
  - what the project is and the Commands table, per OS;
  - the import DAG and `archtest`; the single clock; the protocol rules (`Think` captures values, Esc is the
    host's, input mode is dynamic);
  - how to add a game: catalog entry, package, testkit test, definition of done;
  - goldens and `WOPR_UPDATE_GOLDEN`; provenance tags;
  - the size budget and platforms; repository settings;
  - the milestone checklist: dependency updates, runner labels, re-enabling schedules, `prek update`.

  `CLAUDE.md` contains exactly `@AGENTS.md`.
- **docs/screens.md**: the mockups from Appendix C plus Hearts (M3). Kept current.
- **This plan** is updated at milestone boundaries. Superseded text is deleted.

---

## 14. Verification (per release)

1. `ci-ok` is green on the tagged commit. The release pipeline's `verify`, `build`, `smoke` and `repro` jobs pass.
   The six archives each contain the four notice files and are within the size budget.
2. `wopr -v`, `-h`, `-g`, `-L` and their long forms print plain text and exit 0 without a terminal.
   `wopr --games | head -1` exits 0 on every OS.
3. `wopr`:
   1. LOGON → `FALKEN'S MAZE` → `IDENTIFICATION NOT RECOGNIZED` → re-dial;
   2. two more wrong names → the hint;
   3. `Joshua` → the greeting;
   4. `Love to. How about Global Thermonuclear War?` → the chess offer;
   5. `Later. Let's play Global Thermonuclear War.` → `FINE.` and GTW starts;
   6. Esc, Esc → back in the Shell with a remark;
   7. `list games` → `7` → the chess board. WOPR shows `PROCESSING` and Ctrl+C stays responsive;
   8. `LOGOFF` → exit 0.
4. The film path under `-s 1 -i` with scripted input (the `drive.go` golden `film_path`): GTW → climax → NORAD
   notices → `TIC-TAC-TOE` → 0 players → montage → `HELLO` → `A STRANGE GAME…` → chess offer armed.
   `wopr -p 15`, `wopr -p "global thermonuclear war"` and `wopr gtw` resolve to the same game.
   `wopr -p theaterwide` and `wopr gtw chess` exit 2.
5. `go test -race ./...` is green on the three test OSes. Goldens are stable. `archtest` and the lint
   self-test are green.
6. The e2e test passes on all six smoke runners. A manual pass in Windows Terminal and conhost checks colours,
   cursor, resize, Ctrl+C restore and SmartScreen.
7. On a tag, `gh attestation verify` with the flags in §12 succeeds for each archive, and the release is
   immutable.
8. From M6: `wopr --movie` plays every scene, and `wopr -m 2 -i` exits 0 at the end of the scene list. The
   movie consistency test is green.

---

## 15. Milestones

| M | Deliverable | Exit criteria | Release |
|---|---|---|---|
| 0 | **Scaffold.** go.mod, `.gitattributes`/`.gitignore`, LICENSE/NOTICE/SECURITY, `version`, `proto` (types), `games` (types + catalog with 16 `Planned` entries), `cli`, `theme`, `ui` hello-world, `archtest` + lint fixture, `golden`, `tools/*`, `prek.toml`, `.golangci.yml`, `.goreleaser.yaml`, `ci.yml`/`release.yml`/`scheduled.yml`, README (basic), **AGENTS.md** + `CLAUDE.md`, repository settings. | `ci-ok` green with all six smoke runners; the gitleaks canary PR turned red; the lint fixture fires every rule; a `release.yml` dry run passes `build`/`smoke`/`repro`; settings applied; Appendix A deleted. | — |
| 1 | **Console, persona and protocol.** `proto/host` runner, clock, typewriter, line editor, `Sanitize*`, scrollback, canvas renderer, front panel, `prompt` (menus, yes/no, clauses; moved from M3, P-4), adapter, persona (session, LOGON table, scenes, commands, intent, offers, scripted brain, lines with provenance), themes with ANSI/ASCII fallbacks, too-small handling, `gamestest` stub exercising every Output, goldens, fuzz, e2e. 80×24 mockups for chess and GTW phases. | Every M1 golden flow and the e2e cases are green on all runners. | — |
| 2 | **Film set pieces.** `games/ai`, `board/`, tic-tac-toe, checkers, chess, **GTW** (§6.2), **`games/ending`**, original GTW map art, scenario names (`third-party:abs0`, credited). QA on all OSes. | The `film_path` golden; quality tests; manual QA list. | **v0.1.0** |
| 3 | **Card games.** `cards/` + `trick.go`; Black Jack, Poker, Gin Rummy, Hearts, **Bridge (minimal, last)**; Hearts mockup. | Definition of done per game. | v0.2.0 |
| 4 | **Sims and maze.** Sim engine spec → engine → four scenarios + two bespoke sims; Falken's Maze. | Definition of done per game. | v0.3.0 |
| 5 | **Polish.** Film viewing pass (every `reconstructed` line becomes `film` or is corrected; the montage names verified; the two conflicts in §2.3 settled; the movie scene scripts fixed); README completed with screenshots; accessibility pass; dependency and runner checklist. | No `reconstructed` tags remain; checklist done. | **v1.0.0** |
| 6 | **Movie mode** (§7): `-m/--movie`, director, scenes, scene menu, consistency test, e2e case. | All scenes play; consistency test green; size re-checked. | v1.1.0 |
| 7 (opt) | **LLM brain** (§4.7): opt-in, `net/http`, hardened client, effects allowlist, scripted fallback. | Fuzzed reply parser; size gate; offline behaviour unchanged. | v1.2.0 |

Effort is not estimated per game (P-3). Each milestone's PR description records time spent, which informs the
next one.

**Third-party permission gate** (L-3, L-5). Lines adapted from the brother's prompt carry the `prompt` tag and
are excluded from builds until he gives written permission and a licence. The gate blocks those lines only, not
M1.

---

## 16. Risks

| # | Risk | Likelihood / impact | Mitigation |
|---|---|---|---|
| R-1 | Film text is MGM's expression. Movie mode (M6) multiplies the amount shipped. | Medium / medium | Short on-screen terminal text only; provenance per line; `NOTICE.md` exclusion and disclaimer; no stills or audio; nominative use of "WarGames"; the owner accepts the residual risk. |
| R-2 | "WOPR" is a registered US mark (Frontier Technology, class 42). | Low / medium | Different class (SaaS vs a free game); no logo imitation; revisit if contacted. Not legal advice. |
| R-3 | Bubble Tea v2 patch churn (v2.0.10 in under a year). | Medium / low | Pin; read release notes on each update; goldens catch rendering changes. |
| R-4 | Hosted runner labels retire. | High / low | Pinned labels; checklist at each milestone; `ubuntu:22.04` container for the oldest Linux claim. |
| R-5 | Unsigned binaries trigger Gatekeeper and SmartScreen friction. | High / low | Attestations plus verify-first docs; signing post-1.0. |
| R-6 | No Dependabot means updates lag. | Medium / medium | Weekly scheduled report issue; govulncheck in CI; monthly manual cadence; 14-day response rule. |
| R-7 | 16 games is a large scope. | High / medium | A release per milestone; `Planned` games stay listed and decline in character. |
| R-8 | The chess library fork goes stale. | Low / low | Small API surface used (rules, SAN/UCI); search is our own. |

---

## Appendix A — M0 seeds (delete when M0 lands)

### A.1 `prek.toml`

Validated with prek 0.5.5 (`prek validate-config`, `prek list`, `prek run --all-files` on a sample project).
The codespell exclude was then adapted to v2's paths; M0 re-runs the gate.

```toml
#:schema https://www.schemastore.org/prek.json
minimum_prek_version = "0.5.5"
default_language_version.golang = "1.27"                 # golang hooks are built with Go 1.27.x
default_install_hook_types = ["pre-commit", "pre-push"]  # plain `prek install` installs only pre-commit
default_stages = ["pre-commit", "manual"]

[update]                 # `prek update`; `auto-update` no longer exists
cooldown_days = 14
freeze = true            # rev = "<sha>"  # frozen: vX.Y.Z

# Lower runs first; equal priority runs concurrently. Mutating fixers get distinct priorities.
[priorities]
bom = 0
eol = 1
ws = 2
eof = 3
format = 10
lint = 20

[[repos]]
repo = "builtin"
hooks = [
  { id = "fix-byte-order-marker", priority = "bom" },
  { id = "mixed-line-ending", args = ["--fix=lf"], priority = "eol" },
  { id = "trailing-whitespace", args = ["--markdown-linebreak-ext=md"], priority = "ws" },
  { id = "end-of-file-fixer", priority = "eof" },
  { id = "check-yaml", priority = "lint" },
  { id = "check-toml", priority = "lint" },
  { id = "check-json", priority = "lint" },
  { id = "check-merge-conflict", priority = "lint" },
  { id = "check-case-conflict", priority = "lint" },
  { id = "check-illegal-windows-names", priority = "lint" },
  { id = "check-executables-have-shebangs", priority = "lint" },
  { id = "check-shebang-scripts-are-executable", priority = "lint" },
  { id = "check-added-large-files", args = ["--maxkb=512"], priority = "lint" },
  { id = "detect-private-key", priority = "lint" },
]

[[repos]]
repo = "https://github.com/golangci/golangci-lint"
rev = "114493f9b3e7257d29e4130f2b4a4aadefbb6845"  # frozen: v2.14.0
hooks = [
  { id = "golangci-lint-fmt", priority = "format" },
  { id = "golangci-lint-config-verify", priority = "lint" },
  { id = "golangci-lint-full", entry = "golangci-lint run", priority = "lint" },  # manifest has --fix
]

[[repos]]
repo = "https://github.com/gitleaks/gitleaks"
rev = "83d9cd684c87d95d656c1458ef04895a7f1cbd8e"  # frozen: v8.30.1
hooks = [{ id = "gitleaks", stages = ["pre-commit"], priority = "lint" }]  # staged-only; CI scans history

[[repos]]
repo = "https://github.com/zizmorcore/zizmor-pre-commit"
rev = "fa412071e4f5d44d44f9e365f4676f9df92456a2"  # frozen: v1.30.1
hooks = [{ id = "zizmor", args = ["--no-progress"], priority = "lint" }]

[[repos]]
repo = "https://github.com/codespell-project/codespell"
rev = "57b21406f092110c18776e39b0bda50d37c945c8"  # frozen: v2.4.3
hooks = [{ id = "codespell", args = ["--ignore-words-list", "theaterwide,recognised,falken,wopr,norad,imsai"], exclude = '^(go\.sum|internal/wopr/lines\.go|internal/assets/|internal/movie/scenes/)|(^|/)testdata/|\.golden$', priority = "lint" }]

[[repos]]
repo = "https://github.com/shellcheck-py/shellcheck-py"
rev = "745eface02aef23e168a8afb6b5737818efbea95"  # frozen: v0.11.0.1
hooks = [{ id = "shellcheck", args = ["--severity=style", "--enable=all"], priority = "lint" }]

[[repos]]
repo = "https://github.com/rvben/rumdl-pre-commit"
rev = "83cecb6b523a053d7d3dfdc4798d705b505a88f9"  # frozen: v0.2.78 (rumdl-check needs >= v0.2.66)
hooks = [
  { id = "rumdl-fmt", priority = "format" },
  { id = "rumdl-check", args = ["--disable", "MD013,MD033,MD041"], priority = "lint" },
]

[[repos]]
repo = "local"
hooks = [
  { id = "go-mod-tidy", name = "go mod tidy -diff", language = "system", entry = "go mod tidy -diff", files = '(^|/)go\.(mod|sum)$|\.go$', pass_filenames = false, priority = "lint" },
  { id = "go-test", name = "go test (short)", language = "system", entry = "go test -short ./...", types = ["go"], pass_filenames = false, stages = ["pre-push"], priority = "lint" },
  { id = "govulncheck", name = "govulncheck", language = "system", entry = "go tool -modfile=tools/go.mod govulncheck ./...", pass_filenames = false, stages = ["pre-push"], priority = "lint" },
]
```

### A.2 `.golangci.yml`

The forbidigo, depguard and exclusion rules were verified with golangci-lint 2.14.0 (go1.27.1) against real
Bubble Tea v2.0.10:

- each rule fires on a deliberate violation, including import-alias and dot-import uses;
- methods on a seeded `*rand.Rand` pass;
- the file exemptions work.

```yaml
version: "2"
linters:
  default: standard            # errcheck, govet, ineffassign, staticcheck, unused
  enable: [revive, gocritic, misspell, forbidigo, depguard]
  settings:
    forbidigo:
      analyze-types: true      # match text is "<declared pkg name>.<func>", even via alias or dot import
      forbid:
        - pattern: '^tea\.(Tick|Every|Sequence)$'
          pkg: '^charm\.land/bubbletea/v2$'
          msg: use the single clock in internal/ui/clock.go
        - pattern: '^rand\.(Int|IntN|Int32|Int32N|Int64|Int64N|Uint|UintN|Uint32|Uint32N|Uint64|Uint64N|Float32|Float64|ExpFloat64|NormFloat64|Perm|Shuffle|N)$'
          pkg: '^math/rand/v2$'
          msg: use proto.NewRand(seed, streamID); top-level math/rand/v2 functions cannot be seeded
        - pattern: '^time\.(Sleep|After|AfterFunc|Tick|NewTicker|NewTimer)$'
          pkg: '^time$'
          msg: timing goes through the single clock or a context deadline
    depguard:
      rules:
        bubbletea-only-in-ui:
          list-mode: lax
          files: ["$all", "!${base-path}/internal/ui/**"]
          deny:
            - pkg: charm.land/bubbletea/v2
              desc: only internal/ui may import Bubble Tea
  exclusions:
    warn-unused: true
    rules:
      - path: ^internal/ui/clock\.go$
        linters: [forbidigo]
      - path: ^internal/proto/rand\.go$
        linters: [forbidigo]
      - path: _test\.go$
        linters: [forbidigo]
        text: 'time\.'
      - path: ^internal/(wopr/lines\.go|assets/|movie/scenes/)
        linters: [misspell]
formatters:
  enable: [gofmt, goimports, gofumpt]
issues:
  max-same-issues: 0           # the default of 3 hid identical depguard hits
```

### A.3 `.goreleaser.yaml`

Verified with GoReleaser v2.18.2:

- `goreleaser check` passes;
- `release --snapshot --skip=publish` builds six binaries and six archives, each containing the four notice
  files;
- two runs with the extra files touched in between produce identical `checksums.txt`.

```yaml
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
version: 2
project_name: wopr

builds:
  - id: wopr
    main: ./cmd/wopr
    binary: wopr
    env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]           # the default would add 386
    flags: [-trimpath]
    ldflags:
      - -s -w
      - -X github.com/GhostofGoes/WOPR/internal/version.Version={{ .Version }}
      - -X github.com/GhostofGoes/WOPR/internal/version.Commit={{ .FullCommit }}
      - -X github.com/GhostofGoes/WOPR/internal/version.Date={{ .CommitDate }}
    mod_timestamp: "{{ .CommitTimestamp }}"

archives:
  - id: wopr
    ids: [wopr]
    formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    builds_info:                     # pin the binary's archive entry too
      mtime: "{{ .CommitDate }}"
      owner: root
      group: root
      mode: 0755
    files:
      - src: LICENSE
        info: { mtime: "{{ .CommitDate }}", owner: root, group: root, mode: 0644 }
      - src: README.md
        info: { mtime: "{{ .CommitDate }}", owner: root, group: root, mode: 0644 }
      - src: NOTICE.md
        info: { mtime: "{{ .CommitDate }}", owner: root, group: root, mode: 0644 }
      - src: THIRD_PARTY_NOTICES.txt
        info: { mtime: "{{ .CommitDate }}", owner: root, group: root, mode: 0644 }

checksum:
  name_template: checksums.txt       # the default would be wopr_<version>_checksums.txt

snapshot:
  version_template: "{{ incpatch .Version }}-snapshot.{{ .ShortCommit }}"

report_sizes: true

release:
  disable: true                      # publishing is release.yml's publish job (gh + attest)

changelog:
  use: git
```

---

## Appendix B — Canonical screen text (provenance-tagged; moves to `lines.go`/`assets` in M1)

Tags: **F** = `film` once confirmed, currently `reconstructed`. **A** = `third-party:abs0/wargames@010ed92:wargames.sh`
(BSD-2, credited). **O** = `original`.

**LOGON gate** (F)

```text
LOGON:
IDENTIFICATION NOT RECOGNIZED BY SYSTEM
--CONNECTION TERMINATED--
HELP NOT AVAILABLE
'GAMES' REFERS TO MODELS, SIMULATIONS AND GAMES WHICH HAVE TACTICAL AND STRATEGIC APPLICATIONS.
```

**Games list** (F). The film order, with a blank line before the last entry:

```text
FALKEN'S MAZE
BLACK JACK
GIN RUMMY
HEARTS
BRIDGE
CHECKERS
CHESS
POKER
FIGHTER COMBAT
GUERRILLA ENGAGEMENT
DESERT WARFARE
AIR-TO-GROUND ACTIONS
THEATERWIDE TACTICAL WARFARE
THEATERWIDE BIOTOXIC AND CHEMICAL WARFARE

GLOBAL THERMONUCLEAR WAR
```

**Backdoor header** (A, byte-identical to abs0's transcription), followed by a status burst (A/F), e.g.
`SYSPROC FUNCT READY  ALT NET READY`, `CPU AUTH RY-345-AX3  SYSCOMP STATUS: ALL PORTS ACTIVE`, `(311) 936-2364`,
then a clear:

```text
#45     11456          11009          11893          11972        11315
PRT CON. 3.4.5.  SECTRAN 9.4.3.                      PORT STAT: SD-345

(311) 699-7305
```

**Greeting** (F)

```text
GREETINGS PROFESSOR FALKEN.
HOW ARE YOU FEELING TODAY?
EXCELLENT. IT'S BEEN A LONG TIME. CAN YOU EXPLAIN THE REMOVAL OF YOUR USER ACCOUNT ON 6/23/73?
YES THEY DO. SHALL WE PLAY A GAME?
```

`USER ACCOUNT` follows built1n and the spoken line; abs0 has `USER ACCOUNT NUMBER`. Settled in M5.

**GTW-vs-chess** (F): `WOULDN'T YOU PREFER A GOOD GAME OF CHESS?` → `FINE.`

**Global Thermonuclear War** (F)

```text
WHICH SIDE DO YOU WANT?

  1.    UNITED STATES
  2.    SOVIET UNION

PLEASE CHOOSE ONE:
AWAITING FIRST STRIKE COMMAND
PLEASE LIST PRIMARY TARGETS BY
CITY AND/OR COUNTY NAME:
```

Both fan transcripts have `COUNTY`; v1 had `COUNTRY`. Settled in M5. The `TRAJECTORY HEADING` table values are
generated (O).

**Call-back dialogue** (F)

```text
I'M SORRY TO HEAR THAT, PROFESSOR.
YESTERDAY'S GAME WAS INTERRUPTED.
ALTHOUGH PRIMARY GOAL HAS NOT YET BEEN ACHIEVED, SOLUTION IS NEAR.
YOU SHOULD KNOW PROFESSOR. YOU PROGRAMMED ME.
TO WIN THE GAME.
OF COURSE. I SHOULD REACH DEFCON 1 AND LAUNCH MY MISSILES IN 61 HOURS.
WOULD YOU LIKE TO SEE SOME PROJECTED KILL RATIOS?
WHAT'S THE DIFFERENCE?
YOU ARE A HARD MAN TO REACH. COULD NOT FIND YOU IN SEATTLE AND NO TERMINAL IS IN OPERATION AT YOUR
CLASSIFIED ADDRESS.
DOD PENSION FILES INDICATE CURRENT MAILING AS: DR. ROBERT HUME (A.K.A. STEPHEN W. FALKEN)
5 TALL CEDAR ROAD GOOSE ISLAND, OREGON 97014
```

**Kill-ratio table layout** (F). `UNITS DESTROYED | MILITARY ASSETS | UNITS DESTROYED`, with these rows:

| Group | Rows |
|---|---|
| Military assets | BOMBERS, ICBM, ATTACK SUBS, TACTICAL AIRCRAFT, GROUND FORCES |
| `CIVILIAN ASSETS` | HOUSING, COMMUNICATIONS, TRANSPORTATION, FOOD STOCKPILES, HOSPITALS |
| `HUMAN RESOURCES` | NON-FATAL INJURED, POPULATION DEATHS (millions) |

The values are generated (O).

**NORAD notices** (F; British spelling as on screen): `** IDENTIFICATION NOT RECOGNISED **`,
`** ACCESS DENIED **`, `** GAME ROUTINE RUNNING **`, `** IMPROPER REQUEST **`,
`** ROUTINE MUST COMPLETE BEFORE RESET **`. Launch code `CPE 1704 TKS`. DEFCON ladder 5 → 1.

**Ending** (F)

```text
ONE OR TWO PLAYERS?
PLEASE LIST NUMBER OF PLAYERS:
STALEMATE. WANT TO PLAY AGAIN?
WINNER: NONE
GREETINGS PROFESSOR FALKEN.
A STRANGE GAME.
THE ONLY WINNING MOVE IS
NOT TO PLAY.

HOW ABOUT A NICE GAME OF CHESS?
```

**Montage scenario names** (A, credited; verified or replaced in M5): `U.S. FIRST STRIKE`, `USSR FIRST STRIKE`,
`NATO / WARSAW PACT`, `FAR EAST STRATEGY`, `US USSR ESCALATION`, `MIDDLE EAST WAR`, `USSR CHINA ATTACK`,
`INDIA PAKISTAN WAR`, `MEDITERRANEAN WAR`, `HONGKONG VARIANT`, `SEATO DECAPITATING`, `CUBAN PROVOCATION`,
`ATLANTIC HEAVY`, `CUBAN PARAMILITARY`, `NICARAGUAN PREEMPTIVE`, `PACIFIC TERRITORIAL`, `BURMESE THEATERWIDE`,
`TURKISH DECOY`, `NATO LIGHT`, `ARGENTINA ESCALATION`, `ICELAND MAXIMUM`, `ARABIAN THEATERWIDE`,
`U.S. SUBVERSION`, `AUSTRALIAN MANEUVER`, `SUDAN SURPRISE`, `NATO TERRITORIAL`, `ZAIRE ALLIANCE`,
`ICELAND INCIDENT`, `ENGLISH ESCALATION`, `MIDDLE EAST HEAVY`, `MEXICAN TAKEOVER`, `CZECH OPTION`,
`FRENCH ALLIANCE`, `ARABIAN CLANDESTINE`, `GABON REBELLION`, `SEATO TAKEOVER`, `HAWAIIAN ESCALATION`,
`TAIWAN DOMESTIC`, `MONGOLIAN THRUST`, `POLISH DECOY`, `ALASKAN DISCRETIONARY`, `CANADIAN THRUST`,
`S.AFRICAN DOMESTIC`, `TUNISIAN INCIDENT`, `MALAYSIAN MANEUVER`. The remainder of the list is completed from the
same source in M2.

**Original lines** (O): the LOGON hint, `WHICH GAME?`, `** GAME ROUTINE NOT AVAILABLE **`,
`** REQUEST CANCELLED **`, `** PRESS ESC AGAIN TO END GAME **`, the GTW climax hints, `PROCESSING`, and every
fallback reply.

---

## Appendix C — Target screens (80×24)

**Backdoor greeting** (`imsai`). WOPR is upper case and revealed at modem speed. The user's input is echoed in
mixed case:

```text
GREETINGS PROFESSOR FALKEN.

Hello.

HOW ARE YOU FEELING TODAY?

I'm fine. How are you?

EXCELLENT. IT'S BEEN A LONG TIME. CAN YOU EXPLAIN
THE REMOVAL OF YOUR USER ACCOUNT ON 6/23/73?

People sometimes make mistakes.

YES THEY DO. SHALL WE PLAY A GAME?

Love to. How about Global Thermonuclear War?

WOULDN'T YOU PREFER A GOOD GAME OF CHESS?

Later. Let's play Global Thermonuclear War.

FINE.

█
```

**GTW big board** (`norad`, `LayoutFull`). The map is a placeholder until the original art lands in M2.
Trajectory values are illustrative. Incoming tracks draw `*` and outgoing `+`. The current DEFCON level is
reversed:

```text
                     GLOBAL THERMONUCLEAR WAR                        DEFCON
                                                                     +---+
   +---------------------------------------------------------+       | 5 |
   |                                                         |       | 4 |
   |   [ map: North America left, USSR right; original       |       |[3]|
   |     line-segment ASCII art, about 57x7, drawn in M2 ]   |       | 2 |
   |                                                         |       | 1 |
   +---------------------------------------------------------+       +---+
             UNITED STATES                  SOVIET UNION

TRAJECTORY HEADING   TRAJECTORY HEADING   TRAJECTORY HEADING
------------------   ------------------   ------------------
A-SS20-A 318 742     C-SS20-A 611 095     E-SS20-A 207 468
       B 154 803            B 470 266            B 932 571

STRIKE ASSESSMENT IN PROGRESS
-----------------------------
INCOMING  *   OUTGOING  +


█
-------------------------------------------------------------------------------
 DEFCON 3   PORT STAT: SD-345   1200 BAUD   ALT NET READY
```

**Chess** (`imsai`, `LayoutPanel`, `PanelRows` 12). The board is gridless, per the no-chrome rule. WOPR is
thinking:

```text

                                    CHESS

                         8   r  n  b  q  k  b  n  r
                         7   p  p  p  p  .  p  p  p
                         6   .  .  .  .  .  .  .  .
                         5   .  .  .  .  p  .  .  .
                         4   .  .  .  .  P  .  .  .
                         3   .  .  .  .  .  N  .  .
                         2   P  P  P  P  .  P  P  P
                         1   R  N  B  Q  K  B  .  R
                             a  b  c  d  e  f  g  h

YOUR MOVE: e4

WOPR: E7E5

YOUR MOVE: Nf3



PROCESSING ..
█

```
