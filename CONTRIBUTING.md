# Contributing to wopr

Thanks for your interest. `wopr` is a small fan-made terminal homage to the WOPR computer from *WarGames*
(1983). We welcome bug reports, fixes, new tests and improvements to the games.

When you take part, you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Before you start

- **Bugs and ideas:** for anything larger than a small fix, open an issue first. Use the
  [bug report or feature request form](https://github.com/GhostofGoes/WOPR/issues/new/choose). This lets us
  agree on the approach before you spend time on it. Ask questions in
  [Discussions](https://github.com/GhostofGoes/WOPR/discussions).
- **A first task:** issues with the
  [good first issue](https://github.com/GhostofGoes/WOPR/labels/good%20first%20issue) label are small
  tasks for new contributors.
- **Security problems:** do not open a public issue. Follow [SECURITY.md](SECURITY.md).
- **Design:** the architecture, scope and milestones are in [docs/PLAN.md](docs/PLAN.md). The day-to-day
  conventions are in [AGENTS.md](AGENTS.md): the package rules, the single clock, how to add a game, and
  golden files. AGENTS.md applies to human and AI contributors alike.
- **Decisions and plans:** [GOVERNANCE.md](GOVERNANCE.md) says who decides and who has which role.
  [ROADMAP.md](ROADMAP.md) lists the plans for the next year, and what the project will not do.

## Set up

You need Git and Go. The repository pins its Go toolchain in `go.mod`. With Go's default
`GOTOOLCHAIN=auto`, any Go 1.21 or newer downloads the right version on first use. Some Linux
distributions set their Go to `GOTOOLCHAIN=local`. If `go` says that `go.mod` needs a newer Go, run
`go env -w GOTOOLCHAIN=auto` once, or install Go from [go.dev](https://go.dev/doc/install). The docs site's
[Contributing page](https://ghostofgoes.github.io/WOPR/contributing/) shows how to install Git, Go, prek
and a C compiler on Linux, macOS and Windows. It also shows how to preview the site itself.

```sh
git clone https://github.com/GhostofGoes/WOPR.git
cd WOPR
go test ./...
go run ./cmd/wopr
```

Install the hooks once with [prek](https://github.com/j178/prek). They run the formatters and linters
before each commit. They run the tests and `govulncheck` before each push:

```sh
prek install           # installs the pre-commit and pre-push hooks
prek run --all-files   # run everything on demand
```

## Making a change

1. Branch from `main`. `main` accepts only pull requests. Every PR needs the `ci-ok` check to pass. CI
   lints and scans for secrets. It tests on Linux, macOS and Windows. It builds all six release targets,
   the Linux packages and snaps, the Windows installer and the Mac app. It runs each binary natively. It
   tries each package, installer and app on its own system. It also builds the docs site.
2. Keep each change focused. New functionality must come with automated tests. A bug fix must come
   with a regression test. Regenerate golden files with `WOPR_UPDATE_GOLDEN=1 go test ./...`. Then read
   the diff.
3. Start each new source file with the two SPDX lines that the other files have. A test checks them.
4. Go code must follow [Effective Go](https://go.dev/doc/effective_go) and the
   [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments), and AI agents must follow the
   rules in [AGENTS.md](AGENTS.md). The formatters and linters in the hooks and in CI enforce them.
5. If players will notice the change, add a change note with
   `go tool -modfile=tools/release/go.mod changie new`. A note is one short, plain line that says what
   changed for them, such as "Fixed an issue with the Chess game". [AGENTS.md](AGENTS.md#change-notes) has
   the rules. Do not edit `CHANGELOG.md`. Releases build it from the notes.
6. Run `prek run --all-files` and `go test ./...` before you push. On Linux and macOS, also run
   `go test -race ./...`. The installers' scripts in `packaging/windows/`, `packaging/macos/` and
   `packaging/snap/` run only on their own systems. Because of that, CI's `windows`, `macos` and `snap`
   jobs are their real test. When you change those files, check those jobs on your branch's run.
7. Write commit messages that say what changed and why. We squash-merge PRs, so the PR title and
   description become the commit on `main`.

A maintainer reviews every pull request before it merges. The code owner in
[.github/CODEOWNERS](.github/CODEOWNERS) must approve it. [AGENTS.md](AGENTS.md#pull-requests) says what
the review checks, and what a pull request needs before it can merge.

## Film text and other third-party content

This project quotes short pieces of the film's on-screen text. Each quoted line carries a provenance tag:
`film`, `reconstructed`, `third-party:…`, `original` or `prompt`. A test fails when a line has no tag.

- Do not copy text, ASCII art or code from other projects unless their licence allows it. When it does,
  add the credit to [NOTICE.md](NOTICE.md) in the same PR.
- Write new screen text and art yourself, and tag it `original`. The program's icon is original too. Its
  sources are in `packaging/icons/src/`, and [packaging/icons/README.md](packaging/icons/README.md) says
  how to change it.
- Do not add film stills, audio or other media.

## Licence

By contributing, you agree that your contributions are licensed under the [MIT License](LICENSE). That
grant does not cover the film quotations and third-party text listed in [NOTICE.md](NOTICE.md).
