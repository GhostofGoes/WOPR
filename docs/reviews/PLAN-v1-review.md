# Adversarial review: WOPR plan v1

_Reviewed: `docs/PLAN.md` at commit `ada63a0` ("Add WOPR architecture and scope plan"). Review date 2026-10-06._

This review attacks the plan on purpose. It looks for claims that are false, designs that will break under
real use, ordering mistakes that cause rework, and places where the plan contradicts itself. Every factual
claim it could check was checked against the module proxy, the tool's own source or docs, or a local
experiment. The **Verification log** at the end lists those checks. The updated plan (`docs/PLAN.md` v2)
answers each finding; `PLAN-v2-review.md` reviews that version.

**Severity scale**

| Level | Meaning |
|---|---|
| **Blocker** | Fails as written, or forces a rewrite later if not fixed before the code that depends on it. |
| **Major** | Real risk to correctness, security, schedule or maintenance. Fix in the plan now. |
| **Minor** | Worth fixing; cheap. |
| **Nit** | Wording, consistency, polish. |

---

## Verdict

The plan is far better than most hobby-project plans. Its strongest parts are the import rule that keeps
games and the persona free of Bubble Tea, the single clock, deterministic seeds for golden tests, and the
public-repo hygiene. The CLI surface and size budget are concrete and testable.

It is not ready to implement as written. Five problems would surface as bugs or rework in M0–M2:

1. **The game ⇄ host contract is wrong for the flagship games.** `Kind` decides both layout and input
   routing. Chess (typed `e2e4`) and Global Thermonuclear War (typed target cities) are `Board`/`Fullscreen`,
   which route raw keys to the game. As written, each would have to implement its own line editor. Games
   also have no way to colour their output without importing Lip Gloss, but the GTW big board needs
   red/yellow tracks and DEFCON colours. (A-1, A-2)
2. **Blocking AI on the UI goroutine.** `Game.Handle` is synchronous. A chess search at 3–4 ply plus
   quiescence, run inside `Update`, freezes rendering and Ctrl+C until it returns. (A-3)
3. **A data race designed into the brain call.** The UI runs `Brain.Reply(ctx, *Session, …)` in a `tea.Cmd`
   (a separate goroutine) and hands it the live session, which `Update` keeps mutating. The scripted brain
   writes `s.Said` and reads `s.Rand` from that goroutine. Replies can also arrive out of order or be
   silently dropped by the sequence guard. (A-4)
4. **The lint config does not run, and two checks never fire.** The verification log shows the `prek.toml`
   errors. The `forbidigo` pattern ends in `\(`, which never matches, because forbidigo matches identifiers
   and not call text. In CI the gitleaks hook only scans *staged* changes, so it checks nothing there. (T-1,
   T-2, S-1)
5. **Milestone ordering and scope.** GTW is in M2 but depends on the `sim` engine from M4. Nothing ships
   until all 16 games are done (M5), and Bridge, Gin Rummy and the five sims are budgeted at an unrealistic
   ~600 lines each. (P-1, P-2)

The rest are Major/Minor findings below. None of them argues for a different stack: Bubble Tea v2 + Lip
Gloss v2 remains the right choice.

---

## 1. Requirements and traceability

**R-1 (Major) — Stated requirements and decisions disagree; the provenance header is wrong.**
The Context section says "Linux + Windows, **< 10 MB**". Decision 5 adds macOS. Decision 10 relaxes the
limit to "target ≤ 10 MB, hard limit 15 MB". The decisions header says "1–4 confirmed by the user in Q&A;
5–7 are routine defaults", but the table has 12 rows, and row 5 is labelled "(requirement)". A reader
cannot tell which rows the owner confirmed and which the plan's author chose. That matters most for the
size limit: a user requirement of "< 10 MB" was turned into a warning.
*Fix:* list requirements once, with a source for each (user / derived / default), and record any change in
the decision log. Ask whether 15 MB is acceptable as a hard limit (see Questions).

**R-2 (Minor) — Requirements with no acceptance criteria.** "Persona faithful to the film" and "every game
the film's WOPR offered" are not testable as written. What counts as done for Bridge, or for Desert
Warfare? *Fix:* give each game a short definition of done: rules implemented, AI plays legally, ends with
a verdict, and a testkit transcript.

