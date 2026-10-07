---
title: Contributing
weight: 7
description: Set up a development environment on Linux, macOS or Windows, run the checks, and send a pull request.
tabs:
  sync: true
---

Bug reports, fixes, new tests and improvements to the games are all welcome. By taking part you agree to
follow the [Code of Conduct](/contributing/code-of-conduct).

- **Bugs and ideas.** [Open an issue](https://github.com/GhostofGoes/WOPR/issues) first for anything
  larger than a small fix, so that we can agree on the approach before you spend time on it.
- **Security problems.** Do not open a public issue. Follow the
  [security policy](https://github.com/GhostofGoes/WOPR/blob/main/SECURITY.md).
- **Design and conventions.** The architecture, scope and milestones are in
  [docs/PLAN.md](https://github.com/GhostofGoes/WOPR/blob/main/docs/PLAN.md). The day-to-day rules (the
  package rules, the single clock, how to add a game, golden files, change notes) are in
  [AGENTS.md](https://github.com/GhostofGoes/WOPR/blob/main/AGENTS.md), which applies to human and AI
  contributors alike.

## Set up

You need Git, Go and prek. The race detector also needs a C compiler. Nothing else: the linters, the
release tools and Hugo, which builds this site, are pinned as Go tools and build themselves on first use.

**Go.** The repository pins its Go version in `go.mod`. Any Go from 1.21 on downloads that version the
first time you build, so the Go your system offers is fine.

**prek** runs the formatters and linters before each commit, and the tests before each push. It is a
single program; [prek's README](https://github.com/j178/prek#installation) lists every way to get it.

{{< tabs >}}

{{< tab name="Linux" >}}
Install Git, Go and a C compiler with your distribution's packages, for example:

```sh
sudo apt install git golang-go gcc     # Debian, Ubuntu
sudo dnf install git golang gcc        # Fedora
sudo pacman -S git go gcc              # Arch
```

If your distribution's Go is older than 1.21, install it from [go.dev](https://go.dev/doc/install)
instead. Then install prek with its installer, or with `brew install prek` if you use Homebrew:

```sh
curl --proto '=https' --tlsv1.2 -LsSf https://github.com/j178/prek/releases/download/v0.5.5/prek-installer.sh | sh
```

{{< /tab >}}

{{< tab name="macOS" >}}
The Xcode Command Line Tools bring Git and a C compiler. With [Homebrew](https://brew.sh/), install Go
and prek:

```sh
xcode-select --install
brew install go prek
```

Without Homebrew, install Go from [go.dev](https://go.dev/doc/install) and prek with its installer:

```sh
curl --proto '=https' --tlsv1.2 -LsSf https://github.com/j178/prek/releases/download/v0.5.5/prek-installer.sh | sh
```

{{< /tab >}}

{{< tab name="Windows" >}}
With winget, in PowerShell:

```powershell
winget install --id Git.Git
winget install --id GoLang.Go
winget install --id j178.Prek
```

Or install Go from [go.dev](https://go.dev/doc/install) and prek with its installer:

```powershell
powershell -ExecutionPolicy ByPass -c "irm https://github.com/j178/prek/releases/download/v0.5.5/prek-installer.ps1 | iex"
```

The race detector needs a C compiler, which Windows does not have. On amd64, install a MinGW-w64 `gcc`
(for example through [MSYS2](https://www.msys2.org/)), put it on your `PATH`, and set `CGO_ENABLED=1`.
The race detector does not run on Windows on arm64. CI runs it on Linux and macOS either way.

Use Windows Terminal to run `wopr` and its end-to-end tests.
{{< /tab >}}

{{< /tabs >}}

Then get the code, install the hooks, and check that everything works:

```sh
git clone https://github.com/GhostofGoes/WOPR.git
cd WOPR
prek install
go test ./...
go run ./cmd/wopr
```

`prek install` sets up both the pre-commit and the pre-push hooks.

{{% repo-file "AGENTS.md" "section:Commands" %}}

## Making a change

1. Branch from `main`, and keep each change focused. Add or update tests with it.
2. Regenerate golden files with `WOPR_UPDATE_GOLDEN=1 go test ./...` when a screen changes on purpose, and
   read the diff.
3. Run `prek run --all-files` and `go test ./...` before pushing. On Linux and macOS, also run
   `go test -race ./...`.
4. Add a change note if players will notice the change (see below).
5. Open a pull request. Write the title and description to say what changed and why: pull requests are
   squash-merged, so they become the commit on `main`.

{{% repo-file "AGENTS.md" "section:Pull requests" %}}

## Change notes

Every change a player could notice gets a short note, kept with [changie](https://changie.dev) in the
`.changes/unreleased/` folder. A release collects the notes into the [Changelog](/changelog), the GitHub
Release and the Linux packages' changelogs. Add one with changie's `new` command, pinned as a Go tool;
[AGENTS.md's Change notes](https://github.com/GhostofGoes/WOPR/blob/main/AGENTS.md#change-notes) gives
the exact command.

Write a note for players, in plain words at an 8th-grade reading level. Say what changed, not why. For a
deep technical fix, say only what the player saw: "Fixed a crash in some cases", "Fixed an issue with the
Chess game". Tests, CI, tooling and contributor docs need no note.

## Film text and other third-party content

This project quotes short pieces of the film's on-screen text. Every quoted line carries a provenance tag
(`film`, `reconstructed`, `third-party:...`, `original` or `prompt`), and a test fails without one.

- Do not copy text, ASCII art or code from other projects unless their licence allows it. When it does,
  credit it in [NOTICE.md](https://github.com/GhostofGoes/WOPR/blob/main/NOTICE.md) in the same pull
  request.
- Write new screen text and art yourself and tag it `original`.
- Do not add film stills, audio or other media.

## This site

The site is built with [Hugo](https://gohugo.io/) and the [Hextra](https://github.com/imfing/hextra)
theme, both pinned in the repository. Its pages are in `site/content/`; each game's page is built from
`site/data/games/<game>.json`, which the manual page uses too. Preview it at
<http://localhost:1313/WOPR/> with:

```sh
go tool -modfile=tools/docs/go.mod hugo server --source site
```

Every page has an "Edit this page" link that opens its source on GitHub.

## Licence

By contributing, you agree that your contributions are licensed under the
[MIT License](https://github.com/GhostofGoes/WOPR/blob/main/LICENSE). The film quotations and third-party
text listed in [Credits and notices](/credits) are not covered by that grant.
