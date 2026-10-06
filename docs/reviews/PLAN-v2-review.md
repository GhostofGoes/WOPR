# Adversarial review: WOPR plan v2

_Reviewed: `docs/PLAN.md` v2 at commit `5dc712d` ("Rewrite the WOPR plan as v2"). Review date 2026-10-06. The plan
that answers it is v2.1, in the same branch._

This review attacks v2 the way [PLAN-v1-review.md](PLAN-v1-review.md) attacked v1: it looks for claims that are
false, contracts that are missing, and places where the plan contradicts itself or the code built from it. It
uses the same severity scale (Blocker, Major, Minor, Nit). Finding ids carry a dimension prefix: **AR**
architecture, **RF** runtime behaviour and film fidelity, **CR** CI and releases, **TO** tooling, **SL**
security and legal, **IM** implementability. v1 ids (A-3, B-10, …) refer to the v1 review.

## Method

1. **Find.** Six reviewers, one per dimension, read v2 against the live code and upstream sources, and ran
   local experiments where a claim could be tested: a mini-module running archtest's rules, golangci-lint and
   prek against the plan's configs, GoReleaser on an untagged repository, gitleaks on a repository with a stale
   branch, a chess search under `-race`, WCAG contrast arithmetic, and diffs against the fan transcriptions.
2. **Refute.** One skeptic per dimension tried to refute every finding, re-ran the experiments, and re-graded
   it. No finding was refuted outright: 47 were confirmed and 12 judged overstated. The skeptic also caught one
   problem the reviewers missed (see SL-2).
3. **Coverage.** A final pass checked each of the v1 review's findings, owner decisions and completeness gaps
   (138 items) against v2.

M0 and most of M1 were built on the same branch while the review ran, so many findings had already been met in
code: the plan was behind the code more often than the code was wrong. Each finding's **Status** says where it
now stands, in the code, in plan v2.1, or as a contract for a later milestone.

## Verdict

v2 is implementable, and the code built from it shows that. There are **no Blockers**. The 59 findings reduce
to 46 distinct issues once duplicates across dimensions are merged: 10 Major, 26 Minor and 10 Nit.

The problems that mattered, in order:

1. **A latent CI failure (SL-2).** Verifying the gitleaks finding showed that `go tool gitleaks` no longer
   compiled: golangci-lint, added to `tools/go.mod` in M0, pulled in an `x/ansi` that breaks gitleaks's
   dependencies. The `secrets` job, and with it `ci-ok`, would have failed on every run, and the M0 canary would
   have "turned red" for the wrong reason. Its `ERR` check could not have worked either: gitleaks colours its
   log even into a pipe, and exits 0 when git fails.
2. **M0's exit criteria could not all be met as written.** The lint fixture sat inside `./...`, so the lint gate
   and archtest failed on its deliberate violations (AR-3); the release dry run ran GoReleaser in release mode on
   an untagged commit (CR-1); and archtest's exact toolchain check broke `go test` for anyone with a newer Go
   (AR-2). The code had already worked around all three; the plan had not caught up.
3. **The import table and archtest disagreed** (AR-1, AR-4). The plan had no test-only imports, no subtree rule,
   no root package for the embedded licences, missed edges its own text needed, and never saw the e2e files.
4. **M1 contracts were missing** (IM-6, IM-7, AR-9, AR-10): the game types, the resolver for `Launch`, who
   follows `Result.Next`, what the greeting scene consumes, the `Think` cancellation rules, and per-game seeds.
   Every game launched with the same seed, so chess, checkers and each replay drew identical AI sequences.
5. **Movie mode did not fit the protocol** (AR-7): the director cannot see Esc, cannot pause host pacing, and
   cannot feed the ending. The fixes are additive host hooks, now specified for M6.
6. **Film text and provenance** (RF-3, RF-9, SL-5, SL-6): the climax lost the film's `LIST GAMES` beat, five
   montage names were not abs0's text, the call-back merged two scenes, the status burst copied an
   all-rights-reserved layout byte for byte, and squash merging did not keep the CC BY-SA fragments out of the
   public repository.
7. **Smaller but real:** a `-race` chess search can reach a 1.5 s budget (AR-8); layout geometry ignored the
   front panel and a mockup was 23 rows (AR-11); intent missed ordinary phrasings (RF-7); norad's Dim text was
   2.2:1 in 16 colours (RF-10); and the weekly report would never have closed its issue (CR-6).

Everything above is fixed in code or plan v2.1, except the parts that belong to later milestones (the M2 climax,
the M6 host hooks), which are now written down as contracts.

## Summary of findings

Verified severity first; "was" gives the reviewer's grade where the skeptic changed it. A finding that repeats
another points to it.

