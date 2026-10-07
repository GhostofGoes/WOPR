# AGENTS.md

Instructions for anyone, human or AI, changing this repository. They are the single source for commands
and conventions. The design and milestones are in [docs/PLAN.md](docs/PLAN.md); this file wins where the
two disagree on day-to-day practice. Update both in the same PR when a convention changes.

## What this is

`wopr` is a Go terminal-UI recreation of the WOPR computer from *WarGames* (1983). It is a single static
binary: Bubble Tea v2 renders it, Lip Gloss v2 styles it, and there is no network code. It is built in the
milestones of `docs/PLAN.md` §15.

## Commands

All commands run from the repository root and work on Linux, macOS and Windows. There is no Makefile.

| Task | Command |
|---|---|
| Run | `go run ./cmd/wopr` |
| Test | `go test ./...` |
| Test with the race detector | `go test -race ./...` (Linux and macOS; Windows amd64 needs a C compiler; not on windows/arm64) |
| End-to-end tests (real binary in a pty) | `go test -tags e2e ./internal/e2e` |
| Lint rules self-test | `WOPR_LINT_SELFTEST=1 go test -run TestLintRulesFire ./internal/archtest` |
| Regenerate golden files | `WOPR_UPDATE_GOLDEN=1 go test ./...`, then review the diff |
| All hooks | `prek run --all-files` (install once with `prek install`) |
| Lint | `go tool -modfile=tools/lint/go.mod golangci-lint run ./...` |
| Format | `go tool -modfile=tools/lint/go.mod golangci-lint fmt ./...` |
| Vulnerabilities | `go tool -modfile=tools/go.mod govulncheck ./...` |
| Secret scan (full history) | `go tool -modfile=tools/go.mod gitleaks git --redact .` |
| Third-party notices | `go run ./internal/tools/notices`; CI runs it with `-check` |
| Release build (local dry run) | `go tool -modfile=tools/release/go.mod goreleaser release --snapshot --clean` |
| Size gate | `go run ./internal/tools/sizegate -expect 6` |
| Stage binaries and e2e tests | `go run ./internal/tools/stage` (`-archives -assets dist/release` also checks the archives and collects every release file) |

Go 1.27.1 is pinned in `go.mod` (`toolchain go1.27.1`); `GOTOOLCHAIN=auto` fetches it, and a newer local Go is
fine (CI checks the exact version). Tools are pinned in three modules, never `go run …@latest`:
`tools/go.mod` (gitleaks, govulncheck), `tools/lint/go.mod` (golangci-lint) and `tools/release/go.mod`
(GoReleaser). They are separate because their dependency graphs conflict. No tool module's `go` line may be
newer than the root `toolchain` line (a test checks): bump the toolchain first, then the tool.

## Rules that tests enforce

- **Import DAG.** `internal/archtest` checks every import, test imports included, against the table in
  `internal/archtest/archtest_test.go`, which mirrors `docs/PLAN.md` §4.1. A new package needs a row in
  both. Only `internal/ui` imports Bubble Tea. Only `internal/ui` and `internal/theme` import
  `charm.land/*`. `games` and `wopr` never touch the terminal.
- **One clock.** Only `internal/ui/clock.go` may call `tea.Tick`, `tea.Every` or `time.Sleep`-style
  timers; `tea.Sequence` is banned everywhere (forbidigo).
- **Seeded randomness.** Use `proto.NewRand(seed, streamID)` with a stream from the domains in
  `internal/proto/rand.go`. The top-level `math/rand/v2` functions are banned.
- **Lint rules must fire.** `internal/archtest/testdata/lintfixture` breaks each custom rule on purpose,
  and the self-test requires every rule to report it. Tools pinned both in `prek.toml` and in a tool module
  (golangci-lint, gitleaks) must have the same version.
- **Notices.** `THIRD_PARTY_NOTICES.txt` must match `go run ./internal/tools/notices` for all six targets.

## The program protocol (`internal/proto`)

The persona, every game, the ending and the movie director are `proto.Program`s:

- They get Events and return Outputs. They never block: anything slow is a `Think`, which runs off the UI
  goroutine. A `Think`'s `Fn` captures values only, never a pointer the program keeps using; copy a chess
  position as a FEN string.
- With `Env.Deterministic` (`--seed`, tests, movie mode), searches stop at `Think.Limit`. Wall-clock
  `Budget` is only a cap.
- Input mode is dynamic: `Prompt` asks for a line, `AwaitKeys` for keys. The host owns Esc; programs never
  see it.
