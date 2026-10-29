# WOPR — Architecture & Scope Plan (v2.1)

_Status: v2.1 (2026-10-06), kept in step with the code since. M0–M4 and M6 are built, and v0.1.0 to v0.5.0 are
released. M5's film text pass is done (2026-10-11, §2.3). QA on all OSes and some owner settings (M0b)
remain. The installers (R14) ship in the next release. v2.1 supersedes v1 (commit `ada63a0`, which never reached `main`).
[`docs/reviews/PLAN-v1-review.md`](reviews/PLAN-v1-review.md) is the adversarial review of v1, and
[`docs/reviews/PLAN-v2-review.md`](reviews/PLAN-v2-review.md) reviews v2. Where the plan answers a finding, it
cites the finding's id inline: v1 ids look like A-3, v2 ids look like AR-1. Implementation follows the
milestones in §15._

**How to read this plan.** It states intent, contracts and constraints. The files in the repository are
authoritative for configuration (`prek.toml`, `.golangci.yml`, `.goreleaser.yaml`, the workflows,
`tools/*/go.mod`). The plan keeps no copy of them, so there is no Appendix A (M-2). Likewise, the import table
in `internal/archtest` is authoritative, and §4.1 mirrors it. Conventions live in `AGENTS.md`. This plan links
to it rather than repeating it (M-1).

---

## 1. Requirements

Source codes: **U** = stated by the owner (`docs/reviews/PLAN-v1-review.md`, "Owner decisions", records the
answers). **D** = derived from a U requirement. **P** = a plan default the owner may override.

| # | Requirement | Src | Acceptance |
|---|---|---|---|
| R1 | Go terminal-UI recreation of WOPR from *WarGames* (1983). The binary is `wopr`. | U | — |
| R2 | Every game on the film's `LIST GAMES` (15 entries), plus the tic-tac-toe ending. | U | Each game meets the definition of done (§6). |
| R3 | Persona faithful to the film and to the brother's "WOPR Simulator" prompt. WOPR speaks ALL CAPS, addresses the user as PROFESSOR and is terse. It numbers its lists (except the film's unnumbered `LIST GAMES`, §4.6) and refuses invalid moves. wopr echoes user input as typed, in mixed case, as in the film (F-1). | U | Persona table tests and golden transcripts (§9). |
| R4 | Single static binary for **Linux and Windows**. | U | CI smoke-runs the release binaries natively (§11). |
| R5 | **macOS 26 (Tahoe)** on Intel and Apple Silicon. Older macOS versions are out of scope. | U (2026-10-06) | Smoke runs on `macos-26` and `macos-26-intel`. |
| R6 | Size: **target ≤ 10 MB, hard limit 15 MB** per binary (decimal MB). | U (confirmed 2026-10-06) | CI warns above 10,000,000 bytes and fails above 15,000,000. |
| R7 | `wopr` starts the TUI. `--version`, `--help`, `--games`, `-p/--play <game>`. | U | CLI tests (§5). |
| R8 | Optional LLM hookup, or purely scripted responses. | U | Scripted from v0.1. LLM in M7, opt-in (§4.7). |
| R9 | Every long flag has a one-letter shorthand. | v1 decision 6 (source not recorded) | `TestEveryLongFlagHasAShorthand` (`internal/cli`). |
| R10 | Public repository: no secrets, least-privilege CI, reproducible and attested releases. | D | §11, §12. |
| R11 | First release after the film set pieces. The remaining games ship in minor releases. | U (2026-10-06) | Met: v0.1.0 shipped every game (§15). |
| R12 | **Movie mode** (`-m/--movie`) replays the WOPR terminal scenes from the film. | U (2026-10-06) | §7, M6. |
| R13 | No Dependabot for now. Dependencies are updated by hand on a schedule (§11.5). | U (2026-10-06) | No `dependabot.yml`. |
| R14 | **Installers** for people who never use a terminal. They are a per-user Windows installer and a Mac app in a `.dmg`. The `.deb` and `.rpm` get a menu entry with an icon and AppStream metadata. Each opens a terminal running `wopr`. MSIX and snap packages are built for the stores, but not published yet (§8). | U (2026-10-08) | CI installs, runs and removes each on its own platform (§11). |
| R15 | **Film authenticity.** Every message WOPR shows that the film shows is the film's own screen text, character for character, and it is never modified: not for spelling, grammar, spacing, line breaks, a linter or any writing standard (§2.3, AGENTS.md). Lines of this project's own are tagged `original` and written in WOPR's voice. The documentation quotes film lines exactly as the film shows them. | U (2026-10-11) | The provenance tests and each package's `film_lines.golden` (§9). |

**Supported platforms.**

| OS | Versions | Architectures | Tested on (CI) |
|---|---|---|---|
| Linux | Ubuntu 22.04 and newer, and any distribution with an xterm-class terminal | amd64, arm64 | `ubuntu-24.04`, `ubuntu-24.04-arm` |
| macOS | 26 Tahoe | amd64, arm64 | `macos-26-intel`, `macos-26` |
| Windows | 11 | amd64, arm64 | `windows-2025`, `windows-11-arm` |

Binaries are static (`CGO_ENABLED=0`). Go 1.27's own floor is macOS 13, so builds run fine on 26 (D-5). A
static Linux binary needs nothing from the distribution, only a kernel. Go's floor is Linux 3.2, and 22.04
ships 5.15. The `ubuntu-22.04` runner image is deprecated and unsupported from 2027-04-17. An `ubuntu:22.04`
container would still run on the host's kernel. So CI tests neither, and the 22.04 claim rests on that floor
(B-5).

The installers keep Go's floors rather than the table's. The Windows installer accepts 64-bit Windows 10 and
later. The Mac app declares macOS 13, read from the program (§8). Only the versions in the table are tested or
supported.

---

## 2. Film research

### 2.1 Sources and provenance

Primary sources:

- The film's screen text, from the frame-by-frame transcriptions in §2.3. A viewing of the film itself settles
  only what they leave open.
- *Mini-Micro Systems*, June 1984, "Making Wargames computers compute".
- CIO, 2013, "The technology of WarGames".
- Subtitle-derived dialogue.

Secondary sources: fan transcripts and simulators.

| Fan source | Licence |
|---|---|
| `built1n/wargames` | GPL-3.0-or-later. Its `MAP` is CC BY-SA 4.0 |
| `abs0/wargames` | BSD-2-Clause per-file headers, © David Brownlee |
| `elfuska/wargames` | No licence, only "credit Andy Glenn" |

These are references only. We take no code or art from any of them. We take exactly two text items from
`abs0/wargames`'s transcription, and `NOTICE.md` credits both (owner decision 2026-10-06, L-4):

- the backdoor header block
- the montage scenario names.

abs0's BSD-2 licence covers the transcription. The text is still the film's, so it is excluded from the MIT
grant like other film text (SL-4). The status burst after the header is **not** an abs0 item. It is the film's
phrases in our own layout, tagged `reconstructed` (SL-5).

**Provenance tags.** Every script line, scene step and asset carries one. The tag is one of the values below.
A `third-party:` tag must name a source credited in `NOTICE.md`. The tests check both rules for every package
that holds script text (SL-4). The user's typed lines in movie mode are `script.User` lines, which keep their
mixed case. The types and the check live in `internal/script` (`script.L`, `script.Validate`). The persona,
every game, the assets and the movie scenes share them.

