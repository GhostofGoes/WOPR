# WOPR — Architecture & Scope Plan (v2)

_Status: v2, 2026-10-06. Supersedes v1 (commit `ada63a0`). Every change answers a finding in
[`docs/reviews/PLAN-v1-review.md`](reviews/PLAN-v1-review.md); finding ids such as A-3 are cited inline.
[`docs/reviews/PLAN-v2-review.md`](reviews/PLAN-v2-review.md) reviews this version. Implementation follows the
milestones at the end._

**How to read this plan.** It states intent, contracts and constraints. Config files (prek, golangci-lint,
GoReleaser, workflows) appear only as **M0 seeds** in Appendix A. Once M0 lands, the files in the repository
are authoritative, and Appendix A is deleted rather than kept in sync (M-2).

---

## 1. Requirements

Each requirement has a source: **U** = stated by the owner; **D** = derived from a U requirement; **P** = a
plan default the owner may override.

| # | Requirement | Src | Acceptance |
|---|---|---|---|
| R1 | Go terminal-UI recreation of WOPR from *WarGames* (1983), binary named `wopr`. | U | — |
| R2 | Every game on the film's `LIST GAMES` (15 entries), plus the tic-tac-toe ending. | U | Each game meets the *definition of done* in §6. |
| R3 | Persona faithful to the film and to the brother's "WOPR Simulator" prompt: ALL CAPS, addresses the user as PROFESSOR, terse, numbered lists, refuses invalid moves. | U | Scripted-brain table tests plus golden transcripts (§9). |
| R4 | Single static binary, **Linux and Windows**. | U | CI smoke-runs the release binaries on each OS (§10). |
| R5 | **macOS** (Intel and Apple Silicon) as well. | U (decision 5, v1) | Same as R4. |
| R6 | Size: **target ≤ 10 MB, hard limit 15 MB** per binary. | U (confirmed 2026-10-06) | CI warns above 10 MB and fails above 15 MB. |
| R7 | `wopr` starts the TUI; `--version`, `--help`, `--games`, `-p/--play <game>`. | U | §5 CLI tests. |
| R8 | Optional LLM hookup, or purely scripted responses. | U | Scripted in v1; LLM in M6 behind `Brain` (§4.6). |
| R9 | Every long flag has a one-letter shorthand. | U (decision 6, v1) | `flags_test.go`. |
| R10 | Public repository: no secrets, least-privilege CI, reproducible releases with provenance. | D | §11. |
| R11 | First release ships after the film set pieces; the remaining games arrive in minor releases. | U (2026-10-06) | Milestones (§13). |

Supported platforms (R4/R5): Ubuntu 22.04+ (amd64, arm64), macOS ⟦TBD-macOS-floor⟧+ (amd64, arm64),
Windows 11+ (amd64, arm64). Static binaries with `CGO_ENABLED=0`.

---

## 2. Research findings (what the film showed)