**R-3 (Nit) — The file the request names does not exist.** The plan lives at `docs/PLAN.md`; there is no
`plan.md`. Keep a single path and link it from the README.

---

## 2. Architecture

**A-1 (Blocker) — `Kind` couples layout to input mode, and the flagship games need both.**
`Update` routes keys "to game (Board/Fullscreen) or input line". But:

- Chess is `Board`, yet its input is typed algebraic (`e2e4`, `Nf3`).
- GTW is `Fullscreen`, yet it asks `PLEASE LIST PRIMARY TARGETS BY CITY AND/OR COUNTRY NAME:` and
  `PLEASE CHOOSE ONE:`.
- Gin Rummy and Hearts are "Teletype + hand view" or "+ trick view", which is neither Kind.

As written, chess and GTW each rebuild a line editor out of `KeyEvent`s, which duplicates history, paste
and the skip policy. *Fix:* split the two axes. A static `Layout` (Console, Panel, Fullscreen) says where
`View` goes. The input mode is dynamic, set by output. `Prompt{Text}` means "line input: deliver a
`LineEvent`". A new `AwaitKeys{}` means "deliver `KeyEvent`s" (maze arrows, board cursor). The host owns
one line editor for every game.

**A-2 (Blocker) — Games cannot express colour without breaking the import rule.**
`View(w, h) string` must either return raw ANSI, which means games import Lip Gloss and the theme and
golden tests see escape codes, or plain text, which leaves no red tracks, DEFCON colours or blinking subs.
The plan wants both "games never import Bubble Tea" and a coloured big board, and does not say how.
*Fix:* games draw into a small `games/canvas` grid of runes with **semantic** style tags (`Text`,
`Bright`, `Dim`, `Accent`, `Incoming`, `Outgoing`, `Defcon5…1`, `Blink`). `ui` maps tags to the active
theme. Games stay theme-agnostic, golden tests can assert on the tags, and themes can change without
touching games.

**A-3 (Blocker) — Synchronous `Handle` blocks the event loop during AI search.**
`Update` must return quickly. Bubble Tea processes Ctrl+C as a `KeyPressMsg` in `Update` (see the
verification log), so while chess, checkers (depth ~6) or Bridge play analysis runs inside `Handle`,
nothing renders and Ctrl+C does nothing. The plan's `notnil/chess` dependency makes this worse: its move
generator allocates heavily (see the verification log for nodes/s). *Fix:* add an output
`Think{Fn func(ctx) Event, Budget time.Duration}`. The host runs `Fn` in a `tea.Cmd` with a cancellable
context, shows WOPR "thinking" (blinking cursor, or front-panel lights), and delivers the resulting
`Event` back into `Handle`. AIs use iterative deepening under a time budget, so play speed is the same on
every machine. Esc cancels the context.

**A-4 (Blocker) — `Brain.Reply(ctx, *Session, …)` from a `tea.Cmd` is a data race and an ordering bug.**

- *Race:* Cmds run on their own goroutines while `Update` keeps running (see the verification log). The
  scripted brain's `Once` rules write `s.Said[ID]`, and `Pick` reads `s.Rand`, which is not
  goroutine-safe. The golden driver runs cmds synchronously, so `-race` in tests will never catch it. It
  will only show up in real use.
- *Ordering:* scenes and commands answer synchronously while the brain answers asynchronously. If the user
  types a second line before the first reply lands, the transcript comes out of order. The "sequence
  guard" then discards the first reply, and the user's first line gets no answer at all.

*Fix:* `Reply(ctx, Snapshot, input) (Reply, error)`, where `Snapshot` is a value copy and the brain gets
its own RNG stream seeded from `(seed, turn)`. `Reply` carries an `Effects` list (mark-said, set-flag,
start-game) that `Update` applies. While a reply is pending, the input line is locked (film-faithful:
the terminal is busy), and Esc cancels the context. The sequence guard then only has to handle
cancellation.

