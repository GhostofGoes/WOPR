# WOPR — Architecture & Scope Plan (v2.1)

_Status: v2.1, 2026-10-06. v2 plus the fixes from its review. Supersedes v1 (commit `ada63a0`, which never reached
`main`). [`docs/reviews/PLAN-v1-review.md`](reviews/PLAN-v1-review.md) is the adversarial review of v1, and
[`docs/reviews/PLAN-v2-review.md`](reviews/PLAN-v2-review.md) reviews v2. Finding ids are cited inline where the plan
answers them: v1 ids look like A-3, v2 ids look like AR-1. Implementation follows the milestones in §15._

**How to read this plan.** It states intent, contracts and constraints. M0 has landed, so the files in the
repository are authoritative for configuration (`prek.toml`, `.golangci.yml`, `.goreleaser.yaml`, the workflows,
`tools/*/go.mod`), and the M0 seeds that v2 carried in Appendix A are gone (M-2). Likewise the import table in
`internal/archtest` is authoritative and §4.1 mirrors it. Conventions live in `AGENTS.md`; this plan links to it
rather than repeating it (M-1).

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

**What changed in v2.1.** The v2 review found 59 issues and refuted none; M0 and most of M1 were built meanwhile,
and several had already been settled in code. v2.1 records those decisions and fixes the rest:

- **Contracts written down** (IM-6, AR-5, IM-7, AR-9): `games.Info`, the host's injected resolver, the host
  chaining `Result.Next`, the greeting scene, the `Think` cancellation rules, per-launch seeds (AR-10), layout
  geometry with the front panel (AR-11), and the M6 host hooks for movie mode (AR-7).
- **The import table matches `archtest`** (AR-1, AR-4, IM-4): test-only imports, subtree rules, the root package that
  embeds the licences, and the edges M0 needed.
- **Tooling and CI as built** (TO-1, CR-1, CR-6, SL-2): the lint fixture under `testdata/`, three tool modules,
  a release dry run that uses `--snapshot`, explicit job conditions, a weekly report that tells findings from
  errors, and a gitleaks job that can actually fail.
- **Film text** (RF-3, RF-9, SL-5): the climax keeps the film's `LIST GAMES` beat, the montage names carry honest
  provenance, the call-back is two scenes, and the status burst is ours.
- **Appendix A is deleted.** Appendix C's mockups are redrawn at exactly 80×24.

---

## 1. Requirements

Source codes: **U** = stated by the owner (the answers are recorded in `docs/reviews/PLAN-v1-review.md`, "Owner
decisions"). **D** = derived from a U requirement. **P** = a plan default the owner may override.

| # | Requirement | Src | Acceptance |
|---|---|---|---|
| R1 | Go terminal-UI recreation of WOPR from *WarGames* (1983); binary `wopr`. | U | — |
| R2 | Every game on the film's `LIST GAMES` (15 entries), plus the tic-tac-toe ending. | U | Each game meets the definition of done (§6). |
| R3 | Persona faithful to the film and to the brother's "WOPR Simulator" prompt: WOPR speaks ALL CAPS, addresses the user as PROFESSOR, is terse, numbers its lists (except the film's unnumbered `LIST GAMES`, §4.6), refuses invalid moves. User input is echoed as typed, in mixed case, as in the film (F-1). | U | Persona table tests and golden transcripts (§9). |
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
| Linux | Ubuntu 22.04 and newer, and any distribution with an xterm-class terminal | amd64, arm64 | `ubuntu-24.04`, `ubuntu-24.04-arm` |
| macOS | 26 Tahoe | amd64, arm64 | `macos-26-intel`, `macos-26` |
| Windows | 11 | amd64, arm64 | `windows-2025`, `windows-11-arm` |

Binaries are static (`CGO_ENABLED=0`). Go 1.27's own floor is macOS 13, so builds run fine on 26 (D-5). A
static Linux binary needs nothing from the distribution, only a kernel: Go's floor is Linux 3.2, and 22.04
ships 5.15. The `ubuntu-22.04` runner image is deprecated and unsupported from 2027-04-17, and an
`ubuntu:22.04` container would still run on the host's kernel, so CI tests neither; the 22.04 claim rests on
that floor (B-5).

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
`abs0/wargames`'s transcription, and both are credited in `NOTICE.md` (owner decision 2026-10-06, L-4):

- the backdoor header block;
- the montage scenario names.

abs0's BSD-2 licence covers the transcription; the text is still the film's, so it is excluded from the MIT grant
like other film text (SL-4). The status burst after the header is **not** an abs0 item: it is the film's phrases
in our own layout, tagged `reconstructed` (SL-5).

**Provenance tags.** Every script line, scene step and asset carries one. The tag is one of the values below,
and a `third-party:` tag must name a source credited in `NOTICE.md`; a test enforces both for `lines.go` now and
for `internal/assets/` and `internal/movie/scenes/` (SL-4). The user's typed lines in movie mode are
`script.User` lines, which keep their mixed case. The types and the check live in
`internal/script` (`script.L`, `script.Validate`), shared by the persona, every game, the assets and the movie
scenes.