Sources: fan transcripts and simulators on GitHub (`built1n/wargames`, `abs0/wargames`, `elfuska/wargames`),
production-technology articles, and the brother's prompt. Those repositories are used **as references only**.
No text or art is copied from them (L-1; their licences are listed in the review's verification log). Each
line in `internal/wopr/lines.go` and each asset carries a provenance comment: `film` (as seen on screen), or
`original` (written for this project). Details that could not be confirmed against the film are marked
*reconstructed*; M5 includes a viewing pass to correct them.

### 2.1 Games list (verbatim, film order; a blank line precedes the last entry on screen)

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

Tic-tac-toe is not on the list but is played at the climax ("play tic-tac-toe with yourself").

### 2.2 Canonical screen text (short quotations used by the scripted brain)

⟦TBD-film-facts: confirm LOGON/header order and "connection terminated" behaviour⟧

- **LOGON gate.** `LOGON: ` → unknown name → `IDENTIFICATION NOT RECOGNIZED BY SYSTEM` /
  `--CONNECTION TERMINATED--`. `HELP LOGON` → `HELP NOT AVAILABLE`. `HELP GAMES` → `'GAMES' REFERS TO MODELS,
  SIMULATIONS AND GAMES WHICH HAVE TACTICAL AND STRATEGIC APPLICATIONS.` `LIST GAMES` → the list above.
  `Joshua` → backdoor.
- **Backdoor header**, followed by a burst of status lines (`SYSPROC FUNCT READY  ALT NET READY`, …), a clear,
  then `GREETINGS PROFESSOR FALKEN.`

  ```text
  #45     11456          11009          11893          11972        11315
  PRT CON. 3.4.5.  SECTRAN 9.4.3.                      PORT STAT: SD-345

  (311) 699-7305
  ```

- **Greeting scene.** `HOW ARE YOU FEELING TODAY?` → `EXCELLENT. IT'S BEEN A LONG TIME. CAN YOU EXPLAIN THE
  REMOVAL OF YOUR USER ACCOUNT NUMBER ON 6/23/73?` → `YES THEY DO. SHALL WE PLAY A GAME?`
- **GTW-vs-chess scene** (a separate scene, not part of the greeting): the first request for Global
  Thermonuclear War → `WOULDN'T YOU PREFER A GOOD GAME OF CHESS?`. A repeated request (`LATER. LET'S PLAY
  GLOBAL THERMONUCLEAR WAR.`) → `FINE.`, and the game starts.
- **Global Thermonuclear War.** ASCII map of the US and USSR; `WHICH SIDE DO YOU WANT?` / `1. UNITED STATES` /
  `2. SOVIET UNION` / `PLEASE CHOOSE ONE:`; `AWAITING FIRST STRIKE COMMAND` (underlined) / `PLEASE LIST
  PRIMARY TARGETS BY CITY AND/OR COUNTRY NAME:`; then the map with trajectories and a `TRAJECTORY HEADING`
  table.
- **Call-back dialogue, kill-ratio tables, NORAD notices** (`** IDENTIFICATION NOT RECOGNISED **`,
  `** ACCESS DENIED **`, `** GAME ROUTINE RUNNING **`, `** IMPROPER REQUEST **`, `** ROUTINE MUST COMPLETE
  BEFORE RESET **`), launch code `CPE 1704 TKS`, DEFCON ladder 5 → 1: as listed in v1 §"Canonical screen
  text". These are unchanged and move verbatim into `lines.go` in M1.
- **Ending.** Tic-tac-toe self-play at increasing speed, then the scenario montage (each ends
  `WINNER: NONE`), then `A STRANGE GAME. THE ONLY WINNING MOVE IS NOT TO PLAY.` / `HOW ABOUT A NICE GAME OF
  CHESS?`
- **Scenario montage names** (*reconstructed*, ~60): the list in v1 moves to
  `internal/assets/scenarios.go` with provenance `reconstructed`.

### 2.3 Visual aesthetics

⟦TBD-film-facts: monitor / text colour⟧

- **David's terminal**: bright, slightly cool white text on black, chunky uppercase glyphs, double-spaced
  WOPR lines, a solid blinking block cursor, and text appearing at modem speed. → The default theme is
  `imsai` (white phosphor); `green` and `amber` are alternates.
- **NORAD war room**: dark blue/black field, cyan/blue map outlines, red and yellow missile tracks, white
  labels, a DEFCON column of boxed numerals 5…1, and `** … **` notices. → The `norad` theme and the big
  board.
- **WOPR cabinet**: rows of blinking lights and a `W.O.P.R.` nameplate. → An optional front-panel strip that
  also serves as the "thinking" indicator (§4.3).
- **Verbiage rules**: ALL CAPS, terse, declarative; never asks clarifying questions; numbered lists
  `1.  2.  3.`; refuses invalid actions with `** IMPROPER REQUEST **`-style lines; calls the user PROFESSOR.
- **Anti-pattern**: tview-style bordered panes and labelled input boxes. The console is full-bleed text with
  no chrome. Borders appear only inside the big board and in NORAD notices.

### 2.4 Target screens

The two mockups from v1 (backdoor logon/greeting; GTW big board in `norad`) and the optional status strip
remain the visual targets. They move to `docs/screens.md` in M0. M1 adds two more mockups at **exactly
80×24**: chess (board panel + console strip) and Hearts (trick view + console strip). These prove that the
`Panel` layout fits (U-5).

---

## 3. Decisions

| # | Decision | Choice | Src |
|---|---|---|---|
| 1 | TUI stack | **Bubble Tea v2 + Lip Gloss v2** (no bubbles). Least custom plumbing, declarative views, Windows and colour downsampling built in, best testability. The M0 spike measured ⟦TBD-size⟧, well inside R6, so no fallback renderer is planned (D-3). | U |
| 2 | LLM | **Scripted brain in v1** behind `Brain`; LLM provider is M6, optional. | U |
| 3 | Military sims | **Short turn-based mini-wargames**, each a **data scenario** on one `sim` engine, with kill-ratio tables (G-2). | U (+ review) |
| 4 | LOGON gate | **Film-faithful** (`Joshua` backdoor), hints after repeated failures; `--play` bypasses it. | U |
| 5 | Platforms | Linux/macOS/Windows × amd64/arm64, static, CGO off (R4/R5). | U |
| 6 | Arg parsing | stdlib `flag`; every long flag has a one-letter shorthand; one bare positional means `--play`, and flags may follow it (C-1). No cobra. | U |
| 7 | Persistence | None in v1; `WOPR_*` env vars and flags only. | P |
| 8 | Toolchain | `go` directive = the highest minimum any dependency declares (⟦TBD-go-floor⟧); `toolchain go1.27.1` (D-1). Bubble Tea v2.0.10, Lip Gloss v2.0.6. | U (+ review) |
| 9 | Lint | **prek** with **`prek.toml`**; Go-standard tools, gitleaks, zizmor, codespell, ShellCheck, rumdl; same config locally and in CI. | U |
| 10 | Size budget | **Target ≤ 10 MB, hard limit 15 MB** (R6). | U |
| 11 | CI topology | **Build once** with GoReleaser (snapshot on every run, release on tags); **run each binary natively** on its OS/arch, arm64 included (B-2). | U (2026-10-06) |
| 12 | Docs | README in M0, completed in M5. **AGENTS.md in M0**, updated each milestone (M-1). | U (+ review) |
| 13 | First release | **v0.1.0 at the end of M2** (film path + chess + checkers); a minor release per later milestone; v1.0.0 after M5 (P-2). | U (2026-10-06) |
| 14 | Bridge | **Minimal variant**: WOPR bids all four seats by point count; the user plays declarer (G-3). | U (2026-10-06) |
| 15 | Chess engine | ⟦TBD-chess⟧ | P |

---

## 4. Architecture

### 4.1 Module, layout and the import DAG

Module `github.com/GhostofGoes/WOPR`, binary `wopr`.

**Import DAG** (A-6). Edges not listed are forbidden:

```text
cmd/wopr      → cli, ui, version
cli           → games (registry only), theme (names only), version
ui            → proto, wopr, games, theme, charm.land/*
wopr          → proto, games (registry and Info only)
games/<game>  → proto, games, games/{ai,cards,board,prompt}, sim (M4)
games/*       → proto
sim           → proto
theme         → proto, charm.land/lipgloss/v2
proto         → stdlib only
```

Only `ui` and `theme` may import `charm.land/*`, and only `ui` may import `charm.land/bubbletea/v2`. The
DAG is enforced by **`internal/archtest`**, a plain Go test that runs `go list -deps -json` and checks every
edge. That makes it authoritative, linter-independent, and part of `go test ./...`. The single-clock rule is
enforced by forbidigo (§12).

```text
cmd/wopr/main.go             flags → dispatch (print-and-exit, or run TUI) → exit code
internal/version/            Version (-X) with fallback to debug.ReadBuildInfo().Main.Version, then "(devel)" (D-4)
internal/cli/                Parse(args, stdout, stderr) (Config, Action, error); usage; --games renderer
internal/proto/              the vocabulary shared by ui and every "program" (persona, games):
  event.go output.go         Event / Output types (§4.4)
  key.go                     Key enum (Up, Down, Left, Right, Enter, Esc, Backspace, Tab, Rune) (A-8)
  canvas.go                  Canvas of cells with semantic Style tags (§4.5) (A-2)
  pace.go rand.go            Pace; NewRand(seed, stream) over math/rand/v2 PCG (A-11)
internal/theme/              palettes: map proto.Style → lipgloss.Style per theme
internal/ui/                 Bubble Tea application — the ONLY package that imports bubbletea
  app.go                     thin root router: screen mode, global keys, size, dispatch to sub-models (M-3)
  clock.go                   the single generation-tagged tick source (§4.3)
  host.go                    runs a proto "program" (persona or game): routes Events, applies Outputs, runs Think
  console/                   scrollback, typewriter, line editor, Sanitize, render
  screens/                   connect, too-small, canvas renderer (proto.Canvas → styled string), front panel
  testutil/drive.go          synchronous driver for golden tests
internal/wopr/               the persona (no tea import)
  session.go                 owns the conversation phase (A-5), said-flags, side, history; Snapshot()
  normalize.go intent.go     normalisation; explicit game-intent parsing (A-7)
  scenes.go commands.go      film scenes; phase-gated commands
  brain.go scripted.go       Brain interface; rule-table ScriptedBrain
  lines.go                   all script lines as Go constants, with provenance comments (A-10)
internal/games/              registry and shared game libraries (no tea import)
  registry.go                ordered registry in film order; Resolve(...)
  ai/                        alpha-beta/negamax with iterative deepening + time budget over a Position interface
  cards/ board/ prompt/      decks and trick logic; board drawing and cursor; numbered-menu and yes/no parsing
  testkit/                   play a game from scripted inputs; runs Think on a goroutine so -race sees captured state
  falkensmaze/ blackjack/ ginrummy/ hearts/ bridge/ checkers/ chess/ poker/ gtw/ tictactoe/
  fightercombat/ guerrilla/ desertwarfare/ airtoground/ theaterwide/ biotoxic/   (thin wrappers over sim scenarios)
internal/sim/                M4: engine (map, units, actions, combat-results table, events, turn loop, kill ratios)
internal/assets/             ASCII art (original), scenario names
internal/archtest/           import-DAG test
internal/llm/   (M6)         Brain backed by an LLM over net/http (D-2)
```

### 4.2 Runtime flow and state ownership

1. `main` parses flags. `--version`, `--help` and `--games` print and exit without touching the terminal.
   They ignore `EPIPE` (C-3). Otherwise, if **stdin or stdout** is not a terminal, or `TERM=dumb`, print one
   line naming the cause (on Windows, suggest Windows Terminal) and exit 2 (U-4).
2. The TUI runs on the **alternate screen** and paints the theme background over the whole frame. On exit,
   the normal screen is restored and `--CONNECTION TERMINATED--` is printed (U-1).
3. **State ownership** (A-5). `wopr.Session` owns the conversation phase: `Logon → Greeting → Shell →
   InGame → Ending`. `ui` owns only the *screen mode* (`Connecting`, `Console`, `GameLayout`, `TooSmall`)
   and derives it from the session phase, the active game's `Layout`, and the terminal size.
4. **Connect** (dial animation) → `LOGON:` gate → backdoor header → greeting scene → **Shell**.
   ⟦TBD-film-facts: a wrong name prints `IDENTIFICATION NOT RECOGNIZED BY SYSTEM` / `--CONNECTION
   TERMINATED--`, then re-dials and shows `LOGON:` again; it never exits the program.⟧
5. Shell input passes through `Sanitize` (§4.3), then is dispatched in this order: **active scene** →
   **commands** (phase-gated) → **explicit game intent** (A-7) → **Brain**. Scenes, commands and intent are
   deterministic and synchronous. The Brain is asynchronous, and **input is locked while a reply is
   pending** (A-4).
6. Starting a game hands the host a `games.Game`. When the game emits `Done`, the persona turns its
   `Result` into an in-character verdict and returns to Shell.
7. `--play <game>` starts in `InGame` with the greeting marked as done. `Done` falls through to Shell.
8. The ending sequences (tic-tac-toe self-play → scenario montage → `A STRANGE GAME…`) are a game-like
   program, driven by `Animate` on the same clock.

**Explicit game intent** (A-7). A game starts only on: a numbered selection while a list is on screen;
`PLAY <game>`; `LET'S PLAY <game>`; `HOW ABOUT <game>`; or input that is exactly a game name, slug or alias.
All matching is on word boundaries after normalisation. "I don't want to play chess" and "the Golden Gate
bridge" do not start games. The GTW-vs-chess exchange is a scene and fires once.

### 4.3 Console, clock, cursor, input

- **One clock.** `ui/clock.go` owns the only `tea.Tick`. Ticks carry a generation number. At most one is in
  flight, and stale generations are dropped. The interval is the **minimum over active consumers**
  (typewriter, game `Animate`, cursor blink of `Blink` cells, front-panel lights). The host accumulates
  `dt` per consumer and delivers each at its own cadence (A-9). When nothing is busy, no timer runs.
- **dt-based typewriter**: about 30 cps for speech, fast for tables, instant for boards. `--instant` /
  `WOPR_INSTANT=1` and "skip" both mean "advance by infinity".
- **Native cursor**: `tea.View.Cursor` with block shape and blink, parked at the end of the revealing line
  or on the input line. ⟦TBD-bt-api⟧
- **Skip policy** (U-3). While the console is revealing text, the first key flushes the queue. Printable keys
  are *also* delivered to the input line, so typeahead is kept. Enter and Space only skip. PgUp and PgDn
  always scroll. Ctrl+C always quits. All of this lives in `Update`, so it is unit-testable.
- **Thinking.** While a `Think` (game AI) or a Brain reply is pending, the input line is locked and the
  front-panel lights animate (or the cursor blinks, if the strip is hidden). Esc cancels via the context.
- **Sanitize** (U-2). Every string that did not originate in the binary (typed input, `PasteMsg`, Brain
  replies, LLM output) passes through `console.Sanitize`, which removes C0 controls except `\n` and `\t`, and
  removes C1 controls and ESC. The input line is capped at 256 runes; pastes are truncated to that.
  `Sanitize` and the line editor are fuzz-tested.
- **Scrollback**: logical lines wrapped at render time and cached per width, capped at 2000, auto-follow on
  new output. `Render(w, h)` emits exactly `h` rows.
- **Line editor** (~250 lines): echo as typed, history ↑/↓, Ctrl+U, Backspace, width-aware via
  `ansi.StringWidth` (U-6). WOPR text is upper-case; user input stays as typed.
- **Sizing**: blank until the first `WindowSizeMsg`. Below 80×24, the centred `TERMINAL TOO SMALL` card is
  shown, while ticks keep running and games keep their state. Larger terminals centre the 80-column big
  board. A game receives a `ResizeEvent` only when its layout area changes.

### 4.4 The program protocol (`internal/proto`)

The persona and every game speak the same protocol to the host. `ui/host.go` is therefore the only place
that turns Events and Outputs into Bubble Tea messages and commands (A-1, A-3).

```go
// internal/proto — stdlib only.
type Layout uint8 // static, from Info: where View goes
const (
	LayoutConsole Layout = iota // console only (Teletype games, the persona)
	LayoutPanel                 // View on top; console strip + input below
	LayoutFull                  // View fills the screen; a 3-line console strip at the bottom
)

// Events: host → program.
type Event interface{ isEvent() }
type LineEvent   struct{ Text string }              // answers the last Prompt (sanitised, trimmed)
type KeyEvent    struct{ Key Key; Rune rune }       // only while AwaitKeys is active
type TickEvent   struct{ Dt time.Duration }         // at the cadence the program asked for
type ResizeEvent struct{ Width, Height int }        // the program's layout area
type ThinkDone   struct{ Value any; Err error }     // result of the last Think (Err = context.Canceled on Esc)

// Outputs: program → host.
type Output interface{ isOutput() }
type Say       struct{ Lines []string; Pace Pace }  // through the typewriter
type Prompt    struct{ Text string }                // line-input mode: host shows the line editor
type AwaitKeys struct{ Hint string }                // key mode: host routes KeyEvents (maze, board cursor)
type Animate   struct{ Every time.Duration }        // 0 stops
type Think     struct {                             // run off the UI goroutine
	Fn     func(ctx context.Context) (any, error)   // must use only data captured by value
	Budget time.Duration                            // the host cancels ctx after Budget
}
type Redraw struct{}                                // View changed
type Done   struct{ Result Result }

type Outcome uint8 // Win, Loss, Draw, NoWinner, Aborted
type Result struct{ Outcome Outcome; Lines []string } // Lines = kill ratios, verdict detail
type Pace uint8    // PaceSpeech, PaceTable, PaceInstant
```

```go
// internal/games/game.go
type Status uint8 // Playable, Planned — Planned games are listed; WOPR declines in character (P-2)

type Info struct {
	Number  int      // position on LIST GAMES (1..15); 0 = unlisted (tic-tac-toe) (G-4)
	Name    string   // film spelling, upper case
	Slug    string
	Aliases []string
	Layout  proto.Layout
	Status  Status
	Blurb   string
}

type Env struct {
	Rand          *rand.Rand // math/rand/v2, this game's own stream
	Width, Height int
	Instant       bool
}

type Game interface {
	Info() Info
	Start(Env) []proto.Output
	Handle(proto.Event) []proto.Output
	View(c *proto.Canvas) // draw into the layout area; never called for LayoutConsole
}
```

Rules:

- **Input mode is dynamic.** Emitting `Prompt` puts the host in line mode, and the next `LineEvent` answers
  it. Emitting `AwaitKeys` switches to key mode. Chess and GTW therefore use the shared line editor, and the
  maze uses keys. The two do not overlap.
- **Esc** is handled once, by the host. It cancels a pending `Think`, and a second Esc aborts the game
  (`Done{Aborted}`). Games never handle Esc.
- **Think** is how every AI move runs (A-3). `Fn` must capture an immutable copy of the position. The game
  mutates its state only in `Handle(ThinkDone)`. AIs use iterative deepening and return the best move found
  when `ctx` expires, so play speed is the same on every machine. `testkit` runs `Fn` on a goroutine, so
  `go test -race` catches captured mutable state.
- **The persona speaks `proto` too.** It also emits `StartGame{Slug}`, `Logoff{}` and `AskBrain{Input}`,
  which are host-only outputs declared in `wopr`. The host validates `StartGame` against the registry.

### 4.5 Canvas and themes

`proto.Canvas` is a `W×H` grid of `Cell{R rune; S Style; A Attr}`. `Style` is **semantic**: `Text`,
`Bright`, `Dim`, `Accent`, `Incoming`, `Outgoing`, `Land`, `Label`, `Defcon1`…`Defcon5`, `Alert`. `Attr` is
`Blink`, `Reverse` or `Underline`. Games draw with `Put(x, y, s, style, attr)`. `ui/screens/canvas.go` maps
each Style to the active theme's Lip Gloss style. `Blink` is rendered by toggling on the clock, not with SGR
5, which terminals support unevenly. `Canvas.String()` (text only) and `Canvas.StyleMap()` (one letter per
cell) make golden tests readable and theme-independent (A-2).

| Theme | Text | Bright | Dim | Accent | Background |
|---|---|---|---|---|---|
| `imsai` (default) | `#DCE6F0` | `#FFFFFF` | `#7A8694` | `#9EC5FF` | `#000000` |
| `green` (P1) | `#33FF33` | `#B6FFB6` | `#1A8C1A` | `#33FF33` | `#000000` |
| `amber` (P3) | `#FFB000` | `#FFD27A` | `#8A5E00` | `#FFB000` | `#000000` |
| `norad` | `#5AC8FA` | `#FFFFFF` | `#2F6FBF` | `#FF3B30` / `#FFD60A` | `#02060F` |

DEFCON colours in all themes: 5 blue, 4 green, 3 yellow, 2 red, 1 white-on-red. Tracks: red incoming, yellow
outgoing. Colour downsampling and `NO_COLOR` are handled by Bubble Tea's colour profile ⟦TBD-bt-colour⟧.
Pure ASCII art everywhere by default.

### 4.6 Persona and Brain

```go
// internal/wopr/brain.go
type Snapshot struct {        // a value copy taken in Update; safe to hand to another goroutine
	Phase   Phase
	Turn    int
	Seed    uint64
	Said    map[string]bool   // copied
	Flags   map[string]bool   // copied
	Side    string
	History []Exchange        // bounded (last 20)
}

type Effect interface{ isEffect() }
type MarkSaid  struct{ ID string }
type SetFlag   struct{ Key string }
type StartGame struct{ Slug string }
type Logoff    struct{}

type Reply struct{ Lines []string; Effects []Effect }

type Brain interface {
	Reply(ctx context.Context, s Snapshot, input string) (Reply, error)
}
```

- `Update` takes a `Snapshot`, runs `Reply` in a `tea.Cmd`, locks input, and applies `Effects` when the
  reply arrives (A-4). Nothing outside `Update` mutates the `Session`.
- The scripted brain draws `Pick` choices from `proto.NewRand(s.Seed, uint64(s.Turn))`, so transcripts are
  reproducible and no RNG is shared across goroutines.
- If a reply errors or times out (scripted: never; LLM: 20 s), the persona answers with a scripted fallback
  line.
- The rule table (`Rule{ID, When, Match, Lines|Pick, Once, Effects}`) is unchanged from v1 apart from
  `Effects`. First match wins, specific before general, fallbacks last.

---

## 5. CLI

```text
wopr                         start the TUI (connect → LOGON)
wopr -v | --version          version, commit, build date, Go version
wopr -h | --help             usage (documents LOGON: Joshua)
wopr -g | --games            numbered list in film order (+ slugs; tic-tac-toe under "ALSO AVAILABLE")
wopr -p | --play <game>      launch a game (number, slug, alias, name, or unique prefix)
wopr <game> [flags]          same as --play <game>; flags may come before or after
wopr -t | --theme <name>     imsai | green | amber | norad      (env WOPR_THEME)
wopr -i | --instant          no typewriter pacing               (env WOPR_INSTANT=1)
wopr -s | --seed <n>         deterministic RNG for demos/tests
```

- **Interspersed positional** (C-1). `cli.Parse` calls `fs.Parse` in a loop. Each time it stops at a
  positional, the positional is recorded and parsing resumes on the rest. More than one positional, or a
  positional together with `--play`, is a usage error (exit 2). `--` ends flag parsing. Tests cover
  `wopr gtw -i`, `wopr -i gtw`, `wopr -p gtw -i`, and `wopr gtw chess` (error).
- Stdlib `flag` accepts `-games` and `--games`, and `-p chess` and `-p=chess`. It does not accept `-pchess`
  or combined `-is 1`. The help text shows the accepted forms, and tests pin them (C-2).
- `-v` means version, deliberately; a future verbose flag would be `--verbose`/`-V` (C-4).
- `games.Resolve`: number (1–15) → slug → alias → exact normalised name → unique prefix. `Number 0` is never
  matched by number. Ambiguity (e.g. `theaterwide`) prints the candidates and exits 2. `Planned` games
  resolve, and WOPR declines in character.
- Exit codes: 0 ok; 1 runtime error (Bubble Tea restores the terminal and returns the wrapped panic);
  2 usage, not a TTY, or ambiguous game; 130 on Ctrl+C (`tea.ErrInterrupted`). ⟦TBD-bt-errors⟧

---

## 6. Games

**Definition of done** (R-2), for every game: rules implemented; WOPR plays legally, using `Think` for any
search; Esc abort works; the game ends with an in-character verdict; a `testkit` transcript test exists
with a fixed seed; the 80×24 layout fits; and the README has a one-line "how to play".

| # | Game | Layout | Input | Milestone | Notes |
|---|---|---|---|---|---|
| 1 | Falken's Maze | Panel | keys | M4 | Procedural maze with fog of war. WOPR "learns" a turn bias and re-routes walls; a property test asserts the maze stays solvable after every re-route (G-6). |
| 2 | Black Jack | Console | line | M3 | Dealer stands on soft 17 (configurable); split/double; chips per session. |
| 3 | Gin Rummy | Panel (hand) | line | M3 | Knock/gin scoring; deadwood-minimising heuristic. |
| 4 | Hearts | Panel (trick) | line | M3 | 4 seats; passing; shoot-the-moon; heuristic AI on `cards/trick.go`. |
| 5 | Bridge | Panel (trick) | line | M3 (last) | **Minimal** (G-3): WOPR bids all four seats with point-count rules; the user is declarer (South) and plays both hands; WOPR defends with `cards/trick.go` heuristics. |
| 6 | Checkers | Panel | line + keys | M2 | 8×8, forced captures, kings; `games/ai` with iterative deepening. |
| 7 | Chess | Panel | line | M2 | ⟦TBD-chess⟧ Input `e2e4` / `Nf3`. |
| 8 | Poker | Console | line | M3 | 5-card draw heads-up; betting heuristic + bluff probability. |
| 9–14 | Military sims | Console/Panel | line | M4 | One `sim` engine; each game is a scenario value (map, forces, enabled actions, events, victory rule, turn limit) plus at most a few hooks (G-2). Biotoxic always ends `WINNER: NONE`. |
| 15 | Global Thermonuclear War | Full | line | M2 | **Self-contained** in `games/gtw` (P-1): side choice, targets, salvos, DEFCON ladder, trajectories, kill ratios, montage → `A STRANGE GAME…`. Cannot be won. Shared pieces move to `sim` in M4 only if a second consumer needs them. |
| — | Tic-Tac-Toe | Panel | line | M2 | Perfect minimax. `NUMBER OF PLAYERS: 0` starts the self-play montage and the ending. Unlisted (`Number 0`), resolvable by name. |

**Sim engine (M4)**, specified before any sim is built. A strip or grid map of named regions; a unit
table (type, strength, mobility, range); a per-scenario action set (move, attack, air strike, patrol,
supply, special); a combat-results table drawn with the game's RNG; an event deck (sandstorm, defection,
contamination spread); a turn limit; and a kill-ratio generator in the film's table format. A sim is about
150 lines of scenario data and hooks, not a bespoke game.

