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
| Manual page (`docs/man/wopr.6`) | `go run ./internal/tools/manpage`; CI runs it with `-check`. Lint it with `mandoc -T lint -W all docs/man/wopr.6` |
| Add a change note | `go tool -modfile=tools/release/go.mod changie new` (see [Change notes](#change-notes)) |
| Check the change notes and `CHANGELOG.md` | `go run ./internal/tools/relnotes -check` (a prek hook) |
| A release's notes | `go run ./internal/tools/relnotes -version X.Y.Z -out build/notes` (or `-snapshot`) |
| Release build (local dry run) | `go run ./internal/tools/relnotes -snapshot -out build/notes`, then, with `WOPR_NOTES_DIR=build/notes` in the environment, `go tool -modfile=tools/release/go.mod goreleaser release --snapshot --clean` |
| Size gate | `go run ./internal/tools/sizegate -expect 6` |
| Stage binaries and e2e tests | `go run ./internal/tools/stage` (`-archives -assets dist/release` also checks the archives and collects every release file) |
| Docs site: preview (localhost:1313/WOPR/) | `go tool -modfile=tools/docs/go.mod hugo server --source site` |
| Docs site: build as CI does | `go tool -modfile=tools/docs/go.mod hugo --source site --panicOnWarning --printPathWarnings --minify` (into `site/public/`) |

Go 1.27.1 is pinned in `go.mod` (`toolchain go1.27.1`); `GOTOOLCHAIN=auto` fetches it, and a newer local Go is
fine (CI checks the exact version). Tools are pinned in four modules, never `go run …@latest`:
`tools/go.mod` (gitleaks, govulncheck), `tools/lint/go.mod` (golangci-lint), `tools/release/go.mod`
(GoReleaser, changie) and `tools/docs/go.mod` (Hugo, the standard edition). They are separate because their
dependency graphs conflict. The docs site's theme, Hextra, is a Hugo module pinned in `site/go.mod`. No tool
module's `go` line, nor `site/go.mod`'s, may be newer than the root `toolchain` line (a test checks): bump the
toolchain first, then the tool.

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
- **Game pages and the manual page.** Every game in the catalog has `site/data/games/<slug>.json` (the slug
  `wopr --games` shows): summary, how to play, controls, at least three tips, and screenshots in
  `site/static/img/games/`. The docs site and the manual page are built from these files, so nothing else
  repeats them. `internal/tools/manpage`'s tests check each file against the catalog, and check that
  `docs/man/wopr.6` is what the tool generates. The page takes its version and date from the newest
  `.changes/vX.Y.Z.md`, so regenerate it after batching a release.
- **Accessibility** (`docs/PLAN.md` §4.5). `internal/ui/access_test.go` plays every game in the catalog from a
  playbook (every playable game must have one) and checks each screen: printable ASCII only, a prompt's text
  ends in `:` or `?`, the cursor at the end of the input line, a hint in key mode, no colour codes under
  `NO_COLOR`, the style pairs of `theme.Distinct` still apart without colour wherever a view draws both, no
  attribute the style already has in every theme (`Bright`, `Alert` and `Accent` are bold everywhere, so bold
  on them shows nothing), and nothing flashing more than three times a second. The theme tests cover every
  `proto.Style` for contrast and for meaning without colour.

## The program protocol (`internal/proto`)

The persona, every game, the ending and the movie director are `proto.Program`s:

- They get Events and return Outputs. They never block: anything slow is a `Think`, which runs off the UI
  goroutine. A `Think`'s `Fn` captures values only, never a pointer the program keeps using; copy a chess
  position as a FEN string.
- With `Env.Deterministic` (`--seed` or `WOPR_SEED`, tests, movie mode), searches stop at `Think.Limit`.
  Wall-clock `Budget` is only a cap.
- Input mode is dynamic: `Prompt` asks for a line, `AwaitKeys` for keys. The host owns Esc; programs never
  see it, except a root program that captures keys (`AwaitKeys{Capture: true}`, movie mode).
- Colour is semantic: draw with `proto.Style` values; themes decide the colours. Meaning must not depend on
  colour alone (vary the glyph or use `AttrReverse`), nor on an attribute the style already has (see
  Accessibility above).

## Adding a game

1. Add a package under `internal/games/<slug-without-dashes>/` implementing `games.Game`.
2. Wire its constructor in `internal/games/catalog` and set `Status: games.Playable`.
3. Meet the definition of done in `docs/PLAN.md` §6.1: rules, legal AI through `Think`, a deterministic
   testkit transcript, a quality test, fits 80×24, README line, provenance tags.
4. Add a playbook for it in `internal/ui/access_test.go` that reaches its main screens, with `want` naming
   what they must show, so the accessibility sweeps check them (`TestEveryGameHasAPlaybook` fails without one).
5. Write its page, `site/data/games/<slug>.json`, from the code: how to play, the controls, and tips that
   are true of its AI. Add two or three 80×24 screenshots as `site/static/img/games/<slug>-<n>.png`, each with
   a caption that says what it shows. Then run `go run ./internal/tools/manpage`.

## Film text and provenance

Every script line, scene step and asset carries a provenance tag (movie scenes included: `internal/movie/scenes`,
where the user's typed lines are `script.User` lines that keep their mixed case): `film`, `reconstructed`,
`third-party:<repo>@<commit>:<path>` (or `third-party:<https URL>` for art from a web page, which is kept as
drawn and exempt from the capitals rule), `original` or `prompt`. Do not copy text or art from other projects
without a licence and a `NOTICE.md` entry that credits the source. Lines tagged `prompt` stay out of builds until the brother's
licence is recorded.

## Change notes

Every change a player could notice gets a change note: a small file in `.changes/unreleased/`, kept with
[changie](https://changie.dev). A release collects the notes into `CHANGELOG.md`, the GitHub Release, and the
`.deb` and `.rpm` changelogs.

- **Add one** with `go tool -modfile=tools/release/go.mod changie new`. It asks for the kind and the text;
  `--kind Fixed --body "..."` skips the questions. The kinds are Keep a Changelog's: Added, Changed,
  Deprecated, Removed, Fixed and Security. Write one note per change; edit the file to fix a note.
- **Write it for players**, at an 8th-grade reading level:
  - Use short, plain words, on one line of at most 76 characters (changie and CI enforce the length).
  - Say what changed. Do not explain why, or how the code does it.
  - For a deep technical fix, say only what the player saw that is now fixed: "Fixed a crash in some
    cases", "Fixed an issue with the Chess game".
  - Start a fix with "Fixed". Name games as `LIST GAMES` does. Put commands and flags in backticks
    (`` `--seed` ``). No links.
  - For example: "`WOPR_SEED` sets the seed, as `--seed` does." or "The chess board is now square."
- **When a note is not needed.** Tests, CI, tooling, refactoring and contributor docs need none. CI's
  `relnotes -since origin/main` asks for a note when a branch changes a file under `cmd/` or `internal/`
  that is not a test, test data, or in a test-only package (`internal/tools`, `archtest`, `e2e`, `golden`,
  `games/testkit`, `games/gamestest`). If players will notice nothing, add the trailer `Changelog: none`
  to a commit message on the branch, after a blank line, as the last line.
- **Never edit `CHANGELOG.md` by hand.** It is `changie merge`'s output, and the `change-notes` hook
  (`relnotes -check`) fails when it differs, when a note does not parse or is too long, or when a release
  tag has no `.changes/vX.Y.Z.md`.

## Releasing

The owner merges the release pull request, then pushes the tag. Everything else is prepared in the pull
request or done by `release.yml`.

1. **The release pull request**, titled `vX.Y.Z` (the version from `docs/PLAN.md` §15), batches the notes
   and rebuilds the changelog:

   ```sh
   go tool -modfile=tools/release/go.mod changie batch vX.Y.Z
   go tool -modfile=tools/release/go.mod changie merge
   ```

   The notes move from `.changes/unreleased/` into `.changes/vX.Y.Z.md`, and `CHANGELOG.md` gains the
   version. Read them once more as a player would; to change one, edit `.changes/vX.Y.Z.md` and run
   `changie merge` again. Every CI run's `build` job shows the notes the next release would get, in its
   summary.
2. **The owner merges it**, then tags the merge commit and pushes the tag:
   `git fetch origin && git tag vX.Y.Z origin/main && git push origin vX.Y.Z`.
3. **`release.yml` does the rest.** It waits for `main`'s CI, builds and checks every file, and publishes
   the GitHub Release with `.changes/vX.Y.Z.md` as its notes (`internal/tools/relnotes` adds a footer).
   The packages' changelogs carry the same notes.

If the pull request did not batch the notes, the release still gets them: the workflow batches
`.changes/unreleased/` itself, dated by the tagged commit, and warns. `main` then lags behind the release,
so CI fails on every branch until one pull request runs `go run ./internal/tools/relnotes -catch-up`. It
writes the missing `.changes/vX.Y.Z.md` from the tag, removes those notes from `.changes/unreleased/`,
and merges `CHANGELOG.md`; commit the result.

## Docs site

`site/` is the documentation site: Hugo with the Hextra theme, published to
<https://ghostofgoes.github.io/WOPR/> by `docs.yml` whenever `main` changes something it shows. CI's `docs`
job builds it on every push with `--panicOnWarning`, so a deprecated setting, a broken internal link or a
missing screenshot fails the build.

- Pages are Markdown in `site/content/`, written for players in plain, direct prose.
- Each game's page is built from `site/data/games/<slug>.json` by `site/content/games/_content.gotmpl`,
  which documents the schema; the manual page reads the same files. Change a game's text there.
- Nothing is copied into the site. `CHANGELOG.md`, `CODE_OF_CONDUCT.md`, `NOTICE.md` and this file's
  Commands and Pull requests sections are mounted and rendered by the `repo-file` shortcode; the README's
  screenshots and the manual page are mounted too (`site/hugo.yaml`).
- A screenshot goes in with the `screenshot` shortcode and a caption that says what the screen shows. PNGs
  are 960×564, made smaller with `optipng -o2`. One that shows the big board's map gets the map's credit
  line.
- Tests in `internal/cli` check that the Usage page lists every option and environment variable, and the
  Movie scenes page every scene.

## Pull requests

- `main` is PR-only. The required check is `ci-ok`. Merges are squash merges.
- A change players can notice adds a change note ([Change notes](#change-notes)).
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
   `internal/assets/`, `internal/movie/scenes/`, and the games' film text); the README's screenshots in
   `docs/screenshots/` and the docs site's in `site/static/img/` (game screenshots in
   `site/static/img/games/`) show some of it, and the docs site's pages (`site/content/`,
   `site/data/games/`) quote some, which the manual page (`docs/man/wopr.6`) repeats.
2. Remove or replace it in one pull request, and release a patch version. Merging it also redeploys the
   docs site (`docs.yml`).
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
- **Pages:** Settings → Pages → Build and deployment → Source: **GitHub Actions**. `docs.yml` then
  deploys the docs site to <https://ghostofgoes.github.io/WOPR/> through the `github-pages` environment,
  which GitHub creates and limits to deployments from `main`.

## Milestone checklist

At every milestone boundary:

1. Update dependencies: `go get -u ./... && go mod tidy`, then update the tool modules with
   `go get -tool <tool>@latest` in `tools/`, `tools/lint/`, `tools/release/` and `tools/docs/`, and the
   docs theme with `go -C site get github.com/imfing/hextra@latest`. Build the docs site: a new Hugo can
   deprecate a setting, which `--panicOnWarning` turns into an error.
2. Run `prek update`, and keep golangci-lint and gitleaks in step between `prek.toml` and their tool modules.
3. Bump action SHAs from their release tags.
4. Regenerate the notices.
5. Check the hosted runner labels in `.github/workflows` against GitHub's announcements.
6. Re-enable `scheduled.yml` if GitHub disabled it after 60 quiet days.
7. Update `docs/PLAN.md` and this file.
