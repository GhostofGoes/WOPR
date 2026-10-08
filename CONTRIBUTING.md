# Contributing to wopr

Thanks for your interest. `wopr` is a small fan-made terminal homage to the WOPR computer from *WarGames*
(1983). Bug reports, fixes, new tests and improvements to the games are all welcome.

By taking part you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Before you start

- **Bugs and ideas:** open an issue first for anything larger than a small fix, so we can agree on the
  approach before you spend time on it.
- **Security problems:** do not open a public issue. Follow [SECURITY.md](SECURITY.md).
- **Design:** the architecture, scope and milestones are in [docs/PLAN.md](docs/PLAN.md). The day-to-day
  conventions (package rules, the single clock, how to add a game, golden files) are in
  [AGENTS.md](AGENTS.md), which applies to human and AI contributors alike.

## Set up

You need Git and Go. The repository pins its Go toolchain in `go.mod`. With Go's default
`GOTOOLCHAIN=auto`, any Go 1.21 or newer downloads the right version on first use. Some Linux
distributions set their Go to `GOTOOLCHAIN=local`; if `go` says `go.mod` needs a newer Go, run
`go env -w GOTOOLCHAIN=auto` once, or install Go from [go.dev](https://go.dev/doc/install). The docs site's
[Contributing page](https://ghostofgoes.github.io/WOPR/contributing/) shows how to install Git, Go, prek
and a C compiler on Linux, macOS and Windows, and how to preview the site itself.

```sh
git clone https://github.com/GhostofGoes/WOPR.git
cd WOPR
go test ./...
go run ./cmd/wopr
```

Install the hooks once with [prek](https://github.com/j178/prek). They run the formatters and linters
before each commit, and the tests and `govulncheck` before each push:

```sh
prek install           # installs the pre-commit and pre-push hooks
prek run --all-files   # run everything on demand
```

## Making a change

1. Branch from `main`. `main` only accepts pull requests, and every PR needs the `ci-ok` check to pass.
   CI lints, scans for secrets, tests on Linux, macOS and Windows, builds all six release targets and the
   Linux packages, runs each binary natively (and installs the packages on Linux), and builds the docs
   site.
2. Keep each change focused. Add or update tests with the change. Regenerate golden files with
   `WOPR_UPDATE_GOLDEN=1 go test ./...`, and review the resulting diff.
3. If players will notice the change, add a change note with
   `go tool -modfile=tools/release/go.mod changie new`: one short, plain line that says what changed for
   them, such as "Fixed an issue with the Chess game". [AGENTS.md](AGENTS.md#change-notes) has the rules.
   Do not edit `CHANGELOG.md`; releases build it from the notes.
4. Run `prek run --all-files` and `go test ./...` before pushing. On Linux and macOS, also run
   `go test -race ./...`.
5. Write commit messages that say what changed and why. PRs are squash-merged, so the PR title and
   description become the commit on `main`.

## Film text and other third-party content

This project quotes short pieces of the film's on-screen text. Each quoted line carries a provenance tag
(`film`, `reconstructed`, `third-party:…`, `original` or `prompt`), and a test fails without one.

- Do not copy text, ASCII art or code from other projects unless their licence allows it. When it does,
  add the credit to [NOTICE.md](NOTICE.md) in the same PR.
- Write new screen text and art yourself and tag it `original`.
- Do not add film stills, audio or other media.

## Licence

By contributing, you agree that your contributions are licensed under the [MIT License](LICENSE). The film
quotations and third-party text listed in [NOTICE.md](NOTICE.md) are not covered by that grant.