**AI quality tests** (Q-4): tic-tac-toe never loses; checkers forced-capture puzzles; chess mate-in-1/2
puzzles; and each AI beats a random player in ≥ 95% of 200 seeded games.

---

## 7. Binary size and portability

- **Build** (GoReleaser, both snapshot and release; §10): `CGO_ENABLED=0`, `-trimpath`,
  `-ldflags "-s -w -X github.com/GhostofGoes/WOPR/internal/version.Version={{.Version}}"`,
  `mod_timestamp: {{.CommitTimestamp}}` for reproducible archives.
- **Measured** (M0 spike, run during this review): ⟦TBD-size-table⟧
- **Budget**: target ≤ 10 MB, hard limit 15 MB. CI prints sizes in the step summary and the release notes;
  the README states only the budget (B-8). If a change crosses 10 MB, find the heavy packages with
  `go tool nm -size -sort size` before adding code. **Never UPX** (Windows AV false positives).
- **M6 (LLM)** uses `net/http` + `encoding/json` only, no SDK; ⟦TBD-http-delta⟧ (D-2).
- **Compatibility matrix**:

  | OS | Arch | Notes | CI smoke runner |
  |---|---|---|---|
  | Ubuntu 22.04+ | amd64, arm64 | Static; plain VT/ANSI; any xterm-class terminal, tmux, SSH. | ⟦TBD-runners⟧ |
  | macOS ⟦TBD-macOS-floor⟧+ | amd64, arm64 | Unsigned: document `xattr -d com.apple.quarantine wopr`. No Homebrew tap in v1 ⟦TBD-homebrew⟧. | ⟦TBD-runners⟧ |
  | Windows 11+ | amd64, arm64 | Windows Terminal (ConPTY) and conhost. mintty without ConPTY is unsupported and detected (U-4). | ⟦TBD-runners⟧ |

