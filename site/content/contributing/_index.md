---
title: Contributing
weight: 7
description: Set up a development environment on Linux, macOS or Windows, run the checks, and send a pull request.
tabs:
  sync: true
---

We welcome bug reports, fixes, new tests and improvements to the games. When you take part, you agree to
follow the [Code of Conduct](/contributing/code-of-conduct).

- **Bugs and ideas.** For anything larger than a small fix,
  [open an issue](https://github.com/GhostofGoes/WOPR/issues/new/choose) first. Use the bug report or
  feature request form. This lets us agree on the approach before you spend time on it. Ask questions in
  [Discussions](https://github.com/GhostofGoes/WOPR/discussions).
- **Security problems.** Do not open a public issue. Follow the
  [security policy](https://github.com/GhostofGoes/WOPR/blob/main/SECURITY.md).
- **Design and conventions.** The architecture, scope and milestones are in
  [docs/PLAN.md](https://github.com/GhostofGoes/WOPR/blob/main/docs/PLAN.md). The day-to-day rules are in
  [AGENTS.md](https://github.com/GhostofGoes/WOPR/blob/main/AGENTS.md): the package rules, the single
  clock, how to add a game, golden files, and change notes. AGENTS.md applies to human and AI
  contributors alike.

## Set up

You need Git, Go and prek. The race detector also needs a C compiler. You need nothing else. The
repository pins the linters, the release tools and Hugo, which builds this site, as Go tools. They build
themselves on first use.

**Go.** The repository pins its Go version in `go.mod`. Any Go from 1.21 on downloads that version the
first time you build. A Go set to `GOTOOLCHAIN=local` does not download it (see the Linux tab).

**prek** runs the formatters and linters before each commit, and the tests before each push. It is a
single program. [prek's README](https://github.com/j178/prek#installation) lists every way to get it.

{{< tabs >}}

{{< tab name="Linux" >}}
Install Git, Go and a C compiler with your distribution's packages, for example:

```sh
sudo apt install git golang-go gcc     # Debian, Ubuntu
sudo dnf install git golang gcc        # Fedora
sudo pacman -S git go gcc              # Arch
```

If your distribution's Go is older than 1.21, install Go from [go.dev](https://go.dev/doc/install)
instead. Some distributions set their Go not to download another version. If `go test` says that
`go.mod` needs a newer Go and mentions `GOTOOLCHAIN=local`, run `go env -w GOTOOLCHAIN=auto` once, or
install Go from go.dev. Then install prek with its installer, or with `brew install prek` if you use
Homebrew:

```sh
curl --proto '=https' --tlsv1.2 -LsSf https://github.com/j178/prek/releases/download/v0.5.5/prek-installer.sh | sh
```

{{< /tab >}}

{{< tab name="macOS" >}}
The Xcode Command Line Tools include Git and a C compiler. With [Homebrew](https://brew.sh/), install Go
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

The race detector needs a C compiler, which Windows does not have. On amd64, install a MinGW-w64 `gcc`,
for example through [MSYS2](https://www.msys2.org/). Put it on your `PATH`. Then set `CGO_ENABLED=1`.
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

`prek install` installs both the pre-commit and the pre-push hooks.

{{% repo-file "AGENTS.md" "section:Commands" %}}

## Making a change

1. Branch from `main`. Keep each change focused. Add or update tests with the change.
2. When a screen changes on purpose, regenerate golden files with `WOPR_UPDATE_GOLDEN=1 go test ./...`.
   Then read the diff.
3. Run `prek run --all-files` and `go test ./...` before you push. On Linux and macOS, also run
   `go test -race ./...`.
4. If players will notice the change, add a change note (see below).
5. Open a pull request. Write the title and description to say what changed and why. We squash-merge
   pull requests, so the title and description become the commit on `main`.

{{% repo-file "AGENTS.md" "section:Pull requests" %}}

## Change notes

Every change a player could notice gets a short note, kept with [changie](https://changie.dev) in the
`.changes/unreleased/` folder. A release collects the notes into the [Changelog](/changelog), the GitHub
Release, and the changelogs of the `.deb` and `.rpm` packages. Add a note with changie's `new` command,
which the repository pins as a Go tool.
[AGENTS.md's Change notes](https://github.com/GhostofGoes/WOPR/blob/main/AGENTS.md#change-notes) gives
the exact command.

Write a note for players, in plain words at an 8th-grade reading level. Say what changed, not why. For a
deep technical fix, say only what the player saw. For example: "Fixed a crash in some cases", "Fixed an
issue with the Chess game". Tests, CI, tooling and contributor docs need no note.

## Film text and other third-party content

This project quotes short pieces of the film's on-screen text. Each quoted line carries a provenance tag:
`film`, `reconstructed`, `third-party:...`, `original` or `prompt`. A test fails when a line has no tag.

- Do not copy text, ASCII art or code from other projects unless their licence allows it. When it does,
  add the credit to [NOTICE.md](https://github.com/GhostofGoes/WOPR/blob/main/NOTICE.md) in the same pull
  request.
- Write new screen text and art yourself, and tag it `original`.
- Do not add film stills, audio or other media.

## This site

[Hugo](https://gohugo.io/) and the [Hextra](https://github.com/imfing/hextra) theme build the site. The
repository pins both. The site's pages are in `site/content/`. Hugo builds each game's page from
`site/data/games/<game>.json`, which the manual page uses too. Preview the site at
<http://localhost:1313/WOPR/> with:

```sh
go tool -modfile=tools/docs/go.mod hugo server --source site
```

Every page has an "Edit this page" link that opens its source on GitHub. A change to the site goes
live when its pull request merges into `main`.

## Licence

By contributing, you agree that your contributions are licensed under the
[MIT License](https://github.com/GhostofGoes/WOPR/blob/main/LICENSE). That grant does not cover the film
quotations and third-party text listed in [Credits and notices](/credits).