**A-5 (Major) — Two state machines own "phase".** `ui/app.go` has phases Connect→…→Ending, and
`wopr/session.go` also holds `phase`. The scripted rule table gates on `Phase`. Two copies will drift.
*Fix:* the persona (`wopr.Session`) owns the conversation phase. `ui` owns only the presentation mode
(which screen is drawn) and derives it from session state plus the active game.

**A-6 (Major) — The import rule is stated but not enforced, and it is incomplete.** `wopr` needs the game
registry (`LIST GAMES`, name detection), so `wopr → games` must be allowed. That edge is not in the stated
direction `cmd → ui → (wopr, games, theme)`. Nothing checks the rule, apart from AGENTS.md, which the plan
writes last. *Fix:* write the full DAG (`cmd → cli, ui, version`; `ui → wopr, games, theme`;
`wopr → games`; `games/* → games, games/{ai,cards,board,canvas,prompt}`; no package outside `ui` imports
`charm.land/*`) and enforce it with `depguard` rules in `.golangci.yml` from M0.

**A-7 (Major) — Game-name detection "anywhere in the text" will misfire.** "I don't want to play chess",
"the Golden Gate **bridge**", "my **hearts** not in it" and "**poker** face" all launch games. It also
pre-empts a future LLM brain, which is the plan's own reason for putting scenes and commands first.
*Fix:* start a game only on explicit intent. That means a numbered selection, `PLAY <name>`,
`LET'S PLAY <name>`, `HOW ABOUT <name>`, or input that is exactly a game name or alias. Match on word
boundaries after normalization. Keep the film's one-shot GTW → "WOULDN'T YOU PREFER A GOOD GAME OF
CHESS?" exchange as a scene.

**A-8 (Minor) — `KeyEvent.Key` is "tea's `String()`".** This leaks Bubble Tea's key naming into games,
so a Bubble Tea rename (`" "` vs `"space"`) breaks games silently. *Fix:* `games` defines its own small
key enum (`Up`, `Down`, `Left`, `Right`, `Enter`, `Rune`); `gamehost` maps to it.

**A-9 (Minor) — `Animate{Every}` vs. the single clock is underspecified.** The clock ticks at the fastest
consumer's rate (typewriter ~33 ms), so does a game asking `Every: 500ms` get `TickEvent`s every 33 ms?
*Fix:* the host accumulates `dt` per consumer and delivers `TickEvent` at the requested cadence. The
clock's interval is the minimum over active consumers.

**A-10 (Minor) — Script lines in `lines/*.txt` with string keys.** These need a parser, a key convention
and runtime lookups that fail late, and the plan already splits scenes between Go and text. Unless
non-programmers will edit the lines, plain Go string constants in `lines.go` are simpler: checked at
compile time, `gofmt`'d, and still skippable by codespell by path. If `.txt` stays, add a test that every
key referenced in code exists.

**A-11 (Minor) — RNG type and stability.** `Env.Rand *rand.Rand` does not say whether it is `math/rand`
or `math/rand/v2`. Golden tests depend on byte-identical sequences across Go upgrades. *Fix:* use
`math/rand/v2` with an explicit `rand.NewPCG(seed, stream)` and derive one stream per consumer (persona,
each game, brain). Accept that goldens may need `-update` after a Go upgrade, and say so in AGENTS.md.

**A-12 (Nit) — `Pace` is used in `Say` but never defined.** Same for `Outcome` constants and `Action`
payloads. Define them in the plan, or say they are left to M1.

---

## 3. Terminal UI

**U-1 (Major) — No alternate screen, background or exit-screen policy.** The plan never says whether the
TUI uses the alternate screen. Without it, the 80×24 full-bleed console scrolls the user's shell history,
and the `#000000` theme background only paints cells that were written. *Fix:* use the alt screen
(`View.AltScreen`). Paint the theme background over the full frame. On exit, leave the alt screen and print
one line (`--CONNECTION TERMINATED--`) to the normal screen.

**U-2 (Major) — No sanitisation of text that did not come from the binary.** User input, pasted text
(`PasteMsg`) and a future LLM reply are all echoed into the view. Escape sequences inside them (OSC 52
clipboard writes, OSC 8 links, cursor movement) should never reach the renderer. *Fix:* one
`console.Sanitize` that strips C0 (except `\n`/`\t`), C1 and ESC from any external text. Cap the input line
(e.g. 256 runes) and the paste size. Fuzz-test it.