- **Windows specifics**: Ctrl+C arrives as a key in raw mode, so it is handled in `Update` as `tea.Interrupt`
  ⟦TBD-bt-ctrlc⟧. Leave keyboard enhancements off. Verify `WindowSizeMsg` reports the *window* size in
  conhost with a 9001-line buffer. Ship unsigned, and document the SmartScreen prompt.
- **Panics**: no custom `recover` around `p.Run()`; Bubble Tea restores the terminal ⟦TBD-bt-panic⟧. `main`
  maps errors to exit codes. `WOPR_DEBUG=1` writes `os.UserCacheDir()/wopr/debug.log` with mode 0600, never
  typed input (S-5).

---

## 8. Testing

- **Unit**: game rules via `games/testkit`; `games.Resolve`; `cli.Parse` (including interspersed
  positionals); intent parsing (false-positive table: "I don't want to play chess", "golden gate bridge");
  scripted brain (table-driven, seeded); typewriter `dt` math; line editor; `Sanitize`; canvas.
- **Architecture**: `internal/archtest` checks the import DAG (§4.1).
- **Golden** (Q-3): `ui/testutil/drive.go` runs the model synchronously, executes returned commands, unwraps
  `tea.BatchMsg`, feeds synthetic ticks and keys, and snapshots `View()` content after `ansi.Strip`
  ⟦TBD-bt-view⟧. Goldens live in `testdata/*.golden` (LF-only; §10) and are regenerated with
  `go test ./internal/ui/... -update`; diffs are reviewed in the PR. `tea.Sequence` is banned, because its
  message type is unexported and the driver cannot unwrap it ⟦TBD-bt-seq⟧. Flows: `logon_joshua`,
  `logon_fail_x3_hint`, `help_games`, `list_games`, `intent_false_positives`, `play_stub`, then each set
  piece.