- Colour is semantic: draw with `proto.Style` values; themes decide the colours. Meaning must not depend on
  colour alone (vary the glyph or use `AttrReverse`).

## Adding a game

1. Add a package under `internal/games/<slug-without-dashes>/` implementing `games.Game`.
2. Wire its constructor in `internal/games/catalog` and set `Status: games.Playable`.
3. Meet the definition of done in `docs/PLAN.md` §6.1: rules, legal AI through `Think`, a deterministic
   testkit transcript, a quality test, fits 80×24, README line, provenance tags.

## Film text and provenance

Every script line, scene step and asset carries a provenance tag: `film`, `reconstructed`,
`third-party:<repo>@<commit>:<path>` (or `third-party:<https URL>` for art from a web page, which is kept as
drawn and exempt from the capitals rule), `original` or `prompt`. Do not copy text or art from other projects
without a licence and a `NOTICE.md` entry that credits the source. Lines tagged `prompt` stay out of builds until the brother's
licence is recorded.

## Pull requests

- `main` is PR-only. The required check is `ci-ok`. Merges are squash merges.
- CI (`ci.yml`) runs on every branch push, so a branch is checked before its PR; a PR from a branch here is
  checked by that branch's push run (its `pull_request` run skips every job), and a PR from a fork runs in
  full. The push run tests the branch as it is, not merged with `main`, so the ruleset requires branches to
  be up to date: updating one is a push, which tests the result.
- Run `prek run --all-files` and `go test ./...` before pushing.
- Never re-tag a release. A bad release is fixed with the next patch version and a `retract` in `go.mod`.

## Secret scanning

The `secrets` job runs gitleaks over the checked-out history (the pushed branch, or a fork's pull request
merged with `main`), with `--no-color` so its `ERR` lines can be detected; it fails on a finding, on any
error, and on an empty scan. A finding is handled as `SECURITY.md` describes: rotate the secret, then add its
fingerprint to `.gitleaksignore` by pull request.

To prove the job (the M0 canary), push a throwaway branch containing a fake secret that only gitleaks
recognises (not a GitHub-supported token pattern, which push protection blocks), check that its CI run's
`secrets` log says `leaks found` rather than an error, then delete the branch.

## Takedown runbook

If a rights holder asks for material to be removed:

1. Find every copy: the provenance tags name each line's source (`internal/wopr/lines.go`,
   `internal/assets/`, `internal/movie/scenes/`, and the games' film text).
2. Remove or replace it in one pull request, and release a patch version.
3. Add a `retract` directive to `go.mod` for the affected versions, and delete the affected GitHub
   releases (immutable releases can be deleted, not edited; their tags cannot be reused).
4. Reply to the requester saying what was done. Copies remain in git history and in the Go module mirror,
   which has no documented removal process for this case.

## Repository settings (owner, once)

These live in GitHub settings, not in files. Check them at each milestone:

- **Branch ruleset on `main`:** require a pull request; require the `ci-ok` check from GitHub Actions, with
  "Require branches to be up to date before merging"; block force pushes and deletion. Allow squash merges
  only (merge commits and rebase merging off). `gh api 'repos/{owner}/{repo}/rules/branches/main'` should
  list `pull_request` and `required_status_checks` rules.
- **Tag ruleset on `v*`:** restrict creation, update and deletion to the owner. Turn on immutable releases
  before v0.1.0.
- **Security:** secret scanning with push protection, private vulnerability reporting.
- **Actions:**
  - Allow only `actions/*` and `j178/prek-action`, and require full-length commit-SHA pinning.
  - Set the default `GITHUB_TOKEN` to read-only. Workflows may not create or approve pull requests.
  - Require approval for workflow runs from all external contributors.
- **Dependabot:** off (owner decision). The weekly `scheduled.yml` report covers updates and
  vulnerabilities.

## Milestone checklist

At every milestone boundary:

1. Update dependencies: `go get -u ./... && go mod tidy`, then update the tool modules with
   `go get -tool <tool>@latest` in `tools/`, `tools/lint/` and `tools/release/`.
2. Run `prek update`, and keep golangci-lint and gitleaks in step between `prek.toml` and their tool modules.
3. Bump action SHAs from their release tags.
4. Regenerate the notices.
5. Check the hosted runner labels in `.github/workflows` against GitHub's announcements.
6. Re-enable `scheduled.yml` if GitHub disabled it after 60 quiet days.
7. Update `docs/PLAN.md` and this file.