**U-3 (Minor) — The skip policy throws away typeahead.** "Every `KeyPressMsg` … flushes the whole queue and
is swallowed." A user who types `h` (hit) while WOPR is still printing loses the keystroke, and PgUp during
narration flushes instead of scrolling. *Fix:* the first key flushes. Printable keys are then also
delivered to the input line, and only Enter and Space act purely as "skip". PgUp/PgDn always scroll.

**U-4 (Minor) — TTY detection.** "If stdout is not a terminal, refuse" checks only stdout. Stdin matters
too, and on Windows `mintty` (Git Bash without ConPTY) reports pipes. *Fix:* check both. Make the error
message name the cause and suggest Windows Terminal. Treat `TERM=dumb` as not-a-TTY. Honour `NO_COLOR`.

**U-5 (Minor) — 80×24 is tight for Board layouts.** An ASCII chess board with coordinates takes about 18
rows, which leaves about 5 for the console strip, prompt and status. Mock up chess, Hearts and GTW at
exactly 80×24 in M1, before committing to the layout.

**U-6 (Nit) — Custom input line "~100 lines".** Wide runes (CJK, emoji) need `ansi.StringWidth`, plus
history and paste handling. Budget about 250 lines and fuzz it.

---

## 4. Games and AI

**G-1 (Major) — `notnil/chess`.** Upstream has not released since v1.10.0 (Nov 2024). See the verification
log for archive status and the maintained fork. It is also slow as a search back-end. *Fix:* keep it, or
the maintained fork, for **rules, SAN/UCI parsing and legality**. If the measured nodes/s cannot reach
3–4 ply in about 1 s, use a small bitboard or 0x88 generator for search, cross-checked against the
library in tests (perft). Either way, search runs off the UI goroutine (A-3).

**G-2 (Major) — The five military sims are under-specified, and as written they are five bespoke games.**
Each has one line of description. None has rules, win or lose conditions, or balance. *Fix:* define the
`sim` engine concretely: a hex or square grid or strip map, a unit table, an action set, a combat results
table, events, a turn limit and a kill-ratio generator. Express each sim as **data** (a scenario struct:
map, forces, actions enabled, events, victory rule) plus at most a few hooks. That turns about 5×600 lines
of game code into one engine and five scenario files, and is easier to balance and test.

**G-3 (Major) — Bridge is a project on its own.** A natural-bidding AI plus declarer and defender play is
well beyond 600 lines. The plan already calls it "first to cut". *Fix:* decide now. Either cut it from
v1.0 (it stays on `LIST GAMES`, and WOPR answers in character), or ship a minimal variant: WOPR bids for
all four seats with a simple point-count rule, and the user plays declarer.

**G-4 (Minor) — Tic-tac-toe numbering.** Tic-tac-toe is the 16th registry entry but unlisted. Does
`wopr -p 16` work? Does `--games` show `16`? *Fix:* registry `Number` 0 means unlisted. Resolve by name,
slug or alias only.

**G-5 (Minor) — `stub` game in the registry.** M0 registers a stub, which then appears in `--games` and
can be resolved. Make it test-only (`registry_test.go`), or hidden behind a build tag.

**G-6 (Nit) — Falken's Maze "re-routes walls based on the player's habits".** This needs a sentence on
what counts as a habit (turn bias, for example) and a guarantee that the maze stays solvable (test it).

---

## 5. CLI

**C-1 (Major) — `wopr gtw --instant` silently ignores `--instant`.** Confirmed locally: stdlib `flag`
stops parsing at the first positional argument, so the flags after it end up in `Args()`. The plan
advertises `wopr <game>` as equivalent to `--play`, so users will write exactly this. *Fix:* in
`cli.Parse`, after `fs.Parse`, if one positional remains, take it as the game and re-parse the rest. Or
reject extra positionals with exit 2. Test both orders.

**C-2 (Minor) — Combined short flags (`-is 1`) and `-p=chess`.** Stdlib `flag` rejects `-ip chess` and
`-pchess`. That is fine, but the help text should show the accepted forms, and a test should pin them.