- **Fuzz** (Q-2): `normalize`, `Resolve`, intent, the line editor, `Sanitize`, chess move parsing. In CI,
  each runs for 10 s per target on Linux.
- **End-to-end** (Q-1): on every OS, the real binary runs in a pseudo-terminal with `--instant --seed 1`,
  types `Joshua`, waits for `SHALL WE PLAY A GAME?`, sends Ctrl+C, and asserts exit 130 and a restored
  terminal (alt screen left, cursor visible). ⟦TBD-teatest⟧
- **Race**: `go test -race ./...` on Linux and macOS; `testkit` runs `Think` on goroutines so races are
  visible.

---

## 9. Tooling: prek and static analysis

Intent (the M0 seed is in Appendix A.1; the repository file is authoritative afterwards):

- **One hook set in `prek.toml`**, run identically locally (`prek install`, `prek run --all-files`) and in CI
  (`j178/prek-action`).
- **Ordering**: fixers → formatters → read-only linters, using prek's priority mechanism
  ⟦TBD-prek-priority⟧.
- **Hooks**: built-in file checks (whitespace, EOF, LF line endings, BOM, YAML/TOML/JSON syntax, merge
  markers, case conflicts, Windows-illegal names, shebangs, large files, private keys) ⟦TBD-prek-builtins⟧;
  golangci-lint `config-verify`, `fmt` and `full` ⟦TBD-hook-ids⟧; gitleaks (pre-commit only; CI scans
  separately, §11); zizmor (no `--fix`, T-5); codespell (skips script and art data by **prek `exclude`
  regex**, not codespell `--skip`) ⟦TBD-codespell⟧; ShellCheck on `scripts/*.sh`; rumdl check + fmt
  ⟦TBD-rumdl⟧.