| ID | Finding | Severity | Verdict | Status |
|---|---|---|---|---|
| AR-1 | archtest as specified rejects the test imports the plan itself requires, and its own row forbids the imports its other checks need | Major | confirmed | Fixed |
| AR-2 | archtest's toolchain-equality check fails for any contributor whose local Go is newer than go1.27.1 | Major | confirmed | Fixed |
| AR-3 | The lint fixture sits inside ./..., so the normal lint run and archtest both fail on its deliberate violations | Major | confirmed | Fixed |
| AR-4 | Production edges the plan's own text requires are missing from §4.1, so archtest and depguard reject M0/M1 code | Major | confirmed | Fixed |
| AR-5 | Launch/Result.Next have no resolver: the host cannot build programs, `ending` is not a registry entry, and who follows Next is unstated | Minor (was Major) | overstated | Same as IM-6 |
| AR-6 | games/ending cannot reuse tic-tac-toe (or GTW's board) for the climax; the only ending edge points the wrong way | Minor | confirmed | Fixed |
| AR-7 | Movie mode's director does not fit the protocol it is claimed to run under (Esc, pause, Run, Type) | Major | confirmed | Planned |
| AR-8 | Limit is invisible to the host, and Budget still applies in deterministic mode, where -race makes a time-out likely | Major | confirmed | Fixed |
| AR-9 | ThinkDone's cancellation contract contradicts the Esc rules, and Think results have no correlation | Minor | confirmed | Fixed |
| AR-10 | Stream IDs need inputs that Env does not carry; the AI stream is not keyed by game or play | Minor | confirmed | Fixed |
| AR-11 | Full layout plus the front panel needs H+1 rows; the GTW 80×24 mockup has 23 rows | Minor | confirmed | Fixed |
| RF-1 | Movie-mode controls and the Run step cannot be built on the v2 protocol and host rules | Major | overstated | Same as AR-7 |
| RF-2 | Climax tic-tac-toe has undefined exits, and leaving it silently ends the unwinnable GTW | Minor (was Major) | overstated | Planned for M2 |
| RF-3 | The GTW climax diverges from the film: LIST GAMES is rejected, notices are not tied to inputs, and the launch code has no owner | Minor | confirmed | Planned for M2 |
| RF-4 | The Sanitize* step order deletes tabs and newlines before mapping them, so pasted lines are glued together | Minor | confirmed | Fixed in the plan |
| RF-5 | Movie replays are not deterministic as specified, and the e2e and §14 exit behaviour contradict each other | Minor | confirmed | Planned for M6 |
| RF-6 | Is LIST GAMES numbered? R3, §14 and numbered selection disagree with the film list that the movie consistency test checks | Nit (was Minor) | overstated | Fixed |
| RF-7 | The intent clause rule is ambiguous: either reading gives counterexamples, and the negator rule never fires | Minor | confirmed | Fixed |
| RF-8 | The Appendix C mockups break the §4.3 layout geometry and the "exactly 80×24" claim | Minor | confirmed | Same as AR-11 |
| RF-9 | Appendix B text and provenance: the montage list is not abs0's list, "61 HOURS" contradicts other sources, and the call-back scene merges two scenes | Minor | confirmed | Fixed in the plan |
| RF-10 | ANSI-16 choices make Dim text unreadable on common palettes (norad Dim on the Linux console; imsai Dim on Solarized) | Minor | overstated | Fixed |
| RF-11 | §14 step 3 types the GTW line straight after the greeting, but the active greeting scene is dispatched before intent | Nit | confirmed | Same as IM-7 |
| CR-1 | The release.yml dry run, an M0 exit criterion, fails as written: `goreleaser release` with no tag at HEAD exits 1 | Major | confirmed | Fixed |
| CR-2 | Concurrency rule: main pushes can still be cancelled, and the same block in smoke.yml or scheduled.yml deadlocks or collides | Minor | confirmed | Fixed |
| CR-3 | The reusable smoke workflow has no defined interface, and the release build produces neither per-target artifacts nor e2e binaries | Minor | confirmed | Fixed |
| CR-4 | GoReleaser via `go tool` ties GoReleaser updates to the shipped compiler, costs ~2 min and 415 MB per uncached job, and is outside the weekly report | Minor | confirmed | Fixed |
| CR-5 | archtest's "running Go == toolchain line" fails on any contributor machine with a newer Go | Minor | confirmed | Same as AR-2 |
| CR-6 | The weekly staleness report on `go list -m -u all` always has findings, so the tracking issue never closes | Minor | confirmed | Fixed |
| CR-7 | CI snapshot builds always report version 0.0.1-snapshot, because the build checkout fetches no tags | Nit | confirmed | Fixed |
| CR-8 | `changelog: use: git` is dead config: releases are disabled and the notes come from `gh --generate-notes` | Nit | confirmed | Fixed |
| TO-1 | The lint self-test fixture breaks the lint gate where the plan puts it, and nothing says how the self-test runs | Major | confirmed | Same as AR-3 |
| TO-2 | The builtin whitespace and end-of-file fixers rewrite golden files | Major | confirmed | Fixed |
| TO-3 | The codespell config fails on the repository's own files: `tools/*.sum` and the word `crasher` | Minor | confirmed | Fixed |
| TO-4 | The misspell and `rand.go` exclusions never match, and the RECOGNISED rationale is false | Minor | confirmed | Fixed |
| TO-5 | The determinism lint misses `math/rand` (v1), whose top-level functions are randomly seeded | Minor | confirmed | Fixed |
| TO-6 | The lint gate never lints or type-checks the `e2e` build-tagged tests | Minor | confirmed | Fixed |
| TO-7 | The Go tool modules' version rule is misstated, and the weekly report skips the tool modules | Minor | confirmed | Fixed |
| TO-8 | gitleaks is pinned twice, with two update paths that can drift | Nit | confirmed | Fixed |
| TO-9 | The weekly report keys off exit codes that do not mean what it needs, and the seed is already stale | Nit | confirmed | Fixed |
| SL-1 | THIRD_PARTY_NOTICES.txt depends on the host OS, but one copy is committed, embedded in all six binaries and checked on three OSes | Nit (was Major) | overstated | Fixed in the plan |
| SL-2 | The full-history gitleaks scan covers every branch, has no allowlist procedure, and "fails on any ERR line" has no implementation home | Major (was Minor) | confirmed | Fixed |
| SL-3 | No planned package can embed the root-level LICENSE, NOTICE.md and THIRD_PARTY_NOTICES.txt; M0 already added an unlisted root package | Nit (was Minor) | overstated | Fixed in the plan |
| SL-4 | The provenance test checks only that a tag exists: third-party tags are not checked against NOTICE.md, `prompt` exclusion is not enforced, and film text transcribed by abs0 cannot be both credited and excluded from the MIT grant | Nit (was Minor) | overstated | Fixed |
| SL-5 | "Exactly two text items" from abs0 leaves out the status burst, which Appendix B tags as abs0 but whose wording matches the all-rights-reserved elfuska source | Minor | confirmed | Fixed |
| SL-6 | Squash-only merges keep v1 off `main`, but not out of the public repository, which is what the CC BY-SA concern is about | Minor | confirmed | Fixed |
| SL-7 | The privacy claims for the debug log ignore Bubble Tea's TEA_TRACE, which logs every keystroke and frame to a path taken from the environment | Nit (was Minor) | overstated | Fixed in the plan |
| SL-8 | The R-1 and §7 mitigations for movie mode misdescribe what M6 ships, and there is no takedown runbook despite immutable releases and the Go proxy cache | Minor | confirmed | Fixed |
| IM-1 | The lint self-test fixture is placed where the lint gate itself lints it, so M0 cannot have both a green ci-ok and a fixture that fires | Major | confirmed | Same as AR-3 |
| IM-2 | The M0 release.yml dry run cannot pass: `goreleaser release` without --snapshot needs a tag on HEAD, and no tag exists until v0.1.0 (M2) | Major | confirmed | Same as CR-1 |
| IM-3 | M0 and M1 exit criteria depend on work from later milestones (e2e, movie mode) and on owner actions | Minor (was Major) | overstated | Fixed in the plan |
| IM-4 | The archtest edge table, which also checks test imports, forbids the plan's own test strategy and omits packages M0 and M1 must create | Major | confirmed | Same as AR-1 |
| IM-5 | archtest's "running Go version equals the toolchain line" fails `go test ./...` for any contributor whose Go is newer than 1.27.1 | Major | confirmed | Same as AR-2 |
| IM-6 | The M1 protocol leaves key contracts undefined: games.Info/Status/Game, Env sizing, who resolves Launch, who acts on Result.Next, and how the ending is launched | Major | confirmed | Fixed |
| IM-7 | Persona "scenes" (greeting, GTW-vs-chess) are undefined, and the dispatch order contradicts the §14 walkthrough | Minor (was Major) | overstated | Fixed |
| IM-8 | Layout geometry, the front panel and the Appendix C mockups disagree, and the mockups claim to be exactly 80×24 | Minor (was Major) | overstated | Same as AR-11 |
| IM-9 | §4.2 and §5 make cmd/wopr do Bubble Tea work that the DAG and depguard forbid there | Minor | confirmed | Same as AR-4 |
| IM-10 | CLI rules contradict each other and leave the expected outcome of the pinned cases unstated | Minor | confirmed | Fixed |
| IM-11 | Movie-mode controls and exit behaviour contradict the host's Esc rules and each other | Minor | confirmed | Same as AR-7 |
| IM-12 | Dangling and colliding identifiers and small cross-reference drift | Nit | confirmed | Fixed in the plan |

## Architecture (AR)

### AR-1. archtest as specified rejects the test imports the plan itself requires, and its own row forbids the imports its other checks need

**Severity:** Major. **Verdict:** confirmed. **Also found as:** IM-4.

**Status:** Fixed. archtest (since M0) has a test-only list, `/...` subtree rules and the root package, and now lists packages with `-tags=e2e`; plan §4.1 mirrors it and names archtest authoritative.

**Plan text.** §4.1 (PLAN.md:169, 188-189, 194, 205-207): "Import edges not listed below are forbidden"; "internal/archtest reads each package's Imports, TestImports and XTestImports from go list -json ./... and checks every edge against this table. It also checks that every Playable registry entry has a constructor"; "internal/archtest → stdlib"; §9 (PLAN.md:793-794): "archtest covers the DAG, test imports, catalog completeness, toolchain equality and provenance tags"

**Problem.** The table has no test-only edges. Several tests the plan requires would therefore fail archtest:

- wopr, cli and ui tests that "use fakes from gamestest" (§4.1);
- every game's "deterministic testkit transcript" (§6.1), which needs games/&lt;game&gt; tests → games/testkit;
- every golden test → internal/golden;
- ui tests → ui/testutil.
"Importable from tests only" never says which packages' tests may import gamestest or testkit.

The §7 consistency test has no legal home. It needs movie scenes, the real persona, testkit and the real catalog (the first-contact scene prints the real LIST GAMES names), and no row allows that combination.

The table has no rows for ui/console, ui/screens, ui/testutil or tools/*. `go list ./...` silently drops internal/e2e, which is behind a build tag, so the xpty edge is never checked. Files limited to one GOOS are checked only on that OS.

archtest's own row is "stdlib", yet catalog completeness and provenance-tag checks need the values in catalog, wopr, assets and movie.

**Evidence.** Local experiment: I built a mini-module (scratchpad arch-v2rev-K3/m) with 15 packages and the §4.1 edges verbatim. Its archtest reads Imports, TestImports and XTestImports from `go list -e -json ./...`. The run fails with 7 violations:

- tictactoe XTestImports → games/testkit and internal/golden;
- wopr TestImports → games/gamestest and internal/golden;
- movie XTestImports → wopr, games/testkit and games/catalog.
`go list ./...` omits internal/e2e. `go list -e -f '{{.IgnoredGoFiles}}' ./internal/e2e` shows e2e_test.go, and its TestImports appear only with `-tags e2e`.

**Proposed fix.** Add a "tests may also import" column, for example:

- any package's tests → golden and gamestest;
- games/&lt;game&gt; tests → testkit;
- ui tests → ui/testutil.
Designate one integration test package (for example internal/integration, tests only) that may import wopr, movie, catalog and testkit for the consistency test and the film-path goldens.
Define prefix semantics for subpackages (ui/** inherits ui), or list them.
Run `go list -e -json -tags e2e` once for each of GOOS=linux, darwin and windows.
Move catalog completeness into a catalog_test.go and provenance checks into each data package's tests. Alternatively, let archtest's _test.go import them, but then say so in its row.

**Verification.** The core claim holds. §4.1 says "Import edges not listed below are forbidden", and archtest checks TestImports and XTestImports "against this table" (PLAN.md:169, 205-207). Yet no row lets any package's tests import internal/golden, which §9 uses everywhere. No row lets games/&lt;game&gt; tests import testkit, ui tests import ui/testutil, or movie tests import wopr.

Evidence:

- I re-ran the reviewer's mini-module archtest (copied to skeptic-arch-v2-W5n/m) and got the same 7 violations.
- The repo's own M0 archtest (internal/archtest/archtest_test.go at 218ae61) had to invent rules the plan lacks: a `testOnly` list (golden, catalog, gamestest, testkit, importable from any test) and `/...` subtree rules for ui, games and movie. It also moved the catalog-completeness check into registry validation (internal/games/registry_test.go). So plan and code already disagree.
- e2e: Go 1.27.1 src/cmd/go/internal/modload/search.go:143 drops directories whose scan returns ErrNoGo. In the repo, `go list -f '{{.TestImports}}' ./internal/e2e` is empty, with e2e_test.go in IgnoredGoFiles. xpty appears only with `-tags e2e`.

Two sub-points are wrong:

- GOOS-only files are not a gap. §9 runs `go test -race ./...`, which includes archtest, on ubuntu, macos and windows.
- tools/* is a nested module (tools/go.mod). search.go:131-135 stops at module boundaries, and internal/tools/* already has a row.

The severity stays Major: the plan fails as written, though the fix is cheap.

**Corrected fix.** Write into §4.1 what M0 code already does:

- **Test edges:** any package's tests may also import golden, gamestest, testkit and catalog.
- **Subpackages:** `/...` means a whole subtree (ui/..., games/..., movie/...).
- **Consistency test:** give the §7 test an explicit home. Either movie's _test files may also import wopr, or use a tests-only package such as internal/integration.
- **e2e:** archtest runs `go list` a second time with `-tags e2e`. Per-GOOS runs are unnecessary, because the three-OS test job already covers them.
- **archtest stays stdlib:** catalog completeness lives in games registry validation or a catalog_test.go, and provenance checks live in each data package's tests.

### AR-2. archtest's toolchain-equality check fails for any contributor whose local Go is newer than go1.27.1

**Severity:** Major. **Verdict:** confirmed. **Also found as:** CR-5, IM-5.

**Status:** Fixed. Exact equality only in CI, `>=` locally (`TestToolchainMatchesGoMod`); plan §4.1, §11.1.

**Plan text.** §4.1 (PLAN.md:206-207): "and that the running Go version equals go.mod's toolchain line (D-6)"; §11.1 (PLAN.md:878-879): "archtest asserts the toolchain version"; A.1 go-test pre-push hook `go test -short ./...`

**Problem.** Under the default GOTOOLCHAIN=auto, the toolchain line is a minimum. A local toolchain newer than go1.27.1 is kept, not downgraded. The test binary then reports that newer version, and the equality check fails. Examples: go1.27.2 from Homebrew or a distribution, or go1.28 in Feb 2027.

The effects:

- `go test ./...`, the AGENTS.md command, goes red for those contributors.
- The prek pre-push go-test hook blocks every push for them.
- During the window the 14-day response rule allows between a Go security point release and the toolchain-bump PR (§11.5), everyone who has updated Go is blocked.

D-6's real goal is "CI must not silently downgrade", which only needs to be enforced in CI.

**Evidence.** Go 1.27.1 source, src/cmd/go/internal/toolchain/select.go:193-202: "Accept toolchain only if it is &gt; our min" (`if gover.Compare(toolVers, minVers) > 0`).

Local experiment: I set go.mod to `toolchain go1.27.0` and ran the module-cache go1.27.1 binary, whose go.env has GOTOOLCHAIN=auto. `go version` printed go1.27.1, and the archtest TestToolchain failed with "running go1.27.1, go.mod toolchain go1.27.0".

**Proposed fix.** Require equality only when `CI=true`, which GitHub Actions sets. Locally, assert `runtime.Version() >= toolchain`. Alternatively, move the equality check into a ci.yml step that compares `go env GOVERSION` with go.mod, and keep archtest version-agnostic.

**Verification.** In the Go 1.27.1 source, src/cmd/go/internal/toolchain/select.go:193-202 accepts the go.mod toolchain only "if it is &gt; our min" (`gover.Compare(toolVers, minVers) > 0`). A newer local Go is therefore kept.

Local experiment: I ran the go1.27.1 binary from the module cache (GOTOOLCHAIN=auto) against a module whose go.mod says `toolchain go1.27.0`. `go version` printed go1.27.1, and a runtime.Version()==toolchain test failed with "running go1.27.1, go.mod toolchain go1.27.0".

The plan states the check unconditionally (PLAN.md:206-207, 878-879). The A.1 pre-push `go test -short ./...` hook would also run it. Contributors on a newer Go therefore go red until the toolchain is bumped.

The repo's M0 test, TestToolchainMatchesGoMod, already skips unless GITHUB_ACTIONS=true, which confirms the fix was needed. The plan text was not updated.

### AR-3. The lint fixture sits inside ./..., so the normal lint run and archtest both fail on its deliberate violations

**Severity:** Major. **Verdict:** confirmed. **Also found as:** IM-1, TO-1.

**Status:** Fixed. The fixture is under `internal/archtest/testdata/lintfixture`, linted by `TestLintRulesFire` with golangci-lint from `tools/lint/go.mod`; plan §10.

**Plan text.** §10 (PLAN.md:859-860): "internal/archtest/lintfixture holds a deliberate violation of each rule. A CI step asserts that each rule reports on it."; §15 M0 exit (PLAN.md:1063): "ci-ok green ... the lint fixture fires every rule"; §4.1: "internal/archtest → stdlib"

**Problem.** The depguard rule needs the fixture to import charm.land/bubbletea/v2 outside ui, and the forbidigo rules need it to call time.Sleep and tea.Tick.

- prek's golangci-lint-full hook lints the whole module, and A.2 has no exclusion for the fixture, so `prek run --all-files` (the lint gate) goes red.
- archtest's `go list ./...` includes internal/archtest/lintfixture, which has no row, and its Bubble Tea import is a forbidden edge.
The two M0 exit criteria ("ci-ok green" and "the fixture fires every rule") cannot both hold as written.

**Evidence.** golangci-lint v2.14.0 .pre-commit-hooks.yaml: golangci-lint-full is `entry: golangci-lint run --fix`, `pass_filenames: false` (whole module).

Local experiment (scratchpad arch-v2rev-K3/lf, A.2 config extracted verbatim, golangci-lint 2.14.0 on go1.27.1): with the fixture at internal/archtest/lintfixture, `golangci-lint run ./...` exits 1 with 3 issues (depguard ×1, forbidigo ×2). Moving the fixture to internal/archtest/testdata/lintfixture makes `./...` exit 0, while `golangci-lint run ./internal/archtest/testdata/lintfixture/` still reports all 3.

**Proposed fix.** Put the fixture under internal/archtest/testdata/lintfixture/. The `./...` patterns skip testdata, which keeps both the lint gate and archtest clean. The self-test step then lints that path explicitly and asserts one report per rule.

**Verification.** Local experiment: I ran golangci-lint 2.14.0 (go1.27.1) with the A.2 config verbatim (it diffs identical to PLAN.md:1197-1239).

- With the fixture at internal/archtest/lintfixture, `golangci-lint run ./...` exits 1 and reports depguard ×1 and forbidigo ×2 in the fixture.
- Under internal/archtest/testdata/lintfixture, `./...` does not report the fixture, while `golangci-lint run ./internal/archtest/testdata/lintfixture/` reports all 3.

Go 1.27.1 modload/search.go:105 skips `testdata` in `./...` patterns, so archtest's `go list ./...` would otherwise also see the fixture's Bubble Tea import as a forbidden edge.

The A.1 hook `golangci-lint-full` runs `golangci-lint run` over the whole module. The repo's M0 already placed the fixture at internal/archtest/testdata/lintfixture/fixture.go. The plan text (PLAN.md:859) still names the old path.

### AR-4. Production edges the plan's own text requires are missing from §4.1, so archtest and depguard reject M0/M1 code

**Severity:** Major. **Verdict:** confirmed. **Also found as:** IM-9.

**Status:** Fixed. `legal.go` at the root, `ui.Run` and `ui.TerminalProblem` own the Bubble Tea steps, and the edges are in archtest; plan §4.1, §4.2.

**Plan text.** §4.1 (PLAN.md:172, 174, 183, 185, 197, 200): "cmd/wopr → cli, ui, version, games/catalog, (llm in M7)"; "Only ui imports Bubble Tea. Only ui and theme import charm.land/*"; "internal/games → proto"; §4.2 items 2-3 (PLAN.md:243-249): "main passes tea.WithColorProfile(colorprofile.Ascii)"; §5 (PLAN.md:606-607): "Bubble Tea errors are mapped ErrInterrupted first (130), then ErrProgramPanic (1)"; §12 (PLAN.md:995-997): THIRD_PARTY_NOTICES.txt "embedded in the binary for wopr --licenses"

**Problem.** Several duties the plan assigns to packages need imports their rows do not allow.

1. main's duties need non-stdlib imports its row forbids:
   - the TTY check needs a term package;
   - NO_COLOR needs bubbletea and colorprofile;
   - exit-code mapping needs tea.ErrInterrupted and tea.ErrProgramPanic.
   §4.1's own layout line also gives main "NO_COLOR and TTY handling".
2. games/chess (and games/ai, if it searches chess positions) needs github.com/corentings/chess/v2 (decision 15). The table lists third-party edges for ui, theme and e2e, but none for games.
3. games.Resolve matches an "exact normalised name", but games may not import prompt. A second normaliser can drift from the shell's intent matching, for example on `AIR-TO-GROUND` vs `AIR TO GROUND`.
4. `--theme` validation (a usage error, exit 2) needs the theme names, but cli has no theme edge.
5. go:embed cannot reach NOTICE.md or THIRD_PARTY_NOTICES.txt at the repository root from any listed package, and there is no root package.
6. SanitizeText needs x/ansi, which only ui may import, yet §4.7 applies it to LLM replies in llm/wopr.
7. "cmd/wopr passes the registry to ... wopr", but cmd/wopr has no wopr edge.

**Evidence.** Local experiment (scratchpad arch-v2rev-K3/lf): a cmd/wopr/main.go doing exactly the §4.2/§5 duties builds only with imports of charm.land/bubbletea/v2, github.com/charmbracelet/colorprofile and github.com/charmbracelet/x/term. golangci-lint 2.14.0 with A.2 reports `cmd/wopr/main.go:8:2: import 'charm.land/bubbletea/v2' is not allowed from list 'bubbletea-only-in-ui' (depguard)`.

Local experiment: `//go:embed ../../THIRD_PARTY_NOTICES.txt` in cmd/wopr fails with "pattern ../../THIRD_PARTY_NOTICES.txt: invalid pattern syntax". go1.27.1 src/embed/embed.go:68 says "Patterns may not contain '.' or '..' ... path elements".

**Proposed fix.** Move every Bubble Tea-touching step into `ui.Run(cfg) (exitCode int)`, so main stays tea-free and depguard stays simple. Then add these edges:

- games/chess and games/ai → github.com/corentings/chess/v2;
- games → prompt (one normaliser);
- cli → theme (or have ui validate the theme, mapped to exit 2).
Add a module-root package (for example /notices.go with `//go:embed NOTICE.md THIRD_PARTY_NOTICES.txt`) with the edge cmd/wopr → root. Alternatively, generate the notices into internal/notices/ and point GoReleaser's `files: src:` there.
Put SanitizeText in prompt, with a prompt → x/ansi edge, so llm and wopr can call it.

**Verification.** The hard sub-points are verified:

- **(1) main and Bubble Tea.** A main.go doing §4.2 items 2-3 and the §5 tea error mapping fails A.2's depguard. Local re-run: "cmd/wopr/main.go:8:2: import 'charm.land/bubbletea/v2' is not allowed from list 'bubbletea-only-in-ui'".
- **(5) Notices embed.** Go 1.27.1 src/embed/embed.go:68 says patterns may not contain '.' or '..', so no internal package can embed root files.
- **(2) Chess library.** No row allows github.com/corentings/chess/v2.

The repo's M0 code had to make exactly these changes:

- legal.go, a root package `wopr` that embeds LICENSE, NOTICE.md and THIRD_PARTY_NOTICES.txt, with cmd/wopr → ".";
- ui.TerminalProblem and ui.Run returning an outcome, so main stays tea-free;
- archtest rules adding cli → theme, games → prompt, games/... → corentings/chess/v2 and ui → x/term.

Two sub-points are weak:

- (6) ui could apply SanitizeText to Brain-originated Say lines at render time, so this is a placement choice, not a hard contradiction.
- (7) is wording: ui can build the persona from the registry that main passes it.

The proposed fix also misses that the TTY check needs a term package wherever it moves. A stdlib os.ModeCharDevice test classes /dev/null as a terminal.

**Corrected fix.** Have `ui.Run(opts)` own every tea-touching step: NO_COLOR via WithColorProfile, the TTY check, and mapping Bubble Tea errors to an outcome. main only maps that outcome to an exit code. Rewrite §4.2 items 2-3 and §5 to say so.

Add these edges:

- ui → github.com/charmbracelet/x/term (already in the module graph via Bubble Tea);
- games/&lt;game&gt; (and games/ai only if it touches chess types) → github.com/corentings/chess/v2;
- games → prompt (one normaliser for Resolve and intent);
- cli → theme (for --theme validation, exit 2).

Add a module-root package (legal.go, `//go:embed LICENSE NOTICE.md THIRD_PARTY_NOTICES.txt`) with cmd/wopr → root.

For M7, either put SanitizeText in prompt with prompt → x/ansi, or state that ui sanitises every Brain-originated line at render.

### AR-5. Launch/Result.Next have no resolver: the host cannot build programs, `ending` is not a registry entry, and who follows Next is unstated

**Severity:** Minor (was Major). **Verdict:** overstated. **Same issue as IM-6**, which carries the status.

**Plan text.** §4.2.5 (PLAN.md:254-257): "Launch pushes a game; Done pops it and delivers GameOver{Result} to the program below ... There is no InGame phase"; §4.4 (PLAN.md:401, 409): "Launch struct{ Slug, Mode string } // persona → host", "Next *Launch // hand-off without a verdict, e.g. gtw → tictactoe(climax) → ending"; §4.1: "internal/proto/host → proto", ui row (no games/ending, no catalog); §15 M0: "catalog with 16 Planned entries"; §6.2 (PLAN.md:659, 670): "Done{NoWinner, Next: Launch{ending}}", "The persona's phase goes Ending → Shell with Offer{chess} armed"; §4.6: "When PhaseSet // Greeting | Shell | Ending"

**Problem.** (a) proto/host imports only proto, so it can turn a slug into a Program only through an injected factory. The plan never defines that factory: its signature, its error for unknown slugs, or its handling of Planned games.

(b) `ending` is not one of the 16 catalog entries, and ui, the only package that could build the factory, may not import games/ending. If `ending` is added to the catalog as `Listed: false`, it appears under --games "ALSO AVAILABLE" and resolves via `wopr -p ending` (or the prefix `end`), because §5 makes unlisted entries resolvable.

(c) Nobody is assigned to follow Next.

- If the runner follows it, the persona never sees gtw's or tic-tac-toe's GameOver and never enters Ending. It then receives only the ending's GameOver, which carries no slug, so it cannot tell that apart from any other NoVerdict result in order to arm Offer{chess}.
- If the persona follows it, the plan must say so.

(d) Rules with `When: Ending` can never fire. The persona gets input only when it is on top of the stack, and it goes Ending → Shell at the same moment the ending pops.

**Evidence.** Plan text as quoted. My mini-module host (arch-v2rev-K3/m/internal/proto/host/runner.go) needs a `New func(proto.Launch, proto.Env) (proto.Program, error)` field to compile any push, because it cannot import games or catalog.

**Proposed fix.** In M1, define `host.New(root Program, make func(Launch, Env) (Program, error), s Scheduler)`. Make the persona the only follower of Next: it receives GameOver with Next, emits Launch(*Next), and sets phase Ending when Next.Slug is the ending. The runner never chains.
Add `Slug` to GameOver.
Give Info a `Hidden` (or `Internal`) flag that excludes an entry from Resolve and --games, and register `ending` in the catalog with it. Alternatively, pass an internal-programs map from cmd/wopr.
Drop Ending from Rule.When, or define what input the persona can receive in it.

**Verification.** (a) and (b) are real gaps:

- (a) proto/host may import only proto. The slug→Program factory (signature, unknown-slug error) is never specified.
- (b) `ending` is not among the 16 catalog entries. Registering it as `Listed: false` would leak it into --games "ALSO AVAILABLE" and into Resolve by slug or prefix (§5).

The rest is overstated:

- **"ui is the only package that could build the factory" is wrong.** catalog is "the only place constructors are wired" and may import games packages; the repo's archtest allows catalog → internal/games/.... cmd/wopr → catalog can therefore hand an ending constructor to ui. This is a spec gap, not an impossibility.
- **(c) is answered by the plan.** Launch is commented "persona → host" (PLAN.md:401), and Done delivers GameOver to the program below, which is the persona. So the persona is the only party that can follow Next. It knows which program it launched, so GameOver needs no Slug.
- **(d) is ambiguous, not impossible.** The plan never says Ending → Shell happens at the moment of the pop. Holding Ending for the first input after the pop makes `When: Ending` rules meaningful.

Every remaining fix is a few lines in M1/M2.

**Corrected fix.** In §4.4, define the factory: `host.New(root Program, make func(Launch, Env) (Program, error), s Scheduler)`. ui builds it from the registry plus a catalog-provided internal map, for example `catalog.Internal()` returning {"ending": ending.New}. Alternatively, add a Hidden flag that Resolve, --games and LIST GAMES skip.

State that the persona follows Result.Next by emitting Launch(*Next) and sets phase Ending when it launches the ending.

Define when Ending → Shell happens: on the ending's GameOver, or after the first following input. Keep or drop `When: Ending` to match.

Do not add Slug to GameOver.

### AR-6. games/ending cannot reuse tic-tac-toe (or GTW's board) for the climax; the only ending edge points the wrong way

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. archtest lets `games/ending` import tic-tac-toe, ai and board, and no game import ending; plan §4.1, §6.2.

**Plan text.** §4.1 (PLAN.md:185-186, 226): "internal/games/&lt;game&gt; → proto, prompt, games, games/{ai,cards,board}, games/ending, assets, sim"; "internal/games/ending → proto, prompt, assets"; "ending/ the climax: tic-tac-toe self-play → montage → final dialogue"; §6.2.3: "self-play at increasing speed"

**Problem.** Self-play needs tic-tac-toe's rules, its "perfect minimax" (§6.1 table) and its board drawing. Those live in games/tictactoe (and games/board and games/ai), none of which ending may import. As written, M2 must write a second tic-tac-toe engine inside ending.

If the montage shows the big board as in the film, ending also needs GTW's renderer, which only movie may import.

Meanwhile, nothing needs games/&lt;game&gt; → games/ending: GTW and tic-tac-toe hand off through a slug in Result.Next. And if a game did use that edge, the edge ending needs (ending → tictactoe) would become a cycle.

**Evidence.** Plan text as quoted. §7 already plans an exported GTW "film renderer" for movie (PLAN.md:713), which shows that the drawing code lives in games/gtw.

**Proposed fix.** Replace `games/<game> → games/ending` with `games/ending → games/tictactoe` (and games/board). Move the big-board renderer used by both gtw and the montage into a shared library package, for example games/board/bigboard or assets/render, that ending, gtw and movie may import. Neither tictactoe nor gtw needs to import ending.

**Verification.** The ending row is "proto, prompt, assets" (PLAN.md:186), so ending cannot import games/tictactoe, games/ai or games/board. Its self-play (§6.2.3) must therefore duplicate tic-tac-toe rules and board drawing, or replay precomputed drawn games embedded in assets. The "must write a second engine" claim is slightly strong, because the precomputed-data route exists.

The edge `games/<game> → games/ending` (PLAN.md:185) has no use: §6.2 hands off through Result.Next by slug. If tictactoe used it, the useful reverse edge would become a cycle.

The big-board renderer point is conditional. §6.2 does not say the montage uses the big board, and the M5 viewing pass decides that.

**Corrected fix.** Replace `games/<game> → games/ending` with `games/ending → games/tictactoe` (or games/ending → games/ai and games/board).

Move the big-board renderer into a shared library package (for example games/board/bigboard) that gtw, ending and movie may import, but only if the M5 viewing pass confirms the montage is drawn on the big board.

### AR-7. Movie mode's director does not fit the protocol it is claimed to run under (Esc, pause, Run, Type)

**Severity:** Major. **Verdict:** confirmed. **Also found as:** IM-11, RF-1.

**Status:** Planned. Plan §7 specifies the three M6 host hooks (Esc capture, `Hold`, a drained signal), `Type` as `PaceTyping`, and `Run` as a launched ending; they land in M6.

**Plan text.** §7 (PLAN.md:708-727): "Run{…}: the real games/ending program fed scripted input"; "movie.Director is a proto.Program, so all host rules apply"; controls "Space | Pause / resume", "Esc | Scene menu"; "Type{Text}: the user's line, typed at about 8 chars/s"; §4.3 (PLAN.md:304-321): "While text is revealing, the first key flushes the queue ... In key mode, the key flushes and is delivered", "Esc machine (host-owned; games never see Esc)", "At the Shell, Esc does nothing"; §4.4: KeyEvent "Key: Up Down Left Right Enter Backspace Rune", "The host runner is the only code that interprets it"

**Problem.** 1. Esc cannot reach the director. The host owns Esc, and proto.Key has no Esc value, so the "Scene menu" control is impossible for a proto.Program.

2. Pause cannot pause. The reveal and Wait are timed by the host. Space during a reveal is consumed as a skip (and also delivered), and Wait is "skippable". So pressing Space completes the current line or wait and then stops, instead of freezing it.

3. Run cannot be fed scripted input.

- If the ending is pushed via Launch, only the top program gets input. The director below cannot feed the ending's `Prompt` ("any input, e.g. Hello."), and Space, n, p and Esc go to the ending.
- If the director embeds the ending, it must interpret Done, route ThinkDone and ticks, merge Animate cadences and forward View and SetLayout. That makes it a second host, contrary to §4.4.

4. Type has no Output. Say is WOPR's typewriter, and the user's echo belongs to the host's line editor. Because Output is sealed, M6 must change proto after M1's "gamestest stub exercising every Output" has locked it.

5. The first-contact scene needs the dial, but §4.1 lists `dial` under ui/screens and §4.3 makes "the dial" its own clock consumer. Only §4.2.6 makes it persona Say/Wait steps. If ui owns the dial, the director cannot show it.

**Evidence.** Plan text as quoted. The KeyEvent enum at PLAN.md:380 has no Esc. The skip rules are at PLAN.md:304-308, and Wait is "host-timed pause; skippable" at PLAN.md:392.

**Proposed fix.** Fix the M6 hooks in proto/host now, since M1 locks the protocol:

- an `Env.Root` or `Capture{Esc}` option under which a root program receives Esc as a KeyEvent;
- an `Echo{Text, Pace}` Output for typed user lines;
- a `Hold{On bool}` Output that freezes host pacing and timers.
Cover all three in the M1 stub. Implement Run by letting movie import proto/host and drive a nested Runner whose input comes from the script, so one interpreter is reused. Otherwise, drop Run and make the climax scene pure Say/Wait/Board data.
State that the dial is persona Say/Wait, and remove ui/screens/dial and the separate dial clock consumer.

**Verification.** Points 1-3 and 5 hold against the plan text:

- **(1) Esc.** Esc is host-owned and KeyEvent has no Esc (PLAN.md:315, 380). The repo's proto/key.go says "Esc is not a Key: the host owns it". So the "Esc | Scene menu" control cannot reach a proto.Program.
- **(2) Pause.** There is a deeper root cause than the reviewer stated: no Event tells a program that its queued Say or Wait has finished. A director must emit a scene as one batch, and per §4.3 the first key "flushes the queue" (advance by infinity). Space or n jumps to the end of the scene instead of freezing it.
- **(3) Run.** A Launched ending sits on top and receives input, so the director below cannot feed its Prompt.
- **(5) Dial.** It is a persona Say/Wait in §4.2.6 but ui/screens/dial and a separate clock consumer in §4.1 and §4.3.

Point 4 is refuted. §4.3 says "Movie mode adds a user-typing pace", so Type is Say{Lines, PaceTyping}, and the repo's proto already defines PaceTyping (8 cps). No Echo output is needed.

The urgency argument is also weak. Nothing in the plan locks proto after M1, and Outputs are a sealed set inside proto, so adding one in M6 is additive (the host's switch plus the gamestest stub). The host already has a pause path for TERMINAL TOO SMALL (§4.3), which a Hold output can reuse.

**Corrected fix.** In §7, specify the M6 host hooks now. They can land in M6, since they are additive:

- a root-program Esc capture (Env or SetLayout flag), so the director receives Esc as a KeyEvent, or a host-owned movie menu;
- a `Hold{On bool}` output that reuses the too-small pause of the typewriter, Wait, Animate and Blink;
- a step-completion signal, so the director can emit one step at a time, for example an `OutputsDrained` event or Wait completion delivered as an event.

Use Say{PaceTyping} for Type, with no Echo output.

Implement Run by letting movie import proto/host and drive a nested Runner fed from the script, or drop Run and make the climax scene pure Say/Wait/Board data.

State that the dial is persona Say/Wait data (shared from assets so the director can replay it). Remove ui/screens/dial and the separate dial clock consumer.

### AR-8. Limit is invisible to the host, and Budget still applies in deterministic mode, where -race makes a time-out likely

**Severity:** Major. **Verdict:** confirmed.

**Status:** Fixed. The runner arms only a 60 s safety cap for a deterministic, limited search, and testkit fails a test that reaches it; plan §4.4.

**Plan text.** §4.4 (PLAN.md:396-400, 419-426): "Fn func(ctx context.Context) (any, error)", "Limit Limit // MaxDepth / MaxNodes: binding when Env.Deterministic", "Budget time.Duration // wall-clock cap"; "With Env.Deterministic (--seed, tests, movie mode), searches stop at Limit. Hitting Budget there fails tests"; §6.1 chess (PLAN.md:636): "1.5 s budget ... Deterministic: depth 3 + quiescence (measured median 27 ms, p99 150 ms)"; §9: go test -race on three OSes

**Problem.** Fn receives only ctx, so the host cannot hand Limit to the search. Think.Limit is dead data. "Binding when Deterministic" is really enforced by each game reading Env.Deterministic and capturing its own limit, N times over.

In deterministic mode a wall-clock deadline is still armed, and the plan makes hitting it a test failure. The -race CI jobs run the same depth-3 search several times slower, with packages in parallel, on 3-4 vCPU runners (macos-26, windows-2025). That brings searches close to the 1.5 s budget, which means flaky required checks. When the deadline does hit, Fn "returns its best result as a normal Value", so the transcript silently diverges instead of reporting "budget hit".

**Evidence.** Local experiment (scratchpad arch-v2rev-K3/race; the SZ4 harness on corentings/chess v2.6.0, ID 1..3 + qs self-play, 8 games, 378 searches, idle 4-vCPU Xeon, trivial eval):

- normal build: median 24 ms, p99 141 ms, max 177 ms;
- `go build -race`: median 131 ms, p99 877 ms, max 1.142 s, with 2 searches over 1 s (about 5.8× slower).
The Go race-detector docs say execution time grows 2-20× (<https://go.dev/doc/articles/race_detector#Runtime_Overheads>). A real positional eval would be slower still.

**Proposed fix.** Change the signature to `Fn func(ctx context.Context, lim Limit) (any, error)`. The host passes Think.Limit when Env.Deterministic and a zero (time-governed) Limit otherwise.
In deterministic mode, set no Budget deadline, only a generous hang cap such as 30 s.
When any deadline fires, report it in ThinkDone (for example `BudgetHit bool`), so testkit fails loudly instead of diverging.

**Verification.** `Fn func(ctx context.Context) (any, error)` gets no Limit (PLAN.md:397). Think.Limit therefore has no consumer, and each game must capture its own depth from Env.Deterministic. The plan also keeps the Budget deadline armed when deterministic and makes hitting it a test failure (PLAN.md:399, 423-424).

Local experiment: I rebuilt the reviewer's corentings/chess v2.6.0 self-play harness myself (ID 1..3 + quiescence, 8 games, 378 searches, idle 4 vCPU, go1.27.1).

- Normal build: median 24 ms, p99 158 ms, max 200 ms. This matches the plan's 27/150 ms figures.
- `-race`: median 149 ms, p99 1.022 s, max 1.235 s, with 6 searches over 1 s.

That sits close to the 1.5 s budget before any CI contention from parallel -race packages on 3-4 vCPU runners. The Go race-detector docs give 2-20× overhead (<https://go.dev/doc/articles/race_detector#Runtime_Overheads>).

One sub-claim is wrong: the transcript would not "silently diverge". The plan says a Budget hit "fails tests and is logged", and the runner owns the deadline context, so it can detect ctx.Err()==DeadlineExceeded after Fn returns. A BudgetHit field is optional.

**Corrected fix.** Either change the signature to `Fn func(ctx context.Context, lim Limit) (any, error)`, with the runner passing Think.Limit when Env.Deterministic and a zero Limit otherwise, or drop Think.Limit and document that Fn captures its limit.

In deterministic mode, arm no Budget deadline, only a generous hang cap (for example 30 s).

The runner checks ctx.Err()==context.DeadlineExceeded after Fn returns and surfaces it. testkit fails loudly, and seeded play logs it. A BudgetHit flag in ThinkDone is one way to carry this.

### AR-9. ThinkDone's cancellation contract contradicts the Esc rules, and Think results have no correlation

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. At most one Think per program; a root Esc delivers `ThinkDone{ErrCanceled}`; the proto comment and plan §4.4 state the rules; the snapshot copies History and Last.

**Plan text.** §4.4 (PLAN.md:383): "ThinkDone struct{ Value any; Err error } // Err is context.Canceled only after a confirmed abort"; §4.3 (PLAN.md:318-320): "A second Esc ... cancels the Think context (its late result is dropped by generation) and pops the game with Done{Aborted}", "While a Brain reply is pending, Esc cancels it and prints ** REQUEST CANCELLED **"; §4.4 (PLAN.md:422): "When the deadline is exceeded, Fn returns its best result as a normal Value"; §4.6 (PLAN.md:504-507, 482)

**Problem.** After a confirmed abort the program is popped and the result dropped, so no program ever receives ThinkDone{Err: Canceled}. The persona's Brain cancel is a single, unconfirmed Esc, so by the comment the persona gets nothing either. It cannot clear its pending-reply state, and it cannot decide whether the one-turn offer was consumed.

The "best result as a normal Value" rule contradicts the LLM path, where a timeout must surface as an error so the fallback line fires.

ThinkDone carries no id, and nothing says a program may have only one outstanding Think.

After a cancel, Fn keeps running (cancellation is cooperative) while Handle accepts new input. Yet Snapshot copies only Said and Flags, not History or Last's slices.

**Evidence.** Plan text as quoted. Snapshot's comments (PLAN.md:478-482) mark the maps "copied" but give History []Exchange and Last *proto.Result no copy note.

**Proposed fix.** State the rules:

- at most one Think per program, and a new Think cancels the previous one;
- any cancel that does not pop the program is delivered as ThinkDone{Err: context.Canceled};
- a confirmed abort pops the program and drops its result;
- on Budget expiry, a search returns its best Value, while I/O (Brain/LLM) may return an error.
Deep-copy History and Last into the Snapshot.

**Verification.** The ThinkDone comment says Err is Canceled "only after a confirmed abort" (PLAN.md:383). But a confirmed abort pops the game and drops the late result "by generation" (PLAN.md:318-319), so that delivery never happens. The repo's proto/program.go copies the same comment.

The Brain cancel is a single Esc (PLAN.md:320). If the persona is not told, it cannot emit its next Prompt or clear its pending state.

The plan never limits a program to one outstanding Think, and ThinkDone has no id.

On the Snapshot, the "value copy, safe on another goroutine" claim marks only Said and Flags as copied (PLAN.md:474-482). History and Last share backing storage, which races only if the persona mutates them in place, but the plan should say so.

The LLM-versus-"best result as Value" point is wording: that rule sits in the search context, and §4.6 already makes LLM timeouts errors. Overall these are cheap spec fixes.

### AR-10. Stream IDs need inputs that Env does not carry; the AI stream is not keyed by game or play

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. The host derives each launch's `Env.Seed` from the session seed, slug and play index (`TestLaunchSeeds`); plan §4.4, §4.6.

**Plan text.** §4.4 Env (PLAN.md:368): "Seed uint64 // derive streams with NewRand(Seed, StreamID)"; §4.6 Streams (PLAN.md:540-546): "Game | 2&lt;&lt;56 | hash32(slug)&lt;&lt;16 | playIndex", "AI | 3&lt;&lt;56 | thinkSeq"; §4.4 (PLAN.md:426): "AI randomness comes from a seed captured in Handle"

**Problem.** A game cannot build its Game StreamID, because Env has no playIndex. No one owns thinkSeq.

If each game counts its own Thinks, then for a given --seed, chess and checkers (and every replay of chess within a session) draw the identical AI stream. The domains are disjoint, but the streams within the AI domain collide across games.

If the host derives the seeds instead, Env.Seed is no longer "the seed", and the table misdescribes who builds which ID.

**Evidence.** Plan text: the Env fields are Seed, Width, Height, Instant, Deterministic and Mode only (PLAN.md:367-373).

**Proposed fix.** Have the host derive a per-launch seed, `Env.Seed = NewRand(session, Game|hash32(slug)<<16|playIndex).Uint64()`. Programs then draw AI seeds from their own stream (`AI|thinkSeq` under that per-launch seed). Say this in the Env comment and the Streams table.

**Verification.** Env carries only Seed, Width, Height, Instant, Deterministic and Mode (PLAN.md:367-373). A game cannot know its playIndex, so it cannot build `2<<56 | hash32(slug)<<16 | playIndex` (PLAN.md:543). No one owns thinkSeq.

If each program counts its own Thinks under the shared session seed, every game and every replay draws the identical AI stream. This defeats the stated disjointness within the AI domain.

The repo's M0 Env has the same fields, so nothing resolves this yet.

### AR-11. Full layout plus the front panel needs H+1 rows; the GTW 80×24 mockup has 23 rows

**Severity:** Minor. **Verdict:** confirmed. **Also found as:** IM-8, RF-8.

**Status:** Fixed. Geometry is defined on `H'` (height minus the front panel), and Appendix C is redrawn at exactly 80×24 with a script that checks the counts; plan §4.3.

**Plan text.** §4.3 Layout geometry (PLAN.md:340-342): "Full: the view takes H − 4 rows, then a 3-row strip and the input row. The optional front panel takes the last row. It is shown by default in norad"; Appendix C: "all at exactly 80×24"

**Problem.** The Full layout sums to (H−4) + 3 + 1 = H with no row left for the front panel, which is on by default in the `norad` theme that the GTW board is designed for. At 24 rows the norad Full layout needs 25 rows.

The GTW mockup, which is meant to pin this geometry, has 23 rows and includes a separator plus a panel row. So the registry test "every game fits at 80×24" has no consistent target.

**Evidence.** Local check: an awk row count over the Appendix C code blocks gives 24, 23 and 24 rows. The GTW block (PLAN.md:1488-1510) is the 23-row one, and it ends with a `----` separator and the `DEFCON 3   PORT STAT ...` panel line.

**Proposed fix.** Define Full as `H − 4 − panelRows` (panelRows = 1, or 2 with the separator), and do the same for Panel and Console. Redraw the GTW mockup at exactly 24 rows with the panel shown. Have the 80×24 registry test check both panel on and panel off.

**Verification.** Full is (H−4) + 3 + 1 = H (PLAN.md:340), and the front panel "takes the last row" on top of that (PLAN.md:341). It is on by default in norad, the GTW theme.

Local check: an awk count of Appendix C's code blocks gives 24, 23 and 24 rows. The GTW block (PLAN.md:1488-1510) has 23, and it ends with a `----` separator plus the panel line. That is 2 panel rows where the plan says one.

PLAN.md:134 claims all mockups are "exactly 80×24".

**Corrected fix.** Define each layout against `H − panelRows`, where panelRows is 0, 1, or 2 if the separator counts as part of the panel. State which.

Redraw the GTW mockup at exactly 24 rows with the panel shown.

Have the 80×24 registry test check both panel on and panel off.

## Runtime behaviour and film fidelity (RF)

### RF-1. Movie-mode controls and the Run step cannot be built on the v2 protocol and host rules

**Severity:** Major. **Verdict:** overstated. **Same issue as AR-7**, which carries the status.

**Plan text.** §7 Controls: "Space | Pause / resume", "→ or `n` | Next scene", "Esc | Scene menu"; §7 Data: "`Run{…}`: the real `games/ending` program fed scripted input"; §7 Director: "`movie.Director` is a `proto.Program`, so all host rules apply"; §4.4: "The host runner is the only code that interprets it"; §4.1: "internal/movie → proto, prompt, assets, games/gtw, games/ending"

**Problem.** The director is subject to host rules that defeat its own controls. (1) Esc: the host owns Esc ("games never see Esc"), `KeyEvent.Key` has no Esc value, and outside a game "Esc does nothing". The director never receives Esc, so "Esc → scene menu" cannot work. (2) Space and arrows: playback is almost always revealing text (Type and Say steps), and the skip policy says "the first key flushes the queue" and in key mode "flushes *and* is delivered". So Space first jumps the current text to its end and only then pauses. (3) Pause: nothing in the protocol pauses host-owned pacing. The typewriter queue, `Wait` timers and a running program's `Animate` all keep going. The director can only stop emitting new steps. (4) `Run{}`: the director has no legal way to host the ending. If it calls `ending.Handle` itself, it must interpret the ending's `Prompt`, `Animate`, `Wait`, `SetLayout` and `Done{NoVerdict}`, which §4.4 reserves to the runner. If it emits `Launch{ending}`, the ending sits on top of the stack and waits for the real user at its `Prompt` (no Output lets a lower program inject `LineEvent`s). The director's controls are then dead for the whole climax, which "cannot be aborted with Esc". (5) Scene 5 begins with "Tic-tac-toe with 0 players", whose prompts belong to `games/tictactoe`. The DAG gives movie no edge to that package, so the film text would be duplicated in scene data, which is the drift the consistency test exists to prevent. (6) The `Pace` enum is `PaceSpeech, PaceTable, PaceInstant`, with no user-typing pace for `Type`.

**Evidence.** PLAN.md:380 (`KeyEvent` keys: Up Down Left Right Enter Backspace Rune, no Esc); :315-321 (Esc machine; "At the Shell, Esc does nothing"); :304-307 (skip policy); :388 (Pace values); :356-357 and :427 (only the host interprets Outputs; the runner is pure); :192 (movie DAG has no games/tictactoe); :704 and :714 (scene 5 and Run). Film climax order (tic-tac-toe, "One.", then "Number of players zero"): scratchpad ecolegal/scripts/ss.txt:1792-1803 (springfieldspringfield subtitle transcript) and abs0/wargames.sh@010ed92:557-586.

**Proposed fix.** Fix the movie host contract now, in the §4.4 protocol, rather than in M6. (a) Add a host "show mode" for a root director. In that mode the host delivers Esc, Space, ←/→ and n/p/q to the director as control events, without the skip or flush side effects, and it supports a `Pause{on bool}` Output that freezes the clock consumers, much like the too-small pause. (b) Either add a host-level scripted-input source (e.g. `Feed{Lines}` addressed to the program above) or drop `Run{}` and make scene 5 pure scene data, with the consistency test extended to compare it against `games/ending` and `tictactoe(climax)` output. (c) Add `movie → games/tictactoe` to the DAG, or say scene 5 does not use it. (d) Add `PaceTyping` to the Pace enum.

**Verification.** The core holds. KeyEvent has no Esc (PLAN.md:380), Esc belongs to the host (:315), and outside a game it only skips or does nothing (:308, :321). So 'Esc → scene menu' (:724) cannot be reached while §7 says 'all host rules apply' (:715). No Output pauses the typewriter, Wait or Animate. The too-small freeze (:344-347) is internal to the host. The skip policy flushes the queue on the first key (:304-307), so Space, → and n dump the queued scene before they act. I found one more gap: no Output cancels queued output and no event reports that the queue has drained, so next/prev scene cannot drop the rest of a scene without help from the host. Three parts are overstated. (4) There is a legal route for Run{}. The director is the root and can emit Launch{Slug:"ending", Mode:"movie"}. Env.Mode exists (:372), and the ending can script its own 'Hello.'. What remains true is that the director's controls are dead while the ending sits on top. (5) The tic-tac-toe prompt text can live in assets, and movie → assets is allowed (:192). The consistency test covers only Interactive scenes (:730-731), so scene 5 is unchecked either way. (6) §4.3 already says 'Movie mode adds a user-typing pace' (:299-300). Only the Pace comment at :388 omits it. I keep it at Major because the movie spec the user asked for does not work as written, and the fix belongs in the M1 runner design.

**Corrected fix.** Define the root-director contract in §4.4 now, before M1 builds the runner. (a) Add Esc to proto.Key. For a root non-persona program, the host delivers Esc, Space, ←/→ and n/p/q as KeyEvents without the skip or flush side effects. (b) Add a Pause{On bool} Output that reuses the existing too-small freeze of the typewriter, Animate, Wait and Blink. (c) Give the director a way to drop queued output for next/prev scene: either a Cancel Output, or a 'queue drained' event so it can emit one step at a time. (d) Specify Run{} as Launch{ending, Mode:"movie"}, with the ending supplying its scripted input. While that child is on top, the host routes the control keys to the root director, or §7 states that controls are inactive during the climax. (e) Add the typing pace to the Pace comment. Scene 5's tic-tac-toe lines come from assets, so movie needs no games/tictactoe edge.

### RF-2. Climax tic-tac-toe has undefined exits, and leaving it silently ends the unwinnable GTW

**Severity:** Minor (was Major). **Verdict:** overstated.

**Status:** Planned for M2. Plan §6.2 makes climax tic-tac-toe a closed state machine, allows Esc-Esc with an `AfterAbort` remark, and adds climax goldens.

**Plan text.** §6.2: "`TIC-TAC-TOE` (or any intent naming it) gives `Done{NoWinner, Next: Launch{tictactoe, "climax"}}`." / "`1` plays a game, which ends in `STALEMATE. WANT TO PLAY AGAIN?`." / "`0` (or `ZERO`) gives `Done{NoWinner, Next: Launch{ending}}`."; §4.3: "The ending sequence cannot be aborted with Esc"

**Problem.** `Done` pops GTW before tic-tac-toe starts, so only the 0-players branch reaches the ending. Several inputs at the climax are unspecified, and each falls back to the Shell with the war simply gone, never to be resumed. (1) A 1-player game does not always end in stalemate. WOPR is perfect minimax and only "never loses", so it wins whenever the human blunders, and no line or transition is defined for that case. (2) The plan says nothing about what `WANT TO PLAY AGAIN?` accepts: YES, NO, or `0`/`ZERO`. In the film, ZERO is typed at exactly that prompt. (3) `2` players is undefined. (4) Esc-Esc is allowed in both GTW's climax and tictactoe(climax) because they are games, not "the ending". That one keystroke pair undoes the scene whose whole point is `** ROUTINE MUST COMPLETE BEFORE RESET **`. §14 step 4 and the `film_path` golden cover only the 0-players happy path, so none of this is caught.

**Evidence.** PLAN.md:653-670, :316-321, :640 (tic-tac-toe: "Perfect minimax ... exhaustive 'never loses'"), :1044-1045. abs0/wargames.sh@010ed92 lines 580 (`1`), 583-584 (`STALEMATE.` / `WANT TO PLAY AGAIN?`), 586 (`ZERO` typed at that prompt). The subtitle transcript (ss.txt:1796-1803) shows David playing one-player games before "Number of players zero".

**Proposed fix.** Keep GTW's climax under the tic-tac-toe program (Launch on top of it, not Done plus Next), so that any exit other than 0 returns to the climax and its notices. Specify the climax state machine fully: a line for a WOPR win; YES replays; NO and 2 re-ask the number of players; `0`/`ZERO` is accepted at both prompts. State whether Esc can abort the climax; if it can, the persona's remark should acknowledge the abandoned war. Add goldens for loss, NO and Esc during the climax.

**Verification.** The gaps are real. §6.2 says '`1` plays a game, which ends in STALEMATE' (PLAN.md:658), but tic-tac-toe is minimax that only 'never loses' (:640), so a human blunder gives a WOPR win that has no defined line. Nothing says what WANT TO PLAY AGAIN? accepts, and 2 players is undefined. In abs0, ZERO is typed at that very prompt (wargames.sh@010ed92:583-586). Because of Done+Next (:656), GTW is already popped. Any non-0 exit, including Esc-Esc (§6.1 requires Esc abort for every game; the 'ending' exemption at :321 names only the ending), returns to the Shell, and film_path (:1044-1045) covers only 0 players. Two parts are overstated. These are unspecified branches that M2 can decide without rewriting anything, so the severity is Minor, not Major. Letting a user leave a game with Esc is a defensible UX choice; the plan's flaw is only that it does not say so. The proposed fix is wrong: 'keep GTW's climax under tic-tac-toe (Launch on top of it)' needs games to emit Launch, which §4.4 reserves to the persona (:401). It also undoes the A-16 one-owner hand-off (:644-645).

**Corrected fix.** Keep the A-16 hand-off via Done+Next. Specify tictactoe(climax) as a closed state machine with these parts: a line for a WOPR win; YES replays; NO, 2 or other input re-asks the number of players; 0/ZERO is accepted at both prompts. State whether Esc-Esc may abort GTW's climax and tictactoe(climax). If it may, add a specific AfterAbort remark that acknowledges the abandoned war. Add goldens for WOPR-win, NO and Esc during the climax.

### RF-3. The GTW climax diverges from the film: LIST GAMES is rejected, notices are not tied to inputs, and the launch code has no owner

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Planned for M2. Plan §6.2 maps climax inputs to the film's notices, keeps `LIST GAMES`, gives the launch-code display to `ending`, and prints the map before the side choice.

**Plan text.** §6.2 step 1.4: "Any input other than tic-tac-toe gets a NORAD notice in turn: `** GAME ROUTINE RUNNING **`, `** ROUTINE MUST COMPLETE BEFORE RESET **`, `** ACCESS DENIED **`."; step 1.1: "`SetLayout(Console)`: side choice"; Appendix B: "Launch code `CPE 1704 TKS`"; §7 scene 5: "Tic-tac-toe with 0 players, self-play, montage"

**Problem.** (1) In the film, `List Games` works during the climax. The crew sees the list, tries CHESS, POKER and GTW, and realises tic-tac-toe is "not on the list". That discovery is the beat that leads to TIC-TAC-TOE. v2 answers LIST GAMES with a rotating notice, so the beat cannot happen. (2) The film's notices depend on the input: CHESS gets `** IDENTIFICATION NOT RECOGNISED **` / `** ACCESS DENIED **`, and GTW gets `** GAME ROUTINE RUNNING **`. v2 cycles three notices regardless of input, and never uses `** IDENTIFICATION NOT RECOGNISED **` or `** IMPROPER REQUEST **`, both listed in Appendix B. (3) The launch code is cracked digit by digit while tic-tac-toe is played. Once GTW has been popped, no program owns that display. (4) In the film, the map and the UNITED STATES / SOVIET UNION labels appear above `WHICH SIDE DO YOU WANT?`, but v2's side choice is plain Console with no map. (5) Movie scene 5 starts at 0 players, dropping the notices and the one-player games, which are WOPR terminal text.

**Evidence.** abs0/wargames.sh@010ed92:531-557 (List Games → CHESS, POKER → `** IDENTIFICATION NOT RECOGNISED **` / `** ACCESS DENIED **`; GTW → `** GAME ROUTINE RUNNING **` → IMPROPER REQUEST / ROUTINE MUST COMPLETE / ACCESS DENIED → TIC-TAC-TOE). Subtitles ss.txt:1770-1810 ("List games.", "Put the list back up", "It's not on the list!", "Tic-tac-toe.", "Two numbers" … "Ten! It's got the code"). built1n TRANSCRIPT:32-37 (map, then the labels, then WHICH SIDE). PLAN.md:647-656, :1414-1416, :704.

**Proposed fix.** At the climax, let LIST GAMES print the list (without tic-tac-toe). Map film inputs to film notices (game name → IDENTIFICATION NOT RECOGNISED + ACCESS DENIED; GTW → GAME ROUTINE RUNNING; other input → IMPROPER REQUEST / ROUTINE MUST COMPLETE), with the rotating cycle only as the fallback. Give the code-cracking display an owner, either GTW kept under tic-tac-toe or the climax program. Draw the map on the side-choice screen. Extend scene 5 back to the climax notices.

**Verification.** (1) and (2) are confirmed. abs0 wargames.sh@010ed92:529-557 shows List Games working at the climax: CHESS/POKER → '** IDENTIFICATION NOT RECOGNISED **' and '** ACCESS DENIED **'; GTW → '** GAME ROUTINE RUNNING **', then IMPROPER REQUEST, ROUTINE MUST COMPLETE and ACCESS DENIED, then TIC-TAC-TOE. The subtitles have 'List games… Put the list back up… It's not on the list!… Tic-tac-toe' (ss.txt:1774-1795). v2 sends all input to GTW, which cycles three notices whatever the input (PLAN.md:653-655). Phase-gated LIST GAMES applies only to the persona's Shell (:280-282). (3) is confirmed: Appendix B lists the launch code (:1414-1416), but GTW is popped at TIC-TAC-TOE and no program is named to show the code being cracked. (4) is confirmed as a fidelity gap: built1n TRANSCRIPT:32-37 and abs0:295-300 put the map and the UNITED STATES / SOVIET UNION labels before WHICH SIDE. Console layout does not prevent this, because the art can be printed as Say lines; the plan simply does not specify it. (5) is confirmed: scene 5 starts at 0 players (:704), and no scene contains the climax notices. The plan's tic-tac-toe hint deliberately replaces the discovery beat, so this is fidelity rather than correctness: Minor.

**Corrected fix.** As proposed, with two changes. Print the side-choice map as Console text through Say; no layout change is needed. Give the code-cracking display to games/ending, which already owns self-play and the montage, rather than keeping GTW under tic-tac-toe.

### RF-4. The Sanitize* step order deletes tabs and newlines before mapping them, so pasted lines are glued together

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed in the plan. The M1 code already maps whitespace before dropping controls (tests pin it); plan §4.3 now says so and lists the fuzz properties.

**Plan text.** §4.3 Sanitising: "1. `ansi.Strip`; 2. drop C0, C1 and DEL; 3. map `\t` to a space, and join pasted lines with one space"; "`SanitizeText` …: `ansi.Strip`, drop controls except `\n`, expand tabs to 8-column stops"

**Problem.** TAB (0x09), LF (0x0A) and CR (0x0D) are C0 controls. Step 2 removes them before step 3 can map or join them, and `ansi.Strip` passes them through, so step 2 is the first step to see them. As specified, a pasted `e4\nNf3` becomes `e4Nf3`, `LIST\tGAMES` becomes `LISTGAMES`, and a pasted GTW target list becomes one target, `Las VegasSeattle`. `SanitizeText` has the same bug: it drops tabs before the expansion step, so the M7 "expand tabs" rule never runs. The fuzz targets state no properties that would catch this.

**Evidence.** Local experiment (scratchpad rv2-runtime-film-a7/san/main.go, x/ansi v0.11.8, go1.27.1), implementing the steps in the plan's order: ansi.Strip("a\tb\nc\x1b[31md") = "a\tb\ncd"; SanitizeInput("e4\nNf3") = "e4Nf3"; ("LIST\tGAMES") = "LISTGAMES"; ("Las Vegas\r\nSeattle") = "Las VegasSeattle"; SanitizeText("COL1\tCOL2\nNEXT") = ["COL1COL2" "NEXT"]. PLAN.md:322-330.

**Proposed fix.** Reorder the steps: normalise CRLF, map `\t` to a space and join lines (input), or expand tabs and split on `\n` (text), and only then drop the remaining C0, C1 and DEL. State the fuzz properties: no control runes in the output, every input whitespace run maps to at least one space, and the output is valid UTF-8 and at most 256 grapheme clusters. Also say whether a multi-line paste at the GTW target prompt should become several targets.

**Verification.** PLAN.md:324-330 orders the steps as 'drop C0, C1 and DEL' and then 'map \t … join pasted lines'. SanitizeText does the same: it drops controls except \n before it expands tabs. ansi.Strip keeps C0 bytes. In x/ansi@v0.11.8/width.go:46-48, both PrintAction and ExecuteAction write the byte. Step 2 is therefore the first step to remove \t, \r and \n. Real pastes contain them: ultraviolet's terminal_reader.go:293-308 builds PasteEvent content with '\n' and other control codes. The reviewer's experiment only implements the list literally, but the literal reading is what an implementer following the numbered steps would build. The intent ('join pasted lines') is recoverable, and the fix is cheap, so Minor.

### RF-5. Movie replays are not deterministic as specified, and the e2e and §14 exit behaviour contradict each other

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Planned for M6. The director pins its seed; `-m <scene>` plays to the end and exits 0; the e2e cases are split; plan §7, §9, §14.

**Plan text.** §7: "It runs `Deterministic`, with its own stream, so every replay is identical."; §4.6: "Without `--seed`, the seed comes from `crypto/rand`", Movie StreamID "`4<<56 | scene`"; §9 e2e: "`--movie joshua --instant` → `FINE.` → exit on `q`"; §7: "`LOGOFF` or `q` (in the menu) | Exit 0"; §14.8: "`wopr -m 2 -i` exits 0 at the end of the scene list"

**Problem.** (1) `NewRand(Env.Seed, StreamID)`: the movie's stream fixes only the second half of the seed. Without `--seed`, `Env.Seed` comes from crypto/rand, and `Deterministic` only bounds AI search. So `Type` jitter, the ending's self-play under `Run{}`, and any seeded status or montage choices all change from one replay to the next. (2) §14.8 implies that `-m <scene>` plays on to the end of the list and exits by itself. Under `--instant` there is no pacing at all, so after `FINE.` the director runs scenes 3-5 at once and exits 0. The e2e test's `q` therefore either races the exit, or lands during a scene where `q` is unbound (`q` works only in the menu).

**Evidence.** PLAN.md:537-548, :715-716, :725, :819, :1054, :423.

**Proposed fix.** Have the director pin a constant `Env.Seed` (ignoring or overriding `--seed`) and say so. Choose one end-of-scene behaviour: play only the named scene and exit 0, or continue to the end of the list. Rewrite the e2e case to assert exit 0 at the end without sending `q`, and add a separate menu case that does send `q`.

**Verification.** (1) The movie stream is NewRand(Env.Seed, 4&lt;&lt;56|scene) (PLAN.md:536-545). Without --seed, Env.Seed comes from crypto/rand (:548), and nothing says movie mode pins it. 'Deterministic' only binds AI search to Limit (:371, :423), so the claim 'every replay is identical' (:715-716) has nothing behind it. This is an underspecification rather than a bug. (2) This is a real contradiction. §14.8 says `wopr -m 2 -i` 'exits 0 at the end of the scene list' (:1054), so the director continues past scene 2 and exits on its own. The e2e case '--movie joshua --instant → FINE. → exit on q' (:819) sends q, which is bound only 'in the menu' (:725). After FINE. under --instant, the q either arrives after the process has exited or is ignored, so the case cannot distinguish behaviours.

### RF-6. Is LIST GAMES numbered? R3, §14 and numbered selection disagree with the film list that the movie consistency test checks

**Severity:** Nit (was Minor). **Verdict:** overstated.

**Status:** Fixed. Decision 21: `LIST GAMES` is unnumbered as in the film, a number still selects, and `HELP` and the README say so.

**Plan text.** R3: "numbers its lists"; §2.4: "numbered lists `1.  2.  3.`"; Appendix B: "**Games list** (F). The film order, with a blank line before the last entry" (no numbers); §4.6: "**Numbered selection** is armed by `LIST GAMES`"; §14.3.7: "`list games` → `7` → the chess board"; §7: "`Interactive` (`joshua`, parts of `first-contact`)"

**Problem.** The film's LIST GAMES output has no numbers. If the persona prints it as the film does, "7 → chess" is a hidden feature, and the 15th entry sits after a blank line with nothing showing its number. If the persona numbers it (R3), then the scene-1 movie text is not film-accurate, or the consistency test, which asserts that the persona's lines equal the scene's `Say` lines, fails on the very command that scene shows. The plan never says which rule wins for LIST GAMES, at LOGON or in the Shell.

**Evidence.** PLAN.md:37, :129, :521, :700, :730-732, :1042, :1325-1344. Film list: abs0/wargames.sh@010ed92:372-389 (unnumbered, with a blank line before GLOBAL THERMONUCLEAR WAR); subtitles ss.txt:1774.

**Proposed fix.** Decide explicitly. For example, LIST GAMES prints the film list unnumbered (film, movie and consistency test agree), selection accepts a number with the mapping documented in HELP and README, and `--games` stays numbered. Alternatively, number the Shell list and keep LOGON's list unnumbered, and state which one scene 1 checks. Either way, update §14.3.7.

**Verification.** Appendix B is the 'Canonical screen text' that becomes authoritative in lines.go (PLAN.md:114-115). It gives the games list unnumbered, in film order, with the blank line (:1325-1344), which matches abs0 wargames.sh@010ed92:372-389. An implementer following the canonical text prints it unnumbered at LOGON and in the Shell, so the persona and the scene-1 Say lines agree. The predicted consistency-test failure happens only if someone contradicts Appendix B. What remains is wording. R3 and §2.4 ('numbers its lists', :37, :129) are stated without the exception, and numbered selection ('list games → 7', :1042) is undiscoverable unless HELP or README mentions it.

**Corrected fix.** Add one sentence. LIST GAMES, at LOGON and in the Shell, prints Appendix B's unnumbered film list, as the exception to R3's numbered-list rule. HELP and README say that a list number selects a game. --games stays numbered.

### RF-7. The intent clause rule is ambiguous: either reading gives counterexamples, and the negator rule never fires

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. Intent allows a short list of trailing words, numbers count only as the whole request, and `TestIntentTable` pins every case from the review; plan §4.6.

**Plan text.** §4.6: "that verb is followed by a game number, name, slug or alias, with optional `A GAME OF` / `SOME`"; "no negator (`NOT`, `DON'T`, `DONT`, `NO`, `NEVER`) comes before the verb in that clause"

**Problem.** "Followed by" can mean that the rest of the clause must be exactly a game reference, or only begin with one. If it must be exact, common phrasings fail: "Let's play chess now", "How about chess instead?", "Let's play a game of chess please", "I want to play chess with you". If it only has to begin with one, game numbers produce false starts: "Let's play 2 games of chess" starts Black Jack, "How about 1 more round?" starts Falken's Maze, "Play 3 card monte" starts Gin Rummy. Neither must-not-match list covers numbers. Three more problems: (1) a clause must begin with fillers and then the verb, and no filler is a negator, so the negator rule can never fire; (2) "Shall we play chess", which mirrors WOPR's own catchphrase, and "Now/Alright/Hmm let's play chess" never match; (3) a curly apostrophe (`Let’s`, as pasted from a web transcript) normalises to `LET S` and fails.

**Evidence.** Local experiment: scratchpad rv2-runtime-film-a7/intent.py, a direct model of the §4.6 rules (`python3 -I intent.py`). Both film lines match (15) and all four listed false positives are rejected under both readings. Prefix reading: "Let's play 2 games of chess" → 2, "How about 1 more round?" → 1. Exact reading: "Let's play chess now" → None, "How about chess instead?" → None. Both readings: "Shall we play chess" → None, "Let’s play chess" → None. PLAN.md:508-519, :531-535.

**Proposed fix.** Say that the rest of the clause after the game reference may hold only a short allowlist (NOW, INSTEAD, PLEASE, THEN, WITH ME or WITH YOU), and that a number counts only when it is the whole remainder. Add `SHALL WE PLAY` to the verbs. Map U+2019 to an apostrophe before normalising. Delete the negator rule or make it meaningful (for example, a negator anywhere in the clause vetoes the match). Add the numeric and suffix cases to the intent table tests.

**Verification.** I re-ran the reviewer's model (python3 -I intent.py), and it is a faithful reading of PLAN.md:508-519. Prefix reading: "Let's play 2 games of chess" → 2 (Black Jack) and "How about 1 more round?" → 1. Exact reading: "Let's play chess now" and "How about chess instead?" → no match. Under both readings, "Shall we play chess" and "Let’s play chess" (U+2019) give no match: the rule `[^A-Z0-9']+` turns ’ into a space, giving 'LET S'. The negator rule is dead. A clause must begin with fillers and then a verb, and none of the fillers (:514) is a negator, so a negator before the verb already fails the 'begins with' test. That is harmless redundancy, but it shows the rule was not modelled. The film lines and the four must-not-match cases behave correctly under both readings, so the defect is ambiguity, not breakage.

### RF-8. The Appendix C mockups break the §4.3 layout geometry and the "exactly 80×24" claim

**Severity:** Minor. **Verdict:** confirmed. **Same issue as AR-11**, which carries the status.

**Plan text.** §2.4: "The target screens are in **Appendix C**, all at exactly 80×24"; §4.3: "**Full**: the view takes `H − 4` rows, then a 3-row strip and the input row" / "The optional front panel takes the last row. It is shown by default in `norad`"; §4.3 Thinking: "the input line shows `PROCESSING`"; Appendix C GTW: "original line-segment ASCII art, about 57x7"

**Problem.** (1) The GTW mockup is 23 rows, not 24. (2) Its front panel takes two rows (a dashed separator plus a status line), against "takes the last row". (3) The map box has a 4-row interior while the text says the map is about 57×7, so the real art pushes the screen to 26 rows, and the "registry test checks that every game fits at 80×24" would fail. (4) In Full layout with norad (front panel on by default), H−4 + 3 + 1 + 1 = H+1. The plan never says whether H excludes the front panel. (5) In the chess mockup (imsai, no front panel), `PROCESSING ..` is on row 22, the cursor on row 23, and row 24 is blank. §4.3 instead says the input row is last and that the input line itself shows PROCESSING. It also leaves unclear where the typeahead that is "buffered into the line editor" appears while PROCESSING occupies the input line. (6) The greeting mockup puts one blank line after each user line, but the transcript "made to be as accurate as possible" has two, so at film spacing the exchange does not fit on one 24-row screen.

**Evidence.** Local experiment: counted rows and columns of each Appendix C block with a script. Greeting: 24 rows, max width 49. GTW: 23 rows, max width 79; separator at row 22, status at row 23, cursor at row 21; map box interior rows 4-7. Chess: 24 rows; PROCESSING at row 22, cursor at row 23, row 24 empty. PLAN.md:127, :134, :335-343, :310-313, :1483-1541. built1n TRANSCRIPT:2 and :8-16 (two blank lines after each user line).

**Proposed fix.** Define H as the terminal height minus the front panel when it is shown, and make the front panel exactly one row. Redraw the GTW mockup at 24 rows with a 7-row map (shrinking the trajectory or legend block to fit), and put the input row on the last line in both mockups, with PROCESSING shown as the inline thinking state. State where buffered typeahead is displayed while thinking.

**Verification.** Local experiment: I extracted Appendix C's code blocks with a script and counted them. Greeting: 24 rows, width 49. GTW: 23 rows, width 79. Chess: 24 rows, width 51. (1) and (2) are confirmed: the GTW mockup has 23 rows, and its front panel is two rows (separator at row 22, status at row 23), against 'takes the last row' (PLAN.md:341). (3) is confirmed: the map box has 4 interior rows (rows 4-7) but the text says 'about 57x7', so a 7-row map adds 3 rows and gives 26. (4) is confirmed: Full is H−4 + 3 + 1 = H (:340), so with a front panel it is H+1 unless H excludes the panel, and the plan never says which. (5) is confirmed: chess has PROCESSING on row 22, the cursor on row 23 and row 24 blank. §4.3, however, says the input row is last and the input line shows PROCESSING (:311, :337-339). (6) is weak. built1n TRANSCRIPT puts two blank lines after four of the five user lines but only one after 'Later…'. The console scrolls, so the exchange not fitting on one screen is not a defect.

**Corrected fix.** As proposed. Drop item (6), or reduce it to a note that the film spacing may scroll.

### RF-9. Appendix B text and provenance: the montage list is not abs0's list, "61 HOURS" contradicts other sources, and the call-back scene merges two scenes

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed in the plan. Appendix B tags the five non-abs0 montage names `reconstructed`, splits the call-back from the NORAD session, and uses 28 hours; §2.3 records the third conflict.

**Plan text.** §2.1 tag "`third-party:abs0/wargames@010ed92:wargames.sh` | Copied"; Appendix B: "**Montage scenario names** (A, credited …) … `NATO LIGHT` … `ICELAND INCIDENT` … `MALAYSIAN MANEUVER`. The remainder of the list is completed from the same source in M2."; "OF COURSE. I SHOULD REACH DEFCON 1 AND LAUNCH MY MISSILES IN 61 HOURS."; §7 scene 4 `call-back`: "`IS THIS A GAME OR IS IT REAL?` / `WHAT'S THE DIFFERENCE?`"

**Problem.** (1) The montage list is v1's "reconstructed from fan lists" list relabelled as an abs0 copy. Five of its 45 names are not in abs0 at all: NATO LIGHT, ICELAND INCIDENT (abs0 has ICELANDIC INCIDENT), and the corrected spellings AUSTRALIAN, MALAYSIAN and BURMESE THEATERWIDE. It is also a sparse subsequence of abs0's 157 entries, so the M2 "remainder" is interleaved rather than appended. Tagging these names "Copied" is wrong, and the owner's "credit abs0" decision is not implemented as written. (2) "61 HOURS" comes from abs0 only. The subtitles and the elfuska simulator both say 28 hours, and so does the film's own timeline (52 h remaining in the call-back the day before). Yet §2.3 records only two conflicts. (3) "Is this a game or is it real?" is David's typed line, in mixed case. §7 shows it in WOPR capitals, against §2.2's mixed-case rule. (4) "Call-back" merges two scenes: WOPR phoning David at home (SORRY TO HEAR THAT … TO WIN THE GAME, with GAME TIME ELAPSED / ESTIMATED TIME REMAINING on screen), and the later NORAD terminal session (Joshua → "Are you still playing the game?" → kill ratios → address). Appendix B also omits the elapsed and remaining timers.

**Evidence.** Local experiment: a script diffing PLAN.md's montage names against abs0/wargames.sh@010ed92 (lines 589-791): 157 abs0 names, 45 in the plan, 5 not in abs0, order monotonic, about 98 abs0 entries skipped (e.g. INADVERTENT INCIDENT, U.S. DOMESTIC, NATO ALERT at :614, ICELANDIC INCIDENT at :629). v1 (git show ada63a0:docs/PLAN.md:88-94) has the identical list marked "reconstructed from fan lists". "61 HOURS": abs0 wargames.sh:255 and :501. "28 hours": ss.txt:1129, elfuska wargames.bas:565 and C/imsai8080.c:347. Timer: ss.txt:887-890 and abs0:856. Typed line: abs0 wargames.sh:262 (`actual_d "Is this a game or is it real?"`) and :511. Scene split: abs0 headers "wopr calls back" (:466) and "in data centre" (:494); ss.txt:878-897 vs :1124-1146.

**Proposed fix.** Replace the list with abs0's entries as written, in its order, and record any spelling corrections as edits in NOTICE.md. Alternatively, tag NATO LIGHT and the corrected names `reconstructed` until M5. Record "61 vs 28 HOURS" as a third M5 conflict, with 28 preferred. Split scene 4 into `call-back` (home) and `norad-terminal`, and include the timer lines. Render David's lines as mixed-case `Type` steps.

**Verification.** Local experiment: I diffed the plan against abs0 wargames.sh@010ed92:589-800 (157 entries). Of the plan's 45 names, 40 are in abs0, in monotonic order, with 101 abs0 entries skipped before the last one used. Five are not abs0 text. NATO LIGHT replaces abs0's NATO ALERT at :614. Four are respellings: BURMESE THEATERWIOE :612, AUSTRAILIAN MANEUVER :621, ICELANDIC INCIDENT :629, MAYLASIAN MANEUVER :771. The 'A = Copied' tag is therefore wrong for 5 names. v1 (ada63a0 PLAN.md:88) is the same list, labelled 'reconstructed from fan lists'. '61 HOURS' appears only in abs0 (:255, :501). The subtitles (ss.txt:1129-1130) and elfuska (wargames.bas:565, C/imsai8080.c:347) say 28, and the home call-back shows 52 h remaining (ss.txt:889-890, abs0:856). Yet §2.3 records only two conflicts. The call-back merge is confirmed: abs0 has separate headers for the home scene (:466) and the data centre (:494), and the subtitles show the two scenes far apart (ss.txt:878-897 vs :1124-1146). Appendix B merges them and omits the timer, and §7 scene 4 describes only the NORAD session. David's line in capitals in §7 (:703) is a Nit inside a summary table. The first proposed fix, 'replace the list with abs0's entries', conflicts with owner decision 18, 'Keep the list and credit abs0' (:160).

**Corrected fix.** Keep the list, per decision 18, and fix its provenance. Tag the 40 names that match abs0 as A. Tag NATO LIGHT and the four respelled names `reconstructed` until M5, or record them in NOTICE.md as edits of abs0 entries. Say that completing the list in M2 means re-deriving it in abs0 order, because the current 45 are a sparse subsequence. Record '61 vs 28 HOURS' as a third M5 conflict, with 28 preferred. Split scene 4 into the home call-back, with the GAME TIME ELAPSED / ESTIMATED TIME REMAINING lines, and the NORAD terminal session. Render David's lines in mixed case.

### RF-10. ANSI-16 choices make Dim text unreadable on common palettes (norad Dim on the Linux console; imsai Dim on Solarized)

**Severity:** Minor. **Verdict:** overstated.

**Status:** Fixed. norad Dim moved to `#4A86D6` / index 6, and `TestTextContrast` checks every style in truecolor, xterm and VGA palettes; plan §4.5.

**Plan text.** §4.5 theme table: "`imsai` … Dim `#7A8694` / 8 / Faint"; "`norad` … Dim `#2F6FBF` / 4 / Faint … Background `#02060F` / 0"; §2.4: "**NORAD big board**: dark blue field"

**Problem.** The explicit indices were chosen to stop meaningful styles collapsing, but two of them fail contrast. norad Dim uses index 4 (blue) on painted index 0. That gives 1.58:1 on the Linux console (TERM=linux is one of the cases that gets the ANSI profile), 2.23:1 in xterm and 2.38:1 in Windows Terminal Campbell. Even the truecolor pair is 4.0:1, below WCAG AA's 4.5:1 for text. imsai Dim uses index 8, which Solarized defines as base03, its background colour; on the painted index 0 (base02) that is 1.15:1, effectively invisible. Solarized also remaps 10, 11, 12 and 14 to greys, so the green and amber themes' Text, norad Text and DEFCON 5/4/3 all turn grey in 16-colour mode. Digits and glyphs keep the meaning, but the theme identity is gone. §2.4's "dark blue field" does not match a `#02060F` / index-0 background.

**Evidence.** Local experiment: WCAG contrast computed in Python. #0000AA on #000 = 1.58; #0000ee on #000 = 2.23; #0037DA on #0C0C0C = 2.38; #2F6FBF on #02060F = 4.00; #002b36 on #073642 = 1.15. The Solarized palette table (<https://raw.githubusercontent.com/altercation/solarized/master/README.md>) lists "base03 #002b36 8/4 brblack" and "base02 #073642 0/4 black", with base03 as the dark background. The ANSI profile for TERM=linux and TERM=xterm comes from the review evidence (colorprofile v0.4.3 env detection, scratchpad wfout/gaps-runtime-ux.txt).

**Proposed fix.** For norad Dim in ANSI-16, use 6 (cyan) or 7 instead of 4, and raise the truecolor Dim to at least 4.5:1. Avoid index 8 for any text that carries information; use 7 with Faint, or drop Dim to the Text colour in the ANSI profile. Add a contrast check to the theme tests for every text Style against the background, in both truecolor and the xterm default 16-colour palette. Either fix the norad background or reword §2.4.

**Verification.** Local experiment: I recomputed the WCAG ratios in Python and they match. Index 4 on black: 1.58 (VGA #0000AA), 2.23 (xterm #0000ee). norad Dim truecolor #2F6FBF on #02060F: 4.00. Solarized base03 on base02: 1.15. The ANSI profile is real for TERM=linux and TERM=xterm: colorprofile@v0.4.3 env.go:153-154 sets ANSI for any non-dumb TERM, and only xterm-256color, COLORTERM or a known terminal raises it. Two parts are overstated. The Windows Terminal Campbell figure does not apply, because WT_SESSION forces TrueColor (env.go:177-179). Solarized in 16-colour mode needs a plain TERM=xterm, and Solarized breaks bright-black for every application. The plan never says what Dim carries, so 'unreadable information' is UNKNOWN. '#02060F' is a very dark blue, so the §2.4 wording point is at most a Nit. The norad index-4 choice is still poor on the default xterm, VGA and PuTTY palettes, and the fix is cheap.

**Corrected fix.** Change norad Dim's ANSI-16 index from 4 to 6, or to 7. Use 7 rather than 8 for imsai Dim, or document the Solarized caveat. Add a theme test that checks every text Style's contrast against the background under the xterm and VGA default 16-colour palettes, and under truecolor. Drop the Windows Terminal case, which runs in TrueColor.

### RF-11. §14 step 3 types the GTW line straight after the greeting, but the active greeting scene is dispatched before intent

**Severity:** Nit. **Verdict:** confirmed. **Same issue as IM-7**, which carries the status.

**Plan text.** §4.2.8: "Sanitised input goes, in order, to: host commands … → **active scene** → … → **explicit intent**"; §14.3: "3. `Joshua` → the greeting; 4. `Love to. How about Global Thermonuclear War?` → the chess offer"

**Problem.** After `Joshua` the greeting scene is active, and it is waiting for the three small-talk replies. Because scenes outrank intent, step 4's line, typed right after GREETINGS, would get `HOW ARE YOU FEELING TODAY?` rather than the chess offer. The plan never says whether a scene step consumes any input (the film's echo-replacement behaviour) or only matching input. The answer also decides what the movie consistency test may feed.

**Evidence.** PLAN.md:278-282, :1038-1040. Film order: built1n TRANSCRIPT:6-31 (three user replies precede "Love to…").

**Proposed fix.** Script the three greeting replies in §14.3, and state the rule for off-script input during a scene: either it advances the scene, or intent may interrupt it.

**Verification.** In the dispatch order (PLAN.md:280-282), the active scene comes before explicit intent. §4.2.7 enters 'the greeting scene and Shell' after GREETINGS (:278-279). §14.3 goes straight from 'Joshua → the greeting' to 'Love to. How about Global Thermonuclear War?' (:1038-1039). In the film, three user replies come first (built1n TRANSCRIPT:6-23; Appendix C greeting rows 3-12). The movie consistency test feeds those replies as Type steps (:731), which implies the scene consumes them. Unless the greeting scene consumes only matching input, which the plan never states, step 4 gets HOW ARE YOU FEELING TODAY?. 'The greeting' could be read as including the replies, so this is wording: Nit.

## CI and releases (CR)

### CR-1. The release.yml dry run, an M0 exit criterion, fails as written: `goreleaser release` with no tag at HEAD exits 1

**Severity:** Major. **Verdict:** confirmed. **Also found as:** IM-2.

**Status:** Fixed. The dispatch dry run uses `--snapshot`, and every release job after `verify` states its condition; plan §11.3 and the M0 exit criterion.

**Plan text.** §11.3 step 2: "**`build`** (`contents: read`, no OIDC). `goreleaser release --clean --skip=publish` with release settings"; §11.3: "`workflow_dispatch` runs steps 2–4 as a dry run."; §15 M0 exit criteria: "a `release.yml` dry run passes `build`/`smoke`/`repro`"

**Problem.** A workflow_dispatch run checks out a branch commit, not a tag. GoReleaser in release mode (no `--snapshot`) refuses to build there. In M0 the repository has no tags at all, so the M0 exit criterion cannot be met. After v0.1.0, every dry run on an untagged main commit also fails. Adding `--skip=validate` gets the build through, but it stamps the archives with the previous tag's version (`wopr_0.1.0_*` built from a later commit), which is misleading. The plan also does not say how `build` runs when `verify` is skipped on dispatch. By default a job whose `needs` was skipped is skipped too, so steps 2–4 would not run at all.

**Evidence.** Local experiment with GoReleaser v2.18.2 (go1.27.1, GOTOOLCHAIN=local, no GITHUB_TOKEN) on the exact Appendix A.3 config:

- No tags: `goreleaser release --clean --skip=publish` gives `⨯ release failed ... error=git doesn't contain any tags - either add a tag or use --snapshot`, rc=1.
- Tag v0.1.0 on HEAD~1: `error=git tag v0.1.0 was not made against commit fb1972de...`, rc=1.
- The same run with `--skip=validate`: rc=0, and dist/ contains `wopr_0.1.0_linux_amd64.tar.gz` etc. built from the untagged commit.
- GitHub workflow-syntax docs (jobs.&lt;job_id&gt;.needs): "If a job fails or is skipped, all jobs that need it are skipped unless the jobs use a conditional expression that causes the job to continue."

**Proposed fix.** State that the dispatch dry run uses `goreleaser release --snapshot --clean`, with the same config and the same size, notice and repro checks. The real path runs only on `push` of a `v*` tag. Give `build` the condition `if: ${{ !cancelled() && (needs.verify.result == 'success' || (github.event_name == 'workflow_dispatch' && needs.verify.result == 'skipped')) }}`, and gate `publish` on `github.event_name == 'push'`. Reword the M0 exit criterion to "snapshot dry run".

**Verification.** Nothing elsewhere in the plan says the dispatch dry run uses `--snapshot`. Step 2 says `goreleaser release --clean --skip=publish` with release settings, and the M0 exit criterion needs that dry run to pass before any tag exists (v0.1.0 comes after M2). I reproduced this with GoReleaser v2.18.2 and go1.27.1 under GOTOOLCHAIN=local, on a copy of the A.3 config in a fresh repo with a GitHub remote:

- With no tags, rc=1 and the log says `git doesn't contain any tags - either add a tag or use --snapshot`.
- With v0.1.0 on HEAD~1, rc=1 and the log says `git tag v0.1.0 was not made against commit ...`.
- With `--skip=publish,validate`, rc=0 and dist holds `wopr_0.1.0_*` archives built from the untagged commit.

The workflow-syntax docs (jobs.&lt;job_id&gt;.needs) say: "If a job fails or is skipped, all jobs that need it are skipped unless the jobs use a conditional expression that causes the job to continue." The plan gives no such condition.

The proposed fix is incomplete in two ways.

1. The default implicit `success()` reportedly looks at all ancestors, not only direct `needs`. So `smoke` and `repro` (`needs: build`) would still be skipped once `verify` is skipped. Evidence: docs.github.com expressions says `failure()` "returns true if any ancestor job fails"; reflex-dev/reflex PR #6950 says implicit success() "evaluates over the entire transitive dependency closure"; actions/runner#491 is still open. This is PLAUSIBLE, not confirmed from a primary GitHub source.
2. Under `--snapshot`, the version comes from whatever tags the checkout fetched (see the fetch-depth Nit). If `build` and `repro` use different checkout depths, their checksums differ and `repro` fails.

Corroboration: the repo's own M0 release.yml at HEAD had to invent the `--snapshot` switch and the `always() && ...` condition on `build`. It still leaves `smoke` and `repro` without an `if:`.

**Corrected fix.** On `workflow_dispatch`, `build` and `repro` both run `goreleaser release --snapshot --clean --skip=publish` with the same config, size gate, notice check and checksum diff. On a `v*` tag push they run without `--snapshot`. Give the jobs these conditions:

- `build`: `if: ${{ !cancelled() && (needs.verify.result == 'success' || (github.event_name == 'workflow_dispatch' && needs.verify.result == 'skipped')) }}`
- `smoke` (the job that calls the reusable workflow, where `if` is allowed) and `repro`: `if: ${{ !cancelled() && needs.build.result == 'success' }}`, so a skipped `verify` does not skip them through the implicit success().
- `publish`: `if: ${{ !cancelled() && github.event_name == 'push' && needs.verify.result == 'success' && needs.build.result == 'success' && needs.smoke.result == 'success' && needs.repro.result == 'success' }}`

Use the same checkout (`fetch-depth: 0`) in `build` and `repro`, so snapshot versions and checksums match. Reword the M0 exit criterion to "a snapshot dry run of release.yml (workflow_dispatch) passes build/smoke/repro".

### CR-2. Concurrency rule: main pushes can still be cancelled, and the same block in smoke.yml or scheduled.yml deadlocks or collides

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. Pushes get a per-commit group, release.yml its own prefix, smoke.yml none; plan §11.1.

**Plan text.** §11.1 Common rules: "**Concurrency**, as a block mapping. Pushes to `main` are never cancelled (B-6, B-10): `group: ci-${{ github.ref }}` / `cancel-in-progress: ${{ github.event_name == 'pull_request' }}`"

**Problem.** (1) `cancel-in-progress: false` does not protect pending runs. With the default `queue: single`, if merge A is running and B is pending, merge C cancels B. B's squash commit then has no successful `ci-ok`, so release `verify` refuses to tag it. That fails safe, but the claim "never cancelled" is false and the owner has to re-run CI by hand.
(2) The block sits under "Common rules" next to "`permissions: {}` at the top of each workflow". Copied into the reusable `smoke.yml`, it produces the same group as its caller, because `github.ref` in a called workflow is the caller's ref. GitHub cancels such called workflows as a deadlock, so `smoke` fails and `ci-ok` turns red.
(3) Copied into `scheduled.yml`, which runs on `refs/heads/main`, it shares `ci-refs/heads/main` with ci.yml push runs. Concurrency groups are repository-wide, so a pending CI run and a pending scheduled run can cancel each other.

**Evidence.** - docs.github.com …/control-workflow-concurrency: "By default, any existing `pending` job or workflow in the same concurrency group will be canceled and the new queued job or workflow will take its place." It also documents `queue: single` (default) versus `queue: max`, and "The combination of `queue: max` and `cancel-in-progress: true` is not allowed and will result in a workflow validation error."

- SeleniumHQ/selenium PR #16633 ("[ci]: remove concurrency checks in child workflows"): "child workflows had the same concurrency group so child workflows were getting cancelled due to deadlock".
- The Datadog IaC rule cicd-github-concurrency-limits says the same: reusable workflows should not declare concurrency.

**Proposed fix.** Put concurrency in ci.yml only. Give push runs a group that is unique per commit, for example `group: ci-${{ github.event_name == 'pull_request' && github.ref || github.sha }}`, so main runs never share a group and nothing pending is replaced. Write explicitly "no `concurrency:` in smoke.yml". Give scheduled.yml and release.yml their own prefixes (`sched-…`, `release-${{ github.ref }}`).

**Verification.** All three sub-claims hold up.

1. GitHub's control-workflow-concurrency page says: "By default, any existing `pending` job or workflow in the same concurrency group will be canceled and the new queued job or workflow will take its place." It documents `queue: single` (default, at most one pending) versus `queue: max`. So `cancel-in-progress: false` protects only the running main push, not a pending one, and "never cancelled" is false. This fails safe: release `verify` refuses a commit without a green `ci-ok`.
2. The reusable-workflows docs say "the `github` context is always associated with the caller workflow", and warn not to use the same concurrency group in caller and called workflows. SeleniumHQ/selenium PR #16633 (merged) reports child workflows "getting cancelled due to deadlock" for exactly this setup.
3. The same docs page says group names "must be unique across workflows to avoid canceling in-progress jobs or runs from other workflows".

Sub-claims 2 and 3 only bite if an implementer copies the `ci-` block, which sits under "Common rules". That is plausible but not certain, so Minor is right. The repo's own M0 ci.yml already moved to a per-SHA group for pushes and gave release.yml its own `release-` prefix, which corroborates that the plan text was insufficient. The proposed fix is sound: `github.event_name == 'pull_request' && github.ref || github.sha` evaluates correctly because `github.ref` is never empty.

### CR-3. The reusable smoke workflow has no defined interface, and the release build produces neither per-target artifacts nor e2e binaries

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. `smoke.yml` takes an `artifact` input, `internal/tools/stage` stages binaries with the cross-compiled e2e test for both pipelines, and smoke `chmod +x`es them; plan §11.2.

**Plan text.** §11.2: "`smoke` is a reusable workflow (`workflow_call`), so `release.yml` runs the same job." `build` row: "cross-compiled e2e test binaries. Uploads one artifact per target: `wopr_<os>_<arch>` on `main`, `wopr_<os>_<arch>-pr<N>`". `smoke` row: "Download that target's archive and e2e binary." §11.3: "`dist/` is uploaded." and "3. **`smoke`**: the reusable workflow, run on the release archives themselves."

**Problem.** The mechanics of workflow_call work: the called workflow shares the caller's run, so download-artifact needs no token, and permissions can only narrow. The plan breaks this in two ways.

- **No e2e binary in release.** The release `build` job does not cross-compile the e2e test, so the release smoke, "the same job", has no e2e binary to run. The release gate therefore either fails or silently reduces to the three `--version`/`--games` checks.
- **Two artifact layouts.** CI uploads six per-target artifacts with event-dependent names, while release uploads one `dist/` artifact. No `workflow_call` inputs are defined to tell the shared workflow which layout to fetch.

If the e2e binary is uploaded as a raw file next to the archive, it also loses its executable bit on Linux and macOS.

**Evidence.** - actions/download-artifact README: `run-id` "Optional. Default is ${{ github.run_id }}", and a token is needed only "when downloading artifacts from a different repository or from a different workflow run". The called workflow is part of the same run, so this works.

- actions/upload-artifact README, "Permission Loss": "File permissions are not maintained during zipped artifact upload… all files will have `644`". The archives are fine, because GoReleaser tars `wopr` with mode 0755: local `tar -tvzf` shows `-rwxr-xr-x root/root ... wopr`.
- The §11.3 text lists no e2e compile step in release `build`.

**Proposed fix.** Define `smoke.yml` inputs, e.g. `artifact-prefix` (string), and one producer for both pipelines: an `internal/tools` step that writes `smoke_<os>_<arch>.tar` containing the archive plus `e2e.test[.exe]`. Have the release `build` job run the same e2e cross-compile and upload step as ci.yml's `build`. State that the e2e binary is tarred, or `chmod +x` after download.

**Verification.** The two parts of the plan disagree:

- §9 says "The `build` job cross-compiles the test per target ... and uploads it next to each binary". The ci.yml `build` row lists "cross-compiled e2e test binaries" and per-target artifact names.
- §11.3's release `build` lists GoReleaser, the size gate, the notice check and "`dist/` is uploaded", with no e2e compile. Yet step 3 says smoke is "the reusable workflow, run on the release archives themselves".
- No `workflow_call` inputs are defined anywhere, so one reusable workflow cannot fetch both layouts.

I checked the cited sources at the pinned SHAs:

- actions/upload-artifact README (043fb46, v7.0.1) L462-466: "File permissions are not maintained during zipped artifact upload ... all files will have `644`". It suggests tar plus `archive: false`.
- actions/download-artifact README (3e5f45b) L329: by default it can download only within the current workflow run. A called workflow is part of that run.

Corroboration: the repo's M0 implementation had to add an `artifact` input to smoke.yml and an `internal/tools/stage` step to both pipelines, plus `chmod +x` after download. The plan is underspecified rather than wrong, so Minor stands. The fix is correct; upload-artifact v7's `archive: false` on a tar is the documented way to keep the executable bit.

### CR-4. GoReleaser via `go tool` ties GoReleaser updates to the shipped compiler, costs ~2 min and 415 MB per uncached job, and is outside the weekly report

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. `TestToolModulesFitTheToolchain` enforces the go-line rule, the weekly report covers the tool modules, and the cold-build cost is recorded; plan §10, §11.1.

**Plan text.** §10: "GoReleaser is in `tools/release/go.mod`… run as `go tool -modfile=tools/go.mod <tool>`. Only `tools/release` must not raise the main module's `go` line." §11.1: "setup-go exports `GOTOOLCHAIN=local` itself after installing 1.27.1." and "**Caching.** `cache: false` in `release.yml` and in any job that runs GoReleaser". §11.5 step 2: "update the tool modules with `go get -tool`"

**Problem.** - **Go-line coupling.** GoReleaser sets its `go` directive to the newest Go patch in almost every release (v2.18.2 requires `go 1.27.1`). CI installs exactly the main module's `toolchain`, and setup-go then forces `GOTOOLCHAIN=local`. Any GoReleaser bump made after the next Go patch release therefore breaks ci.yml `build`, release `build` and `repro`. It passes locally, where GOTOOLCHAIN=auto downloads the newer Go. The fix is a `toolchain` bump, so updating the release tool silently changes the compiler of the shipped binaries, up to Go 1.28 once GoReleaser moves there. The plan's sentence frames the constraint backwards and the cadence omits it.

- **Cost.** With `cache: false`, every ci.yml build (on the critical path build → smoke → ci-ok), every release `build` and every `repro` compiles GoReleaser from scratch.
- **Not reported.** scheduled.yml's `go list -m -u all` and `govulncheck` see only the main module, so stale or vulnerable GoReleaser and gitleaks versions never appear in the weekly report.

**Evidence.** - **go-line history.** proxy.golang.org `.mod` files for GoReleaser: v2.14.1 go 1.26.0, v2.14.2 1.26.1, v2.15.3 1.26.2, v2.16.0 1.26.3, v2.17.0 1.26.4, v2.17.1 1.26.5, v2.18.0 1.27.0 (2026-08-23), v2.18.1 and v2.18.2 1.27.1.

- **Failure reproduced.** With real go1.27.1 and GOTOOLCHAIN=local, raising tools/release/go.mod to `go 1.27.2` gives `go: tools/release/go.mod requires go >= 1.27.2 (running go 1.27.1; GOTOOLCHAIN=local)`.
- **Positive control.** Under `go 1.27.1`, `go tool -modfile=tools/release/go.mod goreleaser --version` prints `GitVersion: v2.18.2 GoVersion: go1.27.1`, and `go tool -modfile=… goreleaser release --snapshot --clean` builds six archives whose binary prints the stamped version.
- **Cost measured cold** (fresh GOMODCACHE and GOCACHE, 4 cores): `go get -tool …@v2.18.2` took 42 s and fetched 333 module zips (415 MB, 2.3 GB extracted); the first `go tool goreleaser --version` took 1m27s wall and 4m20s CPU, with a 1.2 GB build cache.
- **Report scope.** `go list -m -u all` in the main module lists no goreleaser line.
- **setup-go source.** setup-go src/main.ts:233-248 calls `core.exportVariable('GOTOOLCHAIN','local')`.

**Proposed fix.** Replace the sentence with a rule: "each `tools/*/go.mod` `go` line must be ≤ the main `toolchain`; bump `toolchain` first, then GoReleaser, in the same PR". Have archtest check it. Add `-modfile=tools/go.mod` and `-modfile=tools/release/go.mod` variants of `go list -m -u` to scheduled.yml. Record the ~2 min per job cost. Optionally allow setup-go caching in ci.yml's snapshot `build` only (not release.yml), keyed on tools/release/go.sum.

**Verification.** The core claims reproduce.

- **Go lines.** On proxy.golang.org, GoReleaser v2.16.0 needs go 1.26.3, v2.17.1 go 1.26.5, v2.18.0 go 1.27.0, and v2.18.1/v2.18.2 go 1.27.1. The `go` line tracks the newest Go patch.
- **The CI failure.** setup-go src/main.ts:233-248 (main 90ad2b3) exports `GOTOOLCHAIN=local`. With real go1.27.1 and GOTOOLCHAIN=local, raising a copy of the repo's tools/release/go.mod to `go 1.27.2` gives `go: tools/release/go.mod requires go >= 1.27.2 (running go 1.27.1; GOTOOLCHAIN=local)`, rc=1.
- **Cost.** I measured a cold GOCACHE with a warm module cache on 4 cores: `go tool -modfile=tools/release/go.mod goreleaser --version` took 1m29s wall, 4m22s user, and left a 1.2 GB build cache. Module download comes on top (reviewer: 42 s, 415 MB), so ~2 min per job is right.
- **Report scope.** The tool modules are separate modules, so `go list -m -u all` and `govulncheck ./...` in the main module never see them.

Overstated: "silently changes the compiler" is wrong. The required toolchain bump is a visible go.mod diff in the same PR, and the plan's 14-day rule already forces toolchain bumps on Go security releases, so the coupling mostly matches existing policy.

The proposed reporting fix has a hole. In tool modules the tools themselves are `// indirect`: tools/release/go.mod:185 `goreleaser/v2 v2.18.2 // indirect`, and tools/go.mod:242 and :261 for gitleaks and x/vuln. A direct-only filter (the finding below), reused with `-modfile`, reports nothing for them. I verified this: `go list -modfile=... -m -u -f '{{... .Indirect ...}}'` gives `indirect=true` for goreleaser, gitleaks, x/vuln and golangci-lint.

**Corrected fix.** Add a rule: "each `tools/*/go.mod` `go` line must be ≤ the main module's `toolchain`; bump `toolchain` first, then the tool, in one PR". Have archtest enforce it.

In scheduled.yml, report tool staleness by tool path, not by the direct-only filter:

```sh
mods=$(go list -modfile=tools/release/go.mod -f '{{.Module.Path}}' tool | sort -u)
go list -modfile=tools/release/go.mod -m -u -f '{{if .Update}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' $mods
```

Do the same for tools/go.mod. I tested both.

Record the ~1.5–2 min cold GoReleaser compile per job. Optionally allow setup-go caching in ci.yml's snapshot `build` only, never in release.yml.

### CR-5. archtest's "running Go == toolchain line" fails on any contributor machine with a newer Go

**Severity:** Minor. **Verdict:** confirmed. **Same issue as AR-2**, which carries the status.

**Plan text.** §4.1: "`internal/archtest` … checks … that the running Go version equals `go.mod`'s `toolchain` line (D-6)." §11.1: "`archtest` asserts the toolchain version."

**Problem.** The check is meant to catch a CI downgrade (D-6). The `toolchain` line is a minimum, though, and GOTOOLCHAIN=auto keeps a local Go that is newer than it. Once Go 1.27.2 ships and the owner upgrades through Homebrew, or anyone runs Go 1.28, `go test ./...` fails locally even though CI is fine. That contradicts "Contributors: `go test ./...` everywhere" (§9).

**Evidence.** Local experiment: a module with `go 1.27` / `toolchain go1.27.0`, run as `GOTOOLCHAIN=auto <go1.27.1>/bin/go run .` with `fmt.Println(runtime.Version())`, prints `go1.27.1`, which differs from the toolchain line. On CI, setup-go installs exactly the toolchain version (src/installer.ts:652-662), so equality holds only there.

**Proposed fix.** Assert `runtime.Version() >= toolchain` everywhere, and equality only when `GITHUB_ACTIONS=true`. That is enough to catch the D-6 downgrade, which would install the lower `go` line.

**Verification.** I ran this locally on a module with `go 1.27` / `toolchain go1.27.0`:

- With the go1.27.1 binary, `go run` of `runtime.Version()` prints `go1.27.1` under both GOTOOLCHAIN=auto and GOTOOLCHAIN=local. A newer local Go is kept, never downgraded.
- With an older local go1.27.0 and `toolchain go1.27.1`, auto switches up and prints go1.27.1.

So equality holds only where exactly the toolchain version is installed: CI via setup-go, src/installer.ts:652-662. §4.1 and §9 put the check in `archtest`, which plain `go test ./...` runs. That contradicts "Contributors: `go test ./...` everywhere" as soon as Go 1.27.2 or 1.28 is installed locally, for example through Homebrew on the owner's macOS 26.

Corroboration: the repo's M0 archtest (internal/archtest/archtest_test.go:248-251) already had to skip the check unless GITHUB_ACTIONS=true.

The proposed fix is correct. A `>=` comparison catches the D-6 failure, where setup-go falls back to the lower `go` line, and stdlib `go/version.Compare` does the comparison.

### CR-6. The weekly staleness report on `go list -m -u all` always has findings, so the tracking issue never closes

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. Direct requirements only for the main module, tools by their `tool` paths, explicit finding and error rules, and a clean week closes the issue; plan §11.5.

**Plan text.** §11.5: "`scheduled.yml`, weekly … It runs: `govulncheck` on `main`; `go list -m -u all` (outdated modules); `prek update --check` (stale hooks); the third-party-notices check. On any finding it opens or updates a single tracking issue."

**Problem.** `go list -m -u all` annotates every module in the build graph, including modules that are never linked (test-only, or listed only in a dependency's go.mod). For a Bubble Tea v2 module there is almost always some `[upgrade]`. The single issue therefore stays open permanently and stops carrying signal, which is the opposite of what R-6 relies on to compensate for dropping Dependabot. Exit codes are also unspecified: `go list -m -u` exits 0 whatever it finds, while govulncheck and `prek update --check` exit nonzero. (`prek update --check` does exist in 0.5.5; it is an alias of `--dry-run --exit-code`.)

**Evidence.** Local experiment: in the scratchpad teacheck module (bubbletea v2.0.10 plus lipgloss v2.0.6), `go list -m -u all` lists 25 modules and 9 carry `[upgrade]` today, including unlinked `github.com/ajstarks/svgo`, `golang.org/x/exp` and `github.com/bits-and-blooms/bitset`, plus `ultraviolet`, `x/sys` and `x/sync`. `/root/.local/bin/prek update --help` (0.5.5) shows `--check  Alias of --dry-run --exit-code`.

**Proposed fix.** Report only direct requirements, for example `go list -m -u -f '{{if and .Update (not .Indirect)}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' all`, plus the tool modules through `-modfile`. Report a toolchain lag (installed toolchain older than the latest go1.27.x from go.dev/dl) as its own finding. Write each check's pass and fail rule into internal/tools so the job can tell findings from noise.

**Verification.** I ran `go list -m -u all` on a copy of the repo's actual go.mod and go.sum (go1.27.1). It exits 0 and lists 26 modules. Eight carry `[upgrade]`: bits-and-blooms/bitset, ultraviolet, x/exp/golden, go-runewidth, xo/terminfo, x/exp, x/sync and x/sys. None of them is a direct requirement; the direct-only template prints nothing today. The plan's report would therefore open an issue on its first run and stay noisy.

`prek update --help` in 0.5.5 shows `--check  Alias of --dry-run --exit-code`, so that check exists and exits nonzero. The plan defines no exit rules or closing behaviour.

The proposed fix needs two caveats:

1. It drops indirect but linked modules such as x/sys, ultraviolet and runewidth. That is acceptable only because govulncheck and the monthly `go get -u` cover them, and the plan should say so.
2. "plus the tool modules through `-modfile`" fails if the same `(not .Indirect)` filter is reused. Tool modules are recorded `// indirect` (tools/release/go.mod:185; verified `indirect=true` for goreleaser, gitleaks, x/vuln and golangci-lint), so they would never be reported.

**Corrected fix.** For the main module, report direct requirements only:

```sh
go list -m -u -f '{{if and .Update (not .Indirect) (not .Main)}}{{.Path}} {{.Version}} -> {{.Update.Version}}{{end}}' all
```

State that indirect upgrades are left to govulncheck and the monthly cadence. Optionally widen this to modules linked into the binary, from `go list -deps -f '{{with .Module}}{{.Path}}{{end}}' ./cmd/wopr`.

For tool modules, select by the `tool` pattern:

```sh
go list -modfile=tools/go.mod -f '{{.Module.Path}}' tool
```

then run `go list -m -u` on those paths. Do the same for tools/release/go.mod.

Define each check's pass and fail rule: govulncheck exit 3, `prek update --check` exit 1, non-empty output from `go list`. Close the issue when a run finds nothing.

### CR-7. CI snapshot builds always report version 0.0.1-snapshot, because the build checkout fetches no tags

**Severity:** Nit. **Verdict:** confirmed.

**Status:** Fixed. Every GoReleaser job checks out with `fetch-depth: 0`; plan §11.1.

**Plan text.** Appendix A.3: `snapshot: version_template: "{{ incpatch .Version }}-snapshot.{{ .ShortCommit }}"`; §11.2 `build` row; §13: "development builds: artifacts from **`main` push runs only** (B-13)"

**Problem.** actions/checkout defaults to `fetch-depth: 1` and fetches no tags. In that checkout GoReleaser falls back to v0.0.0, so every main-branch dev build is `wopr_0.0.1-snapshot.<sha>_*` and its `--version` says 0.0.1 even after v1.0.0. The `incpatch` in the template does nothing in CI.

**Evidence.** Local experiment: `git clone --depth 1 --no-tags` of a repo tagged v0.1.0 and v0.2.0, then `goreleaser release --snapshot --clean`, logs `ignoring errors because this is a snapshot error=git doesn't contain any tags` and `using tags previous=<unknown> current=v0.0.0`, and produces `wopr_0.0.1-snapshot.fb1972d_darwin_amd64.tar.gz`.

**Proposed fix.** In the build job, use `fetch-depth: 0` (or `fetch-tags: true` with enough depth) on the checkout, or drop `incpatch` and use `{{ .ShortCommit }}`-only snapshot naming so nothing implies a version.

**Verification.** The actions/checkout action.yml at the pinned v7.0.1 SHA (3d3c42e) L74-79 sets `fetch-depth` default 1 and `fetch-tags` default false. The plan gives `fetch-depth: 0` only to the `secrets` job.

Local experiment on a repo tagged v0.1.0 and v0.2.0, with HEAD one commit past v0.2.0:

- A `git clone --depth 1 --no-tags` copy running `goreleaser release --snapshot --clean` logs `using tags previous=<unknown> current=v0.0.0` and `version=0.0.1-snapshot.961374c`.
- A full clone gives `wopr_0.2.1-snapshot.961374c_*`.

This is only misleading dev-build naming, so Nit is right. It does interact with the release dry run: if release `build` and `repro` check out at different depths, their snapshot versions, and so their checksums, differ.

**Corrected fix.** Use `fetch-depth: 0` on the checkout in every job that runs GoReleaser: ci.yml `build`, release `build` and `repro`. Snapshot versions then reflect the latest tag and match between `build` and `repro`. The alternative is a snapshot template without `incpatch .Version`, such as `0.0.0-dev.{{ .ShortCommit }}`.

### CR-8. `changelog: use: git` is dead config: releases are disabled and the notes come from `gh --generate-notes`

**Severity:** Nit. **Verdict:** confirmed.

**Status:** Fixed. `changelog: disable: true`; plan §11.3.

**Plan text.** Appendix A.3: `release: disable: true  # publishing is release.yml's publish job (gh + attest)` and `changelog: use: git`; §11.3: "`gh release create vX.Y.Z --draft --verify-tag --generate-notes`"

**Problem.** GoReleaser still generates dist/CHANGELOG.md, but nothing uses it. On the default shallow tag checkout it contains a single commit, which could mislead anyone who picks it up later.

**Evidence.** Local experiment: a shallow `--branch v0.2.0` clone running `goreleaser release --clean --skip=publish` logs `generating changelog`, and dist/CHANGELOG.md contains only `* fb1972d… second`.

**Proposed fix.** Set `changelog: { disable: true }` so the seed reflects the decision that GitHub-generated notes are the release notes.

**Verification.** Local experiment: a `--depth 1 --branch v0.2.0` clone running `goreleaser release --clean --skip=publish` (v2.18.2) logs `generating changelog` and `running against a shallow clone`. dist/CHANGELOG.md contains a single commit line.

Nothing in §11.3 consumes it. `publish` uploads only the archives and checksums.txt, and the notes come from `gh release create --generate-notes`. `use: git` is also GoReleaser's default, so the stanza adds nothing.

I verified the fix: with `changelog: { disable: true }`, `goreleaser check` validates the config, and a release run produces no CHANGELOG.md.

## Tooling (TO)

### TO-1. The lint self-test fixture breaks the lint gate where the plan puts it, and nothing says how the self-test runs

**Severity:** Major. **Verdict:** confirmed. **Same issue as AR-3**, which carries the status.

**Plan text.** §10: "**Self-test.** `config verify` accepts rules that never fire, so `internal/archtest/lintfixture` holds a deliberate violation of each rule. A CI step asserts that each rule reports on it." §4.1: "internal/archtest → stdlib", "Only `ui` imports Bubble Tea." §15 M0 exit: "`ci-ok` green ...; the lint fixture fires every rule".

**Problem.** `internal/archtest/lintfixture` is an ordinary package, so `./...` matches it. Its violations are valid Go, so `go build` and `go vet` pass, but three checks break: (1) `golangci-lint-full` (`golangci-lint run`, which lints `./...`) reports the violations on every run, so the prek gate is permanently red; (2) codespell catches the misspelling planted to test misspell; (3) archtest's `go list -json ./...` sees `internal/archtest/lintfixture → charm.land/bubbletea/v2`, an edge the table forbids. The M0 exit criteria "ci-ok green" and "fixture fires every rule" therefore cannot both hold as written. The plan also never says which golangci-lint the CI step runs. The only pinned copy is the one prek builds into its hook cache, and golangci-lint is not in `tools/go.mod`.

**Evidence.** Local experiment. Scratch module `github.com/GhostofGoes/WOPR` (go 1.27, toolchain go1.27.1, bubbletea v2.0.10) with A.1 and A.2 copied verbatim. A fixture at `internal/archtest/lintfixture` calls `time.Sleep`, `rand.IntN` and `tea.Tick`, imports bubbletea and contains "recieve". Results: `go build ./...` ok and `go vet ./...` ok. golangci-lint v2.14.0 `run ./...` reports 5 issues (depguard 1, forbidigo 3, misspell 1). `prek run --all-files` gives "golangci-lint-full...Failed" and "codespell...Failed: internal/archtest/lintfixture/fixture.go:11: recieve ==&gt; receive". `go list` shows the fixture imports `charm.land/bubbletea/v2 math/rand/v2 time`. Moved to `internal/archtest/testdata/lintfixture`: `go list ./...` no longer includes it; build and vet ok; `golangci-lint run ./...` gives "0 issues."; `golangci-lint run ./internal/archtest/testdata/lintfixture/...` gives the same 5 issues. A manual-stage hook reusing the pinned binary, `{ id = "golangci-lint-full", alias = "lint-fixture", entry = "golangci-lint run --issues-exit-code=0 --output.json.path=lintfixture.json ./internal/archtest/testdata/lintfixture/...", stages = ["manual"], priority = "lint" }`, passes `prek validate-config`. `prek run --hook-stage manual lint-fixture --all-files` wrote JSON whose FromLinter set is {depguard, forbidigo, misspell}.

**Proposed fix.** Move the fixture to `internal/archtest/testdata/lintfixture/`. The `./...` pattern, archtest's `go list ./...` and codespell's `(^|/)testdata/` exclude all skip that path, and it still type-checks in the main module as long as it imports only modules already required. Add the manual-stage `lint-fixture` alias hook shown in the evidence, so the self-test uses the same pinned golangci-lint. Add a small Go checker (for example `internal/tools/lintcheck`) that reads the JSON and asserts one issue per expected (linter, pattern or rule) pair. State this mechanism in §10.

**Verification.** Re-run in a fresh copy of the reviewer's module, with the fixture moved back to the plan's path `internal/archtest/lintfixture`. Results: `go build ./...` and `go vet ./...` ok. `go list ./...` includes `.../internal/archtest/lintfixture`. golangci-lint v2.14.0 `run ./...` exits 1 with 5 issues (depguard 1, forbidigo 3, misspell 1). `prek run --all-files` shows golangci-lint-full Failed and codespell Failed (`internal/archtest/lintfixture/fixture.go:11: recieve ==> receive`). §4.1 says "Import edges not listed below are forbidden", so archtest would also reject the fixture's bubbletea edge. The plan text addresses none of this: §10 L859-860 and §15 M0 L1063 still name the non-testdata path, and §4.1 L234 lists only govulncheck and gitleaks in tools/go.mod, so the self-test binary is unspecified. The live implementation hit the same problem and fixed it the same way. Commit d78df10 put the fixture at `internal/archtest/testdata/lintfixture`, and the package comment says "Go wildcards (./...) skip testdata". It added golangci-lint to tools/go.mod, plus `TestLintRulesFire` and `TestGolangciLintPinsAgree` in internal/archtest/lint_test.go. The reviewer's own fix also works. I validated the manual-stage alias hook with `prek validate-config`. `prek run --hook-stage manual lint-fixture --all-files` wrote JSON with FromLinter {depguard, forbidigo, misspell} (5 issues), and the normal golangci-lint-full hook then passes. Part (2), the codespell hit, applies only if the fixture plants a misspelling; the live fixture does not. Major rather than Blocker: the fix is a path move and nothing built on it needs rewriting.

**Corrected fix.** Either mechanism works; the plan must name one. Option A (the reviewer's): the manual-stage alias hook reusing prek's pinned golangci-lint; add lintfixture.json to .gitignore. Option B (already implemented): golangci-lint/v2 as a `tool` line in tools/go.mod; `WOPR_LINT_SELFTEST=1 go test -run 'TestLintRulesFire|TestGolangciLintPinsAgree' ./internal/archtest` runs `go tool -modfile=tools/go.mod golangci-lint run --output.json.path=stdout --issues-exit-code=0 ./internal/archtest/testdata/lintfixture/`, and a test asserts that the tools/go.mod version equals the prek rev's frozen tag. Either way, update §10 L859-860, §15 M0 and §4.1 L234 (fixture under testdata/, which binary runs it, and golangci-lint in tools/go.mod if B).

### TO-2. The builtin whitespace and end-of-file fixers rewrite golden files

**Severity:** Major. **Verdict:** confirmed.

**Status:** Fixed. The fixers skip `.golden` files and no longer run on pre-push; plan §9, §10.

**Plan text.** A.1: `{ id = "trailing-whitespace", args = ["--markdown-linebreak-ext=md"], priority = "ws" }`, `{ id = "end-of-file-fixer", priority = "eof" }`, neither with an exclude. §9: "`internal/golden.Assert(t, name, got)` stores files under each package's `testdata/`. They are LF-only and CRLF is normalised before comparing." and "snapshots `View().Content` after `ansi.Strip`".

**Problem.** Only codespell excludes `testdata/` and `.golden`. An 80-column TUI snapshot has lines padded with trailing spaces, and it may lack a final newline. On every commit, and in CI's `prek run --all-files`, the fixers strip the padding and append a newline. A freshly regenerated golden therefore fails the gate. Once the hook has rewritten it, the golden test fails, because Assert normalises only CRLF. Both fixers also run on pre-push, because their manifest stages include it and v2 does not override them.

**Evidence.** Local experiment. With lipgloss v2.0.6, `lipgloss.Place(80,3,...)` followed by `ansi.Strip` produces three lines, each len=80 with trailing spaces. With that content in `internal/ui/testdata/logon_joshua.golden`, `prek run --all-files` printed "trim trailing whitespace...Failed - files were modified by this hook / Fixing internal/ui/testdata/logon_joshua.golden" and "fix end of files...Failed ... Fixing internal/ui/testdata/logon_joshua.golden". The SHA-1 went from 8db958d5 to c09738a9, and `cat -A` shows the padding gone and a newline added. prek source (19609fe), crates/prek/src/hooks/builtin_hooks/mod.rs:546-557: trailing-whitespace has `types: text` and stages [PreCommit, PrePush, Manual], with no exclude. `prek list --hook-stage pre-push` lists trailing-whitespace, end-of-file-fixer, check-executables-have-shebangs, check-shebang-scripts-are-executable, check-added-large-files, go-test and govulncheck.

**Proposed fix.** Add `exclude = '(^|/)testdata/|\.golden$'` to fix-byte-order-marker, trailing-whitespace and end-of-file-fixer, or use a top-level `exclude`. Set `stages = ["pre-commit", "manual"]` on the fixers so pushes do not rewrite files. Keep `golden.Assert` byte-exact apart from CRLF, so padding regressions stay visible. If trimming is wanted instead, specify it in §9 and accept that padding bugs will be hidden.

**Verification.** Reproduced. A `.golden` under internal/ui/testdata with padded lines and no final newline went through `prek run trailing-whitespace end-of-file-fixer --all-files`. Result: "trim trailing whitespace...Failed - files were modified by this hook / Fixing internal/ui/testdata/logon.golden", and the SHA-1 changed (15be561b to e3303028). The plan does not address this elsewhere. §9 normalises only CRLF; the live internal/golden/golden.go `normalize` does only `\r\n`→`\n`. §4.5's `Canvas` is a W×H cell grid, so `Canvas.String()` naturally emits padded lines, and padded goldens are near-certain from M1. The fixers really do run on pre-push. prek 0.5.5 source crates/prek/src/hooks/builtin_hooks/mod.rs:546-555 gives TrailingWhitespace `stages: [PreCommit, PrePush, Manual]`, and EndOfFileFixer at :407-419 has the same. These manifest stages win over `default_stages`, and `prek list --hook-stage pre-push` lists both fixers. The fix validated: adding the exclude and `stages = ["pre-commit","manual"]` passes `prek validate-config`, leaves the golden untouched, and drops both fixers from the pre-push list. The live repo has since applied a partial version in commit 56a4f67 (`exclude = '\.golden$'`, no stage change), but plan A.1 is unchanged. Major because hook and test fight each other on every padded golden: the hook trims, `golden.Assert` fails, regenerating restores the padding, and the hook trims again.

**Corrected fix.** `exclude = '\.golden$'` on trailing-whitespace and end-of-file-fixer is enough, because `golden.Assert` always writes `testdata/<name>.golden`. `(^|/)testdata/` is a broader alternative. The pre-push stage override is optional hygiene, not required. Also state in §9 that goldens are byte-exact apart from CRLF, so trailing padding is intentional.

### TO-3. The codespell config fails on the repository's own files: `tools/*.sum` and the word `crasher`

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. codespell excludes every `go.mod`/`go.sum` and accepts `crasher`; plan §10.

**Plan text.** A.1: `{ id = "codespell", args = ["--ignore-words-list", "theaterwide,recognised,falken,wopr,norad,imsai"], exclude = '^(go\.sum|internal/wopr/lines\.go|internal/assets/|internal/movie/scenes/)|(^|/)testdata/|\.golden$', ... }`. §10: "**codespell** skips script and art data with a prek `exclude` regex".

**Problem.** (a) `^(go\.sum|...)` excludes only the root `go.sum`. `tools/go.sum` and `tools/release/go.sum` (§4.1) are scanned, and base64 hash fragments, split on `/` and `+`, produce false positives that can appear after any `go get -tool`. (b) The ignore list lacks `crasher`, the Go fuzzing term that §9 itself uses. The repo's own docs therefore fail the gate at M0. (c) codespell's default dictionaries flag none of the six listed words; RECOGNISED is flagged only with en-GB_to_en-US. So the ignore list, and the film-path excludes as a guard against RECOGNISED, protect against nothing today.

**Evidence.** Local runs with prek-built codespell 2.4.3 and the plan's args. On /home/user/WOPR/docs it reported "docs/PLAN.md:811: crasher ==&gt; crash", "docs/reviews/PLAN-v1-review.md:147: pre-empts ==&gt; preempts" and ":774: crasher ==&gt; crash". `prek run --all-files` with the docs copied in gave "codespell...Failed" (exit 65). GoReleaser v2.18.2's own go.sum:747 (`...otlptranslator v1.0.0/go.mod h1:vRYWnXvI6aWGpsdY/mOT/...`) is flagged "mOT ==&gt; not". I appended that line to both the root go.sum and tools/go.sum and ran `prek run codespell --all-files`: it reported only "tools/go.sum:420: mOT ==&gt; not", so the root file is excluded and the tools file is not. codespell v2.4.3 codespell_lib/_codespell.py:150 has `_builtin_default = "clear,rare"`. Checking the regex with Python `re`: tools/go.sum and tools/release/go.sum give excluded=False.

**Proposed fix.** Use `exclude = '(^|/)go\.sum$|^(internal/wopr/lines\.go|internal/assets/|internal/movie/scenes/)|(^|/)testdata/|\.golden$'`. Add `crasher,crashers` to `--ignore-words-list`, and fix "pre-empts" or exclude `docs/reviews/`. Re-run the gate on the real tree, docs included, before deleting Appendix A.

**Verification.** (a) Python `re` on the plan's regex gives excluded=True for go.sum and False for tools/go.sum and tools/release/go.sum. codespell v2.4.3 (the build from the pinned rev 57b21406) with the plan's args on the live repo's own files reports `tools/go.sum:719,720: decorder ==> decoder` and `tools/release/go.sum:712: mOT ==> not`, exit 65. decorder comes from golangci-lint's decorder linter module, which is now in tools/go.mod. So this is a real false positive in the repository today. The implementer changed the live prek.toml exclude to `(^|/)go\.(mod|sum)$` in 0ed1282, and `prek run codespell --all-files` then passes on a clone. (b) `crasher` is in codespell_lib/data/dictionary_rare.txt, and `_builtin_default = "clear,rare"` is at codespell_lib/_codespell.py:150 in v2.4.3, so it is flagged by default. The v2 docs did contain it (PLAN.md:811 at 5dc712d), but commit 454f6d9 reworded both "crasher" and "pre-empts". The docs part is therefore already fixed; only the ignore-list hardening remains. (c) `recognised` appears only in dictionary_en-GB_to_en-US.txt and the OX dictionary; the other five words are in no dictionary. The ignore list is a no-op today.

**Corrected fix.** The reviewer's regex is correct; `(^|/)go\.(mod|sum)$`, as already used in the live prek.toml, is equivalent for this purpose. The docs rewording is already done (454f6d9), so the remaining change is adding `crasher,crashers` to `--ignore-words-list` (or keep avoiding the word) and updating A.1 to match.

### TO-4. The misspell and `rand.go` exclusions never match, and the RECOGNISED rationale is false

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. Both dead exclusions are removed, so `warn-unused` reports only real staleness; plan §10.

**Plan text.** §10: "**misspell** skips `lines.go`, `assets/` and `movie/scenes/`, because the film's spelling (`RECOGNISED`) is verbatim." A.2 preamble: "the file exemptions work." A.2 rules: `path: ^internal/proto/rand\.go$` and `path: ^internal/(wopr/lines\.go|assets/|movie/scenes/)` with `linters: [misspell]`, under `warn-unused: true`.

**Problem.** With no locale set, misspell uses neutral English and does not flag RECOGNISED, so the misspell exclusion never matches. Per §4.6, `NewRand` builds `rand.New(rand.NewPCG(...))` and takes its seed from crypto/rand. It never calls a banned top-level function, so the `rand.go` exclusion is unused too. Because `warn-unused` is on, both rules log "Skipped 0 issues" on every run, and those lines appear in every failing prek output. That teaches people to ignore the warning that exists to catch stale exclusions. If US-spelling enforcement (`locale: US`) was the intent, the path list is too narrow: the NORAD notice `** IDENTIFICATION NOT RECOGNISED **` (Appendix B) lives with GTW and the ending, not in the excluded paths.

**Evidence.** Local experiment, golangci-lint v2.14.0. On a clean module with RECOGNISED in `internal/wopr/lines.go` and `internal/movie/scenes/joshua.go`, `run ./...` printed "0 issues." plus three warnings: `Skipped 0 issues by rules: [Path: "^internal/proto/rand\\.go$", Linters: "forbidigo"]`, `[Text: "time\\.", Path: "_test\\.go$" ...]` and `[Path: "^internal/(wopr/lines\\.go|assets/|movie/scenes/)", Linters: "misspell"]`. The `_test.go` warning goes away once tests use time.*. With the misspell exclusion removed, nothing is reported. All of the Appendix B/C film text placed in `internal/games/gtw/film.go` also gives 0 issues. Only with `misspell.locale: US` are both RECOGNISED lines flagged. The golangci-lint v2.14.0 .golangci.reference.yml:2271-2275 says misspell's default is "a neutral variety of English".

**Proposed fix.** Delete the misspell exclusion and the RECOGNISED sentence in §10. Alternatively, set `locale: US` on purpose and exclude every film-text location, preferably one film-text package or `//nolint:misspell // film text`. Delete the `rand.go` exclusion, which NewRand does not need, so that `warn-unused` reports only real staleness.

**Verification.** golangci-lint v2.14.0 .golangci.reference.yml:2271-2275 says misspell's default is "a neutral variety of English". With RECOGNISED in internal/wopr/lines.go and internal/movie/scenes/joshua.go and the plan's A.2 config, `run ./...` gives "0 issues." plus `Skipped 0 issues by rules` warnings for the rand.go and misspell rules. Running golangci-lint v2.14.0 with that config on a clone of the live M0 tree prints the same two warnings today. The live internal/proto/rand.go calls only `rand.New(rand.NewPCG(...))`, and neither name is in the forbidigo pattern. With a fresh lint cache and `misspell.locale: US`, only `internal/games/gtw/film.go: RECOGNISED is a misspelling of RECOGNIZED` is reported, because the excluded paths are skipped. That confirms the path list would be too narrow for the NORAD notice (Appendix B L1414). An apparent depguard side-effect of `locale: US` that I first saw was a stale golangci-lint cache entry and disappeared with a fresh GOLANGCI_LINT_CACHE. The A.2 claim "the file exemptions work" is therefore unsupported for these two rules.

### TO-5. The determinism lint misses `math/rand` (v1), whose top-level functions are randomly seeded

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. forbidigo bans every `math/rand` (v1) function and the lint fixture proves it; plan §4.6, §10.

**Plan text.** A.2 forbidigo: `pkg: '^math/rand/v2$'`, `msg: use proto.NewRand(seed, streamID); top-level math/rand/v2 functions cannot be seeded`. §4.6: "`proto.NewRand(seed, id)` is the only sanctioned constructor; forbidigo bans the top-level `math/rand/v2` functions."

**Problem.** `import "math/rand"`, which an editor may auto-import for `rand.Intn`, passes every check. Its top-level functions have been randomly seeded since Go 1.20, and `rand.Seed` has been a no-op since Go 1.24. One stray `rand.Intn` silently breaks `--seed` reproducibility and goldens. archtest's table does not restrict stdlib imports, so it does not catch this either.

**Evidence.** Local experiment, golangci-lint v2.14.0 with the plan's A.2 config: `internal/games/chess/chess.go` calling `mrand.Intn(6)` gives "0 issues." Adding a depguard rule with `deny: [{pkg: "math/rand$"}]` flags "import 'math/rand' is not allowed ..." and does not flag `proto/rand.go`'s `math/rand/v2` import. Go 1.27.1 src/math/rand/rand.go:386: "If Seed is not called, the generator is seeded randomly at program startup"; :398-399: "As of Go 1.24 [Seed] is a no-op."

**Proposed fix.** Add `no-math-rand-v1: {list-mode: lax, files: ["$all"], deny: [{pkg: "math/rand$", desc: "use proto.NewRand (math/rand/v2)"}]}`. The trailing `$` matters: depguard deny entries are prefix matches, so `math/rand` alone would also block `math/rand/v2`. Add a v1 import to the lint fixture.

**Verification.** With the plan's A.2 config, `import mrand "math/rand"; mrand.Intn(6)` in internal/games/chess gives "0 issues.". Adding the proposed depguard rule (`deny: [{pkg: "math/rand$"}]`) flags it and does not flag proto/rand.go's math/rand/v2 import. depguard v2.2.1 is the version golangci-lint v2.14.0's go.mod pins. Its settings.go:224-247 `strInPrefixList` uses exact match when the entry ends in `$` and `strings.HasPrefix` otherwise, so the trailing `$` is required, as the finding says. Go 1.27.1 src/math/rand/rand.go:386 ("seeded randomly at program startup") and :398 ("As of Go 1.24 [Seed] is a no-op") confirm the risk. Nothing else in the plan catches it: staticcheck does not flag rand.Intn, and archtest checks only the module DAG. The live repo has since added a forbidigo rule in 56a4f67 (`pattern: '^rand\..*$'`, `pkg: '^math/rand$'`) and a v1 call in the lint fixture, but plan §4.6, §10 and A.2 are unchanged.

**Corrected fix.** The depguard rule works as stated. The forbidigo alternative already in the live .golangci.yml (`pattern: '^rand\..*$'`, `pkg: '^math/rand$'`) works equally well, with the fixture covering it. The plan text should adopt one of them and mention it in §4.6 and §10.

### TO-6. The lint gate never lints or type-checks the `e2e` build-tagged tests

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. `run.build-tags: [e2e]` in `.golangci.yml`; plan §10.

**Plan text.** §9: "`internal/e2e` (build tag `e2e`) is a Go test that drives the **real binary**...". A.2 has no `run:` block.

**Problem.** golangci-lint and `go vet ./...` load only files that match the default build tags. The whole e2e suite, the xpty driver, waits, timeouts and exit-code assertions, is therefore outside errcheck, forbidigo, revive and misspell. The `build` job does compile it with `-tags e2e`, so compile errors are caught, but no lint rule ever applies to it.

**Evidence.** Local experiment. `internal/e2e/e2e_test.go` has `//go:build e2e`, an unused variable (a compile error), `rand.IntN` and "recieve". golangci-lint v2.14.0 `run ./...` gives "0 issues." `run --build-tags e2e ./internal/e2e/` gives "declared and not used: unusedErr (typecheck)". golangci-lint v2.14.0 .golangci.reference.yml:5010-5014 documents `run.build-tags` with "Default: []".

**Proposed fix.** Add `run: { build-tags: [e2e] }` to `.golangci.yml`. The existing `_test.go` plus `time\.` exclusion then covers the e2e waits. Include an e2e-tagged file in the lint self-test.

**Verification.** Reproduced: `internal/e2e/e2e_test.go` with `//go:build e2e`, `rand.IntN` and "recieve". `go vet ./...` ok; golangci-lint v2.14.0 `run ./...` gives "0 issues."; `run --build-tags e2e ./...` reports the forbidigo and misspell issues. The `time.Sleep` call is correctly exempted by the existing `_test.go` rule. golangci-lint .golangci.reference.yml:5011-5014 documents `run.build-tags` with "Default: []". The plan has no `run:` block, and §11.2 only compiles the suite (`go test -c -tags e2e`). The live implementation independently added `run: build-tags: [e2e]` to .golangci.yml, so plan A.2 just needs to catch up.

### TO-7. The Go tool modules' version rule is misstated, and the weekly report skips the tool modules

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. Same as CR-4.

**Plan text.** §10: "They are pinned and checksummed by `go.sum`, and run as `go tool -modfile=tools/go.mod <tool>`. Only `tools/release` must not raise the main module's `go` line." §11.5 `scheduled.yml`: "`govulncheck` on `main`; `go list -m -u all` (outdated modules); ..."

**Problem.** (a) The binding limit is the root `toolchain` line, not the `go` line, and it applies to both tool modules. setup-go exports `GOTOOLCHAIN=local`, and `go -modfile=X` enforces X's go line. `tools/release` must already declare go 1.27.1 for GoReleaser v2.18.2, which is above the main `go 1.27` but harmless. A `tools/go.mod` raised past 1.27.1 by `go get -tool` fails CI. GoReleaser follows the newest Go patch, so each GoReleaser bump will probably need a toolchain bump first. (b) The weekly checks cover only the root module, so updates and vulnerabilities in the gitleaks, govulncheck and GoReleaser dependency trees are never reported.

**Evidence.** Local experiment. `tools/go.mod` declaring go 1.27 with `tool github.com/zricethezav/gitleaks/v8` (v8.30.1) and `golang.org/x/vuln/cmd/govulncheck` (v1.8.0) works: `go tool -modfile=tools/go.mod gitleaks version` exits 0, printing "version is set by build process", and govulncheck scans the root module ("No vulnerabilities found."). Changing it to `go 1.28` and running go1.27.1 with GOTOOLCHAIN=local fails: "go: tools/go.mod requires go &gt;= 1.28 (running go 1.27.1; GOTOOLCHAIN=local)", exit 1. Go directives from proxy.golang.org: goreleaser/v2 v2.18.2 `go 1.27.1` (v2.19.0 nightlies since 2026-09-06 also 1.27.1); gitleaks v8.30.1 `go 1.24.11`; x/vuln v1.8.0 `go 1.26.0`. actions/setup-go src/main.ts:242-246 exports GOTOOLCHAIN=local. `go list -modfile=tools/go.mod -m -u all` works and lists 143 outdated entries.

**Proposed fix.** Reword the rule: "No tool module's `go` line may exceed the root `toolchain` line. Bump the toolchain first, in its own PR, then the tool." Extend `scheduled.yml` with `go list -modfile=tools/go.mod -m -u github.com/zricethezav/gitleaks/v8 golang.org/x/vuln` and the same for `tools/release/go.mod` (goreleaser). Optionally add `govulncheck -mode=binary` on the built tool binaries. Otherwise, record the gap as accepted.

**Verification.** (a) Local experiment with go1.27.1 and GOTOOLCHAIN=local: `go list -modfile=tools/go.mod -m` fails with "tools/go.mod requires go &gt;= 1.28 (running go 1.27.1; GOTOOLCHAIN=local)" when the go line is 1.28, and also when it is 1.27.2. It succeeds at 1.27.1. actions/setup-go src/main.ts:242-246 exports GOTOOLCHAIN=local, which §11.1 itself acknowledges. So the real limit is the root `toolchain` line, and it applies to every tool module. The plan's sentence is worse than vague. GoReleaser v2.18.2's go.mod says `go 1.27.1`, so the plan's own pinned GoReleaser already sits above the main `go 1.27`, as the live tools/release/go.mod (`go 1.27.1`) shows. The other tools' go lines: gitleaks v8.30.1 `go 1.24.11`, x/vuln v1.8.0 `go 1.26.0`, golangci-lint v2.14.0 `go 1.26.0` (now also in the live tools/go.mod). (b) §11.5 runs `go list -m -u all` and govulncheck on the root module only, and so does the live scheduled.yml. Tool-module updates are reached only by manual-cadence step 2 (`go get -tool`), so this is a reporting gap, not an unmanaged one.

**Corrected fix.** Reword as the reviewer proposes: no tool module's `go` line may exceed the root `toolchain` line; bump the toolchain first in its own PR. In scheduled.yml, also report `go list -modfile=tools/go.mod -m -u <tool module paths>` (include github.com/golangci/golangci-lint/v2 if it moves into tools/go.mod) and the same for tools/release/go.mod, or record the gap as accepted, since manual-cadence step 2 covers the updates.

### TO-8. gitleaks is pinned twice, with two update paths that can drift

**Severity:** Nit. **Verdict:** confirmed.

**Status:** Fixed. `TestToolPinsAgree` checks golangci-lint and gitleaks; plan §10.

**Plan text.** A.1: `repo = "https://github.com/gitleaks/gitleaks"`, `rev = "83d9cd68..."  # frozen: v8.30.1`. §10: "govulncheck and gitleaks are `tool` lines in `tools/go.mod`".

**Problem.** The pre-commit hook builds its own gitleaks from the hook repo, updated by `prek update` with a 14-day cooldown. CI's secrets job uses the `tools/go.mod` copy, updated by `go get -tool` with no cooldown. The two will drift. The go-tool build does not report its version, so the drift is invisible. Contributors also build gitleaks twice.

**Evidence.** Local runs: `prek list` includes the remote gitleaks hook. `go tool -modfile=tools/go.mod gitleaks version` prints "version is set by build process". `go tool -modfile=tools/go.mod gitleaks git --redact --exit-code 1 .` works ("2 commits scanned... no leaks found", exit 0). The hook entry at gitleaks v8.30.1 .pre-commit-hooks.yaml is `gitleaks git --pre-commit --redact --staged --verbose`.

**Proposed fix.** Replace the remote hook with a local one so there is one pin and one update path: `{ id = "gitleaks", name = "gitleaks (staged)", language = "system", entry = "go tool -modfile=tools/go.mod gitleaks git --pre-commit --redact --staged --verbose", pass_filenames = false, stages = ["pre-commit"], priority = "lint" }`.

**Verification.** Confirmed from the plan text. A.1 has a remote gitleaks hook at 83d9cd68 (v8.30.1), updated by `prek update` with a 14-day cooldown. §10 and §11.2 use the tools/go.mod copy, updated by `go get -tool`. gitleaks v8.30.1 version/version.go:5 hard-codes `Version = "version is set by build process"`, and cmd/version.go prints it with no ReadBuildInfo fallback, so a `go tool` build cannot self-report. Two subclaims are overstated. The drift is not invisible, because both pins are readable in prek.toml and tools/go.mod. Contributors build the go-tool copy only if they run that command locally. The live repo now has the same double pin for golangci-lint and handles it with `TestGolangciLintPinsAgree` (internal/archtest/lint_test.go). That is an equally valid fix. The reviewer's local-hook entry matches the upstream manifest (.pre-commit-hooks.yaml: `gitleaks git --pre-commit --redact --staged --verbose`).

**Corrected fix.** Either replace the remote hook with the local `go tool -modfile=tools/go.mod gitleaks git --pre-commit --redact --staged --verbose` hook (one pin), or keep both and extend the existing pin-agreement test to assert that prek.toml's gitleaks `# frozen:` tag equals the tools/go.mod version.

### TO-9. The weekly report keys off exit codes that do not mean what it needs, and the seed is already stale

**Severity:** Nit. **Verdict:** confirmed.

**Status:** Fixed. The report classifies `prek update --check` output, shellcheck-py is bumped, and the rumdl note sits on its own line; plan §10, §11.5.

**Plan text.** §11.5: "`go list -m -u all` (outdated modules); `prek update --check` (stale hooks); ... On any finding it opens or updates a single tracking issue."

**Problem.** `prek update --check` exists and is an alias of `--dry-run --exit-code`. It exits 1 both when updates are available and when a repo fetch fails, so a network failure would file a "stale hooks" issue. `go list -m -u all` always exits 0. The seed is not current today: shellcheck-py v0.11.0.1-1 is past the cooldown, so the first scheduled run will open an issue. `prek update` also strips the "(rumdl-check needs &gt;= v0.2.66)" note from the frozen comment when it rewrites that line.

**Evidence.** Local runs with prek 0.5.5. `prek update --check` on the seed exited 1 with "would update rev `745eface...` (frozen: v0.11.0.1) -&gt; `a6564f5d...` (frozen: v0.11.0.1-1)" and the cooldown notes for golangci-lint and rumdl. On a non-existent repo it exited 1 with "update failed: ... git fetch ... exit status: 128". The prek source crates/prek/src/cli/mod.rs:1034-1038 defines `dry_run`, `exit_code` and "Alias of `--dry-run --exit-code`". `go list -m -u all` printed `[vX]` upgrades and exited 0. `prek update --repo .../rumdl-pre-commit --cooldown-days 0` rewrote the line to `# frozen: v0.2.78` with the note gone.

**Proposed fix.** In the Go report tool, treat "would update" lines as findings and "update failed" as a job error, and parse the `[vX]` brackets from `go list`. Bump shellcheck-py to v0.11.0.1-1 (`a6564f5d57444e123433a5c9af733e3975c77229`) in the seed. Move the rumdl note to its own comment line above `rev`.

**Verification.** prek source crates/prek/src/cli/mod.rs:1038-1040 defines `check` as "Alias of `--dry-run --exit-code`". Local runs with prek 0.5.5: `prek update --check` on the A.1 seed exits 1 with "would update rev `745eface...` (frozen: v0.11.0.1) -&gt; `a6564f5d...` (frozen: v0.11.0.1-1)" for shellcheck-py, so the seed is stale today. On a non-existent repo it also exits 1 with "update failed: ... git fetch ... exit status: 128". `go list -m -u all` exits 0. Starting from rumdl v0.2.75 with the note, `prek update --repo .../rumdl-pre-commit --cooldown-days 0` rewrote the line to `# frozen: v0.2.78` and dropped "(rumdl-check needs &gt;= v0.2.66)". A note on its own line above `rev` survived the same update. The live scheduled.yml already handles `go list` with a `-f` template plus grep, but still treats any non-zero `prek update --check` as a finding. Filing on a fetch error is a misclassification, but the report text shows "update failed", so impact is small.

## Security and legal (SL)

### SL-1. THIRD_PARTY_NOTICES.txt depends on the host OS, but one copy is committed, embedded in all six binaries and checked on three OSes

**Severity:** Nit (was Major). **Verdict:** overstated.

**Status:** Fixed in the plan. The M0 notices tool already takes the union over six targets and accepts `LICENSE*`/`LICENCE*`/`COPYING*`; plan §12 says so.

**Plan text.** §12 Licensing: "`THIRD_PARTY_NOTICES.txt` is generated by `go run ./internal/tools/notices`, from `go list -deps` plus each module's LICENSE and Go's own. It is committed, embedded in the binary for `wopr --licenses`, checked up-to-date in CI, and shipped in every archive." §11.2 `test` row: "`ubuntu-24.04`, `macos-26`, `windows-2025` | `go test -race ./...`; fuzz smoke and `govulncheck` on Linux; the third-party-notices check (§12)."

**Problem.** `go list -deps` applies build constraints for the host's GOOS/GOARCH, and the set of linked modules differs by target. Ultraviolet imports `github.com/charmbracelet/x/termios` only in unix-only files, so Linux and macOS binaries link 18 modules and Windows binaries link 17. As written, the check therefore fails on `windows-2025` against a file generated on Linux. If the file is instead regenerated on Windows, every Linux and macOS archive ships without x/termios's MIT notice. Either way, one committed file cannot be right for all six embedded binaries unless it covers every target. "Each module's LICENSE" is also too narrow: rivo/uniseg ships only `LICENSE.txt`. The M0 tool already has the platform defect: `internal/tools/notices/main.go` runs a plain `go list -deps -json=Standard,Module ./cmd/wopr` with no GOOS/GOARCH loop.

**Evidence.** Local experiment in the scratch module with bubbletea v2.0.10, lipgloss v2.0.6 and chess: `GOOS=<os> GOARCH=<arch> CGO_ENABLED=0 go list -deps -f '{{.Module.Path}}' ./withchessfull` for all six targets gives linux=18, darwin=18, windows=17 modules. The diff is exactly `github.com/charmbracelet/x/termios`, and windows/arm64 matches windows/amd64. Upstream: ultraviolet@v0.0.0-20260811164956-006e29f97886 terminal_unix.go:1 `//go:build darwin || dragonfly || freebsd || linux || ...` with the termios import at :8; winch_unix.go:1 has the same constraint. Module cache: rivo/uniseg@v0.4.7 contains LICENSE.txt and no LICENSE. /home/user/WOPR/internal/tools/notices/main.go generate(): `exec.Command("go", "list", "-deps", "-json=Standard,Module", pattern)`.

**Proposed fix.** Specify that the generator takes the union over the six release targets: loop GOOS/GOARCH with CGO_ENABLED=0, as GoReleaser builds, so the output is the same on any host. Also accept LICENSE*/LICENCE*/COPYING* file names. Add a unit test asserting identical output when the tool runs with GOOS=linux and with GOOS=windows. Run the `-check` once, on Linux, in `test` and in `scheduled.yml`.

**Verification.** The finding's evidence about the M0 tool is stale. It describes the 454f6d9 version of internal/tools/notices/main.go. Since 5f3e30c (and at HEAD 2dba15d), generate() loops a `targets` table of all six GOOS/GOARCH pairs and calls listModules() per target with `CGO_ENABLED=0`, `GOOS=`, `GOARCH=` appended to the env. exec.Cmd keeps the last duplicate, so a host GOOS cannot leak in. It merges everything into one map, so the output is the union and does not depend on the host. licenceNames already includes LICENSE.txt, LICENCE*, COPYING* and lower-case forms, so rivo/uniseg's LICENSE.txt is found. The committed THIRD_PARTY_NOTICES.txt lists both github.com/charmbracelet/x/termios v0.1.1 (unix-only) and x/windows v0.2.2 (windows-only). Two local experiments: (1) `git archive HEAD` into scratch, then `go run ./internal/tools/notices -check`, exit 0; (2) per-target `go list -deps` gives linux 18, darwin 18, windows 17 modules, which confirms the per-target difference that the union absorbs. ci.yml runs the check with `if: runner.os == 'Linux'` only, not on three OSes. The only residue is plan wording: §12 says "from `go list -deps` plus each module's LICENSE", and the §11.2 `test` row does not say the notices check is Linux-only. v2.1 should match the code, since the archtest error text already treats PLAN.md as something kept in sync.

**Corrected fix.** Wording only. In §12, write: "generated from the union of `go list -deps` over the six release targets (CGO_ENABLED=0), with each module's LICENSE*/LICENCE*/COPYING* file and Go's own LICENSE". In §11.2, move the notices check under "on Linux". No code change is needed. An optional unit test of host independence is cheap but not required.

### SL-2. The full-history gitleaks scan covers every branch, has no allowlist procedure, and "fails on any ERR line" has no implementation home

**Severity:** Major (was Minor). **Verdict:** confirmed.

**Status:** Fixed. **gitleaks did not compile** from `tools/go.mod` once golangci-lint joined it; golangci-lint now has its own module. The step uses `--no-color`, fails on errors and empty scans, and scans the checked-out history; SECURITY.md and AGENTS.md have the allowlist and canary procedures; plan §11.2, §12.

**Plan text.** §11.2 `secrets` row: "`fetch-depth: 0`; `go tool -modfile=tools/go.mod gitleaks git --redact --exit-code 1 .` over **full history**. Fails on any `ERR` line (S-1)." §12: "M0 proves the job: a throwaway PR containing a gitleaks test fixture must turn it red before `ci-ok` is made required."

**Problem.** (1) With no `--log-opts`, gitleaks runs `git log -p -U0 --full-history --all`. checkout with `fetch-depth: 0` fetches every branch as `refs/remotes/origin/*`, so the job scans every branch in the repository, not the PR's history. If the M0 canary is pushed as a branch of GhostofGoes/WOPR, which is the natural route for a solo owner, then `secrets`, and with it the only required check `ci-ok`, goes red on every later PR until that branch is deleted. Any stale branch with a finding does the same. (2) `main` forbids force-push, so a finding in `main` history can only be cleared with a `.gitleaksignore` fingerprint. That happens, for example, when a gitleaks bump adds a rule matching an old commit. The plan has no allowlist or rotation procedure, so all merges would be blocked with no documented remedy. (3) Failing on ERR lines is not gitleaks behaviour. It exits non-zero only on findings or a returned error, so a wrapper is needed. §4.1 lists `internal/tools/*` as only sizegate, smoke and notices, and §10 rules out shell logic. (4) M0 also turns on repository push protection. A canary using a GitHub-supported pattern (ghp_, AKIA…) is then blocked at push unless bypassed.

**Evidence.** gitleaks v8.30.1 sources/git.go:92-94: `"log", "-p", "-U0", "--full-history", "--all", "--diff-filter=tuxdb"` when logOpts is empty. cmd/root.go:438-499 (findingSummaryAndExit exits only when err != nil or findings != 0). cmd/root.go:311-318 (.gitleaksignore support). actions/checkout (f548e57) src/ref-helper.ts:69-76: getRefSpecForAllHistory = `+refs/heads/*:refs/remotes/origin/*`, `+refs/tags/*:refs/tags/*`. Local experiment: a bare origin with `main`, `canary` (a ghp_ token fixture) and `feature`; fetched with checkout's refspecs and detached at origin/feature, which has 2 commits. `gitleaks git --redact --exit-code 1 .` then reported "3 commits scanned" and "leaks found: 1", exit=1.

**Proposed fix.** State the following in §11.2 and §12 and in AGENTS.md. Push the canary from a throwaway branch and delete it once the job turns red, or open it from a fork. Use a gitleaks-only rule for the fixture, or run the canary before push protection is enabled. Document a `.gitleaksignore` procedure: rotate the secret, then add the fingerprint by PR. Implement the scan as a Go tool (e.g. `internal/tools/secretscan`) that runs gitleaks with `--report-format json`. It fails on a non-zero exit, on any ERR log line, or on 0 commits scanned. Optionally pass `--log-opts="--full-history HEAD"` so the scan covers exactly the checked-out history (main plus the PR) rather than every branch.

**Verification.** (1) All branches are scanned. gitleaks v8.30.1 sources/git.go:92-94 runs `git log -p -U0 --full-history --all` when --log-opts is empty. actions/checkout v7.0.1 (3d3c42e5, the SHA pinned in ci.yml) src/ref-helper.ts getRefSpecForAllHistory fetches `+refs/heads/*:refs/remotes/origin/*`. Local experiment: a bare origin with main, canary (a high-entropy fake ghp_ token) and feature, fetched with checkout's refspecs and detached at origin/feature. gitleaks reported "4 commits scanned", "leaks found: 1", exit 1. With `--log-opts="--full-history HEAD"` it scanned 2 commits, found no leaks, exit 0. A canary branch left behind therefore turns `secrets` and `ci-ok` red on every PR. GitHub's auto-delete of head branches fires only on merge, not on close. (2) There is no allowlist or rotation procedure. .gitleaksignore is supported (cmd/root.go:311-322), but neither PLAN.md, AGENTS.md nor SECURITY.md mentions it. (3) The finding understates this point. M0 does give the ERR rule a home, as inline bash in ci.yml (`... | tee gitleaks.log; if grep -q ' ERR ' gitleaks.log; then exit 1`), but that grep can never match in CI. gitleaks logs through zerolog's ConsoleWriter, which colours output unless --no-color or NO_COLOR is set, with no TTY detection. Local experiment: `gitleaks git --redact --exit-code 1 --log-opts="x..y" . 2>&1 | tee log` printed `ESC[31mERRESC[0m [git] fatal: ambiguous argument...`, "0 commits scanned", pipe exit 0, and `grep -q ' ERR '` did not match. With `--no-color` or `NO_COLOR=1`, plain ` ERR ` appears and the grep matches. That is S-1's failure mode, a scan that errors yet passes. (4) New, adjacent: the job's tool does not compile at HEAD. Local experiment on a `git archive HEAD` copy: `go tool -modfile=tools/go.mod gitleaks version` fails with `x/cellbuf@v0.0.13-0.20250311204145-2c3ea96c31dd/cell.go:220: b.CurlyUnderline undefined (type ansi.Style ...)`. Cause: M0 commit d78df10 added golangci-lint v2.14.0 to tools/go.mod, which is not in the plan's §4.1/§10 list. `go mod graph` shows it pulling charm.land/lipgloss/v2 v2.0.6 and x/ansi v0.11.8, which breaks gitleaks's lipgloss v1.1.0 → cellbuf. gitleaks v8.30.1 alone, in an isolated module, builds fine, and govulncheck from the same tools module still builds. Consequences: the `secrets` job is red on every run. The M0 exit criterion "the gitleaks canary PR turned red" would be met by the build failure, proving nothing. (5) The push-protection point is weak. Push protection allows a bypass with a reason, and it only blocks a canary that uses a supported pattern, so it is an ordering note, not a defect. The Go-tool wrapper the finding proposes is optional: the secrets job is Linux-only, so a bash step does not break §10's Windows-portability rationale.

**Corrected fix.** (a) Make the gitleaks build work. Move golangci-lint out of tools/go.mod (prek already pins it; give it its own modfile, e.g. tools/lint/go.mod, if `go tool` access is wanted), or otherwise make the selected x/ansi compatible. Add a CI step that runs `go tool -modfile=tools/go.mod gitleaks version`. (b) Run gitleaks with `--no-color --no-banner` (or `NO_COLOR=1`) so the ERR check can match. Better: write `--report-format json --report-path gitleaks.json` and fail on a non-zero exit, on any ERR line, or on "0 commits scanned". (c) Decide the scope explicitly. Either keep `--all`, so every public branch is scanned, and document that a stale branch with a finding blocks ci-ok until it is deleted or ignored; or pass `--log-opts="--full-history HEAD"`. (d) Canary procedure: push it from a throwaway branch or a fork, require the log to show "leaks found" (not a build or other error) before calling it red, then delete the branch. Run it before enabling push protection, or bypass with a reason. (e) Add a .gitleaksignore procedure to SECURITY.md and AGENTS.md: rotate the secret, then add the fingerprint by PR.

### SL-3. No planned package can embed the root-level LICENSE, NOTICE.md and THIRD_PARTY_NOTICES.txt; M0 already added an unlisted root package

**Severity:** Nit (was Minor). **Verdict:** overstated.

**Status:** Fixed in the plan. archtest already had the root rule; plan §4.1 lists `legal.go`.

**Plan text.** §4.1: "Import edges not listed below are forbidden (A-6, A-13)." and "cmd/wopr → cli, ui, version, games/catalog, (llm in M7)"; §12: "embedded in the binary for `wopr --licenses`"; Appendix A.3 archives `files: - src: NOTICE.md ... - src: THIRD_PARTY_NOTICES.txt` (repository root).

**Problem.** `//go:embed` patterns may not contain `..`, so `cmd/wopr` cannot embed files at the repository root. Embedding them needs a package in the root directory, or the files must move. The DAG lists no root package, and `archtest` rejects any edge not in the table. The M0 commit therefore added an unlisted package `wopr` (legal.go) at the root, and `archtest` will fail as soon as `cmd/wopr` imports it for `--licenses`.

**Evidence.** go1.27.1 src/embed/embed.go:68: "Patterns may not contain ‘.’ or ‘..’ or empty path elements". /home/user/WOPR/legal.go (commit 454f6d9: "a root package that embeds them plus THIRD_PARTY_NOTICES.txt") embeds LICENSE, NOTICE.md and THIRD_PARTY_NOTICES.txt. PLAN.md:171-198 has no root-package row.

**Proposed fix.** Add `github.com/GhostofGoes/WOPR (legal.go) → stdlib` and `cmd/wopr → (root)` to the §4.1 table, and therefore to archtest. Alternatively, move the three files to `internal/legal/` and point the GoReleaser archive `files` at them with `dst:` names.

**Verification.** The embed restriction is real: go1.27.1 src/embed/embed.go says patterns may not contain '..'. A root package is therefore needed, and PLAN.md §4.1 does not list one. The claimed consequence is false, though. internal/archtest/archtest_test.go at HEAD already has the rule `{".", nil}` for the root package, and cmd/wopr's rule allows ".". cmd/wopr/main.go already imports `wopr "github.com/GhostofGoes/WOPR"` and prints wopr.License, wopr.Notice and wopr.ThirdPartyNotices for --licenses. Local experiment on a `git archive HEAD` copy: `go test ./internal/archtest -run TestImportDAG` passes. So archtest does not fail. The only gap is that the plan's DAG table is behind the authoritative archtest table. Its failure message itself says "add it to the DAG in internal/archtest and docs/PLAN.md".

**Corrected fix.** Add `github.com/GhostofGoes/WOPR (legal.go) → stdlib (embed)` to the §4.1 table and `.` (root) to the cmd/wopr row, mirroring archtest. Moving the files to internal/legal is unnecessary.

### SL-4. The provenance test checks only that a tag exists: third-party tags are not checked against NOTICE.md, `prompt` exclusion is not enforced, and film text transcribed by abs0 cannot be both credited and excluded from the MIT grant

**Severity:** Nit (was Minor). **Verdict:** overstated.

**Status:** Fixed. The provenance test restricts tags and checks third-party tags against NOTICE.md; plan §2.1 states that rule and keeps `prompt` text out of the repository; NOTICE.md says abs0's licence covers the transcription only.

**Plan text.** §2.1: "Every script line, scene step and asset carries one. A test fails on any entry without a tag." Tag table: "`third-party:abs0/wargames@010ed92:wargames.sh` | Copied; BSD-2 notice in `NOTICE.md`" and "`prompt` | ... Excluded from builds until he grants written permission **and** a licence". §7: "the quotations are listed in `NOTICE.md` as excluded from the MIT grant".

**Problem.** (1) Any `third-party:<x>` string passes the test. A line honestly tagged as coming from built1n (GPL-3.0/CC BY-SA) or elfuska (all rights reserved) would therefore pass with no NOTICE.md entry. (2) Nothing checks that `prompt` lines are kept out of release binaries. If they are Go data filtered at runtime, they are still compiled into the binary. (3) A line has exactly one tag, but the abs0 items are film screen text that abs0 transcribed. Brownlee's BSD-2 grant can cover only his transcription, not MGM's text, so these items must be credited to abs0 and also excluded from the MIT grant. Once M5 confirms them against the film, retagging them `film` drops the credit, and leaving them `third-party` keeps them out of the film-quotation set. The M0 NOTICE.md shows the result: the two items "are used under its BSD 2-clause licence", which presents MGM's screen text as BSD-licensed.

**Evidence.** PLAN.md:88-97, 734-736, 1075-1077. /home/user/WOPR/NOTICE.md, section "Third-party text". github.com/abs0/wargames@010ed92 wargames.sh:3-28 (the BSD-2 header) and :170-173: the backdoor header, which the film shows after `Joshua` (abs0 phase_falken; built1n joshua.c:40 credits that text to Brownlee and Carter).

**Proposed fix.** Split provenance into two fields: `Source` (film | reconstructed | original | prompt) and an optional `Via` (a third-party reference). The test checks `Source` against an enum and `Via` against a table of credited sources parsed from NOTICE.md. Allow `prompt` entries only in files behind a build tag such as `//go:build wopr_prompt`, and add a test that the default build contains none. Reword NOTICE.md so the abs0 items are film text credited to abs0's transcription. Have NOTICE.md reference the tagged data, as M0 does, rather than list the quotations by hand.

**Verification.** (1) This is refuted by the implemented test. internal/wopr/persona_test.go TestEveryLineHasProvenance (HEAD 2dba15d) accepts only film, reconstructed and original, or a `third-party:` tag whose repo name appears in NOTICE.md. A built1n or elfuska tag would fail, because NOTICE.md mentions neither. (2) This is effectively enforced. lines.go defines no `prompt` constant and the test's switch rejects any other value, so a prompt-tagged line cannot pass. I also checked that every `line*` var in lines.go is listed in allLines (comm diff empty), but that list is maintained by hand. The finding's build-tag fix is itself wrong. A `//go:build wopr_prompt` file is still committed to a public repository and shipped in the module zip that proxy.golang.org serves to `go install` users. That is distribution of the brother's text without permission, which L-3/L-5 forbid. (3) This is mostly a wording point. NOTICE.md's Code section excludes both "Film text" and "Third-party text" from the MIT grant, so the abs0 items are excluded either way and are not mislicensed. M5 converts only `reconstructed` lines, so the abs0 credit is not lost. "used under its BSD 2-clause licence" could say that the licence covers the transcription and that the screen text itself remains the film's. Remaining real gaps: the plan states a weaker rule than the code, and the same test must be extended to internal/assets and internal/movie/scenes when they land.

**Corrected fix.** Update the §2.1 wording to the implemented rule: tags are restricted to an enum, and a third-party tag must name a source credited in NOTICE.md. Apply the same test to assets and movie scenes. State that prompt-derived text is not committed to the public repository at all until the brother licenses it, rather than putting it behind a build tag. In NOTICE.md, add one clause saying the abs0 items are film screen text, that abs0's BSD-2 licence covers only the transcription, and that the text remains excluded from the MIT grant like other film text. Do not split the tag into two fields.

### SL-5. "Exactly two text items" from abs0 leaves out the status burst, which Appendix B tags as abs0 but whose wording matches the all-rights-reserved elfuska source

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. The burst uses our own two-column layout without the copied spacing, tagged `reconstructed`; plan §2.1 and Appendix B.

**Plan text.** §2.1: "Exactly two text items are taken from `abs0/wargames` under BSD-2 ...: the backdoor header block; the montage scenario names." Appendix B: "followed by a status burst (A/F), e.g. `SYSPROC FUNCT READY  ALT NET READY`, `CPU AUTH RY-345-AX3  SYSCOMP STATUS: ALL PORTS ACTIVE`, `(311) 936-2364`".

**Problem.** Appendix B tags the status burst A (abs0). That makes it a third abs0 item, missing from §2.1's list and from NOTICE.md, which in M0 names two items. Its punctuation also does not match abs0. `SYSCOMP STATUS: ALL PORTS ACTIVE`, with a colon, is how elfuska/zompiexx's wargames.bas ("(C) 2020, ALL RIGHTS RESERVED") has it. abs0 has `SYSCOMP STATUS  ALL PORTS ACTIVE` and built1n has `SYSCOMP STATUS ALL PORTS ACTIVE`. The tag is wrong whichever source the line came from.

**Evidence.** github.com/abs0/wargames@010ed92 wargames.sh:179 `(311) 936-2364`, :183-184 `SYSPROC FUNCT READY ... ALT NET READY` / `CPU AUTH RY-345-AX3            SYSCOMP STATUS  ALL PORTS ACTIVE`. elfuska/wargames wargames.bas BASIC line 4020: `PRINT "CPU AUTH RY-345-AX3     SYSCOMP STATUS: ALL PORTS ACTIVE"`; line 20: `(C) 2020, ALL RIGHTS RESERVED`. built1n/wargames joshua.c:44. /home/user/WOPR/NOTICE.md lists two items.

**Proposed fix.** Either list the status burst as a third abs0 item (using abs0's exact text) in §2.1 and NOTICE.md, or tag it `reconstructed` with no third-party claim and settle its wording in the M5 viewing pass. Keep the colon variant only if the film shows it.

**Verification.** I checked the facts against the clones of each source at the cited commit. abs0/wargames@010ed92 wargames.sh phase_falken has `CPU AUTH RY-345-AX3            SYSCOMP STATUS  ALL PORTS ACTIVE`, with no colon. built1n joshua.c:44 has `SYSCOMP STATUS ALL PORTS ACTIVE`. elfuska wargames.bas line 4020/4200 and C/imsai8080.c:494 have `CPU AUTH RY-345-AX3     SYSCOMP STATUS: ALL PORTS ACTIVE`, and wargames.bas line 20 says `(C) 2020, ALL RIGHTS RESERVED`. Appendix B tags the burst "A/F", which contradicts §2.1's "Exactly two text items". The M1 implementation has since retagged it: internal/wopr/lines.go lineBurst = recon(...), commented "the film's phrases in our own layout". But its line `CPU AUTH RY-345-AX3     SYSCOMP STATUS: ALL PORTS ACTIVE` is byte-identical to elfuska's, including the five spaces. So the provenance claim is still inaccurate in code. Legal exposure is negligible: it is one line of film screen text, and a verbatim transcription gives elfuska little or no protectable expression. It stays Minor rather than Nit because it is a factual provenance error in a system whose purpose is accurate provenance, and it is cheap to fix. Whether the film shows the colon is UNKNOWN until the M5 viewing pass.

**Corrected fix.** Change Appendix B's status-burst tag from A/F to F (reconstructed). Until M5, use a layout not copied from elfuska, e.g. abs0's or built1n's spacing without the colon, or record in the line's comment that the wording matches elfuska and must be settled in M5. Keep the colon only if the film shows it. Do not add a third abs0 item.

### SL-6. Squash-only merges keep v1 off `main`, but not out of the public repository, which is what the CC BY-SA concern is about

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. NOTICE.md credits Franklin Wei's map under CC BY-SA 4.0; plan §12 no longer presents squash merging as the mitigation. Rebuilding the branch without v1 history is left to the owner (see Questions).

**Plan text.** §12: "Merges are **squash only**, so v1's history (which contains the CC BY-SA map fragments, L-4) never reaches `main`."

**Problem.** The repository is public, and commit ada63a0, whose GTW mockup contains fragments of built1n's CC BY-SA 4.0 MAP, is already served on branch `claude/beautiful-ritchie-51u065`. That branch is also where M0 lives. A PR opened from it pins ada63a0 in `refs/pull/N/head` for good, and deleting the branch does not remove it. Squash merging only controls what lands on `main`. The attribution obligation applies to what is distributed, and here that is the whole public repository.

**Evidence.** `gh api repos/GhostofGoes/WOPR` gives visibility "public". `gh api repos/GhostofGoes/WOPR/branches` gives `claude/beautiful-ritchie-51u065` and `main`. `gh api repos/GhostofGoes/WOPR/commits/ada63a0` resolves to "Add WOPR architecture and scope plan". `git show ada63a0:docs/PLAN.md` lines 152-154 contain `__----/^\.` and `/__/   \/^^\`. docs.github.com, removing-sensitive-data-from-a-repository: commits stay accessible "Through any pull requests that reference them", and removal needs GitHub Support.

**Proposed fix.** Before opening any PR, create a branch from origin/main whose single commit holds the current tree, with no ada63a0 ancestry. Open the PR from that branch and delete `claude/beautiful-ritchie-51u065`. Alternatively, add a CC BY-SA 4.0 attribution to Franklin Wei for the v1 mockup in NOTICE.md and stop presenting squash merging as the mitigation.

**Verification.** `gh api repos/GhostofGoes/WOPR` reports visibility public, and its branches are `claude/beautiful-ritchie-51u065` and `main`. `gh api repos/GhostofGoes/WOPR/compare/ada63a0...claude/beautiful-ritchie-51u065` returns status "ahead", behind_by 0, ahead_by 16, so ada63a0 is an ancestor of the public branch. `git show ada63a0:docs/PLAN.md` holds a GTW mockup whose Eurasia outline (`__----/^\.`, `_/._-_`, `/__/   \/^^\`, `\.\_--`) matches built1n MAP lines 9-12, and MAP's header says CC BY-SA 4.0, "Attribution should be given to Franklin Wei". The finding is right that squash merging controls only main. It also misses that the mitigation fails for main too. docs/reviews/PLAN-v1-review.md:738 at HEAD quotes three of the fragments verbatim, so they reach main with the M0 squash merge anyway, though short quotation in a review is likely de minimis or fair use. No PR exists yet: `gh api repos/GhostofGoes/WOPR/pulls?state=all` is empty, so nothing is pinned in refs/pull yet. The first fix is incomplete. Rebuilding the branch without ada63a0 ancestry and deleting the old branch shrinks exposure, but the commit stays fetchable by SHA from GitHub until Support purges it, per the docs page the finding cites. The legal stakes are small, with a few lines of ASCII outline, which keeps this Minor.

**Corrected fix.** Primary, and cheap and complete: add a NOTICE.md entry crediting Franklin Wei's `MAP` (built1n/wargames, CC BY-SA 4.0). It covers the v1 mockup in commit ada63a0 and the fragments quoted in docs/reviews/PLAN-v1-review.md, and states that those fragments are available under CC BY-SA 4.0. Reword §12 so squash merging is not presented as the mitigation. Optional, and only before the first PR is opened: open the PR from a branch with no ada63a0 ancestry and delete `claude/beautiful-ritchie-51u065`, noting that a full purge needs GitHub Support.

### SL-7. The privacy claims for the debug log ignore Bubble Tea's TEA_TRACE, which logs every keystroke and frame to a path taken from the environment

**Severity:** Nit (was Minor). **Verdict:** overstated.

**Status:** Fixed in the plan. §8 scopes the claim to wopr's own log and mentions `TEA_TRACE`/`TEA_DEBUG`.

**Plan text.** §8 Debug log: "Typed input is never logged." "`TEA_DEBUG` is never set by the app; Bubble Tea would write a panic log into the CWD." §4.7: "prompts and replies are never logged."

**Problem.** Bubble Tea v2.0.10 reads `TEA_TRACE` from the process environment in NewProgram. When it is set, Bubble Tea opens that file, and the input reader logs every raw input chunk while the renderer logs every output frame. A user who has TEA_TRACE exported, as is common among Bubble Tea developers, gets every keystroke written to an arbitrary path: LOGON attempts, all typed text and, in M7, LLM replies. That breaks both claims. Likewise, a TEA_DEBUG that the user has set still makes Bubble Tea write panic logs into the CWD; "never set by the app" does not prevent it. In addition, `os.MkdirAll(dir, 0700)` and `OpenFile(..., 0600)` leave the mode of an existing directory or file unchanged.

**Evidence.** bubbletea v2.0.10 tea.go:640-645 (`os.LookupEnv("TEA_TRACE")` → `os.OpenFile(tracePath, ...)` → p.logger); tty.go:74-75 (`drv.SetLogger(p.logger)`); ultraviolet terminal_reader.go:200 `d.logf("input: %q", read)`; cursed_renderer.go:261 `output: %q`; tea.go:1306-1313 (TEA_DEBUG panic log in the CWD). Local experiment: built a bubbletea v2.0.10 program, ran it in a pty with TEA_TRACE=trace.log and typed "J". trace.log line 63 reads `terminal_reader.go:502: input: "J"`.

**Proposed fix.** In `main`, before `tea.NewProgram`, call `os.Unsetenv("TEA_TRACE")` and `os.Unsetenv("TEA_DEBUG")`. If tracing is wanted, re-enable it only through an explicit `WOPR_DEBUG` level that writes into the 0600 cache-dir log and warns on stderr that keystrokes are recorded. Chmod the log directory and file after opening them. Add a test and state the rule in §8.

**Verification.** The mechanics are correct. bubbletea v2.0.10 tea.go:640-645 opens $TEA_TRACE (O_APPEND, 0600) and sets p.logger. tty.go:75 passes it to the driver. ultraviolet terminal_reader.go:200 logs `input: %q`, cursed_renderer.go:261/614 log `output: %q`, and tea.go:1306-1313 writes a TEA_DEBUG panic log in the CWD. The app does not unset either variable (no TEA_ or Unsetenv in the repo's Go code). The privacy impact is negligible, though. Both are opt-in developer switches the user sets in their own environment. They write the user's own input and screen to a path the user chose, with 0600 on creation, and expose nothing to any other party. The plan's claims are about wopr's own debug log ("The debug log holds no typed input"), and those stay true. The typed data is LOGON names and game moves, not secrets. M7 keys come from the environment and are never rendered. The MkdirAll/OpenFile point is true Go behaviour (perm applies only on creation), but wopr is the only writer of that path and the file holds no typed input. Requiring a WOPR_DEBUG re-implementation of tracing is more than this warrants.

**Corrected fix.** In §8, scope the claim to wopr's own debug log, and add one line: "Bubble Tea's own TEA_TRACE and TEA_DEBUG, if the user sets them, record raw input/output or panic logs; wopr does not set them." Optionally unset TEA_TRACE and TEA_DEBUG in main unless WOPR_DEBUG is set. No chmod or test is needed.

### SL-8. The R-1 and §7 mitigations for movie mode misdescribe what M6 ships, and there is no takedown runbook despite immutable releases and the Go proxy cache

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. Plan §7 and RK-1 say what M6 ships; NOTICE.md has a rights-holder contact and AGENTS.md a takedown runbook.

**Plan text.** §7 Legal scope: "Movie mode concentrates film text. Scenes contain only text that appears on the WOPR terminal on screen ... Every line is tagged, and the quotations are listed in `NOTICE.md` as excluded from the MIT grant. The owner accepts the remaining risk by requesting the feature." §16 R-1 mitigation: "Short on-screen terminal text only; provenance per line; `NOTICE.md` exclusion and disclaimer; ..."

**Problem.** This does not reopen the feature decision; it concerns how the risk is described and handled. "Short" is not a constraint that M6 meets: the five scenes replay every WOPR terminal exchange in order, with David's typed lines and the climax's closing lines. Amount and substantiality is the one factor the plan controls, and it is left unbounded. The NOTICE.md exclusion and the non-affiliation disclaimer prevent mislicensing, but they do not reduce infringement exposure. Nothing plans for a rights-holder request. Immutable releases forbid editing or deleting assets, so only whole releases can be deleted. proxy.golang.org keeps serving a version "even if it is not available at the origin", and `retract` only hides a version from @latest.

**Evidence.** PLAN.md:696-704 (the five-scene table), 734-737 and 1085. docs.github.com immutable-releases: "Release assets cannot be modified or deleted" and "If you delete the immutable release, you can delete the tag, but you cannot reuse the same tag name". proxy.golang.org FAQ: "this bad release may still be available in the mirror even if it is not available at the origin"; the FAQ recommends `retract`.

**Proposed fix.** Rewrite R-1 to say that M6 ships the film's full terminal script and that the owner accepts this, and drop "short" as a mitigation. Add a takedown runbook to SECURITY.md and AGENTS.md: delete the affected releases; ship a patch with `internal/movie/scenes` removed (keep all film text in the three tagged locations so this is a single PR); add `retract` for the affected range; contact the Go proxy team as its FAQ describes. Add a contact line for rights holders to NOTICE.md.

**Verification.** The decision to build the feature is not challenged. R-1's mitigation lists "Short on-screen terminal text only", but §7's five scenes (PLAN.md:696-704) replay every WOPR terminal exchange, including David's typed lines. "Short" is not a constraint M6 meets. No takedown procedure exists: SECURITY.md and §11.3 cover only bad releases (patch plus retract), and R-2 says only "revisit if contacted". The constraints check out. docs.github.com immutable-releases says "All files attached to the release ... are protected from modification or deletion" and "If you delete the immutable release, you can delete the tag, but you cannot reuse the same tag name". The proxy.golang.org FAQ says a bad release "may still be available in the mirror even if it is not available at the origin" and recommends retract. Two corrections to the proposed fix. First, the proxy FAQ describes no removal-request process. The only contacts it lists are for traffic problems or malicious modules, so "contact the Go proxy team as its FAQ describes" is unsupported (UNKNOWN whether Google honours rights-holder removals there). Second, removing internal/movie/scenes alone would not remove the film text. The interactive persona (internal/wopr/lines.go, games/ending, games/gtw's film renderer, assets) already ships most of the same lines, and M6 adds mainly David's typed lines and the call-back scene. That also makes "multiplies the amount shipped" an overstatement.

**Corrected fix.** Rewrite R-1 to say that M6 ships the film's complete WOPR terminal script, much of which the interactive persona already contains. Drop "short" as a mitigation, and keep the owner's acceptance of the risk. Add a takedown runbook to SECURITY.md and AGENTS.md: delete the affected immutable releases; ship a patch that removes or replaces the named text wherever it lives (the provenance tags locate every line); add a `retract` for the affected versions; and record that copies remain in the Go module mirror and in git history, with no documented removal path for the mirror. Add a rights-holder contact line to NOTICE.md.

## Implementability (IM)

### IM-1. The lint self-test fixture is placed where the lint gate itself lints it, so M0 cannot have both a green ci-ok and a fixture that fires

**Severity:** Major. **Verdict:** confirmed. **Same issue as AR-3**, which carries the status.

**Plan text.** §10 (lines 859-860): "`config verify` accepts rules that never fire, so `internal/archtest/lintfixture` holds a deliberate violation of each rule. A CI step asserts that each rule reports on it." §10 (line 837): "The gate is `prek run --all-files`." A.1 (line 1148): `golangci-lint-full`, entry = "golangci-lint run". M0 exit (line 1063): "`ci-ok` green ...; the lint fixture fires every rule".

**Problem.** `internal/archtest/lintfixture` is an ordinary package directory, so the `./...` pattern includes it. The golangci-lint-full hook runs on the whole module with `pass_filenames: false`, so every gate run reports the fixture's deliberate violations and fails. `prek run --all-files`, the `lint` job and `ci-ok` therefore go red permanently. A.2 has no exclusion for the fixture. archtest's DAG table (line 194: `internal/archtest → stdlib`) also forbids the fixture's imports of bubbletea, math/rand/v2 and time. The plan also says nothing about uniq-by-line: when two violations share a line, only one is reported, so a rule can look as if it never fires.

**Evidence.** Hook manifest: golangci-lint v2.14.0 `.pre-commit-hooks.yaml:9-16` (golangci-lint-full: "Runs on all files in the module", `pass_filenames: false`). Local experiment: a module with the repo's `.golangci.yml` (identical to A.2), `internal/archtest/lintfixture/fixture.go` and golangci-lint v2.14.0 (go1.27.1). `golangci-lint run ./...` exits 1 with depguard plus forbidigo `time.Sleep` and `tea.Tick` reports in the fixture. The `rand.IntN` on the same line as `tea.Tick` is not reported (uniq-by-line). After moving the fixture to `internal/archtest/testdata/lintfixture/`, `run ./...` no longer sees it, and `run ./internal/archtest/testdata/lintfixture/` reports all three issues and exits 1.

**Proposed fix.** Put the fixture under `internal/archtest/testdata/lintfixture/`. Go's `./...`, `go list`, `go vet` and golangci-lint all skip testdata. Put one violation per line. The self-test step should run `golangci-lint run ./internal/archtest/testdata/lintfixture/` and assert that each rule's message appears. Note in §10 that the fixture sits outside both the gate and archtest.

**Verification.** Reproduced. I built a module with the A.2 `.golangci.yml` extracted verbatim from PLAN.md and a fixture at `internal/archtest/lintfixture/`, then ran golangci-lint v2.14.0 (go1.27.1). Plain `golangci-lint run`, which is what the hook entry runs, exited 1 with depguard and forbidigo reports in the fixture. The manifest at v2.14.0 `.pre-commit-hooks.yaml:9-16` confirms golangci-lint-full runs on the whole module with `pass_filenames: false`. After moving the fixture to `internal/archtest/testdata/lintfixture/`, `run` no longer saw it and `go list ./...` omitted it; `run ./internal/archtest/testdata/lintfixture/` reported every rule and exited 1. The uniq-by-line point also holds: `tea.Tick` on the same line as `rand.IntN` was hidden and only appeared with `--uniq-by-line=false`. In my first run it was worse: revive's `exported function X should have comment` report on the same line hid every forbidigo report on that line. No other part of the plan excludes the fixture: codespell excludes testdata, but A.2 has no exclusion and the archtest row allows stdlib only. The M0 code already made exactly this move (`internal/archtest/testdata/lintfixture/fixture.go`, `internal/archtest/lint_test.go`), which confirms the plan text is wrong as written. Kept at Major because the M0 exit criterion cannot be met as written, even though the fix is cheap.

**Corrected fix.** As proposed: move the fixture to `internal/archtest/testdata/lintfixture/` and have the self-test lint that path explicitly. Put one violation per line. Also give the fixture a package comment and a doc comment on each function, or run the self-test with `--uniq-by-line=false` (or with only forbidigo and depguard enabled), because revive's reports on the same line also hide forbidigo. Assert each rule by linter name plus a message fragment. Say in §10 that the fixture sits outside both `./...` and archtest.

### IM-2. The M0 release.yml dry run cannot pass: `goreleaser release` without --snapshot needs a tag on HEAD, and no tag exists until v0.1.0 (M2)

**Severity:** Major. **Verdict:** confirmed. **Same issue as CR-1**, which carries the status.

**Plan text.** §11.3 step 2 (line 908): "`goreleaser release --clean --skip=publish` with release settings"; line 923-924: "`workflow_dispatch` runs steps 2–4 as a dry run." M0 exit (line 1063): "a `release.yml` dry run passes `build`/`smoke`/`repro`". Decision 13: v0.1.0 after M2.

**Problem.** A non-snapshot GoReleaser run reads the git state first. If the repository has no tags, it aborts. If HEAD is ahead of the latest tag, the validate pipe aborts. The `workflow_dispatch` dry run runs on a `main` commit, which has no tag in M0 and is generally ahead of any tag later. The exit criterion therefore fails as written, and the dry run keeps failing after v0.1.0 too. A `--skip=validate` flag does not help when there are no tags at all.

**Evidence.** Local experiment with GoReleaser v2.18.2 on a copy of the A.3 config (untagged repository with an origin remote): `goreleaser release --clean --skip=publish` → `error=git doesn't contain any tags - either add a tag or use --snapshot`, exit 1. After tagging an older commit: `error=git tag v0.0.1 was not made against commit 98751a0…`, exit 1. Source: goreleaser v2.18.2 `internal/pipe/git/errors.go:28` (ErrNoTag) and `internal/pipe/git/git.go:178-193` (validate returns ErrWrongRef unless snapshot or skip-validate).

**Proposed fix.** Specify that the `workflow_dispatch` dry run uses `goreleaser release --snapshot --clean` (the same config, so the `repro` checksum diff still works), and that only tag runs use the non-snapshot form. Alternatively, drop the release dry run from M0 exit and make it a pre-v0.1.0 gate in M2, run against a throwaway `v0.0.0-rc` tag on a fork.

**Verification.** Reproduced with GoReleaser v2.18.2 on the A.3 config, extracted verbatim, in an untagged repository with an origin remote. `release --clean --skip=publish` failed with `git doesn't contain any tags - either add a tag or use --snapshot` (exit 1). `--skip=publish,validate` failed with the same error, so skip-validate does not help without tags. After tagging an older commit, it failed with `git tag v0.0.1 was not made against commit f54759e…`. Source at `goreleaser/v2@v2.18.2` matches: `internal/pipe/git/errors.go:28` (ErrNoTag) and `git.go:178-200` (validate returns ErrWrongRef unless snapshot or skip-validate). `release --snapshot --clean` run twice on the same commit gave identical `checksums.txt`, so a snapshot dry run keeps the repro diff meaningful. The plan's step 2 literally says `goreleaser release --clean --skip=publish` and the dispatch path runs steps 2–4, so the dry run fails as written. The M0 `release.yml` already passes `--snapshot` on `workflow_dispatch`.

### IM-3. M0 and M1 exit criteria depend on work from later milestones (e2e, movie mode) and on owner actions

**Severity:** Minor (was Major). **Verdict:** overstated.

**Status:** Fixed in the plan. e2e cases are tagged by milestone (§9), `games/testkit` is an M1 deliverable, and owner-only items are M0b, gating v0.1.0 rather than M1 (§15).

**Plan text.** §11.2 (lines 898-899): build "cross-compiled e2e test binaries"; smoke "…then the e2e test". M0 exit (line 1063): "`ci-ok` green with all six smoke runners … settings applied". M1 (line 1064): deliverables "…goldens, fuzz, e2e"; exit "Every M1 golden flow and the e2e cases are green on all runners." §9 e2e cases (lines 815-819) include "`--movie joshua --instant` → `FINE.` → exit on `q`".

**Problem.** (a) M0's `ci-ok` needs `smoke`, and `smoke` runs the e2e binary. `internal/e2e` is an M1 deliverable, and its first case (`Joshua` → `GREETINGS PROFESSOR FALKEN.`) needs the M1 persona. M0 must therefore either ship a stub e2e package or omit steps that §11.2 requires, and the plan says neither. (b) M1's exit includes "the e2e cases", and one of those is the M6 movie case. (c) M0 exit also needs owner-applied repository settings (line 966: "The owner applies them in M0") plus a canary PR, so an implementing agent cannot close M0 alone, yet nothing says M1 may start in parallel. (d) `games/testkit`, which §6.1 makes part of every game's definition of done and §7 uses for the consistency test, appears in no milestone's deliverables. M0 is also the largest milestone: three workflows, a reusable smoke workflow over 6 runners plus a container, a 5-job attested release pipeline, archtest, the lint self-test and settings. Its release half cannot be exercised until a tag exists anyway.

**Evidence.** Plan text as quoted. `grep -n testkit docs/PLAN.md` hits only lines 189, 227, 430, 622, 731, 788, 827 and 1018, none of them in §15. Experiment for the release half: see the GoReleaser finding.

**Proposed fix.** Rewrite the exit criteria per milestone. M0: smoke runs `--version`, `--games` and the closed-pipe check only, and the e2e step is added in M1. M1: "the e2e cases marked M1" (the M6 case is tagged M6). Add `games/testkit` to M1 (it wraps the `proto/host` runner M1 builds). Split M0 into M0a (scaffold, ci.yml, archtest, AGENTS.md; agent-completable) and M0b (owner settings, canary, release.yml dry run), with M0b gating v0.1.0 rather than M1.

**Verification.** The text-level claims are accurate:

- §11.2 smoke runs "the e2e test", but `internal/e2e` is an M1 deliverable and the first §9 case needs the M1 persona.
- M1 exit says "the e2e cases", which literally include the `--movie` case.
- "settings applied" and the gitleaks canary PR are owner and GitHub actions.
- `testkit` appears only at lines 189, 227, 430, 622, 731, 788, 827 and 1018, never in §15.

The practical impact is small, though:

- The M0 implementer shipped an M0-level e2e test (`internal/e2e/e2e_test.go`: TestVersion, TestGamesAndLicenses, TestClosedPipe, TestRefusesWithoutTerminal, TestTUIStartsAndInterrupts).
- M1 work (runner, console, persona, testkit) is already proceeding before the owner-only M0 items. Nothing blocked the schedule.
- "Its release half cannot be exercised until a tag exists anyway" is wrong. `goreleaser release --snapshot --clean` works on an untagged repository (local experiment), so the dry run can run in M0 once the previous finding's fix is applied.

These are sequencing and wording gaps, not a risk to correctness or schedule.

**Corrected fix.** Tag each §9 e2e case with its milestone (M0: version, games, closed pipe, non-TTY refusal and Ctrl+C; M1: Joshua and resize; M6: movie). Make M1 exit "the e2e cases marked M0–M1". Add `games/testkit` to M1 deliverables. Mark the owner-only M0 items (repository settings, canary PR) as gating v0.1.0, not M1, and say M1 may start once `ci-ok` is green. A formal M0a/M0b split is optional.

### IM-4. The archtest edge table, which also checks test imports, forbids the plan's own test strategy and omits packages M0 and M1 must create

**Severity:** Major. **Verdict:** confirmed. **Same issue as AR-1**, which carries the status.

**Plan text.** §4.1 line 169: "Import edges not listed below are forbidden"; lines 205-206: archtest "reads each package's `Imports`, `TestImports` and `XTestImports` … and checks every edge against this table." Rows: `internal/wopr → proto, prompt, games`; `internal/cli → games …, movie …, version`; `internal/games/<game> → proto, prompt, games, games/{ai,cards,board}, games/ending, assets, sim`; `internal/movie → proto, prompt, assets, games/gtw, games/ending`; `internal/games/ending → proto, prompt, assets`.

**Problem.** Test-only edges are missing everywhere, though the plan relies on them:

- cli and wopr tests use gamestest fakes (lines 203-204), but their rows do not list it;
- every package's goldens use `internal/golden` (§9 line 796), which only testkit's row lists;
- game packages need `games/testkit` for the §6.1 transcript;
- the §7 consistency test in movie needs `wopr` and `testkit`.
Rows are also missing for packages the layout creates:
- `internal/ui/{console,screens,testutil}` and `internal/movie/scenes` (does `ui`'s row cover subpackages? `proto` and `proto/host` are separate rows);
- the lint fixture;
- a package to embed `LICENSE`/`NOTICE.md`/`THIRD_PARTY_NOTICES.txt` for `--licenses`. Those files sit at the repo root for the archives, and `go:embed` cannot reach `..`, so the M0 implementer already had to add an unlisted root `package wopr` (`/home/user/WOPR/legal.go`, commit 454f6d9) whose name collides with `internal/wopr`.
Third-party edges are inconsistent. The ui and theme rows list modules, so third-party imports are evidently table-checked, but `games/chess` has no `github.com/corentings/chess/v2` (decision 15). `games/ending` must do tic-tac-toe "self-play" (§6.2.3) but may not import tictactoe, board or ai.

**Evidence.** Local experiment: `//go:embed ../../NOTICE.md` in cmd/wopr fails with `pattern ../../NOTICE.md: invalid pattern syntax` (go1.27.1). See also pkg.go.dev/embed: patterns may not contain '..'. `/home/user/WOPR/legal.go` declares `package wopr` with `//go:embed LICENSE`, `NOTICE.md` and `THIRD_PARTY_NOTICES.txt`.

**Proposed fix.** Restate the table as: package pattern (state whether subpackages inherit), allowed imports, and extra test-only imports (`internal/golden`, `games/gamestest`, `games/testkit`, `ui/testutil`, allowed in TestImports/XTestImports of any package below them). Add rows for the embed package (and rename it, e.g. `internal/legal` with a generated copy, or a root `package notices`), for `games/chess → corentings/chess/v2`, and for `games/ending → games/board` (or canned frames). Exempt `testdata/`. Say whether stdlib is always allowed.

**Verification.** The core claim holds. §4.1 says unlisted edges are forbidden and that archtest checks TestImports and XTestImports, yet:

- only testkit's row lists `golden`, which every package's goldens use (§9);
- the cli and wopr rows omit `gamestest`, which their tests use (lines 203-204);
- the movie row omits `wopr` and `testkit`, which the §7 consistency test needs;
- there are no rows for `ui/{console,screens,testutil}` or `movie/scenes`, and nothing says whether subpackages inherit a row;
- `games/chess` lacks `corentings/chess/v2`, while the ui and theme rows do list modules.

The go:embed point is reproduced: `//go:embed ../../NOTICE.md` fails with `pattern ../../NOTICE.md: invalid pattern syntax`, so the M0 implementer added an unlisted root `package wopr` (`/home/user/WOPR/legal.go`).

The M0 archtest shows how much had to be invented (`internal/archtest/archtest_test.go:27-75`):

- a `testOnly` list (golden, catalog, gamestest, testkit, proto/host);
- subtree rules for ui, games, movie and tools;
- a root `.` rule;
- edges the plan does not list: cli→theme, games→prompt, cards→prompt, theme→x/ansi, ui→x/term, ui→rivo/uniseg;
- stdlib fences.

The plan's table therefore no longer matches the enforced one.

Three sub-points are weak:

- The name clash with `internal/wopr` is cosmetic, since Go package names only clash within one importing file.
- Ending self-play does not need board or ai; tic-tac-toe minimax is tiny or can use canned frames.
- "Exempt testdata/" in the fix is unnecessary, because `go list ./...` already skips testdata.

**Corrected fix.** Restate §4.1 as: package pattern (with explicit `/...` subtree semantics); allowed imports; a global test-only allow list (`internal/golden`, `games/gamestest`, `games/testkit`, `games/catalog`, `proto/host` for testkit setup) that is valid in TestImports and XTestImports of any package. Stdlib is allowed by default except for fenced packages (`net/http` only in llm; `os/exec` only in tools, archtest and e2e). Add rows for the root embed package (or a generated `internal/legal` copy checked in CI), `games/chess → github.com/corentings/chess/v2`, and the edges the M0 code already enforces. Drop the testdata exemption, which go list already provides. Make `internal/archtest` the authoritative copy after M0 and have the plan point to it, as Appendix A does, so the two cannot drift.

### IM-5. archtest's "running Go version equals the toolchain line" fails `go test ./...` for any contributor whose Go is newer than 1.27.1

**Severity:** Major. **Verdict:** confirmed. **Same issue as AR-2**, which carries the status.

**Plan text.** §4.1 lines 206-207: archtest checks "that the running Go version equals `go.mod`'s `toolchain` line (D-6)"; §9 line 828: "Contributors: `go test ./...` everywhere"; A.1 line 1183: pre-push hook `go test -short ./...`.

**Problem.** Under the default GOTOOLCHAIN=auto, the toolchain line is a minimum. A machine with a newer local Go (go1.27.2 once it ships, or Homebrew's auto-updated Go 1.28) runs the tests with its own version. `runtime.Version()` then differs from `go1.27.1`, archtest fails, and so do the documented test command and the pre-push hook. This repeats on every Go release until someone bumps `toolchain`. CI passes only because setup-go pins exactly 1.27.1, so the check guards CI while breaking every local machine.

**Evidence.** go.dev/doc/toolchain: "the go command uses its own bundled toolchain when that toolchain is at least as new as the go or toolchain lines in the main module". Local experiment: module with `toolchain go1.27.0`, run with the go1.27.1 binary and GOTOOLCHAIN=auto, prints `runtime.Version()` = `go1.27.1`.

**Proposed fix.** Check equality only when `CI=true` (or under a `-tags ci`/env switch set in ci.yml), and locally assert `>=` the toolchain line, or skip with a message. Write this rule into AGENTS.md next to the test command.

**Verification.** Reproduced. I used a module with `toolchain go1.27.0`, run by the go1.27.1 toolchain binary with GOTOOLCHAIN=auto. A test logging `runtime.Version()` printed `go1.27.1`, so a newer local Go is used as-is and never downgraded, matching go.dev/doc/toolchain. A strict equality check therefore fails for every contributor and every pre-push run (A.1 `go-test` hook, `go test -short ./...`) on any Go other than exactly 1.27.1. That will happen as soon as go1.27.2 ships, or under distro Go builds with GOTOOLCHAIN=local, until `toolchain` is bumped. CI passes only because setup-go pins 1.27.1. The M0 implementer already had to weaken it: `TestToolchainMatchesGoMod` skips unless `GITHUB_ACTIONS=true`. Kept at Major because it breaks the documented local test command and the pre-push gate on every Go release.

**Corrected fix.** As proposed: enforce equality only in CI (`GITHUB_ACTIONS=true` or `CI=true`); locally skip with a message, or assert `>=` the toolchain line. Document the rule in AGENTS.md next to the test command.

### IM-6. The M1 protocol leaves key contracts undefined: games.Info/Status/Game, Env sizing, who resolves Launch, who acts on Result.Next, and how the ending is launched

**Severity:** Major. **Verdict:** confirmed. **Also found as:** AR-5.

**Status:** Fixed. `games.Info`, `Entry` and `New func() Game` are defined, with Env given only at Start; the host takes an injected `Resolver` and chains `Result.Next` itself; Env's size is the placement's area; plan §4.2, §4.4. The `ending` slug comes from `catalog.Internal()` in M2.

**Plan text.** Line 183: `internal/games → proto  types only: Info, Status, Game, Registry, Resolve`; line 202: "`Registry` of `{Info, New func(Env) Game}`"; line 254-255: "`proto/host.Runner` owns the program stack… `Launch` pushes a game"; line 176: `internal/proto/host → proto`; line 369: `Width, Height int // the program's layout area, always ≥ its minimum`; lines 401-411: `Launch`, `Result.Next *Launch`, `NoVerdict`; §6.2: "`Done{NoWinner, Next: Launch{ending}}`", "The persona's phase goes `Ending → Shell`"; M0 (line 1063): "catalog with 16 `Planned` entries".

**Problem.** An M0/M1 implementer must invent all of these:

- `Info`, `Status` and `Game` are never defined. Their fields are scattered: Number, Listed, slug, aliases, name, PanelRows, Planned/Playable, a Layout column (§6.1), and a "coming in vX.Y" version (§13). Nothing says whether `Game` is `proto.Program`.
- `Env` is passed twice (`New(Env)` and `Start(Env)`) with no rule for which wins.
- Env.Width/Height must be "≥ its minimum", but no minimum is declared anywhere. The layout is chosen only by the `SetLayout` that `Start` returns, so the host cannot know the area when it builds `Env`.
- The Runner may import only `proto`, so it cannot map `Launch{Slug}` to a constructor, and no resolver interface is specified.
- `Launch{ending}` names a program outside the 16-entry catalog.
- Nothing says whether the host chains `Result.Next` itself or delivers `GameOver` to the persona, which re-emits it. Nothing says when the persona enters `Ending`; Rule.When=Ending is dead if the persona never receives input while the ending runs. Nothing says why `games/<game> → games/ending` exists if hand-off is by slug.
M1 freezes the runner and the `gamestest` stub that "exercises every Output", and GTW, tic-tac-toe and the ending (M2) are coded on top of it.

**Evidence.** `grep -n 'Info\|Planned\|Listed' docs/PLAN.md`: Info appears only at lines 183, 202 and 338 as a name; there is no struct anywhere. The DAG rows are at lines 176, 186 and 187.

**Proposed fix.** Add a §4.4 block that defines `games.Info{Number int; Listed bool; Slug, Name string; Aliases []string; Status Status; PanelRows int; Since string}`, `type Game = proto.Program`, and `New func() Game` with Env given only at Start. Add a host-side `Spawner func(Launch) (proto.Program, error)` injected by ui, which resolves registry slugs plus the reserved slug `ending`. Rule: the host delivers `GameOver` to the program below. If `Result.Next` is set, the persona (not the host) re-emits it and sets `Ending` when the slug is `ending`. Env.Width/Height is the area of the layout in effect when Start's outputs have been applied, and a `ResizeEvent` follows every `SetLayout` that changes it.

**Verification.** Confirmed against the text:

- `Info` appears only as a name (lines 183, 202, 338) and no struct is defined.
- Env is passed twice, via `New func(Env) Game` (line 202) and `Start(Env)`, with no rule for which wins.
- `proto/host → proto` cannot map `Launch{Slug}` to a constructor, and no injected resolver is specified.
- `ending` is not one of the 16 catalog entries.
- Neither who acts on `Result.Next` nor when the persona enters `Ending` is stated, and `Rule.When=Ending` has no input while the ending is on top of the stack.

The M1 implementer had to decide all of this. `internal/games/registry.go` defines `Info{Number, Listed, Name, Slug, Aliases, Layout, PanelRows, Status, Blurb}` and `New func() Game`. `internal/proto/host/runner.go:25-26` defines an injected `Resolver func(Launch) (Program, Placement, error)`. At `runner.go:389-395` the host itself chains `Result.Next`, the opposite of what the proposed fix prescribes.

The proposed fix also contradicts itself on Env: Env is handed to `Start` before Start's outputs exist, so it cannot be "the area of the layout in effect when Start's outputs have been applied".

**Corrected fix.** Define in §4.4:

- `games.Info` (Number, Listed, Slug, Name, Aliases, Layout, PanelRows, Status, Since), `type Game = proto.Program`, and `New func() Game`, with Env given only at Start.
- An injected host `Resolver func(Launch) (Program, Placement, error)` that also resolves the reserved slug `ending`. Its Placement comes from Info (Layout, PanelRows), and Env.Width/Height is that placement's area at the current size. A `ResizeEvent` follows any `SetLayout` that changes the area.
- Pick one owner for `Result.Next` and write it down. Either the host chains it (as M1's runner does), or the host delivers `GameOver` and the persona re-emits it.
- If the host chains, state how the persona learns about the ending: for example, the `GameOver{NoVerdict}` from the ending moves it Ending→Shell with Offer{chess} armed. Then either drop `When: Ending` from Rule or define what it matches.
- Say why `games/<game> → games/ending` exists, or remove that edge.

### IM-7. Persona "scenes" (greeting, GTW-vs-chess) are undefined, and the dispatch order contradicts the §14 walkthrough

**Severity:** Minor (was Major). **Verdict:** overstated. **Also found as:** RF-11.

**Status:** Fixed. The M1 persona settles the greeting scene, and plan §4.2 and §4.6 describe it; §14 lists the greeting replies.

**Plan text.** §4.2 line 256-257: phases "`Dialing → Logon → Disconnected → … → Greeting → Shell ⇄ Ending`"; line 278-282: "…then the greeting scene and **Shell**. … host commands → **active scene** → **phase-gated commands** … → **explicit intent** → **offer acceptance** → **Brain**"; line 495: `When PhaseSet // Greeting | Shell | Ending`; §14.3 (lines 1038-1039): "3. `Joshua` → the greeting; 4. `Love to. How about Global Thermonuclear War?` → the chess offer".

**Problem.** `scenes.go` and the "active scene" are central to M1, but nothing defines them: their data shape, whether a step accepts any input or matches it, whether `LIST GAMES` or an intent typed mid-greeting is swallowed by the scene (the dispatch order says the scene goes first), and when a scene ends. The greeting is also claimed by two mechanisms at once: a scene, and scripted Rules with `When: Greeting`. The phase list contains a literal "…". §14.3 jumps from `Joshua` straight to the GTW intent, skipping the three replies Appendix C shows ("Hello.", "I'm fine…", "People sometimes make mistakes."). That is only possible if intents interrupt the greeting scene, which dispatch forbids. The M6 consistency test then asserts that the persona's lines equal the movie script, so whatever M1 guesses becomes a cross-milestone contract.

**Evidence.** `grep -n -i scene docs/PLAN.md` (excluding movie): the only persona mentions are at lines 110, 223, 279, 281 and 528, and none defines a scene.

**Proposed fix.** Define a persona scene in §4.6, e.g. `Scene{ID; Steps []Step{Say, Expect: Any|Match, OnOther: fallthrough|reprompt}}`, with these rules: at most one active, consumed only by its own Expect, and falls through to dispatch otherwise (so intents interrupt it, as in §14.3). Drop `Greeting` from Rule.When or say how the two interact. Write out the full phase graph without "…". Make §14.3 list the greeting replies, or say explicitly that they may be skipped.

**Verification.** The "undefined" half is true. Apart from the GTW-vs-chess behaviour (§4.6 line 528, which is defined as intent and offer handling), nothing defines a persona scene's shape, what it consumes, or whether commands and intents interrupt it. Rule.When=Greeting overlaps with the greeting scene, and the phase list contains a literal "…".

The "contradicts §14" half is a misreading. §14.3 step 3 says "`Joshua` → the greeting", which naturally covers the three greeting replies in Appendix C. The greeting ends with `SHALL WE PLAY A GAME?`, and step 4's "Love to. How about Global Thermonuclear War?" then goes through normal dispatch (intent outranks offer, §4.6). No interruption of an active scene is needed.

The in-progress M1 persona settled the semantics in a few lines (`internal/wopr/persona.go` `greeting()`): commands and game intents end the greeting early, any other line advances one scripted step, and the last step arms Offer{Any} and enters Shell. The film transcript in Appendix C, plus the M6 consistency test, constrain the result. This is a cheap spec gap, not a major risk.

**Corrected fix.** Define the greeting scene in §4.6:

- A command or an explicit intent ends it early and is dispatched normally.
- Any other line advances to the next scripted WOPR line.
- The last line (`YES THEY DO. SHALL WE PLAY A GAME?`) arms Offer{Any} and enters Shell.

In the dispatch list, say that the active scene only consumes input it does not hand on. Either drop `Greeting` from Rule.When or state that scripted rules do not run during the scene. Replace the "…" with the full phase graph. §14.3 can stay as is, optionally noting that step 3 includes the Appendix C replies.

### IM-8. Layout geometry, the front panel and the Appendix C mockups disagree, and the mockups claim to be exactly 80×24

**Severity:** Minor (was Major). **Verdict:** overstated. **Same issue as AR-11**, which carries the status.

**Plan text.** §2.4 line 127-128: "an optional one-row front panel"; line 134: target screens "all at exactly 80×24"; §4.3 lines 337-342: "**Panel**: … `Info.PanelRows` (≤ 12) on top. The console strip takes the rest, which is at least 10 rows at 24. **Full**: the view takes `H − 4` rows, then a 3-row strip and the input row. The optional front panel takes the last row. It is shown by default in `norad`"; line 310-311: "the input line shows `PROCESSING`"; Appendix C GTW (`norad`, `LayoutFull`) and Chess (`imsai`, `PanelRows` 12).

**Problem.** Full plus the front panel needs (H−4)+3+1+1 = 25 rows at H=24, and norad, the GTW theme, shows the panel by default. Panel's "at least 10" only works if the front panel is subtracted (12+10+1+1). The GTW mockup is 23 rows, not 24. Its front panel is two rows (a dashed rule plus a status line `DEFCON 3 PORT STAT…`) instead of one row of lights and a nameplate, and its view/strip split matches neither H−4 nor H−5. The chess mockup (imsai, so no panel) puts `PROCESSING ..` on row 22, the cursor on row 23 and leaves row 24 empty, whereas §4.3 says the input row is last and that PROCESSING is shown on it. Env.Width/Height, the contract every M2–M4 View is drawn against, is therefore ambiguous, and M1 must build the canvas renderer, the front panel and "80×24 mockups for chess and GTW phases" (line 1064) against it.

**Evidence.** Local count (awk over PLAN.md lines 1451-1542): greeting block 24 rows (max width 49), GTW block 23 rows (max width 79), chess block 24 rows (max width 51). GTW rows 21-23 are `█`, a dashed line, and `DEFCON 3   PORT STAT: SD-345   1200 BAUD   ALT NET READY`.

**Proposed fix.** Give one formula: H' = H − (panel ? 1 : 0). Console = H'−1 text rows + input. Panel = PanelRows + (H'−1−PanelRows) strip + input. Full = (H'−4) view + 3 strip + input. Define the front panel's single-row content (lights, nameplate, optional status text). Redraw the GTW mockup to 24 rows and the chess mockup with PROCESSING on the last (input) row. Add a golden that renders each layout at 80×24 with and without the panel.

**Verification.** The mockup facts are confirmed by my own count of PLAN.md lines 1456-1540:

- The greeting is 24 rows and chess is 24 rows, but GTW is only 23 rows.
- GTW's footer is two rows (a dashed rule plus `DEFCON 3 PORT STAT…`), not a one-row panel of lights and nameplate.
- The chess mockup has `PROCESSING ..` on row 22, the cursor on row 23 and a blank row 24, contradicting "the input line shows `PROCESSING`" (§4.3).

The formulas, however, are under-defined rather than contradictory. If H means the height left after the optional front panel, Panel at 24 gives 12 + 10 + 1 = 23 plus the panel row (hence "at least 10"), and Full gives (23−4) + 3 + 1 = 23 plus the panel row = 24. The plan just never defines H. The mockups are illustrative, and programs draw into whatever area Env/ResizeEvent supplies, so this is a cheap documentation fix, not a contract risk to M2–M4.

**Corrected fix.** As proposed: define H' = H − (panel ? 1 : 0) and give the three layout formulas in terms of H'. Define the front panel's single row. Redraw the GTW mockup to 24 rows with a one-row panel, and the chess mockup with PROCESSING on the input row. Add a golden for each layout at 80×24, with and without the panel.

### IM-9. §4.2 and §5 make cmd/wopr do Bubble Tea work that the DAG and depguard forbid there

**Severity:** Minor. **Verdict:** confirmed. **Same issue as AR-4**, which carries the status.

**Plan text.** Line 200: "**Only `ui` imports Bubble Tea.** Only `ui` and `theme` import `charm.land/*`."; line 172: `cmd/wopr → cli, ui, version, games/catalog`; line 247-248: "If `NO_COLOR` is set …, `main` passes `tea.WithColorProfile(colorprofile.Ascii)`"; line 245-246: "Bubble Tea's \"error opening TTY\" also maps to the same message and exit 2"; line 606-607: "Bubble Tea errors are mapped `ErrInterrupted` first (130), then `ErrProgramPanic` (1)"; line 212: "cmd/wopr/main.go flags → dispatch → exit code; NO_COLOR and TTY handling".

**Problem.** Implemented as written, main imports `charm.land/bubbletea/v2` and `github.com/charmbracelet/colorprofile`, which violates archtest's cmd/wopr row and the A.2 depguard rule. The stdout TTY check also needs a terminal library (stdlib `os.ModeCharDevice` also accepts /dev/null and NUL), and the cmd/wopr row lists none.

**Evidence.** Local experiment: a `cmd/wopr/main.go` doing exactly §4.2 step 3 and the `errors.Is(err, tea.ErrInterrupted)` mapping, linted with the repo's `.golangci.yml` (A.2) and golangci-lint v2.14.0 → `cmd/wopr/main.go:8:2: import 'charm.land/bubbletea/v2' is not allowed from list 'bubbletea-only-in-ui'` (exit 1).

**Proposed fix.** Make it `ui.Run(ctx, ui.Options{NoColor, …}) (exitCode int, err error)`. ui builds the tea options and maps the tea errors, and main only maps exit codes. Name the TTY helper (e.g. `github.com/charmbracelet/x/term`, already in the graph) and add it to the cmd/wopr or ui row. Fix the §4.1 note "(§4.3, §5)" to §4.2.

**Verification.** §4.2 step 3 has `main` pass `tea.WithColorProfile(colorprofile.Ascii)`, but the cmd/wopr row (line 172) allows only cli, ui, version and games/catalog, and line 200 says only ui imports Bubble Tea. A.2 depguard denies `charm.land/bubbletea/v2` outside `internal/ui/**`; my fixture run showed this rule firing on any non-ui file. The stdlib TTY point is reproduced: `os.Stdout.Stat()` reports `ModeCharDevice` true when stdout is `/dev/null`, so a real terminal check is needed. The M0 code already moved this into ui: `ui.Run` uses `term.IsTerminal` (`internal/ui/app.go:60,92`), and `cmd/wopr/main.go` imports no Bubble Tea. The §4.1 note "(§4.3, §5)" should indeed point to §4.2.

### IM-10. CLI rules contradict each other and leave the expected outcome of the pinned cases unstated

**Severity:** Minor. **Verdict:** confirmed.

**Status:** Fixed. Plan §5 has a table of pinned argv and outcomes, the terminator rule and the env rule; `--movie` exits 2 until M6; slugs are the full names.

**Plan text.** §5 lines 586-602: "If the consumed tokens end in `--`, flag parsing stops there for good… Tests pin each case: `wopr gtw -i`, `wopr -i gtw`, `wopr -- gtw -i`, `wopr -t -- gtw`, `wopr gtw chess`, `wopr -m 2 -i`"; "Ambiguity (e.g. `theaterwide`) prints the candidates and exits 2"; §14.4 line 1047: "`wopr -p theaterwide` … exit 2"; §4.1 line 229: package dirs `… theaterwide/ biotoxic/`; line 214: `Parse(args, reg, scenes, …)`; line 174: `internal/cli → … movie (scene index)`.

**Problem.** (1) With stdlib `flag`, `-t --` consumes `--` as the theme value, so the literal "consumed tokens end in `--`" rule misfires: `wopr -t -- gtw -i` gives theme="--" and two positionals. The plan never says what any pinned case should produce. (2) Slugs evidently follow package dirs (`gtw`, `tictactoe`, `chess`), which makes `theaterwide` the exact slug of game 13. Resolve tries slug before prefix, so `-p theaterwide` is not ambiguous, and §5/§14.4 expect exit 2. No slug table exists to settle it. (3) cli both receives `scenes` and imports movie, while cmd/wopr may not import movie, and movie does not exist until M6. Nothing says whether `-m` exists, errors, or is hidden in M0–M5, though `--help` must list scenes. (4) Truthiness and precedence of `WOPR_INSTANT`, `WOPR_REDUCE_MOTION` and `WOPR_PANEL` against flags are unspecified. `NO_COLOR` alone is precise.

**Evidence.** Local experiment (go1.27.1 stdlib flag implementing the §5 loop literally): `["-t" "--" "gtw" "-i"] -> theme="--" positionals=["gtw" "-i"]`; `["--" "gtw" "-i"] -> positionals=["gtw" "-i"]`; `["-m" "2" "-i"] -> movie=true positionals=["2"]`; `-is 1` and `-pchess` → "flag provided but not defined".

**Proposed fix.** Add a table of each pinned argv → (Config, exit code). Detect the terminator by the token fs.Parse stopped at, not by a value. Publish the 16 slugs and aliases in §6.1 (e.g. `theaterwide-tactical`, `biotoxic`) and change the ambiguity example to a real prefix (`theater`). Make `scenes` a `[]string` injected by cmd/wopr via ui, or keep the cli → movie edge, but not both. State that `-m` prints "available in v1.1" and exits 2 before M6. Env rule: flag &gt; env, and env is true only for `1`/`true`.

**Verification.** (1) Reproduced with go1.27.1 stdlib `flag`: `-t -- gtw -i` gives theme="--", so a literal "consumed tokens end in `--`" rule mis-detects a terminator, and no pinned case states its expected Config or exit code. In practice every value flag rejects "--" (unknown theme, bad seed, no such game), so the outcome is exit 2 either way; only the error message differs.
(2) This sub-point is conditional and largely refuted. The plan never says slugs equal package directory names. The M0 catalog uses full-name slugs (`theaterwide-tactical-warfare`, `theaterwide-biotoxic-and-chemical-warfare`, `internal/games/catalog/catalog.go:63-67`), under which `theaterwide` really is a two-way prefix ambiguity, so §5 and §14.4 are right.
(3) cli both takes a `scenes` parameter (line 214) and imports movie (line 174). That redundancy is real but a Nit, and `-m` behaviour before M6 is unstated.
(4) Precedence and truthiness between flags and `WOPR_*` variables are indeed unspecified.

**Corrected fix.** Add a table mapping each pinned argv to (Config, exit code). Detect `--` by where fs.Parse stopped (`fs.Args()` and the arg before it), not by a consumed value. Publish the 16 slugs and aliases in §6.1 to match the catalog's full-name slugs, and keep `theaterwide` as the ambiguity example. Choose either the `scenes` parameter or the cli→movie edge. State `-m`'s behaviour before M6 (for example, "available in v1.1" and exit 2). Env rule: a flag beats its env variable, and env booleans are true only for `1` or `true`.

### IM-11. Movie-mode controls and exit behaviour contradict the host's Esc rules and each other

**Severity:** Minor. **Verdict:** confirmed. **Same issue as AR-7**, which carries the status.

**Plan text.** §4.3 lines 315-321: "**Esc machine** (host-owned; games never see Esc)… At the Shell, Esc does nothing"; §4.4 line 380: `KeyEvent … Key: Up Down Left Right Enter Backspace Rune`; §7 controls: "Esc | Scene menu", "`LOGOFF` or `q` (in the menu) | Exit 0"; §9 line 819: "`--movie joshua --instant` → `FINE.` → exit on `q`"; §14.8: "`wopr -m 2 -i` exits 0 at the end of the scene list".

**Problem.** The director is a root `proto.Program`. Esc is host-owned, there is no Esc key in `KeyEvent`, and at depth one Esc "does nothing", so the director can never open its scene menu. The e2e case expects `q` to exit right after `FINE.`, but `q` works only in the menu. §14.8 says playback exits by itself at the end of the list, and §7 says nothing about end-of-list or whether `-m 2` plays only scene 2 or scenes 2 through 5. Because M1 freezes `KeyEvent` and the Esc machine, M6 would need a protocol change.

**Evidence.** Plan text as quoted. The Key enum at line 380 has no Esc, Space or PgUp/PgDn member (Space arrives as Rune ' ').

**Proposed fix.** In M1, add a host rule that a root program which opts in (e.g. `AwaitKeys{Esc: true}`) receives `KeyEvent{Key: Esc}`. Define movie end-of-list as "return to the scene menu" or "exit 0", and make §9 and §14.8 match. State whether `[scene]` plays one scene or starts the list there.

**Verification.** Confirmed:

- Esc is host-owned and "games never see Esc", and at depth one "Esc does nothing" (§4.3).
- The director is a root `proto.Program` under all host rules (§4.2 item 9, §7).
- The `KeyEvent` Key set (line 380) has no Esc, so §7's "Esc | Scene menu" cannot be delivered.
- `q` exits only "in the menu", but the §9 e2e case exits on `q` right after `FINE.`.
- §14.8 says `-m 2 -i` exits 0 at the end of the scene list, which also conflicts with an `--instant` run waiting for `q`.
- Whether `[scene]` plays one scene or starts the list there is unstated.

The M6 impact is smaller than implied: adding an Esc key constant and an opt-in is an additive protocol change, not a rewrite.

**Corrected fix.** As proposed. Add an opt-in for root programs to receive `KeyEvent{Key: Esc}`; this can be reserved in M1 even if unused until M6. Define end-of-list as either return-to-menu or exit 0, and make the §9 e2e case and §14.8 agree. State whether `[scene]` plays one scene or starts the list there.

### IM-12. Dangling and colliding identifiers and small cross-reference drift

**Severity:** Nit. **Verdict:** confirmed.

**Status:** Fixed in the plan. A-18 is gone, risks are `RK-n`, the §4.2 reference is right, and the status burst is no longer an abs0 item.

**Plan text.** Line 419: "**`Think` contract** (A-3, A-18, G-1)"; line 734: "(risk R-1 in §16)" and the §16 ids R-1…R-8; line 4-5: "finding ids such as A-3 are cited inline"; line 82-86: "Exactly two text items are taken from `abs0/wargames`" vs line 1346: "a status burst (A/F)"; line 212: "NO_COLOR and TTY handling (§4.3, §5)".

**Problem.** A-18 does not exist in PLAN-v1-review.md, whose A-series ends at A-17. §16's risk ids R-1…R-8 reuse the review's R-1…R-3 namespace (requirement findings) and sit next to requirement ids R1…R13, so "R-1" is ambiguous in a plan that cites review ids bare. Appendix B tags part of the status burst `A`, a third abs0 item beyond the "exactly two" of §2.1, which affects how M1 tags `lines.go`. NO_COLOR and TTY handling are §4.2 items 2-3, not §4.3. rumdl reports nothing else.

**Evidence.** `grep -oE '\*\*[A-Z]{1,2}-[0-9]+' docs/reviews/PLAN-v1-review.md`: the A ids run A-1…A-17, with no A-18. `uvx rumdl@0.2.78 check --disable MD013,MD033,MD041 docs/PLAN.md` → only `5:80 [MD057] Relative link 'reviews/PLAN-v2-review.md' does not exist`.

**Proposed fix.** Replace A-18 with the intended id (likely A-3 or G-1) or drop it. Rename the risks to RK-1…RK-8. Either say the status burst is part of the "header block" item or make the list three items. Fix the §4.1 reference to §4.2.

**Verification.** Verified each point:

- `grep -oE '\*\*[A-Z]{1,2}-[0-9]+' docs/reviews/PLAN-v1-review.md` lists A-1…A-17 and no A-18, while PLAN.md:419 cites A-18.
- The review defines R-1…R-3, and §16 reuses R-1…R-8 for risks next to requirement ids R1…R13. Line 734 says "risk R-1 in §16", which disambiguates locally, but bare citations stay ambiguous.
- Appendix B tags the status burst "(A/F)", which conflicts with "exactly two" abs0 items in §2.1.
- Line 212 says "(§4.3, §5)", but NO_COLOR and TTY handling are §4.2 items 2-3.
- `uvx rumdl@0.2.78 check --disable MD013,MD033,MD041 docs/PLAN.md` reports only `5:80 [MD057] Relative link 'reviews/PLAN-v2-review.md' does not exist`.

## Coverage of the v1 review

Each of v1's findings, its verification log, its owner decisions and the gaps its completeness passes found was
checked against v2. 118 of 138 items are addressed, 16 partly, and 4 are closed by an owner decision. The
partial items are the ones that matter; v2.1 resolves them as the last column says.

| v1 item | Status | Note | v2.1 |
|---|---|---|---|
| R-1 | partial | The Src codes are in. The refute correction is used: R9 is now 'v1 decision 6 (source not recorded)' and R5 cites the owner's round-2 answer. Still mislabelled U: Decision 12 (AGENTS.md and @AGENTS.md come from review M-1), the seat rule in 14 (from G-3) and 'v1.0.0 after M5' in 13. No owner answer links to a durable record, which the review's Owner-decisions section asks for. Nit. | v2.1 relabels decisions 12-14 and points the U code at the owner-decisions record. |
| R-2 | addressed | Each game has a definition of done. Fidelity is left to the M5 viewing pass, as the refute says it must be. |  |
| R-3 | addressed | The README outline links to this plan. |  |
| A-1 | addressed | Layout and input mode are separate, and modes toggle (checkers). SetLayout(Console) handles the GTW menus. Both refute FIXPROBs are resolved: the overlap rule and the 3-line strip. |  |
| A-2 | addressed | The semantic canvas is in proto, with SuitRed/SuitBlack. Attr is separate from Style. Blink cells count as clock consumers. StyleMap goldens. |  |
| A-3 | addressed | Think has Limit and Budget. Deterministic runs obey Limit. Budget uses a deadline and Esc uses WithCancelCause. testkit runs Fn concurrently with Handle and View. Chess timings are measured. Nit: L383 says ThinkDone Err=Canceled comes 'only after a confirmed abort', but a confirmed abort drops the result and pops the game (L318-319), while a Brain Esc (L320) delivers Canceled with no confirmation. |  |
| A-4 | addressed | Snapshot is a value copy. Turn increments on every call, cancelled ones included. Effects are applied in Handle. Keys are buffered and Enter is held, which matches the typeahead rule. |  |
| A-5 | addressed | The runner owns the program stack and Session owns the phase, including Dialing and Disconnected. There is no InGame phase: an active game is stack depth &gt; 1. This fixes the refute's dual-owner point. |  |
| A-6 | partial | archtest reads Imports, TestImports and XTestImports; depguard is set with max-same-issues 0. The refute correction 'cli → theme' was not adopted (L174). The table also omits: games → prompt (Resolve normalises with prompt, L217/L598); the external github.com/corentings/chess/v2 edge; theme → x/ansi; ui → x/term and uniseg; the module-root package that embeds the notices. The repo's archtest rules (internal/archtest/archtest_test.go) already had to add all of these, so §4.1 and archtest have drifted even though AGENTS.md says the table 'mirrors' §4.1. The M6 consistency test (movie tests importing wopr, L730-733) has no allowed edge either. Minor. | Plan §4.1 now mirrors archtest, including cli → theme, games → prompt, the chess module and theme → x/ansi. |
| A-7 | addressed | Corrected fix used: matching is per clause, with fillers, verbs, negators, LETS/LET'S/LET US, both film lines as must-match tests, one-turn offers, and numbered selection that disarms. Adding ',' as a clause separator is an extension of the review's fix and does no harm. |  |
| A-8 | addressed | There is a Key enum with Backspace, and Esc is removed as the refute asked. Nit: the plan does not say that Space arrives as Rune ' '. |  |
| A-9 | addressed | The clock arms for the earliest deadline, bumps its generation on change and coalesces dt. |  |
| A-10 | addressed | Lines live in lines.go, which misspell and codespell exclude. |  |
| A-11 | addressed | The arguments are described correctly as seed halves, not a PCG 'stream'. Domains are disjoint and forbidigo bans the top-level functions. Nit: the AI domain 3&lt;&lt;56\|thinkSeq carries no game or play id. |  |
| A-12 | addressed | Pace constants are declared (pace.go). |  |
| A-13 | addressed | games holds types only; catalog does the wiring; the registry is injected; Info() is dropped; there are assets and testkit edges, test-import checks, and a constructor-completeness test. The cycle is gone. The remaining DAG omissions are under A-6. |  |
| A-14 | addressed | A pure proto/host Runner with a Scheduler. ui is an adapter. testkit and drive.go use a fake scheduler. The Esc machine is specified. |  |
| A-15 | addressed | Clear, Wait and SetLayout exist, and an empty Enter arrives as LineEvent{""}. Nit: the GTW mockup has 23 rows (script count), although L134 says exactly 80x24. Full layout (H-4 + 3 + input, L340) plus the front panel, on by default in norad (L341-342), adds up to H+1. |  |
| A-16 | partial | games/ending owns the climax, with Next/NoVerdict, NORAD notices, the HELLO prompt, Ending→Shell with the chess offer, and 0 players from a normal shell. Gaps: (1) the DAG gives ending only proto, prompt and assets, so its tic-tac-toe self-play must re-implement tictactoe, board or ai. (2) Launch{ending} needs a registry entry that the 16-entry catalog lacks; if one is added, Resolve's name/prefix rules let `wopr ending` start it. (3) How the persona enters the Ending phase from GameOver{Next} is only implied. Minor. | Ending may import tic-tac-toe; the host resolves `ending` from `catalog.Internal()` (M2); §6.2 rewritten. |
| A-17 | addressed | LOGON is a closed table with Dialing and Disconnected phases. The counter lives in Session. The dial is skippable. The two requested goldens are listed. |  |
| U-1 | addressed | Alt screen, every cell painted, OSC 11 rejected with a reason, exit line on 0 and 130. |  |
| U-2 | addressed | Corrections used: ansi.Strip first, DEL, tab→space, pasted lines joined, invalid UTF-8 replaced, grapheme cap. Nit: SanitizeText does not say how it handles invalid UTF-8. |  |
| U-3 | addressed | Key mode, the empty-line Enter/Space rule, PgUp/PgDn, the same rule while a request is pending, and new README wording. |  |
| U-4 | addressed | Corrected: only stdout and TERM=dumb are checked; Bubble Tea's TTY error maps to exit 2; any non-empty NO_COLOR counts. |  |
| U-5 | addressed | Gridless chess at PanelRows 12 with a double-spaced strip. The Hearts mockup moves to M3 with Hearts. Nit: the mockup shows PROCESSING on its own row above the cursor, while L311 puts it in the input line. |  |
| U-6 | addressed | Corrected: width uses the renderer's method. Nit: backspace by grapheme and horizontal scrolling are not specified. |  |
| U-7 | addressed | Hex, ANSI-16 and ASCII per Style, glyph and attribute redundancy, a 3 Hz cap, -r/--reduce-motion, and StyleMap goldens under 3 profiles. |  |
| U-8 | addressed | Paused while too small, Think held, keys dropped, no start before the first valid size, an 80x24 pty and a 60x20→80x24 e2e. |  |
| U-9 | closed | Documented as known behaviour and revisited in the M5 accessibility pass. This is one of the fix options the review offered. |  |
| U-10 | addressed | LOGOFF works at every depth with exit 0, Ctrl+D, SIGTERM→0, tea.Suspend, ErrInterrupted checked first, e2e covers both exits. Nit: BYE, which is in the review's list, is missing. |  |
| U-11 | addressed | Print-and-exit output is plain unstyled text in one write. |  |
| G-1 | addressed | corentings/chess/v2 v2.6.0, and Think captures a FEN (refute correction). |  |
| G-2 | addressed | The spec is the first M4 task. Fighter Combat and Air-to-Ground are bespoke (refute correction). The 150-line estimate is dropped. |  |
| G-3 | addressed | Corrected seat rule (declaring side, rotation) and redeal. |  |
| G-4 | addressed | Listed bool and a test that 1..15 appear once each. Nit: unlisted entries are not barred from unique-prefix matching. |  |
| G-5 | addressed | Corrected: gamestest is a non-_test package, plus injection. |  |
| G-6 | addressed | The refute's precise guarantees: reachable from the current cell, no wall on the player, consistent reveals. |  |
| C-1 | addressed | Corrected loop: detects the -- terminator and tests `-t -- gtw`. |  |
| C-2 | addressed | Accepted and rejected forms are pinned. Nit: `-i false` (a flag followed by a positional) is not pinned. |  |
| C-3 | addressed | SIGPIPE ignored, one buffered write, Windows errnos 232/109. |  |
| C-4 | addressed | -v is version on purpose; verbose will not use -V. |  |
| D-1 | closed | Owner decision: go 1.27 + toolchain go1.27.1. |  |
| D-2 | addressed | net/http only, no SDK, no build tag. |  |
| D-3 | addressed | No fallback renderer. |  |
| D-4 | partial | Fallback to ReadBuildInfo and 'unknown' are in. The review's 'normalise the v prefix' is missing: GoReleaser stamps '0.1.0' while --version shows 'wopr v0.1.0'. Nit. | Already done in code: `internal/version` adds the `v` prefix. |
| D-5 | addressed | Owner chose macOS 26 only; macos-26 and macos-26-intel runners; no Homebrew tap. |  |
| D-6 | partial | Correct: nothing sets GOTOOLCHAIN before setup-go. But archtest asserts the running Go equals the toolchain line with no condition. Under GOTOOLCHAIN=auto a contributor with a newer local Go keeps it, so `go test ./...` fails locally. The repo's TestToolchainMatchesGoMod had to skip unless GITHUB_ACTIONS=true. Minor. | Equality only in CI, `>=` locally. |
| B-1 | addressed | Explicit artifact names. Smoke uses the archive plus the e2e binary. Paths come from artifacts.json (refute correction). |  |
| B-2 | addressed | Build once, smoke natively on 6 runners. The 'exact binaries' claim is dropped in favour of release smoke plus repro. |  |
| B-3 | addressed | * text=auto eol=lf. |  |
| B-4 | addressed | cache:false in every GoReleaser job (refute correction). |  |
| B-5 | addressed | Labels pinned and checked at each milestone. An ubuntu:22.04 container backs the oldest-Linux claim. |  |
| B-6 | addressed | Block mapping with a conditional cancel (refute correction). |  |
| B-7 | addressed | The size gate is Go, reads artifacts.json and fails on an empty list (refute correction to nullglob). |  |
| B-8 | addressed | The README states only the budget. |  |
| B-9 | addressed | project_name, goos/goarch, zip override, checksum name, ldflags, -trimpath. |  |
| B-10 | partial | verify→build→smoke→repro→publish, with conditional cancel. Gap: the reusable smoke needs each target's e2e test binary (L899), but the release build (L908-910) never cross-compiles the e2e tests, so release smoke has no e2e binary to run. Minor. | Both pipelines stage the e2e test with `internal/tools/stage`. |
| B-11 | addressed | Archive files, a generated and CI-checked notices file, --licenses. Embedding them needs a module-root package (the repo's legal.go), which §4.1 omits (see A-6). |  |
| B-12 | addressed | Pinned mtimes and builds_info, plus a fresh-checkout repro job. |  |
| B-13 | addressed | PR artifacts get a -pr&lt;N&gt; suffix and 3-day retention; the README links main push runs only. |  |
| B-14 | addressed | No Makefile; tools in Go; -race per OS; e2e compiled per target. |  |
| T-1 | addressed | rumdl v0.2.78 with rumdl-check, gitleaks pinned by SHA, and the gate is `prek run --all-files` (correction). |  |
| T-2 | addressed | Uses the refute's pattern ^tea\.(Tick\|Every\|Sequence)$ with pkg and analyze-types, not the broken ^(Tick\|Every\|Sequence)$. A lint fixture self-tests it. Nit: the global ban on time.Sleep and NewTimer conflicts with M7's 'retries with backoff' (L560). |  |
| T-3 | addressed | SHAs with # frozen comments, and [update] freeze plus cooldown. |  |
| T-4 | addressed | govulncheck is a Go tool and runs in CI. Nit: L849 'Only tools/release must not raise the main module's go line' is garbled. The real rule (each tool module's go line ≤ the CI toolchain under GOTOOLCHAIN=local) is not stated, and GoReleaser v2.18.2 already needs exactly go 1.27.1. |  |
| T-5 | addressed | zizmor runs without --fix; GH_TOKEN goes to a separate step; the golangci-lint-full entry is overridden. |  |
| T-6 | addressed | No go-vet hook, and only additional linters are enabled. |  |
| T-7 | closed | Owner decision: no Dependabot and keep prek.toml. Updates follow a manual cadence plus a weekly report issue, with no PR-creating workflow (correction). |  |
| T-8 | addressed | Each fixer has its own priority and every hook has an explicit one. The golden-file side effect is under Q-3. |  |
| T-9 | partial | The install hook types, default_stages and the run --all-files gate are set. The review's 'stages on the five builtins' is not applied. Local experiment with prek 0.5.5: `prek list --hook-stage pre-push` still lists trailing-whitespace, end-of-file-fixer, the two shebang checks and check-added-large-files, so mutating fixers run on git push. Nit. | The four fixers now run on pre-commit and manual only. |
| T-10 | addressed | Uses `prek update`. `prek update --check` exists in 0.5.5 (prek update --help). |  |
| T-11 | addressed | prek-version is 0.5.5; GoReleaser and gitleaks run as go tools; privileges are split. |  |
| S-1 | addressed | Corrected: fetch-depth 0, full history, fail on ERR, an M0 canary, repo-level push protection. The broken before..sha range is gone. |  |
| S-2 | addressed | actions/attest with subject-checksums. |  |
| S-3 | addressed | ci-ok is the only required check; rulesets, push protection and SECURITY.md are covered. |  |
| S-4 | addressed | Explicit opt-in plus an in-character notice. |  |
| S-5 | addressed | Directory 0700 and file 0600, logging disabled when there is no cache dir, the path printed on exit, no TEA_DEBUG. |  |
| S-6 | closed | Moot: owner dropped Dependabot. |  |
| S-7 | addressed | Tag creation, update and deletion restricted; immutable releases; never re-tag, with retract; draft→publish; signer-workflow and source-ref flags. |  |
| S-8 | addressed | Allowlist, SHA pinning, read-only token, no PR creation, approval for all external contributors. |  |
| S-9 | addressed | No redirects, https except loopback, LimitReader, effects allowlist, fuzzed parser. |  |
| S-10 | addressed | Issue on findings, re-enable checklist, govulncheck in CI, 14-day response rule. |  |
| L-1 | addressed | Licences named inline; provenance tags (film, reconstructed, third-party, original, prompt) with a test; art drawn from scratch. |  |
| L-2 | addressed | WarGames used nominatively; no stills. |  |
| L-3 | addressed | Corrected: the gate covers prompt-tagged lines only, not M1. |  |
| L-4 | addressed | Placeholder map, abs0 credited, squash merge keeps v1 off main. Nit: L83 credits the backdoor header to the owner decision, but round 2 (d) covered only the montage names. The status burst is tagged 'A/F' (L1347), not one tag per line, though it is abs0 wargames.sh:179-184. |  |
| L-5 | addressed | An MIT or CC0 licence is needed before prompt lines ship or reach a provider. |  |
| L-6 | addressed | Recorded as risk R-2. |  |
| Q-1 | addressed | Corrected: xpty runs the real binary at 80x24; GREETINGS→LOGOFF→0; Ctrl+C→130; a resize case; a weaker ConPTY assertion; per-target compile. |  |
| Q-2 | addressed | One target per invocation; crashing inputs are uploaded. |  |
| Q-3 | partial | Corrected: internal/golden with WOPR_UPDATE_GOLDEN. Gap: the A.1 seed runs trailing-whitespace and end-of-file-fixer on goldens with no exclude, so prek would rewrite byte-exact goldens. The repo's prek.toml L25-26 had to add exclude='\.golden$'. Minor. | The fixers skip `.golden` files. |
| Q-4 | addressed | A quality test per game under depth limits. |  |
| M-1 | partial | Corrected: AGENTS.md from M0 and CLAUDE.md = @AGENTS.md. But the §4.1 DAG remains a second copy of archtest's table, and they already differ (see A-6). Minor. | §4.1 is declared a mirror of archtest's table, which is authoritative. |
| M-2 | addressed | Appendix A is deleted in M0. The M0 code has landed (454f6d9…) but Appendix A is still in the plan, and the repo prek.toml already diverges from it. |  |
| M-3 | addressed | app.go is a thin router. |  |
| M-4 | partial | Local experiment: rumdl 0.2.78 with the plan's args on a copy of docs/ reports only MD057 at PLAN.md:5, a link to the missing reviews/PLAN-v2-review.md. The plan's own rumdl-check gate fails until that file exists. Nit. | This document fixes the dangling link. |
| P-1 | addressed | GTW is self-contained. The withdrawn 'sim core in M2' alternative is not used. |  |
| P-2 | addressed | v0.1.0 after M2. Planned games decline with an original-tagged line. |  |
| P-3 | addressed | No estimates; the actuals sentence is reworded. |  |
| P-4 | addressed | prompt, the canvas renderer and Panel geometry are in M1. |  |
| F-1 | addressed | Reconstructed tags, with USER ACCOUNT and COUNTY settled in M5. |  |
| VERDICT-verification-log | addressed | The verification log is filled in. v2 no longer cites an empty log. |  |
| QUESTIONS-provenance | partial | Q4 was asked in round 2 (e), so Decision 8 is legitimately U. Answers are still not linked to a durable record, and some review-derived choices are labelled U (see R-1). | The U code now points at the owner-decisions record. |
| OWNER-DECISIONS | addressed | macOS 26, no Dependabot, prek.toml, PR-only with ci-ok and squash, montage credit to abs0 verified in M5, go 1.27 + go1.27.1, v0.1.0 after M2, minimal Bridge, GoReleaser once, 10/15 MB, movie M6 -m/--movie and LLM M7 are all implemented. Only the header-credit attribution (L-4) is wrong. |  |
| CORRECTIONS-USED | partial | Refute corrections used: T-2, S-1, T-3/T-10, B-6, C-1, G-5, Q-1, Q-3, M-1, D-6, B-4, B-7, U-2, U-4, U-6, A-3, A-7, A-9, A-11, G-1, G-2, G-3, G-4, L-3, Q-4, B-1/B-9. Not carried over: A-6 cli→theme, T-9 builtin stages, D-4 v-prefix, U-10 BYE, T-4 tool go-line rule. v2 also cites a nonexistent A-18 (L419), and its §16 risk ids R-1..R-8 collide with the review's R-1..R-3 (e.g. L734). | cli → theme, builtin stages, the `v` prefix and the tool go-line rule are in; `BYE` (U-10) is not adopted, since `LOGOFF`, `EXIT`, `QUIT` and Ctrl+D cover it. |
| GAP-impl-1 DAG/registry cycle | addressed | Covered by A-13. The remaining edge omissions are under A-6. |  |
| GAP-impl-2 persona cannot speak proto | addressed | A Program interface; every output, including Launch and Quit, declared in proto; Brain via Think; no AskBrain; persona Esc defined. |  |
| GAP-impl-3 host semantics only in ui | addressed | Covered by A-14. |  |
| GAP-impl-4 time-budgeted Think | addressed | Covered by A-3. |  |
| GAP-impl-5 ending owner | partial | See A-16. | See A-16. |
| GAP-impl-6 M1 locks protocol early | addressed | Covered by P-4. |  |
| GAP-impl-7 v2 not self-contained | partial | Text and mockups are now in Appendices B and C. Rule is restated with PhaseSet and AfterGame. Cross-references checked OK. Left over: a dangling A-18, a broken PLAN-v2-review.md link, the R-id collision, and no link or anchor check. Nit. | A-18, the link and the R-id collision are fixed. |
| GAP-impl-8 Windows contributor path | addressed | Covered by B-14. |  |
| GAP-impl-9 matrix required checks | addressed | ci-ok. |  |
| GAP-impl-10 -update flag | addressed | Covered by Q-3. The golden-fixer side effect is noted there. |  |
| GAP-ux-1 DAG edges to games/assets | addressed | Covered by A-13. |  |
| GAP-ux-2 LOGON loop owner | addressed | Covered by A-17. |  |
| GAP-ux-3 intent anchoring and offers | addressed | Covered by A-7. |  |
| GAP-ux-4 ending defined three ways | partial | See A-16: ending's DAG blocks reuse of tic-tac-toe, and the ending is not in the registry. | See A-16. |
| GAP-ux-5 clear/wait/layout | addressed | Covered by A-15. |  |
| GAP-ux-6 time budget, RNG streams, transcripts | addressed | Stream domains are defined. §14.4 now uses the drive.go film_path golden. |  |
| GAP-ux-7 Esc semantics | addressed | Two-Esc confirm with a 3 s window; Brain cancel line; ending cannot be aborted with Esc. |  |
| GAP-ux-8 too small | addressed | Covered by U-8. |  |
| GAP-ux-9 colour/NO_COLOR/flash | addressed | Covered by U-7. |  |
| GAP-ux-10 Sanitize \n \t | addressed | Covered by U-2. |  |
| GAP-ux-11 no normal quit | addressed | Covered by U-10. |  |
| GAP-ux-12 thinking indicator/front panel | addressed | PROCESSING indicator with a steady cursor; WOPR_PANEL; one-row panel. Nit: the row budget overflows (Full + panel = H+1; GTW mockup has 23 rows). |  |
| GAP-sec-1 secrets scans 0 commits | addressed | Covered by S-1. |  |
| GAP-sec-2 release not gated | partial | See B-10: release smoke lacks e2e binaries. | See B-10. |
| GAP-sec-3 mutable tags/releases | addressed | Covered by S-7. |  |
| GAP-sec-4 unpinned tools in actions | addressed | Covered by T-11. |  |
| GAP-sec-5 scheduled.yml decay | addressed | Covered by S-10. No PR creation is needed. |  |
| GAP-sec-6 required checks/Actions policies | addressed | Covered by S-3 and S-8. |  |
| GAP-sec-7 archives lack NOTICE | addressed | Covered by B-11. |  |
| GAP-sec-8 non-reproducible archives | addressed | Covered by B-12. |  |
| GAP-sec-9 montage copied / brother licence | addressed | Credit option per owner decision 18; provenance values defined; licence required. |  |
| GAP-sec-10 fork PR artifacts | addressed | Covered by B-13. |  |
| GAP-sec-11 M6 HTTP hardening | addressed | Covered by S-9. |  |

## Questions for the owner

None of these block M1; the plan's current answer is in brackets.

1. **Esc during the war** (RF-2). May Esc twice abandon GTW and the climax tic-tac-toe, like any game? [Yes,
   with an in-character remark about the abandoned war. The ending itself cannot be aborted.]
2. **`LIST GAMES` numbering** (RF-6). Keep the film's unnumbered list, with numbers still accepted and
   mentioned in `HELP`? [Yes, decision 21.]
3. **Movie mode with a scene** (RF-5). Should `wopr -m 2` play scene 2 only, or from scene 2 to the end?
   [From scene 2 to the end, then exit 0.]
4. **v1 history in the public repository** (SL-6). `NOTICE.md` now credits the CC BY-SA map fragments. Do you
   also want the pull request opened from a fresh branch without v1's commits? [Not done; the credit is enough,
   and a full purge would need GitHub Support anyway.]