**C-3 (Minor) — Exit-code contract.** Keep 0/1/2/130. Add: `--games` and `--version` exit 0 even when
stdout is closed early (`| head -1`). Ignore `EPIPE` on stdout.

**C-4 (Nit) — `-v` for version.** Many tools use `-v` for verbose. Acceptable, but say it is deliberate so
a later `--verbose` does not collide.

---

## 6. Dependencies and toolchain

**D-1 (Major) — The `go` directive is set to the newest release, not the minimum.** `go 1.27` in `go.mod`
forces every consumer, including `go install …@latest` users and distro packagers, onto Go ≥ 1.27, even
though the stack needs only 1.26 (Bubble Tea v2's floor; see the verification log). The `toolchain`
directive exists to state "build with this" separately. *Fix:* `go 1.26.0` + `toolchain go1.27.1`. CI and
GoReleaser still build with 1.27.1. *Owner decision: see Questions.*

**D-2 (Major) — The future LLM client must not pull in an SDK.** `net/http` and `crypto/tls` already add
several MB (see the size spike), and provider SDKs add much more. *Fix:* state now that M6 uses
`net/http` + `encoding/json` against an OpenAI-compatible and/or Anthropic Messages endpoint, behind a
build tag if size requires, and that the size gate applies.

**D-3 (Minor) — tcell fallback is not a cheap fallback.** Swapping the renderer means rewriting all of
`internal/ui`. That is fine as long as the M0 spike retires the risk. The size spike in the verification
log shows plenty of headroom, so remove tcell from the dependency table and keep a one-line note.

**D-4 (Minor) — Version reporting.** `go install …@v0.1.0` builds without VCS stamping and without
`-X Version`. *Fix:* `version.Version` falls back to `debug.ReadBuildInfo().Main.Version`, then to
`(devel)`.

---

## 7. Build, CI and release

**B-1 (Major) — Three artifact naming schemes, and the smoke test would fail.** `build.sh` writes
`dist/wopr_${os}_${arch}` (for example `wopr_darwin_arm64`). The smoke test runs
`./dist/wopr-macos-arm64`. Releases are `wopr_<v>_darwin_arm64.tar.gz`. As written, the smoke step fails on
macOS, and on the other two runners it fails on `-` vs `_`. *Fix:* one scheme, one producer (B-2).

**B-2 (Major) — CI artifacts are not the release artifacts.** CI builds with `scripts/build.sh` on three
runners. Releases build with GoReleaser on one runner. Two build definitions will drift: ldflags, `-trimpath`
and file names. With `CGO_ENABLED=0`, building on the target OS adds nothing; the binary is the same.
What matters is **running** each binary on its OS and architecture. *Fix:* one `build` job runs
`goreleaser release --snapshot --clean` (the same `.goreleaser.yaml` as releases), then uploads the six
binaries. A `smoke` matrix downloads each one and runs it on a native runner. That includes the **arm64**
targets, on `ubuntu-24.04-arm` and `windows-11-arm` (free for public repos; see the verification log) and
an Intel macOS label while one exists. The size gate runs on the GoReleaser output. What CI tests is
then exactly what ships. *Owner decision: see Questions.*

**B-3 (Major) — Line endings on the Windows runner.** `.gitattributes` forces LF only for `*.go` and
`*.txt`. GitHub's Windows runners check out with `core.autocrlf=true`, so `scripts/*.sh` arrive with CRLF
and Git Bash fails on them (`$'\r': command not found`). Golden files (`*.golden`) arrive with CRLF and
every golden comparison fails on `windows-latest`. *Fix:* `* text=auto eol=lf` globally. Mark binary
fixtures `-text`.

**B-4 (Major) — Release cache poisoning.** `actions/setup-go` caches by default. In a tag-triggered release
workflow, a poisoned cache from a PR run can end up in the release build. zizmor's `cache-poisoning` audit
flags exactly this. *Fix:* `cache: false` on setup-go in `release.yml`, and no prek or Go cache in the
release job.

**B-5 (Minor) — Runner labels.** `macos-latest` is arm64 and changes without notice, and the darwin/amd64
binary is never run. `ubuntu-22.04` has a retirement date (see the verification log). *Fix:* pin explicit
labels and record a date to revisit them.

**B-6 (Minor) — Duplicate CI runs and no concurrency group.** `on: [push, pull_request]` runs everything
twice for PR branches. *Fix:* `push: branches: [main]` + `pull_request`, plus
`concurrency: { group: ${{ github.workflow }}-${{ github.ref }}, cancel-in-progress: true }` for PRs.

**B-7 (Minor) — size-gate.sh fails its own ShellCheck config.** `--enable=all` turns on optional checks
such as `require-variable-braces` and `require-double-brackets`, which the sample trips (see the
verification log). An empty `dist/` makes the loop run on the literal `dist/*` and `wc` fails under
`set -e`. *Fix:* `shopt -s nullglob`, `[[ … ]]`, `${var}`, and `tr -d ' '` on `wc -c` output (macOS pads
it). Or make the size gate a small Go program (`go run ./scripts/sizegate`) that also runs on Windows
without Bash.

**B-8 (Minor) — The README holds a measured size table.** This drifts on every dependency bump. Put sizes
in the CI step summary and the release notes; the README states only the budget.

---

## 8. Lint and static analysis

**T-1 (Blocker) — The `prek.toml` block as written does not validate.** See the verification log for the
exact errors from `prek validate-config` on the plan's TOML. In short: priorities are integers, so the
named `[priorities]` table and `priority = "fixers"` are invalid; some keys and hook ids differ from the
real ones; and `gitleaks rev = "v8.x"` is a placeholder that cannot resolve. *Fix:* the v2 plan carries a
config that validates against the installed prek, and M0's first CI run proves it.

**T-2 (Blocker) — The `forbidigo` tick rule never fires.** forbidigo matches the identifier or selector
expression (`tea.Tick`), not source text with a trailing `(`. The pattern `tea\.(Tick|Every|Sequence)\(`
therefore matches nothing, and the "single clock" convention has no enforcement. It also misses
`bubbletea.Tick` or any other import alias. *Fix:* `analyze-types: true` with
`{ pattern: '^(Tick|Every|Sequence)$', pkg: '^charm\.land/bubbletea/v2$' }`, plus an exclusion rule for
`internal/ui/clock.go` under `linters.exclusions.rules`.

**T-3 (Major) — Hook revisions are pinned by mutable tags, while Actions are pinned by SHA.** That is
inconsistent supply-chain posture: anyone who controls a hook repo can move a tag. *Fix:* pin hooks by
commit SHA with a `# frozen: vX.Y.Z` comment (`prek auto-update --freeze`).

**T-4 (Major) — `govulncheck@latest` runs unpinned code from the network on every pre-push.** It also
contradicts "all revs pinned". *Fix:* declare it as a Go tool in a separate tool module
(`tools/go.mod`, `go tool -modfile=tools/go.mod govulncheck ./...`), so it is pinned, checked by
Dependabot, and kept out of the main module graph. In CI, also run it on a schedule (weekly), because new
vulnerabilities appear without any commit.

**T-5 (Minor) — zizmor `--fix` runs in the lint tier.** A fixer that rewrites workflow files runs
concurrently with read-only linters, and in CI it only turns a finding into a "files modified" failure.
*Fix:* run zizmor without `--fix` in hooks, and document `zizmor --fix` as a manual command. Run it in CI
with `GH_TOKEN` so its online audits (known-vulnerable actions, impostor commits) also run.

**T-6 (Minor) — Redundant checks.** `go vet` runs twice (golangci-lint's `govet` and the local hook).
`linters.default: standard` already enables `staticcheck`, `errcheck`, `govet` and `unused`. *Fix:* drop
the local `go vet` hook and list only additions to `standard`.

**T-7 (Minor) — Choosing `prek.toml` has a cost.** No bot updates its revs: Dependabot's pre-commit
support (see the verification log) and pre-commit.ci read `.pre-commit-config.yaml`, not `prek.toml`. The
plan acknowledges this and schedules `prek auto-update` by hand. *Fix:* keep the decision, but automate it
with a monthly scheduled workflow that runs `prek auto-update --freeze --cooldown-days 14` and opens a PR
using only `GITHUB_TOKEN`.

---

## 9. Security and supply chain

**S-1 (Blocker) — gitleaks never scans anything in CI.** The upstream hook's entry is
`gitleaks git --pre-commit --staged …`. `prek run --all-files` on a fresh CI checkout has nothing staged,
so the CI pass is a no-op, and locally the hook can be skipped with `--no-verify`. The plan's claim that
"gitleaks runs on every commit (prek) and in CI" is false for the CI half. *Fix:* in CI, run
`gitleaks git --log-opts="${{ github.event.before }}..${{ github.sha }}"` or a full-history scan, pinned by
SHA. **Enable GitHub secret scanning with push protection** in the repository settings (free for public
repos). That is the control that cannot be bypassed.

**S-2 (Major) — Releases have checksums but no provenance.** `checksums.txt` lives in the same release it
protects, so anyone who can alter the release can alter both. *Fix:* add
`actions/attest-build-provenance` to `release.yml` (keyless, uses only `GITHUB_TOKEN` with
`id-token: write` and `attestations: write`) and document `gh attestation verify`. This stays within
"no signing keys in v1".

**S-3 (Major) — Repository settings are part of the threat model and are missing.** *Fix:* add to the
hygiene section: a ruleset protecting `main` (required checks, no force-push); a tag ruleset restricting
`v*` creation to the owner, since anyone with write access can otherwise cut a release; secret scanning
with push protection; private vulnerability reporting plus `SECURITY.md`; and Dependabot security updates.

**S-4 (Major) — The LLM milestone needs a privacy and abuse design, not just "keys from env".** Sending
typed text to a provider contradicts "never phones home" unless it is explicit opt-in. *Fix:* the LLM
brain is enabled only by an explicit flag or env var and shows a one-line in-character notice. It has
request timeouts, max tokens, a bounded history window, no retries without backoff, output sanitised (U-2)
and validated actions (already planned), and no logging of prompts or replies.

**S-5 (Minor) — Debug log location and contents.** `WOPR_DEBUG` writes a log, but the plan does not say
where. *Fix:* write under `os.UserCacheDir()/wopr/`, mode 0600, never in the CWD (a public repo checkout).
Keep "never typed input".

**S-6 (Minor) — `dependabot.yml` has no cooldown or grouping.** *Fix:* `cooldown: { default-days: 7 }`
and group minor and patch updates per ecosystem, so supply-chain incidents get time to surface and PR
noise stays low.

---

## 10. Legal and licensing

**L-1 (Major) — Provenance of film text and ASCII art.** The research section says the lines came from
fan repositories (`built1n/wargames`, `abs0/wargames`, `elfuska/wargames`). Copying transcript text or
ASCII maps from those repos brings in *their* licences (see the verification log), on top of the film's
copyright. *Fix:* draw ASCII art from scratch for this project. Take lines from the film itself and keep
them short, limited to what the screens show. Record provenance per data file. `NOTICE.md` must list
anything third-party.

**L-2 (Minor) — Trademark.** Do not use "WarGames" or MGM marks in the project name, binary name or
artwork. Use them only nominatively in the README ("inspired by the 1983 film *WarGames*"), with the
non-affiliation disclaimer the plan already includes. No film stills in the README; use screenshots of
this program only.

**L-3 (Minor) — The brother's prompt.** Get permission in writing (an issue comment or a commit by him is
enough) before any adapted text lands, and credit him the way he prefers. The plan says this; make it an
M1 entry gate.

---

## 11. Testing

**Q-1 (Major) — The real TUI binary is never exercised in CI.** Smoke tests run only `--version` and
`--games`. Golden tests drive the model, not the program. *Fix:* one end-to-end test per OS that starts
the real binary in a pseudo-terminal with `--instant --seed 1`, types `Joshua`, waits for
`SHALL WE PLAY A GAME?`, and sends Ctrl+C, then asserts exit code 130 and that the terminal is restored.
On Linux and macOS this can be `script`-based or Go-pty-based. On Windows, use ConPTY (via
`teatest/v2`, or a small harness).

**Q-2 (Minor) — Fuzzing.** Normalisation, `games.Resolve`, the line editor, SAN/UCI parsing and
`Sanitize` are pure functions that take untrusted input. Native Go fuzz targets are cheap. Run them
briefly in CI (`-fuzztime=10s`).

**Q-3 (Minor) — Golden-file mechanics.** Specify the `-update` flag and where goldens live
(`testdata/`, LF-only). Say how to review diffs.

**Q-4 (Minor) — AI quality tests.** Tic-tac-toe "never loses" is good. Add perft tests for the chess
generator (if custom), checkers forced-capture puzzles, and a "WOPR beats random play in ≥ 95% of 200
seeded games" check per AI.

---

## 12. Documentation and maintainability

**M-1 (Major) — AGENTS.md is written last, but implementation is done by agent sessions.** The plan says
"implementation happens in later sessions", yet the conventions those sessions must follow (import DAG,
single clock, no Bubble Tea in `games/`) only arrive at the end of M5. *Fix:* write AGENTS.md in M0 and
update it each milestone. Have a `CLAUDE.md` that points to it.

**M-2 (Major) — The plan embeds full config files.** A 200-line `prek.toml` and shell scripts inside the
plan will drift from the real files the day M0 lands, and the plan's copy already has errors (T-1). *Fix:*
the plan states intent and constraints. After M0 the real files are the source of truth, and the plan
links to them.

**M-3 (Minor) — `ui` becomes a god package.** The root model, phase machine, gamehost, screens and keys
all live in `internal/ui`. *Fix:* sub-models per screen (connect, console, game host, ending) behind a
small interface, with the root as a thin router.

**M-4 (Nit) — Markdown issues the plan's own linter would flag.** There is a missing blank line before
`### Tooling` (line 489–490), and some lines are very long. Run `rumdl` on the plan too.

---

## 13. Milestones and implementability

**P-1 (Blocker) — GTW (M2) depends on `sim` (M4).** As written, either M2 builds GTW without the engine
and M4 rewrites it, or M2 quietly pulls in M4. *Fix:* make GTW a scripted set piece that uses a small
`gtw`-local model (targets, salvos, DEFCON ladder, kill-ratio tables). Extract shared pieces into `sim`
when the second consumer arrives in M4. Alternatively, move the minimal `sim` core into M2.

**P-2 (Major) — Nothing ships until every game is done.** M5 is the first release. Five of the 16 games
are under-specified (G-2), and Bridge is open-ended (G-3), so v0.1.0 could slip indefinitely. *Fix:*
release v0.1.0 at the end of M2: the full film path (LOGON → greeting → GTW → tic-tac-toe → montage →
ending) plus chess and checkers. Games not yet implemented stay on `LIST GAMES`, as in the film, and
WOPR answers in character (`** GAME ROUTINE NOT AVAILABLE **`). Each later milestone ships a minor
release. *Owner decision: see Questions.*

**P-3 (Minor) — Line-count estimates.** "~600 lines each" is low for Hearts, Gin Rummy, Poker AI and GTW,
and very low for Bridge. Size by milestone instead, and track actuals.

---

## Questions for the owner (answered 2026-10-06)

| # | Question | Answer | Effect on v2 |
|---|---|---|---|
| 1 | Ship v0.1.0 after the film set pieces (M2), with other games added in minor releases? | **Yes.** | v0.1.0 at the end of M2. Unbuilt games stay listed and WOPR answers in character. |
| 2 | Bridge: cut, minimal, or full? | **Minimal variant.** | WOPR bids all four seats by point count. The user plays declarer. Built last in the card milestone. |
| 3 | Build once with GoReleaser and run natively on each OS/arch, instead of native builds per OS? | **Yes.** | One GoReleaser snapshot build. A smoke matrix runs each binary on its own OS and architecture. |
| 4 | `go 1.26.0` (the stack's minimum) with `toolchain go1.27.1`, instead of `go 1.27`? | Not asked; the recommended default is adopted. | `go` = the highest minimum any dependency declares; `toolchain go1.27.1`. |
| 5 | 10 MB hard limit, or keep the 10/15 MB warn/fail split? | **Keep the 10/15 split.** | Decision 10 stands. R-1 is resolved as "owner confirmed". |

---

## Verification log

Each line is a claim from the plan, how it was checked, and the result.

_(filled in below)_