- **Pinning** (T-3): every hook `rev` is a full commit SHA with a `# frozen: vX.Y.Z` comment
  ⟦TBD-prek-freeze⟧. A monthly scheduled workflow runs `prek auto-update --freeze` with a cooldown and opens a
  PR using only `GITHUB_TOKEN` (T-7).
- **govulncheck** (T-4): a Go tool in `tools/go.mod` (`go tool -modfile=tools/go.mod govulncheck ./...`).
  It is pinned, updated by Dependabot, and kept out of the main module graph. It runs as a pre-push hook,
  in CI, and on a weekly schedule.
- **golangci-lint** (`.golangci.yml`, v2): `linters.default: standard`, plus `revive`, `gocritic`,
  `misspell`, `forbidigo`. Formatters `gofmt`, `goimports`, `gofumpt`. **forbidigo** (T-2):
  `analyze-types: true` with `{pattern: '^(Tick|Every|Sequence)$', pkg: '^charm\.land/bubbletea/v2$'}`,
  exempted for `internal/ui/clock.go` via `linters.exclusions.rules` ⟦TBD-forbidigo⟧. No local `go vet` hook,
  since golangci-lint's `govet` covers it (T-6).
- **Windows contributors**: every hook works without Bash, except ShellCheck, which needs only the
  shellcheck-py binary. `make` is optional: every Make target is a single command documented in the README
  and AGENTS.md.