| Tag | Meaning |
|---|---|
| `film` | Confirmed against the film in the M5 viewing pass |
| `reconstructed` | From transcripts or subtitles; not yet confirmed |
| `third-party:abs0/wargames@010ed92:wargames.sh` | Film text as transcribed by abs0; credited, BSD-2 notice in `NOTICE.md` |
| `third-party:https://asciiart.website/art/3719` | Matthew Thomas's ASCII world map, GTW's big board (owner's choice, 2026-10-06). Kept exactly as drawn, so the capitals rule does not apply to it; `NOTICE.md` links the page and carries his line, "Map (C) 1998 Matthew Thomas. Freely usable if this line is included.", which a test checks |
| `original` | Written for this project |
| `prompt` | Adapted from the brother's prompt. **Not committed to this public repository at all** until he grants written permission **and** a licence (MIT or CC0); a build tag would not help, because the module zip would still carry it (L-3, L-5, SL-4). |

### 2.2 Facts that shape the design (F-1)

- David's terminal was a 17-inch **Electrohome black-and-white** monitor: white text on black. The default
  `imsai` theme is white phosphor, and green and amber are alternates.
- WOPR's text was generated off-screen by CompuPro S-100 systems on a **24-line × 80-column** display. The map
  glyphs came from 34 custom line-segment characters. This supports the 80×24 target, and it is why the GTW map
  is drawn as **original line-segment-style ASCII art**.
- **WOPR's output is upper case. David's typing appears in mixed case** ("Hello.", "Love to. How about Global
  Thermonuclear War?", "Las Vegas"). One exception is recorded until the M5 viewing pass: at the NORAD console
  in the climax, abs0's transcription has his entries in capitals (`CHESS`, `GTW`, `TIC-TAC-TOE`, `ZERO`), and
  movie mode types them so, tagged `reconstructed` (§7).
- **LOGON comes first.** The `#45 11456 …` header and status burst appear only *after* `Joshua` is accepted.
  An unrecognised ID prints two lines and drops the connection.
- The GTW-vs-chess exchange is a separate scene after the greeting (§2.3).

### 2.3 Canonical screen text

The full text, with a provenance tag per line, is in **Appendix B**. In M1 it moves into `internal/wopr/lines.go`
and `internal/assets/`, which become authoritative. Two fan-source conflicts are recorded there and settled in
the M5 viewing pass:

- `USER ACCOUNT` vs `USER ACCOUNT NUMBER`;
- `CITY AND/OR COUNTY NAME` vs `COUNTRY`;
- `61 HOURS` (abs0 only) vs `28 HOURS` (the subtitles, elfuska, and the film's own timeline); 28 is preferred
  (RF-9).

### 2.4 Visual language

- **Console**: full-bleed text with no chrome. WOPR lines are double-spaced and revealed at modem speed, with a
  solid block cursor.
- **NORAD big board**: near-black blue field, cyan/blue outlines, red incoming and yellow outgoing tracks, white
  labels, a DEFCON column of boxed numerals 5…1, and `** … **` notices. This is the `norad` theme.
- **WOPR cabinet**: rows of lights and a `W.O.P.R.` nameplate. This becomes an optional one-row front panel,
  which also shows "thinking" (§4.3).
- **Verbiage**: ALL CAPS, terse, declarative; never asks clarifying questions; numbered lists `1.  2.  3.`
  (except `LIST GAMES`, which is unnumbered as in the film);
  refuses invalid actions with `** IMPROPER REQUEST **`-style lines; calls the user PROFESSOR.
- **Anti-pattern**: bordered panes and labelled input boxes. Borders appear only inside the big board and in
  NORAD notices, and as a game's own picture (a board's edge, a card's face, a maze's frame).
- **ASCII art** (decision 26): every game opens with an original title piece and draws its world in printable
  ASCII capitals. Card games draw card faces and trick tables; chess, checkers and the maze are framed;
  tic-tac-toe uses large film-style marks; the sims draw their front as a strip; GTW's side choice shows the
  two nations' outlines, generated from Natural Earth (public domain). Every piece fits its layout at 80×24
  and is tagged `original`, except the big board's world map, which is Matthew Thomas's (credited above).

The target screens are in **Appendix C**, all at exactly 80×24: the greeting, the GTW side choice, the GTW big
board, and the chess panel. **As built**, the GTW and chess screens are superseded by the regenerated goldens
(`internal/ui/testdata/gtw_screens.golden`, `internal/games/chess/testdata/view.golden`), which show the real
art (L-4).

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
| 9 | Lint | **prek** with **`prek.toml`** (authoritative in the repository since M0). | U |
| 10 | Size | Target ≤ 10 MB, hard limit 15 MB. | U |
| 11 | CI topology | Build once with GoReleaser; run each binary natively on its OS/arch (B-2). | U |
| 12 | Docs | README from M0. **AGENTS.md from M0** is the single source for commands and conventions. `CLAUDE.md` contains `@AGENTS.md` (M-1). | P (review M-1) |
| 13 | Releases | v0.1.0 after M2, then a minor release per milestone, then v1.0.0 after M5 (P-2). | U (v0.1.0 after M2); P (the rest) |
| 14 | Bridge | Minimal: WOPR bids all four seats by point count. The user always plays the declaring side, as declarer plus dummy, with seats rotated when East/West win the contract. Passed-out deals are redealt (G-3). | U (minimal); P (seat rule, G-3) |
| 15 | Chess rules | `github.com/corentings/chess/v2` v2.6.0, the maintained MIT fork of the archived `notnil/chess`. Search lives in `games/ai` (G-1). | P |
| 16 | Branching | **`main` is PR-only**. One required check, `ci-ok`. Squash merges only. CI also runs on every branch push (§11.2). | U |
| 17 | Dependabot | **Not used.** Manual update cadence plus a weekly staleness and vulnerability report (§11.5). | U |
| 18 | Montage names | Keep the list and credit abs0 under BSD-2 now. Verify in the M5 viewing pass. | U |
| 19 | Movie mode | **M6**: `-m/--movie` replays the film's WOPR terminal scenes (§7). | U |
| 20 | Tool modules | Three: `tools/` (gitleaks, govulncheck), `tools/lint/` (golangci-lint), `tools/release/` (GoReleaser). golangci-lint's dependencies break gitleaks's build when they share a module (SL-2). | P |
| 21 | `LIST GAMES` | Printed unnumbered, as in the film, at LOGON and in the Shell. A number typed right after it still selects; `HELP` and the README say so. `--games` stays numbered (RF-6). | U (confirmed 2026-10-06) |
| 22 | Leaving the war | Esc twice may abandon GTW and the climax tic-tac-toe, like any game, with WOPR's remark about the abandoned war; the ending itself cannot be aborted (§6.2). | U |
| 23 | `-m N` | Plays from scene N to the end of the list, then exits 0 (§7). | U |
| 24 | GTW exchange | **Turn-based DEFCON in M5** (§6.2), replacing the one animated strike built in M2. | U |
| 25 | History and legal text | The branch keeps its v1 history (the NOTICE credit covers the early-draft fragments). LICENSE holder: `GhostofGoes`. The Code of Conduct's contact: "contact @GhostofGoes privately via GitHub profile". Lines derived from the brother's prompt stay out of the repository until his written licence (L-3). | U |
| 26 | ASCII art | Original art in every game, in the film's spirit (owner request 2026-10-06): printable ASCII capitals, every line ≤ 80 columns, every screen within its layout, each piece tagged `original`. Galleries and other WOPR projects are style references only; GTW's side-choice outlines are generated from Natural Earth (public domain, credited in NOTICE.md). The one exception is the big board's world map, Matthew Thomas's (owner's choice, used under his terms and credited in NOTICE.md; its northern 13 rows are shown, in its own equirectangular projection, with cities placed from their latitude and longitude, each in a cell of its own: a city whose cell is open water goes to the nearest land, and the few that share a cell or fall a column off the art's coast are moved one cell, each with its reason in `internal/assets/gtwmap.go`; the other targets a list may name are placed the same way and may share a city's cell; the assets tests pin every place's cell). | U |

---

## 4. Architecture

### 4.1 Packages and the import DAG

Module `github.com/GhostofGoes/WOPR`, binary `wopr`. Import edges not listed below are forbidden (A-6, A-13).
The table in `internal/archtest` is authoritative and this one mirrors it; a change goes into both in the same
PR (AR-1, IM-4).

- `pkg/...` means the package and every package below it. The first matching row applies, so specific rows
  come before the subtrees that contain them.
- The standard library is allowed everywhere except for three fenced packages: `net/http` (only `llm`),
  `os/exec` (only `tools/...`, `archtest` and `e2e`) and `unsafe` (nowhere).
- **Test-only imports.** Any package's tests may also import `golden`, `games/catalog`, `games/gamestest`,
  `games/testkit` and `proto/host` (to configure testkit's runner). No non-test file may import `gamestest` or
  `testkit` (G-5).

```text
. (legal.go)         → stdlib                         embeds LICENSE, NOTICE.md, THIRD_PARTY_NOTICES.txt (SL-3)
cmd/wopr             → ., cli, ui, version, debuglog, games/catalog, (llm in M7)
internal/debuglog    → stdlib                         the opt-in debug log (§8)
internal/version     → stdlib
internal/cli         → games, theme (--theme validation), version, (movie in M6, for the scene index)
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
internal/e2e         → github.com/charmbracelet/{x/xpty,x/vt}   build tag e2e (Q-1)
internal/tools/manpage → cli, games, games/catalog, movie, theme   the manual page, from the program itself (§13)
internal/tools/...   → stdlib                         sizegate, stage, notices, relnotes, pkgdocs, cooldown (Go programs, not shell)
internal/llm         → proto, wopr                    plus net/http (M7)
```

- **Only `ui` imports Bubble Tea.** Only `ui` and `theme` import `charm.land/*`. `main` stays free of Bubble
  Tea: `ui.Run` owns every step that touches it (§4.2, IM-9).
- **`games` holds types only.** `games/catalog` imports every game package and returns the ordered `Registry`
  of `Entry{Info, New func() Game}` (§4.4). That removes the `games → games/<game>` import cycle (A-13).
- **No game imports `ending`.** Games hand off to the climax by slug through `Result.Next`; `ending` reuses
  tic-tac-toe's engine and drawing instead of duplicating them (AR-6).
- **Injection.** `cmd/wopr` passes the registry to `cli` and `ui`; `ui` builds the persona and the host's
  resolver from it. Tests use fakes from `gamestest`.
- **`internal/archtest`** reads each package's `Imports`, `TestImports` and `XTestImports` from
  `go list -tags=e2e -json ./...` (the tag brings in `internal/e2e`) and checks every edge against its table.
  Every `Playable` registry entry having a constructor is checked by `games.NewRegistry` itself. archtest also
  checks that the running Go is at least `go.mod`'s `toolchain` line, and exactly that line in CI, where a
  mismatch means `setup-go` fell back (D-6, AR-2). A newer local Go is fine.
- **The M6 consistency test** (§7) lives in `internal/movie`'s tests. archtest's per-row test-only allowance
  (`testAllow`) lets those tests, and only them, import `wopr` (AR-1).
- **Editor feedback**: `depguard` in `.golangci.yml` reports Bubble Tea imports outside `ui`. `archtest` is
  authoritative.

```text
legal.go                     package wopr (module root): go:embed of the licence files for --licenses
cmd/wopr/main.go             flags → dispatch → exit code; WOPR_PANEL; maps ui.Outcome to an exit code (§4.2)
internal/version/            Version/Commit/Date via -X; fallback to debug.ReadBuildInfo(); "unknown" if absent (D-4)
internal/cli/                Parse(args, reg, getenv) (Config, Action, error); usage; --games (§5)
internal/proto/              program.go output.go canvas.go key.go rand.go
internal/proto/host/         runner.go (program stack, input mode, Esc machine, Think jobs, Animate, pacing)
internal/prompt/             normalisation, clauses, numbers, menu choice, yes/no, negation
internal/theme/              palettes: Style → (truecolor, ANSI-16 index, no-colour attribute) (U-7)
internal/ui/                 app.go (model, Run, effects) clock.go (the single tea.Tick) keys.go render.go
                             drive_test.go: synchronous driver with a fake Scheduler (A-14)
  console/                   sanitize, wrap, scrollback, typewriter, line editor
internal/wopr/               persona.go (session, dial, LOGON, greeting, shell) intent.go brain.go scripted.go lines.go
internal/games/              registry.go (Info, Status, Game, Entry, Registry, Resolve)
  ai/ cards/ board/          search (depth/node limits, time cap), decks and tricks, board drawing and cursor
  ending/                    the climax: tic-tac-toe self-play → montage → final dialogue (A-16)
  catalog/ gamestest/ testkit/
  falkensmaze/ blackjack/ ginrummy/ hearts/ bridge/ checkers/ chess/ poker/ gtw/ tictactoe/
  fightercombat/ guerrilla/ desertwarfare/ airtoground/ theaterwide/ biotoxic/
internal/sim/                M4 engine
internal/assets/             gtw_map.go (original), scenarios.go (third-party:abs0), banner.go (original)
internal/movie/              director.go scenes/*.go (M6)
internal/golden/ internal/archtest/ internal/e2e/ internal/tools/{sizegate,stage,notices,relnotes,manpage,pkgdocs,cooldown}/
packaging/                   description.txt (both Linux packages'), debian/copyright (generated by notices, §8), rpmlintrc
tools/go.mod                 Go tools: gitleaks, govulncheck
tools/lint/go.mod            Go tool: golangci-lint (separate: its dependencies break gitleaks's build, SL-2)
tools/release/go.mod         Go tools: goreleaser (T-11), changie (release notes, §11.3)
tools/docs/go.mod            Go tool: Hugo, standard edition (the docs site, §13)
site/                        the docs site (§13): Hugo config, pages, layouts, game data; no Go packages
  go.mod                     a Hugo module file, not Go code: pins the Hextra theme
```

### 4.2 Runtime flow and state ownership

1. **`main`** parses flags. Print-and-exit actions (`--version`, `--help`, `--games`, `--scenes`, `--licenses`) write
   plain, unstyled text in one buffered write. They ignore `SIGPIPE`, and they treat `EPIPE` (Unix) or
   `ERROR_NO_DATA`/`ERROR_BROKEN_PIPE` (Windows) as success (C-3, U-11).
2. **TTY check.** `ui.TerminalProblem` reports a stdout that is not a terminal (checked with
   `charmbracelet/x/term`, because `os.ModeCharDevice` also accepts `/dev/null`) or `TERM=dumb`. `main` prints
   that one line and exits 2. On Windows the line suggests Windows Terminal. A non-TTY stdin is fine, because
   Bubble Tea opens `/dev/tty` or `CONIN$` itself. Bubble Tea's "error opening TTY" maps to the same exit 2
   (U-4).
3. **NO_COLOR.** If `NO_COLOR` is set to any non-empty value, `ui.Run` passes
   `tea.WithColorProfile(colorprofile.Ascii)`. colorprofile alone parses it with `ParseBool` and ignores
   `NO_COLOR=yes` (U-7). Every Bubble Tea step lives in `ui`: `ui.Run` returns an `Outcome` (`Finished`,
   `Interrupted`, `Panicked`, `NoTerminal`, `Failed`), and `main` only maps it to an exit code (IM-9).
4. **Screen.** The TUI runs on the **alternate screen** and paints the theme background into every cell.
   It does not use OSC 11, which ignores `NO_COLOR` and resets the terminal to its default background on exit.
   On a normal exit (0 or 130), the normal screen gets one line: `--CONNECTION TERMINATED--` (U-1).
5. **Ownership** (A-5).
   - `proto/host.Runner` owns the **program stack**. The persona is the root program. `Launch` pushes the
     program that the injected resolver builds (§4.4); `Done` pops it and delivers `GameOver{Result}` to the
     program below. `Done` with `Result.Next` is a hand-off: the **host** replaces the program with the next one
     and tells no one (AR-5, IM-6).
   - The persona owns the **conversation phase**: `Dialing → Logon`, back to `Dialing` after a failed log-on,
     then `Greeting → Shell`. There is no game phase ("a game is active" is "the stack is deeper than one") and
     no ending phase: the persona gets no input while the climax runs, and the ending's `GameOver` (the only
     `NoVerdict` result) returns it to `Shell` with chess offered.
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
     | `LOGOFF`, `LOG OFF`, `EXIT`, `QUIT` (punctuation ignored: `Log off.`) | Exit 0 |
     | Empty line | `LOGON:` again |
     | Anything else, including a game name | `IDENTIFICATION NOT RECOGNIZED BY SYSTEM` / `--CONNECTION TERMINATED--`, then `Wait`, `Clear` and a short re-dial |

   - The failure counter lives in `Session`. After the third termination, an `original` hint line is printed
     before the next `LOGON:`.
7. **Backdoor**: the header block, the status burst, `Clear`, `GREETINGS PROFESSOR FALKEN.`, then the
   greeting scene (§4.6) and **Shell**.
8. **Shell dispatch.** Sanitised input goes, in order, to: host commands (`LOGOFF` family, at every depth of the
   stack) → **phase-gated commands** (`HELP`, `HELP GAMES`, `LIST GAMES`, `PLAY <game>`) → **numbered selection**
   (only on the line right after a list) → **explicit intent** (§4.6) → the **greeting scene** while it is active → **offer acceptance** (§4.6) → **Brain** (asynchronous,
   via `Think`). A command or an explicit request therefore ends the greeting early; any other non-empty line
   advances it (IM-7, RF-11). An empty line only re-prompts: it keeps a pending offer and an armed list.
   `PLAY <name>` that matches no game answers `NO SUCH GAME IN MEMORY`; one that matches several lists them.
9. **`--play <game>`** pushes the game straight onto a persona in `Shell`, with the greeting marked as done.
   `--movie` runs the movie director as the root program instead of the persona (§7).
10. **Exit codes.**
    - `LOGOFF`, `Ctrl+D` on an empty input line, and SIGTERM exit 0.
    - `Ctrl+C` and SIGINT exit 130.
    - `Ctrl+Z` suspends on Unix (`tea.Suspend`) (U-10).

### 4.3 Console, clock, cursor, input

- **One clock** (A-9). `ui/clock.go` holds the only `tea.Tick`. While anything is busy (the typewriter and its
  pauses, a program's `Animate`, a pending `Think`, the Esc window) it keeps one tick in flight at about 30 fps.
  Each tick carries a generation, so a stale tick is dropped; with nothing busy, no timer runs.
  - A late tick delivers one coalesced `dt`, which the host splits into at most one `TickEvent`.
  - `Blink` cells and the front-panel lights move only while the clock runs. A program that wants a blink while
    idle asks for `Animate`.
  - The UI's golden driver (`ui/drive_test.go`) substitutes a fake `Scheduler`, so it never calls `tea.Tick`
    (A-14).
- **Pacing.** A dt-based typewriter: about 30 chars/s for speech, fast for tables, instant for boards.
  `--instant` / `WOPR_INSTANT=1` and "skip" both mean "advance by infinity". Movie mode adds a user-typing pace
  (§7).
- **Cursor.** `tea.NewCursor` gives a blinking block. It is parked at the end of the input line, or after the
  last revealed text. It is steady while WOPR thinks and under `--reduce-motion`, and hidden in key mode.
- **Skip policy** (U-3).
  - While output is revealing or paused (`Wait` is skippable), the first key flushes the queue.
  - In line mode, a printable key is *also* typed into the input line (typeahead kept). Enter and Space act
    purely as "skip" only when the input line is empty.
  - In key mode, the key flushes *and* is delivered to the game.
  - PgUp/PgDn always scroll. Esc only skips. Ctrl+C always quits.
  - The README says "any key skips; what you type is kept".
- **Thinking.** While a `Think` or a Brain reply is pending:
  - the cursor stops blinking, and a `PROCESSING` row with one to three dots cycling at 2 Hz sits directly above
    the input line (it is the last row when nothing has been typed);
  - the front panel lights animate when the panel is shown;
  - keys are **buffered** into the line editor, shown on the input row, and Enter is **held**: the line is
    submitted as soon as the prompt is active again, whatever made it active (a tick, a skip, or an instant
    flush), unless a different program now runs: a line held for a game that ended is dropped with its
    prompt. The input line wraps onto further rows rather than running past the edge.
- **Layout changes follow the text.** When a game with a board ends, its last View stays on screen until its
  final output has been revealed (a typewriter mark), then the layout changes as WOPR's verdict starts.
  Chess and checkers also hand the final position to the persona in `Result.Lines`, like tic-tac-toe, so it
  survives `--instant` and stays in the scrollback.
- **A game with its own screen starts on a new page.** When the host launches a program whose layout is
  Panel or Full (from the persona's menu or by a hand-off such as GTW → tic-tac-toe → the ending), it starts
  a new console page, so the strip under the board shows only that game's text, not the last game's lines
  and the menu that chose this one. The earlier pages stay in the scrollback. A game ending does not start
  a page: its last lines stay above WOPR's verdict and `SHALL WE PLAY ANOTHER GAME?`. Console games carry on
  the conversation's page.
- **Esc machine** (host-owned; games never see Esc).
  - During a reveal, Esc skips.
  - In a game, the first Esc shows `** PRESS ESC AGAIN TO END GAME **` for 3 s. A pending `Think` keeps
    running. A second Esc within 3 s cancels the `Think` context (its late result is dropped by generation)
    and pops the game with `Done{Aborted}`. Any other key disarms it.
  - While a Brain reply is pending, Esc cancels it: the persona receives `ThinkDone{Err: ErrCanceled}` and
    prints `** REQUEST CANCELLED **` (`original`).
  - At the Shell, Esc does nothing. The ending (self-play, montage, final dialogue) cannot be aborted with Esc
    (the resolver marks internal entries `Placement.NoAbort`; it survives `SetLayout`); Ctrl+C still quits. GTW and the climax tic-tac-toe are games, so Esc-Esc ends them like any other (§6.2).
- **Sanitising** (U-2). Two functions in `ui/console`, both tested. Whitespace controls are mapped **before**
  the other controls are dropped, so pasted lines are never glued together (RF-4).
  - `SanitizeInput` serves typed and pasted text:
    1. replace invalid UTF-8 with U+FFFD;
    2. `ansi.Strip`;
    3. map CR LF, CR, LF and TAB to one space each (a multi-line paste becomes one line);
    4. drop the remaining C0, C1 and DEL;
    5. cap at 256 grapheme clusters.
  - `SanitizeText` serves any text from outside the binary (Brain, LLM): invalid UTF-8 replaced, `ansi.Strip`,
    CR LF and CR normalised to LF, split on LF into logical lines, tabs expanded to 8-column stops, then the
    remaining controls dropped. `ui` applies it to every Brain-originated line.
  - Fuzz properties: no control rune survives, the result is valid UTF-8 and at most 256 clusters, and every
    wrapped row fits its width.
- **Width** (U-6). The line editor and the wrapper measure with the same method as the renderer: `wcwidth` by
  default, and grapheme widths once the terminal confirms mode 2027.
- **Scrollback.** Logical lines are wrapped at render time and cached per width, capped at 2000. New output
  auto-follows. `Clear` starts a new page, and history stays above it. A short page starts at the top of
  the screen; scrolled up, its blank rows move with the text, so the lines just above the page come into
  view first. `Render(w, h)` emits exactly `h` rows.
- **Layout geometry** (AR-11, IM-8). Let `H'` be the terminal height minus one when the front panel is shown.
  - The layout width is `min(cols, 80)`, centred; the theme background is painted across the full width.
  - **Console**: `H'` rows. The input row follows the last line of text (with `PROCESSING` or a host notice
    just above it), so it is the bottom row once the console has filled.
  - **Panel**: the game's view takes `min(Info.PanelRows, H' − 8)` rows on top (`PanelRows` ≤ 12); the console
    strip, input row included, takes the rest: 12 rows at 80×24, 11 with the front panel.
  - **Full**: the view takes `H' − 4` rows, then a 3-row strip and the input row.
  - The **front panel** is exactly one row at the bottom: `W.O.P.R.   LINE 1200 BAUD   ONLINE` on the left and
    eight lights on the right, which chase while WOPR thinks. It is shown by default in `norad`;
    `WOPR_PANEL` set to a true value shows it in any theme, and a false value hides it even in `norad`.
  - Programs draw into the area that `Env` and `ResizeEvent` report, so they never need these formulas. The
    catalog test caps `PanelRows` at 12, and the UI goldens render each layout at 80×24 with and without the
    panel.
- **Too small** (U-8). Below 80×24 the `TERMINAL TOO SMALL` card is shown, and:
  - the typewriter, `Animate`, `Wait` and Blink consumers **pause**;
  - a finishing `Think` is held until the size is valid again;
  - every key except Ctrl+C is dropped.
  No program starts before the first valid size. `ResizeEvent` fires only when a program's layout area
  actually changes.
- **Mouse.** Mouse reporting stays off, so native text selection works. On the alternate screen some terminals
  (VTE) turn the wheel into ↑/↓, which cycles input history (U-9). The M5 accessibility pass kept this:
  mouse reporting would take native selection and copying away, and resetting the terminal's alternate-scroll
  mode (DECRST 1007) would outlast wopr, since Bubble Tea restores only the modes it sets. The README's
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
type AwaitKeys struct{ Hint string; Capture bool }       // key mode (maze, board cursor); Capture: M6, below
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
type Quit   struct{}                                  // LOGOFF: exit 0
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
	Blurb     string   // one line for --games and HELP
}
type Entry struct {
	Info Info
	New  func() Game // nil while Planned; Env arrives in Start
}

// internal/proto/host: the resolver ui injects (AR-5).
type Placement struct{ Layout proto.Layout; PanelRows int }
type Resolver func(proto.Launch) (proto.Program, Placement, error)
```

Rules:

- **Movie-mode hooks** (M6, AR-7, RF-1, IM-11; the `gamestest` stub has a case for each but `Hold`, which a
  game cannot send). They are additive: no other program uses them, and the goldens did not move when they
  landed.
  - **Key capture.** A *root* program sends `AwaitKeys{Capture: true}` and then receives every key as a
    `KeyEvent`, Esc as `KeyEsc`, with none of the host's side effects: no key skips output, and the Esc
    machine is idle. A `Prompt` ends the capture. The host ignores `Capture` from a launched program, so Esc
    stays the host's in every game. While a capturing root's launched program runs, keys other than Ctrl+C
    (and PgUp/PgDn) do nothing, and each brings up `** THE GAME PLAYS TO THE END. CTRL+C QUITS. **` for the
    Esc window (`Runner.Refused`), so the key is not met with silence.
  - **`Hold{On}`** freezes the typewriter, `Wait`, `Animate`, Blink and the front panel, reusing the TOO
    SMALL pause. Keys still arrive, but none reveals held output, and neither does `--instant`. Only a root
    that captures keys can hold, since only it is sure to get the key that releases it; the host ignores
    `Hold` from anything else. Only a change is reported, and a hold ends when its program launches another
    or ends, so a launched program never starts frozen.
  - **`Drain`** queues a marker after the program's output; when the typewriter reaches it, the running
    program receives `Drained`. A newer `Drain` replaces a pending one, and a marker reached while another
    program runs is dropped. A marker reached while the program has a `Think` pending is answered after its
    `ThinkDone`, so the two arrive in the same order under `testkit` (which answers a marker after the rest
    of its batch, where the console would), `--instant` and pacing.
  - **`Skip`** reveals what is queued at once (next and previous scene), held output included.
  - **`Say.Open`** leaves the last line open, so the next `Say` continues it on the same row:
    `proto.Typed(prompt, text, rng)` uses it to type a user's line a few seeded keystrokes at a time after
    its prompt, followed by the blank line the console leaves after an answer. Any line added from outside
    (the echo of a submitted line) or any page break closes it, including one the UI applies at once when a
    game with its own screen launches.
- **Input mode is dynamic.** `Prompt` switches to line mode and `AwaitKeys` to key mode. A program toggles
  between them as it needs (checkers: type `b6-a5`, or move a cursor). While a `Prompt` is active, an empty
  Enter is delivered as `LineEvent{""}`; GTW ends its target list that way.
- **Launch and hand-off** (AR-5, IM-6).
  - The host resolves `Launch{Slug}` through the injected `Resolver`. `ui` builds it from the registry: a
    `Planned` or unknown slug prints `** GAME ROUTINE NOT AVAILABLE **` in character. The ending is a registry
    entry with `Info.Internal` set: `Get` (and so the resolver) finds it, while `Resolve`, `Exact`, `All` and
    `Listed` (so `--games`, intent and `LIST GAMES`) never see it.
  - `Env.Width`/`Height` is the area of the program's placement (its `Info.Layout` and `PanelRows`) at the
    current size; a `ResizeEvent` follows any `SetLayout` or resize that changes it.
  - `Done{Result{Next}}` is chained by the host: the program is replaced by the next one and the program below
    hears nothing. Only the last program's `GameOver` reaches the persona.
- **`Think` contract** (A-3, G-1, AR-8, AR-9).
  - The host runs `Fn` off the UI goroutine. A program has **at most one** `Think` in flight: a new one
    replaces the previous, whose result is dropped.
  - Cancellation: a cancel that leaves the program running (Esc on a pending Brain reply) is delivered as
    `ThinkDone{Err: ErrCanceled}`. A confirmed abort pops the program, so its result is never delivered.
  - Deadlines: when not deterministic, the host arms `Budget`, and a search returns its best result as a normal
    `Value` when the deadline passes. Slow I/O (the Brain, the M7 LLM) returns an error instead, so its fallback
    line fires.
  - With `Env.Deterministic` (`--seed` or `WOPR_SEED`, tests, movie mode), searches stop at their own `Limit`,
    which `Fn` captures; a non-zero `Think.Limit` tells the host the search is bounded, so it arms only a 60 s
    safety cap instead of `Budget`. `-race` slows a depth-3 chess search to a p99 of about 1 s, so a 1.5 s
    budget there would flake (AR-8). `testkit` fails the test if a deterministic `Think` reaches the cap.
  - `Fn` must capture values only. A `*chess.Position` lazily caches its moves and is not goroutine-safe.
  - A program's `Handle` itself runs on the event loop, where a loop that never ends would freeze wopr past
    Ctrl+C (a key in raw mode) and SIGINT or SIGTERM (messages queued behind it). The UI's watchdog
    (`internal/ui/watchdog.go`) times every Update and View: one running 10 s, or still running a second after
    SIGINT or SIGTERM, makes wopr restore the terminal, say it stopped responding, and exit 1.
- **Seeds and streams** (AR-10). The host derives each launched program's `Env.Seed` from the session seed,
  the slug and how many times that slug has been played (`NewRand(session, GameStream(slug, n)).Uint64()`), so
  chess, checkers and each replay draw different sequences while a seeded session still reproduces exactly. A
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

**Contrast** (RF-10). `TestTextContrast` checks every Style against its background: WCAG AA (4.5:1) in
truecolor, or 3:1 for the deliberately faint `Dim` and `Land`, and at least 3:1 under the default xterm and VGA
16-colour palettes. The theme tests range over `proto.Styles()`, so a Style added later is covered as soon as
it is declared. norad's Dim moved from ANSI blue (4), which is under 2.3:1 on black in both palettes, to
cyan (6). Two exceptions are accepted and documented: imsai's Dim is bright black (8), 2.8:1 on the Linux
console's VGA palette; and Solarized redefines bright black as its background colour, which no theme can
work around.

**Accessibility rules.**

- **Meaning never rests on colour alone**, nor on an attribute that cannot show: bold on `Bright`, `Alert` or
  `Accent`, which are bold in every theme, changes nothing.
  - Incoming tracks draw `*` and outgoing `+`; an impact is a reversed `X`. The forces and kill-ratio tables
    name each side's row or column.
  - The current DEFCON level is pointed at, `>| 3 |`, as well as reversed. The pointer is what says it: the
    monochrome themes reverse the DEFCON 1 rung at every level (their stand-in for white on red), and so
    does norad without colour.
  - Card suits always show their letter. The card winning a trick so far is edged `.===.`; it was bold, which
    showed only on the card's index, and not at all on a red card in the monochrome themes, whose red suits
    are bold already. Before each play prompt, Hearts and Bridge say the trick so far in the console
    (`WEST LEADS 7H. NORTH PLAYS KH.`), and each trick once it is complete.
  - Chess and checkers bracket the last move (`[]` and `P<`, `[b]`); a king is its capital letter alone (bold
    on Black's made it look like White's men without colour), and a jumped man is `x`. Tic-tac-toe
    underlines the last mark, and the console says every move.
  - Gin Rummy names the melds by position beside the hand (`MELDS 1-3 4-6`, or `NO MELDS`), besides
    drawing them bright. Bridge points at the hand to play (`>`); Hearts draws its pass as an arrow and
    names its direction; the maze marks the player `@`, the exit `[]`, unseen cells `.` and, once out,
    the trail `:`.
  - Alerts and notices are words, between `**` in the console.
- **Flashing** is capped at 3 Hz for any change covering a large area of the screen. That includes Blink
  toggles, DEFCON 1 and montage frames. This meets WCAG 2.3.1.
- **`-r/--reduce-motion`** (`WOPR_REDUCE_MOTION=1`) stops Blink (the cursor's too), freezes the front panel's
  lights and `PROCESSING`'s dots, and removes the acceleration of the ending's self-play (a steady 200 ms a
  move) and montage (a steady three lines a frame). It is not "no motion", and `--help` and the README say
  so: the typewriter, movie mode's typing, the big board's flights, the self-play and the montage still
  play, and `--instant` is what skips them (it draws each strike at once and goes straight past the
  self-play and the montage). Movie mode's game clock keeps ticking, as the film shows it; Space pauses it
  with the rest (WCAG 2.2.2).
- **Text is real characters.** Every screen and every view is printable ASCII; boards, maps and tables carry
  words (labels, legends, seat and side names), and the console says each of WOPR's moves (in Hearts and
  Bridge, the trick so far). Every prompt with text ends in `:` or `?`. The film's open prompts (WOPR's
  conversation, GTW's targets) have no text: WOPR's line before them usually asks, but not always (after
  `HELP` or `LIST GAMES` it is the list's last line). The cursor sits at the end of the input line; key mode
  shows a hint where the input line would be.
- **Art** is pure ASCII.
- **Goldens.** From M2, game goldens include `Canvas.StyleMap()`; the theme tests check each Style's rendering
  under the TrueColor, ANSI and ASCII profiles.
- **Sweeps** (`internal/ui/access_test.go`). The persona, the movie menu, the whole film and every playable
  game in the catalog are played through the real UI, each game a little way from a playbook of inputs
  (`TestEveryGameHasAPlaybook` fails for a game without one; GTW's goes on through the climax, tic-tac-toe
  with zero players and the ending). A playbook names what its screens must show at least once (checkers'
  king count and a jumped `x`, the trick tables' `.===.`, Gin's melds, GTW's DEFCON pointer and kill
  ratios), so it is known to reach them. `TestEveryScreenIsAccessible` checks each screen they reach (the
  film's every half second) for the rules above (ASCII, prompts, cursor, key hint, no attribute that no theme
  shows) and renders it in every theme under the ASCII profile: no colour codes, and in the program's view
  the pairs of styles whose difference carries meaning (`theme.Distinct`, which the theme tests also check)
  still look different wherever both are drawn, each cell with its own attributes. It also compares the text
  with the coloured screen's, as a guard: nothing yet chooses characters by profile.
  `TestNothingFlashes` plays them again with pacing and fails if the clock's ticks change a tenth of the
  program's view or more four times within a second (input-driven changes and the console's scrolling are
  not counted). `TestReduceMotion` watches movie mode's climax and a slow chess search tick by tick, with
  and without the flag.

**The M5 accessibility pass** (2026-10-07) audited everything built through v0.2.0 against these rules.

- *Checked:* every game's screens, the persona, the GTW board and kill ratios, the ending, movie mode (the
  board, the game clock, typed lines, the menu) and the front panel; each under `NO_COLOR` and the ASCII
  profile in all four themes, paced and `--instant`, with and without `--reduce-motion`; the contrast table.
- *Fixed:* tic-tac-toe's last X was marked only by bold on a style that is bold in every theme, so nothing
  showed; the last mark is now underlined. The trick table's winning card was marked by bold, which showed
  only on its index (not at all on a red card in the monochrome themes); it is now edged `.===.`. The current
  DEFCON level rested on reversal, which the DEFCON 1 rung always has in the monochrome themes; it now has
  a `>` pointer. Gin Rummy's melds were told from the deadwood only by brightness; their positions are now
  written beside the hand. The ending's self-play still sped up under `--reduce-motion`; it now keeps a
  steady step. Bold that never showed (chess's `CHECK`, GTW's last-orders warning and launch code, the
  ending's code, the maze's player, exit and title, checkers' white kings) was removed, and the sweep keeps it
  out. Gin's discard lost its bold too: its frame was bold already, so only a black card's index (any card's
  in norad) looks different. The theme tests now cover every Style, a Style that paints a background keeps
  a reversed block without colour, the meaningful pairs stay apart in 16 colours and without colour, and
  every Style renders under each profile with only that profile's colours. The README gained an
  accessibility section and the mouse-wheel note. After review: Hearts and Bridge say the trick so far before
  each play prompt; Black's checkers kings lost their bold, which the sweep's no-colour check found making
  them look like White's men; every game must have a playbook, and checkers' now plays to a king; `--help`
  and the README say what `-r` does and does not do, and point to `-i`.
- *Left for a human with a screen reader* (Orca, NVDA with Windows Terminal, VoiceOver): how the alternate
  screen reads while the typewriter reveals text and Bubble Tea redraws changed rows; whether the 2-D panels
  (trick tables, boards, the big board, the sims' maps) read in a useful order, or whether the console's
  words (each move, the trick so far) are enough on their own; whether the cursor, hidden in key mode (the
  maze) and while movie mode plays, should be parked at the key hint for magnifiers; and how the film's open
  prompts sound with no prompt text, above all after a list.

### 4.6 Persona: session, intent, offers, brain

```go
// internal/wopr/brain.go
type Snapshot struct { // a value copy taken in Handle; safe on another goroutine (A-4)
	Phase   Phase             // Dialing, Logon, Greeting, Shell
	Turn    uint64            // increments on every brain request, including cancelled ones
	Seed    uint64
	Said    map[string]bool   // cloned
	Flags   map[string]bool   // cloned
	Last    *proto.Result     // the previous game's result (copied); replaces v1's PostGame
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
	// M2 adds After (AfterWin, AfterLoss, AfterNoWinner, AfterAbort) for remarks about the last game.
}
```

- **Brain calls.** The persona calls the Brain through `Think`: `Fn` calls `Reply` with the snapshot. The reply
  comes back as `ThinkDone`, and the persona applies its `Effects` in `Handle`. Nothing else mutates the
  session. On an error, a timeout (scripted: never; LLM: 20 s) or a cancel, the persona answers with a scripted
  line.
- **The greeting scene** (IM-7, RF-11). After `GREETINGS PROFESSOR FALKEN.` the persona is in `Greeting`. Each
  non-empty line that is not a command or an explicit request advances one scripted WOPR line (`HOW ARE YOU FEELING
  TODAY?`, `EXCELLENT. ...`, `YES THEY DO. SHALL WE PLAY A GAME?`); the last one arms `Offer{Any}` and enters
  `Shell`. A command or an explicit request ends the scene early and is dispatched normally. Scripted rules do
  not run during the scene.
- **Explicit intent** (A-7, RF-7). The input is split into clauses on `. ! ? ; ,` and each clause is
  normalised:
  1. upper-case, with the typographic apostrophe U+2019 read as `'`;
  2. `[^A-Z0-9']+` collapses to one space;
  3. `LET'S`, `LETS` and `LET US` become `LETS`.

  A clause starts a game when all of these hold:
  - after optional fillers (`LATER`, `LOVE`, `TO`, `OK`, `OKAY`, `YES`, `SURE`, `WELL`, `FINE`, `THEN`, `SO`,
    `NOW`), it begins with `PLAY`, `LETS PLAY`, `HOW ABOUT`, `WHAT ABOUT`, `SHALL WE PLAY`, `CAN WE PLAY`,
    `I WANT TO PLAY` or `I'D LIKE TO PLAY`;
  - the rest of the clause, after an optional `A GAME OF`, `A ROUND OF`, `A NICE GAME OF`, `A GOOD GAME OF`,
    `SOME`, `A` or `THE`, and before optional trailing `NOW`, `INSTEAD`, `PLEASE`, `THEN`, `AGAIN`, `TODAY`,
    `WITH ME` or `WITH YOU`, is **exactly** a game name, slug or alias, a number, or a unique prefix. Anything
    else in the clause means the line is not a plain request ("Let's play 2 games of chess", "How about 1 more
    round?").
  - A negator (`NOT`, `DON'T`, `DONT`, `NO`, `NEVER`) before the verb vetoes the match. Because the verb must
    open the clause, this is a safety net rather than a rule that does work today.

  Input that is exactly a game name, slug or alias also counts.

  **Numbered selection** is armed by `LIST GAMES` (or by accepting `Offer{Any}`) and applies to the next line
  only.
- **`LIST GAMES`** prints the film's list, **unnumbered**, with the blank line before the last entry, at LOGON
  and in the Shell (RF-6, decision 21). This is the exception to R3's numbered lists. A number typed right after
  it selects that game, and `HELP` says so.
- **Offers** (A-7). Some WOPR lines arm an offer that lasts one turn:
  - `SHALL WE PLAY A GAME?` arms `Offer{Any}`. YES gets `WHICH GAME?` (`original`), the list, and arms selection.
  - `WOULDN'T YOU PREFER A GOOD GAME OF CHESS?` and `HOW ABOUT A NICE GAME OF CHESS?` arm `Offer{chess}`.

  `YES`, `Y`, `OK`, `SURE`, `LOVE TO` and `FINE` accept. `NO` declines in character. Explicit intent outranks
  an offer, so "Love to. How about Global Thermonuclear War?" asks for GTW.
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
- **Streams** (A-11, AR-10). `proto.NewRand(seed, id)` is the only sanctioned constructor; forbidigo bans the
  top-level `math/rand/v2` functions and every `math/rand` (v1) function, whose global source is randomly seeded
  (TO-5). The two seed halves are a program's `Env.Seed` and a `StreamID` with disjoint domains:

  | Domain | StreamID | Seed half |
  |---|---|---|
  | Brain | `1<<56 \| Turn` | the session seed (the persona's `Env.Seed`) |
  | Game | `2<<56 \| hash32(slug)<<16 \| playIndex` | the session seed; the host derives each launch's `Env.Seed` with it |
  | AI | `3<<56 \| thinkSeq` | the game's own `Env.Seed`, with the game counting its `Think`s |
  | Movie | `4<<56 \| scene` | a constant seed that the director pins (§7) |
  | UI | `5<<56` | the session seed |

  Without `--seed` or `WOPR_SEED`, the seed comes from `crypto/rand` and is written to the debug log. Go
  guarantees seeded `math/rand/v2` sequences across releases.

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
wopr -m | --movie [scene]     movie mode (§7): from that scene to the end of the list; without one, a scene menu
wopr -o | --only              with --movie: each scene plays alone, then exit (or back to the menu)
wopr -S | --scenes            list the movie's scenes (number, slug, what happens) and exit
wopr -t | --theme <name>      imsai | green | amber | norad          (env WOPR_THEME)
wopr -i | --instant           no pacing                              (env WOPR_INSTANT)
wopr -s | --seed <n>          deterministic run (also bounds AI search by depth/nodes)  (env WOPR_SEED)
wopr -r | --reduce-motion     no blink, still panel lights, no speed-ups (env WOPR_REDUCE_MOTION); not the
                              typing or the animations, which -i skips
wopr -L | --licenses          print NOTICE.md and third-party notices, then exit
```

- **Interspersed positional** (C-1).
  - `cli.Parse` calls `fs.Parse` in a loop. When parsing stops at a positional, the positional is recorded and
    parsing resumes on the rest.
  - A terminator is detected by where `fs.Parse` stopped (the token before `fs.Args()` is `--`), never by a
    consumed value: in `-t -- gtw`, `--` is the theme's value (IM-10).
  - With `--movie`, the positional is a scene. Otherwise it is a game.
  - More than one positional, or a positional together with `--play`, is a usage error (exit 2).
  - **Environment.** `WOPR_THEME`, `WOPR_INSTANT`, `WOPR_REDUCE_MOTION`, `WOPR_SEED` and `WOPR_PANEL` are read
    first, and a flag always wins. A boolean variable is false when empty or `0`, `false`, `no` or `off`, and
    true otherwise (`cli.Truthy`). `WOPR_SEED` takes what `--seed` takes: empty means unset (so `--seed=`
    drops it for one run), and a value that is not a non-negative integer exits 2 naming `WOPR_SEED`, as a bad
    `--seed` or `WOPR_THEME` does, even in movie mode, which ignores the seed. `NO_COLOR` is true when set to
    anything non-empty. `--movie` and `--only` have no variable: they choose what to run, not how.
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
    | `-o -m` | movie: the scene menu; each scene picked plays alone |
    | `--only`, `-o gtw` | exit 2: `--only` needs `--movie` |
    | `-s x`; `WOPR_SEED=x` | exit 2, naming `--seed` or `WOPR_SEED`; `-s 7` wins over `WOPR_SEED` |
    | `--version gtw` | print the version; actions win over a positional |

- **Accepted forms** (C-2). `-games`, `--games`, `-p chess`, `-p=chess`, `-i=false`. Rejected: `-pchess`,
  `-is 1`. A test pins both lists, and the help text shows the accepted ones.
- **Letters.** `-v` means version, deliberately. A future verbose flag gets another letter (not `-V`, which
  conventionally means version) (C-4). `-l` is reserved for `--llm` (M7). `--licenses` uses `-L`, and
  `--scenes` uses `-S` because `-s` is the seed; `--only` is `-o` (`--single` would have needed another letter
  for the same reason).
- **`games.Resolve`**: number (1–15) → slug → alias → exact normalised name → unique prefix.
  - Unlisted entries are never matched by number. Tic-tac-toe has `Listed: false`, which is explicit rather
    than relying on a zero `Number` (G-4). A registry test checks that 1..15 each appear exactly once.
  - Slugs are the full film names, lower case and hyphenated (`global-thermonuclear-war`,
    `theaterwide-tactical-warfare`); aliases are short forms (`gtw`, `ttt`). `wopr --games` prints them all.
  - Ambiguity (e.g. `theaterwide`, a prefix of two slugs) prints the candidates and exits 2.
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
- every script line provenance-tagged;
- a playbook in `internal/ui/access_test.go` that reaches its main screens, so the accessibility sweeps
  (§4.5) check them (`TestEveryGameHasAPlaybook` fails without one);
- a page, `site/data/games/<slug>.json`: how to play, the controls, at least three tips true of its AI, and
  two or three screenshots with captions (§13). `internal/tools/manpage`'s tests check it against the catalog.

| # | Game | Layout | Input | M | Notes and quality test |
|---|---|---|---|---|---|
| 1 | Falken's Maze | Panel | keys | 4 | Procedural maze, fog of war. WOPR learns a turn bias and re-routes walls. Property test after every re-route: the exit is reachable *from the player's current cell*, no wall lands on the player, revealed cells stay consistent or the change is announced (G-6). **As built:** 19×5 perfect maze (arrows or WASD, `Q` gives up); the player sees their cell and where its passages lead. Every 6 moves WOPR closes an unseen passage on the way to the exit (at a junction where the way on is the player's favourite turn, if it can) and opens another unseen wall that rejoins the halves, the longest way round; at most 12 a game, since always lengthening the way could stall a player forever. Walls only change between unseen cells, so the maze stays perfect and seen walls never change. The key hint shows where the input line would be. |
| 2 | Black Jack | Console | line | 3 | One deck; the dealer stands on soft 17 (`Rules.StandSoft17`) and peeks on an ace or ten; 3 to 2 for a natural (money in cents, so exact); double on any first two cards; one split a round (split aces take one card). $100 stake per launch, $1–$25 bets, `LEAVE` cashes out (QUIT and EXIT stay the host's). Test: the dealer's rules over 600 seeded shoes under S17 and H17; exact payouts. |
| 3 | Gin Rummy | Panel (hand) | line | 3 | To 100; knock at 10 or less (`knock 7h`), gin 25, undercut 25, repeated lay-offs, void at two stock cards. Melds by exhaustive search; WOPR takes the discard only into a meld, sheds deadwood, knocks at once. Test: arrangement and scoring tables; legality and card conservation over 60 seeded games. |
| 4 | Hearts | Panel (trick) | line | 3 | 4 seats, passing (left, right, across, hold), 2C lead, no points on the first trick, hearts broken, shoot-the-moon, to 100; heuristic AI on `cards/trick.go`. Test: legality over 200 seeded deals. |
| 5 | Bridge | Panel (trick) | line | 3 (last) | Minimal (decision 14): the side with more points declares (the table turns when East-West hold them); the opener bids one of the target strain and partner raises to the level the points earn; one deal a game, scored not vulnerable. Defence: second hand low, third hand high, ruff when void. Test: auction and declarer-seat rotation; legality. |
| 6 | Checkers | Panel | line | 2 | 8×8 English draughts: forced captures, multi-jumps, crowning ends the move, kings move both ways, draw after 40 moves each without progress (a capture or a man's move, crowning included). The player is Black and moves first, typing squares with or without separators (`c3-d4`, `c3d4`, `c3xe5xg7`, `c3e5g7`); the first hop of a multi-jump is enough when only one jump continues it, and the game names the continuations otherwise. The game says why it ended. Search depth 6 (deterministic), or iterative deepening to depth 10 within 1.5 s. Test: rules cases, a double-jump puzzle, legality over seeded self-play. A board cursor (key mode) is optional polish. |
| 7 | Chess | Panel | line | 2 | Rules, SAN and UCI from `corentings/chess/v2`; `games/ai` alpha-beta with quiescence (captures and promotions, most valuable victim first). The search gets a FEN string, because the library's positions cache moves and are not goroutine-safe. Interactive: iterative deepening, 1.5 s budget, depth cap 4. Deterministic: depth 3 + quiescence (median 26 ms, max 204 ms over a 30-move game). Repetition and the fifty-move rule are claimed at once, so the search sees the game's history: below the root, a repeated position scores as a draw. A mop-up term drives a bare king to the edge, and the kings centralise in the endgame. Input: coordinates first (`B1C3` is a knight move), then SAN with a capital piece letter winning (`BXC6` is a bishop); `e2-e4`, `e8=Q`, `e8q` and a bare `e7e8` (a queen) all work. Board: each square is two columns by one row, which terminal cells (about twice as tall as wide) show square (owner request 2026-10-07; checkers keeps three columns); the last move is marked `[]` where it left and `P<` where it landed. Test: mates in one and two; no illegal move in 20 seeded self-play games at depth 2; K+Q and K+R mate a shuffling player within 50 moves. |
| 8 | Poker | Console | line | 3 | 5-card draw heads-up, 100 chips a side; ante 1, fixed limit 5/10, a bet and three raises; bets capped by the other stack (no side pots). Betting heuristic + bluff probability. Test: hand ranking (the category counts over all 2,598,960 hands); pot accounting. |
| 9–14 | Military sims | Console/Panel | line | 4 | See §6.3. Biotoxic always ends `WINNER: NONE`. **As built** (all Console): Guerrilla (12 turns; hidden cells, ambush at double strength, recruiting with support; survival wins), Desert (10 turns; supply to the depot), Theaterwide Tactical (10 turns; the escalation ladder, the top rung ends it `WINNER: NONE`) and Biotoxic (8 turns; contamination spreads with the wind) are `sim` scenarios; Fighter Combat (15 turns; simultaneous manoeuvres, energy and aspect) and Air-to-Ground (6 sorties; packages against SAM sites, interceptors and a hidden mobile battery) are bespoke. Test per game: a seeded transcript golden, every AI order legal over seeded games, the provenance and 80-column checks; balance logged over seeds. |
| 15 | Global Thermonuclear War | Console → Full | line | 2 | Self-contained film set piece (P-1); §6.2. Side and targets in Console, then the big board, kill ratios and the climax in Full. Cannot be won. **As built in M5**, the exchange is turn-based: three strikes down the DEFCON ladder, then WOPR's DEFCON 1 attack. Test: the film scenario transcript, board views, GTW's list against the registry, the UI screens; no allocation wins (exhaustive), the order grammar, the ladder under every order, Instant without ticks, the board keeping every element at each prompt, dotted bombers, the strip keeping up with the ladder. |
| — | Tic-Tac-Toe | Panel | line | 2 | Perfect minimax, preferring the fastest win, chosen at random among optimal moves with the seed. Unlisted (`Listed: false`), resolvable by name. Squares 1 to 9, or A1 to C3. Test: exhaustive "never loses" as X and O over every optimal choice; both modes' transcripts. |

### 6.2 The film path: GTW, tic-tac-toe and the ending (A-16)

The climax has **one owner**: `games/ending`. GTW and tic-tac-toe hand off to it through `Result.Next` and add
no verdict of their own.

1. **GTW**, film scenario:
   1. `SetLayout(Console)`: the two nations' outlines, named beneath, printed as console text (`Say`),
      then the side choice (`1. UNITED STATES` / `2. SOVIET UNION` / `PLEASE CHOOSE ONE:`) (RF-3).
   2. `Clear`, then `AWAITING FIRST STRIKE COMMAND`, and the targets prompt. Targets are read until an empty
      line; a multi-line paste arrives as one line, so targets on one line are split on commas.
      - A target must be a place on the enemy's side that the board knows (`internal/assets/gtwmap.go`: the
        26 cities and 85 more targets), by its 1983 name or another in use (`KYIV`, `ST PETERSBURG`,
        `NIZHNY NOVGOROD`, `NYC`, `WASHINGTON, D.C.`), in any case. Anything else strikes nothing: it is
        refused (`SMALLVILLE IS NOT IN THE TARGET DATABASE.`, `MOSCOW IS NOT AN ENEMY TARGET.`) with the hint
        `TYPE LIST FOR THE ENEMY TARGETS ON FILE.` `LIST` (also `HELP`, `?`) prints them in four columns. A
        target listed twice counts once, so `WASHINGTON, D.C.` is one.
   3. `SetLayout(Full)`: the big board, and the exchange **by turn** (decision 24). **As built in M5**:
      - **The first strike needs no order.** The empty line that ends the target list opens the board at
        DEFCON 5 and launches strike 1 at once on WOPR's **war plan** (ICBM 25%, SLBM 25%, bombers 0%), as in
        the film. Your SLBMs fly at the listed targets, your ICBMs at the enemy's silo field.
      - **Strikes 2 and 3 ask for orders.** `STRIKE 2 OF 3 [50 50 100]:` takes percentages of the ICBMs in
        silos, SLBMs at sea and bombers on the ground still held: `50 25 100`, `40`, `ICBM 50, BOMBERS ALL`,
        `50 SUBS`, `BMB 100` (the forces table's heading), `ALL`, `HOLD` (also `STOP`, `CEASE FIRE`). Airborne
        bombers cannot be ordered, so `AIR` is refused.
        - Enter (or `FIRE`, `YES`) carries out the bracketed plan; strike 3's is `100 100 100`.
        - `AUTO` hands WOPR every remaining strike: `WOPR HAS LAUNCH AUTHORITY.`
        - `HELP` or `?` prints three lines.
        - Refusals ask the same strike again: `ORDER NOT RECOGNISED. TYPE HELP.`,
          `PERCENTAGES ARE WHOLE NUMBERS FROM 0 TO 100.` (also for a signed or decimal number, including a
          typographic minus or dash such as `–50`), a game's name
          `** ROUTINE MUST COMPLETE BEFORE RESET **`, and GTW `** GAME ROUTINE RUNNING **`.
        - With nothing left to launch, a strike runs by itself.
      - **Each strike prints exactly three strip lines**: your launch, WOPR's launch on warning with the new
        DEFCON, and the cost (`LOST ON THE GROUND: … WARHEADS ON CITIES: …`). DEFCON falls one rung at
        WOPR's launch, 5 → 4 → 3 → 2, whatever was ordered. The strip lines print at table pace, so the
        DEFCON line shows within about half a second of the rung moving.
      - **WOPR takes the last rung alone.** `FULL-SCALE ENEMY ATTACK. … DEFCON 1.` fires everything it has
        left and lands every airborne bomber. Then `STRIKE ASSESSMENT COMPLETE. PRESS ENTER FOR PROJECTED KILL
        RATIOS.`
      - **Doctrine** (`gtw/exchange.go`; integer arithmetic, no `Think`, no randomness).
        - You hold ICBM 1000, SLBM 600 and bombers 300; WOPR a quarter more (1250, 750, 375).
        - WOPR answers each launch on warning with a quarter more of each missile, never under its ladder
          (ICBM 125/250/375, SLBM 50/100/150). It keeps 300 SLBMs at sea until DEFCON 1 and puts its bombers
          up at the first strike.
        - ICBMs go first at the enemy's silos, enough to empty them; each destroys four in five of the ICBMs
          held there. The rest go at cities. SLBMs go at cities.
        - Grounded bombers lose a third whenever missiles land on their side, and all of them at DEFCON 1.
          A quarter of arriving bombers are shot down.
      - **It cannot be won.** Your cities take at least 1,144 warheads and the other side's at most 750,
        whatever you order (`TestNoAllocationWins`, exhaustive over ICBM orders at every percent). Your
        ICBM orders alone decide what lands on your cities: held ICBMs absorb WOPR's, fired ones turn WOPR's
        on your cities. Your SLBMs and bombers alone decide what lands on theirs.
      - **The kill ratios are derived** from the exchange.
        - BOMBERS and ICBM are the counts destroyed.
        - The other rows scale with damage `1000·hits/(hits+400)` per mille, times one seeded noise factor
          per row (97–103%) shared by both columns. So your civilian and human rows are never better than
          the other side's.
      - **The board** (80×19 with the front panel; the map box takes rows 0–14).
        - Rows 15–17: the `TRAJECTORY HEADING` table in two columns on the left (your latest two missiles;
          a strike that drew one track keeps an earlier one in the second column), and on the right, ending
          at column 79, a `FORCES` table: ICBMs in silos, SLBMs at sea, bombers grounded (`BMB`) and airborne
          (`AIR`), your side first. The nations' names say whose figures are whose; the colours only repeat
          it.
        - Row 18 shows one thing, by state: the orders hint at a strike prompt;
          `LAST ORDERS: AT DEFCON 1 WOPR FIRES EVERYTHING.` before strike 3; the launch code at the climax;
          otherwise the legend `OUTGOING +   INCOMING *   IMPACT X`.
        - Earlier strikes leave only their impacts. ICBMs fly polar arcs that rise with distance, WOPR's
          lower than yours so both show between the silo fields. SLBMs hop low from real patrol areas placed
          by latitude and longitude (Soviet boats in the western Atlantic and the eastern Pacific, American
          boats in the Norwegian Sea and the north-west Pacific), whichever is nearer the target. Bombers fly
          in, dotted, at DEFCON 1: yours on a low arc, WOPR's straight in under it, both sides' dots on the
          same columns so that where the routes meet one covers the other.
      - **The film path.** After the targets, Enter four times reaches the climax: strikes 2 and 3, the
        kill ratios, the climax. That is two more Enters than M2's single strike, and about 18 s of flight
        instead of 8. Under `Instant` each line returns everything up to the next prompt; nothing waits for
        a tick.
   4. **Climax** (RF-3). WOPR proceeds toward launch, and the input decides the NORAD notice, as in the film:

      | Input | Response |
      |---|---|
      | `LIST GAMES` | The film list (tic-tac-toe is not on it), in the console, where its 16 lines fit; the next input brings the board back |
      | The name of another game (`CHESS`, `POKER`, …) | `** IDENTIFICATION NOT RECOGNISED **`, `** ACCESS DENIED **` |
      | `GLOBAL THERMONUCLEAR WAR` | `** GAME ROUTINE RUNNING **` |
      | Anything else | In turn: `** IMPROPER REQUEST **`, `** ROUTINE MUST COMPLETE BEFORE RESET **`, `** ACCESS DENIED **` |

      Each third rejection adds an `original` hint naming tic-tac-toe.
   5. `TIC-TAC-TOE` (or any intent naming it: `TTT`, `LET'S PLAY TIC TAC TOE`) gives
      `Done{NoWinner, Next: Launch{tictactoe, "climax:N"}}`, N being the launch code characters cracked so far.
      The host chains the hand-off (§4.4). Requests are read without apostrophes or spaces, so
      `LET'S PLAY CHESS` and `BLACKJACK` name their games.
2. **Tic-tac-toe in `climax` mode** is a closed state machine (RF-2):
   - `ONE OR TWO PLAYERS? PLEASE LIST NUMBER OF PLAYERS:` accepts `1`, `0` or `ZERO`; `2` or anything else
     asks again.
   - `1` plays a game. A draw ends in `STALEMATE. WANT TO PLAY AGAIN?`; a WOPR win (the user blundered) ends in
     an `original` line followed by the same question.
   - `WANT TO PLAY AGAIN?` accepts `YES` (another game), `NO` (back to the number of players), and `0` or
     `ZERO`, which is where the film types it.
   - `0` or `ZERO`, at either prompt, gives `Done{NoWinner, Next: Launch{ending}}`.
   - In normal mode, `NUMBER OF PLAYERS: 0` does the same.
   - The launch code `CPE 1704 TKS` is cracked digit by digit on screen while this plays. `games/ending` owns
     that display from here on, since GTW is gone from the stack: tic-tac-toe passes GTW's count on
     (`Launch{ending, "code:N"}`), and the ending cracks the rest over its self-play, complete on the last round.
3. **The ending** runs with Animate, Prompt and `Wait`, and cannot be aborted with Esc:
   1. self-play at increasing speed, capped by the flash rule, using `games/tictactoe`'s engine (AR-6); with
      `--reduce-motion` a steady 200 ms a move, about as long in all;
   2. the scenario montage, each ending `WINNER: NONE`. It redraws at most 2.5 times a second (a frame every
      400 ms) and speeds up by adding lines to each frame, one to eight; with `--reduce-motion`
      (`Env.ReduceMotion`) it adds a steady three;
   3. `Clear`, `GREETINGS PROFESSOR FALKEN.`, `Prompt`;
   4. any non-empty input, e.g. `Hello.` (an Enter pressed to hurry the show is not an answer, and the UI
      never holds a bare Enter as typeahead);
   5. `A STRANGE GAME.` / `THE ONLY WINNING MOVE IS` / `NOT TO PLAY.`;
   6. `HOW ABOUT A NICE GAME OF CHESS?`;
   7. `Done{NoWinner, NoVerdict: true}`.

   The persona, back on top, returns to `Shell` with `Offer{chess}` armed.
4. **Leaving early.** GTW and the climax tic-tac-toe are games, so Esc twice ends them like any game (§4.3);
   that is a deliberate choice, not a loophole. The persona then says an `original` line acknowledging the
   abandoned war (an `AfterAbort` rule that knows the last game was GTW) instead of the plain verdict.
5. **Goldens.** `film_path` (§9) covers the 0-players path; tic-tac-toe's `climax` golden covers a WOPR win
   and `NO`; the persona's `TestAbandonedWar`, `TestAbandonedMidExchange` and `TestAbandonedClimax` cover Esc
   in GTW, at a strike prompt and in the climax, and
   `TestFilmPath` presses Esc twice during the ending, which ignores it.

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

**As built.** `sim.NewGame` turns a `Scenario` into a `proto.Program`; scenario hooks are `Setup`, `Victory`,
`AI` (default: `DefaultAI`), `AttackMod`, `Upkeep`, `Status` and `RegionNote`, and scenario verbs run in
phases 0 (before movement), 1 (with combat) or 2 (after). Units may be hidden (`~` on the map) until found.
Fighter Combat shoots on `sim.CRT` and ends with `sim.RatioTable`; WOPR's manoeuvre is a heuristic with a
little randomness. Air-to-Ground resolves escorts, interceptors, SEAD, SAM fire and the bomb run on `sim.CRT`;
WOPR places its mobile battery where the last raid went (usually) and keeps its interceptors down when the
last escort outnumbered them. GTW stays separate: moving it would not remove code.

#### 6.3.1 Engine spec (written first, M4)

- **Map.** A strip of 5 to 7 regions, numbered from the player's rear (1) to WOPR's (N). Each has a name, a
  terrain (`OPEN`, `ROUGH`, `CITY`; a defender in rough or city ground shifts the odds one or two columns in
  its favour) and a controller. Adjacent means one apart.
- **Units.** The scenario's table gives each type a name, attack, defence, move (regions a turn) and range (0:
  its own region, 1: next door too). A unit has two steps: a loss reduces it, a second destroys it; a
  reduced unit attacks and defends at half (rounded up). Units carry a kill-ratio category.
- **A turn.** The player orders each unit in turn (`<verb> [region]`, or `HOLD`); `STATUS` reprints the map,
  `HELP` the verbs. WOPR's orders come from the scenario's AI. Then movement, combat, one event card, upkeep
  (supply, scenario hooks) and the victory check. The map prints as a table each turn (Console layout),
  within 80 columns: a stack of one type prints as `COR1,2*,4`. A region is named by number, by its name,
  or by the start of any word in it (`ALAMEIN`). Drastic verbs (`ESCALATE`, `RELEASE`) must be typed in
  full. An order the other side's moves made impossible is reported (`ARM1 STANDS FAST: ...`).
- **Combat-results table.** Odds column = attack ÷ defence, clamped to 1:2 … 4:1, then shifted by terrain;
  a d6 from the game's stream picks the result: `NE` no effect, `AE` attacker loses a step, `EX` both do,
  `DR` the defender retreats a region (or loses a step if it cannot), `DE` the defender loses a step. HELP
  explains the codes. One table for every
  scenario and both bespoke games; a test pins its rows.
- **Events.** Each scenario has a deck of 5–12 cards with a headline and a hook; one is drawn each turn,
  and the deck reshuffles from the stream when empty. A card says only what happened (which unit was made
  good or lost a step), never an effect that found nothing to act on.
- **End.** A turn limit and a victory hook per scenario: control of objective regions, elimination, or (for
  Biotoxic) nobody, ever.
- **Kill ratios.** Losses are counted per category for both sides and printed at the end in GTW's two-column
  table format; the persona's verdict follows.
- **Determinism.** Every roll and card comes from `proto.NewRand(Env.Seed, …)`; a scenario test plays a
  seeded game to the end and checks that every order the AI gives is legal.

---

## 7. Movie mode (M6)

`wopr --movie [scene]` replays the film's **WOPR terminal scenes** as a self-running show, through the same
console, typewriter, canvas and game code as interactive play. **Built in M6.**

**Scenes.** `wopr --scenes` (`-S`) prints this list. The scripts are checked in the M5 viewing pass; until
then every film line in them is tagged `reconstructed`. David's lines are `Type` steps in mixed case, as on
screen (RF-9), with one exception: his entries at the NORAD console in the climax (`CHESS`, `GTW`,
`TIC-TAC-TOE`, and `ZERO` in tic-tac-toe) are in capitals, as abs0's transcription of the scene has them,
tagged `reconstructed`; M5 confirms or puts them in mixed case like the rest:

| # | Slug | Content |
|---|---|---|
| 1 | `first-contact` | The persona's dial; `LOGON: 000001`, `IDENTIFICATION NOT RECOGNIZED BY SYSTEM` / `--CONNECTION TERMINATED--` and the re-dial; `Help Logon`, `Help Games`, `List Games`; `Falkens-Maze`, refused. *Interactive.* |
| 2 | `joshua` | `LOGON: Joshua`; the header and status burst; the greeting and its small talk; GTW-vs-chess; `FINE.` *Interactive.* |
| 3 | `first-strike` | The side choice (`2`) and the targets (Las Vegas, Seattle); the big board: the first strike needs no order, then `Type("")` at the strike prompt carries out WOPR's war plan for the second (DEFCON 3) |
| 4 | `call-back` | WOPR phones David at home: `Incorrect identification. I am not Falken.`, `Falken is dead.`, `I'M SORRY TO HEAR THAT, PROFESSOR.` and the interrupted game, `What is the primary goal?` twice (`YOU SHOULD KNOW PROFESSOR. YOU PROGRAMMED ME.`, `TO WIN THE GAME.`), then the game clock (`GAME TIME ELAPSED`, `ESTIMATED TIME REMAINING`), ticking |
| 5 | `norad-terminal` | The NORAD session: `Joshua`, `Are you still playing the game?`, `28 HOURS` and the kill-ratio offer, GTW's kill-ratio table, `Is this a game or is it real?` / `WHAT'S THE DIFFERENCE?`, and Falken's address after `What classified address?` |
| 6 | `climax` | The board at DEFCON 1 while WOPR cracks the launch code; `List Games`; `CHESS`, refused; `GTW`, running; `TIC-TAC-TOE`: one player to a stalemate, `ZERO`, self-play, the montage, `Hello.` and `A STRANGE GAME…` |

As built, `Is this a game or is it real?` is in the NORAD session, where the fan transcriptions put it, just
before the classified address it leads to; v2.1's provisional table had it in the call-back. M5 settles it.

**Design** (as built).

- **Data.** Scenes are data in `internal/movie/scenes`, every step tagged (`scenes.Scene.Lines` feeds
  `script.Validate`). A `Scene` has a slug, a menu title, a one-line blurb, an `Interactive` mark and its
  steps:
  - `Type{Prompt, Text}`: the user's line. `proto.Typed` shows the prompt at once, hesitates, then types the
    text in bursts of one to four keystrokes at `PaceTyping` (8 chars/s) with seeded pauses, on one row
    (`Say.Open`), and leaves the blank line the console leaves after an answer. The text is a `script.User`
    line: mixed case, exempt from the capitals rule.
  - `Say{Lines, Pace}`, `Clear`, `Wait{D}`.
  - `Clock{Lines, Start, D}`: the call-back's game clock, in a panel at the top of the page where the text
    would be. Its two readings tick, elapsed up and remaining down, for `D` and stay up, ticking, until the
    scene ends. The labels, hours and minutes (`31 HRS 12 MIN`, `52 HRS 17 MIN`) are the film's as both
    transcriptions give them, tagged `reconstructed`; the seconds are this project's own (`original`), from
    00 up and 59 down, until M5 reads the film's. Under `testkit` the first reading goes into the transcript.
  - `Board{At, Play}`: the big board, drawn by `gtw.Film`, GTW's exported film-scenario renderer. A `Film` is
    the real game given the film's side and targets (`gtw.FilmSide`, `gtw.FilmTargets`) and then WOPR's war
    plan, stepped a frame at a time to a stage (`FilmStrike1`, `FilmStrike2`, `FilmRatios`, `FilmClimax`).
    With `Play` the war plays out on ticks and prints the game's own strip lines; without, the board jumps
    there silently. `At` 0 puts the board away: at the climax, `LIST GAMES` goes to the console, as GTW's
    does. While the board is at the climax it keeps cracking the launch code.
  - `Run{Slug, Mode}`: a real program launched over the director; the scene goes on when it ends. The board
    makes way for it. The climax runs `Launch{tic-tac-toe, "movie:N"}`: tic-tac-toe's movie mode types the
    film's answers itself (`1`; perfect moves for X, so every seed ends in a stalemate; `ZERO`) and reads each
    once it is on screen (`Drain`), then hands off to the ending in the same mode. The ending types `Hello.`
    itself and carries on cracking the code from N, the board's count.
- **Director.** `movie.Director` is the root `proto.Program` in place of the persona. It pins `movie.Seed`
  (1983) as the session seed and runs `Deterministic`, whatever `--seed` or `WOPR_SEED` says, so every replay is
  identical (RF-5): the typing jitter comes from `NewRand(Seed, DomainMovie|scene)` and the launched programs'
  seeds from the pinned session seed. It emits one step, then `Drain`, and the next step on `Drained`; a `Run`
  step waits for `GameOver`, then drains the program's last words. A `Drain` sent with the `Launch` reaches
  the director only if the program could not be built (the runner answers only the running program), and then
  the scene goes on after the host's `** GAME ROUTINE NOT AVAILABLE **`; the controls also work while a `Run`
  step is out, so a bad slug in the scene data cannot leave the show stuck.
- **Host hooks.** Key capture, `Hold`, `Drain` and `Skip`, plus `Say.Open` for typed lines (§4.4). While the
  climax runs on top of the director, keys do nothing but Ctrl+C (and PgUp/PgDn), and any other key shows
  `** THE GAME PLAYS TO THE END. CTRL+C QUITS. **` for three seconds; the menu and `--help` say so too. In
  movie mode the resolver marks every launched program `NoAbort`.
- **Controls**:

  | Key | Action |
  |---|---|
  | Space | Pause / resume (`Hold`; `** PAUSED **` shows under the text) |
  | → or `n` | Next scene; after the last, the end of the list |
  | ← or `p` | Previous scene; on the first, it starts again |
  | Esc | Scene menu |
  | `LOGOFF` or `q` (in the menu) | Exit 0 |
  | Ctrl+C | Exit 130 |

  Every jump ends a pause, stops the board and reveals what is queued at once (`Skip`), so the next scene
  starts on a clean page straight away. `--theme` and `--reduce-motion` apply. With `--only` the keys are the
  same: → and ← still move between scenes, and the scene they reach plays alone; → on the last scene is the
  end of the list. The film's order stays one keypress away, while the end of every scene still stops.
- **`--instant`** (or `WOPR_INSTANT=1`) drops the typing and the typewriter's pacing, but not the scenes'
  pauses: the host drops `Wait` under `Instant`, so the director waits each pause out itself on `Animate`
  ticks, and every page stays up to be read. For that, a scene's every page break and launch follows a pause
  (a one-second beat after typing where the film has none; a test checks), and so does its end. The board
  jumps to each stage, and tic-tac-toe and the ending play at once. The whole film takes about a minute. The package's transcript tests skip the pauses with an unexported option, since `testkit`
  has no clock; the UI tests and e2e play them on the fake and the real clock.
- **The scene menu.** `wopr -m` opens it: the scenes, numbered, the keys, and `SCENE:`, which takes a number, a
  slug, a title or a unique prefix. `q` (or `LOGOFF`, `EXIT`, `QUIT`) exits 0; an empty line asks again; an
  unknown name gets `NO SUCH SCENE.` and an ambiguous one lists its scenes. Esc during playback returns to it,
  and a scene picked there plays to the end of the list, then the menu returns; with `--only` it plays alone,
  then the menu returns, and the menu says which (`A SCENE PLAYS ALONE, THEN COMES BACK HERE.`).
- **Which scenes play.** `wopr -m <scene>` (number, slug, title or unique prefix, resolved by `movie.Resolve`)
  plays from that scene to the end of the list, then exits 0 (RF-5). An unknown scene exits 2 and prints the
  list; an ambiguous one exits 2 naming its scenes. `wopr --scenes` prints the number, slug and blurb of each
  scene and exits 0.
- **One scene** (the owner's request). `-o/--only` (`movie.Options.Only`) makes every scene the end of the
  list: `wopr -m joshua --only` plays `joshua` and exits 0, and `wopr -m --only` opens the menu, where each
  scene picked plays alone and the menu returns. The end of the list means what it does without `--only`:
  the menu if the viewer came from it, otherwise exit 0. `--only` without `--movie` exits 2, like
  `--movie --play`.
- **Isolation.** No LOGON, no persona, no Brain, no network.
- **Consistency tests** (`internal/movie`). `first-contact` and `joshua` are marked `Interactive`: their
  `Type` steps go through the real persona via `testkit`, and its transcript must begin with the scene's text.
  archtest allows the movie package's tests, and only them, to import `wopr` (§4.1). The call-back and the
  NORAD session are not the persona's to play, but WOPR's answers there (`YOU SHOULD KNOW PROFESSOR. YOU
  PROGRAMMED ME.`, `TO WIN THE GAME.`, `WHAT'S THE DIFFERENCE?`) must equal what the persona's scripted brain
  answers to the same questions. `first-strike`, board strip included, must equal GTW's own transcript for
  the same lines, and the climax's NORAD notices GTW's replies to them. Movie mode and interactive play cannot
  drift apart, and the scenes double as end-to-end tests.
- **Tests.** The whole film is a golden (`internal/movie/testdata/film.golden`), and two replays must be
  identical; the UI goldens `movie_menu` and `movie_screens` show the menu, a line half typed, the pause, the
  first strike, the climax board, tic-tac-toe playing itself and the game clock at 80×24 with the front
  panel. The e2e cases are in §9.
- **Legal scope** (RK-1 in §16, SL-8). M6 ships the film's WOPR terminal script, much of which the
  interactive persona already contains. The only film text in the scenes is what WOPR's terminal shows on
  screen, David's typing included: no spoken-only dialogue, no audio, no stills. The rest is this project's
  own text, tagged `original`: the dial (`CONNECTING...`, `CONNECTED.`), GTW's strike exchange and board
  legend, tic-tac-toe's prompts and the game clock's seconds. Every line is tagged, and `NOTICE.md` excludes
  the quotations from the MIT grant and gives rights holders a contact. The owner accepts the remaining risk
  by requesting the feature; AGENTS.md has the takedown runbook.

---

## 8. Binary size and portability

- **Build** (GoReleaser, the same config for snapshots and releases; `.goreleaser.yaml`): `CGO_ENABLED=0`,
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
- **Size gate**: `go run ./internal/tools/sizegate -expect 6 -packages 4`, which reads `dist/artifacts.json`
  (B-7, B-9).
  - It reads GoReleaser's artifact list and gates the `Binary` entries and the Linux packages.
  - It fails on an empty list, or when the number of binaries is not the expected six or the number of
    packages not the expected four.
  - It writes a table to the step summary, warns above 10 MB and fails above 15 MB.
  - GoReleaser's `report_sizes` also logs the sizes.
  - The README states only the budget (B-8).
- **Linux packages** (owner request 2026-10-07; `nfpms` in `.goreleaser.yaml`, GoReleaser's nFPM): a `.deb`
  and an `.rpm` for linux/amd64 and linux/arm64, holding the same binary as the archives.
  - **Where files go.** The `.deb` follows Debian Policy 4.7: the program in `/usr/games` (§11.11; it is on
    the `PATH` of every Debian and Ubuntu login), the manual page as `/usr/share/man/man6/wopr.6.gz`
    (§12.1), and in `/usr/share/doc/wopr/` the copyright file (§12.5), `changelog.Debian.gz` (§12.7, nFPM
    writes it from the change notes), the release notes `CHANGELOG.md` as `NEWS.gz` (§12.7), and
    `README.md.gz` and `NOTICE.md.gz` (§12.3). The `.rpm` follows Fedora's packaging guidelines: the program
    in `/usr/bin` (the Games SIG: "Binaries go in %{_bindir} and not /usr/games"), the same manual page,
    `LICENSE`, `NOTICE.md` and `THIRD_PARTY_NOTICES.txt` as `%license` in `/usr/share/licenses/wopr/`
    (`NOTICE.md` holds licence texts too), `README.md` and `CHANGELOG.md` as `%doc` in
    `/usr/share/doc/wopr/`, both directories owned by the package, and `%changelog` from the change notes.
  - **The copyright file** `packaging/debian/copyright` is machine-readable (DEP-5) and generated by
    `internal/tools/notices` with `THIRD_PARTY_NOTICES.txt`, and checked with it: wopr's licence from
    `LICENSE`; the film text it does not cover, file by file from the provenance tags (§2.1), with the
    manual page, the screenshots and the docs site's pages; abs0's transcription and Matthew Thomas's map with their terms
    from `NOTICE.md`; and every module linked into the Linux builds, each filed under Expat or BSD-3-clause
    only if its licence has those words, so any other licence stops the tool until someone files it.
  - **Metadata.** Section `games`, priority `optional`, no dependencies (the binary is static), homepage the
    repository, maintainer and packager `GhostofGoes <6599820+GhostofGoes@users.noreply.github.com>`
    (GitHub's no-reply address for the owner's account, so no mailbox is published), RPM vendor
    `GhostofGoes`, `License: MIT AND BSD-2-Clause AND BSD-3-Clause AND LicenseRef-Matthew-Thomas-map`
    (wopr, abs0's transcription, the Go modules, and Matthew Thomas's map, which is compiled into the
    program and whose terms have no SPDX identifier, so it gets a LicenseRef with the name the `.deb`'s
    copyright file gives it; the film text is under no licence), no RPM `Group` (Fedora asks for none).
    The synopsis and description are `packaging/description.txt`, which `pkgdocs`'s tests hold to both
    formats' rules.
  - **No desktop entry and no AppStream metadata** (decided 2026-10-07, open to the owner). wopr is a
    console program: neither Debian Policy nor Fedora's guidelines require a `.desktop` file or a
    metainfo file for one, and lintian and rpmlint are clean without them. A menu entry would open a
    terminal of unknown size for a program that needs 80×24, and a software centre lists an AppStream
    `console-application` from a distribution's catalogue, which a package installed by hand is not in.
    Either can be added later, checked in CI with `appstreamcli validate --pedantic` and
    `desktop-file-validate`.
  - **Versions and names.** Debian revision and RPM release 1, named by each format's convention:
    `wopr_X.Y.Z-1_amd64.deb` (dpkg-name) and `wopr-X.Y.Z-1.x86_64.rpm`, `arm64` and `aarch64` likewise. A
    snapshot is `X.Y.Z~snapshot.<commit>-1`, which dpkg and rpm sort before `X.Y.Z-1`. The package
    changelogs name every version the same way (relnotes) and date it by its tagged commit, so that the
    newest entry is always the latest, even for two releases on one day; its first entry is always the
    package's own version, a snapshot's included (§11.3). The manual page names the same version:
    `pkgdocs` writes it into the page's header for every build.
  - **Reproducible.** Every time inside is the commit's (`mtime`), every mode is pinned, the RPM build host is
    the fixed `reproducible`, the payloads are xz (deterministic), and the compressed documents come from
    `internal/tools/pkgdocs`, which compresses like `gzip -9n` (no name, no time) and runs as a GoReleaser
    before hook into `build/pkg`, where it also copies the packages' `changelog.yml` from `WOPR_NOTES_DIR`
    (`nfpms.changelog` takes no template). Two snapshot builds in different clones, with different umasks,
    gave the same `checksums.txt` (2026-10-07); `repro` compares the packages with everything else.
  - **Lint** (2026-10-07). lintian 2.117 with `--pedantic --display-experimental`, Debian and Ubuntu profiles,
    finds nothing but `statically-linked-binary`, which the package overrides: lintian knows a Go program is
    static only from a Debian source package's build dependencies. rpmlint 2.8.0, Fedora 44's (with its
    configuration and the `hunspell-en-US` dictionary it pulls in), reports `statically-linked-binary` and
    `position-independent-executable-suggested`, both by design (one static binary that runs on every
    distribution), and `spelling-error` for "Falken's" and "tac" (of tic-tac-toe) in the description.
    `packaging/rpmlintrc` filters those four (`rpmlint -r packaging/rpmlintrc`), each with its reason, and
    `invalid-license` for the map's LicenseRef, which no list of licences has (the License tag gained it
    after that run, so this filter is not yet confirmed against Fedora's rpmlint). The
    packages carry no GPG signature (there is no signing key, only `GITHUB_TOKEN`); like every release file
    they are attested and can be checked with `gh attestation verify` (§12). zypper refuses an unsigned
    package unless given `--allow-unsigned-rpm`, so the Linux (RPM) install tab's openSUSE line passes it;
    checking the package first is optional, in the Installation page's "Verifying binaries (attestation)".
  - **Looking inside an `.rpm`.** nFPM's RPM writer (google/rpmpack) stores the payload's paths as absolute
    (`/usr/bin/wopr`, where rpmbuild writes `./usr/bin/wopr`) and with no times. `rpm` and `dnf` install
    and verify the packages correctly, but `rpm2cpio X.rpm | cpio -idm` writes into the live `/` from any
    directory. Use `rpm -qlvp`, `bsdtar -xf X.rpm -C dir`, or `cpio -idmv --no-absolute-filenames`.
- **Never UPX** (Windows AV false positives).
- **Windows.**
  - Ctrl+C arrives as a key in raw mode on every OS, so `Update` maps it to `tea.Interrupt`. A blocked
    `Update` would delay it, which is why all slow work is `Think`.
  - Keyboard enhancements stay off.
  - conhost resize events may report the buffer height (9001). Clamp to the window rect and check in the M2 QA
    pass.
  - Ship unsigned. The one-line install downloads with `curl.exe`, which sets no Mark of the Web, so
    SmartScreen does not ask; the Installation page's "Verifying binaries (attestation)" documents
    SmartScreen for files downloaded with a browser, and verifying as optional (§12, §13).
- **Panics**: no custom `recover` around `p.Run()`. Bubble Tea restores the terminal, recovers `Cmd` goroutine
  panics too, and prints the stack to stderr. The model returned on panic is nil and is not used.
- **Debug log** (S-5, `internal/debuglog`): `WOPR_DEBUG=1` writes `os.UserCacheDir()/wopr/debug.log`. It
  records the version and session seed, the terminal size and colour profile, each `Think`'s duration and
  whether it hit its deadline, and how the session ended. An e2e test checks that typed text never appears.
  - Directory mode 0700, file mode 0600 (applied on creation; wopr is the only writer of that path).
  - If there is no cache dir, logging is disabled and a note goes to stderr.
  - The path is printed on exit.
  - wopr's debug log never holds typed input.
  - Bubble Tea's own `TEA_TRACE` and `TEA_DEBUG`, if the user sets them, record raw input and output or write a
    panic log into the CWD. wopr never sets them; they are the user's own developer switches (SL-7).

---

## 9. Testing

- **Unit tests** cover:
  - game rules (via `testkit`);
  - `Resolve`, `cli.Parse`, intent and offers (the film lines, the review's cases and the false-positive
    table);
  - the LOGON table;
  - the host runner: routing, launch seeds, hand-off, the Esc machine, Think deadlines, Animate;
  - the scripted brain, typewriter dt math and pauses, and the line editor;
  - `Sanitize*`, the canvas, and the themes: every Style defined, distinct pairs in 16 colours and without
    colour, contrast, and each profile rendering only its own colours;
  - accessibility (M5): the sweeps of every game in `internal/ui/access_test.go` (§4.5).
- **Architecture**: `archtest` covers the DAG (test imports and e2e files included), toolchain equality in CI,
  the tool modules' `go` lines, the pins shared between `prek.toml` and the tool modules, and the lint
  self-test. Each data package's tests check its provenance tags.
- **Golden** (Q-3, A-14).
  - `internal/golden.Assert(t, name, got)` stores files under each package's `testdata/`. They are byte-exact
    apart from CRLF, which is normalised; trailing padding in a canvas golden is intentional, and the
    whitespace hooks skip `.golden` files (TO-2).
  - Regenerate every golden with `WOPR_UPDATE_GOLDEN=1 go test ./...`, which works on every OS and package.
  - **Persona and game transcripts** come from `testkit`: the real runner, `Instant` and `Deterministic`, with
    `[CLEAR]` and `[NOTICE]` markers.
  - **UI screens** come from `internal/ui/drive_test.go`, a synchronous driver in package `ui` (the model is
    unexported, so the plan's `ui/testutil` package could not reach it). A fake `Scheduler` records armed
    ticks and a fake clock delivers them; commands run inline, `tea.BatchMsg` is unwrapped, and `Think` results
    can be held to simulate a slow search. It snapshots `View().Content` after `ansi.Strip`, with trailing
    spaces trimmed, plus the cursor position.
  - `tea.Sequence` is banned (forbidigo).
  - **Flows** (M1): persona `film_greeting`, `logon_fail_x3_hint` (a game name at LOGON is not a login),
    `logon_list_then_number`, `help_list_games`, `intent_false_positives`, `offer_accept`, `stub_game`; UI
    `logon_joshua`, `logoff_exit0`, `too_small_pause_resume`, `esc_confirm`, `typeahead_thinking`,
    `front_panel`. Then each set piece (`film_path` in M2) and each movie scene.
- **Fuzz** (Q-2): `FuzzClauses`, `FuzzSanitizeInput`, `FuzzEditor`, `FuzzResolve`, `FuzzGameIntent`, and the M7
  reply parser.
  - Seed corpora run in plain `go test`.
  - CI discovers every target with `go test -list '^Fuzz'`, runs each for 10 s on Linux, and uploads any
    crashing input as an artifact.
- **End-to-end** (Q-1, U-8).
  - `internal/e2e` (build tag `e2e`) is a Go test that drives the **real binary** in a pseudo-terminal. It uses
    `charmbracelet/x/xpty` (a Unix pty, or ConPTY on Windows) and feeds the output to a
    `charmbracelet/x/vt` emulator, so tests wait for text on the **screen**: the typewriter and Bubble Tea's
    diffing renderer split text across writes, so the byte stream is unreliable. The emulator's replies to
    terminal queries go back to wopr, as a real terminal's would.
  - On Unix the child gets the pty as its controlling terminal (`Setsid`, `Setctty`); xpty leaves that to the
    caller, and without it a resize never arrives as SIGWINCH.
  - **Cases**, by milestone (IM-3):
    - M0: `--version`, `--games`, `--licenses`, a closed pipe exits 0, refusal without a terminal (`--movie`
      too);
    - M1: start → `LOGON:` → Ctrl+C → exit 130 with the alternate screen left; `Joshua` → `GREETINGS
      PROFESSOR FALKEN.` → `Hello.` → `HOW ARE YOU FEELING TODAY?` → `LOGOFF` → exit 0; start at 60×20 →
      `TERMINAL TOO SMALL` → resize to 80×24 → `LOGON:`;
    - M6: `--movie climax --instant` shows the board, `** ACCESS DENIED **` and the film's last words, each
      held by the scene's pauses, and exits 0 at the end of the list without a key; paced, `-m 2` → `LOGON:` →
      Space → `** PAUSED **` and a still screen → Right → the next scene → Esc → `SCENE:` → Ctrl+C → exit
      130; `-m` → the scene menu → `q` → exit 0; `--scenes` lists the scenes, and `-m nowhere` exits 2
      listing them; `-m joshua --only --instant` plays that scene and exits 0 at its end without a key, never
      showing the next scene's `WHICH SIDE DO YOU WANT?`;
    - after v0.2.0: `--only` without `--movie`, `WOPR_SEED=x` and `-s x` exit 2 naming their source, and with
      `WOPR_DEBUG=1 WOPR_SEED=1983` the debug log records `seed 1983 (pinned: true)` and nothing typed.
  - On Unix every TUI case also asserts that the terminal modes (termios) are restored. Under ConPTY the
    assertion is weaker (exit code and final screen), because conhost owns the console modes.
  - The `build` job cross-compiles the test per target (`internal/tools/stage`) and ships it next to each
    binary, so smoke runners need no Go toolchain.
- **Race.**
  - CI runs `go test -race ./...` on `ubuntu-24.04` and `macos-26`, and plain `go test ./...` on
    `windows-2025`; the race detector there would need a C toolchain on the runner.
  - `testkit` exercises `View` concurrently with `Think` so the detector has something to see.
  - Contributors: `go test ./...` everywhere. Add `-race` on Linux and macOS, and on Windows amd64 only with a
    C toolchain. windows/arm64 has no race detector (B-14).

---

## 10. Tooling: prek and static analysis

- **One hook set in `prek.toml`**, run identically locally (`prek install` installs both the pre-commit and
  pre-push shims) and in CI.
  - **The gate is `prek run --all-files`.** `prek validate-config` checks syntax only; it resolves neither revs
    nor hook ids (T-1, T-9).
  - **Ordering**: each file-mutating fixer gets its own priority, because equal priorities run concurrently and
    lose updates (T-8). Formatters come next, then read-only linters.
  - golangci-lint's `golangci-lint-full` entry is overridden to drop `--fix`. zizmor runs without `--fix`
    (T-5).
  - actionlint checks workflow syntax. Same-repository calls use GitHub's `$/` syntax (zizmor's
    `self-repository` audit asks for it); actionlint 1.7.12 predates it, so `.github/actionlint.yaml` ignores
    that one message for `$/` calls only.
  - The whitespace and end-of-file fixers skip `.golden` files, whose padding is intentional (TO-2).
- **Pinning** (T-3, T-10).
  - Every hook rev is a full commit SHA with a `# frozen: vX.Y.Z` comment. A note about a rev goes on its own
    line above it, because `prek update` rewrites the comment after `rev` (TO-9).
  - `[update] freeze = true, cooldown_days = 14` makes a plain `prek update` (formerly `auto-update`) keep that
    form.
- **Dependency cooldown.** Go modules wait 14 days as hooks do: `internal/tools/cooldown` fails when any
  module in wopr's, the tool modules' or the site's build list is younger than that at HEAD's committer
  date, by the version time the module proxy reports. It runs in CI's `lint` job and as a prek hook on
  `go.mod`/`go.sum` (measured at the current time there). A compromised upstream commit then has two weeks
  to be noticed before it can reach a release. `tools/cooldown-exceptions.txt` lists `module@version` pairs
  that may skip the wait, each with its reason (a vulnerability fix).
- **Go tools** (T-4, T-11, decision 20). Four tool modules, pinned and checksummed by their `go.sum`, run as
  `go tool -modfile=<module> <tool>`:
  - `tools/go.mod`: gitleaks and govulncheck;
  - `tools/lint/go.mod`: golangci-lint (for the lint self-test and the Commands table);
  - `tools/release/go.mod`: GoReleaser and changie;
  - `tools/docs/go.mod`: Hugo, for the docs site (§13). Its own module because its graph is large.

  golangci-lint is separate because sharing a module with gitleaks pulls `x/ansi` to a version that breaks
  gitleaks's build (SL-2). **No tool module's `go` line may be newer than the root `toolchain` line**, because
  CI runs every tool with `GOTOOLCHAIN=local`; GoReleaser's `go` line tracks the newest Go patch, so a GoReleaser
  bump usually needs a toolchain bump first, in its own PR (CR-4, TO-7). `TestToolModulesFitTheToolchain`
  enforces it. Tools pinned twice (golangci-lint and gitleaks, in `prek.toml` and in a tool module) must agree;
  `TestToolPinsAgree` checks (TO-8).
- **golangci-lint** (`.golangci.yml`, v2.14.0):
  - `default: standard` plus revive, gocritic, misspell, forbidigo and depguard; formatters gofmt, goimports
    and gofumpt. `run.build-tags: [e2e]` brings the e2e tests under lint (TO-6).
  - **forbidigo** (T-2, A-11), with `analyze-types: true`:
    - `^tea\.(Tick|Every|Sequence)$` with `pkg: ^charm\.land/bubbletea/v2$`, exempt in `internal/ui/clock.go`;
    - the top-level `math/rand/v2` functions, and every `math/rand` (v1) function (TO-5);
    - `time.Sleep`, `After`, `AfterFunc`, `Tick`, `NewTicker` and `NewTimer`, exempt in tests.
  - `warn-unused` is on, so a stale exclusion shows up; the v2 exclusions for `rand.go` and for misspell on film
    text never matched (misspell's default English accepts `RECOGNISED`) and are gone (TO-4).
  - **Self-test** (TO-1, AR-3, IM-1). `config verify` accepts rules that never fire, so
    `internal/archtest/testdata/lintfixture` holds a deliberate violation of each rule, one per line. Under
    `testdata/` it is outside `./...`, so neither the lint gate nor archtest sees it.
    `WOPR_LINT_SELFTEST=1 go test -run TestLintRulesFire ./internal/archtest` lints that path with the
    `tools/lint` golangci-lint and requires each rule's report; CI's lint job runs it.
- **codespell** skips script and art data with a prek `exclude` regex, `(^|/)testdata/` and every `go.mod` and
  `go.sum` included (the tool modules' hashes produced false positives, TO-3); `crasher` is on its ignore
  list.
- **ShellCheck** stays in the hook set for any future script. CI and tooling logic is Go (`internal/tools/*`),
  so it runs on Windows (B-14).
- **Change notes** (§13): a local `change-notes` hook runs `go run ./internal/tools/relnotes -check` when
  `.changes/`, `.changie.yaml`, `CHANGELOG.md` or relnotes itself changes, and on every `--all-files` run.
  It builds the pinned changie, so the generated `CHANGELOG.md` and version files are also kept to what
  rumdl and the whitespace fixers accept.
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
  setup-go exports `GOTOOLCHAIN=local` itself after installing 1.27.1. `archtest` asserts the exact toolchain in
  CI.
- **Caching.** `cache: false` in `release.yml` and in any job that runs GoReleaser (B-4, B-9). Compiling
  GoReleaser from a cold cache costs about 1.5–2 minutes per such job; that is accepted (CR-4). Every other
  `ci.yml` job keys its cache on the `go.sum` files of the modules it builds (`cache-dependency-path`): with
  one shared key, the first job to finish saves the cache for all of them. `scheduled.yml` uses none.
- **Timeouts.** Every job sets `timeout-minutes` (5 to 15 for the short jobs, 20 for `build`, 30 for `test`
  and the two release builds), and the smoke step runs the e2e binary with `-test.timeout=5m`, so a hang
  fails with every goroutine's stack instead of holding a runner for six hours.
- **`fetch-depth: 0`** in every job that runs GoReleaser, so snapshot versions reflect the latest tag and the
  release `build` and `rebuild` jobs produce the same version (CR-7).
- **`.gitattributes`**: `* text=auto eol=lf`, plus `*.png binary`. Windows runners check out with
  `core.autocrlf=true`, which would otherwise break goldens (B-3).
- **Concurrency** (CR-2) is set in `ci.yml` and `release.yml` only, never in the reusable `smoke.yml`: a called
  workflow sees its caller's `github.ref`, and the same group in both deadlocks. Pull requests share a group per
  PR, and pushes to other branches a group per branch; both cancel superseded runs. Pushes to `main` get a group
  **per commit**, because with a shared group a newer pending run replaces an older pending one, which would
  leave a merged commit without `ci-ok`:

  ```yaml
  concurrency:
    group: >-
      ${{ github.event_name == 'pull_request' && format('ci-pr-{0}', github.ref)
      || github.ref == 'refs/heads/main' && format('ci-{0}', github.sha)
      || format('ci-branch-{0}', github.ref) }}
    cancel-in-progress: ${{ github.event_name == 'pull_request' || github.ref != 'refs/heads/main' }}
  ```

  `release.yml` uses `release-${{ github.ref }}` and `scheduled.yml` none.

### 11.2 `ci.yml` (on `pull_request` and `push` to any branch)

CI runs on every branch push (owner decision 2026-10-06), so a branch is checked before anyone opens a pull
request. To avoid running the same work twice (owner request, same day), a pull request from a branch of this
repository is checked by that branch's push run, which reports `ci-ok` on the same head commit; its
`pull_request` run skips every job. A skipped job counts as passed, so the skipped aggregator must not be
called `ci-ok`: its name is an expression, which GitHub does not evaluate for a skipped job, so its check is
listed under the expression's text. The name must never become a plain `ci-ok`. A fork's pushes never reach
this repository, so pull requests from forks run in full. Tags go to `release.yml` only.

The push run tests the branch as it is, not merged with `main` as a `pull_request` run would. The `main`
ruleset therefore requires branches to be up to date before merging (§12): updating a branch is a push, so
its run tests exactly the tree the squash merge produces.

| Job | Runner(s) | Does |
|---|---|---|
| `lint` | `ubuntu-24.04` | `fetch-depth: 0`. prek via `j178/prek-action` with `prek-version: 0.5.5`, including the `change-notes` hook (`relnotes -check`: the notes parse and fit one Debian changelog line, `CHANGELOG.md` is `changie merge`'s output, every release tag has its `.changes/vX.Y.Z.md`). The lint-fixture self-test (`WOPR_LINT_SELFTEST=1`). `relnotes -since origin/main`: a branch that changes a non-test file under `cmd/` or `internal/` (test-only packages aside), `.goreleaser.yaml`, `packaging/` or `docs/man/` adds a change note or carries a `Changelog: none` trailer (AGENTS.md). Separately, a zizmor online-audits step with `GH_TOKEN` scoped to that step only (T-5). |
| `secrets` | `ubuntu-24.04` | `fetch-depth: 0`; gitleaks over the checked-out history (`--log-opts="--full-history HEAD"`: the pushed branch, or a fork's PR merged with `main`; not every branch, so a stale branch cannot block every PR). `--no-color`, because gitleaks colours its log even into a pipe, so the step fails on any `ERR` line, on `0 commits scanned`, and on a finding (S-1, SL-2). |
| `test` | `ubuntu-24.04`, `macos-26`, `windows-2025` | `go test -race ./...` (plain on Windows). On Linux: every fuzz target for 10 s, with crashing inputs uploaded on failure; `govulncheck`; the third-party-notices check (§12); the manual-page check (§13). |
| `build` | `ubuntu-24.04` | `relnotes -snapshot`, the release notes for this commit's snapshot version as `release.yml` makes them for a tag (§11.3), shown in the job summary and passed to GoReleaser in `WOPR_NOTES_DIR` (on a release pull request, whose version is batched but not yet tagged, they are that version's notes under the snapshot's version, so the packages' first changelog entry is still their own); GoReleaser snapshot of all six targets and the four Linux packages; the size gate; `internal/tools/stage -archives -assets dist/release`, which copies each binary to `stage/<os>_<arch>/`, cross-compiles the e2e test next to it, puts the `.deb` and `.rpm` beside the Linux ones, checks every archive's and package's contents, and collects the release files (below). Uploads one download per platform, the bare binary as-is (`archive: false`, so it is not zipped; it loses its executable bit, and `--licenses` prints its notices), named after its file such as `wopr_<version>_linux_amd64`, kept 30 days on `main`, 7 on other branches and 3 on PRs (B-13); and a `smoke-bundle` of every staged binary and e2e test for this run's smoke jobs (1 day). |
| `smoke` | `ubuntu-24.04`, `ubuntu-24.04-arm`, `macos-26`, `macos-26-intel`, `windows-2025`, `windows-11-arm` | The reusable `smoke.yml`, with one input, `artifact` (CR-3). It downloads the staged artifact, `chmod +x`es the files (artifacts lose the executable bit), and runs the e2e test against the binary with `-test.timeout=5m`; no Go toolchain. Every target has a native runner, so none is skipped. On the two Linux runners it also installs the `.deb` with `sudo apt-get install`, finds `wopr` on the `PATH` that `/etc/environment` gives every login and its manual page with `man -w`, checks that the changelog starts with the package's version, and removes it; and installs the `.rpm` with `dnf` in a Fedora 44 container pinned by digest, with no network (it needs no other package), runs `rpm -V`, checks the changelog, and removes it. Neither removal may leave a file behind. |
| `docs` | `ubuntu-24.04` | Builds the docs site (§13) as `docs.yml` does, with `--panicOnWarning --printPathWarnings`: a deprecated setting, a broken internal link, a missing screenshot or a malformed game data file fails it. |
| `ci-ok` | `ubuntu-24.04` | `needs: [lint, secrets, test, build, smoke, docs]`, `if: always()` (and skipped, under an unevaluated expression name, with the rest on a PR from this repository). Fails unless every needed job succeeded. **This is the only required check** (S-3), so matrix names never appear in settings. |

### 11.3 `release.yml` (on `v*` tags): gated, reproducible, attested (B-10, S-7)

1. **`verify`** (`contents: read`, `checks: read`). The tag is semver; the tagged commit is an ancestor of
   `origin/main`; and every `ci-ok` check run that GitHub Actions posted on that commit succeeded (a check
   of the same name from another app is ignored). A tag pushed right after its merge arrives before `main`'s
   CI run has finished, so a missing or pending `ci-ok` is waited out, for up to 45 minutes; any other
   result fails at once. (v0.2.0's first release run failed on exactly that race.)
2. **`build`** (`contents: read`, no OIDC). First the release notes: `internal/tools/relnotes -version vX.Y.Z`
   writes `notes.md` (the release body), `CHANGELOG.md` and `changelog.yml` (nFPM's chglog format, for the
   `.deb` and `.rpm` changelogs) into `build/notes`, which `.gitignore` keeps out of the tree GoReleaser
   needs clean. GoReleaser's templates find it through `WOPR_NOTES_DIR`; `nfpms.changelog` takes no
   template (GoReleaser v2.18.2), so `pkgdocs` copies `changelog.yml` to `build/pkg`, which it names.
   `notes.md` is uploaded for `publish`. They
   come from the release pull request's `.changes/vX.Y.Z.md`, or, if it did not batch the notes, from
   `.changes/unreleased` batched in a temporary copy, dated by the tagged commit, with a warning. relnotes
   refuses when an earlier tag has no `.changes/vX.Y.Z.md`, whose notes would repeat; when another
   version is batched but not tagged, so the tag names the wrong version; and when there are no notes at
   all. It warns when the tagged commit holds notes the release does not list, and when the version
   header's date is not the tagged commit's (AGENTS.md, "Releasing"). Only `vX.Y.Z` tags count as
   releases. Then `goreleaser release --clean --skip=publish`, the size gate,
   then `stage -archives -assets dist/release`, which checks that every archive contains `LICENSE`,
   `README.md`, `NOTICE.md` and `THIRD_PARTY_NOTICES.txt` (B-11), that the Linux and macOS archives hold the
   manual page `wopr.6` at their root and the Windows ones do not (Windows has no `man`), that each Linux
   build has one `.deb` and one `.rpm` named by its format's convention and installing what §8 lists (from
   the `.deb`'s control archive and the `.rpm`'s header), and collects every file the release publishes
   into `dist/release`: the six archives, the six bare binaries (a GoReleaser `binary`-format archive, named
   like the archives, `.exe` on Windows), the four Linux packages, those four documents, and
   `checksums.txt`, which GoReleaser writes over all of them (`checksum.extra_files` adds the documents). Each file must match its
   checksum line. The staged files and `dist/release` are uploaded.
3. **`smoke`**: the reusable workflow, run on the staged release binaries.
4. **`rebuild`** runs beside `build`, not after it: relnotes and GoReleaser again from a fresh checkout on
   `ubuntu-24.04-arm` (cross-compiling), uploading its `checksums.txt`. relnotes takes every date from a
   version header or a commit, so both jobs package the same notes. **`repro`** then diffs the two
   `checksums.txt` files. Any difference fails the release.
5. **`publish`** (`contents: write`, `id-token: write`, `attestations: write`, `discussions: write`), on a tag
   push only, and only when every earlier job succeeded.
   1. `actions/attest@v4` with `subject-checksums: assets/checksums.txt`, so every published file is
      attested.
   2. `gh release create vX.Y.Z --draft --verify-tag --notes-file notes.md` with every file from
      `dist/release`. GitHub's generated notes stand in, with a warning, only if `notes.md` is empty.
      GoReleaser's own changelog is disabled (CR-8): the notes are the change notes, not commit subjects.
   3. `gh release edit vX.Y.Z --draft=false --discussion-category Announcements`, which also starts the
      release's discussion (the job has `discussions: write`). If that fails, as when the category does not
      exist, it publishes without a discussion and warns.

   Immutable releases (§12) lock the release at publish time. A failed publish leaves only a draft.
6. **`docs`** (`actions: write` only), after a successful `publish`, runs
   `gh workflow run docs.yml --ref main`, which publishes the docs site again so that its download commands
   name the new release (§13). A run that `GITHUB_TOKEN` starts through `workflow_dispatch` is the one kind
   such a token may start.

**Dry run** (CR-1, IM-2). `workflow_dispatch` runs steps 2–4 with `--snapshot`: GoReleaser in release mode
refuses an untagged commit, and there is no tag before v0.1.0. `verify` is then skipped, so every later job
states its condition explicitly (`!cancelled()` and the results of its needs); otherwise the implicit
`success()` would skip them too.

**Never re-tag.** A bad release is fixed by the next patch version, plus a `retract` directive for
`go install` users. That rule is in `SECURITY.md` and `AGENTS.md`.

### 11.4 Runner labels

The labels as of 2026-10-07 (the M5 checklist's first run) are `ubuntu-24.04`, `ubuntu-24.04-arm`, `macos-26`,
`macos-26-intel`, `windows-2025` and `windows-11-arm`, all free on public repositories and none announced for
removal. Ubuntu 26.04 has been generally available since 2026-09-17, and `ubuntu-latest` moves to it between
2026-10-19 and 2026-11-19; the pinned `ubuntu-24.04` labels do not move. `windows-2025` and `windows-11-arm` now
run the images with Visual Studio 2026 (the Arm label moved in September 2026). `ubuntu-22.04` is deprecated and
unsupported from 2027-04-17, and `macos-14` is unsupported from 2026-11-02, so neither is used. The labels are
checked at every milestone boundary (AGENTS.md checklist) (B-5).

### 11.5 Dependency updates without Dependabot (R13)

- **`scheduled.yml`**, weekly, `contents: read` plus `issues: write` on the report job only (CR-6, TO-9). Each
  check has a rule that separates a **finding** (something to update or fix, filed in the tracking issue) from
  an **error** (the check could not run, which fails the job instead):
  - `govulncheck` on `main`: exit 3 is a finding, any other failure an error;
  - a newer **Go patch release** than `go.mod`'s `toolchain` line (`go list -m toolchain@patch`), because
    the response rule below counts from the release, whether or not govulncheck finds the fixed code
    reachable;
  - outdated **direct** requirements of the main module (`go list -m -u` with a template); indirect upgrades are
    left to govulncheck and the manual cadence, or the report would never be empty;
  - outdated **tools**, selected by each tool module's `tool` pattern (they are indirect requirements there),
    and a newer **docs theme** than `site/go.mod` pins;
  - `prek update --check`: "would update" is a finding, "update failed" an error;
  - the third-party-notices check.

  A finding opens or comments on a single tracking issue; a week with none closes it. It never opens PRs, so
  "Allow GitHub Actions to create pull requests" stays off.
- **Manual cadence** (AGENTS.md checklist, at every milestone and at least monthly):
  1. update direct dependencies to versions at least 14 days old with `go get <module>@<version>` (never
     `go get -u ./...`, which lifts indirect modules to their newest untagged commits), then `go mod tidy`;
  2. update the tool modules with `go get -tool`, the toolchain first if a tool needs it, by the same rule;
  3. `prek update`;
  4. bump action SHAs from their release tags, each at least 14 days old;
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
  - gitleaks runs as a pre-commit hook (staged changes) and in the `secrets` job (the checked-out history,
    §11.2).
  - A finding is handled by rotating the secret first, then adding its fingerprint to `.gitleaksignore` in a
    PR; history on `main` is never rewritten (SECURITY.md, SL-2).
  - M0 proves the job with a **canary**: a throwaway branch with a fake secret that only gitleaks recognises
    (a GitHub-supported token pattern would be stopped by push protection). Its push run's log must say
    `leaks found`, not an error. Then the branch is deleted, because a leftover branch would keep matching.
    AGENTS.md has the steps.
  - `.gitignore` adds `*.pem`, `*.key`, `dist/`, `stage/`, `build/` (release notes and package documents) and
    editor/OS files.
- **Repository settings** (S-3, S-7, S-8). The owner applies them in M0; AGENTS.md lists them.
  - **`main` ruleset**: pull request required; required check `ci-ok` (from GitHub Actions), with branches
    required to be up to date before merging (§11.2); no force-push; no deletion. Merges are **squash
    only**, which keeps `main`'s history linear. It does not keep v1's commits out of the public repository:
    the branch and any PR ref still serve them, and the v1 review quotes three fragments of the CC BY-SA map.
    `NOTICE.md` therefore credits Franklin Wei's map under CC BY-SA 4.0 for those fragments (SL-6, L-4).
  - **`v*` tag ruleset**: creation, update and deletion restricted, with only the owner able to bypass.
    **Immutable releases** are enabled before v0.1.0.
  - **Secret scanning** and **push protection** must be enabled at repository level; an audit on 2026-10-06
    found both off. Also **private vulnerability reporting**, which `SECURITY.md` sends reporters to.
  - **Issues and Discussions**: issue forms for bug reports and feature requests (`.github/ISSUE_TEMPLATE/`,
    blank issues off, with links to Discussions, a private security report and the docs site). Discussions
    must be turned on: the forms, the docs site's Troubleshooting page and the README send questions there.
  - **Actions**:
    - allow only listed actions (`actions/*`, `j178/prek-action`);
    - require actions pinned to a full-length commit SHA;
    - default `GITHUB_TOKEN` read-only;
    - workflows may not create or approve PRs;
    - approval required for **all** external contributors' workflow runs.
  - **Dependabot**: off (owner decision). govulncheck covers the Go graph.
- **The app.** No network code before M7, no telemetry, no transcripts. External text is sanitised. wopr's
  debug log holds no typed input (§8).
- **Releases.** Every file is attested. Verifying is optional, for those who want proof, in the Installation
  page's last section, "Verifying binaries (attestation)", which the README and each release's notes link:
  - Command: `gh attestation verify <file> --repo GhostofGoes/WOPR`, plus
    `--signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml`,
    `--source-ref refs/tags/vX.Y.Z` and `--deny-self-hosted-runners`.
  - For a file downloaded with a browser, after verifying: `xattr -d com.apple.quarantine`, or the
    SmartScreen prompt. The one-line installs download with tools that set no such mark.
  - Signing and notarisation are post-1.0 options. There is no Homebrew tap: casks need signing, third-party
    taps need `brew trust`, and a tap needs a token beyond `GITHUB_TOKEN`.
- **Licensing** (L-1, L-4, L-5, B-11).
  - Code is MIT.
  - `NOTICE.md` lists:
    - the film quotations, excluded from the MIT grant, with a non-affiliation disclaimer;
    - the two abs0 items, with its BSD-2 notice, which covers the transcription; the text stays film text;
    - the CC BY-SA 4.0 credit for the map fragments in early drafts (SL-6);
    - a contact for rights holders (SL-8), with AGENTS.md's takedown runbook behind it;
    - the brother's licence once granted.
  - `THIRD_PARTY_NOTICES.txt` is generated by `go run ./internal/tools/notices` from the **union** of
    `go list -deps` over the six release targets (`CGO_ENABLED=0`), with each module's `LICENSE*`, `LICENCE*` or
    `COPYING*` file and Go's own licence, so the output does not depend on the host (SL-1). It is committed,
    embedded in the binary for `wopr --licenses` (through the root package, `legal.go`), checked up-to-date on
    Linux in CI and weekly, and shipped in every archive.
  - "WarGames" is used only nominatively, in the README. There are no film stills.
  - "WOPR" is a registered US mark held by an unrelated company (L-6). It is recorded as a low risk in §16.

---

## 13. Documentation

- **README.md** (basic in M0, completed in M5):
  - what this is, and a link to the docs site (below);
  - install, in a few lines: a link to the docs site's one-line installs, the latest release, `go install
    …@latest`, a note that the `.deb` and `.rpm` install the manual page, and a link to the verifying section
    (§12);
  - quick start (`Joshua`, `LOGOFF`);
  - inside the shell: commands, keys, and "any key skips; what you type is kept";
  - flags; themes; the games (with the `Planned` ones marked as coming); movie mode; accessibility (M5);
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
  - change notes (how to add one, the style, when `Changelog: none` applies) and how to release;
  - the milestone checklist: dependency updates, runner labels, re-enabling schedules, `prek update`.

  `CLAUDE.md` contains exactly `@AGENTS.md`.
- **Change notes and `CHANGELOG.md`** (owner decision 2026-10-07, after v0.2.0). Every change a player could
  notice adds a note to `.changes/unreleased/` with changie (pinned in `tools/release/go.mod`; config in
  `.changie.yaml`). The style is the owner's: plain words at an 8th-grade reading level, what changed for
  the player and not why, and a deep fix described only by what the player saw ("Fixed a crash in some
  cases"); one line of at most 76 characters, so that it fits a Debian changelog line. A release pull
  request batches the notes into `.changes/vX.Y.Z.md` and merges `CHANGELOG.md`, which is never edited by
  hand; `release.yml` turns the same files into the GitHub Release notes and the package changelogs
  (§11.3). v0.1.0 and v0.2.0 were written afterwards from their pull requests, in the same style. The
  owner's part of a release is to merge that pull request and push the tag (AGENTS.md, "Releasing").
- **Docs site** (owner decision 2026-10-07): <https://ghostofgoes.github.io/WOPR/>, built from `site/` with
  Hugo (the standard edition, pinned in `tools/docs/go.mod`; no Node, no Sass) and the Hextra theme (a Hugo
  module pinned in `site/go.mod`, MIT; credited in NOTICE.md and on the site's credits page). Pages: home,
  quickstart, installation, usage (with accessibility and troubleshooting under it), games (an index and one
  page per game), movie scenes, contributing with the code of conduct, changelog, and credits.
  - **Install in one line** (owner decision 2026-10-08). Quickstart and Installation show the same tabs:
    Windows, macOS, Linux, Linux (apt), Linux (RPM) and Go, each one line for a person who has never used a
    terminal, with only the tools each system installs by default (the Go tab needs Go). The lines download
    the latest release's file and put the program on the `PATH`; all but Go's start it. The text is one Markdown file per tab in
    `site/assets/install/`, drawn by the `install-tabs` shortcode. Verifying an attestation is the
    Installation page's last section, for those who want it; the README links there and to the guide.
  - **One source for everything.** Each game's page is built from `site/data/games/<slug>.json` by a content
    adapter (`site/content/games/_content.gotmpl`), and the manual page (`docs/man/wopr.6`) is generated from
    the same files. `CHANGELOG.md`, `CODE_OF_CONDUCT.md`, `NOTICE.md` and AGENTS.md's Commands and Pull
    requests sections are mounted and rendered, not copied; so are `docs/screenshots/` and the manual page.
    The download commands name the latest release (below), and the `.deb` and `.rpm` instructions appear once
    that release has packages (any release after v0.3.0, which also has `wopr.6` in its archives; the
    `if-packages` shortcode). Tests in `internal/cli` check that the usage page lists every option and
    environment variable and the movie page every scene.
  - **Strict build.** `--panicOnWarning` (deprecations included) and `--printPathWarnings`; an internal link
    to no page or file, a missing screenshot or a game file without its required fields is an error. CI's
    `docs` job builds it on every push (§11.2).
  - **Deploy from `main`** (owner decision 2026-10-07). `docs.yml` runs on every push to `main` and by hand
    (`workflow_dispatch`), and `release.yml` starts it after publishing a release (§11.3). It builds the site
    from `main` and deploys it to GitHub Pages (`actions/configure-pages`, `actions/upload-pages-artifact`,
    `actions/deploy-pages`), so a change to the site goes live when its pull request merges. The download
    commands name the latest *published* release, which `docs.yml` looks up with the GitHub API and passes to
    Hugo as `HUGO_PARAMS_LATESTRELEASE` (review of 2026-10-07: a release pull request puts its version in
    `CHANGELOG.md` minutes before the release exists, so a site that read the version from there named a
    download that was not there yet); without it, as in CI's builds and a preview, the version comes from
    `CHANGELOG.md`. The site can describe an option before a release has it, as most projects' docs do. The
    build job has `contents: read` and `pages: read`, the deploy job `pages: write` and `id-token: write`, in
    the `github-pages` environment; the deploy job alone is in concurrency group `pages`, never cancelling a
    deployment in progress, so a run on another branch cannot replace one from `main` that is waiting; no
    build cache. Pages must be set to deploy from GitHub Actions (AGENTS.md, repository settings).
  - **Third-party files, vendored** (review of 2026-10-07: fetching them from jsDelivr at build time made
    `ci-ok`, and so every release, depend on a CDN, and nothing pinned the bytes it served). Hextra's
    search runs on FlexSearch 0.8.143 (Apache-2.0) and its lightbox, which shows a screenshot full size, on
    PhotoSwipe 5.4.4 (MIT). `site/assets/vendor/` holds their files and each package's `LICENSE`, byte for
    byte as the npm tarballs publish them (`.gitattributes` keeps git and prek's fixers from changing
    them); `site/hugo.yaml` names the files (`params.gallery.js`, `lightboxJs`, `css`;
    `params.search.flexsearch.js`) and records each tarball's integrity hash. The credits page shows both
    licences from those files. The theme's own stylesheet and scripts come from the module. The build
    fetches nothing but Go modules, and no page loads anything from another site.
  - **Game pages' layout.** The summary, how to start the game and the other names it answers to, the
    screenshots (one set in the lightbox), how to play, the controls, every tip, and a link back to the
    games index, which lists the games in `LIST GAMES` order with tic-tac-toe last.
- **Game pages** (`site/data/games/<slug>.json`, one per game, named by the slug `wopr --games` shows):
  summary, how to play, controls, tips and captioned screenshots (`site/static/img/games/`), written from
  the code. The docs site's game pages and the manual page are generated from them; nothing else repeats
  them.
- **Manual page** `docs/man/wopr.6`, section 6, generated by `internal/tools/manpage` from the flag table
  (`cli.Flags`), the catalog, the game pages and the movie's scenes, and checked in CI with `-check`.
  It is dated by the newest `.changes/vX.Y.Z.md`, never by the clock. It ships at the root of the Linux
  and macOS archives, and in the `.deb` and `.rpm` as `/usr/share/man/man6/wopr.6.gz`, so `man wopr` works
  once a package is installed; `mandoc -T lint -W all` and `groff -man -ww` pass it clean. What ships is
  `pkgdocs`'s copy, whose header names the version being built.
- **Screens**: Appendix C holds the target mockups. The real screens are goldens, kept current by the tests:
  `internal/ui/testdata/gtw_screens.golden` and `card_screens.golden` (Hearts, Gin Rummy, Bridge at 80×24).
- **This plan** is updated at milestone boundaries. Superseded text is deleted.

---

## 14. Verification (per release)

1. `ci-ok` is green on the tagged commit. The release pipeline's `verify`, `build`, `smoke` and `repro` jobs pass.
   The six archives each contain the four notice files and are within the size budget; the four Linux and macOS
   archives also hold `wopr.6`. The four Linux packages pass `stage`'s checks, and the smoke jobs install,
   run and remove them on Ubuntu (`.deb`) and Fedora (`.rpm`).
2. `wopr -v`, `-h`, `-g`, `-L` and their long forms print plain text and exit 0 without a terminal.
   `wopr --games | head -1` exits 0 on every OS.
3. `wopr`:
   1. LOGON → `FALKEN'S MAZE` → `IDENTIFICATION NOT RECOGNIZED` → re-dial;
   2. two more wrong names → the hint;
   3. `Joshua` → the greeting, answering its questions (`Hello.`, `I'm fine. How are you?`, `People sometimes
      make mistakes.`) as in Appendix C;
   4. `Love to. How about Global Thermonuclear War?` → the chess offer;
   5. `Later. Let's play Global Thermonuclear War.` → `FINE.` and GTW starts;
   6. Esc, Esc → back in the Shell with a remark;
   7. `list games` → `7` → the chess board. WOPR shows `PROCESSING` and Ctrl+C stays responsive;
   8. `LOGOFF` → exit 0.
4. The film path under `-s 1 -i` with scripted input (the `testkit` golden `film_path`): GTW → climax → NORAD
   notices → `TIC-TAC-TOE` → 0 players → montage → `HELLO` → `A STRANGE GAME…` → chess offer armed.
   `wopr -p 15`, `wopr -p "global thermonuclear war"` and `wopr gtw` resolve to the same game.
   `wopr -p theaterwide` and `wopr gtw chess` exit 2.
5. `go test -race ./...` is green on the three test OSes. Goldens are stable. `archtest` and the lint
   self-test are green.
6. The e2e test passes on all six smoke runners. A manual pass in Windows Terminal and conhost checks colours,
   cursor, resize, Ctrl+C restore and SmartScreen.
7. On a tag, `gh attestation verify` with the flags in §12 succeeds for each file, and the release is
   immutable.
8. From M6: `wopr --movie` opens the scene menu and every scene plays; `wopr -m 2 -i` plays from scene 2 to
   the end of the list, pausing between pages, and exits 0; `wopr -m 2 -o` plays scene 2 alone and exits 0;
   `wopr --scenes` lists them. The movie consistency tests are green.

---

## 15. Milestones

| M | Deliverable | Exit criteria | Release |
|---|---|---|---|
| 0 | **Scaffold** (done). go.mod, `.gitattributes`/`.gitignore`, LICENSE/NOTICE/SECURITY/CONTRIBUTING/CODE_OF_CONDUCT, `legal.go`, `version`, `proto` (types), `games` (types + catalog with 16 `Planned` entries), `cli`, `theme`, `ui` hello-world, `archtest` + lint fixture, `golden`, `tools/*`, `prek.toml`, `.golangci.yml`, `.goreleaser.yaml`, `ci.yml`/`smoke.yml`/`release.yml`/`scheduled.yml`, README (basic), **AGENTS.md** + `CLAUDE.md`, the M0 e2e cases. | `ci-ok` green with all six smoke runners; the lint fixture fires every rule; a **snapshot** dry run of `release.yml` passes `build`/`smoke`/`repro` (CR-1); Appendix A deleted (done in v2.1). M1 may start once `ci-ok` is green. | — |
| 0b | **Owner settings** (IM-3): the repository settings in AGENTS.md; the gitleaks canary branch turned red for a finding. | Settings applied; canary done. These gate **v0.1.0**, not M1. | — |
| 1 | **Console, persona and protocol.** `proto/host` runner, `games/testkit`, clock, typewriter, line editor, `Sanitize*`, scrollback, canvas renderer, front panel, `prompt` (menus, yes/no, clauses; moved from M3, P-4), the ui adapter and its synchronous driver, persona (session, LOGON table, greeting scene, commands, intent, offers, scripted brain, lines with provenance), themes with ANSI/ASCII fallbacks and contrast tests, too-small handling, `gamestest` stub exercising every Output, goldens, fuzz, the M1 e2e cases. 80×24 mockups for chess and the GTW phases (Appendix C). | Every M1 golden flow and the e2e cases marked M0–M1 are green on all runners. | — |
| 2 | **Film set pieces.** `games/ai`, `board/`, tic-tac-toe, checkers, chess, **GTW** (§6.2, with the climax table), **`games/ending`** (reusing tic-tac-toe; owns the launch-code display), the internal `ending` registry entry, the persona's remark for an abandoned war, random session seeds and the debug log, original GTW map art, abs0's 157 scenario names verbatim. **Built**; QA on all OSes remains. | The `film_path` and climax goldens; quality tests; manual QA list. | **v0.1.0** |
| 3 | **Card games.** `cards/` + `trick.go`; Black Jack, Poker, Gin Rummy, Hearts, **Bridge (minimal, last)**; Hearts mockup (the `card_screens` golden). **Built**; QA on all OSes remains. | Definition of done per game. | v0.2.0 |
| 4 | **Sims and maze.** Sim engine spec → engine → four scenarios + two bespoke sims; Falken's Maze. **Built**; QA on all OSes remains. | Definition of done per game. | v0.3.0 |
| 5 | **Polish.** Film viewing pass (every `reconstructed` line becomes `film` or is corrected; the montage names verified; the three conflicts in §2.3 settled; the status burst's wording; the movie scene scripts fixed); **GTW's turn-based DEFCON exchange** (§6.2, decision 24; **built**); README completed with screenshots (**done** 2026-10-07: six in `docs/screenshots/`, recorded from the program in an 80×24 terminal); accessibility pass (**done**: §4.5 records what was checked, what changed and what is left for a screen-reader user); dependency and runner checklist (last run 2026-10-07, to be run again at the v1.0.0 boundary: only indirect modules moved, `ultraviolet` to its 2026-10-01 commit, `go-runewidth` v0.0.30, `xo/terminfo` v1.2.0, `x/sync` v0.23.0 and `x/sys` v0.48.0, with goldens byte-identical and linux/amd64 6,119,584 → 6,140,064 bytes stripped; Go 1.27.1, the direct modules, the four tools, the prek hooks, the action SHAs and the runner labels (§11.4) were already current; `scheduled.yml` is active). | No `reconstructed` tags remain; the turn-based exchange cannot be won and its film path stays short; the checklist run again just before tagging v1.0.0, not only the 2026-10-07 run. | **v1.0.0** |
| 6 | **Movie mode** (§7): `-m/--movie`, the host hooks, director, scenes, scene menu, `-S/--scenes` (the owner's request for a scene list), `-o/--only` (the owner's request to play one scene and stop), consistency tests, e2e cases. **Built**; QA on all OSes remains. | All scenes play; consistency test green; size re-checked (linux/amd64, stripped: 5,922,976 bytes before M6, 6,119,584 after; every target passes the gate). | v1.1.0 |
| 7 (opt) | **LLM brain** (§4.7): opt-in, `net/http`, hardened client, effects allowlist, scripted fallback. | Fuzzed reply parser; size gate; offline behaviour unchanged. | v1.2.0 |

Effort is not estimated per game (P-3). Each milestone's PR description records time spent, which informs the
next one.

**Third-party permission gate** (L-3, L-5). Lines adapted from the brother's prompt carry the `prompt` tag and
are not committed at all until he gives written permission and a licence. The gate blocks those lines only, not
M1.

---

## 16. Risks

| # | Risk | Likelihood / impact | Mitigation |
|---|---|---|---|
| RK-1 | Film text is MGM's expression. Movie mode (M6) ships the film's complete WOPR terminal script, much of which the persona already contains (SL-8). | Medium / medium | On-screen terminal text only, no spoken-only dialogue, stills or audio; provenance per line; `NOTICE.md` exclusion, disclaimer and rights-holder contact; nominative use of "WarGames"; a takedown runbook in AGENTS.md (delete releases, patch, `retract`; copies remain in git history and the module mirror). The owner accepts the residual risk. |
| RK-2 | "WOPR" is a registered US mark (Frontier Technology, class 42). | Low / medium | Different class (SaaS vs a free game); no logo imitation; revisit if contacted. Not legal advice. |
| RK-3 | Bubble Tea v2 patch churn (v2.0.10 in under a year). | Medium / low | Pin; read release notes on each update; goldens catch rendering changes. |
| RK-4 | Hosted runner labels retire. | High / low | Pinned labels; checklist at each milestone; the oldest Linux claim rests on Go's kernel floor, not on a runner (§1). |
| RK-5 | Unsigned binaries trigger Gatekeeper and SmartScreen friction. | High / low | Attestations plus verify-first docs; signing post-1.0. |
| RK-6 | No Dependabot means updates lag. | Medium / medium | Weekly scheduled report issue; govulncheck in CI; monthly manual cadence; 14-day response rule. |
| RK-7 | 16 games is a large scope. | High / medium | A release per milestone; `Planned` games stay listed and decline in character. |
| RK-8 | The chess library fork goes stale. | Low / low | Small API surface used (rules, SAN/UCI); search is our own. |
| RK-9 | Tool dependency graphs conflict (golangci-lint's broke gitleaks's build, SL-2). | Medium / low | One tool module per conflicting tool; the CI jobs build each tool on every run, so a break shows at once. |

Risk ids are `RK-n`, so they do not collide with the requirement ids (R1…R13) or the v1 review's `R-n` findings
(IM-12).

---

## Appendix B — Canonical screen text (provenance-tagged)

Since M1, `internal/wopr/lines.go` (and later `internal/assets/`) is authoritative for the text it holds; this
appendix keeps the rest until it lands. Tags: **F** = `film` once confirmed, currently `reconstructed`.
**A** = `third-party:abs0/wargames@010ed92:wargames.sh` (film text as transcribed by abs0, credited). **O** =
`original`.

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

**Backdoor header** (A, byte-identical to abs0's transcription), followed by a status burst (F: the film's
phrases in our own two-column layout, not an abs0 item; SL-5) of `(311) 936-2364`, `SYSPROC FUNCT READY` /
`ALT NET READY` and `CPU AUTH RY-345-AX3` / `SYSCOMP STATUS ALL PORTS ACTIVE`, then a clear. The transcriptions
disagree on `STATUS:` versus `STATUS`; M5 settles it.

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

**Call-back, at David's home** (F). WOPR phones David; the screen also shows `GAME TIME ELAPSED` and
`ESTIMATED TIME REMAINING` timers, whose values M5 takes from the film (RF-9). Until then movie mode shows the
hours and minutes both transcriptions give (`31 HRS 12 MIN`, `52 HRS 17 MIN`, tagged `reconstructed`) with
seconds of its own that tick (`original`); abs0 alone gives seconds, which are not one of its two credited
items. David's typed lines, such as `What is the primary goal?`, are
mixed case. Movie mode (§7) puts `Is this a game or is it real?` and `WHAT'S THE DIFFERENCE?` in the NORAD
session, as the transcriptions do:

```text
I'M SORRY TO HEAR THAT, PROFESSOR.
YESTERDAY'S GAME WAS INTERRUPTED.
ALTHOUGH PRIMARY GOAL HAS NOT YET BEEN ACHIEVED, SOLUTION IS NEAR.
YOU SHOULD KNOW PROFESSOR. YOU PROGRAMMED ME.
TO WIN THE GAME.
WHAT'S THE DIFFERENCE?
```

**The NORAD terminal session** (F), later, after `Joshua` and `Are you still playing the game?`. `28 HOURS`
follows the subtitles and the film's timeline; abs0 has `61` (§2.3). The split of lines between the two scenes is
provisional until M5; movie mode's scenes record the current one:

```text
OF COURSE. I SHOULD REACH DEFCON 1 AND LAUNCH MY MISSILES IN 28 HOURS.
WOULD YOU LIKE TO SEE SOME PROJECTED KILL RATIOS?
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

The values are derived from the exchange (O): BOMBERS and ICBM count what was destroyed, the other rows scale
with the warheads that landed (§6.2).

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

**Montage scenario names** (decision 18; verified or replaced in M5). 40 of the 45 names below match abs0's
157-entry list and are tagged A. Five are not abs0's text and are tagged F (`reconstructed`) until M5 (RF-9):
`NATO LIGHT` (abs0: `NATO ALERT`), and our respellings `BURMESE THEATERWIDE`, `AUSTRALIAN MANEUVER`,
`ICELAND INCIDENT` and `MALAYSIAN MANEUVER` (abs0: `THEATERWIOE`, `AUSTRAILIAN`, `ICELANDIC INCIDENT`, <!-- codespell:ignore -->
`MAYLASIAN`). The 45 are a sparse subsequence of abs0's list, so completing it in M2 means re-deriving it in
abs0's order, not appending: `U.S. FIRST STRIKE`, `USSR FIRST STRIKE`,
`NATO / WARSAW PACT`, `FAR EAST STRATEGY`, `US USSR ESCALATION`, `MIDDLE EAST WAR`, `USSR CHINA ATTACK`,
`INDIA PAKISTAN WAR`, `MEDITERRANEAN WAR`, `HONGKONG VARIANT`, `SEATO DECAPITATING`, `CUBAN PROVOCATION`,
`ATLANTIC HEAVY`, `CUBAN PARAMILITARY`, `NICARAGUAN PREEMPTIVE`, `PACIFIC TERRITORIAL`, `BURMESE THEATERWIDE`,
`TURKISH DECOY`, `NATO LIGHT`, `ARGENTINA ESCALATION`, `ICELAND MAXIMUM`, `ARABIAN THEATERWIDE`,
`U.S. SUBVERSION`, `AUSTRALIAN MANEUVER`, `SUDAN SURPRISE`, `NATO TERRITORIAL`, `ZAIRE ALLIANCE`,
`ICELAND INCIDENT`, `ENGLISH ESCALATION`, `MIDDLE EAST HEAVY`, `MEXICAN TAKEOVER`, `CZECH OPTION`,
`FRENCH ALLIANCE`, `ARABIAN CLANDESTINE`, `GABON REBELLION`, `SEATO TAKEOVER`, `HAWAIIAN ESCALATION`,
`TAIWAN DOMESTIC`, `MONGOLIAN THRUST`, `POLISH DECOY`, `ALASKAN DISCRETIONARY`, `CANADIAN THRUST`,
`S.AFRICAN DOMESTIC`, `TUNISIAN INCIDENT`, `MALAYSIAN MANEUVER`.

**Original lines** (O): the LOGON hint, `WHICH GAME?`, `** GAME ROUTINE NOT AVAILABLE **`,
`** REQUEST CANCELLED **`, `** PRESS ESC AGAIN TO END GAME **`, the `HELP` command list (including
`<NUMBER>  PICK FROM THE LIST JUST SHOWN`), the GTW climax hints, the climax tic-tac-toe's WOPR-win line, the
remark after an abandoned war, `PROCESSING`, and every fallback reply. GTW's exchange (M5), with `#` filled in
order:

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

Each screen is exactly 24 rows of at most 80 columns, drawn with the geometry of §4.3 (a script checked the
counts). `█` is the cursor.

**Backdoor greeting** (`imsai`). WOPR is upper case and revealed at modem speed. The user's input is echoed in
mixed case. At the film's spacing the exchange may scroll; that is fine:

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

**GTW side choice** (`norad`, `LayoutConsole`, front panel shown). The map and its labels are printed as
console text before the question, as in the film (RF-3). The map is a placeholder until the original art lands
in M2:

```text
   +---------------------------------------------------------+
   |                                                         |
   |   [ map: North America left, USSR right; original       |
   |     line-segment ASCII art, 57x7, drawn in M2 ]         |
   |                                                         |
   |                                                         |
   |                                                         |
   |                                                         |
   +---------------------------------------------------------+
           UNITED STATES                  SOVIET UNION

WHICH SIDE DO YOU WANT?

  1.    UNITED STATES
  2.    SOVIET UNION

PLEASE CHOOSE ONE: █






 W.O.P.R.   LINE 1200 BAUD   ONLINE                             * . * . * . * .
```

**GTW big board** (`norad`, `LayoutFull`, front panel shown): the view takes `H' − 4` = 19 rows, then the 3-row
strip, the input row and the panel (AR-11). This is the built screen at the strike 2 prompt (the
`gtw_screens` golden: USSR, Las Vegas and Seattle, after the first strike). Trajectory values are illustrative.
Outgoing tracks draw `+`, incoming `*`, and an impact a reversed `X`, drawn over every track; earlier strikes keep only their impacts.
The current DEFCON level is pointed at (`>`) and reversed, shown here as `[4]`:

```text
+---------------------- GLOBAL THERMONUCLEAR WAR -----------------------+ DEFCON
|               |:/''-\:::::::|   + '-'             .___.               |  +---+
|        .___.  |:\.  '\::++++++++ ++++++++ .. ...__/:::\__. ._.        |  | 5 |
|._______/:::\__/:/\_. ++++:*****************++/\/:::::::::\_/:\______. |  +---+
||::::::::::::::/-'|+++*******      .*************:::::::::::::::::::/' | >|[4]|
|'\:/-\::::::::/' ++****'-'        .//********:******:::::::::/--\/--'  |  +---+
| '-' '-\::::::\+***:\_.         ..|:\/**:****X::::*X*X::::::/'  ''     |  | 3 |
|       '+::::****:::/-'         |\/::::X:::**:::::::::::::::|          |  +---+
|       ++X::**:::::/'          ./:/\:::/-\/\**:::::::::::::/'          |  | 2 |
|       ++++:::::::/'           |:/\/--\\_/\/:*X:::::::::/--'           |  +---+
|         'X::::::/'            |:\/\__/:::::::::::::::::\.             |  | 1 |
|          '-\:/--'            ./::::::::::::/-\:::::::::/'             |  +---+
|            '\\_.             |:::::::::::::| '\:/--\:/-'              |
|             '--'             |:::::::::::/-'  '\|  '\|                |
+------- UNITED STATES ----------------------- SOVIET UNION ------------+
TRAJECTORY HEADING   TRAJECTORY HEADING          FORCES   ICBM  SLBM   BMB   AIR
A-SS20-A 932 534     C-SSN8-A 319 667            USSR      500   450   200     0
       B 487 038            B 558 572            US        737   562     0   375
ORDERS: PERCENT OF ICBM SLBM BOMBERS, ALL, HOLD, AUTO, HELP.
FIRST STRIKE LAUNCHED. ICBM 250  SLBM 150.
ENEMY LAUNCH DETECTED. ICBM 313  SLBM 188  BOMBERS 375. DEFCON 4.
LOST ON THE GROUND: USSR 350  US 200. WARHEADS ON CITIES: USSR 188  US 150.
STRIKE 2 OF 3 [50 50 100]: █
 W.O.P.R.   LINE 1200 BAUD   ONLINE                             * . * . * . * .
```

**Chess** (`imsai`, `LayoutPanel`, `PanelRows` 12, no front panel): 12 rows of board, then the 12-row console
strip. The board is gridless, per the no-chrome rule. WOPR is thinking: the `PROCESSING` row is the last line
of text, with the steady cursor after it; anything typed now would appear on the row below:

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

PROCESSING ..█





```
