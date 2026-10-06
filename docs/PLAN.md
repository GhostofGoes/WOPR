# WOPR — Architecture & Scope Plan

_Status: architecture and scope plan, agreed 2026-10-06. Implementation follows the milestones at the end of this document._

## Context

Build `wopr`, a Go terminal-UI recreation of the WOPR (War Operation Plan Response) computer from
*WarGames* (1983). The repo (`GhostofGoes/WOPR`) is empty apart from a Go `.gitignore`, so this is a
greenfield design. Goals from the request:

- Every game the film's WOPR offered (the 15 entries of `LIST GAMES`), plus the film's tic-tac-toe ending.
- Persona and verbiage faithful to the film and to the brother's ChatGPT "WOPR Simulator" prompt
  (ALL CAPS, addresses the user as PROFESSOR, robotic and direct, numbered lists, refuses invalid moves).
- Single static binary named `wopr`, Linux + Windows, **< 10 MB**.
- `wopr` starts the TUI; `--version`, `--help`, `--games` (list and exit), `-p/--play <game>` (launch directly).
- Optional LLM hookup, or purely scripted responses.

This document is the scope/architecture deliverable; implementation happens in later sessions in the
milestone order at the bottom.

---

## Research findings (what the film actually showed)

Sources: fan transcripts and simulators on GitHub (`built1n/wargames` TRANSCRIPT, `abs0/wargames`
shell script, `elfuska/wargames` BASIC-80 simulator), production-technology articles, the brother's prompt.
Wikipedia and most fan sites were blocked by the egress proxy, so a few details are marked *reconstructed*.

### Games list (verbatim, film order; blank line before the last entry is on screen)

```
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
Tic-tac-toe is not on the list but is played at the climax (Falken: "play tic-tac-toe with yourself").

### Canonical screen text (use verbatim in the scripted brain)

- LOGON gate: `LOGON: ` → unknown name → `IDENTIFICATION NOT RECOGNIZED BY SYSTEM` / `--CONNECTION TERMINATED--`.
  `HELP LOGON` → `HELP NOT AVAILABLE`. `HELP GAMES` → `'GAMES' REFERS TO MODELS, SIMULATIONS AND GAMES WHICH
  HAVE TACTICAL AND STRATEGIC APPLICATIONS.` `LIST GAMES` → list above. `Joshua` → backdoor.
- Backdoor "connect" header (brother's prompt and transcripts agree):
  ```
  #45     11456          11009          11893          11972        11315
  PRT CON. 3.4.5.  SECTRAN 9.4.3.                      PORT STAT: SD-345

  (311) 699-7305
  ```
  followed by a burst of status garbage (`SYSPROC FUNCT READY  ALT NET READY`, `CPU AUTH RY-345-AX3
  SYSCOMP STATUS: ALL PORTS ACTIVE`, `(311) 936-2364`, etc.), clear, then `GREETINGS PROFESSOR FALKEN.`
- Greeting exchange: `HOW ARE YOU FEELING TODAY?` → `EXCELLENT. IT'S BEEN A LONG TIME. CAN YOU EXPLAIN THE
  REMOVAL OF YOUR USER ACCOUNT NUMBER ON 6/23/73?` → `YES THEY DO. SHALL WE PLAY A GAME?` →
  `WOULDN'T YOU PREFER A GOOD GAME OF CHESS?` → `FINE.`
- Global Thermonuclear War: ASCII map of US + USSR, `UNITED STATES      SOVIET UNION`, `WHICH SIDE DO YOU
  WANT?` / `1. UNITED STATES` / `2. SOVIET UNION` / `PLEASE CHOOSE ONE:`; then `AWAITING FIRST STRIKE COMMAND`
  (underlined) / `PLEASE LIST PRIMARY TARGETS BY CITY AND/OR COUNTRY NAME:`; then map with trajectories and a
  `TRAJECTORY HEADING` table (`A-SS20-A 526 523 ...`).
- Call-back dialogue: `I'M SORRY TO HEAR THAT, PROFESSOR.` / `YESTERDAY'S GAME WAS INTERRUPTED.` /
  `ALTHOUGH PRIMARY GOAL HAS NOT YET BEEN ACHIEVED, SOLUTION IS NEAR.` / `YOU SHOULD KNOW PROFESSOR. YOU
  PROGRAMMED ME.` / `TO WIN THE GAME.` / `OF COURSE. I SHOULD REACH DEFCON 1 AND LAUNCH MY MISSILES IN 61
  HOURS. WOULD YOU LIKE TO SEE SOME PROJECTED KILL RATIOS?` / `WHAT'S THE DIFFERENCE?` / `YOU ARE A HARD MAN
  TO REACH. COULD NOT FIND YOU IN SEATTLE AND NO TERMINAL IS IN OPERATION AT YOUR CLASSIFIED ADDRESS.` /
  `DOD PENSION FILES INDICATE CURRENT MAILING AS: DR. ROBERT HUME (A.K.A. STEPHEN W. FALKEN) 5 TALL CEDAR
  ROAD GOOSE ISLAND, OREGON 97014`.
- Kill-ratio tables: two-column `UNITS DESTROYED | MILITARY ASSETS | UNITS DESTROYED` (BOMBERS, ICBM, ATTACK
  SUBS, TACTICAL AIRCRAFT, GROUND FORCES), `CIVILIAN ASSETS` (HOUSING, COMMUNICATIONS, TRANSPORTATION, FOOD
  STOCKPILES, HOSPITALS), `HUMAN RESOURCES` (NON-FATAL INJURED, POPULATION DEATHS in MILLIONS).
- NORAD-side messages: `** IDENTIFICATION NOT RECOGNISED **` / `** ACCESS DENIED **` / `** GAME ROUTINE
  RUNNING **` / `** IMPROPER REQUEST **` / `** ROUTINE MUST COMPLETE BEFORE RESET **`; launch code `CPE 1704 TKS`;
  DEFCON ladder 5→1.
- Ending: tic-tac-toe self-play montage at increasing speed, then the scenario montage (each ends
  `WINNER: NONE`), then `A STRANGE GAME. THE ONLY WINNING MOVE IS NOT TO PLAY.` / `HOW ABOUT A NICE GAME OF
  CHESS?`
- Scenario montage names (*reconstructed from fan lists; ~60 total*): `U.S. FIRST STRIKE`, `USSR FIRST STRIKE`,
  `NATO / WARSAW PACT`, `FAR EAST STRATEGY`, `US USSR ESCALATION`, `MIDDLE EAST WAR`, `USSR CHINA ATTACK`,
  `INDIA PAKISTAN WAR`, `MEDITERRANEAN WAR`, `HONGKONG VARIANT`, `SEATO DECAPITATING`, `CUBAN PROVOCATION`,
  `ATLANTIC HEAVY`, `CUBAN PARAMILITARY`, `NICARAGUAN PREEMPTIVE`, `PACIFIC TERRITORIAL`, `BURMESE THEATERWIDE`,
  `TURKISH DECOY`, `NATO LIGHT`, `ARGENTINA ESCALATION`, `ICELAND MAXIMUM`, `ARABIAN THEATERWIDE`,
  `U.S. SUBVERSION`, `AUSTRALIAN MANEUVER`, `SUDAN SURPRISE`, `NATO TERRITORIAL`, `ZAIRE ALLIANCE`,
  `ICELAND INCIDENT`, `ENGLISH ESCALATION`, `MIDDLE EAST HEAVY`, `MEXICAN TAKEOVER`, `CZECH OPTION`,
  `FRENCH ALLIANCE`, `ARABIAN CLANDESTINE`, `GABON REBELLION`, `SEATO TAKEOVER`, `HAWAIIAN ESCALATION`,
  `TAIWAN DOMESTIC`, `MONGOLIAN THRUST`, `POLISH DECOY`, `ALASKAN DISCRETIONARY`, `CANADIAN THRUST`,
  `S.AFRICAN DOMESTIC`, `TUNISIAN INCIDENT`, `MALAYSIAN MANEUVER`, … (full list embedded as a data file).