---

## 10. CI/CD (GitHub Actions)

All actions are pinned to full commit SHAs. `permissions: contents: read` is the workflow default, and jobs
elevate individually. Only `GITHUB_TOKEN` is used. `actions/setup-go` takes `go-version-file: go.mod`
⟦TBD-setup-go⟧ with `GOTOOLCHAIN=local`. `persist-credentials: false` on every checkout.

**`.gitattributes`** (B-3): `* text=auto eol=lf`, plus `*.png *.gif binary`. With this, shell scripts and
goldens are LF on the Windows runners.

**`ci.yml`**, on `pull_request` and on `push` to `main` (B-6), with
`concurrency: {group: ci-${{ github.ref }}, cancel-in-progress: true}`:

| Job | Runner(s) | Does |
|---|---|---|
| `lint` | `ubuntu-24.04` | `prek run --all-files --show-diff-on-failure` via `j178/prek-action` ⟦TBD-prek-action⟧ |
| `secrets` | `ubuntu-24.04` | gitleaks over the pushed commit range (full history on `main`), pinned binary (S-1) |
| `test` | `ubuntu-22.04`, ⟦TBD-runners: macOS⟧, `windows-2025` | `go test ./...` (`-race` on Linux/macOS); fuzz smoke on Linux; govulncheck on Linux |
| `build` | `ubuntu-24.04` | `goreleaser release --snapshot --clean` → six binaries; `scripts/size-gate.sh`; upload one artifact per target (`wopr_<os>_<arch>`, the release naming) (B-1) |
| `smoke` | one runner per os/arch ⟦TBD-runners⟧ | download that target's artifact; `scripts/smoke.sh`: `--version`, `--games`, `--games \| head -1`, then the pty end-to-end test (§8) |

The smoke matrix runs the *exact* binaries the release would ship (B-2). A target with no native runner is
marked `smoke: skipped (no runner)` in the step summary, never silently omitted.

**`release.yml`** on `v*` tags: the same GoReleaser config, with `setup-go` `cache: false` and no other
caches (B-4). Permissions `contents: write`, `id-token: write` and `attestations: write`, on this job only.
It publishes six archives plus `checksums.txt` and runs `actions/attest-build-provenance` on the archives
(S-2). The README documents `gh attestation verify`. `workflow_dispatch` runs a `--snapshot` dry run with
read-only permissions.

**`scheduled.yml`** (weekly): govulncheck on `main`, and the prek auto-update PR (monthly).

**`dependabot.yml`**: `gomod` (root and `tools/`) and `github-actions`, weekly, grouped minor+patch, with
`cooldown` ⟦TBD-dependabot⟧ (S-6).

**`scripts/`**: `size-gate.sh` and `smoke.sh`, both ShellCheck-clean under `--enable=all`, with
`shopt -s nullglob` and whitespace-trimmed `wc -c` (B-7) ⟦TBD-shellcheck⟧. Builds go through GoReleaser
only, so there is no `build.sh`.

---

## 11. Security and public-repository hygiene

- **Secrets**: none in the tree, ever. gitleaks runs as a pre-commit hook and as the CI `secrets` job.
  **GitHub secret scanning with push protection is enabled**, the control a local `--no-verify` cannot
  bypass (S-1). `.gitignore` adds `*.pem`, `*.key`, `dist/`, `coverage.*` and editor/OS files.
- **Repository settings** (S-3), applied in M0 and listed in AGENTS.md: a ruleset on `main` (required
  checks `lint`, `secrets`, `test`, `build`, `smoke`; no force-push; no deletion); a tag ruleset so only the
  owner can create `v*`; secret scanning + push protection; private vulnerability reporting with
  `SECURITY.md`; Dependabot security updates.
- **Workflows**: `GITHUB_TOKEN` only, least privilege, SHA-pinned, `persist-credentials: false`, no
  `pull_request_target`, no caches in release. zizmor runs in prek, and in CI with `GH_TOKEN` for its online
  audits (T-5).