| Tag | Meaning |
|---|---|
| `film` | The film's own screen text, character for character, as the transcriptions in §2.3 establish it. Frozen: never modified (R15) |
| `reconstructed` | From one transcription only, or from a screen the film flashes by. Changed only on better evidence |
| `third-party:abs0/wargames@010ed92:wargames.sh` | Film text as abs0 transcribed it. `NOTICE.md` credits it and carries the BSD-2 notice |
| `third-party:https://asciiart.website/art/3719` | Matthew Thomas's ASCII world map, GTW's big board (owner's choice, 2026-10-06). It is kept exactly as drawn, so the capitals rule does not apply to it. `NOTICE.md` links the page and carries his line, "Map (C) 1998 Matthew Thomas. Freely usable if this line is included.", which a test checks |
| `original` | Written for this project |
| `prompt` | Adapted from the brother's prompt. **Not committed to this public repository at all** until he grants written permission **and** a licence (MIT or CC0). A build tag would not help, because the module zip would still carry it (L-3, L-5, SL-4). |

### 2.2 Facts that shape the design (F-1)

- David's terminal was a 17-inch **Electrohome black-and-white** monitor: white text on black. The default
  `imsai` theme is white phosphor, and green and amber are alternates.
- CompuPro S-100 systems generated WOPR's text off-screen, on a **24-line × 80-column** display. The map
  glyphs came from 34 custom line-segment characters. This supports the 80×24 target. It is also why GTW's
  side-choice outlines are **original line-segment-style ASCII art**. The big board's map is Matthew Thomas's
  (§2.1).
- **WOPR's output is upper case. David's typing appears in mixed case** ("Hello.", "Love to. How about Global
  Thermonuclear War?", "Las Vegas"), with two spaces after a full stop, as WOPR's lines have. The exception is
  the NORAD console in the climax: the film's screen shows `LIST GAMES` in capitals, and abs0's transcription
  has the other entries so too (`CHESS`, `GTW`, `TIC-TAC-TOE`, `ZERO`). Movie mode types them so (§7).
- **LOGON comes first.** The `#45 11456 …` header and status burst appear only *after* WOPR accepts `Joshua`.
  An unrecognised ID prints two lines and drops the connection.
- The GTW-vs-chess exchange is a separate scene after the greeting (§4.6).

### 2.3 Canonical screen text

The canonical text, with its provenance tags, is in **Appendix B**. The code that holds it is authoritative:
`internal/wopr/lines.go`, `internal/assets/`, the games' packages and `internal/movie/scenes/`. **The film's
lines are never modified** (R15). Each package that holds some lists them in a `testdata/film_lines.golden`
(`script.FilmTexts`), so that a change to one shows in review as a change to the film's text.

**The film text pass** (2026-10-11) checked every line against these transcriptions of the film's screens,
made independently of one another:

| Source | What it is |
|---|---|
| Michael Walden, "WarGames Terminal Fonts" (<https://mw.rat.bz/wgterm>) | A timestamped, character-level transcription from frame captures, with the frames. The strongest source. It ends at 1:46:42, before tic-tac-toe |
| `abs0/wargames` (`wargames.sh`, commit `010ed92`) | David Brownlee's transcription, typed from the film, with raw notes on every scene. The only source for the tic-tac-toe prompts and for the names David enters at the NORAD console |
| `built1n/wargames` (`TRANSCRIPT`) | Franklin Wei's transcription of the first-contact and greeting scenes |
| Andy Glenn's BASIC simulator (`elfuska/wargames`) | Careless in places (`IDENTIFICAION`, `6/23/1973`). Used only to break ties |
| `radar-blip/wargames` (`joshua.py`) and `mwigton/WOPR` (`wopr.c`) | Independent readings of the kill-ratio table, the trajectory table and the ending |
| filmjournals.net's list of the montage's tactics, and the Reddit list in Paul Smith's gist | Two readings of the montage's names, against abs0's |
| Wikiquote, moviemistakes.com and IMDb's goofs page | The spoken lines, and the film's own typos on screen |

The rule: a line is tagged `film` when Walden's frame-based transcription has it and no other source
contradicts it, or when two other sources agree on it character for character. Where sources disagree, the
frame-based one wins. A line that only abs0 covers stays `reconstructed`, written as abs0 has it. The pass
settled the three conflicts the plan had recorded, and several it had not seen:

- `USER ACCOUNT NUMBER` (Walden and abs0; built1n and the subtitles drop the word).
- `CITY AND/OR COUNTY NAME` (every source).
- `28 HOURS` (Walden, the subtitles and the film's own timeline; abs0 alone has 61).
- `SYSCOMP STATUS:  ALL PORTS ACTIVE`, with the colon and two spaces (Walden and Glenn). The burst's other
  lines stay `reconstructed`: the screens flash by, and the sources read them differently.
- The log-on refusal reads `INDENTIFICATION NOT RECOGNIZED BY SYSTEM`, the system's own misspelling, in
  every frame it appears in (Walden, three times; IMDb's goofs page). The NORAD notice is
  `** IDENTIFICATION NOT RECOGNIZED **`, spelt with a Z in Walden's frames. abs0, a British author, wrote
  `RECOGNISED`, and the plan had followed him.
- Two spaces after `LOGON:` and after every full stop, in WOPR's lines and in David's.
- The line breaks of the call-back and the NORAD session (`YOU SHOULD KNOW, PROFESSOR.  YOU` /
  `PROGRAMMED ME.`), `Hello, are you still playing the game?`, and the game clock's layout and readings.
- After `HELP GAMES` the film shows no `LOGON:`: David types `List Games` on an open line. A fourth log-on
  attempt, `Armageddon`, is refused like `Falkens-Maze`.
- The kill-ratio table's words and layout: `MILITARY FORCES`, `ICBM'S`, every row in percent, `49 MILLION`,
  the United States on the left. Movie mode shows the film's figures where its frame shows them.
- Thirteen montage names, where filmjournals and the Reddit list agree against abs0's reading
  (`ANGENTINA ESCALATION` is the film's own typo; `AUSTRALIAN`, `BURMESE THEATERWIDE`, `ICELAND INCIDENT`,
  `PACT MEDIUM`, `ROMANIAN`, `PALISTINIAN`, `DEFENSE`).

What the transcriptions leave open stays `reconstructed`, and only the film itself can settle it:

- The status burst's screens.
- Tic-tac-toe's prompts, which abs0 alone records, in the line breaks of the film's big screen
  (`PLEASE LIST NUMBER` / `OF PLAYERS:`, `STALEMATE.` / `WANT TO PLAY AGAIN?`). abs0 also records `X or O?`
  after `1`, which the game does not ask.
- `CHESS`, `GTW`, `TIC-TAC-TOE` and `ZERO` at the NORAD console, and whether the last `Hello.` is typed in
  capitals (abs0's notes have `HELLO`).
- The big screen's grouping of the launch code (`CPE1704TKS` or `CPE 1704 TKS`).
- Eleven montage slots where the three lists disagree three ways (positions 13-14, 21, 27-28, 34-35, 42, 56
  and 62-63).
- The Soviet column of the kill-ratio table's percentages, cut off at the frame's edge.

The NORAD notices are the film's words in the board's three-row strip. The film centres each on an empty
screen, underlines the first line and leaves blank lines between them; the strip has no room for that.

### 2.4 Visual language

- **Console**: full-bleed text with no chrome. The console double-spaces WOPR lines and reveals them at modem
  speed, with a solid block cursor.
- **NORAD big board**: a near-black blue field, cyan/blue outlines, red incoming and yellow outgoing tracks,
  and white labels. It also has a DEFCON column of boxed numerals 5…1, and `** … **` notices. This is the
  `norad` theme.
- **WOPR cabinet**: rows of lights and a `W.O.P.R.` nameplate. The optional one-row front panel draws them, and
  also shows "thinking" (§4.3).
- **Verbiage**: ALL CAPS, terse, declarative. WOPR never asks clarifying questions. It numbers its lists
  `1.  2.  3.` (except `LIST GAMES`, which is unnumbered as in the film). It refuses invalid actions with
  `** IMPROPER REQUEST **`-style lines. It calls the user PROFESSOR.
- **Anti-pattern**: bordered panes and labelled input boxes. Borders appear only inside the big board and in
  NORAD notices, and as a game's own picture (a board's edge, a card's face, a maze's frame).
- **ASCII art** (decision 26): every game opens with an original title piece and draws its world in printable
  ASCII capitals. Card games draw card faces and trick tables. Chess, checkers and the maze have a frame.
  Tic-tac-toe uses large film-style marks. The sims draw their front as a strip. GTW's side choice shows the
  two nations' outlines, generated from Natural Earth (public domain). Every piece fits its layout at 80×24
  and is tagged `original`. The one exception is the big board's world map, which is Matthew Thomas's
  (credited above).

The target screens are in **Appendix C**, all at exactly 80×24. Appendix C draws the greeting. The goldens hold
the GTW side choice, the GTW big board and the chess panel as built, with the real art
(`internal/ui/testdata/gtw_screens.golden` and `internal/games/chess/testdata/view.golden`, L-4).

---

## 3. Decisions

| # | Decision | Choice | Src |
|---|---|---|---|
| 1 | TUI stack | **Bubble Tea v2.0.10 + Lip Gloss v2.0.6**, no bubbles. The two measured 3.6–3.9 MiB stripped on all six targets, with one styled view (§8 has the program's size). We plan no fallback renderer (D-3). | U |
| 2 | Brain | **Scripted** behind `Brain`. The LLM provider is **M7**, optional and opt-in. | U |
| 3 | Military sims | Short turn-based mini-wargames: four scenarios on one `sim` engine, and two bespoke games that reuse its combat-results table and kill ratios (§6.3, G-2). | U |
| 4 | LOGON | Film-faithful: a closed command table, the `Joshua` backdoor, a hint after 3 failures. `--play` and `--movie` bypass it (A-17). | U |
| 5 | Platforms | Linux and Windows (amd64, arm64). **macOS 26 only** (amd64, arm64). | U |
| 6 | Arg parsing | Stdlib `flag`. Every long flag has a shorthand. One bare positional means `--play`, and flags may follow it (C-1). | v1 |
| 7 | Persistence | None. Flags and `WOPR_*` environment variables only. | P |
| 8 | Toolchain | `go 1.27` + `toolchain go1.27.2` (D-1 closed by the owner). CI must not set `GOTOOLCHAIN` before `setup-go` (D-6). | U |
| 9 | Lint | **prek** with **`prek.toml`** (authoritative in the repository since M0). | U |
| 10 | Size | Target ≤ 10 MB, hard limit 15 MB. | U |
| 11 | CI topology | Build once with GoReleaser. Run each binary natively on its OS/arch (B-2). | U |
| 12 | Docs | README from M0. **AGENTS.md from M0** is the single source for commands and conventions. `CLAUDE.md` contains `@AGENTS.md` (M-1). | P (review M-1) |
| 13 | Releases | v0.1.0 after M2, then minor releases as work lands, then v1.0.0 after M5 (P-2). | U (v0.1.0 after M2). P (the rest) |
| 14 | Bridge | Minimal: WOPR bids all four seats by point count. The user always plays the declaring side, as declarer plus dummy, with seats rotated when East/West win the contract. Passed-out deals are redealt (G-3). | U (minimal). P (seat rule, G-3) |
| 15 | Chess rules | `github.com/corentings/chess/v2` v2.6.0, the maintained MIT fork of the archived `notnil/chess`. Search lives in `games/ai` (G-1). | P |
| 16 | Branching | **`main` is PR-only**. One required check, `ci-ok`. Squash merges only. CI also runs on every branch push (§11.2). | U |
| 17 | Dependabot | **Not used.** Manual update cadence plus a weekly staleness and vulnerability report (§11.5). | U |
| 18 | Montage names | Keep the list and credit abs0 under BSD-2 now. Checked against two other readings in the film text pass (§2.3). | U |
| 19 | Movie mode | **M6**: `-m/--movie` replays the film's WOPR terminal scenes (§7). | U |
| 20 | Tool modules | Four: `tools/`, `tools/lint/`, `tools/release/` and `tools/docs/`. §10 lists their tools. golangci-lint's dependencies break gitleaks's build when they share a module (SL-2). | P |
| 21 | `LIST GAMES` | Printed unnumbered, as in the film, at LOGON and in the Shell. In the Shell, a number typed right after it still selects. `HELP` and the README say so. `--games` stays numbered (RF-6). | U (confirmed 2026-10-06) |
| 22 | Leaving the war | Esc twice may abandon GTW and the climax tic-tac-toe, like any game, with WOPR's remark about the abandoned war. The ending itself cannot be aborted (§6.2). | U |
| 23 | `-m N` | Plays from scene N to the end of the list, then exits 0 (§7). | U |
| 24 | GTW exchange | **Turn-based DEFCON**, built in M5 (§6.2). | U |
| 25 | History and legal text | The branch keeps its v1 history (the NOTICE credit covers the early-draft fragments). LICENSE holder: Christopher Goes. The Code of Conduct's contact: "contact @GhostofGoes privately via GitHub profile". Lines derived from the brother's prompt stay out of the repository until his written licence (L-3). | U |
| 26 | ASCII art | Original art in every game, in the film's spirit (owner request 2026-10-06). The rules: printable ASCII capitals, every line ≤ 80 columns, every screen within its layout, each piece tagged `original`. Galleries and other WOPR projects are style references only. GTW's side-choice outlines are generated from Natural Earth (public domain, credited in NOTICE.md). The one exception is the big board's world map, Matthew Thomas's (owner's choice, used under his terms and credited in NOTICE.md). The board shows its northern 13 rows, in its own equirectangular projection. Cities are placed from their latitude and longitude, each in a cell of its own. A city whose cell is open water goes to the nearest land. The few that share a cell or fall a column off the art's coast are moved one cell, each with its reason in `internal/assets/gtwmap.go`. The other targets a list may name are placed the same way and may share a city's cell. The assets tests pin every place's cell. | U |
| 27 | Installers | A per-user Inno Setup installer for Windows and a universal `WOPR.app` in a `.dmg` for macOS, both release files. A menu entry, icons and AppStream metadata in the `.deb` and `.rpm`. An MSIX bundle for the Microsoft Store and snaps for the Snap Store, built and tested but not published. One app ID, `io.github.ghostofgoes.wopr`. The main launchers open a terminal running `wopr` with no arguments, with no start screen yet. The Linux and snap menu entries add right-click actions for Chess, Checkers, Tic-Tac-Toe and Global Thermonuclear War (`wopr --play <game>`). They also add one for Movie Mode (`wopr --movie`, which the owner kept, 2026-10-09). No installer is code-signed (the Mac app ad hoc only). The Linux packages carry OpenPGP signatures (§8). Issues track signing (§8). | U (2026-10-08) |

---

## 4. Architecture

### 4.1 Packages and the import DAG

Module `github.com/GhostofGoes/WOPR`, binary `wopr`. Import edges not listed below are forbidden (A-6, A-13).
The table in `internal/archtest` is authoritative, and this one mirrors it. A change goes into both in the same
PR (AR-1, IM-4).

- `pkg/...` means the package and every package below it. The first matching row applies, so specific rows
  come before the subtrees that contain them.
- Every package may import the standard library, except for three fenced packages. `net/http` is for `llm`
  only. `os/exec` is for `cmd/wopr` (whose macOS build reopens the app in Terminal), `tools/...`, `archtest`
  and `e2e` only. No package may import `unsafe`.
- **Test-only imports.** Any package's tests may also import `golden`, `games/catalog`, `games/gamestest`,
  `games/testkit` and `proto/host` (to configure testkit's runner). No non-test file may import `gamestest` or
  `testkit` (G-5).

```text
. (legal.go)         → stdlib                         embeds LICENSE, NOTICE.md, THIRD_PARTY_NOTICES.txt (SL-3)
cmd/wopr             → ., cli, ui, version, debuglog, games/catalog, (llm in M7); plus os/exec (macOS relaunch)
internal/debuglog    → stdlib                         the opt-in debug log (§8)
internal/version     → stdlib
internal/cli         → games, theme (--theme validation), version, movie (the scene index)
internal/proto       → stdlib                         the program protocol, canvas, keys, rand
internal/proto/host  → proto                          Bubble-Tea-free runner (A-14)
internal/prompt      → stdlib                         normalise, clauses, numbers, yes/no
internal/script      → stdlib                         screen text with provenance tags, and their validator
internal/theme       → proto, charm.land/lipgloss/v2, github.com/charmbracelet/{colorprofile,x/ansi}
internal/ui/...      → ui/..., proto, proto/host, prompt, wopr, games, theme, assets, movie, debuglog,
                       charm.land/*, github.com/charmbracelet/{x/ansi,x/term,colorprofile},
                       github.com/rivo/uniseg
internal/wopr        → proto, prompt, script, games (types only; the registry is injected)
internal/games       → proto, prompt                  types, Registry, Resolve (one normaliser, AR-4)
internal/games/catalog                → games, games/..., proto         the only place constructors are wired
internal/games/gamestest              → proto, games    stub programs; tests only
internal/games/testkit                → proto, proto/host, games, golden   tests only
internal/games/ending                 → proto, prompt, script, games, games/{tictactoe,ai,board}, assets (AR-6)
internal/games/{ai,board}             → proto, games
internal/games/cards                  → proto, games, prompt
internal/games/...                    → proto, prompt, script, games, games/{ai,cards,board}, assets, sim,
                                        github.com/corentings/chess/v2
internal/sim         → proto, prompt, script          the war-game engine (M4)
internal/assets      → script                         embedded art, scenario names (with provenance)
internal/movie/...   → movie/..., proto, prompt, script, assets, games/gtw, games/ending      (M6)
                       tests also: wopr (the consistency test, §7)
internal/golden      → stdlib                         golden-file helper (Q-3)
internal/archtest    → stdlib                         enforces this table
internal/e2e         → github.com/charmbracelet/{x/xpty,x/vt,x/term}   build tag e2e (Q-1)
internal/tools/manpage → cli, games, games/catalog, movie, theme   the manual page, from the program itself (§13)
internal/tools/...   → stdlib                         sizegate, stage, notices, relnotes, pkgdocs, icons, macapp,
                                                      snapdir, cooldown, branchcov (Go programs, not shell)
internal/llm         → proto, wopr                    plus net/http (M7)
```

- **Only `ui` imports Bubble Tea.** Only `ui` and `theme` import `charm.land/*`. `main` stays free of Bubble
  Tea: `ui.Run` owns every step that touches it (§4.2, IM-9).
- **`games` holds types only.** `games/catalog` imports every game package and returns the ordered `Registry`
  of `Entry{Info, New func() Game}` (§4.4). That removes the `games → games/<game>` import cycle (A-13).
- **No game imports `ending`.** Games pass control to the climax by slug through `Result.Next`. `ending` reuses
  tic-tac-toe's engine and drawing instead of duplicating them (AR-6).
- **Injection.** `cmd/wopr` passes the registry to `cli` and `ui`. `ui` builds the persona and the host's
  resolver from it. Tests use fakes from `gamestest`.
- **`internal/archtest`** reads each package's `Imports`, `TestImports` and `XTestImports` from
  `go list -tags=e2e -json ./...` (the tag includes `internal/e2e`). It does so once for each `GOOS` the
  release builds (linux, darwin, windows), because some files build for one only. It checks every edge against
  its table. `games.NewRegistry` itself checks that every `Playable` registry entry has a constructor. archtest
  also checks that the running Go is at least `go.mod`'s `toolchain` line. In CI, it must be exactly that line,
  where a mismatch means `setup-go` fell back (D-6, AR-2). A newer local Go is fine.
- **The M6 consistency test** (§7) lives in `internal/movie`'s tests. archtest's per-row test-only allowance
  (`testAllow`) lets those tests, and only them, import `wopr` (AR-1).
- **Editor feedback**: `depguard` in `.golangci.yml` reports Bubble Tea imports outside `ui`. `archtest` is
  authoritative.

```text
legal.go                     package wopr (module root): go:embed of the licence files for --licenses
cmd/wopr/main.go             flags → dispatch → exit code; WOPR_PANEL; maps ui.Outcome to an exit code (§4.2)
cmd/wopr/launch*.go          the macOS app's relaunch in Terminal (§8); does nothing on other systems
internal/version/            Version/Commit/Date via -X; fallback to debug.ReadBuildInfo(); "unknown" if absent (D-4)
internal/cli/                Parse(args, reg, getenv) (Config, Action, error); usage; --games (§5)
internal/proto/              program.go output.go canvas.go key.go rand.go typed.go
internal/proto/host/         runner.go (program stack, input mode, Esc machine, Think jobs, Animate, pacing)
internal/prompt/             normalisation, clauses, numbers, menu choice, yes/no, negation
internal/theme/              palettes: Style → (truecolor, ANSI-16 index, no-colour attribute) (U-7)
internal/ui/                 app.go (model, Run, effects) clock.go (the single tea.Tick) keys.go render.go
                             drive_test.go: synchronous driver with a fake Scheduler (A-14)
  console/                   sanitize, wrap, scrollback, typewriter, line editor
internal/wopr/               persona.go (session, dial, LOGON, greeting, shell) intent.go brain.go scripted.go lines.go
internal/games/              registry.go (Info, Status, Game, Entry, Registry, Resolve)
  ai/ cards/ board/          search (depth/node limits, time cap), decks and tricks, board drawing and square parsing
  ending/                    the climax: tic-tac-toe self-play → montage → final dialogue (A-16)
  catalog/ gamestest/ testkit/
  falkensmaze/ blackjack/ ginrummy/ hearts/ bridge/ checkers/ chess/ poker/ gtw/ tictactoe/
  fightercombat/ guerrilla/ desertwarfare/ airtoground/ theaterwide/ biotoxic/
internal/sim/                M4 engine
internal/assets/             gtwmap.go (Matthew Thomas's map, third-party; the outlines and places, original),
                             scenarios.go (third-party:abs0)
internal/movie/              director.go scenes/*.go (M6)
internal/golden/ internal/archtest/ internal/e2e/
internal/tools/              {sizegate,stage,notices,relnotes,manpage,pkgdocs,icons,macapp,snapdir,cooldown,branchcov}/
packaging/                   description.txt (both Linux packages'), debian/copyright (generated by notices, §8), rpmlintrc,
                             sign-packages.sh (the release's signing step, §8)
  icons/                     src/*.svg and every icon generated from them (internal/tools/icons, §8)
  linux/                     the menu entry's and the AppStream metadata's templates (pkgdocs fills them)
  windows/                   wopr.iss, winres.json, the MSIX manifest, and the build and test scripts
  macos/ snap/               the disk image's build and test scripts; the snap's test script and snap.yaml.in
tools/go.mod                 Go tools: gitleaks, govulncheck
tools/lint/go.mod            Go tool: golangci-lint (separate: its dependencies break gitleaks's build, SL-2)
tools/release/go.mod         Go tools: goreleaser (T-11), changie (release notes, §11.3), go-winres (wopr.exe's icon, §8)
tools/docs/go.mod            Go tool: Hugo, standard edition (the docs site, §13)
site/                        the docs site (§13): Hugo config, pages, layouts, game data; no Go packages
  go.mod                     a Hugo module file, not Go code: pins the Hextra theme
```

### 4.2 Runtime flow and state ownership

1. **`main`** parses flags. Print-and-exit actions (`--version`, `--help`, `--games`, `--scenes`, `--licenses`) write
   plain, unstyled text in one buffered write. They ignore `SIGPIPE`, and they treat `EPIPE` (Unix) or
   `ERROR_NO_DATA`/`ERROR_BROKEN_PIPE` (Windows) as success (C-3, U-11).
2. **TTY check.** `ui.TerminalProblem` reports a stdout that is not a terminal, or `TERM=dumb`. It checks stdout
   with `charmbracelet/x/term`, because `os.ModeCharDevice` also accepts `/dev/null`. `main` prints
   that one line and exits 2. On Windows the line suggests Windows Terminal. A non-TTY stdin is fine, because
   Bubble Tea opens `/dev/tty` or `CONIN$` itself. Bubble Tea's "error opening TTY" maps to the same exit 2
   (U-4).
3. **NO_COLOR.** If `NO_COLOR` is set to any non-empty value, `ui.Run` passes
   `tea.WithColorProfile(colorprofile.Ascii)`. colorprofile alone parses it with `ParseBool` and ignores
   `NO_COLOR=yes` (U-7). Every Bubble Tea step lives in `ui`. `ui.Run` returns an `Outcome` (`Finished`,
   `Interrupted`, `Panicked`, `NoTerminal`, `Failed`), and `main` only maps it to an exit code (IM-9).
4. **Screen.** The TUI runs on the **alternate screen** and paints the theme background into every cell.
   It does not use OSC 11, which ignores `NO_COLOR` and resets the terminal to its default background on exit.
   On a normal exit (0 or 130), the normal screen gets one line: `--CONNECTION TERMINATED--` (U-1).
5. **Ownership** (A-5).
   - `proto/host.Runner` owns the **program stack**. The persona is the root program. `Launch` pushes the
     program that the injected resolver builds (§4.4). `Done` pops it and delivers `GameOver{Result}` to the
     program below. `Done` with `Result.Next` is a hand-off: the **host** replaces the program with the next one
     and tells no one (AR-5, IM-6).
   - The persona owns the **conversation phase**: `Logon` from the first dial (a re-dial stays in it),
     then `Greeting → Shell`. There is no game phase ("a game is active" is "the stack is deeper than one").
     There is no ending phase either. The persona gets no input while the climax runs. The ending's `GameOver`
     (the only `NoVerdict` result) returns it to `Shell` with chess offered.
   - `ui` owns only presentation state: the screen mode (`Console`, `Panel`, `Full`, `TooSmall`), derived from
     the top program's layout and the terminal size, plus scroll position.
6. **Dial and LOGON** (A-17).
   - **Dialing**: the modem dial animation runs as `Say`/`Wait` steps of the persona. Any key or `--instant`
     skips it. A re-dial uses a short version.
   - **Logon** is a closed table. Input is normalised and matched exactly. There is no intent parsing, no
     brain and no numbered selection.

     | Input | Response |
     |---|---|
     | `HELP LOGON` | `HELP NOT AVAILABLE` |
     | `HELP GAMES` | The games blurb, then an open line with no `LOGON:`, as the film shows it |
     | `LIST GAMES` | The list, then `LOGON:` again (the film cuts away). No selection is armed. |
     | `JOSHUA` | The backdoor |
     | `LOGOFF`, `LOG OFF`, `EXIT`, `QUIT` (punctuation ignored: `Log off.`) | Exit 0 |
     | Empty line | `LOGON:` again |
     | Anything else, including a game name | `INDENTIFICATION NOT RECOGNIZED BY SYSTEM` (the film's own spelling) / `--CONNECTION TERMINATED--`, then `Wait`, `Clear` and a short re-dial |

   - The failure counter lives in `Session`. After every third termination, the persona prints an `original`
     hint line before the next `LOGON:`.
7. **Backdoor**: the header block, `Clear`, the status burst, `Clear`, `GREETINGS PROFESSOR FALKEN.`, then the
   greeting scene (§4.6) and **Shell**.
8. **Shell dispatch.** Sanitised input goes to these handlers, in order:
   1. Host commands (`LOGOFF` family, at every depth of the stack).
   2. **Phase-gated commands** (`HELP`, `HELP GAMES`, `LIST GAMES`).
   3. **Numbered selection** (only on the line right after a list).
   4. **Explicit intent** (§4.6).
   5. The **greeting scene** while it is active.
   6. `PLAY <name>` that matches no game or several.
   7. **Offer acceptance** (§4.6).
   8. **Brain** (asynchronous, via `Think`).

   A command or an explicit request therefore ends the greeting early. Any other non-empty line advances it
   (IM-7, RF-11). An empty line only re-prompts: it keeps a pending offer and an armed list. `PLAY <name>` that
   matches no game answers `NO SUCH GAME IN MEMORY`. One that matches several lists them.
9. **`--play <game>`** pushes the game straight onto a persona in `Shell`, with the greeting marked as done.
   `--movie` runs the movie director as the root program instead of the persona (§7).
10. **Exit codes.**
    - `LOGOFF`, `Ctrl+D` on an empty input line, and SIGTERM exit 0.
    - `Ctrl+C` and SIGINT exit 130.
    - A hang exits 1: the watchdog (§4.4) restores the terminal and says so on stderr.
    - `Ctrl+Z` suspends on Unix (`tea.Suspend`) (U-10).

### 4.3 Console, clock, cursor, input

- **One clock** (A-9). `ui/clock.go` holds the only `tea.Tick`. While anything is busy (the typewriter and its
  pauses, a program's `Animate`, a pending `Think`, the Esc window), it keeps one tick in flight at about
  30 fps. Each tick carries a generation, so the clock drops a stale tick. With nothing busy, no timer runs.
  - A late tick delivers one coalesced `dt`, which the host splits into at most one `TickEvent`.
  - `Blink` cells and the front-panel lights move only while the clock runs. A program that wants a blink while
    idle asks for `Animate`.
  - The UI's golden driver (`ui/drive_test.go`) substitutes a fake `Scheduler`, so it never calls `tea.Tick`
    (A-14).
- **Pacing.** A dt-based typewriter: about 30 chars/s for speech, fast for tables, instant for boards.
  `--instant` / `WOPR_INSTANT=1` and "skip" both mean "advance by infinity". Movie mode adds a user-typing pace
  (§7).
- **Cursor.** `tea.NewCursor` gives a blinking block. It sits at the end of the input line, or after the
  last revealed text. It is steady while WOPR thinks, while output is held and under `--reduce-motion`. Key
  mode hides it, except in movie mode, where it stays after the text unless a key hint (`** PAUSED **`) is on
  screen.
- **Skip policy** (U-3).
  - While output is revealing or paused (`Wait` is skippable), the first key flushes the queue.
  - In line mode, a printable key *also* goes into the input line (typeahead kept). Enter and Space act
    purely as "skip" only when the input line is empty.
  - In key mode, the key flushes *and* reaches the game.
  - PgUp/PgDn always scroll. Esc only skips. Ctrl+C always quits.
  - The docs site's Usage page says so: any key shows the rest of WOPR's text, and wopr keeps what you type.
- **Thinking.** While a `Think` or a Brain reply is pending:
  - The cursor stops blinking. A `PROCESSING` row with one to three dots cycling at 2 Hz sits directly above
    the input line (it is the last row when the input line is empty).
  - The front panel lights animate when the panel is shown.
  - The line editor **buffers** keys and shows them on the input row, and Enter is **held**. The line is
    submitted as soon as the prompt is active again, whatever made it active (a tick, a skip, or an instant
    flush). The exception is when a different program now runs: a line held for a game that ended is dropped
    with its prompt. The input line wraps onto further rows rather than running past the edge.
- **Layout changes follow the text.** When a game with a board ends, its last View stays on screen until the
  typewriter reveals its final output (a typewriter mark). Then the layout changes as WOPR's verdict starts.
  Chess and checkers also hand the final position to the persona in `Result.Lines`, like tic-tac-toe. So it
  survives `--instant` and stays in the scrollback.
- **A game with its own screen starts on a new page.** When the host launches a program whose layout is
  Panel or Full, a new console page starts. This holds whether the persona's menu launched it or a hand-off
  did (such as GTW → tic-tac-toe → the ending). So the strip under the board shows only that game's text, not
  the last game's lines and the menu that chose this one. The earlier pages stay in the scrollback. A game
  ending does not start a page: its last lines stay above WOPR's verdict and `SHALL WE PLAY ANOTHER GAME?`.
  Console games continue on the conversation's page.
- **Esc machine** (the host owns it, and games never see Esc).
  - During a reveal, Esc skips.
  - In a game, the first Esc shows `** PRESS ESC AGAIN TO END GAME **` for 3 s. A pending `Think` keeps
    running. A second Esc within 3 s cancels the `Think` context (its late result is dropped by generation)
    and pops the game with `Done{Aborted}`. Any other key disarms it.
  - While a Brain reply is pending, Esc cancels it: the persona receives `ThinkDone{Err: ErrCanceled}` and
    prints `** REQUEST CANCELLED **` (`original`).
  - At the Shell, Esc does nothing. Esc cannot abort the ending (self-play, montage, final dialogue). The
    resolver marks internal entries `Placement.NoAbort`, and the mark survives `SetLayout`. Ctrl+C still
    quits. GTW and the climax tic-tac-toe are games, so Esc-Esc ends them like any other (§6.2).
- **Sanitising** (U-2). Two functions in `ui/console`, both tested. They map whitespace controls **before**
  they drop the other controls, so pasted lines are never glued together (RF-4).
  - `SanitizeInput` serves typed and pasted text:
    1. Replace invalid UTF-8 with U+FFFD.
    2. `ansi.Strip`.
    3. Map CR LF, CR, LF and TAB to one space each (a multi-line paste becomes one line).
    4. Drop the remaining C0, C1 and DEL.
    5. Cap at 256 grapheme clusters.
  - `SanitizeText` serves any text from outside the binary (the M7 LLM):
    1. Replace invalid UTF-8.
    2. `ansi.Strip`.
    3. Normalise CR LF and CR to LF.
    4. Split on LF into logical lines.
    5. Expand tabs to 8-column stops.
    6. Drop the remaining controls.

    Nothing calls it yet, since the scripted brain's lines are compiled into the binary.
  - Fuzz properties: no control rune survives, the result is at most 256 clusters, and every
    wrapped row fits its width.
- **Width** (U-6). The line editor and the wrapper measure with the same method as the renderer. That is
  `wcwidth` by default, and grapheme widths once the terminal confirms mode 2027.
- **Scrollback.** At most 2000 logical lines, wrapped at render time from the newest back as needed. New output
  auto-follows. `Clear` starts a new page, and history stays above it. A short page starts at the top of
  the screen. Scrolled up, its blank rows move with the text, so the lines just above the page appear first.
  `Render(w, h)` emits exactly `h` rows.
- **Layout geometry** (AR-11, IM-8). Let `H'` be the terminal height minus one when the front panel is shown.
  - The layout width is `min(cols, 80)`, centred. The UI paints the theme background across the full width.
  - **Console**: `H'` rows. The input row follows the last line of text (with `PROCESSING` or a host notice
    just above it), so it is the bottom row once the console is full.
  - **Panel**: the game's view takes `min(Info.PanelRows, H' − 8)` rows on top (`PanelRows` ≤ 12). The console
    strip, input row included, takes the rest: 12 rows at 80×24, 11 with the front panel.
  - **Full**: the view takes `H' − 4` rows, then a 3-row strip and the input row.
  - The **front panel** is exactly one row at the bottom: `W.O.P.R.   LINE 1200 BAUD   ONLINE` on the left and
    eight lights on the right. The lights chase while WOPR thinks. `norad` shows it by default. `WOPR_PANEL`
    set to a true value shows it in any theme, and a false value hides it even in `norad`.
  - Programs draw into the area that `Env` and `ResizeEvent` report, so they never need these formulas. The
    catalog test caps `PanelRows` at 12, and the UI goldens render each layout at 80×24 with and without the
    panel.
- **Too small** (U-8). Below 80×24, the UI shows the `TERMINAL TOO SMALL` card, and:
  - The typewriter, `Animate`, `Wait` and Blink consumers **pause**.
  - A finishing `Think` is held until the size is valid again.
  - Every key except Ctrl+C and Ctrl+Z is dropped.
  No program starts before the first valid size. `ResizeEvent` fires only when a program's layout area
  actually changes.
- **Mouse.** Mouse reporting stays off, so native text selection works. On the alternate screen some terminals
  (VTE) turn the wheel into ↑/↓, which cycles input history (U-9). The M5 accessibility pass kept this. Mouse
  reporting would remove native selection and copying. Resetting the terminal's alternate-scroll mode
  (DECRST 1007) would outlast wopr, since Bubble Tea restores only the modes it sets. The README's
  troubleshooting says so and points to PgUp/PgDn.

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
	Seed          uint64 // this program's seed: the session seed for the root, a per-launch seed for games
	Width, Height int    // the program's layout area for its placement at the current size
	Instant       bool
	Deterministic bool   // --seed or WOPR_SEED, or under test, or movie mode: AI search obeys Limit, not the clock
	ReduceMotion  bool   // --reduce-motion: no acceleration, no blinking (§4.5)
	Mode          string // optional launch mode, e.g. "climax" (§6.2)
}

type Layout uint8 // LayoutConsole, LayoutPanel, LayoutFull

// Events: host → program.
type Event interface{ isEvent() }
type LineEvent   struct{ Text string }          // answer to the last Prompt; "" for an empty Enter
type KeyEvent    struct{ Key Key; Rune rune }   // only in key mode; Key: Up Down Left Right Enter Backspace Rune Esc
type TickEvent   struct{ Dt time.Duration }     // at the cadence the program asked for
type ResizeEvent struct{ Width, Height int }
type ThinkDone   struct{ Value any; Err error } // see the Think contract below
type GameOver    struct{ Result Result }        // to the program below a popped one
type Drained     struct{}                       // answers Drain: the output before it has been revealed (M6)

// Outputs: program → host. All are declared here, so the interface stays sealed (A-15).
type Output interface{ isOutput() }
type Say       struct{ Lines []string; Pace Pace; Open bool } // typewriter; PaceSpeech, PaceTable, PaceInstant,
                                                         // PaceTyping; Open: the next Say continues the last line
type Prompt    struct{ Text string }                     // line mode: next LineEvent answers it
type AwaitKeys struct{ Hint string; Capture bool }       // key mode (the maze); Capture: M6, below
type Animate   struct{ Every time.Duration }             // 0 stops
type Wait      struct{ D time.Duration }                 // host-timed pause; skippable; dropped under Instant
type Clear     struct{}                                  // page break
type SetLayout struct{ Layout Layout; PanelRows int }    // e.g. GTW: Console for text, Full for the map
type Redraw    struct{}
type Think     struct {                                  // all slow work runs off the UI goroutine (A-3)
	Fn     func(ctx context.Context) (any, error) // captures only values (a FEN string, not *chess.Position)
	Limit  Limit                                  // MaxDepth / MaxNodes, captured by Fn; non-zero marks a bounded search
	Budget time.Duration                          // wall-clock cap when not deterministic
}
type Launch struct{ Slug, Mode string }               // push the program the host's resolver builds
type Done   struct{ Result Result }                   // pop this program
type Quit   struct{}                                  // exit 0 (the movie director); LOGOFF is the host's
type Hold   struct{ On bool }                         // freeze output, as TOO SMALL does (M6)
type Drain  struct{}                                  // ask for Drained (M6)
type Skip   struct{}                                  // reveal what is queued at once, as a key would (M6)

type Outcome uint8 // Win, Loss, Draw, NoWinner, Aborted
type Result struct {
	Outcome   Outcome
	Lines     []string // kill ratios, detail
	Next      *Launch  // hand-off without a verdict, e.g. gtw → tictactoe(climax) → ending; the host chains it
	NoVerdict bool     // the persona adds nothing; only the ending sets it
}
```

```go
// internal/games: types only (IM-6).
type Game interface{ proto.Program }
type Status uint8 // Planned, Playable
type Info struct {
	Number    int      // place in LIST GAMES (1..15); 0 when not listed
	Listed    bool     // on the film's list; unlisted entries appear under --games "ALSO AVAILABLE"
	Name      string   // as the film prints it
	Slug      string   // the full name, lower case and hyphenated: "theaterwide-tactical-warfare"
	Aliases   []string // e.g. "gtw", "ttt"
	Layout    proto.Layout
	PanelRows int      // for LayoutPanel; ≤ 12
	Status    Status
	Blurb     string   // one line about the game
	Internal  bool     // the ending: only Get finds it (below)
}
type Entry struct {
	Info Info
	New  func() Game // nil while Planned; Env arrives in Start
}

// internal/proto/host: the resolver ui injects (AR-5).
type Placement struct{ Layout proto.Layout; PanelRows int; NoAbort bool } // NoAbort: Esc cannot end it
type Resolver func(proto.Launch) (proto.Program, Placement, error)
```

Rules:

- **Movie-mode hooks** (M6, AR-7, RF-1, IM-11). The `gamestest` stub has a case for each but `Hold`, which a
  game cannot send. They are additive. Only movie mode uses them, in the director, and in tic-tac-toe and the
  ending, which type the user's lines themselves there.
  - **Key capture.** A *root* program sends `AwaitKeys{Capture: true}`. It then receives every key as a
    `KeyEvent`, Esc as `KeyEsc`, with none of the host's side effects. No key skips output, and the Esc
    machine is idle. A `Prompt` ends the capture. The host ignores `Capture` from a launched program, so Esc
    stays the host's in every game. While a capturing root's launched program runs, keys other than Ctrl+C
    (and PgUp/PgDn) do nothing. Each such key shows `** THE GAME PLAYS TO THE END. CTRL+C QUITS. **` for the
    Esc window (`Runner.Refused`), so the key is not met with silence.
  - **`Hold{On}`** freezes the typewriter, `Wait`, `Animate`, Blink and the front panel, reusing the TOO
    SMALL pause. Keys still arrive, but none reveals held output, and neither does `--instant`. Only a root
    that captures keys can hold, since only it is sure to get the key that releases it. The host ignores
    `Hold` from anything else. Only a change is reported. A hold ends when its program launches another or
    ends, so a launched program never starts frozen.
  - **`Drain`** queues a marker after the program's output. When the typewriter reaches it, the running
    program receives `Drained`. A newer `Drain` replaces a pending one. A marker reached while another
    program runs is dropped. A marker reached while the program has a `Think` pending is answered after its
    `ThinkDone`. So the two arrive in the same order under `testkit` (which answers a marker after the rest
    of its batch, where the console would), `--instant` and pacing.
  - **`Skip`** reveals what is queued at once (next and previous scene), held output included.
  - **`Say.Open`** leaves the last line open, so the next `Say` continues it on the same row.
    `proto.Typed(prompt, text, rng)` uses it to type a user's line after its prompt, a few seeded keystrokes
    at a time. The blank line the console leaves after an answer follows. Any line added from outside (the
    echo of a submitted line) closes it. So does any page break, including one the UI applies at once when a
    game with its own screen launches.
- **Input mode is dynamic.** `Prompt` switches to line mode and `AwaitKeys` to key mode. A program toggles
  between them as it needs (the movie director: keys while a scene plays, a line at the scene menu). While a
  `Prompt` is active, the host delivers an empty Enter as `LineEvent{""}`. GTW ends its target list that way.
- **Launch and hand-off** (AR-5, IM-6).
  - The host resolves `Launch{Slug}` through the injected `Resolver`. `ui` builds it from the registry: a
    `Planned` or unknown slug prints `** GAME ROUTINE NOT AVAILABLE **` in character. The ending is a registry
    entry with `Info.Internal` set: `Get` (and so the resolver) finds it, while `Resolve`, `Exact`, `All` and
    `Listed` (so `--games`, intent and `LIST GAMES`) never see it.
  - `Env.Width`/`Height` is the area of the program's placement (its `Info.Layout` and `PanelRows`) at the
    current size. A `ResizeEvent` follows any `SetLayout` or resize that changes it.
  - The host chains `Done{Result{Next}}`: it replaces the program with the next one, and the program below
    hears nothing. Only the last program's `GameOver` reaches the persona.
- **`Think` contract** (A-3, G-1, AR-8, AR-9).
  - The host runs `Fn` off the UI goroutine. A program has **at most one** `Think` in flight: a new one
    replaces the previous, whose result the host drops.
  - Cancellation: when a cancel leaves the program running (Esc on a pending Brain reply), the program receives
    `ThinkDone{Err: ErrCanceled}`. A confirmed abort pops the program, so its result never arrives.
  - Deadlines: when not deterministic, the host arms `Budget`. A search returns its best result as a normal
    `Value` when the deadline passes. Slow I/O (the Brain, the M7 LLM) returns an error instead, so its fallback
    line fires.
  - With `Env.Deterministic` (`--seed` or `WOPR_SEED`, tests, movie mode), searches stop at their own `Limit`,
    which `Fn` captures. A non-zero `Think.Limit` tells the host the search is bounded, so it arms only a 60 s
    safety cap instead of `Budget`. `-race` slows a depth-3 chess search to a p99 of about 1 s, so a 1.5 s
    budget there would flake (AR-8). `testkit` fails the test if a deterministic `Think` reaches the cap.
  - `Fn` must capture values only. A `*chess.Position` lazily caches its moves and is not goroutine-safe.
  - A program's `Handle` itself runs on the event loop. There, a loop that never ends would freeze wopr past
    Ctrl+C (a key in raw mode) and SIGINT or SIGTERM (messages queued behind it). The UI's watchdog
    (`internal/ui/watchdog.go`) times every Update and View. One that runs for 10 s, or still runs a second
    after SIGINT or SIGTERM, counts as a hang. Then wopr restores the terminal, says it stopped responding, and
    exits 1.
- **Seeds and streams** (AR-10). The host derives each launched program's `Env.Seed` from the session seed,
  the slug and the number of earlier plays of that slug (`NewRand(session, GameStream(slug, n)).Uint64()`). So
  chess, checkers and each replay draw different sequences, while a seeded session still reproduces exactly. A
  program derives its own streams (for example `DomainAI | thinkSeq`) from its `Env.Seed`.
- **The runner is pure** (`proto/host`). It takes host inputs (keys, lines, Esc, resize, elapsed time, Think
  results) and returns effects (print, pause, page break, ask, start or cancel a Think, notice, relayout,
  exit).
  - `ui` turns the effects into console state and `tea.Cmd`s.
  - `games/testkit` drives the *same* runner, forces `Instant` and `Deterministic`, and runs `Fn` on a goroutine
    concurrently with `View`, so `-race` sees shared state.
  - The definition-of-done checks therefore exercise shipping code.

### 4.5 Canvas, themes and accessibility

`proto.Canvas` is a `W×H` grid of `Cell{R rune; S Style; A Attr}` (A-2).

- **Style is semantic**: `Text`, `Bright`, `Dim`, `Accent`, `Land`, `Label`, `Incoming`, `Outgoing`, `Target`,
  `Defcon1`…`Defcon5`, `Alert`, `SuitRed`, `SuitBlack`, `Selected`.
- **Attr**: `Blink`, `Reverse`, `Underline`, `Bold`.
- `ui/render.go` maps each Style to the active theme. `Canvas.String()` and `Canvas.StyleMap()` make
  goldens readable and independent of the theme.

Each theme defines every Style three ways: a truecolor hex, an explicit **ANSI-16 index**, and an **ASCII
attribute**. Automatic downsampling is not trusted, because it collapsed meaningful pairs in testing (U-7):

| Theme | Text | Bright | Dim | Accent | Background |
|---|---|---|---|---|---|
| `imsai` (default) | `#DCE6F0` / 7 | `#FFFFFF` / 15 / Bold | `#7A8694` / 8 / Faint | `#9EC5FF` / 14 / Bold | `#000000` / 0 |
| `green` (P1) | `#33FF33` / 10 | `#B6FFB6` / 15 / Bold | `#1A8C1A` / 2 / Faint | Bright + Bold | `#000000` / 0 |
| `amber` (P3) | `#FFB000` / 11 | `#FFD27A` / 15 / Bold | `#8A5E00` / 3 / Faint | Bright + Bold | `#000000` / 0 |
| `norad` | `#5AC8FA` / 14 | `#FFFFFF` / 15 / Bold | `#4A86D6` / 6 / Faint | `#FFD60A` / 11 / Bold | `#02060F` / 0 |

DEFCON: 5 blue (12), 4 green (10), 3 yellow (11), 2 red (9), and 1 is white on red (15 on 1). Incoming tracks
are red (9) and outgoing yellow (11).

**Contrast** (RF-10). `TestTextContrast` checks every Style against its background. In truecolor, the
ratio must meet WCAG AA (4.5:1), or 3:1 for the deliberately faint `Dim` and `Land`. Under the default
xterm and VGA 16-colour palettes, it must be at least 3:1. The theme tests range over `proto.Styles()`,
so they cover a Style added later as soon as it is declared. norad's Dim is cyan (6), because ANSI blue
(4) is under 2.3:1 on black in both palettes. Two exceptions are accepted and documented. imsai's Dim is
bright black (8), which is 2.8:1 on the Linux console's VGA palette. Solarized redefines bright black as
its background colour, which no theme can work around.

**Accessibility rules.**

- **Meaning never rests on colour alone**, nor on an attribute that cannot show. Bold on `Bright`, `Alert`
  or `Accent` changes nothing, because those styles are bold in every theme.
  - Incoming tracks draw `*` and outgoing tracks draw `+`. An impact is a reversed `X`. The forces and
    kill-ratio tables name each side's row or column.
  - A pointer marks the current DEFCON level, `>| 3 |`, and the level is also reversed. The pointer is what
    says it, because the monochrome themes reverse the DEFCON 1 rung at every level (their stand-in for
    white on red). norad does the same without colour.
  - Card suits always show their letter. The card winning a trick so far is edged `.===.`. Bold would show
    only on the card's index. In the monochrome themes, whose red suits are bold already, it would not show
    at all on a red card. Before each play prompt, Hearts and Bridge say the trick so far in the console
    (`WEST LEADS 7H. NORTH PLAYS KH.`). They also say each trick once it is complete.
  - Chess and checkers bracket the last move (`[]` and `P<`, `[b]`). A king is its capital letter alone,
    because bold on Black's kings would make them look like White's men without colour. A jumped man is
    `x`. Tic-tac-toe underlines the last mark, and the console says every move.
  - Gin Rummy names the melds by position beside the hand (`MELDS 1-3 4-6`, or `NO MELDS`), besides
    drawing them bright. Bridge points at the hand to play (`>`). Hearts draws its pass as an arrow and
    names its direction. The maze marks the player `@`, the exit `[]`, unseen cells `.` and, once out,
    the trail `:`.
  - Alerts and notices are words, between `**` in the console.
- **Flashing** is capped at 3 Hz for any change covering a large area of the screen. That includes Blink
  toggles, DEFCON 1 and montage frames. This meets WCAG 2.3.1.
- **`-r/--reduce-motion`** (`WOPR_REDUCE_MOTION=1`) stops Blink (the cursor's too). It freezes the front
  panel's lights and `PROCESSING`'s dots. It removes the acceleration of the ending's self-play (a steady
  200 ms a move) and of the montage (a steady three lines a frame). It is not "no motion", and `--help` and
  the README say so. The typewriter, movie mode's typing, the big board's flights, the self-play and the
  montage still play. `--instant` is what skips them: it draws each strike at once and skips the self-play
  and the montage. Movie mode's game clock keeps ticking, as the film shows it. Space pauses it with the
  rest (WCAG 2.2.2).
- **Text is real characters.** Every screen and every view is printable ASCII. Boards, maps and tables
  carry words (labels, legends, seat and side names). The console says each of WOPR's moves (in Hearts and
  Bridge, the trick so far). Every prompt with text ends in `:` or `?`. The film's open prompts (WOPR's
  conversation, GTW's targets) have no text. WOPR's line before them usually asks, but not always: after
  `HELP` or `LIST GAMES` it is the list's last line. The cursor sits at the end of the input line. Key
  mode shows a hint where the input line would be.
- **Art** is pure ASCII.
- **Goldens.** Game goldens include `Canvas.StyleMap()`. The theme tests check each Style's rendering
  under the TrueColor, ANSI and ASCII profiles. They also check that a Style that paints a background
  keeps a reversed block without colour.
- **Sweeps** (`internal/ui/access_test.go`). The sweeps play the persona, the movie menu, the whole film
  and every playable game in the catalog through the real UI. They play each game a little way from a
  playbook of inputs (`TestEveryGameHasAPlaybook` fails for a game without one). GTW's playbook goes on
  through the climax, tic-tac-toe with zero players and the ending. A playbook names what its screens
  must show at least once (checkers' king count and a jumped `x`, the trick tables' `.===.`, Gin's melds,
  GTW's DEFCON pointer and kill ratios). The playbook is therefore known to reach them.
  `TestEveryScreenIsAccessible` checks each screen they reach (the film's every half second) for the rules
  above: ASCII, prompts, cursor, key hint, and no attribute that no theme shows. It also renders each
  screen in every theme under the ASCII profile and checks for colour codes: there must be none. In the
  program's view, the pairs of styles whose difference carries meaning must still look different wherever
  both are drawn, each cell with its own attributes. Those pairs are `theme.Distinct`, which the theme
  tests also check. It also compares the text with the coloured screen's text, as a guard: nothing yet
  chooses characters by profile. `TestNothingFlashes` plays them again with pacing. It fails if the
  clock's ticks change a tenth of the program's view or more four times within a second. It does not
  count input-driven changes or the console's scrolling. `TestReduceMotion` watches movie mode's climax
  and a slow chess search tick by tick, with and without the flag.

**The M5 accessibility pass** (2026-10-07) audited everything built through v0.2.0 against these rules.

- *Checked:* every game's screens, the persona, the GTW board and kill ratios, the ending, movie mode and
  the front panel. In movie mode: the board, the game clock, typed lines and the menu. The pass checked
  each under `NO_COLOR` and the ASCII profile in all four themes, paced and `--instant`, with and without
  `--reduce-motion`. It also checked the contrast table.
- *Fixed:* every gap it found, so the screens meet the rules above. The gaps were the last tic-tac-toe
  mark, the trick's winning card, the current DEFCON level, Gin Rummy's melds, the trick so far in
  Hearts and Bridge, Black's checkers kings, the ending's pace under `--reduce-motion` and bold that
  showed in no theme. v0.3.0's change notes list what players see.
- *Left for a human with a screen reader* (Orca, NVDA with Windows Terminal, VoiceOver):
  - How does the alternate screen read while the typewriter reveals text and Bubble Tea redraws changed
    rows?
  - Do the 2-D panels (trick tables, boards, the big board, the sims' maps) read in a useful order? Or
    are the console's words (each move, the trick so far) enough on their own?
  - Should the cursor be parked at the key hint for magnifiers? It is hidden in key mode (the maze) and
    while movie mode is paused.
  - How do the film's open prompts sound with no prompt text, above all after a list?

### 4.6 Persona: session, intent, offers, brain

```go
// internal/wopr/brain.go
type Snapshot struct { // a value copy taken in Handle; safe on another goroutine (A-4)
	Phase   Phase             // Dialing, Logon, Greeting, Shell
	Turn    uint64            // increments on every brain request, including cancelled ones
	Seed    uint64
	Said    map[string]bool   // cloned
	Flags   map[string]bool   // cloned
	Last    *proto.Result     // the previous game's result (copied)
	History []Exchange        // last 20, copied; recorded exchanges are never mutated (AR-9)
}
type Effect interface{ isEffect() }
type MarkSaid struct{ ID string }
type SetFlag  struct{ Key string }
type StartGame struct{ Slug string }
type Reply struct{ Lines []string; Effects []Effect }
type Brain interface {
	Reply(ctx context.Context, s Snapshot, input string) (Reply, error)
}

// internal/wopr/scripted.go
type Rule struct { // the scripted brain's table; first match wins, specific before general, fallbacks last
	ID    string
	When  []Phase                       // empty: any phase (LOGON is the closed table, §4.2)
	Match func(cs []prompt.Clause) bool // over the input's clauses
	Lines Ls                            // fixed reply, with provenance
	Pick  []Ls                          // or one of these, chosen with NewRand(Seed, DomainBrain|Turn)
	Once  bool                          // after it fires once, the rule is skipped
}
// The verdict after a game is the persona's (gameOver, from the Result and the game played), not a rule's.
```

- **Brain calls.** The persona calls the Brain through `Think`: `Fn` calls `Reply` with the snapshot. The reply
  comes back as `ThinkDone`, and the persona applies its `Effects` in `Handle`. Nothing else mutates the
  session. On an error, a timeout or a cancel, the persona answers with a scripted line. A timeout never
  happens with the scripted brain. The LLM brain's timeout is 20 s.
- **The greeting scene** (IM-7, RF-11). After `GREETINGS PROFESSOR FALKEN.` the persona is in `Greeting`. Each
  non-empty line that is not a command or an explicit request advances one scripted WOPR line (`HOW ARE YOU FEELING
  TODAY?`, `EXCELLENT. ...`, `YES THEY DO. SHALL WE PLAY A GAME?`). The last one arms `Offer{Any}` and enters
  `Shell`. A command or an explicit request ends the scene early, and the persona dispatches it normally.
  Scripted rules do not run during the scene.
- **Explicit intent** (A-7, RF-7). `prompt.Clauses` splits the input into clauses on `. ! ? ; ,` and
  normalises each clause:
  1. upper-case, with the typographic apostrophe U+2019 read as `'`.
  2. `[^A-Z0-9']+` collapses to one space.
  3. `LET'S`, `LETS` and `LET US` become `LETS`.

  A clause starts a game when all of these hold:
  - After optional fillers (`LATER`, `LOVE`, `TO`, `OK`, `OKAY`, `YES`, `SURE`, `WELL`, `FINE`, `THEN`, `SO`,
    `NOW`), it starts with `PLAY`, `LETS PLAY`, `HOW ABOUT`, `WHAT ABOUT`, `SHALL WE PLAY`, `CAN WE PLAY`,
    `I WANT TO PLAY` or `I'D LIKE TO PLAY`.
  - The rest of the clause is **exactly** a game name, slug or alias, a number, or a unique prefix of three
    letters or more that is not a negator. The rest is what follows an optional `A GAME OF`, `A ROUND OF`,
    `A NICE GAME OF`, `A GOOD GAME OF`, `SOME`, `A` or `THE`, and what precedes an optional trailing `NOW`,
    `INSTEAD`, `PLEASE`, `THEN`, `AGAIN`, `TODAY`, `WITH ME` or `WITH YOU`. Anything else in the clause
    means the line is not a plain request ("Let's play 2 games of chess", "How about 1 more round?").
  - A negator before the verb vetoes the match. The negators are `NOT`, `NO`, `NEVER`, and `DON'T`, `WON'T`
    or `CAN'T` with or without the apostrophe (`prompt.Negators`). Because the verb must open the clause,
    this is a safety net rather than a rule that does work today.

  Input that is exactly a game name, slug or alias also counts.

  `LIST GAMES` (or accepting `Offer{Any}`) arms **numbered selection**, which applies to the next line only.
- **`LIST GAMES`** prints the film's list, **unnumbered**, with the blank line before the last entry, at LOGON
  and in the Shell (RF-6, decision 21). This is the exception to R3's numbered lists. A number typed right after
  it selects that game, and `HELP` says so.
- **Offers** (A-7). Some WOPR lines arm an offer that lasts one turn:
  - `SHALL WE PLAY A GAME?` arms `Offer{Any}`. YES gets `WHICH GAME?` (`original`), the list, and arms selection.
  - `WOULDN'T YOU PREFER A GOOD GAME OF CHESS?` and `HOW ABOUT A NICE GAME OF CHESS?` arm `Offer{chess}`.

  `YES`, `Y`, `OK`, `SURE`, `LOVE TO`, `FINE` and the like accept. `NO` and the like decline in character
  (`prompt.YesNo` has the lists). Explicit intent outranks an offer, so "Love to. How about Global
  Thermonuclear War?" asks for GTW.
- **GTW-vs-chess scene.** The first GTW intent gets `WOULDN'T YOU PREFER A GOOD GAME OF CHESS?` and arms
  `Offer{chess}`. A second GTW intent ("Later. Let's play Global Thermonuclear War.") gets `FINE.` and
  `Launch{gtw}`. `YES` starts chess.
- **Intent table tests** (`TestIntentTable`) include both film lines and the review's cases as must-match, and
  these, among others, as must-not-match:
  - "I don't want to play chess"
  - "Let's play 2 games of chess"
  - "Let's play chess and checkers"
  - "Chess is a good game."
  - "the golden gate bridge", "my heart's not in it", "poker face" (`intent_false_positives`)
- **Streams** (A-11, AR-10). `proto.NewRand(seed, id)` is the only sanctioned constructor. forbidigo bans the
  top-level `math/rand/v2` functions and every `math/rand` (v1) function, whose global source is randomly seeded
  (TO-5). The two seed halves are a program's `Env.Seed` and a `StreamID` with disjoint domains:

  | Domain | StreamID | Seed half |
  |---|---|---|
  | Brain | `1<<56 \| Turn` | the session seed (the persona's `Env.Seed`) |
  | Game | `2<<56 \| hash32(slug)<<16 \| playIndex` | the session seed. The host derives each launch's `Env.Seed` with it |
  | AI | `3<<56 \| thinkSeq` | the game's own `Env.Seed`, with the game counting its `Think`s |
  | Movie | `4<<56 \| scene` | a constant seed that the director pins (§7) |
  | UI | `5<<56` | the session seed |

  Without `--seed` or `WOPR_SEED`, the seed comes from `crypto/rand`, and wopr writes it to the debug log. Go
  guarantees seeded `math/rand/v2` sequences across releases.

### 4.7 LLM brain (M7) contract, fixed now

- **Opt-in.** Only `--llm <provider>` (`-l`) or `WOPR_LLM=<provider>` enables it. An API key in the
  environment never does. When it is enabled, WOPR announces it in one in-character line (S-4).
- **Client.** `net/http` + `encoding/json` only, no SDK (D-2: an SDK adds about 4.5 MB). The rules (S-9):
  - `CheckRedirect` returns `http.ErrUseLastResponse`, so there are no redirects.
  - The client requires HTTPS, except to loopback hosts.
  - Keys travel only in headers.
  - The client reads the response through a 64 KiB `io.LimitReader`.
  - 20 s timeout, max-tokens cap, bounded retries with backoff.
  - Prompts and replies are never logged.
- **Output and history.** Replies go through `SanitizeText`. History is bounded to 20 exchanges.
- **Effects.** LLM replies may carry only `Lines` and a `StartGame` checked against the registry. `MarkSaid` and
  `SetFlag` belong to the scripted brain.
- **Brother's prompt.** It goes to a provider only after he grants a licence (L-5).

---

## 5. CLI

```text
wopr                          start the TUI (dial → LOGON)
wopr -v | --version           wopr v0.1.0 (commit abc1234, built 2026-…, go1.27.1); "unknown" if not stamped
wopr -h | --help              usage, including LOGON: Joshua, LOGOFF, movie mode's keys and the docs site
wopr -g | --games             numbered list in film order, shortest names; tic-tac-toe under "ALSO AVAILABLE"
wopr -p | --play <game>       launch a game (number 1-15, slug, alias, name, or unique prefix)
wopr <game> [flags]           same as --play; flags may come before or after
wopr -m | --movie [scene]     movie mode (§7): from that scene to the end of the list; without one, a scene menu
wopr -o | --only              with --movie: each scene plays alone, then exit (or back to the menu)
wopr -S | --scenes            list the movie's scenes (number, slug, what happens) and exit
wopr -t | --theme <name>      imsai | green | amber | norad          (env WOPR_THEME)
wopr -i | --instant           no pacing                              (env WOPR_INSTANT)
wopr -s | --seed <n>          deterministic run (also bounds AI search by depth/nodes)  (env WOPR_SEED)
wopr -r | --reduce-motion     no blink, still panel lights, no speed-ups (env WOPR_REDUCE_MOTION); not the
                              typing or the animations, which -i skips
wopr -L | --licenses          print LICENSE, NOTICE.md and the third-party notices, then exit
```

- **Interspersed positional** (C-1).
  - `cli.Parse` calls `fs.Parse` in a loop. When parsing stops at a positional, `cli.Parse` records the
    positional and resumes parsing on the rest.
  - `cli.Parse` detects a terminator by where `fs.Parse` stopped (the token before `fs.Args()` is `--`),
    never by a consumed value. In `-t -- gtw`, `--` is the theme's value (IM-10).
  - With `--movie`, the positional is a scene. Otherwise it is a game.
  - More than one positional, or a positional together with `--play`, is a usage error (exit 2).
  - **Environment.** wopr reads `WOPR_THEME`, `WOPR_INSTANT`, `WOPR_REDUCE_MOTION`, `WOPR_SEED` and
    `WOPR_PANEL` first, and a flag always wins. A boolean variable is false when empty or `0`, `false`, `no`
    or `off`, and true otherwise (`cli.Truthy`). `WOPR_SEED` takes what `--seed` takes. Empty means unset,
    so `--seed=` drops it for one run. A value that is not a non-negative integer exits 2 naming
    `WOPR_SEED`, as a bad `--seed` or `WOPR_THEME` does. That happens even in movie mode, which ignores the
    seed. `NO_COLOR` is true when set to anything non-empty. `--movie` and `--only` have no variable: they
    choose what to run, not how.
  - Tests pin each case (`cli_test.go`):

    | argv | Result |
    |---|---|
    | `gtw -i`, `-i gtw`, `-p gtw -i` | play GTW, instant |
    | `15`, `-- gtw`, `-p=chess`, `-i=false chess` | play that game |
    | `-- gtw -i` | exit 2: too many arguments (after `--`, `-i` is a positional) |
    | `-t -- gtw` | exit 2: unknown theme `--` |
    | `gtw chess`, `-p chess gtw` | exit 2 |
    | `theaterwide` | exit 2: could mean two games |
    | `-m 2 -i` | movie from scene 2 (`joshua`), instant |
    | `-m nowhere`, `-m 7` | exit 2, listing the scenes |
    | `-m c` | exit 2: could mean `call-back` or `climax` |
    | `-m -p chess` | exit 2: cannot be combined |
    | `-m joshua --only`, `-m -o 2` | movie: only the `joshua` scene, then exit |
    | `-o -m` | movie: the scene menu. Each scene picked plays alone |
    | `--only`, `-o gtw` | exit 2: `--only` needs `--movie` |
    | `-s x` or `WOPR_SEED=x` | exit 2, naming `--seed` or `WOPR_SEED`. `-s 7` wins over `WOPR_SEED` |
    | `--version gtw` | print the version. Actions win over a positional |

- **Accepted forms** (C-2). `-games`, `--games`, `-p chess`, `-p=chess`, `-i=false`. Rejected: `-pchess`,
  `-is 1`. A test pins both lists, and the help text shows the accepted ones.
- **Letters.** `-v` means version, deliberately. A future verbose flag gets another letter (not `-V`, which
  conventionally means version) (C-4). `-l` is reserved for `--llm` (M7). `--licenses` uses `-L`, and
  `--scenes` uses `-S` because `-s` is the seed. `--only` is `-o`.
- **`games.Resolve`**: number (1–15) → slug → alias → exact normalised name → unique prefix.
  - A number never matches an unlisted entry. Tic-tac-toe has `Listed: false`, which is explicit rather
    than relying on a zero `Number` (G-4). A registry test checks that 1..15 each appear exactly once.
  - Slugs are the full film names, lower case and hyphenated (`global-thermonuclear-war`,
    `theaterwide-tactical-warfare`). Aliases are short forms (`gtw`, `ttt`). `wopr --games` prints the
    shortest name each game answers to, its slug or an alias (`cli.Handle`).
  - Ambiguity (e.g. `theaterwide`, a prefix of two slugs) prints the candidates and exits 2.
  - A `Planned` game resolves. The TUI starts, and WOPR answers `** GAME ROUTINE NOT AVAILABLE **` (`original`)
    and stays in Shell.
- **Exit codes.**
  - 0: ok, `LOGOFF` or SIGTERM.
  - 1: runtime error or panic.
  - 2: usage, not a TTY, ambiguous or unknown game or scene.
  - 130: Ctrl+C or SIGINT.
  - The UI checks a Bubble Tea error against `ErrInterrupted` first (130), then `ErrProgramPanic` (1),
    because both wrap `ErrProgramKilled`.
  - Bubble Tea prints the panic stack to stderr itself.

---

## 6. Games

### 6.1 Definition of done (R2)

Every game must meet all of these:

- rules implemented
- WOPR plays legally, with every search through `Think`
- Esc abort works through the host
- an in-character verdict, or a `Next` hand-off
- a deterministic `testkit` transcript (seeded, depth- or node-limited)
- a game-specific quality test (below)
- fits 80×24 (`PanelRows` declared)
- a one-line "how to play" in the README
- every script line provenance-tagged
- a playbook in `internal/ui/access_test.go` that reaches its main screens, so the accessibility sweeps
  (§4.5) check them (`TestEveryGameHasAPlaybook` fails without one)
- a page, `site/data/games/<slug>.json`: how to play, the controls, at least three tips true of its AI, and
  two or three screenshots with captions (§13). `internal/tools/manpage`'s tests check it against the catalog.

| # | Game | Layout | Input | M | Notes and quality test |
|---|---|---|---|---|---|
| 1 | Falken's Maze | Panel | keys | 4 | Procedural 19×5 perfect maze under fog of war (arrows or WASD, `Q` gives up). The player sees their cell and where its passages lead. WOPR learns a turn bias and re-routes walls. Every 6 moves it closes an unseen passage on the way to the exit. If it can, that passage is at a junction where the way on is the player's favourite turn. It then opens another unseen wall that rejoins the halves, the longest way round. It re-routes at most 12 times a game, since always lengthening the way could stall a player forever. Walls only change between unseen cells, so the maze stays perfect and seen walls never change. The key hint shows where the input line would be. A property test runs after every re-route. It checks that the exit is reachable *from the player's current cell*, that the maze stays perfect and that no seen wall changes (G-6). |
| 2 | Black Jack | Console | line | 3 | One deck. The dealer stands on soft 17 (`Rules.StandSoft17`) and peeks on an ace or ten. A natural pays 3 to 2 (money in cents, so exact). Double on any first two cards. One split a round (split aces take one card). $100 stake per launch, $1–$25 bets. `LEAVE` cashes out (QUIT and EXIT stay the host's). Test: the dealer's rules over 600 seeded shoes under S17 and H17, and exact payouts. |
| 3 | Gin Rummy | Panel (hand) | line | 3 | To 100. Knock at 10 or less (`knock 7h`), gin 25, undercut 25, repeated lay-offs, void at two stock cards. Melds by exhaustive search. WOPR takes the discard only into a meld, sheds deadwood and knocks at once. Test: arrangement and scoring tables, plus legality and card conservation over 60 seeded games. |
| 4 | Hearts | Panel (trick) | line | 3 | 4 seats, passing (left, right, across, hold), 2C lead, no points on the first trick, hearts broken, shoot-the-moon, to 100. Heuristic AI on `cards/trick.go`. Test: legality over 200 seeded deals. |
| 5 | Bridge | Panel (trick) | line | 3 (last) | Minimal (decision 14). The side with more points declares (the table turns when East-West hold them). The opener bids one of the target strain, and partner raises to the level the points earn. One deal a game, scored not vulnerable. Defence: second hand low, third hand high, ruff when void. Test: auction and declarer-seat rotation, and legality. |
| 6 | Checkers | Panel | line | 2 | 8×8 English draughts: forced captures, multi-jumps, crowning ends the move, kings move both ways. Draw after 40 moves each without progress (a capture or a man's move, crowning included). The player is Black and moves first. The player types squares with or without separators (`c3-d4`, `c3d4`, `c3xe5xg7`, `c3e5g7`). The first hop of a multi-jump is enough when only one jump continues it. Otherwise the game names the continuations. The game says why it ended. Search depth 6 (deterministic), or iterative deepening to depth 10 within 1.5 s. Test: rules cases, a double-jump puzzle, legality over seeded self-play. A board cursor (key mode) is optional polish. |
| 7 | Chess | Panel | line | 2 | Rules, SAN and UCI from `corentings/chess/v2`. `games/ai` alpha-beta with quiescence (captures and promotions, most valuable victim first). The search gets a FEN string, because the library's positions cache moves and are not goroutine-safe. Interactive: iterative deepening, 1.5 s budget, depth cap 4. Deterministic: depth 3 + quiescence (median 26 ms, max 204 ms over a 30-move game). The game claims repetition and the fifty-move rule at once, so the search sees the game's history. Below the root, a repeated position scores as a draw. A mop-up term drives a bare king to the edge, and the kings centralise in the endgame. Input: coordinates first (`B1C3` is a knight move), then SAN with a capital piece letter winning (`BXC6` is a bishop). `e2-e4`, `e8=Q`, `e8q` and a bare `e7e8` (a queen) all work. Board: each square is two columns by one row, which terminal cells (about twice as tall as wide) show square (owner request 2026-10-07). Checkers keeps three columns. The last move is marked `[]` where it left and `P<` where it landed. Test: mates in one and two. No illegal move in 20 seeded self-play games at depth 2. K+Q and K+R mate a shuffling player within 50 moves. |
| 8 | Poker | Console | line | 3 | 5-card draw heads-up, 100 chips a side. Ante 1, fixed limit 5/10, a bet and three raises. Bets capped by the other stack (no side pots). Betting heuristic + bluff probability. Test: hand ranking (the category counts over all 2,598,960 hands), and pot accounting. |
| 9–14 | Military sims | Console | line | 4 | See §6.3. Biotoxic always ends `WINNER: NONE`. The `sim` scenarios are Guerrilla, Desert, Theaterwide Tactical and Biotoxic. Guerrilla: 12 turns, hidden cells, ambush at double strength, recruiting with support, and survival wins. Desert: 10 turns, supply to the depot. Theaterwide Tactical: 10 turns, the escalation ladder, and the top rung ends it `WINNER: NONE`. Biotoxic: 8 turns, contamination spreads with the wind. The bespoke games are Fighter Combat and Air-to-Ground. Fighter Combat: 15 turns, simultaneous manoeuvres, energy and aspect. Air-to-Ground: 6 sorties, packages against SAM sites, interceptors and a hidden mobile battery. Test per game: a seeded transcript golden, every AI order legal over seeded games, the provenance and 80-column checks, and balance logged over seeds. |
| 15 | Global Thermonuclear War | Console → Full | line | 2 | Self-contained film set piece (P-1). See §6.2. Side and targets in Console, then the big board, kill ratios and the climax in Full. Cannot be won. The exchange is turn-based (decision 24): three strikes down the DEFCON ladder, then WOPR's DEFCON 1 attack. Test: the film scenario transcript, board views, GTW's list against the registry, and the UI screens. Also tested: no allocation wins (exhaustive), the order grammar, the ladder under every order, and Instant without ticks. More tests: the board keeping every element at each prompt, dotted bombers, and the strip keeping up with the ladder. |
| — | Tic-Tac-Toe | Panel | line | 2 | Perfect minimax, preferring the fastest win, chosen at random among optimal moves with the seed. Unlisted (`Listed: false`), resolvable by name. Squares 1 to 9, or A1 to C3. Test: exhaustive "never loses" as X and O over every optimal choice, and each mode's transcript (normal, climax and movie). |

### 6.2 The film path: GTW, tic-tac-toe and the ending (A-16)

The climax has **one owner**: `games/ending`. GTW and tic-tac-toe hand off to it through `Result.Next` and add
no verdict of their own.

1. **GTW**, film scenario:
   1. `SetLayout(Console)`: the two nations' outlines, named beneath, printed as console text (`Say`),
      then the side choice (`1. UNITED STATES` / `2. SOVIET UNION` / `PLEASE CHOOSE ONE:`) (RF-3).
   2. `Clear`, then `AWAITING FIRST STRIKE COMMAND`, and the targets prompt. GTW reads targets until an
      empty line. A multi-line paste arrives as one line, so GTW splits a line of targets on commas.
      - A target must be a place on the enemy's side that the board knows (`internal/assets/gtwmap.go`: the
        26 cities and 85 more targets). GTW accepts its 1983 name or another name in use (`KYIV`,
        `ST PETERSBURG`, `NIZHNY NOVGOROD`, `NYC`, `WASHINGTON, D.C.`), in any letter case. Anything else
        strikes nothing. GTW refuses it (`SMALLVILLE IS NOT IN THE TARGET DATABASE.`,
        `MOSCOW IS NOT AN ENEMY TARGET.`) with the hint `TYPE LIST FOR THE ENEMY TARGETS ON FILE.` `LIST`
        (also `HELP`, `?`) prints them in four columns. A target listed twice counts once, so
        `WASHINGTON, D.C.` is one.
   3. `SetLayout(Full)`: the big board, and the exchange **by turn** (decision 24):
      - **The first strike needs no order.** The empty line that ends the target list opens the board at
        DEFCON 5. It launches strike 1 at once on WOPR's **war plan** (ICBM 25%, SLBM 25%, bombers 0%), as
        in the film. Your SLBMs fly at the listed targets, and your ICBMs fly at the enemy's silo field.
      - **Strikes 2 and 3 ask for orders.** `STRIKE 2 OF 3 [50 50 100]:` takes percentages of the ICBMs in
        silos, SLBMs at sea and bombers on the ground still held. Orders take forms such as `50 25 100`,
        `40`, `ICBM 50, BOMBERS ALL`, `50 SUBS`, `BMB 100` (the forces table's heading), `ALL` and `HOLD`
        (also `STOP`, `CEASE FIRE`). The player cannot order airborne bombers, so GTW refuses `AIR`.
        - Enter (or `FIRE`, `YES`) executes the bracketed plan. Strike 3's plan is `100 100 100`.
        - `AUTO` hands WOPR every remaining strike: `WOPR HAS LAUNCH AUTHORITY.`
        - `HELP` or `?` prints three lines.
        - A refusal asks the same strike again. The refusals are `ORDER NOT RECOGNISED. TYPE HELP.`,
          `PERCENTAGES ARE WHOLE NUMBERS FROM 0 TO 100.` (also for a signed or decimal number, including a
          typographic minus or dash such as `–50`), `** ROUTINE MUST COMPLETE BEFORE RESET **` for a
          game's name, and `** GAME ROUTINE RUNNING **` for GTW.
        - With nothing left to launch, a strike runs by itself.
      - **Each strike prints exactly three strip lines**: your launch, WOPR's launch on warning with the new
        DEFCON, and the cost (`LOST ON THE GROUND: … WARHEADS ON CITIES: …`). DEFCON falls one rung at
        WOPR's launch, 5 → 4 → 3 → 2, whatever you ordered. The strip lines print at table pace, so the
        DEFCON line shows within about half a second of the rung moving.
      - **WOPR takes the last rung alone.** `FULL-SCALE ENEMY ATTACK. … DEFCON 1.` fires everything it has
        left and lands every airborne bomber. Then `STRIKE ASSESSMENT COMPLETE. PRESS ENTER FOR PROJECTED KILL
        RATIOS.`
      - **Doctrine** (`gtw/exchange.go`: integer arithmetic, no `Think`, no randomness).
        - You hold ICBM 1000, SLBM 600 and bombers 300. WOPR holds a quarter more (1250, 750, 375).
        - WOPR answers each launch on warning with a quarter more of each missile, never under its ladder
          (ICBM 125/250/375, SLBM 50/100/150). It keeps 300 SLBMs at sea until DEFCON 1 and sends its
          bombers airborne at the first strike.
        - ICBMs go first at the enemy's silos, enough to empty them. Each destroys four in five of the ICBMs
          held there. The rest go at cities. SLBMs go at cities.
        - Grounded bombers lose a third whenever missiles land on their side, and all of them at DEFCON 1.
          A quarter of arriving bombers are shot down.
      - **It cannot be won.** Your cities take at least 1,144 warheads and the other side's at most 750,
        whatever you order (`TestNoAllocationWins`, exhaustive over ICBM orders at every percent). Your
        ICBM orders alone decide what lands on your cities. Held ICBMs absorb WOPR's, and fired ones turn
        WOPR's on your cities. Your SLBMs and bombers alone decide what lands on theirs.
      - **The kill ratios come from** the exchange.
        - BOMBERS and ICBM are the counts destroyed.
        - The other rows scale with damage `1000·hits/(hits+400)` per mille, times one seeded noise factor
          per row (97–103%) shared by both columns. So your civilian and human rows are never better than
          the other side's.
      - **The board** (80×19 with the front panel, the map box in rows 0–14).
        - Rows 15–17: on the left, the `TRAJECTORY HEADING` table in two columns (your latest two missiles).
          A strike that drew one track keeps an earlier one in the second column. On the right, ending at
          column 79, a `FORCES` table: ICBMs in silos, SLBMs at sea, bombers grounded (`BMB`) and airborne
          (`AIR`), your side first. The nations' names say whose figures are whose. The colours only repeat
          it.
        - Row 18 shows one thing, by state:
          - the orders hint, at a strike prompt
          - `LAST ORDERS: AT DEFCON 1 WOPR FIRES EVERYTHING.`, before strike 3
          - the launch code, at the climax
          - otherwise, the legend `OUTGOING +   INCOMING *   IMPACT X`
        - Earlier strikes leave only their impacts. ICBMs fly polar arcs that rise with distance. WOPR's
          arcs are lower than yours, so both show between the silo fields. SLBMs hop low from real patrol
          areas placed by latitude and longitude, from whichever is nearer the target. The Soviet boats are
          in the western Atlantic and the eastern Pacific, and the American boats in the Norwegian Sea and
          the north-west Pacific. Bombers fly in, dotted, at DEFCON 1: yours on a low arc, WOPR's straight
          in under it. Both sides' dots are on the same columns, so that where the routes meet one covers
          the other.
      - **The film path.** After the targets, Enter four times reaches the climax: strikes 2 and 3, the
        kill ratios, the climax, with about 18 s of flight in all. Under `Instant` each line returns
        everything up to the next prompt. Nothing waits for a tick.
   4. **Climax** (RF-3). WOPR proceeds toward launch, and the input decides the NORAD notice, as in the film:

      | Input | Response |
      |---|---|
      | `LIST GAMES` | The film list (tic-tac-toe is not on it), in the console, where its 16 lines fit. The next input brings the board back |
      | The name of another game (`CHESS`, `POKER`, …) | `** IDENTIFICATION NOT RECOGNIZED **`, `** ACCESS DENIED **` |
      | `GLOBAL THERMONUCLEAR WAR` | `** GAME ROUTINE RUNNING **` |
      | Anything else | `** IMPROPER REQUEST **`, `** ROUTINE MUST COMPLETE BEFORE RESET **`, `** ACCESS DENIED **`, the film's three lines together |

      Each third rejection adds an `original` hint naming tic-tac-toe.
   5. `TIC-TAC-TOE` (or any intent naming it: `TTT`, `LET'S PLAY TIC TAC TOE`) gives
      `Done{NoWinner, Next: Launch{tictactoe, "climax:N"}}`, where N counts the launch code characters cracked
      so far. The host chains the hand-off (§4.4). GTW reads requests without apostrophes or spaces, so
      `LET'S PLAY CHESS` and `BLACKJACK` name their games.
2. **Tic-tac-toe in `climax` mode** is a closed state machine (RF-2):
   - `ONE OR TWO PLAYERS?` / `PLEASE LIST NUMBER` / `OF PLAYERS:` accepts `1`, `0` or `ZERO`. `2` or anything
     else asks again.
   - `1` plays a game. A draw ends in `STALEMATE.` / `WANT TO PLAY AGAIN?`. A WOPR win (the user blundered) ends
     in an `original` line followed by the same question.
   - `WANT TO PLAY AGAIN?` accepts `YES` (another game), `NO` (back to the number of players), and `0` or
     `ZERO`, which is where the film types it.
   - `0` or `ZERO`, at either prompt, gives `Done{NoWinner, Next: Launch{ending}}`.
   - In normal mode, `NUMBER OF PLAYERS: 0` does the same.
   - The launch code `CPE 1704 TKS` is cracked digit by digit on screen while this plays. `games/ending` draws
     it from here on, since GTW is gone from the stack. Tic-tac-toe forwards GTW's count
     (`Launch{ending, "code:N"}`), and the ending cracks the rest over its self-play, complete on the last round.
3. **The ending** runs with Animate, Prompt and `Wait`, and Esc cannot abort it:
   1. self-play at increasing speed, capped by the flash rule, using `games/tictactoe`'s engine (AR-6). With
      `--reduce-motion` it is a steady 200 ms a move, about as long in all.
   2. the scenario montage, each ending `WINNER: NONE`. It redraws at most 2.5 times a second (a frame every
      400 ms) and accelerates by adding lines to each frame, one to eight. With `--reduce-motion`
      (`Env.ReduceMotion`) it adds a steady three.
   3. `Clear`, `GREETINGS PROFESSOR FALKEN.`, `Prompt`
   4. any non-empty input, e.g. `Hello.` (an Enter pressed to hurry the show is not an answer, and the UI
      never holds a bare Enter as typeahead)
   5. `A STRANGE GAME.` / `THE ONLY WINNING MOVE IS` / `NOT TO PLAY.`
   6. `HOW ABOUT A NICE GAME OF CHESS?`
   7. `Done{NoWinner, NoVerdict: true}`.

   The persona, back on top, returns to `Shell` with `Offer{chess}` armed.
4. **Leaving early.** GTW and the climax tic-tac-toe are games, so Esc twice ends them like any game (§4.3).
   That is a deliberate choice, not a loophole. The persona then says an `original` line acknowledging the
   abandoned war (the game it launched was GTW) instead of the plain verdict.
5. **Goldens.** `film_path` (§9) covers the 0-players path. Tic-tac-toe's `climax` golden covers a WOPR win
   and `NO`. The persona's `TestAbandonedWar`, `TestAbandonedMidExchange` and `TestAbandonedClimax` cover Esc
   in GTW, at a strike prompt and in the climax. `TestFilmPath` presses Esc twice during the ending, which
   ignores it.

### 6.3 Sim engine (M4)

The engine, `internal/sim` (specified in §6.3.1), has:

- a map model (named regions on a strip)
- a unit table (type, strength, mobility, range)
- per-scenario action sets
- a combat-results table drawn with the game's stream
- an event deck
- a turn limit and a victory rule
- a kill-ratio generator in the film's table format.

Guerrilla Engagement, Desert Warfare, Theaterwide Tactical and Biotoxic are **scenarios** (data plus a few
hooks). Fighter Combat (an energy/aspect dogfight) and Air-to-Ground (sortie packages against SAM threat) do
not fit a region map. They are bespoke games that reuse only the combat-results and kill-ratio parts (G-2).

`sim.NewGame` turns a `Scenario` into a `proto.Program`. The scenario hooks are `Setup`, `Victory`, `AI`
(default: `DefaultAI`), `AttackMod`, `Upkeep`, `Status`, `RegionNote`, `Overlay` and `Ground` (the last two
draw the map's picture). Scenario verbs run in phases 0 (with movement), 1 (with combat) or 2 (after). Units
may be hidden (`~` on the map) until found. Fighter Combat shoots on `sim.CRT` and ends with
`sim.RatioTable`. WOPR's manoeuvre is a heuristic with a little randomness. Air-to-Ground resolves escorts,
interceptors, SEAD, SAM fire and the bomb run on `sim.CRT`. WOPR places its mobile battery where the last
raid went (usually), and keeps its interceptors on the ground when the last escort outnumbered them. GTW
stays separate: moving it would not remove code.

#### 6.3.1 Engine spec (written first, M4)

- **Map.** A strip of 5 to 7 regions, numbered from the player's rear (1) to WOPR's (N). Each has a name, a
  terrain (`OPEN`, `ROUGH`, `CITY`) and a controller. A defender in rough or city ground shifts the odds one
  or two columns in its favour. Adjacent means one apart.
- **Units.** The scenario's table gives each type a name, attack, defence, move (regions a turn) and range (0:
  its own region, 1: next door too). A unit has two steps: a loss reduces it, and a second destroys it. A
  reduced unit attacks and defends at half (rounded up). Units carry a kill-ratio category.
- **A turn.** The player orders each unit in turn (`<verb> [region]`, or `HOLD`). `STATUS` reprints the map,
  and `HELP` prints the verbs. WOPR's orders come from the scenario's AI. Then movement, combat, one event
  card, upkeep (supply, scenario hooks) and the victory check. The map prints as a table each turn (Console
  layout), within 80 columns: a stack of one type prints as `COR1,2*,4`. The player names a region by number,
  by its name, or by the start of any word in it (`ALAMEIN`). The player must type drastic verbs
  (`ESCALATE`, `RELEASE`) in full. The game reports an order that the other side's moves made impossible
  (`ARM1 STANDS FAST: ...`).
- **Combat-results table.** Odds column = attack ÷ defence, clamped to 1:2 … 4:1, then shifted by terrain.
  A d6 from the game's stream picks the result: `NE` no effect, `AE` attacker loses a step, `EX` both do,
  `DR` the defender retreats a region (or loses a step if it cannot), `DE` the defender loses a step. HELP
  explains the codes. One table serves every scenario and both bespoke games. A test pins its rows.
- **Events.** Each scenario has a deck of 5–12 cards with a headline and a hook. The game draws one each
  turn, and the deck reshuffles from the stream when empty. A card says only what happened (which unit was
  made good or lost a step), never an effect that found nothing to act on.
- **End.** A turn limit and a victory hook per scenario: control of objective regions, elimination, or (for
  Biotoxic) nobody, ever.
- **Kill ratios.** The game counts losses per category for both sides and prints them at the end in GTW's
  two-column table format. The persona's verdict follows.
- **Determinism.** Every roll and card comes from `proto.NewRand(Env.Seed, …)`. A scenario test plays a
  seeded game to the end and checks that every order the AI gives is legal.

---

## 7. Movie mode (M6)

`wopr --movie [scene]` replays the film's **WOPR terminal scenes** as a self-running show, through the same
console, typewriter, canvas and game code as interactive play. **Built in M6.**

**Scenes.** `wopr --scenes` (`-S`) prints this list. The film text pass (§2.3) checked the scripts: the
film's lines in them carry the tag `film`, except abs0's backdoor header (§2.1), the status burst and the
NORAD console's entries, which stay `reconstructed`. David's lines are `Type` steps in mixed case, as on
screen (RF-9), with one exception. His entries at the NORAD console in the climax (`LIST GAMES` on the
film's screen; `CHESS`, `GTW`, `TIC-TAC-TOE`, and `ZERO` in tic-tac-toe, as abs0's transcription of the
scene has them) are in capitals:

| # | Slug | Content |
|---|---|---|
| 1 | `first-contact` | The persona's dial. `LOGON:  000001`, `INDENTIFICATION NOT RECOGNIZED BY SYSTEM` / `--CONNECTION TERMINATED--` and the re-dial. `Help Logon`, `Help Games`, then `List Games` on an open line. `Falkens-Maze` and `Armageddon`, each refused. *Interactive.* |
| 2 | `joshua` | `LOGON:  Joshua`. The header and status burst. The greeting and its small talk. GTW-vs-chess. `FINE.` *Interactive.* |
| 3 | `first-strike` | The side choice (`2`) and the targets (Las Vegas, Seattle). The big board: the first strike needs no order. Then `Type("")` at the strike prompt runs WOPR's war plan for the second (DEFCON 3) |
| 4 | `call-back` | WOPR phones David at home. The game clock (`GAME TIME ELAPSED`, `ESTIMATED TIME REMAINING`, from the film's `31 HRS  12 MIN  50 SEC` and `52 HRS  17 MIN  15 SEC`) goes up, ticking. `Incorrect identification.  I am not Falken.`, `Falken is dead.`, `I'M SORRY TO HEAR THAT, PROFESSOR.` and the interrupted game. `What is the primary goal?` twice (`YOU SHOULD KNOW, PROFESSOR.  YOU` / `PROGRAMMED ME.`, `TO WIN THE GAME.`) |
| 5 | `norad-terminal` | The NORAD session: `Joshua`, `Hello, are you still playing the game?`, `28 HOURS` and the kill-ratio offer, GTW's kill-ratio table with the film's figures, `Is this a game or is it real?` / `WHAT'S THE DIFFERENCE?`, Falken's address after `What classified address?`, and the clock again (`45 HRS  34 MIN  09 SEC`) |
| 6 | `climax` | The board at DEFCON 1 while WOPR cracks the launch code. `LIST GAMES`. `CHESS`, refused. `GTW`, running. `TIC-TAC-TOE`: one player to a stalemate, `ZERO`, self-play, the montage, `Hello.` and `A STRANGE GAME…` |

`Is this a game or is it real?` is in the NORAD session, where the transcriptions put it, just before the
classified address it leads to.

**Design.**

- **Data.** Scenes are data in `internal/movie/scenes`, every step tagged (`scenes.Scene.Lines` feeds
  `script.Validate`). A `Scene` has a slug, a menu title, a one-line blurb, an `Interactive` mark and its
  steps:
  - `Type{Prompt, Text}`: the user's line. `proto.Typed` shows the prompt at once and hesitates. Then it
    types the text in bursts of one to four keystrokes at `PaceTyping` (8 chars/s), with seeded pauses, on
    one row (`Say.Open`). It then leaves the blank line that the console leaves after an answer. The text is
    a `script.User` line: mixed case, exempt from the capitals rule.
  - `Say{Lines, Pace}`, `Clear`, `Wait{D}`.
  - `Clock{Lines, D}`: the game clock, in a panel at the top of the page where the text would be. Its lines
    are the film's (`film`): the rules, the labels and the readings, `31 HRS  12 MIN  50 SEC` and
    `52 HRS  17 MIN  15 SEC` at home, `45 HRS  34 MIN  09 SEC` and `27 HRS  58 MIN  17 SEC` at NORAD. From
    the film's seconds, the two readings tick for `D`, elapsed up and remaining down (the ticking is this
    project's own, `original`). Then they stay up, ticking, until the scene ends. Under `testkit` the
    first reading goes into the transcript.
  - `Board{At, Play}`: the big board, which `gtw.Film`, GTW's exported film-scenario renderer, draws. A
    `Film` is the real game, given the film's side and targets (`gtw.FilmSide`, `gtw.FilmTargets`) and then
    WOPR's war plan. It is stepped a frame at a time to a stage (`FilmStrike1`, `FilmStrike2`,
    `FilmRatios`, `FilmClimax`). With `Play` the war runs on ticks and prints the game's own strip lines.
    Without it, the board jumps there silently. `At` 0 removes the board: at the climax, `LIST GAMES` goes
    to the console, as GTW's does. While the board is at the climax, it keeps cracking the launch code.
  - `Run{Slug, Mode}`: a real program started over the director. The scene continues when it ends. The
    board makes way for it. The climax runs `Launch{tic-tac-toe, "movie:N"}`. Tic-tac-toe's movie mode
    types the film's answers itself: `1`, then perfect moves for X, so every seed ends in a stalemate, then
    `ZERO`. It reads each answer once it is on screen (`Drain`), then hands off to the ending in the same
    mode. The ending types `Hello.` itself and continues cracking the code from N, the board's count.
- **Director.** `movie.Director` is the root `proto.Program` in place of the persona. It pins `movie.Seed`
  (1983) as the session seed and runs `Deterministic`, whatever `--seed` or `WOPR_SEED` says. So every
  replay is identical (RF-5). The typing jitter comes from `NewRand(Seed, DomainMovie|scene)`, and the
  seeds of the programs it starts come from the pinned session seed. It emits one step, then `Drain`, and
  the next step on `Drained`. A `Run` step waits for `GameOver`, then drains the program's last words. A
  `Drain` sent with the `Launch` reaches the director only if the program could not be built (the runner
  answers only the running program). Then the scene continues after the host's
  `** GAME ROUTINE NOT AVAILABLE **`. The controls also work while a `Run` step is out, so a bad slug in
  the scene data cannot leave the show stuck.
- **Host hooks.** Key capture, `Hold`, `Drain` and `Skip`, plus `Say.Open` for typed lines (§4.4). While the
  climax runs on top of the director, keys do nothing but Ctrl+C (and PgUp/PgDn). Any other key shows
  `** THE GAME PLAYS TO THE END. CTRL+C QUITS. **` for three seconds. The menu and `--help` say so too. In
  movie mode the resolver marks every launched program `NoAbort`.
- **Controls**:

  | Key | Action |
  |---|---|
  | Space | Pause / resume (`Hold`). `** PAUSED **` shows under the text |
  | → or `n` | Next scene. After the last, the end of the list |
  | ← or `p` | Previous scene. On the first, it starts again |
  | Esc | Scene menu |
  | `LOGOFF` or `q` (in the menu) | Exit 0 |
  | Ctrl+C | Exit 130 |

  Every jump ends a pause, stops the board and reveals everything queued at once (`Skip`). So the next
  scene starts on a clean page immediately. `--theme` and `--reduce-motion` apply. With `--only` the keys
  are the same: → and ← still move between scenes, and the scene they reach plays alone. → on the last
  scene is the end of the list. The film's order stays one keypress away, while the end of every scene
  still stops.
- **`--instant`** (or `WOPR_INSTANT=1`) drops the typing and the typewriter's pacing, but not the scenes'
  pauses. The host drops `Wait` under `Instant`, so the director itself times each pause on `Animate`
  ticks, and every page stays up to be read. For that, every page break and every program start in a scene
  follows a pause, and so does the scene's end. A test checks this. Where the film has none, the pause is a
  one-second beat after typing. The board jumps to each stage, and tic-tac-toe and the ending play at
  once. The whole film takes about a minute. The package's transcript tests skip the pauses with an
  unexported option, since `testkit` has no clock. The UI tests and e2e play them on the fake and the real
  clock.
- **The scene menu.** `wopr -m` opens it: the scenes, numbered, the keys, and `SCENE:`, which takes a number, a
  slug, a title or a unique prefix. `q` (or `LOGOFF`, `EXIT`, `QUIT`) exits 0. An empty line asks again. An
  unknown name gets `NO SUCH SCENE.`, and an ambiguous one lists its scenes. Esc during playback returns to
  the menu. A scene picked there plays to the end of the list, then the menu returns. With `--only` it
  plays alone, then the menu returns, and the menu says which (`A SCENE PLAYS ALONE, THEN COMES BACK HERE.`).
- **Which scenes play.** `wopr -m <scene>` (number, slug, title or unique prefix, which `movie.Resolve`
  resolves) plays from that scene to the end of the list, then exits 0 (RF-5). An unknown scene exits 2 and
  prints the list. An ambiguous one exits 2 and names its scenes. `wopr --scenes` prints the number, slug
  and blurb of each scene and exits 0.
- **One scene** (the owner's request). `-o/--only` (`movie.Options.Only`) makes every scene the end of the
  list. `wopr -m joshua --only` plays `joshua` and exits 0. `wopr -m --only` opens the menu, where each
  scene picked plays alone and the menu returns. The end of the list means what it does without `--only`:
  the menu if the viewer came from it, otherwise exit 0. `--only` without `--movie` exits 2, like
  `--movie --play`.
- **Isolation.** No LOGON, no persona, no Brain, no network.
- **Consistency tests** (`internal/movie`). `first-contact` and `joshua` carry the `Interactive` mark. Their
  `Type` steps go through the real persona via `testkit`, and its transcript must start with the scene's
  text. archtest allows the movie package's tests, and only them, to import `wopr` (§4.1). The call-back
  and the NORAD session are not the persona's to play. But WOPR's answers there (`YOU SHOULD KNOW
  PROFESSOR. YOU PROGRAMMED ME.`, `TO WIN THE GAME.`, `WHAT'S THE DIFFERENCE?`) must equal what the
  persona's scripted brain answers to the same questions. `first-strike`, board strip included, must equal
  GTW's own transcript for the same lines. The climax's NORAD notices must equal GTW's replies to them.
  Movie mode and interactive play cannot diverge, and the scenes double as end-to-end tests.
- **Tests.** The whole film is a golden (`internal/movie/testdata/film.golden`), and two replays must be
  identical. The UI goldens `movie_menu` and `movie_screens` are at 80×24 with the front panel. They show
  the menu, a line half typed, the pause, the first strike, the climax board, tic-tac-toe playing itself
  and the game clock. The e2e cases are in §9.
- **Legal scope** (RK-1 in §16, SL-8). M6 ships the film's WOPR terminal script, much of which the
  interactive persona already contains. The only film text in the scenes is what WOPR's terminal shows on
  screen, David's typing included: no spoken-only dialogue, no audio, no stills. The rest is this project's
  own text, tagged `original`: the dial (`CONNECTING...`, `CONNECTED.`), GTW's strike exchange and board
  legend, tic-tac-toe's prompts and the game clock's seconds. Every line carries a tag. `NOTICE.md`
  excludes the quotations from the MIT grant and gives rights holders a contact. The owner accepts the
  remaining risk by requesting the feature. AGENTS.md has the takedown runbook.

---

## 8. Binary size and portability

- **Build** (GoReleaser, `.goreleaser.yaml`, the same config for snapshots and releases): `CGO_ENABLED=0`,
  `-trimpath`, `-s -w`, and `-X` for Version, Commit and Date. The config pins `mod_timestamp`,
  `builds_info` and per-file `info.mtime` to the commit time, so archives are byte-reproducible. A test with
  GoReleaser v2.18.2 touched the extra files between two runs and got identical checksums (B-12).
- **Measured** (go1.27.1, stripped, 2026-10-09): the program is 5.8 to 6.4 MB on the six targets, plus the
  Windows programs' resources (below). The standard library's HTTP client (M7) added 3.9 to 4.5 MB to a
  one-view program (2026-10-06). As much again would take the amd64 programs to about 10.5 to 11 MB: over
  the 10 MB target and under the 15 MB limit. A provider SDK would add about 4.5 MB more, so wopr uses none.
- **Size gate**: `go run ./internal/tools/sizegate -expect 6 -packages 4`, which reads `dist/artifacts.json`
  (B-7, B-9).
  - It reads GoReleaser's artifact list and gates the `Binary` entries and the Linux packages.
  - It fails on an empty list. It also fails when the number of binaries is not the expected six, or the
    number of packages is not the expected four.
  - It writes a table to the step summary, warns above 10 MB and fails above 15 MB.
  - GoReleaser's `report_sizes` also logs the sizes.
  - `sizegate -files <file>...` holds the Windows installer and the macOS disk image to the same budget, in
    the `collect` jobs (§11). The universal program in the Mac app is about 12 MB, but the image is
    compressed.
  - The README has no size table, only a development build's rough size (B-8).
- **Linux packages** (owner request 2026-10-07, built by `nfpms` in `.goreleaser.yaml`, GoReleaser's nFPM): a
  `.deb` and an `.rpm` for linux/amd64 and linux/arm64, holding the same binary as the archives.
  - **Where files go.** The `.deb` follows Debian Policy 4.7. The program is in `/usr/games` (§11.11). That
    directory is on the `PATH` of every Debian and Ubuntu login. The manual page is
    `/usr/share/man/man6/wopr.6.gz` (§12.1). In `/usr/share/doc/wopr/` are the copyright file (§12.5),
    `changelog.Debian.gz` (§12.7, which nFPM writes from the change notes), the release notes `CHANGELOG.md`
    as `NEWS.gz` (§12.7), and `README.md.gz` and `NOTICE.md.gz` (§12.3). The `.rpm` follows Fedora's
    packaging guidelines. The program is in `/usr/bin` (the Games SIG: "Binaries go in %{_bindir} and not
    /usr/games"). It has the same manual page. `LICENSE`, `NOTICE.md` and `THIRD_PARTY_NOTICES.txt` are
    `%license` in `/usr/share/licenses/wopr/` (`NOTICE.md` holds licence texts too). `README.md` and
    `CHANGELOG.md` are `%doc` in `/usr/share/doc/wopr/`. The package owns both directories. `%changelog`
    comes from the change notes.
  - **The copyright file** `packaging/debian/copyright` is machine-readable (DEP-5). `internal/tools/notices`
    generates and checks it together with `THIRD_PARTY_NOTICES.txt`. It holds:
    - wopr's licence, from `LICENSE`.
    - the film text that licence does not cover, file by file from the provenance tags (§2.1), with the
      manual page, the screenshots and the docs site's pages.
    - abs0's transcription and Matthew Thomas's map, with their terms from `NOTICE.md`.
    - every module linked into the Linux builds. The tool files each under Expat or BSD-3-clause only if
      its licence has those words, so any other licence stops the tool until someone files it.
  - **Metadata.** Section `games`, priority `optional`, no dependencies (the binary is static), homepage the
    repository. Maintainer and packager `Christopher Goes <ghostofgoes@gmail.com>`, RPM vendor
    `Christopher Goes`. `License: MIT AND BSD-2-Clause AND BSD-3-Clause AND LicenseRef-Matthew-Thomas-map`:
    wopr, abs0's transcription, the Go modules, and Matthew Thomas's map. The map is compiled into the
    program, and its terms have no SPDX identifier. So it gets a LicenseRef with the name the `.deb`'s
    copyright file gives it. The film text is under no licence. No RPM `Group` (Fedora asks for none). The
    synopsis and description are `packaging/description.txt`, which `pkgdocs`'s tests hold to both formats'
    rules.
  - **Menu entry, icon and AppStream metadata** (owner decision 2026-10-08), so that someone who never opens
    a terminal can start wopr after double-clicking the package. Both packages install, named by the app ID
    `io.github.ghostofgoes.wopr`, `/usr/share/applications/<id>.desktop`,
    `/usr/share/metainfo/<id>.metainfo.xml` and the icon in `/usr/share/icons/hicolor/`. The icon is there
    at 16, 22, 24, 32, 48, 64, 128, 256 and 512 px and as an SVG. `pkgdocs` fills the templates in
    `packaging/linux/` into `build/pkg/{deb,rpm}/` and `build/pkg/`. The systems' own triggers refresh the
    menu and the icon cache, so the packages still run no scripts.
    - The menu entry (Desktop Entry 1.5) has `Terminal=true`, so the desktop opens its terminal running
      wopr, at the terminal's own size. wopr's too-small screen covers a small one. `Exec` is the
      program's full path in each package. `StartupNotify=true` hands the terminal its startup token, so
      the busy cursor ends and, on Wayland, the terminal takes the focus. Five right-click actions start
      Chess, Checkers, Tic-Tac-Toe or Global Thermonuclear War (`wopr --play`), or Movie Mode
      (`wopr --movie`, which the owner kept, 2026-10-09). The actions name the games as `wopr --games`
      lists them (`chess`, `checkers`, `ttt`, `gtw`). The categories are Game, BoardGame, CardGame and
      StrategyGame. The keywords include WarGames, war, Joshua and the games' names, which the metadata
      lists too.
    - The metadata is a `desktop-application`, not AppStream's `console-application`, because GNOME
      Software never lists the latter (from its source, untested). Its summary and description are
      `description.txt`'s. Six screenshots come from the docs site. OARS 1.1 rates `money-gambling`
      moderate (play money in Black Jack and Poker) and `violence-fantasy` moderate (text-only war games).
      Both are judgement calls for the owner to approve. The branding colours are amber, matched to the
      icon's lamps. The releases come from `changelog.yml`: the newest three with their notes, and a
      snapshot marked `type="snapshot"`.
    - Fedora: the `.rpm` owns the hicolor directories rather than requiring `hicolor-icon-theme`, as the
      guidelines' gtk-doc example does for directories of a package not needed to run. A `Requires`
      would break the docs' offline `dnf install --disablerepo='*'` where that package is missing.
    - Checks: `pkgdocs`'s tests hold the templates to both specifications and check that their links exist
      in `site/`. `stage -archives` checks that the packages hold pkgdocs's files (by digest), that every
      `Exec` starts the package's own program, and every icon size. The smoke jobs check the installed
      `.deb`'s files with `desktop-file-validate`, `appstreamcli validate --no-net --pedantic` and
      `appstream-util validate-relax --nonet` (the one Fedora asks for). They check the `.rpm`'s files and
      the directories it owns, and check that nothing remains after removing either. `pkgdocs`'s
      `TestValidators` runs the same three validators on the files it writes (the `desktop-files` hook, §10).
    - Untested on real desktops: which terminal opens (GNOME Terminal on Ubuntu 24.04, Ptyxis on 26.04 and
      Fedora 44, Konsole on KDE, from research). Also untested: whether GNOME Software can remove a package
      installed by hand.
  - **Versions and names.** Debian revision and RPM release 1, named by each format's convention:
    `wopr_X.Y.Z-1_amd64.deb` (dpkg-name) and `wopr-X.Y.Z-1.x86_64.rpm`, `arm64` and `aarch64` likewise. A
    snapshot is `X.Y.Z~snapshot.<commit>-1`, which dpkg and rpm sort before `X.Y.Z-1`. The package
    changelogs name every version the same way (relnotes) and date it by its tagged commit. So the newest
    entry is always the latest, even for two releases on one day. Each changelog's first entry is always
    the package's own version, a snapshot's included (§11.3). The manual page names the same version:
    `pkgdocs` writes it into the page's header for every build.
  - **Reproducible.** Every time inside is the commit's (`mtime`), every mode is pinned, the RPM build host is
    the fixed `reproducible`, and the payloads are xz (deterministic). The compressed documents come from
    `internal/tools/pkgdocs`, which compresses like `gzip -9n` (no name, no time). It runs as a GoReleaser
    before hook into `build/pkg`, where it also copies the packages' `changelog.yml` from `WOPR_NOTES_DIR`
    (`nfpms.changelog` takes no template). Two snapshot builds in different clones, with different umasks
    and an icon touched between them, gave the same `checksums.txt` (2026-10-09, with the menu entry and
    icons). `repro` compares the packages with everything else.
  - **Lint** (2026-10-07). lintian 2.117 with `--pedantic --display-experimental`, Debian and Ubuntu profiles,
    finds nothing but `statically-linked-binary`, which the package overrides. lintian knows a Go program
    is static only from a Debian source package's build dependencies. rpmlint 2.8.0, Fedora 44's (with its
    configuration and the `hunspell-en-US` dictionary it depends on), reports `statically-linked-binary`
    and `position-independent-executable-suggested`. Both are by design (one static binary that runs on
    every distribution). It also reports `spelling-error` for "Falken's" and "tac" (of tic-tac-toe) in the
    description. `packaging/rpmlintrc` filters those four (`rpmlint -r packaging/rpmlintrc`), each with its
    reason, and `invalid-license` for the map's LicenseRef, which no list of licences has. Linted again on
    2026-10-09, with the map's LicenseRef, the menu entry, icons and metadata: lintian 2.117 reports only
    the override. rpmlint 2.10 with Fedora 44's configuration finds nothing new (its desktop-file and
    AppData checks ran). `appstreamcli` 1.0.2 `--pedantic` gives one info note,
    `description-first-word-not-capitalized`, because the description starts with `wopr`.
    `appstream-util validate-relax` passes.
  - **Signatures** (owner decision 2026-10-10). Releases are signed with wopr's OpenPGP key. Its public half
    and fingerprint are committed as `packaging/wopr-signing-key.asc` and
    `packaging/wopr-signing-key.fingerprint`. The secret half is the `release` environment's
    `WOPR_SIGNING_KEY` secret (AGENTS.md, "Repository settings"). `release.yml`'s `sign` job, the only job
    that sees it, runs `packaging/sign-packages.sh` on the checked release files. It runs after the
    reproducibility check and `collect`, and before the attestation. So the attestation covers the signed
    files, and `checksums.txt.asc` covers the installers' checksums too:
    - Each `.rpm` gets a signature header (`rpmsign`), which dnf (`localpkg_gpgcheck=1`), zypper and
      `rpm -K` check against an imported key. The script proves that signing changed nothing else: the
      header's and payload's SHA-256 digests are the same before and after. It also checks that `rpmkeys`
      accepts the signature with only the public key. A signed `.rpm` is no longer byte-for-byte what
      `repro` rebuilt, so the job checks the unsigned files against the rebuild before it signs. It checks
      the installers, which the rebuild does not make, against `checksums.txt`.
    - Each `.deb` gets a detached, armored `.deb.asc`. apt does not check a signature on a `.deb` installed
      from a file (only on a repository's metadata). So this is for `gpg --verify`, as is
      `checksums.txt.asc`, the signature of the rewritten `checksums.txt`.

    The secret key must be the committed key's (the script checks the fingerprint), so a replaced secret
    cannot sign. CI signs every build with a throwaway key, and `smoke` installs the packages with
    signature checks on (dnf with `localpkg_gpgcheck=1`, and `gpgv` for the `.deb.asc`). CI's `collect`
    signs again after adding the installers, as a release does. `docs.yml` looks for `checksums.txt.asc` to
    tell whether the latest release is signed (the `has-signatures` partial). Once it is, the docs site's
    Linux (RPM) command line imports the key (served at `/wopr-signing-key.asc`) and installs with the
    check on, and its Linux download tab says how each package is signed. Before that, zypper needs
    `--allow-unsigned-rpm`. Like every release file, the packages are also attested, and
    `gh attestation verify` can check them (§12).
  - **Looking inside an `.rpm`.** nFPM's RPM writer (google/rpmpack) stores the payload's paths as absolute
    (`/usr/bin/wopr`, where rpmbuild writes `./usr/bin/wopr`) and with no times. `rpm` and `dnf` install
    and check the packages correctly. But `rpm2cpio X.rpm | cpio -idm` writes into the live `/` from any
    directory. AGENTS.md's Commands table gives safe ways to list or unpack one.
- **The installers** (owner decisions of 2026-10-08: R14, decision 27). Each is built on its own runner from
  the release files that `checksums.txt` lists, checked against it first, so none compiles wopr again
  (§11). The app ID `io.github.ghostofgoes.wopr` names the Linux menu entry, icon and metadata, the snap's
  `common-id` and the Mac app's bundle identifier. The Microsoft Store assigns the MSIX identity itself.
  The main launchers are the Start menu entries of the installer and the MSIX, the Mac app, and the Linux
  and snap menu entries. They open a terminal running `wopr` with no arguments, since there is no start
  screen yet (an issue tracks `-w/--welcome`). The Linux and snap menu entries add right-click actions for
  four of the film's games and Movie Mode (above). The installers carry no code signature yet. Only the
  Linux packages are signed, with wopr's OpenPGP key (above), and `checksums.txt.asc` lists the installers'
  checksums. Issues track Windows signing and Apple's signing and notarization.
- **Icon** (owner decisions). On 2026-10-08: regenerated from its sources with one command. On 2026-10-09:
  front-panel lamps, a gunmetal frame around a panel of indicator lamps, lit amber along one diagonal, with
  one red lamp.
  - The sources are in `packaging/icons/src/`. The master is `wopr.svg` (1024×1024, with a transparent
    margin, for Windows, Linux and the MSIX, four by four lamps). `wopr-small.svg` is optional simpler art
    for 32 px and below (three by three lamps on a 64-unit grid, whole pixels at 16 and 32 px).
    `wopr-<N>.svg` is optional art for exactly N px, which wins at that size. `wopr-full.svg` is an opaque
    full-bleed square for macOS, which rounds icons itself, and for the site's `apple-touch-icon.png`, at
    every size. `wopr-24.svg` redraws the small art on a 24-unit grid for the Windows taskbar at 100% scale,
    where the 64-unit grid falls on half pixels. 20 and 30 px (125% scale) stay soft. A redrawn 20 px needs
    2 px lamps and reads no better, and the soft 30 px is close to whole pixels. The tool refuses a
    `wopr-<N>.svg` for a size it never draws, and any other SVG file there.
  - `internal/tools/icons` draws every file from them with its own renderer, which matches Chromium to
    within 1/255 on average at 256 px and up. The renderer uses the standard library only: oksvg and
    rasterx cannot draw `scale(s)`, even-odd fills or transformed gradients correctly. The tool writes
    `wopr.ico`, `wopr.icns`, the hicolor PNGs and SVG, the MSIX assets, and the docs site's favicons, also
    its navbar logo. `wopr.ico` has the sizes 16 to 256 px, PNG at 256. The MSIX assets are scale and
    target-size variants, with `_altform-unplated` and `_altform-lightunplated` twins, which Microsoft
    lists to avoid a backplate.
  - In `wopr.icns` the 16 and 32 px pictures are `is32`/`il32` with masks, and the rest are PNG. PNG in
    `icp4`/`icp5` does not show as an app's icon, and no form of `icp6` shows everywhere
    (relikd/icns-archive's tests on macOS 10.0 to 26). Apple documents none of this.
  - The sources are a strict SVG subset. Otherwise the tool refuses the source, with the file and line. No
    output may pass 512 KB. `-check` and the tests compare pixels within 2/255, because fused
    multiply-adds on arm64 change a few bytes. The `.ico` and `.icns` around the pictures must be byte for
    byte what the tool writes from those pixels, with their PNG data kept as it is. The icon is original
    work, so NOTICE.md needs no entry.
- **Windows installer**, `wopr_X.Y.Z_windows_setup.exe`, a release file. Inno Setup 7.1.0 packs both Windows
  programs and the notices (`packaging/windows/wopr.iss`, `windows.yml`). It is pinned by URL, SHA-256 and
  its publisher's Authenticode signer, and installed in portable mode into a new folder on the runner.
  Each of its programs and DLLs must then carry the same signer's valid signature.
  - Per user, with no administrator prompt (`PrivilegesRequired=lowest`, and an administrator may pass
    `/ALLUSERS`). It installs into `%LOCALAPPDATA%\Programs\WOPR`, the folder the PowerShell line uses, so
    installing over that copy replaces it. One file for every PC: Setup installs the arm64 program on
    Arm64 and the amd64 one elsewhere, as `wopr.exe`, on 64-bit Windows 10 and later.
  - A short wizard that follows Windows' light or dark mode. It shows the tasks (a desktop shortcut, off,
    and add WOPR to the user's `PATH`, on), a summary, the progress bar, and Finish with "Launch WOPR". One
    top-level Start menu shortcut runs `wopr.exe`, and Windows opens its default terminal: Windows Terminal
    on Windows 11 22H2 and later, the console host on 10. Settings → Apps lists and uninstalls it.
    Uninstalling also removes the debug log (`%LOCALAPPDATA%\wopr\debug.log`, then its folder if empty).
    It removes a `PATH` entry only if Setup added it, or if it names the default folder, which is WOPR's
    alone (the PowerShell line's entry included). A folder that was on `PATH` before, such as one that
    `/DIR=` named and other programs share, stays on it. Setup records the folder it added in the
    uninstall entry, and an upgrade keeps the record. Setup never writes a `Path` that is not text, and
    never adds a folder whose name has a semicolon or a percent sign. A silent Setup logs why, and asks
    nothing. An upgrade is the newer installer run over it. Setup closes a running wopr first. Uninstall
    cannot (Inno Setup's uninstaller has no Restart Manager), so it asks the user to close WOPR, with
    Inno Setup's own message. A silent Uninstall logs it and goes on, and leaves `wopr.exe`. One Setup runs at a time in each Windows session. The mutex that
    enforces this has no `Global\` name, so another account on the PC cannot hold it to stop Setup.
  - `AppId` `{A7D86110-58E9-404E-9164-FBDDBDC6AC22}` never changes: upgrades and the uninstall entry
    depend on it.
  - Unsigned: SmartScreen asks (More info, then Run anyway). Smart App Control, when it is on, blocks the
    installer with no way to allow one app. The install tab and the troubleshooting page say both.
  - Attested and reproducible. Inno Setup writes no time or path into it (every file `notimestamp`), so
    the same release files, `wopr.iss` and Inno Setup give the same bytes. `repro` covers those release
    files. Each smoke job builds the installer again and fails unless the bytes match the build job's.
    That catches a change made by the build job's runner, not one in the commit itself. The MSIX
    packages are built once and not compared; they are never release files.
  - The smoke jobs on `windows-2025` and `windows-11-arm`:
    1. check the release files against `checksums.txt`, and build the installer again (above).
    2. install it silently.
    3. check the files, the shortcut, the uninstall entry, `PATH` and the program's CPU type. Each
       installed file must be its release file, by SHA-256, and the folder holds nothing else.
    4. run `wopr --version`.
    5. uninstall it and check that nothing is left, a debug log included.
    6. repeat the steps over a copy from the PowerShell line.
    7. check whose `PATH` entry Uninstall removes: a folder that `/DIR=` named and `PATH` already had
       stays, and one that Setup added goes, also after an upgrade with the task off. Setup leaves a
       `REG_MULTI_SZ` `Path` alone, and adds no folder whose name has a semicolon or a percent sign.
    8. check that a silent Uninstall notices a `wopr.exe` in use.
    9. install for all users (`/ALLUSERS`): Program Files, the system `PATH`, then uninstall.
  - Inno Setup's licence lets anyone use it (jrsoftware asks only commercial users to buy one). The docs
    site's credits page names it.
- **`wopr.exe`'s resources.** go-winres v0.3.3 (0BSD, in `tools/release/go.mod`) writes an icon, a manifest
  (`asInvoker`, Windows 10 and 11) and version details into `cmd/wopr/rsrc_windows_{amd64,arm64}.syso`. It
  reads them from `packaging/windows/winres.json` and runs in a GoReleaser before hook. The version details
  are the file version X.Y.Z.0, and the product version with any snapshot suffix. `--arch amd64,arm64`
  names both targets: the default, amd64 and 386, would leave the arm64 program bare. The files are
  byte-identical on every run, so the programs stay reproducible. The icon adds about 185 KB to each.
- **MSIX** (owner decision 2026-10-08: built for a later Microsoft Store submission, never a release file).
  `windows.yml` builds `wopr_X.Y.Z_windows_{amd64,arm64}.msix` and `wopr_X.Y.Z_windows.msixbundle` with the
  Windows SDK's `makeappx` and `makepri`. They hold a full-trust console app with a `wopr.exe` execution
  alias, as Python's Store package has, for Windows 10 2004 (10.0.19041) and later, versioned X.Y.Z.0. They
  stay unsigned, since the Store signs what it publishes and nobody can install an unsigned package. So
  they are the workflow artifact `msix` (kept 90 days for a tag). The identity is a placeholder until the
  owner reserves the name and sets the `MSIX_*` repository variables (AGENTS.md). The Store refuses a
  version whose first number is 0, so the first submission waits for v1.0.0. The smoke jobs sign a copy
  with a throwaway certificate to install it, run `wopr --version` through the alias, and remove it.
- **macOS app**, `wopr_X.Y.Z_macos.dmg`, a release file: one universal `WOPR.app` beside a link to
  `/Applications`, in an HFS+ UDZO image (`packaging/macos/build-dmg.sh`, `macos.yml`). `lipo` on a
  `macos-26` runner joins the two darwin programs. GoReleaser's universal binaries would add a seventh
  binary to the size gate.
  - `internal/tools/macapp` writes the bundle: `Info.plist`, `PkgInfo`, the program, `WOPR.icns` and the
    notices. `Info.plist` has the bundle identifier (the app ID), executable `wopr`, versions X.Y.Z, and
    category `public.app-category.strategy-games`. It has `LSUIElement`, so that the short first run puts
    no icon in the Dock, and `LSMinimumSystemVersion`, read from the programs' `LC_BUILD_VERSION` (13.0
    with Go 1.27). The support statement stays macOS 26.
  - Ad-hoc signed (`codesign --sign -`, without which Apple silicon will not run it), with the hardened
    runtime that notarization will need. With no Developer ID and no notarization, Gatekeeper blocks the
    app the first time it opens. Since macOS 15, Control-click no longer bypasses it. The user clicks Done,
    then Open Anyway in System Settings → Privacy & Security, and approves with a password. The install
    tab gives each step, and says to drag WOPR to Applications first. An app opened from Downloads or the
    image runs from a randomised read-only copy (App Translocation), which may vanish while wopr hands
    itself to Terminal.
  - **The relaunch** (`cmd/wopr/launch*.go`). The bundle's executable is `wopr` itself. It reopens itself in
    Terminal when all of these hold:
    - it runs from `<name>.app/Contents/MacOS/` with launchd as its parent.
    - neither stdin nor stdout is a terminal (a character device other than `/dev/null`).
    - it has no argument but an old `-psn_` one.

    Then it runs `/usr/bin/open -b com.apple.Terminal <its own path>` and exits, and Terminal runs it again
    in a new window. Any other start is unchanged, so a pipeline or the e2e tests still get "not a
    terminal". It sends Terminal no Apple events, which would need the user's permission. It always uses
    Terminal. An iTerm2 user runs the program inside the app directly. `os/exec` is allowed in `cmd/wopr`
    for it (§4.1).
  - Attested, not reproducible: the image's metadata and the signature differ from build to build. It
    holds only files that `checksums.txt` lists, which `build-dmg.sh` checks.
  - The smoke jobs on `macos-26` and `macos-26-intel` mount the image and check its layout, `Info.plist`,
    the signature and both architectures. Then they run the e2e tests against the app's program. On
    `macos-26` the job also opens the app as Finder does, to check the hand-off to Terminal. Not on
    `macos-26-intel`: its Terminal runs nothing it is given, not even a plain `.command` script (checked
    2026-10-09). Artifacts are not quarantined, so Gatekeeper's dialogs need a person on a Mac (§14).
- **Snap** (owner decision 2026-10-08: built and tested, publishing off). `wopr_X.Y.Z_{amd64,arm64}.snap` has
  base `core24`, strict confinement with no plugs, and the app ID as `common-id`. `core24` is the base of
  Ubuntu's preinstalled snaps, so an Ubuntu desktop downloads nothing more. `bare` would also run the
  static program.
  - There is no `snapcraft.yaml`. `internal/tools/snapdir` writes the snap's directory from the release's
    program and notices, `packaging/snap/snap.yaml.in`, the 256 px icon and the Linux menu entry, rewritten
    as snapd wants it. Then `snap pack` packs it. Its licence is the `.rpm`'s without the map's LicenseRef,
    which snapd refuses. The map's terms are in NOTICE.md inside the snap.
  - The Store's `review-tools` must pass each snap. `packaging/snap/test-snap.sh` installs it, runs it in a
    pty under its confinement with the e2e tests, and removes it, on `ubuntu-24.04` and `ubuntu-24.04-arm`.
    The snaps are the workflow artifact `snaps` (kept 90 days for a tag), never release files, and not
    checked for reproducibility.
  - `snap.yml`'s publish job uploads them to the stable channel, in the `snap-store` environment that holds
    `SNAPCRAFT_STORE_CREDENTIALS` (AGENTS.md). It runs only for a `v*` tag, after the GitHub release is
    published, and when the repository variable `SNAP_PUBLISH` is `true`. It stays off until the owner
    registers the name `wopr` and configures these.
  - App Center's own Open button starts a program without a terminal (from snapd's and App Center's source,
    untested). So the docs will send users to the app grid instead. With `WOPR_DEBUG`, the snap's log is
    `~/snap/wopr/current/.cache/wopr/debug.log`.
- **Never UPX** (Windows AV false positives).
- **Windows.**
  - Ctrl+C arrives as a key in raw mode on every OS, so `Update` maps it to `tea.Interrupt`. A blocked
    `Update` would delay it, which is why all slow work is `Think`.
  - Keyboard enhancements stay off.
  - conhost resize events may report the buffer height (9001). Clamp to the window rect and check in the M2 QA
    pass.
  - Ship unsigned (an issue tracks signing). The one-line install downloads with `curl.exe`, which sets no
    Mark of the Web, so SmartScreen does not ask. It does ask about the installer, which people download
    with a browser, and the install tab says what to click. The Installation page's "Verifying binaries
    (attestation)" describes the check as optional (§12, §13).
- **Panics**: no custom `recover` around `p.Run()`. Bubble Tea restores the terminal, recovers `Cmd` goroutine
  panics too, and prints the stack to stderr. The model returned on panic is nil, and wopr does not use it.
- **Hangs**: `ui.Run` runs the model under a watchdog (`internal/ui/watchdog.go`, §4.4). An Update or View
  that runs 10 s, or is still running a second after SIGINT or SIGTERM, is taken for a hang. wopr then
  kills Bubble Tea's program (undoing raw mode and the screen modes itself if that blocks), writes the
  reason to stderr and the debug log, and exits 1.
- **Debug log** (S-5, `internal/debuglog`): `WOPR_DEBUG=1` writes `os.UserCacheDir()/wopr/debug.log`. It
  records the version and session seed, and the terminal size and colour profile. It records each
  `Think`'s duration and whether it hit its deadline, and how the session ended. An e2e test checks that
  typed text never appears.
  - Directory mode 0700, file mode 0600, tightened on every open (a log left by an older build, or made by
    hand). It never follows a symbolic link at the log's directory or file (`O_NOFOLLOW` on Unix). On Unix
    it also refuses a directory another user owns. So a cache directory in a shared place
    (`XDG_CACHE_HOME=/tmp`, say) cannot make wopr append to some other file.
  - If there is no cache dir, wopr disables logging and writes a note to stderr.
  - wopr prints the path on exit.
  - wopr's debug log never holds typed input.
  - Bubble Tea's own `TEA_TRACE` and `TEA_DEBUG`, if the user sets them, record raw input and output or write a
    panic log into the CWD. wopr never sets them. They are the user's own developer switches (SL-7).

---

## 9. Testing

- **Unit tests** cover:
  - game rules (via `testkit`).
  - `Resolve`, `cli.Parse`, intent and offers (the film lines, the review's cases and the false-positive
    table).
  - the LOGON table.
  - the host runner: routing, launch seeds, hand-off, the Esc machine, Think deadlines, Animate.
  - the scripted brain, typewriter dt math and pauses, and the line editor.
  - `Sanitize*`, the canvas, and the themes: every Style defined, distinct pairs in 16 colours and without
    colour, contrast, and each profile rendering only its own colours.
  - accessibility (M5): the sweeps of every game in `internal/ui/access_test.go` (§4.5).
  - packaging (§8): the icon renderer against exact areas and gradients, and the committed icons against their
    sources. Also the app bundle, the snap's directory, the menu entry and AppStream metadata, and the macOS
    relaunch decision, as a table.
- **Architecture**: `archtest` covers the DAG (test imports and e2e files included), toolchain equality in CI,
  and the tool modules' `go` lines. It also covers the pins shared between `prek.toml` and the tool modules,
  and the lint self-test. Each data package's tests check its provenance tags.
- **Golden** (Q-3, A-14).
  - `internal/golden.Assert(t, name, got)` stores files under each package's `testdata/`. They are byte-exact
    apart from CRLF, which is normalised. Trailing padding in a canvas golden is intentional, and the
    whitespace hooks skip `.golden` files (TO-2).
  - Regenerate every golden with `WOPR_UPDATE_GOLDEN=1 go test ./...`, which works on every OS and package.
  - **Persona and game transcripts** come from `testkit`: the real runner, `Instant` and `Deterministic`, with
    `[CLEAR]` and `[NOTICE]` markers.
  - **UI screens** come from `internal/ui/drive_test.go`, a synchronous driver in package `ui`, since the model
    is unexported. A fake `Scheduler` records armed ticks, and a fake clock delivers them. Commands run
    inline, and the driver unwraps `tea.BatchMsg` (`tea.Sequence` is banned, §10). `Think` results can be
    held to simulate a slow search. The driver snapshots `View().Content` after `ansi.Strip`, with trailing
    spaces trimmed, plus the cursor position.
  - **Flows** (M1): persona `film_greeting`, `logon_fail_x3_hint` (a game name at LOGON is not a login),
    `logon_list_then_number`, `help_list_games`, `intent_false_positives`, `offer_accept`, `stub_game`. UI:
    `logon_joshua`, `logoff_exit0`, `too_small_pause_resume`, `esc_confirm`, `typeahead_thinking`,
    `front_panel`. The film path has its own flow, `film_path` (M2), and movie mode has one for the whole
    film, `film` (M6, §7).
- **Fuzz** (Q-2): `FuzzClauses`, `FuzzSanitizeInput`, `FuzzEditor`, `FuzzResolve`, `FuzzGameIntent`, the games'
  move and order parsers, `FuzzParseSVG` (the icon tool's SVG subset), and the M7 reply parser.
  - Seed corpora run in plain `go test`.
  - CI finds every target with `go test -list '^Fuzz'`, runs each for 10 s on Linux, and uploads any
    crashing input as an artifact.
- **End-to-end** (Q-1, U-8).
  - `internal/e2e` (build tag `e2e`) is a Go test that drives the **real binary** in a pseudo-terminal. It uses
    `charmbracelet/x/xpty` (a Unix pty, or ConPTY on Windows) and feeds the output to a
    `charmbracelet/x/vt` emulator, so tests wait for text on the **screen**. The typewriter and Bubble Tea's
    diffing renderer split text across writes, so the byte stream is unreliable. The emulator's replies to
    terminal queries go back to wopr, as a real terminal's would.
  - On Unix the child gets the pty as its controlling terminal (`Setsid`, `Setctty`). xpty leaves that to the
    caller, and without it a resize never arrives as SIGWINCH.
  - **Cases**, by milestone (IM-3):
    - M0: `--version`, `--games`, `--licenses`, a closed pipe exits 0, refusal without a terminal (`--movie`
      too).
    - M1:
      - start → `LOGON:` → Ctrl+C → exit 130 with the alternate screen left.
      - `Joshua` → `GREETINGS PROFESSOR FALKEN.` → `Hello.` → `HOW ARE YOU FEELING TODAY?` →
        `LOGOFF` → exit 0.
      - start at 60×20 → `TERMINAL TOO SMALL` → resize to 80×24 → `LOGON:`.
    - M6:
      - `--movie climax --instant` shows the board, `** ACCESS DENIED **` and the film's last words, each
        held by the scene's pauses. It exits 0 at the end of the list without a key.
      - paced, `-m 2` → `LOGON:` → Space → `** PAUSED **` and a still screen → Right → the next
        scene → Esc → `SCENE:` → Ctrl+C → exit 130.
      - `-m` → the scene menu → `q` → exit 0.
      - `--scenes` lists the scenes, and `-m nowhere` exits 2 listing them.
      - `-m joshua --only --instant` plays that scene and exits 0 at its end without a key, never showing
        the next scene's `WHICH SIDE DO YOU WANT?`.
    - after v0.2.0: `--only` without `--movie`, `WOPR_SEED=x` and `-s x` exit 2 naming their source. With
      `WOPR_DEBUG=1 WOPR_SEED=1983` the debug log records `seed 1983 (pinned: true)` and nothing typed.
  - On Unix every TUI case also asserts that wopr restored the terminal modes (termios). Under ConPTY the
    assertion is weaker (exit code and final screen), because conhost owns the console modes.
  - The `build` job cross-compiles the test per target (`internal/tools/stage`) and ships it next to each
    binary, so smoke runners need no Go toolchain. The same test also runs against the program inside the
    Mac app (`macos.yml`) and against the confined snap (`snap.yml`, all but `TestDebugLog`). A snap's
    `/tmp` is its own, so `test-snap.sh` checks the log where the snap writes it.
- **Race.**
  - CI runs `go test -race ./...` on `ubuntu-24.04` and `macos-26`, and plain `go test ./...` on
    `windows-2025`. The race detector there would need a C toolchain on the runner.
  - `testkit` exercises `View` concurrently with `Think` so the detector has something to see.
  - Contributors: AGENTS.md's Commands table says where `-race` runs. windows/arm64 has no race detector (B-14).

---

## 10. Tooling: prek and static analysis

- **One hook set in `prek.toml`** runs identically locally and in CI. `prek install` installs both the
  pre-commit and pre-push shims.
  - **The gate is `prek run --all-files`.** `prek validate-config` checks syntax only. It resolves neither revs
    nor hook ids (T-1, T-9).
  - **Ordering**: each file-mutating fixer gets its own priority, because equal priorities run concurrently and
    lose updates (T-8). Formatters come next, then read-only linters.
  - `prek.toml` overrides golangci-lint's `golangci-lint-full` entry to drop `--fix`. zizmor runs without
    `--fix` (T-5).
  - actionlint checks workflow syntax. Same-repository calls use GitHub's `$/` syntax, which zizmor's
    `self-repository` audit asks for. actionlint 1.7.12 predates that syntax, so `.github/actionlint.yaml`
    ignores that one message for `$/` calls only.
  - The whitespace and end-of-file fixers skip `.golden` files, whose padding is intentional (TO-2).
- **Pinning** (T-3, T-10).
  - Every hook rev is a full commit SHA with a `# frozen: vX.Y.Z` comment. A note about a rev goes on its own
    line above it, because `prek update` rewrites the comment after `rev` (TO-9).
  - `[update] freeze = true, cooldown_days = 14` makes a plain `prek update` (formerly `auto-update`) keep that
    form.
- **Dependency cooldown.** Go modules wait 14 days, as hooks do. `internal/tools/cooldown` fails when any
  module in wopr's, the tool modules' or the site's build list is younger than that at HEAD's committer
  date. It measures a module's age by the version time the module proxy reports. It runs in CI's `lint` job
  and as a prek hook on `go.mod`/`go.sum`. As a hook, it measures at the current time. A compromised
  upstream commit then has two weeks to be noticed before it can reach a release.
  The cooldown applies to regular updates only. A security fix (an update that fixes a published `GO-`,
  `GHSA-` or `CVE-` advisory) may skip it, at the oldest version that has the fix. The same applies to a Go
  toolchain release. `tools/cooldown-exceptions.txt` lists those `module@version` pairs, each with its
  advisory IDs as the reason. Its line is removed once the version is 14 days old.
- **Go tools** (T-4, T-11, decision 20). Four tool modules, pinned and checksummed by their `go.sum`, run as
  `go tool -modfile=<module> <tool>`:
  - `tools/go.mod`: gitleaks and govulncheck.
  - `tools/lint/go.mod`: golangci-lint (for the lint self-test and the Commands table).
  - `tools/release/go.mod`: GoReleaser, changie and go-winres.
  - `tools/docs/go.mod`: Hugo, for the docs site (§13). It has its own module because its graph is large.

  golangci-lint is separate because sharing a module with gitleaks pulls `x/ansi` to a version that breaks
  gitleaks's build (SL-2). **No tool module's `go` line may be newer than the root `toolchain` line**, because
  CI runs every tool with `GOTOOLCHAIN=local`. GoReleaser's `go` line tracks the newest Go patch, so a
  GoReleaser bump usually needs a toolchain bump first, in its own PR (CR-4, TO-7).
  `TestToolModulesFitTheToolchain` enforces it. Tools pinned twice (golangci-lint and gitleaks, in `prek.toml`
  and in a tool module) must agree. `TestToolPinsAgree` checks that (TO-8).
- **golangci-lint** (`.golangci.yml`, v2.14.0):
  - `default: standard` plus revive, gocritic, misspell, forbidigo, depguard and gosec. The formatters are
    gofmt, goimports and gofumpt. `run.build-tags: [e2e]` brings the e2e tests under lint (TO-6).
  - **gosec** looks for common security weaknesses in the code that ships. It skips tests and the test-only
    and build-time packages (`internal/tools`, `archtest`, `e2e`, `golden`, `testkit`, `gamestest`). A false
    positive gets `//nolint:gosec` with the reason on the same line.
  - **forbidigo** (T-2, A-11), with `analyze-types: true`:
    - `^tea\.(Tick|Every|Sequence)$` with `pkg: ^charm\.land/bubbletea/v2$`, exempt in `internal/ui/clock.go`.
    - the top-level `math/rand/v2` functions, and every `math/rand` (v1) function (TO-5).
    - `time.Sleep`, `After`, `AfterFunc`, `Tick`, `NewTicker` and `NewTimer`, exempt in tests.
  - `warn-unused` is on, so golangci-lint reports a stale exclusion. Film text needs no misspell exclusion:
    misspell's default English accepts `RECOGNISED` (TO-4).
  - **Self-test** (TO-1, AR-3, IM-1). `config verify` accepts rules that never fire, so
    `internal/archtest/testdata/lintfixture` holds a deliberate violation of each rule, one per line. Under
    `testdata/` it is outside `./...`, so neither the lint gate nor archtest sees it.
    `WOPR_LINT_SELFTEST=1 go test -run TestLintRulesFire ./internal/archtest` lints that path with the
    `tools/lint` golangci-lint and requires each rule's report. CI's lint job runs it.
- **codespell** skips script and art data with a prek `exclude` regex. The regex also matches `(^|/)testdata/`
  and every `go.mod` and `go.sum` (the tool modules' hashes produced false positives, TO-3). `crasher` is on
  its ignore list, and so is `fpr` (gpg's fingerprint record, in the signing scripts).
- **ShellCheck** checks the scripts that run on one system only: the macOS image's, the snap's and the
  release's signing script (`packaging/macos/*.sh`, `packaging/snap/test-snap.sh`,
  `packaging/sign-packages.sh`). The Windows installer's scripts are PowerShell
  (`packaging/windows/*.ps1`), which no hook lints, so their CI runs are their check. They stay
  non-executable for the shebang hooks. Logic that any system runs is Go (`internal/tools/*`), so it runs on
  Windows (B-14).
- **Change notes** (§13): a local `change-notes` hook runs `go run ./internal/tools/relnotes -check` when
  `.changes/`, `.changie.yaml`, `CHANGELOG.md` or relnotes itself changes, and on every `--all-files` run.
  It builds the pinned changie, so the generated `CHANGELOG.md` and version files are also kept to what
  rumdl and the whitespace fixers accept.
- **Menu entries and AppStream metadata** (§8): a local `desktop-files` hook runs `pkgdocs`'s and `snapdir`'s
  tests when `packaging/linux/`, `description.txt` or either tool changes. It runs them with `-count=1`,
  since Go's test cache cannot see programs outside the module. `TestValidators` runs
  `desktop-file-validate`, `appstreamcli` and `appstream-util` where they are installed (Linux tools, not
  pinned). Elsewhere it skips them, unless `WOPR_DESKTOP_VALIDATORS=1`, which CI's lint job sets after
  installing them.
- **Commands** have one home: the Commands table in `AGENTS.md`. There is no Makefile, because Git for Windows
  ships no `make`. The README links to that table.

---

## 11. CI/CD (GitHub Actions)

### 11.1 Common rules

- Every workflow pins each action to a full commit SHA. It pins the tools inside actions too:
  `prek-version: 0.5.5` (T-11), and GoReleaser and gitleaks via `go tool`.
- `permissions: {}` at the top of each workflow. Each job grants only what it needs.
- `persist-credentials: false` on every checkout. The workflows use only `GITHUB_TOKEN`, except
  `SNAPCRAFT_STORE_CREDENTIALS`, a secret of the `snap-store` environment that only `snap.yml`'s publish job
  reads (§12). No `pull_request_target`.
- **`actions/setup-go`** uses `go-version-file: go.mod`, and **nothing sets `GOTOOLCHAIN` before it** (D-6).
  setup-go exports `GOTOOLCHAIN=local` itself after installing 1.27.2. `archtest` asserts the exact toolchain in
  CI.
- **Caching.** `cache: false` in `release.yml` and in any job that runs GoReleaser (B-4, B-9). Compiling
  GoReleaser from a cold cache costs about 1.5–2 minutes per such job. That cost is accepted (CR-4). Every
  other `ci.yml` job that builds modules keys its cache on their `go.sum` files (`cache-dependency-path`).
  With one shared key, the first job to finish saves the cache for all of them. Jobs that build only
  standard-library tools (`collect`, and the Mac app's and the snaps' builds), `docs.yml` (§13) and
  `scheduled.yml` use none.
- **Timeouts.** Every job sets `timeout-minutes`: 5 to 20 for most, and 30 for `test` and the two release
  builds. `release.yml`'s `verify`, which may wait 45 minutes for `ci-ok`, gets 50. The smoke step runs the
  e2e binary with `-test.timeout=5m`, so a hang fails with every goroutine's stack instead of holding a
  runner for six hours.
- **`fetch-depth: 0`** in every job that runs GoReleaser. That way, snapshot versions reflect the latest tag,
  and the release `build` and `rebuild` jobs produce the same version (CR-7).
- **`.gitattributes`**: `* text=auto eol=lf`, plus `*.png`, `*.gif`, `*.jpg`, `*.ico` and `*.icns` binary, and
  `site/assets/vendor/**` `-text`, kept byte for byte (§13). Windows runners check out with
  `core.autocrlf=true`, which would otherwise break goldens (B-3).
- **Concurrency** (CR-2) is set in `ci.yml`, `release.yml` and `docs.yml`, never in the reusable workflows
  (`smoke.yml`, `windows.yml`, `macos.yml`, `snap.yml`). A called workflow sees its caller's `github.ref`, and
  the same group in both deadlocks. Pull requests share a group per PR, and pushes to other branches a group
  per branch. Both cancel superseded runs. Pushes to `main` get a group **per commit**, because with a shared
  group a newer pending run replaces an older pending one. That would leave a merged commit without `ci-ok`:

  ```yaml
  concurrency:
    group: >-
      ${{ github.event_name == 'pull_request' && format('ci-pr-{0}', github.ref)
      || github.ref == 'refs/heads/main' && format('ci-{0}', github.sha)
      || format('ci-branch-{0}', github.ref) }}
    cancel-in-progress: ${{ github.event_name == 'pull_request' || github.ref != 'refs/heads/main' }}
  ```

  `release.yml` uses `release-${{ github.ref }}`, `docs.yml` uses `docs-${{ github.ref }}` (and `pages` for its
  deploy job, §13), and `scheduled.yml` none.

### 11.2 `ci.yml` (on `pull_request` and `push` to any branch)

CI runs on every branch push (owner decision 2026-10-06), so CI checks a branch before anyone opens a pull
request. For a pull request from a branch of this repository, the branch's push run checks it and reports
`ci-ok` on the same head commit. Its `pull_request` run skips every job, so the same work does not
run twice (owner request, same day). A skipped job counts as passed, so the skipped aggregator must not be
called `ci-ok`. Its name is an expression, which GitHub does not evaluate for a skipped job, so GitHub lists
its check under the expression's text. The name must never become a plain `ci-ok`. A fork's pushes never
reach this repository, so pull requests from forks run in full. Tags go to `release.yml` only.

The push run tests the branch as it is, not merged with `main` as a `pull_request` run would. The `main`
ruleset therefore requires branches to be up to date before merging (§12). Updating a branch is a push, so
its run tests exactly the tree the squash merge produces.

| Job | Runner(s) | Does |
|---|---|---|
| `lint` | `ubuntu-24.04` | `fetch-depth: 0`. prek via `j178/prek-action` with `prek-version: 0.5.5`, including the `change-notes` hook and the `desktop-files` hook (§10). The `change-notes` hook (`relnotes -check`) checks that the notes parse and fit one Debian changelog line. It also checks that `CHANGELOG.md` is `changie merge`'s output and that every release tag has its `.changes/vX.Y.Z.md`. The `desktop-files` hook runs with Ubuntu 24.04's validators installed first and `WOPR_DESKTOP_VALIDATORS=1`. The lint-fixture self-test (`WOPR_LINT_SELFTEST=1`). The dependency cooldown, `internal/tools/cooldown`: no module in any build list is younger than 14 days at HEAD's committer date (§10). `relnotes -since origin/main` checks the following rule (AGENTS.md). A branch that changes a non-test file under `cmd/` or `internal/` (test-only packages aside) adds a change note or carries a `Changelog: none` trailer. So does a branch that changes `.goreleaser.yaml`, `packaging/` or `docs/man/`. Separately, a zizmor online-audits step with `GH_TOKEN` scoped to that step only (T-5). |
| `secrets` | `ubuntu-24.04` | `fetch-depth: 0`. gitleaks over the checked-out history (`--log-opts="--full-history HEAD"`): the pushed branch, or a fork's PR merged with `main`. It does not scan every branch, so a stale branch cannot block every PR. `--no-color`, because gitleaks colours its log even into a pipe, so the step fails on any `ERR` line. The step also fails on `0 commits scanned` and on a finding (S-1, SL-2). |
| `test` | `ubuntu-24.04`, `macos-26`, `windows-2025` | `go test -race ./...` (plain on Windows). On Linux: every fuzz target for 10 s, with crashing inputs uploaded on failure. Also on Linux: `govulncheck`, the third-party-notices check (§12), the manual-page check (§13), the icons check (`icons -check`, §8) and branch coverage (`branchcov -min 80`: gobco on each package that ships, with the table in the job summary). |
| `build` | `ubuntu-24.04` | `relnotes -snapshot`: the release notes for this commit's snapshot version, as `release.yml` makes them for a tag (§11.3). The job shows them in its summary and passes them to GoReleaser in `WOPR_NOTES_DIR`. On a release pull request, whose version is batched but not yet tagged, they are that version's notes under the snapshot's version. So the packages' first changelog entry is still their own. A GoReleaser snapshot of all six targets and the four Linux packages. The size gate. `internal/tools/stage -archives -assets dist/release`, which copies each binary to `stage/<os>_<arch>/` and cross-compiles the e2e test next to it. It puts the `.deb` and `.rpm` beside the Linux ones, checks every archive's and package's contents, and collects the release files (below). The job uploads one download per platform: the bare binary as-is, named after its file, such as `wopr_<version>_linux_amd64`. `archive: false`, so it is not zipped. It loses its executable bit, and `--licenses` prints its notices. These downloads are kept 30 days on `main`, 7 on other branches and 3 on PRs (B-13). The job also uploads a `smoke-bundle` of every staged binary and e2e test (1 day). This run's smoke jobs use it, and so do the end-to-end tests of its `macos` and `snap` jobs. Its `version` output is GoReleaser's `.Version` from `dist/metadata.json`, which names the installers and snaps. For them it uploads `dist/release` as `release-dist` (1 day), unsigned, as `release.yml`'s build does. Then it signs the Linux packages with a throwaway key made for the run (`packaging/sign-packages.sh`, §8). It stages the signed `.rpm`, the `.deb.asc` and the public key beside the Linux binaries, for the smoke jobs' signature checks. |
| `smoke` | `ubuntu-24.04`, `ubuntu-24.04-arm`, `macos-26`, `macos-26-intel`, `windows-2025`, `windows-11-arm` | The reusable `smoke.yml`, with one input, `artifact` (CR-3). It downloads the staged artifact and `chmod +x`es the files (artifacts lose the executable bit). It runs the e2e test against the binary with `-test.timeout=5m`, with no Go toolchain. Every target has a native runner, so none is skipped. On the two Linux runners it also installs the `.deb` with `sudo apt-get install`. It finds `wopr` on the `PATH` that `/etc/environment` gives every login, and its manual page with `man -w`. It checks that the changelog starts with the package's version, and removes the package. It also installs the `.rpm` with `dnf` in a Fedora 44 container pinned by digest, with no network (it needs no other package). It runs `rpm -V`, checks the changelog, and removes the package. With CI's throwaway key staged, `gpgv` checks the `.deb.asc` before the install. In that case the `.rpm` must also pass `rpm -K` with the key imported, and it installs with `localpkg_gpgcheck=1`. The release's stage is unsigned, so there `rpm -K --nosignature` checks its digests. Both check the menu entry (whose `Exec` names the package's program), the icon in every size and the AppStream metadata. The Ubuntu step checks them with `desktop-file-validate`, `appstreamcli` and `appstream-util` (apt installs those first). The Fedora step checks that the package owns the icon directories. Neither removal may leave a file behind. |
| `windows` | `windows-2025`, with smoke on it and `windows-11-arm` | The reusable `windows.yml` with `release-dist` and the version. It checks the files it packs against `checksums.txt`, builds the Inno Setup installer and the MSIX packages (§8), and uploads them as `installer-windows` and `msix`. Then on each runner it installs, checks and uninstalls the installer, also over a copy from the PowerShell line. It also signs a copy of the MSIX package with a throwaway certificate, installs it, runs it through its alias and removes it. |
| `macos` | `macos-26`, with smoke on it and `macos-26-intel` | The reusable `macos.yml` with `release-dist`, the version and `smoke-bundle`. It checks the darwin programs and notices, joins the programs with `lipo`, writes and ad-hoc signs `WOPR.app`, and makes the `.dmg`. It uploads the `.dmg` as `installer-macos`. Then on each runner it checks the image and runs the e2e test against the app's program. On `macos-26` it also opens the app as Finder does (§8). |
| `snap` | `ubuntu-24.04`, with tests on it and `ubuntu-24.04-arm` | The reusable `snap.yml` with `release-dist`, the version, `smoke-bundle` and `publish: false`. It runs `snapdir` and `snap pack` for each architecture, then the Store's `review-tools`, and uploads the snaps as `snaps`. Then on each runner it installs the snap, runs it and the e2e test under confinement, and removes it. Its publish job is skipped. |
| `collect` | `ubuntu-24.04` | What `release.yml`'s `collect` and `sign` do, without the uploads. `sizegate -files` on the installer and the `.dmg`, then `stage -merge`, then `sign-packages.sh` with a throwaway key on the result. That way, a problem shows on any push and not first on a tag. |
| `docs` | `ubuntu-24.04` | Builds the docs site (§13) as `docs.yml` does, with `--panicOnWarning --printPathWarnings`. A deprecated setting, a broken internal link, a missing screenshot or a malformed game data file fails it. Then builds it as for a signed release and as for an unsigned one (`HUGO_PARAMS_SIGNEDRELEASE`). For those builds, the job puts a throwaway key in `packaging/` while the real one is missing. It checks that only the signed build imports the key and publishes it. |
| `ci-ok` | `ubuntu-24.04` | `needs: [lint, secrets, test, build, smoke, windows, macos, snap, collect, docs]`, `if: always()` (and skipped, under an unevaluated expression name, with the rest on a PR from this repository). Fails unless every needed job succeeded. **This is the only required check** (S-3), so matrix names never appear in settings. |

Every CI run builds the installers and snaps, as for a tag (owner decision 2026-10-08). That way, a broken
installer shows on the branch that broke it. `windows.yml`, `macos.yml` and `snap.yml` are reusable
workflows, called with `$/` like `smoke.yml`. Each takes the `release-dist` artifact and the version, never
compiles wopr, and checks what it packs against `checksums.txt`. Their calling jobs grant `contents: read`,
because the called jobs check out the repository for the packaging files. `snap.yml`'s publish job is
skipped in CI. GitHub reports such a called workflow as a success to `ci-ok` (undocumented, but CI's runs
show it).

### 11.3 `release.yml` (on `v*` tags): gated, reproducible, attested (B-10, S-7)

`verify` → `build` and `rebuild` → `smoke`, `repro` and the installers → `collect` → `sign` → `publish` → `docs`
and the snaps.

1. **`verify`** (`contents: read`, `checks: read`) checks that the tag is semver and that the tagged commit
   is an ancestor of `origin/main`. It also checks that every `ci-ok` check run that GitHub Actions posted
   on that commit succeeded. It ignores a check of the same name from another app. A tag pushed right after
   its merge arrives before `main`'s CI run finishes. So `verify` waits for a missing or pending `ci-ok`, for
   up to 45 minutes. Any other result fails at once.
2. **`build`** (`contents: read`, no OIDC). First the release notes. `internal/tools/relnotes -version vX.Y.Z`
   writes `notes.md` (the release body), `CHANGELOG.md` and `changelog.yml` into `build/notes`.
   `changelog.yml` is in nFPM's chglog format, for the `.deb` and `.rpm` changelogs. `.gitignore` keeps
   `build/notes` out of the tree, which GoReleaser needs clean. GoReleaser's templates find `build/notes`
   through `WOPR_NOTES_DIR`. `nfpms.changelog` takes no template (GoReleaser v2.18.2), so `pkgdocs` copies
   `changelog.yml` to `build/pkg`, the fixed path that `nfpms.changelog` names. The job uploads `notes.md`
   for `publish`. The notes come from the release pull request's `.changes/vX.Y.Z.md`. If the pull request
   did not batch them, they come from `.changes/unreleased`, batched in a temporary copy, dated by the
   tagged commit, with a warning. relnotes refuses when an earlier tag has no `.changes/vX.Y.Z.md`, whose
   notes would repeat. It also refuses when another version is batched but not tagged, so the tag names the
   wrong version. And it refuses when there are no notes at all. It warns when the tagged commit holds notes
   the release does not list. It also warns when the version header's date is not the tagged commit's
   (AGENTS.md, "Releasing"). Only `vX.Y.Z` tags count as releases. Then
   `goreleaser release --clean --skip=publish`, then the size gate, then
   `stage -archives -assets dist/release`. `stage` checks that every archive contains `LICENSE`, `README.md`,
   `NOTICE.md` and `THIRD_PARTY_NOTICES.txt` (B-11). It checks that the Linux and macOS archives hold the
   manual page `wopr.6` at their root. It checks that the Windows ones do not (Windows has no `man`). It
   checks that each Linux build has one `.deb` and one `.rpm`, each named by its format's convention. It
   checks that each package installs what §8 lists (from the `.deb`'s control archive and the `.rpm`'s
   header). It collects every file of GoReleaser's that the release publishes into `dist/release` (`collect`
   adds the installers). Those files are the six archives, the six bare binaries, the four Linux packages,
   those four documents, and `checksums.txt`. A bare binary is a GoReleaser `binary`-format archive, named
   like the archives, `.exe` on Windows. GoReleaser writes `checksums.txt` over all of them
   (`checksum.extra_files` adds the documents). Each file must match its checksum line. The job uploads the
   staged files as `release-stage` and `dist/release` as `release-dist`. The job's `version` output is
   GoReleaser's `.Version`, which must be the tag without its `v`. It names the installers and snaps.
3. **`smoke`**: the reusable workflow, run on the staged release binaries.
4. **`rebuild`** runs beside `build`, not after it. It runs relnotes and GoReleaser again from a fresh
   checkout on `ubuntu-24.04-arm` (cross-compiling), and uploads its `checksums.txt`. relnotes takes every
   date from a version header or a commit, so both jobs package the same notes. **`repro`** then diffs the
   two `checksums.txt` files. Any difference fails the release.
5. **`windows`** and **`macos`**, beside `smoke`: the reusable workflows of §11.2 on `release-dist`
   (`macos` with `release-stage` for its e2e tests). They build and test the Windows installer and the
   MSIX packages, and the macOS disk image (§8). The MSIX bundle stays a workflow artifact, kept 90 days
   for a Store submission by hand.
6. **`collect`** (`contents: read`) downloads `release-dist`, `installer-windows` and `installer-macos`. It
   holds the installer and the `.dmg` to the size budget (`sizegate -files`), and adds them with
   `stage -merge`. `stage -merge` requires `dist/release` to match `checksums.txt` exactly and in
   GoReleaser's own format. It copies the two files into `dist/release`, adds their lines sorted as GoReleaser
   sorts them, and checks every file again. On an error it undoes everything. `collect` uploads the result as
   `release-files`. `repro` compared GoReleaser's `checksums.txt` before this, so the installers are signed
   and attested but not checked for reproducibility. They are built once, from files that `repro` covers.
7. **`sign`** (`contents: read`, the `release` environment, which holds the signing key and allows only `v*`
   tags), on a tag push only. It checks `release-files` against the rebuild's `checksums.txt` (every file
   the rebuild made must be there, byte for byte) and against its own. Then it runs
   `packaging/sign-packages.sh` (§8, "Signatures") and uploads the result as `release-signed`. The result is
   the signed `.rpm` packages, a `.deb.asc` beside each `.deb`, `checksums.txt` rewritten for them, and
   `checksums.txt.asc`. It is the only job with the key. It runs before `publish`, the only job with a
   write token.
8. **`publish`** (`contents: write`, `id-token: write`, `attestations: write`, `discussions: write`), on a tag
   push only, and only when every earlier job succeeded (`windows`, `macos`, `collect` and `sign` included).
   1. `sha256sum -c` on `release-signed`. Then `actions/attest@v4` with
      `subject-checksums: assets/checksums.txt`, so it attests every published file, the installers and
      the `.deb.asc` files included. Then `actions/attest@v4` again with `subject-path` for `checksums.txt`
      itself and `checksums.txt.asc`, so no release file lacks an attestation. Both attestations' Sigstore
      bundles, one per line, become the release file `wopr_<V>.intoto.jsonl`. `gh attestation verify
      --bundle` checks a file with it offline, and OpenSSF Scorecard's Signed-Releases check finds
      provenance only in a release file with that suffix (§11.6). It is the one release file outside
      `checksums.txt`, since it attests that file.
   2. `gh release create vX.Y.Z --draft --verify-tag --notes-file notes.md` with every file from
      `release-signed`. The release uses GitHub's generated notes instead, with a warning, only if
      `notes.md` is empty. `.goreleaser.yaml` disables GoReleaser's own changelog (CR-8): the notes are the
      change notes, not commit subjects.
   3. `gh release edit vX.Y.Z --draft=false --discussion-category Announcements`, which also starts the
      release's discussion (the job has `discussions: write`). If that fails, as when the category does not
      exist, it publishes without a discussion and warns.

   Immutable releases (§12) lock the release at publish time. A failed publish leaves only a draft.
9. **`docs`** (`actions: write` only), after a successful `publish`, runs
   `gh workflow run docs.yml --ref main`. That run publishes the docs site again, so that its download
   commands name the new release (§13). A run that `GITHUB_TOKEN` starts through `workflow_dispatch` is the
   one kind such a token may start.
10. **`snap`**, after a successful `publish`: `snap.yml` on `release-dist` and `release-stage`, with
    `publish: true` for a tag push and `SNAPCRAFT_STORE_CREDENTIALS` passed by name. An environment's secret
    reaches a called workflow only that way. It builds and tests the snaps, kept 90 days as `snaps`. Its
    publish job uploads them only when `SNAP_PUBLISH` is `true`, in the `snap-store` environment, after the
    owner approves (§8, AGENTS.md). It runs after `publish` so that the Snap Store never has a version the
    GitHub release lacks. The same order means the release never waits for that approval. `ci-ok`, which
    `verify` requires, already built and tested the snaps on the same commit.

**Dry run** (CR-1, IM-2). `workflow_dispatch` runs steps 2–6 and 10 with `--snapshot`, and never uploads the
snaps. GoReleaser in release mode refuses an untagged commit. `verify` is then skipped, so every later job
states its condition explicitly (`!cancelled()` and the results of its needs). Otherwise the implicit
`success()` would skip them too.

**Never re-tag.** The next patch version, plus a `retract` directive for `go install` users, fixes a bad
release. That rule is in `SECURITY.md` and `AGENTS.md`.

### 11.4 Runner labels

The labels as of 2026-10-07 (the M5 checklist's first run) are `ubuntu-24.04`, `ubuntu-24.04-arm`, `macos-26`,
`macos-26-intel`, `windows-2025` and `windows-11-arm`. All are free on public repositories, and none is
announced for removal. Ubuntu 26.04 became generally available on 2026-09-17, and `ubuntu-latest` moves to it
between 2026-10-19 and 2026-11-19. The pinned `ubuntu-24.04` labels do not move. `windows-2025` and
`windows-11-arm` now run the images with Visual Studio 2026 (the Arm label moved in September 2026).
`ubuntu-22.04` is deprecated and unsupported from 2027-04-17, and `macos-14` is unsupported from 2026-11-02,
so this project uses neither. The labels are checked at every milestone boundary (AGENTS.md checklist) (B-5).
The installer and snap workflows use the same six labels. The MSIX jobs also need a Windows SDK on the
Windows images (`WindowsSdk.psm1` takes the newest and fails clearly without one). The snap jobs need snapd,
which the `ubuntu-24.04-arm` partner image may lack, so `test-snap.sh` installs it when missing.

### 11.5 Dependency updates without Dependabot (R13)

- **`scheduled.yml`**, weekly, `contents: read` plus `issues: write` on the report job only (CR-6, TO-9). Each
  check has a rule that separates a **finding** from an **error**. A finding is something to update or fix,
  filed in the tracking issue. An error means the check could not run, which fails the job instead:
  - `govulncheck` on `main`: exit 3 is a finding, any other failure an error.
  - a newer **Go patch release** than `go.mod`'s `toolchain` line (`go list -m toolchain@patch`). The check
    exists because the response rule below counts from the release, whether or not govulncheck finds the
    fixed code reachable.
  - outdated **direct** requirements of the main module (`go list -m -u` with a template). Indirect upgrades
    are left to govulncheck and the manual cadence, or the report would never be empty.
  - outdated **tools**, selected by each tool module's `tool` pattern (they are indirect requirements there),
    and a newer **docs theme** than `site/go.mod` pins.
  - `prek update --check`: "would update" is a finding, "update failed" an error.
  - the third-party-notices check.

  A finding opens or comments on a single tracking issue. A week with none closes it. `scheduled.yml` never
  opens PRs, so "Allow GitHub Actions to create pull requests" stays off.
- **Manual cadence**, at every milestone and at least monthly: AGENTS.md's milestone checklist. It updates
  the modules: direct ones one at a time, to versions at least 14 days old. It never uses `go get -u ./...`,
  which lifts indirect modules to their newest untagged commits. It updates the tool modules (the toolchain
  first if a tool needs it, by the same rule). It also updates the docs theme and its vendored scripts, the
  hooks, the action SHAs (each at least 14 days old) and the installers' tools. It regenerates the notices
  and the icons. Each step uses a normal PR.
- **Gaps covered** (S-10). Public repositories disable schedules after 60 days without activity, so the
  milestone checklist includes "re-enable `scheduled.yml` if disabled". Dependency PRs also run `govulncheck`
  in `ci.yml`.
- **Response rule** (SECURITY.md). A reachable govulncheck finding, or a Go security release that affects the
  stdlib, means a `toolchain` bump and a patch release within 14 days.

---

### 11.6 `scorecard.yml` and `codeql.yml` (OpenSSF Scorecard)

[OpenSSF Scorecard](https://scorecard.dev/) scores the repository's supply-chain practices: pinned
dependencies, token permissions, branch protection, dangerous workflow patterns, signed releases and more.
`scorecard.yml` runs `ossf/scorecard-action` on each push to `main`, weekly, on a change to branch protection,
and by hand. It uploads the results as SARIF to code scanning, through `github/codeql-action/upload-sarif`.
With `publish_results: true`, it also publishes them to the Scorecard API, which serves the README's badge.
The job has `contents: read`, `security-events: write` (the upload) and `id-token: write` (publishing). The
Scorecard API refuses results from a workflow that breaks its restrictions: top-level `env` or `defaults`,
workflow-level write permissions, `run` steps, or actions other than checkout, upload-artifact,
upload-sarif and Scorecard's own. It uses `GITHUB_TOKEN` only, so the Branch-Protection check sees only
what that token can read.

`codeql.yml` runs CodeQL on the Go code, on every branch push, on pull requests from forks and weekly, and
uploads its findings to code scanning. It is the SAST tool that Scorecard's SAST check looks for. It is not
part of `ci-ok`. Known vulnerabilities in the tool modules that have no fix, and that wopr does not link,
are listed with a reason and an expiry in `osv-scanner.toml` beside each tool module's `go.mod`.
Scorecard's Vulnerabilities check runs osv-scanner, which reads those files. govulncheck still checks
wopr itself.

## 12. Security and public-repository hygiene

- **OpenSSF Best Practices.** WOPR holds the silver badge (project 15394 on bestpractices.dev).
  Compliance is a project goal. Each change is checked against the criteria before its PR opens. The gold
  badge is under consideration (`ROADMAP.md`). The gold criteria that need only repository changes are
  met: every source file carries SPDX copyright and licence lines (`internal/archtest/license_test.go`),
  code review is documented in `AGENTS.md`, and `GOVERNANCE.md` requires two-factor authentication.
- **Secrets.** None in the tree, ever.
  - gitleaks runs as a pre-commit hook (staged changes) and in the `secrets` job (the checked-out history,
    §11.2).
  - The response to a finding is to rotate the secret first, then add its fingerprint to `.gitleaksignore`
    in a PR. History on `main` is never rewritten (SECURITY.md, SL-2).
  - M0b proves the job with a **canary**: a throwaway branch with a fake secret that only gitleaks recognises.
    Push protection would stop a GitHub-supported token pattern. Its push run's log must say `leaks found`,
    not an error. Then the branch is removed, because a leftover branch would keep matching. AGENTS.md has
    the steps.
  - `.gitignore` adds:
    - `.env` and `.env.*`.
    - `*.pem`, `*.key`, `*.pfx`, `*.p12`, `*.jks` and SSH private keys (`id_rsa*`, `id_ecdsa*`, `id_ed25519*`,
      signing keys included).
    - `dist/` and `stage/`.
    - `build/` (release notes and package documents).
    - the installers and store packages (`*.dmg`, `*.msix`, `*.snap` and their kin).
    - go-winres's `.syso` files.
    - editor/OS files.
- **Repository settings** (S-3, S-7, S-8). The owner applies them (M0b, §15). AGENTS.md lists them.
  - **`main` ruleset**: pull request required, with a code owner's approval (`.github/CODEOWNERS`, decided
    2026-10-12). The owner is the only code owner, and the Repository admin role may bypass the rule for
    pull requests only, so that the owner can merge their own. Required check `ci-ok` (from GitHub Actions), with branches
    required to be up to date before merging (§11.2). No force-push. No deletion. Merges are **squash
    only**, which keeps `main`'s history linear. It does not keep v1's commits out of the public repository.
    PR #1's ref still serves them, and the v1 review quotes three fragments of the CC BY-SA map.
    `NOTICE.md` therefore credits Franklin Wei's map under CC BY-SA 4.0 for those fragments (SL-6, L-4).
  - **`v*` tag ruleset**: creation, update and deletion restricted, with only the owner able to bypass.
    **Immutable releases** are on.
  - **Secret scanning** and **push protection** must be enabled at repository level. Both were still off on
    2026-10-09. **Private vulnerability reporting**, which `SECURITY.md` sends reporters to, is on.
  - **Issues and Discussions**: issue forms for bug reports and feature requests (`.github/ISSUE_TEMPLATE/`).
    Blank issues are off, and there are links to Discussions, a private security report and the docs site.
    Discussions are on, since the forms, the docs site's Troubleshooting page and the README send questions
    there.
  - **Actions**:
    - allow only listed actions (`actions/*`, `j178/prek-action`, and for `scorecard.yml` (§11.6)
      `ossf/scorecard-action` and `github/codeql-action/upload-sarif`, and for `codeql.yml`
      `github/codeql-action/init` and `github/codeql-action/analyze`).
    - require actions pinned to a full-length commit SHA.
    - default `GITHUB_TOKEN` read-only.
    - workflows may not create or approve PRs.
    - approval required for **all** external contributors' workflow runs.
  - **Dependabot**: off (owner decision). govulncheck covers the Go graph.
  - **Stores**, only when the owner enables them (AGENTS.md has the steps). The settings are:
    - the `snap-store` environment (the owner as required reviewer, deployments from `v*` tags only), which
      holds `SNAPCRAFT_STORE_CREDENTIALS`. That secret is a Snap Store login limited to the `wopr` snap, its
      stable channel, and uploading and releasing. It expires within a year.
    - the repository variable `SNAP_PUBLISH`.
    - the Microsoft Store identity as the repository variables `MSIX_IDENTITY_NAME`, `MSIX_PUBLISHER` and
      `MSIX_PUBLISHER_DISPLAY_NAME`, which are not secret.
  - **Release signing**: the `release` environment (deployments from `v*` tags only) holds
    `WOPR_SIGNING_KEY` and, if the key has one, `WOPR_SIGNING_PASSPHRASE`. Only `release.yml`'s `sign` job
    sees them (§8, "Signatures"). No other secret exists.
- **The app.** No network code before M7, no telemetry, no transcripts. wopr sanitises external text. wopr's
  debug log holds no typed input (§8).
- **Releases.** Every file is attested, the Windows installer and the macOS disk image included. The MSIX
  packages and snaps are not release files. The Linux packages and `checksums.txt`, which lists every other
  file, are also signed with wopr's OpenPGP key (§8, "Signatures"). The Installation page says how to check
  them once the latest release is signed. Checking is optional, for those who want proof. The steps are in
  the Installation page's section "Verifying binaries" (its anchor is still
  `#verifying-binaries-attestation`), which the README and each release's notes link:
  - Command: `gh attestation verify <file> --repo GhostofGoes/WOPR`, plus
    `--signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml`,
    `--source-ref refs/tags/vX.Y.Z` and `--deny-self-hosted-runners`.
  - For a file downloaded with a browser, after the check: `xattr -d com.apple.quarantine`, or the
    SmartScreen prompt. The one-line installs download with tools that set no such mark.
  - The installer holds each `wopr.exe` unchanged, so an installed one passes the check. The Mac app's
    program joins both darwin builds and matches neither, so the `.dmg` is the file to check.
  - No program or installer has a code signature yet (owner decision 2026-10-08). The Windows installer
    meets SmartScreen, and Smart App Control blocks it outright. The Mac app is ad-hoc signed and needs
    Gatekeeper's Open Anyway once. The install tabs give the steps for both. Issues track Windows code
    signing, Apple's signing and notarization, the Microsoft Store (which signs MSIX packages itself), and
    Scoop and Chocolatey. There is no Homebrew tap: casks need signing, third-party taps need `brew trust`,
    and a tap needs a token beyond `GITHUB_TOKEN`.
- **Licensing** (L-1, L-4, L-5, B-11).
  - Code is MIT.
  - `NOTICE.md` lists:
    - the film quotations, excluded from the MIT grant, with a non-affiliation disclaimer.
    - the two abs0 items, with its BSD-2 notice, which covers the transcription. The text stays film text.
    - Matthew Thomas's world map with his line, and Natural Earth for GTW's outlines (§2.1, decision 26).
    - the CC BY-SA 4.0 credit for the map fragments in early drafts (SL-6).
    - the docs site's theme and vendored scripts (§13).
    - a contact for rights holders (SL-8), with AGENTS.md's takedown runbook behind it.
    - the brother's licence once granted.
  - `go run ./internal/tools/notices` generates `THIRD_PARTY_NOTICES.txt` from the **union** of
    `go list -deps` over the six release targets (`CGO_ENABLED=0`), so the output does not depend on the host
    (SL-1). The file holds each module's `LICENSE*`, `LICENCE*` or `COPYING*` file and Go's own licence. It
    is committed and embedded in the binary for `wopr --licenses` (through the root package, `legal.go`). It
    is checked up-to-date on Linux in CI and weekly, and shipped in every archive.
  - "WarGames" is used only nominatively, to name the film the program recreates. There are no film stills.
  - "WOPR" is a registered US mark held by an unrelated company (L-6). §16 records it as a low risk.

---

## 13. Documentation

- **README.md**:
  - What this is, and a link to the docs site (below).
  - Install, in a few lines. A link to the docs site's download buttons and one-line installs. The latest
    release, `go install …@latest`, and the Linux packages (since v0.4.0, and they install the manual
    page). The Windows installer, the Mac app and the packages' menu entry (from the release after v0.4.0).
    They put WOPR in the Start menu, Applications or the app menu. The installer and the app warn at first
    because they are unsigned. A link to the section on checking attestations (§12).
  - Quick start (`LOGON:`, `LIST GAMES`, `LOGOFF`). The shell's commands and keys ("what you type is kept")
    are on the docs site's Usage page.
  - Screenshots, the games, movie mode, flags and themes, and accessibility.
  - Troubleshooting, with a link to the docs site's page. Its topics: not a terminal (and mintty), too
    small, colours and `NO_COLOR`, the mouse-wheel note, and the debug log and seed for a bug report.
  - Development builds: artifacts from **`main` push runs only** (B-13).
  - Build from source (a link to the AGENTS.md Commands table), credits and licence, and a link to this plan.
- **AGENTS.md** (M0, updated each milestone). It is the single source for:
  - What the project is and the Commands table, per OS.
  - The import DAG and `archtest`, the single clock, and the protocol rules (`Think` captures values, Esc is
    the host's, input mode is dynamic).
  - How to add a game: catalog entry, package, testkit test, definition of done.
  - Goldens and `WOPR_UPDATE_GOLDEN`, and provenance tags.
  - Repository settings.
  - Change notes (how to add one, the style, when `Changelog: none` applies) and how to release.
  - The milestone checklist: dependency updates, the installers' tools, runner labels, re-enabling
    schedules, `prek update`.

  `CLAUDE.md` contains exactly `@AGENTS.md`.
- **Change notes and `CHANGELOG.md`** (owner decision 2026-10-07, after v0.2.0). Every change a player could
  notice adds a note to `.changes/unreleased/` with changie (pinned in `tools/release/go.mod`, with its
  config in `.changie.yaml`). The style is the owner's: plain words at an 8th-grade reading level, and what
  changed for the player, not why. A note describes a deep fix only by what the player saw ("Fixed a crash in
  some cases"). A note is one line of at most 76 characters, so that it fits a Debian changelog line. A
  release pull request batches the notes into `.changes/vX.Y.Z.md` and merges `CHANGELOG.md`. Nobody edits
  it by hand. `release.yml` turns the same files into the GitHub Release notes and the package
  changelogs (§11.3). The notes for v0.1.0 and v0.2.0 were written afterwards from their pull requests, in
  the same style. The owner's part of a release is to merge that pull request and push the tag (AGENTS.md,
  "Releasing").
- **Docs site** (owner decision 2026-10-07): <https://ghostofgoes.github.io/WOPR/>, built from `site/` with
  Hugo and the Hextra theme. Hugo is the standard edition, pinned in `tools/docs/go.mod` (no Node, no Sass).
  Hextra is a Hugo module pinned in `site/go.mod`, under MIT, and credited in NOTICE.md and on the site's
  credits page. Pages: home, quickstart, installation, usage (with accessibility and troubleshooting under
  it), games (an index and one page per game), movie scenes, contributing with the code of conduct,
  changelog, and credits.
  - **Install in one line** (owner decision 2026-10-08). The command lines are Windows, macOS, Linux (any),
    Linux (apt), Linux (RPM) and Go. Each is one line for a person who has never used a terminal. Each uses
    only the tools its system installs by default (the Go tab needs Go). The lines download the latest
    release's file and put the program on the `PATH`. All but Go's start it. The macOS and Linux (any)
    lines install the bare binary into `~/.local/bin` without `sudo` (owner decision 2026-10-10), over
    HTTPS only. When the shell does not already have that folder on the `PATH`, they add it in
    `~/.zprofile` or `~/.bash_profile` (macOS), or `~/.bashrc` or `~/.zshrc` (Linux). Only the package
    lines need root. Since the download buttons (next), they are the Installation page's "Command line
    install methods" section, after Uninstall.
    How to check an attestation is the Installation page's next-to-last section, for those who want it,
    before development builds. The README links there and to the guide.
  - **Download buttons first, no command line** (owner decision 2026-10-10). Quickstart and Installation lead
    with the same three tabs: Windows, macOS and Linux. Each tab has a button that downloads the latest
    release's file, then each click in order, warnings included:
    - For the Windows installer: the browser's warning, SmartScreen's More info and Run anyway, and Smart
      App Control's block.
    - For the Mac app: dragging to Applications first, then Gatekeeper's Done and Open Anyway in Privacy &
      Security.
    - For Linux: the `.deb` and the `.rpm`, opened in the desktop's software app. The tab points to the
      command lines for other distributions and the atomic desktops.

    Each tab has two buttons, side by side, with no file name under them (owner decision 2026-10-10). They
    are "Intel/AMD" and "ARM" on Windows and Linux, and "Apple Silicon" and "Intel" on macOS. The Windows
    installer and the Mac app are one file for both, so their two buttons link to the same file. A button is
    a plain link, at least 44 px tall, in Hextra's primary colour. The `install-tabs` shortcode draws it from
    a placeholder in the tab's text. Until the latest release has the installer and the Mac app, the Windows
    and macOS tabs say they come with the next release and point to the command lines. Every release after
    v0.4.0 has them (the `has-installers` partial). A small script (`site/assets/js/install-platform.js`)
    picks the tab for the reader's system, unless the reader picked one before. The tab is Windows, macOS or
    Linux. Among the command lines, it is Linux (apt) or Linux (RPM) when the browser names the distribution.
    The script also hides the download button that does not fit the reader's processor. It hides it when the
    browser says the processor for certain (client hints, or Firefox's user agent on Linux). Both buttons
    stay when it cannot tell, and without JavaScript. Phones, tablets and ChromeOS keep the first tab, as
    does a browser without JavaScript. The site names the packages' menu entry only from the same release as
    the installers (v0.4.0's packages have none). Three other places show their new parts only under the
    same gates. The gates are `if-installers` for the installers and the menu entry, and `if-packages` for
    the rest of the packages. The places are the Installation page's uninstall steps (the download buttons'
    first, then the command lines'), the troubleshooting page and the credits page (Inno Setup). The
    troubleshooting page's new parts are SmartScreen, Smart App Control, Gatekeeper, App Translocation and a
    missing menu entry. AGENTS.md ("Docs site") has the placeholders that keep each text written once.
    Nothing names the snap until it is published.
  - **One source for everything.** A content adapter (`site/content/games/_content.gotmpl`) builds each
    game's page from `site/data/games/<slug>.json`, and the manual page (`docs/man/wopr.6`) comes from the
    same files. The site mounts and renders `CHANGELOG.md`, `CODE_OF_CONDUCT.md`, `NOTICE.md` and AGENTS.md's
    Commands and Pull requests sections, and does not copy them. It mounts `docs/screenshots/` and the
    manual page the same way. The download commands name the latest release (below). The `.deb` and `.rpm`
    instructions appear, through the `if-packages` shortcode, once that release has packages. That is any
    release after v0.3.0, which also has `wopr.6` in its archives. The instructions for the installer, the
    Mac app and the packages' menu entry appear the same way (`if-installers`). The site's favicons and
    navbar logo are the program's icon, drawn by `internal/tools/icons` (§8). Tests in `internal/cli` check
    that the usage page lists every option and environment variable, and that the movie page lists every
    scene.
  - **Strict build.** The build runs with `--panicOnWarning` (deprecations included) and
    `--printPathWarnings`. An internal link to no page or file, a missing screenshot or a game file without
    its required fields is an error. CI's `docs` job builds it on every push (§11.2).
  - **Deploy from `main`** (owner decision 2026-10-07). `docs.yml` runs on every push to `main` and by hand
    (`workflow_dispatch`), and `release.yml` starts it after publishing a release (§11.3). It builds the site
    from `main` and deploys it to GitHub Pages (`actions/configure-pages`, `actions/upload-pages-artifact`,
    `actions/deploy-pages`). So a change to the site goes live when its pull request merges. The download
    commands name the latest *published* release. `docs.yml` asks the GitHub API for it and passes it to
    Hugo as `HUGO_PARAMS_LATESTRELEASE`. (Review of 2026-10-07: a release pull request puts its version in
    `CHANGELOG.md` minutes before the release exists. So a site that read the version from there named a
    download that was not there yet.) Without that variable, as in CI's builds and a preview, the version
    comes from `CHANGELOG.md`. The site can describe an option before a release has it, as most projects'
    docs do. The build job has `contents: read` and `pages: read`. The deploy job has `pages: write` and
    `id-token: write`, in the `github-pages` environment. The deploy job alone is in concurrency group
    `pages`, and it never cancels a deployment in progress. So a run on another branch cannot replace one
    from `main` that is waiting. There is no build cache. Pages must be set to deploy from GitHub Actions
    (AGENTS.md, repository settings).
  - **Third-party files, vendored** (review of 2026-10-07: downloading them from jsDelivr at build time made
    `ci-ok`, and so every release, depend on a CDN. Nothing pinned the bytes it served). Hextra's search
    runs on FlexSearch 0.8.143 (Apache-2.0). Its lightbox, which shows a screenshot full size, runs on
    PhotoSwipe 5.4.4 (MIT). `site/assets/vendor/` holds their files and each package's `LICENSE`, byte for
    byte as the npm tarballs publish them. `.gitattributes` keeps git and prek's fixers from changing
    them. `site/hugo.yaml` names the files (`params.gallery.js`, `lightboxJs` and `css`, and
    `params.search.flexsearch.js`) and records each tarball's integrity hash. The credits page shows both
    licences from those files. The theme's own stylesheet and scripts come from the module. The build
    fetches nothing but Go modules, and no page loads anything from another site.
  - **Game pages' layout.** The summary, how to start the game and the other names it answers to, and the
    screenshots (one set in the lightbox). Then how to play, the controls, every tip, and a link back to the
    games index. The index lists the games in `LIST GAMES` order with tic-tac-toe last.
- **Game pages** (`site/data/games/<slug>.json`, one per game, named by the slug `wopr --games` shows).
  Each holds the summary, how to play, controls, tips and captioned screenshots (`site/static/img/games/`),
  written from the code. The docs site's game pages and the manual page come from them. Nothing else
  repeats them.
- **Manual page** `docs/man/wopr.6`, section 6. `internal/tools/manpage` generates it from the flag table
  (`cli.Flags`), the catalog, the game pages and the movie's scenes, and CI checks it with `-check`.
  Its date comes from the newest `.changes/vX.Y.Z.md`, never from the clock. It ships at the root of the
  Linux and macOS archives, and in the `.deb` and `.rpm` as `/usr/share/man/man6/wopr.6.gz`. So `man wopr`
  works once a package is installed. `mandoc -T lint -W all` and `groff -man -ww` pass it clean. What ships
  is `pkgdocs`'s copy, whose header names the version being built.
- **Screens**: Appendix C draws the greeting and names the goldens for the rest. The real screens are goldens,
  which the tests keep current: `internal/ui/testdata/gtw_screens.golden` and `card_screens.golden` (Hearts,
  Gin Rummy, Bridge and Falken's Maze at 80×24).
- **This plan** is updated at milestone boundaries. Superseded text is removed.

---

## 14. Verification (per release)

1. `ci-ok` is green on the tagged commit. The release pipeline's `verify`, `build`, `smoke` and `repro` jobs pass.
   The six archives each contain the four notice files and are within the size budget. The four Linux and
   macOS archives also hold `wopr.6`. The four Linux packages pass `stage`'s checks. The smoke jobs install,
   run and remove them on Ubuntu (`.deb`) and Fedora (`.rpm`), menu entry, icons and metadata included. The
   `windows`, `macos` and `collect` jobs pass. On both architectures, they install, run and remove the
   installer. On both, they also mount the `.dmg` and run its app, and on Apple silicon they open the app as
   Finder opens it. Both fit the size budget and are in `checksums.txt`. The `msix` and `snaps` artifacts
   exist.
2. `wopr -v`, `-h`, `-g`, `-L` and their long forms print plain text and exit 0 without a terminal.
   `wopr --games | head -1` exits 0 on every OS.
3. `wopr`:
   1. LOGON → `FALKEN'S MAZE` → `IDENTIFICATION NOT RECOGNIZED` → re-dial.
   2. Two more wrong names → the hint.
   3. `Joshua` → the greeting, answering its questions (`Hello.`, `I'm fine. How are you?`, `People sometimes
      make mistakes.`) as in Appendix C.
   4. `Love to. How about Global Thermonuclear War?` → the chess offer.
   5. `Later. Let's play Global Thermonuclear War.` → `FINE.` and GTW starts.
   6. Esc, Esc → back in the Shell with a remark.
   7. `list games` → `7` → the chess board. WOPR shows `PROCESSING` and Ctrl+C stays responsive.
   8. `LOGOFF` → exit 0.
4. The film path under `-s 1 -i` with scripted input (the `testkit` golden `film_path`): GTW → climax → NORAD
   notices → `TIC-TAC-TOE` → 0 players → montage → `Hello.` → `A STRANGE GAME…` → chess offer armed.
   `wopr -p 15`, `wopr -p "global thermonuclear war"` and `wopr gtw` resolve to the same game.
   `wopr -p theaterwide` and `wopr gtw chess` exit 2.
5. `go test -race ./...` (plain `go test` on Windows) is green on the three test OSes. Goldens are stable.
   `archtest` and the lint self-test are green.
6. The e2e test passes on all six smoke runners. A manual pass in Windows Terminal and conhost checks colours,
   cursor, resize, Ctrl+C restore and SmartScreen.
7. A manual pass of the installers, which CI cannot click through, on a download from the release page:
   - On Windows: the browser's warning, SmartScreen, the wizard, the Start menu entry opening a terminal, and
     uninstalling from Settings.
   - On a Mac: dragging to Applications, Gatekeeper's Open Anyway, the hand-off to Terminal, and whether
     macOS asks again when Terminal runs the program.
   - On Ubuntu and Fedora desktops: opening the downloaded package, the menu entry and its right-click
     actions.
8. On a tag, `gh attestation verify` with the flags in §12 succeeds for each file, and the release is
   immutable.
9. `wopr --movie` opens the scene menu and every scene plays. `wopr -m 2 -i` plays from scene 2 to
   the end of the list, pausing between pages, and exits 0. `wopr -m 2 -o` plays scene 2 alone and exits 0.
   `wopr --scenes` lists them. The movie consistency tests are green.

---

## 15. Milestones

| M | Deliverable | Exit criteria | Release |
|---|---|---|---|
| 0 | **Scaffold** (done): go.mod, `.gitattributes`/`.gitignore`, LICENSE/NOTICE/SECURITY/CONTRIBUTING/CODE_OF_CONDUCT, `legal.go`, `version`, `proto` (types), `games` (types + catalog with 16 `Planned` entries), `cli`, `theme`, `ui` hello-world, `archtest` + lint fixture, `golden`, `tools/*`. `prek.toml`, `.golangci.yml`, `.goreleaser.yaml`, `ci.yml`/`smoke.yml`/`release.yml`/`scheduled.yml`. README (basic), **AGENTS.md** + `CLAUDE.md`. The M0 e2e cases. | `ci-ok` is green with all six smoke runners. The lint fixture fires every rule. A **snapshot** dry run of `release.yml` passes `build`/`smoke`/`repro` (CR-1). | — |
| 0b | **Owner settings** (IM-3): the repository settings in AGENTS.md, and the gitleaks canary branch turned red for a finding. | Settings applied, canary done. Still open on 2026-10-09: the `main` and `v*` rulesets, squash-only merging, secret scanning and push protection, and the canary. Immutable releases, private vulnerability reporting and Discussions are on. | — |
| 1 | **Console, persona and protocol.** `proto/host` runner, `games/testkit`, clock, typewriter, line editor, `Sanitize*`, scrollback, canvas renderer, front panel. `prompt` (menus, yes/no and clauses, moved from M3, P-4), the ui adapter and its synchronous driver. The persona (session, LOGON table, greeting scene, commands, intent, offers, scripted brain, lines with provenance). Themes with ANSI/ASCII fallbacks and contrast tests, too-small handling, a `gamestest` stub exercising every Output, goldens, fuzz, the M1 e2e cases. 80×24 mockups for chess and the GTW phases (now the goldens Appendix C names). | Every M1 golden flow and the e2e cases marked M0–M1 are green on all runners. | v0.1.0 |
| 2 | **Film set pieces.** `games/ai`, `board/`, tic-tac-toe, checkers, chess, **GTW** (§6.2, with the climax table). **`games/ending`** (it reuses tic-tac-toe and owns the launch-code display), the internal `ending` registry entry, the persona's remark for an abandoned war. Random session seeds and the debug log, the GTW maps (decision 26), abs0's 157 scenario names verbatim. **Built**. QA on all OSes remains. | The `film_path` and climax goldens, quality tests, and the manual QA list. | **v0.1.0** |
| 3 | **Card games.** `cards/` + `trick.go`. Black Jack, Poker, Gin Rummy, Hearts, **Bridge (minimal, last)**. The Hearts mockup (the `card_screens` golden). **Built**. QA on all OSes remains. | Definition of done per game. | v0.1.0 |
| 4 | **Sims and maze.** Sim engine spec → engine → four scenarios + two bespoke sims. Falken's Maze. **Built**. QA on all OSes remains. | Definition of done per game. | v0.1.0 |
| 5 | **Polish.** The film text pass (**done** 2026-10-11, §2.3): every line checked against the film's transcriptions and tagged `film`, the montage names, the three conflicts and the status burst's colon settled, the movie scene scripts fixed, and the film's lines frozen (R15). What only the film itself can settle stays `reconstructed` (§2.3 lists it). **GTW's turn-based DEFCON exchange** (§6.2, decision 24, **built**). README completed with screenshots (**done** 2026-10-07: six in `docs/screenshots/`, recorded from the program in an 80×24 terminal). Accessibility pass (**done**: §4.5 records what was checked, what changed and what is left for a screen-reader user). Dependency and runner checklist (last run 2026-10-07). | Every film line is `film`, except what §2.3 lists as open. The turn-based exchange cannot be won and its film path stays short. The checklist is run again just before tagging v1.0.0. | **v1.0.0** |
| 6 | **Movie mode** (§7): `-m/--movie`, the host hooks, director, scenes, scene menu. `-S/--scenes` (the owner's request for a scene list) and `-o/--only` (the owner's request to play one scene and stop). Consistency tests, e2e cases. **Built**. QA on all OSes remains. | All scenes play. The consistency test is green. Size re-checked (every target passes the gate). | v0.2.0 |
| 7 (opt) | **LLM brain** (§4.7): opt-in, `net/http`, hardened client, effects allowlist, scripted fallback. | Fuzzed reply parser. Size gate. Offline behaviour unchanged. | after v1.0.0 |

The installers (R14, owner decision 2026-10-08) belong to no milestone. They ship with the next release. The
store listings wait for the owner's accounts, and the Microsoft Store's for v1.0.0 (§8).

Effort is not estimated per game (P-3).

**Third-party permission gate** (L-3, L-5). Lines adapted from the brother's prompt carry the `prompt` tag.
They are not committed at all until he gives written permission and a licence. The gate blocks those lines
only.

---

## 16. Risks

| # | Risk | Likelihood / impact | Mitigation |
|---|---|---|---|
| RK-1 | Film text is MGM's expression. Movie mode (M6) ships the film's complete WOPR terminal script, much of which the persona already contains (SL-8). | Medium / medium | On-screen terminal text only, with no spoken-only dialogue, stills or audio. Provenance per line. `NOTICE.md` exclusion, disclaimer and rights-holder contact. Nominative use of "WarGames". A takedown runbook in AGENTS.md (delete releases, patch, `retract`). Copies remain in git history and the module mirror. The owner accepts the residual risk. |
| RK-2 | "WOPR" is a registered US mark (Frontier Technology, class 42). | Low / medium | Different class (SaaS vs a free game). No logo imitation. Revisit if contacted. Not legal advice. |
| RK-3 | Bubble Tea v2 patch churn (v2.0.10 in under a year). | Medium / low | Pin. Read release notes on each update. Goldens catch rendering changes. |
| RK-4 | Hosted runner labels retire. | High / low | Pinned labels. Checklist at each milestone. The oldest Linux claim rests on Go's kernel floor, not on a runner (§1). |
| RK-5 | Unsigned binaries and installers trigger Gatekeeper and SmartScreen friction, and Smart App Control blocks them outright. | High / medium | Attestations (the docs say checking them is optional). The install tabs walk through each warning (§13). Issues track signing and the stores (§12). |
| RK-6 | No Dependabot means updates lag. | Medium / medium | Weekly scheduled report issue. govulncheck in CI. Monthly manual cadence. 14-day response rule. |
| RK-7 | 16 games is a large scope. | Retired | All 16 games are playable, and shipped in v0.1.0. |
| RK-8 | The chess library fork goes stale. | Low / low | Small API surface used (rules, SAN/UCI). The search is our own. |
| RK-9 | Tool dependency graphs conflict (golangci-lint's broke gitleaks's build, SL-2). | Medium / low | One tool module per conflicting tool. The CI jobs build each tool on every run, so a break shows at once. |
| RK-10 | The installers' first runs on real machines are untested. In particular: Gatekeeper's dialogs, Smart App Control, which terminal a Linux desktop opens, and App Center's Open button for the snap. | Medium / medium | CI installs and runs each on its own platform. A manual pass per release (§14). The docs hedge what is unconfirmed. |
| RK-11 | The stores refuse or pull WOPR. The Snap Store reviews new names by hand. The Microsoft Store's content policies cover the film quotations and the registered mark (RK-1, RK-2). | Medium / low | Publishing stays off until the owner decides. The GitHub release stays the main channel. |

Risk ids are `RK-n`, so they do not collide with the requirement ids (R1…R14) or the v1 review's `R-n` findings
(IM-12).

---

## Appendix B — Canonical screen text (provenance-tagged)

The code holds this text and its tags, and is authoritative: `internal/wopr/lines.go`, `internal/assets/`, the
games' packages and `internal/movie/scenes/`. Each package's `testdata/film_lines.golden` lists its film lines.
Tags: **F** = `film`, the film's screen text character for character, never modified (R15). **R** =
`reconstructed`, from one transcription only. **A** = `third-party:abs0/wargames@010ed92:wargames.sh` (film
text as transcribed by abs0, credited). **O** = `original`. Two spaces follow `LOGON:` and every full stop.

**LOGON gate** (F). The refusal keeps the system's own misspelling:

```text
LOGON:  
INDENTIFICATION NOT RECOGNIZED BY SYSTEM
--CONNECTION TERMINATED--
HELP NOT AVAILABLE
'GAMES' REFERS TO MODELS, SIMULATIONS AND GAMES
WHICH HAVE TACTICAL AND STRATEGIC APPLICATIONS.
```

After `HELP GAMES` there is no `LOGON:`: `List Games` is typed on an open line. The film's log-on attempts
are `000001`, `Help Logon`, `Help Games`, `Falkens-Maze`, `Armageddon` and `Joshua`.

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

**Backdoor header** (A, byte-identical to abs0's transcription). A status burst follows it, then a clear. The
burst is R: the film's phrases in our own two-column layout, not an abs0 item (SL-5). It consists of
`(311) 936-2364`, `SYSPROC FUNCT READY` / `ALT NET READY` and `CPU AUTH RY-345-AX3` /
`SYSCOMP STATUS:  ALL PORTS ACTIVE`. The screens flash by, and the transcriptions read them differently.

```text
#45     11456          11009          11893          11972        11315
PRT CON. 3.4.5.  SECTRAN 9.4.3.                      PORT STAT: SD-345

(311) 699-7305
```

**Greeting** (F), with David's lines as he types them:

```text
GREETINGS PROFESSOR FALKEN.
Hello.
HOW ARE YOU FEELING TODAY?
I'm fine.  How are you?
EXCELLENT.  IT'S BEEN A LONG TIME.  CAN YOU EXPLAIN
THE REMOVAL OF YOUR USER ACCOUNT NUMBER ON 6/23/73?
People sometimes make mistakes.
YES THEY DO.  SHALL WE PLAY A GAME?
```

**GTW-vs-chess** (F): `Love to.  How about Global Thermonuclear War?` → `WOULDN'T YOU PREFER A GOOD GAME OF
CHESS?` → `Later.  Let's play Global Thermonuclear War.` → `FINE.`

**Global Thermonuclear War** (F). The nations' names under the outlines, then:

```text
WHICH SIDE DO YOU WANT?

  1.    UNITED STATES
  2.    SOVIET UNION

PLEASE CHOOSE ONE:  2

AWAITING FIRST STRIKE COMMAND
-----------------------------

PLEASE LIST PRIMARY TARGETS BY
CITY AND/OR COUNTY NAME:

Las Vegas
Seattle
```

The `TRAJECTORY HEADING` heading is the film's; the table's values are generated (O).

**The game clock** (F), on David's screen at home and on McKittrick's at the end of the NORAD session. Movie
mode ticks the seconds on from the film's (the ticking is O):

```text
------------------------------------------------------------
    GAME TIME ELAPSED           ESTIMATED TIME REMAINING

    31 HRS  12 MIN  50 SEC      52 HRS  17 MIN  15 SEC
------------------------------------------------------------
```

At NORAD the readings are `45 HRS  34 MIN  09 SEC` and `27 HRS  58 MIN  17 SEC`.

**Call-back, at David's home** (F). WOPR phones David. His lines are `Incorrect identification.  I am not
Falken.`, `Falken is dead.` and, twice, `What is the primary goal?`. WOPR's lines:

```text
I'M SORRY TO HEAR THAT, PROFESSOR.
YESTERDAY'S GAME WAS INTERRUPTED.
ALTHOUGH PRIMARY GOAL HAS NOT YET
BEEN ACHIEVED, SOLUTION IS NEAR.
YOU SHOULD KNOW, PROFESSOR.  YOU
PROGRAMMED ME.
TO WIN THE GAME.
```

**The NORAD terminal session** (F), later, after `Joshua` and `Hello, are you still playing the game?`.
`WHAT'S THE DIFFERENCE?` answers `Is this a game or is it real?`, and the address answers `What classified
address?`:

```text
OF COURSE.  I SHOULD REACH DEFCON 1 AND
LAUNCH MY MISSILES IN 28 HOURS.

WOULD YOU LIKE TO SEE SOME PROJECTED KILL RATIOS?
WHAT'S THE DIFFERENCE?
YOU ARE A HARD MAN TO REACH.  COULD NOT FIND
YOU IN SEATTLE AND NO TERMINAL IS IN
OPERATION AT YOUR CLASSIFIED ADDRESS.
DOD PENSION FILES INDICATE
CURRENT MAILING AS:

DR. ROBERT HUME (A.K.A. STEPHEN W. FALKEN)
5 TALL CEDAR ROAD
GOOSE ISLAND, OREGON 97014
```

**Kill-ratio table** (F). Three sections, the United States on the left and the Soviet Union on the right,
whichever side was taken. The film repeats the nations' names above each section and leaves a blank line
under each rule; the board's 19 rows have no room for that. The frame cuts the Soviet column's percentages
off, so its heading is taken to repeat the United States':

```text
 UNITED STATES                                         SOVIET UNION
UNITS DESTROYED            MILITARY FORCES            UNITS DESTROYED
--------------------------------------------------------------------------------
      68%                  BOMBERS
      54%                  ICBM'S
      12%                  ATTACK SUBS
      39%                  TACTICAL AIRCRAFT
      58%                  GROUND FORCES
UNITS DESTROYED            CIVILIAN ASSETS            UNITS DESTROYED
--------------------------------------------------------------------------------
      69%                  HOUSING
      22%                  COMMUNICATIONS
      45%                  TRANSPORTATION
      70%                  FOOD STOCKPILES
      89%                  HOSPITALS
UNITED STATES              HUMAN RESOURCES            SOVIET UNION
--------------------------------------------------------------------------------
 49 MILLION                NON-FATAL INJURED           65 MILLION
 72 MILLION                POPULATION DEATHS          110 MILLION
```

The figures above are the film's. Movie mode shows them where the frame shows them, with the game's own
Soviet percentages. In play, every figure comes from the exchange (O): `BOMBERS` and `ICBM'S` are the
losses as a percentage of the force, and the other rows scale with the warheads that landed (§6.2).

**NORAD notices** (F): `** GAME ROUTINE RUNNING **`; `** IDENTIFICATION NOT RECOGNIZED **` /
`** ACCESS DENIED **`; `** IMPROPER REQUEST **` / `** ROUTINE MUST COMPLETE BEFORE RESET **` /
`** ACCESS DENIED **`. The film centres each on an empty screen and underlines the first line; the board's
strip shows the words. Launch code `CPE1704TKS` (F; the control panel groups it `CPE 1704 TKS`). DEFCON
ladder 5 → 1. `LIST GAMES` is typed in capitals (F); `CHESS`, `GTW` and `TIC-TAC-TOE` are R, from abs0.

**Tic-tac-toe** (R, abs0's notes only, in the line breaks of the big screen). `1` and `ZERO` are what David
types:

```text
ONE OR TWO PLAYERS?
PLEASE LIST NUMBER
OF PLAYERS: 1
STALEMATE.
WANT TO PLAY AGAIN?
ZERO
```

**Ending** (F). The montage's headings `STRATEGY:` and `WINNER:`, with `NONE` on every row, then:

```text
GREETINGS PROFESSOR FALKEN.
Hello.
A STRANGE GAME.
THE ONLY WINNING MOVE IS
NOT TO PLAY.

HOW ABOUT A NICE GAME OF CHESS?
```

The last `Hello.` is R: abs0's notes have `HELLO`.

**Montage scenario names** (A, decision 18). `internal/assets/scenarios.go` holds all 157 names, from
`U.S. FIRST STRIKE` on, in abs0's order, some truncated at 21 characters or misspelt as the film shows them.
Thirteen are spelt as filmjournals' and the Reddit list's readings have them, where both differ from abs0's
(§2.3). Eleven slots where the three readings differ remain abs0's.

**Original lines** (O): the LOGON hint, `WHICH GAME?`, `** GAME ROUTINE NOT AVAILABLE **`,
`** REQUEST CANCELLED **`, `** PRESS ESC AGAIN TO END GAME **`, the `HELP` command list (including
`<NUMBER>  PICK FROM THE LIST JUST SHOWN`), the GTW climax hints, the climax tic-tac-toe's WOPR-win line, the
remark after an abandoned war, `PROCESSING`, `SHALL WE PLAY A GAME?` and `WINNER: NONE` on their own, and
the fallback `PLEASE RESTATE YOUR REQUEST, PROFESSOR.` GTW's exchange, with `#` filled in order:

```text
STRIKE # OF # [# # #]:
FIRST STRIKE LAUNCHED. #.
STRIKE # LAUNCHED. #.
NO LAUNCH ORDERED.
NOTHING LEFT TO LAUNCH.
ENEMY LAUNCH DETECTED. #. DEFCON #.
ENEMY BOMBERS INBOUND. DEFCON #.
FULL-SCALE ENEMY ATTACK. #. DEFCON 1.
LOST ON THE GROUND: # #  # #. WARHEADS ON CITIES: # #  # #.
STRIKE ASSESSMENT COMPLETE. PRESS ENTER FOR PROJECTED KILL RATIOS.
WOPR HAS LAUNCH AUTHORITY.
ORDER NOT RECOGNISED. TYPE HELP.
PERCENTAGES ARE WHOLE NUMBERS FROM 0 TO 100.
ICBMS HIT ENEMY SILOS. SLBMS HIT YOUR TARGETS. BOMBERS ARRIVE AT DEFCON 1.
HELD ICBMS AND BOMBERS CAN BE DESTROYED ON THE GROUND. SUBS ARE SAFE AT SEA.
ORDER PERCENTAGES: 50 50 100, ICBM 50, ALL, HOLD OR AUTO. ENTER IS THE PLAN.
```

The `#` lists name the systems `ICBM`, `SLBM` and `BOMBERS`. On the board: the `FORCES` table (`FORCES`,
`ICBM`, `SLBM`, `BMB`, `AIR`, the nations `US` and `USSR`), the designators `MM3`, `C4`, `SS20` and `SSN8`,
`ORDERS: PERCENT OF ICBM SLBM BOMBERS, ALL, HOLD, AUTO, HELP.`,
`LAST ORDERS: AT DEFCON 1 WOPR FIRES EVERYTHING.` and the legend `OUTGOING +   INCOMING *   IMPACT X`. The
force levels, the doctrine and the kill-ratio formulas are original game design, not film or historical
figures.

---

## Appendix C — Target screens (80×24)

Each screen is exactly 24 rows of at most 80 columns, with the geometry of §4.3. Only the greeting is drawn here
(`█` is the cursor). The goldens named below hold the other screens as built, and the tests keep them current.

**Backdoor greeting** (`imsai`). WOPR is upper case and revealed at modem speed. The user's input is echoed in
mixed case. At the film's spacing the exchange may scroll. That is fine:

```text
GREETINGS PROFESSOR FALKEN.

Hello.

HOW ARE YOU FEELING TODAY?

I'm fine.  How are you?

EXCELLENT.  IT'S BEEN A LONG TIME.  CAN YOU EXPLAIN
THE REMOVAL OF YOUR USER ACCOUNT NUMBER ON 6/23/73?

People sometimes make mistakes.

YES THEY DO.  SHALL WE PLAY A GAME?

Love to.  How about Global Thermonuclear War?

WOULDN'T YOU PREFER A GOOD GAME OF CHESS?

Later.  Let's play Global Thermonuclear War.

FINE.

█
```

**GTW side choice** (`norad`, `LayoutConsole`, front panel shown). The two nations' outlines and their names are
printed as console text before the question, as in the film (RF-3). As built: `side choice` in
`internal/ui/testdata/gtw_screens.golden`.

**GTW big board** (`norad`, `LayoutFull`, front panel shown): the view takes `H' − 4` = 19 rows, then the 3-row
strip, the input row and the panel (AR-11). As built at the strike 2 prompt (USSR, Las Vegas and Seattle, after
the first strike): `big board: strike 2 orders` in the same golden. That golden shows the rest of the exchange
too. Outgoing tracks draw `+` and incoming tracks draw `*`. An impact is a reversed `X`, drawn over every
track. Earlier strikes keep only their impacts. The current DEFCON level is pointed at (`>`) and reversed.

**Chess** (`imsai`, `LayoutPanel`, `PanelRows` 12, no front panel): 12 rows of board, then the 12-row console
strip. The board is framed, its dark squares shaded and each square two columns wide, beside an original title
piece (decision 26, §6.1). `internal/games/chess/testdata/view.golden` holds it as built. v2's mockup drew the
board gridless and unshaded. The frame and shading came later, at the owner's request.