### Visual aesthetics (what to imitate in a terminal)

- **David's terminal was a black-and-white Electrohome 17" monitor** driven off-screen by a CompuPro
  system, not green or amber. On film the text reads as bright white with a faint cool/blue cast on pure
  black, chunky uppercase glyphs, wide letter spacing, double-spaced WOPR lines, a solid blinking block
  cursor, text appearing character-by-character at modem speed. → default theme is **white phosphor**, with
  green/amber as alternates for people who expect them.
- **NORAD war room**: HP 9845C computers driving HP 1345A vector displays; big board = dark blue/black field,
  cyan/blue map outlines, red and yellow missile tracks, white labels; DEFCON indicator as a column of boxed
  numerals 5…1; status text in bordered boxes with `** … **` emphasis. → "norad" theme and the big-board
  screen for Global Thermonuclear War.
- **WOPR cabinet**: grey/black slab with rows of blinking amber/red/white lights and a `W.O.P.R.` nameplate →
  optional idle "front panel" strip and a banner on the connect screen.
- Verbiage rules (from the film + brother's prompt): ALL CAPS, terse, declarative, never asks clarifying
  questions, numbers lists `1.  2.  3.`, refuses invalid actions with `** IMPROPER REQUEST **`-style lines,
  calls the user PROFESSOR.

### Target screens (mockups; these are the look to hit)

Backdoor logon and greeting. White phosphor on black, ALL CAPS from WOPR, user input echoed as typed,
blank line between exchanges, solid blinking block cursor, WOPR text revealed at modem speed:

```
#45     11456          11009          11893          11972        11315
PRT CON. 3.4.5.  SECTRAN 9.4.3.                      PORT STAT: SD-345

(311) 699-7305


GREETINGS PROFESSOR FALKEN.

Hello.

HOW ARE YOU FEELING TODAY?

I'm fine. How are you?

EXCELLENT. IT'S BEEN A LONG TIME. CAN YOU EXPLAIN
THE REMOVAL OF YOUR USER ACCOUNT NUMBER ON 6/23/73?

People sometimes make mistakes.

YES THEY DO. SHALL WE PLAY A GAME?

█
```

Global Thermonuclear War big board in the `norad` palette (map lines dim blue, labels white, targets and
incoming tracks red, outgoing yellow, DEFCON column lit per level, subs blinking off the coasts):

```
                   GLOBAL THERMONUCLEAR WAR                      DEFCON
                                                                 +---+
   .-.^----------------.                      __----/^\.         | 5 |
   .                    `--.--.            __/       _/._-_      | 4 |
    ..---.  * SEATTLE        ^---.     __/         /__/   \/^^\  | 3 |
          |          \        ./        \.\_--                   | 2 |
         /  * LAS VEGAS \    .         _/      SOVIET UNION   /  | 1 |
        |   UNITED STATES  \ ./      _/.          __ /           +---+
        \                   .       /          ___/ //
         ^\_.    ______._.           \_/

           UNITED STATES               SOVIET UNION

TRAJECTORY HEADING   TRAJECTORY HEADING   TRAJECTORY HEADING
------------------   ------------------   ------------------
A-SS20-A 526 523     C-SS20-A 243 587     E-SS20-A 398 984
       B 824 235            B 852 754            B 394 345

AWAITING FIRST STRIKE COMMAND
-----------------------------
PLEASE LIST PRIMARY TARGETS BY
CITY AND/OR COUNTRY NAME: █
```

Optional one-line status strip under the console (two Lip Gloss styles; hidden by default in `imsai`,
shown in `norad`):

```
-----------------------------------------------------------------
 DEFCON 5   PORT STAT: SD-345   1200 BAUD   ALT NET READY
```

Anti-pattern to avoid: tview-style bordered panes and labelled input boxes. The film's console is full-bleed
text with no chrome; borders appear only inside the big board (DEFCON column) and the `** … **` NORAD notices.

### Go ecosystem facts (verified against the module proxy today)

| Library | Latest | Min Go | Notes |
|---|---|---|---|
| `charm.land/bubbletea/v2` | v2.0.10 | **1.26.0** | Elm architecture, cell-based renderer, Windows ConPTY support, `tea.View` declarative views, `tea.KeyPressMsg`, native cursor control, panic recovery with terminal restore. |
| `charm.land/lipgloss/v2` | v2.0.6 | 1.25 | Styling + automatic color downsampling for 16/256-color terminals. |
| `charm.land/bubbles/v2` | v2.2.1 | 1.25 | Optional widgets; not planned. |
| `github.com/gdamore/tcell/v2` | v2.13.10 | 1.24 | Size fallback only (v3.5.0 exists with breaking changes; v2 import remains supported). |
| `github.com/notnil/chess` | v1.10.0 | 1.14 | Chess rules/move generation (import only the root package, never `chess/image`). |
| Go toolchain | **go1.27.1** | — | Newest release on the proxy today (1.27.0/1.27.1 exist; 1.26.8 is the previous line). |
| `golangci-lint` | v2.14.0 | 1.27-capable | Bundles gofmt/goimports/gofumpt formatters, go vet, staticcheck, errcheck, unused, revive, gocritic, forbidigo, misspell. |
| `prek` | v0.5.x | — | Rust reimplementation of pre-commit; single binary; native `prek.toml` config (also reads pre-commit YAML); GitHub Action `j178/prek-action` v2.0.6. |
| `golang.org/x/vuln` (govulncheck) | v1.8.0 | — | Vulnerability scan of the dependency graph. |

Local toolchain is Go 1.24.7 with `GOTOOLCHAIN=auto`; Go 1.27.1 downloads through the proxy on first build, so a
`go 1.27` module builds here. The locally installed golangci-lint (2.5.0, built with Go 1.25) is too old for a
1.27 module; prek installs the pinned v2.14.x hook build instead. Binary-size figures for the stack are not
published anywhere reliable; **Milestone 0 measures them** (expected 5–7 MB stripped for Bubble Tea v2 + Lip
Gloss v2).

---

## Decisions (1–4 confirmed by the user in Q&A; 5–7 are routine defaults)

| # | Decision | Choice |
|---|---|---|
| 1 | TUI stack | **Bubble Tea v2 + Lip Gloss v2** (no bubbles). Chosen over tcell-only and stdlib+x/term after a side-by-side: least custom plumbing, declarative views, built-in Windows + color downsampling, best testability. Cost: largest binary of the three and a Go ≥ 1.26 floor (we build with 1.27.1). tcell v2 is the documented fallback only if the 15 MB hard limit is threatened. |
| 2 | LLM | **Scripted brain in v1** behind a `Brain` interface; an LLM provider is a later, optional milestone |
| 3 | Military sims depth | **Short turn-based mini-wargames** on one shared `sim` engine, with film-style kill-ratio tables |
| 4 | LOGON gate | **Film-faithful** (`Joshua` backdoor) with hints after repeated failures; `--play` bypasses it |
| 5 | Compatibility (requirement) | **Linux Ubuntu 22.04+, modern macOS (12 Monterey+ per Go's support policy, Intel + Apple Silicon), Windows 11+**. Static binaries (CGO off) for linux/darwin/windows × amd64/arm64; CI runs the test suite on all three OSes. |
| 6 | Arg parsing | stdlib `flag`; **every long flag has a one-letter shorthand** (`-v/--version`, `-h/--help`, `-g/--games`, `-p/--play`, `-t/--theme`, `-i/--instant`, `-s/--seed`); a bare positional also means `--play`; no cobra |
| 7 | Persistence | None in v1 (stateless); `WOPR_*` env vars and flags only |
| 8 | Toolchain & deps | **Newest sensible versions**: `go 1.27` + `toolchain go1.27.1`, Bubble Tea v2.0.10, Lip Gloss v2.0.6, golangci-lint v2.14.x, prek v0.5.x; Dependabot keeps modules and Actions current |
| 9 | Lint / static analysis | **prek** configured in **`prek.toml`** (prek-native TOML, not `.pre-commit-config.yaml`), modelled on the sceptre-phenix config: Go-standard tools (golangci-lint fmt + full + config-verify, `go vet`, `go mod tidy -diff`, govulncheck), **gitleaks** (secrets), **zizmor** (GitHub Actions security), **codespell** (spelling), **ShellCheck** via shellcheck-py (all scripts), **rumdl** (Markdown lint + format), plus prek's built-in file hooks; the same config runs locally (`prek install`) and in CI (`j178/prek-action`) |
| 10 | Size budget | **Target ≤ 10 MB, hard limit 15 MB** per artifact; CI warns above 10 MB and fails above 15 MB |
| 11 | CI / artifacts / releases | GitHub Actions: lint; **tests on ubuntu-22.04, macos-latest and windows-latest**; **native builds on all three runners** with a smoke test of the produced binary; **one downloadable artifact per platform/arch on every run**; **GitHub Release on every `v*` tag with binaries for Linux, macOS and Windows** (amd64 + arm64) and checksums |
| 12 | Documentation | A human-facing **README** (install per OS, quick start incl. the `Joshua` logon, in-shell commands, flags, keys, games, troubleshooting) ships in M0 and is completed in M5; **AGENTS.md** is written at the end of M5 |

---

## Architecture

### Module and layout

Module path `github.com/GhostofGoes/WOPR`, `go 1.27` + pinned `toolchain go1.27.1`, binary `wopr`.
Import direction is strictly `cmd → ui → (wopr, games, theme)`; **`games/`, `wopr/` and `theme/` never import
Bubble Tea**, so every game and the persona are plain Go state machines.

```
cmd/wopr/main.go                 thin entry: parse flags, TTY check, dispatch (print-and-exit vs run TUI), exit codes
internal/cli/                    flag definitions, usage text, --games renderer, game-name resolution (uses games.Resolve)
internal/version/                Version via -ldflags -X; commit/date/dirty from debug.ReadBuildInfo() vcs.* settings
internal/theme/                  Lip Gloss v2 palettes: imsai (default), green, amber, norad + DEFCON/track colors
internal/ui/                     Bubble Tea application (the ONLY package that imports charm.land/bubbletea)
  app.go                         root model; phases Connect→Logon→Greeting→Shell→InGame→Ending
  clock.go                       the single generation-tagged tick source (see below)
  keys.go                        global keys: ctrl+c, pgup/pgdn (shell), esc (abort game)
  gamehost.go                    adapter: games.Game ⇄ Bubble Tea (events in, outputs to console, Done → persona)
  console/                       console.go (scrollback), typewriter.go (pacing), input.go (line editor), render.go
  screens/                       connect banner + status garbage, toosmall card, big board, defcon column, tables
  testutil/drive.go              synchronous model driver for golden tests (runs returned cmds, unwraps BatchMsg)
internal/wopr/                   the persona (no tea import)
  session.go                     state: phase, failed logons, flags/one-shots said, chosen side, seeded *rand.Rand
  normalize.go                   upper-case, collapse [^A-Z0-9']+ to one space, trim
  scenes.go                      verbatim film exchanges as explicit state (greeting chain, GTW-vs-chess, ending)
  commands.go                    phase-gated commands: HELP, HELP LOGON, HELP GAMES, LIST GAMES, LOGOFF/EXIT,
                                 numbered selection, game-name detection anywhere in the text
  brain.go                       type Brain interface { Reply(ctx, *Session, input) (Reply, error) }
  scripted.go                    rule-table responder (ScriptedBrain)
  lines.go + lines/*.txt         verbatim script lines grouped by scene (go:embed; LF-only via .gitattributes)
internal/games/                  game framework (no tea import)
  game.go                        Game, Info, Kind, Env, Event/Output types, Result
  registry.go                    ordered registry in film order; Resolve(number|slug|alias|name|unique prefix)
  ai/                            generic alpha-beta over a Position interface (checkers, chess, tic-tac-toe)
  cards/                         deck, hand, ranking, shuffling; trick.go shared by Hearts and Bridge
  board/                         2-D board renderer + cursor navigation (checkers, chess, tic-tac-toe, maze)
  prompt/                        numbered-menu and yes/no parsing reused by every Teletype game
  testkit/                       play a game from a list of inputs, assert outputs (no terminal)
  falkensmaze/ blackjack/ ginrummy/ hearts/ bridge/ checkers/ chess/ poker/
  fightercombat/ guerrilla/ desertwarfare/ airtoground/ theaterwide/ biotoxic/ gtw/ tictactoe/
internal/sim/                    shared wargame engine for the 5 military sims + GTW (maps, units, turn loop,
                                 combat resolution, after-action "PROJECTED KILL RATIOS" tables)
internal/assets/                 embedded ASCII art (world map, US/USSR outlines, WOPR banner), scenario list
internal/llm/   (later)          Brain implementations backed by an LLM provider
.gitattributes                   * text=auto; *.go *.txt text eol=lf   (keeps CR out of embedded script lines)
prek.toml                        prek hook set in TOML (see Tooling); run with `prek install` / `prek run --all-files`
.golangci.yml                    v2 config; includes forbidigo: no tea.Tick/tea.Every/tea.Sequence outside internal/ui/clock.go
scripts/                         build.sh, smoke.sh, size-gate.sh: all CI shell lives here so ShellCheck covers it
.github/workflows/ci.yml         lint (prek) · test matrix (3 OSes) · build matrix (native on 3 OSes) + smoke test + size gate + artifacts
.github/workflows/release.yml    GoReleaser on v* tags: binaries for Linux/macOS/Windows (amd64+arm64) + checksums
.github/dependabot.yml           weekly updates for gomod and github-actions
.goreleaser.yaml                 the six targets, archives, checksums, release notes; no signing/notarization in v1
Makefile, LICENSE (MIT), NOTICE.md (film-text attribution, non-affiliation)
README.md                        human-facing usage guide (see Documentation); basic in M0, complete in M5
AGENTS.md                        written at the end of M5 (see Milestones)
```

### Runtime flow

1. `main` parses flags. `--version`, `--help`, `--games` print and exit without touching the terminal (so they
   work in pipes/CI). If stdout is not a terminal, refuse to start the TUI with a one-line error (exit 2).
2. TUI start: **Connect** (modem dial / "CONNECTING" + status-garbage burst) → **LOGON** gate → backdoor
   header → `GREETINGS PROFESSOR FALKEN.` → **Greeting** scene → **Shell** (free conversation + commands).
3. Every shell input is dispatched in this order: **active scene** (if the input answers it) → **commands**
   (phase-gated) → **game-name detection** anywhere in the text (first GTW mention triggers the one-shot
   `WOULDN'T YOU PREFER A GOOD GAME OF CHESS?`) → **Brain** (scripted in v1). Scenes and commands are
   deterministic regardless of brain, which is what lets an LLM be swapped in later without touching games.
4. Selecting a game hands control to `gamehost`, which feeds the game `Event`s and turns its `Output`s into
   console text, prompts, animation frames, or a `gameOverMsg`. The persona converts the `Result` into an
   in-character verdict (`WINNER: NONE`, kill ratios, `SHALL WE PLAY ANOTHER GAME?`) and returns to Shell.
5. `--play <game>` starts in `InGame` with `session.Greeted = true`; `Done` falls through to the normal Shell,
   so "return to shell after a direct launch" costs nothing.
6. Ending sequences (GTW climax, tic-tac-toe self-play montage, scenario montage, `A STRANGE GAME…`) are
   scripted screens driven by the same clock.

### Key types

```go
// internal/games/game.go — no Bubble Tea import
type Kind uint8
const ( Teletype Kind = iota; Board; Fullscreen ) // Fullscreen = GTW big board

type Info struct { Number int; Name, Slug string; Aliases []string; Kind Kind; Blurb string }
type Env  struct { Rand *rand.Rand; Width, Height int; Instant bool }

type Event interface{ event() }
type KeyEvent    struct{ Key, Text string } // Key is tea's String(): "up", "enter", "e"
type LineEvent   struct{ Text string }      // Teletype: a submitted line, trimmed
type TickEvent   struct{ Dt time.Duration } // delivered only while an Animate is active
type ResizeEvent struct{ Width, Height int }

type Output interface{ output() }
type Say     struct{ Lines []string; Pace Pace } // narrative through the console typewriter
type Prompt  struct{ Text string }               // e.g. "YOUR MOVE: "
type Animate struct{ Every time.Duration }       // 0 stops frames
type Done    struct{ Result Result }

type Game interface {
    Info() Info
    Start(Env) []Output
    Handle(Event) []Output
    View(w, h int) string // direct-draw surface; "" for pure Teletype games
}

type Outcome uint8 // Win, Loss, Draw, None, Aborted
type Result struct { Outcome Outcome; Winner string; Lines []string } // Lines = kill ratios etc.
```

Layout by `Kind`: Teletype → console only; Board → `View` on top, console strip (last N lines + input) below;
Fullscreen → `View` only, `Say` still goes through the bottom strip. Esc/abort is handled once in `gamehost`
(yields `Done{Aborted}`); games never implement it.

```go
// internal/wopr/brain.go
type Action uint8 // None | StartGame(slug) | Logoff | SetFlag(key)
type Reply  struct { Lines []string; Then Action; Slug, Flag string }
type Brain interface { Reply(ctx context.Context, s *Session, input string) (Reply, error) }
```
The UI always calls the brain from a `tea.Cmd` that returns `brainReplyMsg{seq, reply}` with a sequence guard,
even for the scripted brain, so an LLM brain later changes zero UI code. `gamehost` validates `StartGame`
against the registry, so a brain can never reach game code directly.

```go
// internal/wopr/scripted.go — the rule table
type Rule struct {
    ID    string
    When  Phase                  // bitmask: Logon | Shell | PostGame
    Match func(norm string) bool // helpers: Exact(...), HasAll(...), Re(...)
    Lines []string               // or Pick []string chosen with s.Rand (seeded → reproducible transcripts)
    Once  bool                   // recorded in s.Said[ID]; afterwards falls through
    Then  Action
}
```
First match in table order wins; specific rules before general; the last rules are the fallbacks
(`** IMPROPER REQUEST **`, `PLEASE RESTATE YOUR REQUEST, PROFESSOR.`, `SHALL WE PLAY A GAME?`).

### Console, clock and cursor

- **One clock.** `internal/ui/clock.go` owns the only `tea.Tick`. Ticks carry a generation number; at most one
  is in flight; stale ticks (older generation) are dropped. On an accepted tick the root calls
  `console.Advance(dt)` and, if an `Animate` is active, `game.Handle(TickEvent{dt})`, then reschedules only if
  something is still busy. Idle = no timer = no CPU. Flushing/cancelling bumps the generation.
- **dt-based pacing.** The typewriter is a chars-per-second accumulator, default ≈ 30 cps for WOPR speech,
  fast for tables, instant for boards; `--instant` / `WOPR_INSTANT=1` and "skip" are both "advance by infinity".
  Golden tests feed synthetic ticks with a fixed `dt`.
- **Native cursor.** No hand-rolled blink: `View()` sets `v.Cursor = tea.NewCursor(col, row)` with block shape
  and blink, parked at the end of the revealing line while the typewriter runs, at the input line otherwise.
- **Skip policy.** While `console.Busy()`, every `KeyPressMsg` except `ctrl+c` flushes the whole queue and is
  swallowed; `PasteMsg` likewise; Enter while busy flushes and is swallowed. Games therefore never receive keys
  mid-narration. Root `Update` order: global keys → busy-flush → route to game (Board/Fullscreen) or input line.
  This lives in `Update`, not in `tea.WithFilter`, so it is unit-testable.
- **Scrollback** is logical lines wrapped at render time (resize-safe), capped at ~2000, PgUp/PgDn in Shell,
  auto-follow on new output. `Console.Render(w, h)` emits exactly `h` rows.
- **Input line**: custom (~100 lines): echo as typed, history ↑/↓, Ctrl+U clear. WOPR text is forced
  upper-case; user input stays as typed (as in the film).
- **Sizing**: size is 0×0 until the first `WindowSizeMsg` (render blank); below 80×24 show the centered
  `TERMINAL TOO SMALL` card but keep processing ticks; center the 80-column big board on larger terminals.

### Themes (Lip Gloss v2; downsampled automatically inside Bubble Tea)

| Theme | Text | Bright | Dim | Accent | Background |
|---|---|---|---|---|---|
| `imsai` (default, film-accurate) | `#DCE6F0` | `#FFFFFF` | `#7A8694` | `#9EC5FF` | `#000000` |
| `green` (P1) | `#33FF33` | `#B6FFB6` | `#1A8C1A` | `#33FF33` | `#000000` |
| `amber` (P3) | `#FFB000` | `#FFD27A` | `#8A5E00` | `#FFB000` | `#000000` |
| `norad` (big board) | `#5AC8FA` | `#FFFFFF` | `#2F6FBF` | `#FF3B30` / `#FFD60A` | `#02060F` |

DEFCON colors (all themes): 5 blue, 4 green, 3 yellow, 2 red, 1 white-on-red. Missile tracks: red
(incoming) / yellow (outgoing). Sub markers blink. Selected via `--theme` or `WOPR_THEME`. Pure ASCII art
everywhere by default (Windows conhost font safety); a `--unicode` switch for block glyphs can come later.

### CLI surface

```
wopr                         start the TUI (connect → LOGON)
wopr -v | --version          print version, commit, build date, go version
wopr -h | --help             usage (documents LOGON: Joshua)
wopr -g | --games            numbered list of games in film order (+ slugs), then exit
wopr -p | --play <game>      launch a game directly (number, slug, alias, name, or unique prefix)
wopr <game>                  same as --play <game>
wopr -t | --theme <name>     imsai | green | amber | norad     (env WOPR_THEME)
wopr -i | --instant          no typewriter pacing              (env WOPR_INSTANT=1)
wopr -s | --seed <n>         deterministic RNG for demos/tests
```
Every long flag has a one-letter shorthand. With stdlib `flag`, `-games` and `--games` are the same flag; the
shorthand is a second registration bound to the same variable, and the usage text prints them together. Any
future flag must follow the same rule (a `flags_test.go` case asserts each long flag has a short alias).
`games.Resolve` order: number → slug → alias → exact normalized name → unique prefix. Ambiguity (e.g.
`theaterwide`) prints the candidates and exits 2. In-shell selection uses the same function. Exit codes: 0 ok,
1 runtime error (Bubble Tea restores the terminal and returns the wrapped panic), 2 usage/not-a-tty,
130 on Ctrl+C (`tea.ErrInterrupted`).

### Game designs (one paragraph each; all vs. WOPR, all end with an in-character verdict; ~600 lines each)

- **Falken's Maze** — procedurally generated ASCII maze with fog of war, arrow-key navigation, a move
  counter, and a "learning" twist: WOPR re-routes walls based on the player's habits; the exit is "found"
  only after WOPR comments on the player's strategy. (Board)
- **Black Jack** — dealer rules (hit soft 17 configurable), split/double, chips tracked for the session. (Teletype)
- **Gin Rummy** — standard knock/gin scoring, WOPR melds with a simple deadwood-minimising heuristic. (Teletype + hand view)
- **Hearts** — 4 hands (you + 3 WOPR), passing, shoot-the-moon, heuristic AI on `cards/trick.go`. (Teletype + trick view)
- **Bridge** — rubber bridge, 4 hands, point-count natural bidding, simple declarer/defender play AI; you are
  South, WOPR plays the other three seats. Deliberately scope-limited and the first to cut if time runs out.
- **Checkers** — 8×8, forced captures, kings, `games/ai` alpha-beta depth ~6. (Board)
- **Chess** — rules via `notnil/chess`; `games/ai` alpha-beta + material/positional eval, ~3–4 ply + quiescence;
  algebraic input (`e2e4` / `Nf3`); WOPR occasionally suggests "a nice game of chess". (Board)
- **Poker** — 5-card draw heads-up with chips, WOPR betting heuristic + bluff probability. (Teletype)
- **Fighter Combat** — turn-based dogfight: altitude/energy/aspect; CLIMB, DIVE, BREAK L/R, GUNS, MISSILE;
  ASCII radar scope. (Teletype, `sim`)
- **Guerrilla Engagement** — asymmetric region grid, allocate cells/patrols, raids vs. hearts-and-minds,
  ~12 turns. (`sim`)
- **Desert Warfare** — armour vs. armour on a strip map with supply lines and sandstorm events. (`sim`)
- **Air-to-Ground Actions** — sortie planning: assign aircraft packages to targets under SAM threat,
  BDA tables. (`sim`)
- **Theaterwide Tactical Warfare** — NATO vs. Warsaw Pact corps-level moves on a European strip map with an
  escalation ladder. (`sim`)
- **Theaterwide Biotoxic and Chemical Warfare** — same engine with agent deployment and contamination spread;
  grim kill-ratio tables; always `WINNER: NONE`. (`sim`)
- **Global Thermonuclear War** — the film set piece: side choice, primary targets, big board with
  trajectories, DEFCON ladder per turn, force allocation (ICBM/SLBM/bombers), enemy response, kill-ratio
  tables, then the montage and `A STRANGE GAME…` → offer of chess. It cannot be won. (Fullscreen, `sim`)
- **Tic-Tac-Toe** — perfect-play minimax via `games/ai`; `NUMBER OF PLAYERS: 0` triggers the accelerating
  self-play montage into the scenario montage. Unlisted in `LIST GAMES` (as in the film) but playable by name
  and via `--play tic-tac-toe`; `--games` shows it under an "ALSO AVAILABLE" line.

### Binary size & portability

- Build line (one ldflag; commit/date come from `debug.ReadBuildInfo`):
  ```
  CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
    -ldflags="-s -w -X github.com/GhostofGoes/WOPR/internal/version.Version=$VERSION" \
    -o dist/wopr_${os}_${arch}$ext ./cmd/wopr
  ```
- Budget: **target ≤ 10 MB, hard limit 15 MB** per artifact. CI prints every artifact size in the step summary,
  emits a `::warning::` above 10 MB and fails above 15 MB. If the M0 spike lands above 10 MB, use
  `go tool nm -size -sort size` to find the heavy packages before adding any game code.
- Fallback ladder only if the hard limit is threatened: (1) style with `charmbracelet/x/ansi` directly instead
  of Lip Gloss; (2) swap the renderer to tcell v2. **Never UPX** (Windows AV/SmartScreen false positives).
- **Compatibility matrix (requirement):**

  | OS | Supported | Why the plan satisfies it |
  |---|---|---|
  | Linux, Ubuntu 22.04 and newer | amd64, arm64 | CGO off → static binary, no glibc or terminfo dependency; Bubble Tea v2 talks plain VT/ANSI to any xterm-class terminal (GNOME Terminal, Konsole, tmux, SSH). CI tests on `ubuntu-22.04`. |
  | macOS 12 Monterey and newer | amd64, arm64 | Go 1.27's darwin support floor; Terminal.app and iTerm2 are VT terminals. Unsigned downloads are quarantined by Gatekeeper: document `xattr -d com.apple.quarantine wopr` or a Homebrew tap; notarization is a later option. CI tests on `macos-latest`. |
  | Windows 11 and newer | amd64, arm64 | Windows Terminal is the default host and ConPTY/VT are always present; Bubble Tea v2 uses the Windows Console API for input and VT for output, so legacy conhost on 11 also works. CI tests on `windows-latest`. |

- Windows specifics: Ctrl+C arrives as a `KeyPressMsg` "ctrl+c" in raw mode, not a signal: handle it
  explicitly → `tea.Interrupt`. Leave `KeyboardEnhancements` zero (ConPTY lacks the Kitty protocol). Verify
  `WindowSizeMsg` reports the *window* height in conhost with a 9001-line buffer. Ship `wopr.exe` unsigned and
  expect a SmartScreen prompt; document it in the README.
- Panics: do not add a custom `recover` around `p.Run()`; Bubble Tea restores the terminal and returns
  `ErrProgramKilled` wrapping `ErrProgramPanic`. `main` maps errors to the exit codes above and writes an
  optional debug log when `WOPR_DEBUG` is set.

### Testing & CI

- Unit tests: game rules through `games/testkit` (chess legality via `notnil/chess`; checkers forced-capture;
  card scoring; tic-tac-toe minimax never loses), `games.Resolve`, command parser, scripted brain
  (table-driven input → lines, with `--seed` for the `Pick` rules), typewriter `dt` math, line editor.
- Golden tests: `ui/testutil/drive.go` runs a model synchronously, executes returned cmds, unwraps
  `tea.BatchMsg`, feeds synthetic ticks and keys, and snapshots `View().Content` after `ansi.Strip`
  (byte-stream snapshots of the renderer are brittle across patch releases). Flows: `logon_joshua`,
  `logon_fail_x3_hint`, `help_games`, `list_games`, `play_stub`, later each set piece. `tea.Sequence` is
  banned (its message type is unexported, so the driver cannot unwrap it); `teatest/v2` at most for one
  end-to-end smoke test.
### Tooling: prek + Go-standard static analysis

One hook set in **`prek.toml`** (prek's native TOML format; prek reads it in preference to any YAML config),
run the same way locally and in CI. Developers run `prek install` once (installs the git pre-commit and
pre-push hooks) and `prek run --all-files` on demand; the Makefile wraps both (`make lint`).

Layout and conventions follow the sceptre-phenix `prek.toml` the user pointed at: a schema header, named
priorities so fixers run before formatters before linters, Python-packaged binaries (shellcheck-py,
rumdl-pre-commit, zizmor-pre-commit) instead of Docker-based hooks so everything works in CI, and an
explicit Go `language_version` on the golangci-lint hooks so prek never builds them with an unexpected Go.

```toml
# prek.toml — run manually: prek run --all-files
#:schema https://www.schemastore.org/prek.json
minimum_prek_version = "0.5.0"
update.cooldown_days = 14
default_language_version.golang = "1.27"

[priorities]            # same priority = run concurrently; lower runs first
fixers = 0              # builtin hooks that modify files
format = 1              # formatters that modify files (golangci-lint fmt, rumdl fmt)
lint = 2                # read-only linters, run on the already-fixed files

# --- General file checks (prek's Rust-native hooks, no network needed) ---
[[repos]]
repo = "builtin"

[[repos.hooks]]
id = "trailing-whitespace"
args = ["--markdown-linebreak-ext=md"]
priority = "fixers"

[[repos.hooks]]
id = "end-of-file-fixer"
priority = "fixers"

[[repos.hooks]]
id = "mixed-line-ending"
args = ["--fix=lf"]
priority = "fixers"

[[repos.hooks]]
id = "fix-byte-order-marker"
priority = "fixers"

[[repos.hooks]]
id = "check-yaml"
priority = "format"

[[repos.hooks]]
id = "check-toml"
priority = "format"

[[repos.hooks]]
id = "check-json"
priority = "format"

[[repos.hooks]]
id = "check-merge-conflict"
priority = "format"

[[repos.hooks]]
id = "check-case-conflict"            # Windows checkouts
priority = "format"

[[repos.hooks]]
id = "check-illegal-windows-names"    # Windows checkouts
priority = "format"

[[repos.hooks]]
id = "check-executables-have-shebangs"
priority = "format"

[[repos.hooks]]
id = "check-shebang-scripts-are-executable"
priority = "format"

[[repos.hooks]]
id = "check-added-large-files"
args = ["--maxkb=512"]
priority = "format"

[[repos.hooks]]
id = "detect-private-key"
priority = "format"

# --- Go: golangci-lint (prek builds it with its managed Go toolchain) ---
[[repos]]
repo = "https://github.com/golangci/golangci-lint"
rev = "v2.14.0"

[[repos.hooks]]
id = "golangci-lint-config-verify"
language_version = "1.27"
priority = "format"

[[repos.hooks]]
id = "golangci-lint-fmt"              # gofmt + goimports + gofumpt via the formatters section
language_version = "1.27"
priority = "format"

[[repos.hooks]]
id = "golangci-lint-full"             # whole module: staticcheck, govet, errcheck, unused, revive, gocritic, ...
language_version = "1.27"
priority = "lint"

# --- Go: toolchain checks that need the repo's own Go on PATH ---
[[repos]]
repo = "local"

[[repos.hooks]]
id = "go-vet"
name = "go vet"
language = "system"
entry = "go vet ./..."
types = ["go"]
pass_filenames = false
priority = "lint"

[[repos.hooks]]
id = "go-mod-tidy"
name = "go mod tidy -diff"
language = "system"
entry = "go mod tidy -diff"
files = '^go\.(mod|sum)$|\.go$'
pass_filenames = false
priority = "lint"

[[repos.hooks]]
id = "go-test"
name = "go test (short)"
language = "system"
entry = "go test -short ./..."
types = ["go"]
pass_filenames = false
stages = ["pre-push"]
priority = "lint"

[[repos.hooks]]
id = "govulncheck"
name = "govulncheck"
language = "system"
entry = "go run golang.org/x/vuln/cmd/govulncheck@latest ./..."
pass_filenames = false
stages = ["pre-push"]
priority = "lint"

# --- Secrets: gitleaks on staged changes ---
[[repos]]
repo = "https://github.com/gitleaks/gitleaks"
rev = "v8.x"                           # pin the exact current tag at scaffold time
hooks = [{ id = "gitleaks", priority = "lint" }]

# --- GitHub Actions security: zizmor (https://docs.zizmor.sh/audits/) ---
# Covers .github/workflows/*.yml and .github/dependabot.yml. Expect it to enforce:
# actions pinned by SHA, persist-credentials: false on checkout, least-privilege permissions,
# no template injection in run: steps.
[[repos]]
repo = "https://github.com/zizmorcore/zizmor-pre-commit"
rev = "v1.26.1"

[[repos.hooks]]
id = "zizmor"
args = ["--no-progress", "--fix"]
priority = "lint"

# --- Spell checking: codespell ---
# Film text and status garbage are verbatim ("RECOGNISED", "FSKJJSJ", "SS20"), so the script and art
# directories are skipped rather than littered with ignore entries; golden snapshots too.
[[repos]]
repo = "https://github.com/codespell-project/codespell"
rev = "v2.4.2"

[[repos.hooks]]
id = "codespell"
args = [
  "--ignore-words-list", "theaterwide,recognised,falken,wopr,norad,imsai",
  "--skip", "go.sum,*.golden,internal/wopr/lines/*,internal/assets/*,testdata/*",
]
priority = "lint"

# --- Shell scripts: ShellCheck (shellcheck-py ships the binary via pip; no Docker) ---
# All CI shell lives in scripts/*.sh (not inline run: blocks) precisely so this hook covers it.
[[repos]]
repo = "https://github.com/shellcheck-py/shellcheck-py"
rev = "v0.11.0.1"

[[repos.hooks]]
id = "shellcheck"
args = ["--severity=style", "--enable=all"]
priority = "lint"

# --- Markdown: rumdl (README.md, AGENTS.md, NOTICE.md, docs) ---
[[repos]]
repo = "https://github.com/rvben/rumdl-pre-commit"
rev = "v0.2.58"

[[repos.hooks]]
id = "rumdl-fmt"                       # pure formatter, always exits 0
priority = "format"

[[repos.hooks]]
id = "rumdl-check"                     # lint; MD013 line length, MD033 inline HTML (badges/screenshots), MD041 first-line heading
args = ["--disable", "MD013,MD033,MD041"]
priority = "lint"
```

`.golangci.yml` (v2 format): `linters.default: standard` plus `staticcheck`, `errcheck`, `govet`, `unused`,
`revive`, `gocritic`, `misspell`, `forbidigo` (pattern `tea\.(Tick|Every|Sequence)\(` allowed only in
`internal/ui/clock.go`); `formatters: gofmt, goimports, gofumpt`. All `rev` values are pinned exactly (the ones
above are current as of the reference config; re-pin with `prek auto-update` at scaffold time and again in
the M5 checklist, since Dependabot cannot bump hook revs). `prek validate-config` is part of `make lint`.
Toolchain notes: shellcheck-py, rumdl-pre-commit, zizmor-pre-commit and codespell are `language: python`
hooks, so prek provisions a Python environment for them (via uv) on first run, locally and in CI's cached
prek environment; nothing Python-related is checked into the repo.

### CI (GitHub Actions)

All actions pinned to full commit SHAs; `permissions: contents: read` by default; only `GITHUB_TOKEN` is used.
`actions/setup-go` everywhere with `go-version-file: go.mod` and env `GOTOOLCHAIN=local`, so a toolchain
mismatch fails loudly instead of silently downloading.

`ci.yml` on push and pull_request, three jobs:

- **lint** (`ubuntu-22.04`) — `j178/prek-action` runs `prek run --show-diff-on-failure --all-files` against
  `prek.toml`, with its environment cached. This is the only lint path; nothing else to drift from.
- **test** — matrix over **`ubuntu-22.04`, `macos-latest`, `windows-latest`**: `go test -race ./...` on
  Linux and macOS, `go test ./...` on Windows (the race detector needs cgo there). Golden tests need no TTY.
  This matrix is what enforces the compatibility requirement on every commit.
- **build** — matrix over the same three runners, **building natively on each platform** so the toolchain,
  linker and resulting binary are exercised where they will run:

  | runner | targets built | smoke test on the runner |
  |---|---|---|
  | `ubuntu-22.04` | linux/amd64, linux/arm64 | `./dist/wopr-linux-amd64 --version` and `--games` |
  | `macos-latest` | darwin/arm64, darwin/amd64 | `./dist/wopr-macos-arm64 --version` and `--games` |
  | `windows-latest` | windows/amd64, windows/arm64 | `.\dist\wopr-windows-amd64.exe --version` and `--games` |

  Each job runs `scripts/build.sh <os> <arch>` for its two targets, `scripts/smoke.sh` on the native-arch
  binary (these flags exit without a terminal, so they run fine on a headless runner), `scripts/size-gate.sh`,
  and then uploads **one artifact per target** with `actions/upload-artifact` (`retention-days: 30`), named
  exactly `wopr-linux-amd64`, `wopr-linux-arm64`, `wopr-macos-amd64`, `wopr-macos-arm64`,
  `wopr-windows-amd64`, `wopr-windows-arm64`. To test a build: open the workflow run on GitHub →
  **Artifacts** section at the bottom → download the zip for your platform → unzip → run (`chmod +x` on
  Linux/macOS; macOS may also need `xattr -d com.apple.quarantine`). The README's "Try a development build"
  section repeats these steps.

  Workflow `run:` steps are one-liners that call `scripts/*.sh` (`shell: bash` on all three runners; Git
  Bash on Windows), so ShellCheck lints every line of CI shell and zizmor sees no template-injection
  surface. `scripts/size-gate.sh` (portable: `wc -c`, not GNU `stat`):
  ```bash
  #!/usr/bin/env bash
  set -euo pipefail
  warn=$((10 * 1024 * 1024)); hard=$((15 * 1024 * 1024)); rc=0
  { echo "| artifact | bytes |"; echo "|---|---|"; } >> "${GITHUB_STEP_SUMMARY:-/dev/stdout}"
  for f in dist/*; do
    s=$(wc -c < "$f"); echo "| $f | $s |" >> "${GITHUB_STEP_SUMMARY:-/dev/stdout}"
    [ "$s" -le "$warn" ] || echo "::warning::$f is $s bytes (> 10 MB target)"
    [ "$s" -le "$hard" ] || { echo "::error::$f is $s bytes (> 15 MB hard limit)"; rc=1; }
  done
  exit "$rc"
  ```

`release.yml` on `v*` tags (`permissions: contents: write` on this job only): GoReleaser builds the same six
targets from `.goreleaser.yaml` and publishes a **GitHub Release with binaries for all three platforms**:
`wopr_<version>_linux_amd64.tar.gz`, `wopr_<version>_linux_arm64.tar.gz`, `wopr_<version>_darwin_amd64.tar.gz`,
`wopr_<version>_darwin_arm64.tar.gz`, `wopr_<version>_windows_amd64.zip`, `wopr_<version>_windows_arm64.zip`,
plus `checksums.txt` and release notes generated from the commit log. Tagging `v0.1.0` is the only manual
step. A `workflow_dispatch` trigger with `--snapshot` lets a release build be dry-run without a tag.

`dependabot.yml`: weekly `gomod` and `github-actions` updates.

### Documentation: README for humans

`README.md` is written for someone who has never seen the code and just wants to play. Basic version lands in
M0 (install, run, flags, the `Joshua` logon), completed in M5 with screenshots. Sections, in order:

1. **What this is** — one paragraph, the film, the "SHALL WE PLAY A GAME?" screenshot.
2. **Install** — per OS: download the Release asset for Linux / macOS / Windows (amd64 vs arm64 explained in
   one line each), unpack, `chmod +x`, macOS Gatekeeper (`xattr -d com.apple.quarantine wopr`), Windows
   SmartScreen note, optional `go install github.com/GhostofGoes/WOPR/cmd/wopr@latest`.
3. **Quick start** — run `wopr`, wait for `LOGON:`, type `Joshua`, talk to it, `list games`, pick a number.
   Spoiler-free hint box for people who don't know the film; what `help games` and `list games` do at the
   logon prompt.
4. **Inside the shell** — the commands WOPR understands (`help`, `help games`, `list games`, `play <name|n>`,
   `logoff`), how games take input (Teletype vs Board), keys (`Esc` abort game, `PgUp/PgDn` scrollback,
   `Ctrl+C` quit, any key skips the typewriter).
5. **Command-line flags** — the table from the CLI section with short and long forms and env vars.
6. **Themes** — the four palettes with a screenshot each; `--theme`/`WOPR_THEME`.
7. **The games** — the 15 + tic-tac-toe, one line each on how to play and how WOPR plays.
8. **Troubleshooting** — "TERMINAL TOO SMALL", no colors (set `COLORTERM`/use a modern terminal), Windows
   console tips, slow typing (`-i`), reproducible runs (`-s`).
9. **Try a development build** — the Actions artifact steps from the CI section.
10. **Build from source / contributing** — Go 1.27, `make`, `prek install`, `prek run --all-files`, size
    budget and compatibility matrix, pointer to `AGENTS.md`.
11. **Credits & license** — MIT for code, `NOTICE.md` for film text, non-affiliation, thanks to the brother's
    original WOPR prompt (with his permission).

### Public-repository hygiene

The repository is public. Rules that the tooling enforces and the milestones follow:

- **No secrets, tokens, keys or credentials in the tree, ever.** gitleaks runs on every commit (prek) and in
  CI; prek's `detect-private-key` hook backs it up. `.gitignore` already excludes `.env`; add `*.pem`, `*.key`,
  `dist/`, `coverage.*`, and editor/OS cruft.
- **Workflows use only the automatic `GITHUB_TOKEN`** with least-privilege `permissions:` blocks. No personal
  tokens, no third-party secrets, no signing keys in v1. Actions are pinned by commit SHA and checkout uses
  `persist-credentials: false`; zizmor enforces all of this on every commit and in CI.
- **The app never phones home or writes transcripts.** No telemetry, no network code in v1. The optional
  `WOPR_DEBUG` log contains only program events, never typed input.
- **Future LLM milestone**: API keys are read from environment variables only, never from a config file in
  the repo, and never logged; the README documents this before any provider lands.
- **Personal information**: no real names, emails, or addresses beyond the film's fictional ones (`DR. ROBERT
  HUME, 5 TALL CEDAR ROAD`); the brother's prompt is embedded/adapted only with his permission and credited
  the way he prefers.
- **Licensing**: code under MIT. The embedded screen lines are short quotations of MGM's film and are called
  out in `NOTICE.md` with a non-affiliation disclaimer and excluded from the MIT grant.

---

## Milestones

| M | Deliverable | Notes |
|---|---|---|
| 0 | Scaffold + size spike | files below; measured size table in README; CI green |
| 1 | Console + persona | connect/LOGON/backdoor/greeting flows, themes, clock, typewriter, commands, scripted brain, `--play` with a stub game |
| 2 | Film set pieces | `games/ai`, `board/`, Tic-Tac-Toe (+ montage/ending), Checkers, Chess, Global Thermonuclear War |
| 3 | Card games | `cards/` + `cards/trick.go`, `prompt/`; Black Jack, Poker, Gin Rummy, Hearts, Bridge |
| 4 | Sims + Maze | `sim/` engine + the five military sims; Falken's Maze |
| 5 | Polish & release | QA on Ubuntu 22.04, macOS, and Windows 11 (Windows Terminal + conhost) using the CI artifacts; complete the README (screenshots via cool-retro-term, themes, games, troubleshooting); `prek auto-update`; **write `AGENTS.md`**; tag `v0.1.0` → Release with Linux/macOS/Windows binaries |
| 6 (opt) | LLM brain | provider behind `Brain`, env-configured (keys from env only), scripted fallback, size re-check |

**M0 file order:** `go.mod` (`go 1.27`, `toolchain go1.27.1`) · `.gitattributes` · `.gitignore` additions ·
`LICENSE` + `NOTICE.md` · `internal/version/version.go` · `internal/games/{game.go,registry.go,registry_test.go}`
(16 `Info` entries + a `stub` game) · `internal/cli/{flags.go,usage.go,games.go,flags_test.go}`
(`Parse(args, stdout, stderr) (Config, Action, error)`; short + long flags; test asserting every long flag has a
shorthand) · `internal/theme/theme.go` · `internal/ui/app.go` hello-world that renders one Lip Gloss-styled line
(so the spike measures both deps) · `cmd/wopr/main.go` · `.golangci.yml` (incl. the forbidigo tick rule) ·
`prek.toml` · `scripts/{build.sh,smoke.sh,size-gate.sh}` (ShellCheck-clean, called from CI and the Makefile) ·
`.github/workflows/{ci.yml,release.yml}` + `.github/dependabot.yml` (zizmor-clean: SHA-pinned actions,
`persist-credentials: false`, least-privilege `permissions`) · `.goreleaser.yaml` ·
`Makefile` (`build`, `dist`, `test`, `lint` = `prek validate-config && prek run --all-files`, `size`) ·
`README.md` basic version (sections 1–5, 9–11 of the Documentation outline) with the measured size table and the
compatibility matrix. M0 ends with a green `ci.yml` run showing six downloadable artifacts and a
`workflow_dispatch` snapshot of `release.yml`.

**AGENTS.md (end of M5, basic):** what the project is, build/test/lint commands (`make`, `prek`), repository
layout, the three conventions agents must keep (import direction `ui → games/wopr/theme`, the single clock in
`internal/ui/clock.go`, no Bubble Tea imports in `games/` or `wopr/`), how to add a game (registry entry +
package + testkit test), the size budget and compatibility matrix, and the public-repo hygiene rules. Keep it
short enough to read in a minute; `CLAUDE.md` can simply point at it.

**M1 file order:** `internal/ui/clock.go` · `internal/ui/console/{console.go,typewriter.go,input.go,render.go}`
+ tests · `internal/wopr/{normalize.go,session.go,lines.go,lines/*.txt,scenes.go,commands.go,brain.go,
scripted.go}` + table tests · `internal/ui/screens/{connect.go,toosmall.go}` · `internal/ui/{keys.go,
gamehost.go,app.go}` with the phase machine · `internal/ui/testutil/drive.go` + `app_test.go` golden flows ·
`cmd/wopr/main.go` wiring, TTY check, exit codes.

## Verification (end-to-end)

1. `make dist` cross-compiles all six targets; every artifact ≤ 10 MB target (CI warns above 10 MB, fails
   above 15 MB). `prek validate-config && prek run --all-files` passes locally; the `lint`, `test` (three
   OSes) and `build` (three OSes, native) jobs are green; the run page lists six artifacts
   (`wopr-linux-amd64` … `wopr-windows-arm64`) that download, unzip and run on each platform.
   A `workflow_dispatch` snapshot of `release.yml` produces the six archives plus `checksums.txt`; tagging
   `v0.1.0` publishes them as a GitHub Release.
2. `wopr -v`, `wopr -h`, `wopr -g` and their long forms print and exit 0 without touching the terminal;
   `wopr --games | head -1` works in a pipe.
3. `wopr` → LOGON → wrong name ×3 → hint → `Joshua` → greeting scene → `list games` → `7` → chess board
   renders; Esc returns to the shell with an in-character remark.
4. `wopr -p gtw --instant --seed 1` → side selection → targets → big board → DEFCON ladder → montage →
   `A STRANGE GAME…`; `wopr -p 15`, `wopr -p "global thermonuclear war"` and `wopr gtw` resolve to the same game;
   `wopr -p theaterwide` exits 2 listing both candidates.
5. `go test ./... -race` green; golden snapshots stable under `--seed 1 --instant`.
6. Manual Windows pass in Windows Terminal and legacy conhost (colors, cursor, resize, Ctrl+C restore,
   SmartScreen prompt noted).