- **Releases**: checksums plus build-provenance attestations (S-2). No signing keys in v1. Notarisation and
  Authenticode are post-1.0 options.
- **The app**: no network code in v1, no telemetry, no transcripts. All external text is sanitised before
  rendering (U-2). The debug log never holds typed input (S-5).
- **LLM (M6)** (S-4): explicit opt-in (`--llm` / `WOPR_LLM=provider`), with a one-line in-character notice
  when enabled. API keys are read from the environment only and never logged. 20 s timeout, max-tokens cap,
  history bounded to 20 exchanges, sanitised output, actions validated against the registry, no logging of
  prompts or replies. On any error the persona falls back to scripted.
- **Personal information**: none beyond the film's fictional names and addresses. The brother's prompt is
  adapted only with **written permission** (an issue comment or a commit by him), which is an entry gate for
  M1 (L-3).
- **Licensing** (L-1, L-2): code under MIT. Short film quotations are listed in `NOTICE.md`, excluded from
  the MIT grant, with a non-affiliation disclaimer. ASCII art is original. "WarGames" is used only
  nominatively, in the README. No film stills.

---

## 12. Documentation

- **README.md** (M0 basic; completed in M5): what this is; install per OS (Release assets, `chmod +x`,
  Gatekeeper, SmartScreen, `go install …@latest`, `gh attestation verify`); quick start (`Joshua`); inside the
  shell; flags; themes; the games (with "coming in vX.Y" for `Planned` ones); troubleshooting (too small,
  colours/`NO_COLOR`, Windows/mintty, `-i`, `-s`); try a development build (Actions artifacts); build from
  source; credits and licence.
- **AGENTS.md** (M0, updated every milestone; `CLAUDE.md` points to it) (M-1): what the project is; commands;
  the import DAG and how `archtest` enforces it; the single clock; the `proto` protocol rules (dynamic input
  mode, `Think` captures by value, Esc belongs to the host); how to add a game (registry entry, package,
  testkit test, definition of done); goldens and `-update`; size budget and compatibility; hygiene and
  repository settings.
- **docs/screens.md**: the target-screen mockups.
- **This plan**: kept current at milestone boundaries; superseded sections are deleted, not annotated.

---

## 13. Milestones

| M | Deliverable | Release |
|---|---|---|
| 0 | Scaffold: go.mod, `.gitattributes`, LICENSE/NOTICE/SECURITY, version, `proto` (types only), registry with 16 `Info` entries (all `Planned`), `cli`, `theme`, `ui` hello-world, `archtest`, `.golangci.yml`, `prek.toml`, GoReleaser, `ci.yml`/`release.yml`/`scheduled.yml`/Dependabot, `scripts/`, README basic, **AGENTS.md**, repository settings. Exit: green CI showing six binaries smoke-run on native runners; `release.yml` snapshot dry run; Appendix A deleted. | — |
| 1 | Console and persona: clock, typewriter, line editor, `Sanitize`, scrollback, host + protocol (with a test-only stub game that emits every Output), persona (session, scenes, commands, intent, scripted brain, lines), connect/LOGON/greeting flows, themes, too-small, golden and fuzz tests, pty e2e test, 80×24 mockups for chess/Hearts/GTW. Gate: written permission for the brother's prompt. | — |
| 2 | Film set pieces: `games/ai` (iterative deepening + budget), `board/`, canvas renderer, Tic-Tac-Toe + ending/montage, Checkers, Chess, **GTW (self-contained)**. QA on all three OSes. | **v0.1.0** |
| 3 | Card games: `cards/` + `trick.go`, `prompt/`; Black Jack, Poker, Gin Rummy, Hearts, **Bridge (minimal, last)**. | v0.2.0 |
| 4 | Sims and maze: `sim` engine spec → engine → five scenarios; Falken's Maze. | v0.3.0 |
| 5 | Polish: film viewing pass for *reconstructed* text, README completion with screenshots, accessibility pass (`--instant`, `NO_COLOR`), `prek auto-update`, QA. | **v1.0.0** |
| 6 (opt) | LLM brain over `net/http`, opt-in, scripted fallback, size re-check. | v1.1.0 |

Effort is tracked per milestone against actuals, not per-game line estimates (P-3).

---

## 14. Verification (end-to-end, per release)

1. CI green: `lint`, `secrets`, `test` (three OSes), `build`, and `smoke` on every native runner. Six
   artifacts named `wopr_<os>_<arch>`; sizes in the step summary, all ≤ 15 MB (warning above 10 MB).
2. `wopr -v`, `-h`, `-g` and their long forms print and exit 0 without a terminal. `wopr --games | head -1`
   exits 0.
3. `wopr` → LOGON → wrong name ×3 → hint → `Joshua` → greeting → `list games` → `7` → chess board renders →
   WOPR "thinks" with the front panel animating and Ctrl+C still responsive → Esc returns to the shell with
   a remark.
4. `wopr gtw -i -s 1` and `wopr -p 15 -i -s 1` produce identical transcripts: side selection → targets → big
   board → DEFCON ladder → montage → `A STRANGE GAME…`. `wopr -p theaterwide` exits 2 listing both
   candidates. `wopr gtw chess` exits 2.
5. `go test -race ./...` green; goldens stable under `--seed 1 --instant`; `archtest` green.
6. The pty end-to-end test passes on Linux, macOS and Windows. Manual pass in Windows Terminal and conhost
   (colours, cursor, resize, Ctrl+C restore, SmartScreen).
7. On a tag: the release has six archives, `checksums.txt`, and attestations that verify with
   `gh attestation verify`.

---

## Appendix A — M0 seeds (delete when M0 lands)

### A.1 `prek.toml`

⟦TBD-prek-seed⟧

### A.2 `.golangci.yml` (excerpt)

⟦TBD-golangci-seed⟧

### A.3 `scripts/size-gate.sh`

⟦TBD-size-gate-seed⟧
