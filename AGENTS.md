# AGENTS.md

Instructions for anyone, human or AI, who changes this repository. They are the single source for
commands and conventions. The design and milestones are in [docs/PLAN.md](docs/PLAN.md). Where the two
disagree on day-to-day practice, this file wins. Update both in the same PR when a convention changes.

## What this is

`wopr` is a Go terminal-UI recreation of the WOPR computer from *WarGames* (1983). It is a single static
binary. Bubble Tea v2 renders it, and Lip Gloss v2 styles it. There is no network code. It is built in
the milestones of `docs/PLAN.md` §15.

`SECURITY.md` states what users can expect from `wopr`'s security, and `docs/ASSURANCE.md` explains why it
holds. A change that moves a trust boundary updates both: new network access, a new file or environment
variable, or a new program that `wopr` starts. `GOVERNANCE.md` names the roles, and `ROADMAP.md` the plans.

## Commands

All commands run from the repository root. They work on Linux, macOS and Windows, unless a row names a
system. There is no Makefile.

| Task | Command |
|---|---|
| Run | `go run ./cmd/wopr` |
| Test | `go test ./...` |
| Test with the race detector | `go test -race ./...` (Linux and macOS. Windows amd64 needs a C compiler. It does not run on windows/arm64) |
| End-to-end tests (real binary in a pty) | `go test -tags e2e ./internal/e2e` |
| Lint rules self-test | `WOPR_LINT_SELFTEST=1 go test -run TestLintRulesFire ./internal/archtest` |
| Regenerate golden files | `WOPR_UPDATE_GOLDEN=1 go test ./...`, then review the diff |
| All hooks | `prek run --all-files` (install once with `prek install`) |
| Lint | `go tool -modfile=tools/lint/go.mod golangci-lint run ./...` |
| Format | `go tool -modfile=tools/lint/go.mod golangci-lint fmt ./...` |
| Vulnerabilities | `go tool -modfile=tools/go.mod govulncheck ./...` |
| Dependency cooldown (no module newer than 14 days) | `go run ./internal/tools/cooldown` |
| Secret scan (full history) | `go tool -modfile=tools/go.mod gitleaks git --redact .` |
| Third-party notices and the `.deb`'s copyright file | `go run ./internal/tools/notices` (writes `THIRD_PARTY_NOTICES.txt` and `packaging/debian/copyright`). CI runs it with `-check` |
| Manual page (`docs/man/wopr.6`) | `go run ./internal/tools/manpage`. CI runs it with `-check`. Lint it with `mandoc -T lint -W all docs/man/wopr.6` |
| Icons (`packaging/icons/`, the docs site's favicons) | `go run ./internal/tools/icons` (draws every icon file from `packaging/icons/src/`. `packaging/icons/README.md` says how to change the icon). CI runs it with `-check` |
| Add a change note | `go tool -modfile=tools/release/go.mod changie new` (see [Change notes](#change-notes)) |
| Check the change notes and `CHANGELOG.md` | `go run ./internal/tools/relnotes -check` (a prek hook) |
| A release's notes | `go run ./internal/tools/relnotes -version X.Y.Z -out build/notes` (or `-snapshot`) |
| Release build (local dry run) | `go run ./internal/tools/relnotes -snapshot -out build/notes`, then, with `WOPR_NOTES_DIR=build/notes` in the environment, `go tool -modfile=tools/release/go.mod goreleaser release --snapshot --clean` |
| Size gate | `go run ./internal/tools/sizegate -expect 6 -packages 4`. `-files <file>...` gates the installers (the `collect` jobs) |
| Stage binaries and e2e tests | `go run ./internal/tools/stage`. With `-archives -assets dist/release`, it also checks the archives and the Linux packages, and it collects every release file into `dist/release`. With `-merge -assets dist/release <file>...`, it adds the installers to them and to `checksums.txt`, sorted as GoReleaser sorts it |
| Lint the Linux packages (Linux, after a release build. The tools are not pinned) | `lintian --pedantic dist/*.deb` and, with Fedora's rpmlint configuration, `rpmlint -r packaging/rpmlintrc dist/*.rpm`. See `docs/PLAN.md` §8 for what they report. List an `.rpm`'s files with `rpm -qlvp`, or unpack it with `bsdtar -xf X.rpm -C dir`. Never pipe `rpm2cpio` into a plain `cpio -idm`, which writes into `/` (the payload's paths are absolute) |
| Check the menu entries and the AppStream metadata (Linux, after a release build. The tools are not pinned) | `desktop-file-validate build/pkg/deb/*.desktop build/pkg/rpm/*.desktop` (prints nothing when they pass), `appstreamcli validate --no-net --pedantic build/pkg/*.metainfo.xml` and `appstream-util validate-relax --nonet build/pkg/*.metainfo.xml`. The `desktop-files` hook runs them on what `pkgdocs` writes (its `TestValidators`, which skips a validator that is not installed unless `WOPR_DESKTOP_VALIDATORS=1`). The smoke jobs run them on the installed `.deb` |
| Windows installer and MSIX packages (Windows, PowerShell 7, after a release build and `stage -archives -assets dist/release`) | `pwsh -File packaging/windows/check-release-files.ps1 -Version <V> -BinDir dist/release`, then `pwsh -File packaging/windows/build-installer.ps1 -Version <V> -BinDir dist/release -OutputDir build/installer` (downloads the pinned Inno Setup) and `pwsh -File packaging/windows/build-msix.ps1 -Version <V> -BinDir dist/release -OutputDir build/msix` (needs the Windows SDK) |
| Test them (Windows, only on a machine without WOPR, such as a CI runner) | `pwsh -File packaging/windows/test-installer.ps1 -Installer build/installer/wopr_<V>_windows_setup.exe -Version <V>` (changes the user's `PATH` while it runs) and, as an administrator, `pwsh -File packaging/windows/test-msix.ps1 -PackageDir build/msix -Version <V>` (trusts a throwaway certificate while it runs) |
| `wopr.exe`'s icon, manifest and version details by hand | `go tool -modfile=tools/release/go.mod go-winres make --in packaging/windows/winres.json --out cmd/wopr/rsrc --arch amd64,arm64` (GoReleaser's before hook runs it. Git ignores the `.syso` files go-winres writes) |
| macOS app bundle (any OS) | `go run ./internal/tools/macapp -binary <darwin program> -version <V> -out build/WOPR.app` (`-notices dist/release` takes the notices from the release files) |
| macOS disk image (macOS, after a release build and `stage -archives -assets dist/release`) | `packaging/macos/build-dmg.sh dist/release <V> build/dmg`, then `packaging/macos/test-dmg.sh build/dmg/wopr_<V>_macos.dmg <V> [stage/darwin_<arch>/e2e.test]`. `packaging/macos/test-launch.sh build/dmg/wopr_<V>_macos.dmg` opens the app as Finder does, and it stops Terminal afterwards. Run it on a CI runner, or on a Mac where that does not matter |
| Snaps (Linux with snapd, after a release build and `stage -archives -assets dist/release`) | `go run ./internal/tools/snapdir -dist dist/release -version <V> -arch amd64 -out build/snap/amd64`, then `snap pack build/snap/amd64 build/snaps` (without snapd, `mksquashfs build/snap/amd64 build/snaps/wopr_<V>_amd64.snap -noappend -comp xz -no-fragments -all-root -no-xattrs` packs it as snapd does). The Store's checks: `sudo snap install review-tools`, then `review-tools.snap-review <snap>` |
| Test a snap (Ubuntu, sudo. The script installs and removes the snap) | `packaging/snap/test-snap.sh build/snaps/wopr_<V>_amd64.snap <V> [stage/linux_amd64/e2e.test]` |
| Docs site: preview (localhost:1313/WOPR/) | `go tool -modfile=tools/docs/go.mod hugo server --source site` |
| Docs site: build as CI does | `go tool -modfile=tools/docs/go.mod hugo --source site --panicOnWarning --printPathWarnings --minify` (into `site/public/`) |

`<V>` is the build's version as GoReleaser writes it in `dist/metadata.json`: `1.2.3`, or
`1.2.4-snapshot.abc1234` for a snapshot. The installers' scripts run only on their own systems. CI runs
them (`windows.yml`, `macos.yml`, `snap.yml`).

`go.mod` pins Go 1.27.2 (`toolchain go1.27.2`). `GOTOOLCHAIN=auto` downloads it, and a newer local Go is
fine (CI checks the exact version). Four modules pin the tools: `tools/go.mod` (gitleaks, govulncheck),
`tools/lint/go.mod` (golangci-lint), `tools/release/go.mod` (GoReleaser, changie, go-winres) and
`tools/docs/go.mod` (Hugo, the standard edition). Never use `go run …@latest`. The four modules are
separate because their dependency graphs conflict. The docs site's theme, Hextra, is a Hugo module that
`site/go.mod` pins. No tool module's `go` line, nor `site/go.mod`'s, may be newer than the root `toolchain`
line (a test checks). Bump the toolchain first, then the tool. Inno Setup is not a Go tool.
`build-installer.ps1` pins it by URL, SHA-256 and Authenticode signer. `snap.yml` installs snapcraft from
its `9.x/stable` channel, and review-tools from `latest/stable`.

## Rules that tests enforce

- **Import DAG.** `internal/archtest` checks every import, test imports included, against the table in
  `internal/archtest/archtest_test.go`. The table mirrors `docs/PLAN.md` §4.1. A new package needs a row
  in both. Only `internal/ui` imports Bubble Tea. Only `internal/ui` and `internal/theme` import
  `charm.land/*`. `games` and `wopr` never access the terminal. archtest lists the imports for linux,
  darwin and windows, since some files build for one only. Only `cmd/wopr` (its macOS build reopens the
  app in Terminal), `internal/tools/...`, `internal/archtest` and `internal/e2e` import `os/exec`.
- **One clock.** Only `internal/ui/clock.go` may call `tea.Tick`, `tea.Every` or `time.Sleep`-style
  timers (tests may use `time`'s). forbidigo bans `tea.Sequence` everywhere.
- **Seeded randomness.** Use `proto.NewRand(seed, streamID)` with a stream from the domains in
  `internal/proto/rand.go`. The top-level `math/rand/v2` functions are banned.
- **Licence headers.** Every source file that git tracks (Go, shell, PowerShell, Inno Setup, the docs
  site's JavaScript, CSS and layouts) starts with an `SPDX-FileCopyrightText:` line and
  `SPDX-License-Identifier: MIT`, after any `#!` line (`internal/archtest/license_test.go`). Vendored
  files keep their own.
- **Lint rules must fire.** `internal/archtest/testdata/lintfixture` breaks each custom rule on purpose.
  The self-test requires every rule to report it. A tool pinned both in `prek.toml` and in a tool module
  (golangci-lint, gitleaks) must have the same version in both.
- **Film lines are frozen.** Each package that holds lines tagged `film` lists them in its
  `testdata/film_lines.golden`, through `script.FilmTexts`. A change to that file is a change to the film's
  text, which never happens without new evidence from the film and the owner's decision ([Film text and
  provenance](#film-text-and-provenance)).
- **Dependency cooldown.** Regular updates wait 14 days. No module in any build list (wopr's, the four
  tool modules' and `site/go.mod`'s) may be less than 14 days old at the commit being checked. A module's
  age counts from the time the module proxy gives for its version. `go run ./internal/tools/cooldown`
  checks it (CI's `lint` job, and a prek hook on `go.mod` and `go.sum`). The Go toolchain follows the same
  rule, from its release date.
  - **Security fixes do not wait.** An update that fixes a published vulnerability (a `GO-`, `GHSA-` or
    `CVE-` ID) may be newer than 14 days. Use the oldest version that has the fix. Put each such module
    in `tools/cooldown-exceptions.txt`, with the advisory IDs as its reason. Remove the line when the
    version is 14 days old. A toolchain fix needs no line: name the advisories in the commit message.
  - Any other update waits, even one that a security fix brings. Update only the modules that carry
    the fix.
- **Notices.** `THIRD_PARTY_NOTICES.txt` and `packaging/debian/copyright` (the `.deb`'s machine-readable
  copyright file) must match `go run ./internal/tools/notices`. The copyright file lists the files that
  quote the film. The tool finds them from their provenance tags, so regenerate the file when film text
  moves. The tool files each linked module under Expat or Go's BSD-3-clause only when the module's
  licence has the same words. A module under any other licence stops the tool until you add it there.
- **Linux packages.** `.goreleaser.yaml`'s `nfpms` build a `.deb` and an `.rpm` for each Linux build
  (`docs/PLAN.md` §8). The `.deb` follows Debian Policy: the program in `/usr/games`, the manual page in
  section 6, the documents in `/usr/share/doc/wopr`. The `.rpm` follows Fedora: `/usr/bin`, `%license`,
  `%doc`. `stage -archives` checks what each installs. `relnotes`'s tests check that the packages'
  maintainer and release match the changelogs it writes. Change both together. `pkgdocs` writes the
  packages' documents and their `changelog.yml` into `build/pkg`. Its tests check
  `packaging/description.txt`, both packages' description, against both formats' rules. Both packages
  also put WOPR in the desktop's menu, in files named by the app ID `io.github.ghostofgoes.wopr`. The
  files are a menu entry that opens a terminal running `wopr`, the icon in every hicolor size, and
  AppStream metadata. `pkgdocs` writes the menu entry for each package and the metadata from their
  templates in `packaging/linux/`. Its tests check them against the Desktop Entry Specification and
  AppStream's rules. The tests also check that the screenshots and pages the files link exist in `site/`.
  `stage -archives` checks that the packages hold `build/pkg`'s files (by digest). It also checks that
  each menu entry starts that package's own program, and that the packages hold every icon size. The
  `.rpm`'s directories in `.goreleaser.yaml` also list the sizes, and so does `smoke.yml`. Change all
  three together. A release signs the packages with the key in `packaging/wopr-signing-key.asc`
  (`packaging/sign-packages.sh`: a signature inside each `.rpm`, a `.deb.asc` beside each `.deb`, and
  `checksums.txt.asc`). CI signs with a throwaway key. The smoke jobs install, run and remove both
  packages, with signature checks on in CI. They also check the installed menu entry and metadata.
  `pkgdocs`'s `TestValidators` runs the same validators on its own files, wherever the validators are
  installed. CI's lint job installs them, so none skips there.
- **Icons.** `internal/tools/icons` draws everything in `packaging/icons/` except `src/`, from the SVG
  sources in `packaging/icons/src/`. It also draws the docs site's `favicon.svg` (also its navbar logo),
  `favicon.ico` and `apple-touch-icon.png`. Its tests and `-check` fail when a file is stale. They compare
  pictures pixel by pixel, within 2/255, because floating point differs between CPUs. The `.ico` and
  `.icns` around them must be byte for byte what the tool writes from those pixels. The tool refuses a
  source it cannot draw or would never use. It also refuses a design that makes any file larger than
  512 KB. `packaging/icons/README.md` lists the sources and the SVG subset. The design, front-panel
  lamps, is the owner's choice (`docs/PLAN.md` §8).
- **Windows installer and MSIX.** `windows.yml` checks the release files against `checksums.txt`. It
  builds `wopr_<V>_windows_setup.exe` with Inno Setup (`packaging/windows/wopr.iss`: per user, no
  administrator prompt, both architectures in one file). It builds the MSIX packages for the Microsoft
  Store. Then it installs, runs and removes both on x64 and Arm64 Windows (`test-installer.ps1`,
  `test-msix.ps1`). Never change the installer's `AppId`: upgrades and uninstalling find WOPR by it.
  go-winres puts the icon, a manifest and version details into `wopr.exe` (`packaging/windows/winres.json`).
  The `.syso` files it writes are the same on every run, so `repro` still covers the program.
- **macOS app.** `macos.yml` checks the darwin programs against `checksums.txt` and joins them with
  `lipo`. It writes `WOPR.app` with `internal/tools/macapp` (whose tests check the bundle on any OS). It
  signs the app ad hoc and makes `wopr_<V>_macos.dmg` (`packaging/macos/build-dmg.sh`). On Apple silicon
  and on Intel, it then checks the image and runs the e2e tests against the app's program
  (`test-dmg.sh`). On Apple silicon, it also opens the app as Finder does (`test-launch.sh`: the Intel
  runner's Terminal runs nothing it is given). Started from Finder, with no terminal, `wopr` reopens
  itself in Terminal (`cmd/wopr/launch*.go`, with table tests). Started from a terminal, it runs there.
- **Snap.** `snap.yml` prepares each architecture's snap with `internal/tools/snapdir`. The tool checks
  the program and notices against `checksums.txt`, and the metadata against snapd's limits. It converts
  the Linux menu entry to the snap's. `snap pack` packs the snap. The Store's `review-tools` must pass
  it. `packaging/snap/test-snap.sh` installs, runs and removes it on both architectures. Publishing is
  off until the owner enables it (Repository settings).
- **Game pages and the manual page.** Every game in the catalog has `site/data/games/<slug>.json` (the slug
  `wopr --games` shows): summary, how to play, controls, and at least three tips. It also has screenshots
  in `site/static/img/games/`. The docs site and the manual page are built from these files, so nothing
  else repeats them. `internal/tools/manpage`'s tests check each file against the catalog, and check that
  `docs/man/wopr.6` is what the tool generates. The page takes its version and date from the newest
  `.changes/vX.Y.Z.md`, so regenerate it after you batch a release. The archives and packages carry the
  page with the version being built in its header instead (`pkgdocs`).
- **Accessibility** (`docs/PLAN.md` §4.5). `internal/ui/access_test.go` plays every game in the catalog from a
  playbook (every playable game must have one). It checks each screen for these conditions:
  - Printable ASCII only.
  - A prompt's text ends in `:` or `?`.
  - The cursor is at the end of the input line.
  - A hint in key mode.
  - No colour codes under `NO_COLOR`.
  - The style pairs of `theme.Distinct` are still apart without colour, wherever a view draws both.
  - No attribute the style already has in every theme (`Bright`, `Alert` and `Accent` are bold
    everywhere, so bold on them shows nothing).
  - Nothing flashes more than three times a second.

  The theme tests cover every `proto.Style` for contrast and for meaning without colour.

## The program protocol (`internal/proto`)

The persona, every game, the ending and the movie director are `proto.Program`s:

- They receive Events and return Outputs. They never block: anything slow is a `Think`, which runs outside
  the UI goroutine. A `Think`'s `Fn` captures values only, never a pointer the program keeps using. Copy a
  chess position as a FEN string. An Update or View that runs for 10 s (`internal/ui/watchdog.go`) is
  taken for a hang. So is one that is still running a second after SIGINT or SIGTERM. On a hang, wopr
  restores the terminal and exits 1.
- With `Env.Deterministic` (`--seed` or `WOPR_SEED`, tests, movie mode), searches stop at `Think.Limit`.
  Wall-clock `Budget` is only a cap.
- Input mode is dynamic: `Prompt` asks for a line, and `AwaitKeys` asks for keys. The host owns Esc.
  Programs never see it, except a root program that captures keys (`AwaitKeys{Capture: true}`, movie
  mode).
- Colour is semantic: draw with `proto.Style` values, and themes decide the colours. Meaning must not
  depend on colour alone (vary the glyph or use `AttrReverse`). It must also not depend on an attribute
  the style already has (see Accessibility above).

## Adding a game

1. Add a package under `internal/games/<slug-without-dashes>/` that implements `games.Game`.
2. Wire its constructor in `internal/games/catalog`. Set `Status: games.Playable`.
3. Meet the definition of done in `docs/PLAN.md` §6.1: rules, legal AI through `Think`, a deterministic
   testkit transcript, a quality test, fits 80×24, README line, provenance tags.
4. Add a playbook for it in `internal/ui/access_test.go` that reaches its main screens, with `want` naming
   what they must show. The accessibility sweeps then check them (`TestEveryGameHasAPlaybook` fails
   without one).
5. Write its page, `site/data/games/<slug>.json`, from the code: how to play, the controls, and tips that
   are true of its AI. Add two or three 80×24 screenshots as `site/static/img/games/<slug>-<n>.png`, each with
   a caption that says what it shows. Then run `go run ./internal/tools/manpage`.

## Film text and provenance

Every script line, scene step and asset carries a provenance tag. This includes the movie scenes in
`internal/movie/scenes`, where the user's typed lines are `script.User` lines that keep their mixed case.
The tag is `film`, `reconstructed`, `third-party:<repo>@<commit>:<path>` or `original`. Art from a web
page gets `third-party:<https URL>` instead. Such art is kept as drawn, and it is exempt from the capitals
rule. Do not copy text or art from other projects without a licence and a `NOTICE.md` entry that credits
the source. Do not commit any line adapted from the brother's prompt (the `prompt` tag) until he grants
written permission and a licence (`docs/PLAN.md` §2.1).

**The film's lines are never modified.** A line tagged `film` is the film's own screen text, character for
character, checked against the transcriptions in `docs/PLAN.md` §2.3 (design goal R15). Do not change its
words, spelling, punctuation, spacing, capitals or line breaks, for any reason:

- Not to fix what looks like a typo. `INDENTIFICATION NOT RECOGNIZED BY SYSTEM` is what the film's screen
  says, and `ANGENTINA ESCALATION` is what the montage shows.
- Not for a spell checker or a linter. Exempt the line instead, as `prek.toml`'s codespell ignore list and
  the `//nolint:misspell` comments do.
- Not for any writing standard or style rule ([Writing style](#writing-style)).

Every message WOPR shows that the film shows must be the film's. Where the film shows a line for a moment,
use the film's line; do not write one. The `film_lines.golden` files freeze the lines, so that a change to
one shows in review as a change to the film's text. Such a change needs new evidence from the film itself
and the owner's decision before it is made. The documentation quotes film lines exactly as the film shows
them, two spaces and misspellings included.

A line tagged `reconstructed` comes from one transcription only, or from a screen the film flashes by.
Change it only on better evidence, and tag it `film` once the rule in `docs/PLAN.md` §2.3 is met. That
section lists what is still open. A line tagged `original` is this project's own. Write it in WOPR's voice:
capitals, terse, and PROFESSOR for the user.

## Writing style

Technical documentation follows ASD-STE100, Simplified Technical English: the README, the docs site, the
manual page, the packages' descriptions, the change notes and the contributor docs. Write short sentences,
of at most 20 words, in the active voice and the simple tenses. Use one meaning per word, a list for a
sequence of steps, and no semicolons, phrasal verbs or idioms. Write for a reader who has never used a
terminal, except in the advanced subjects: verifying binaries, contributing and developing.

**This directive never overrides film authenticity, under any circumstances.** A quoted film line is
reproduced exactly as the film shows it, even where it breaks every rule above, in the documentation as in
the program. No documentation pass, rewrite, linter or spell check edits the film's text, the lines WOPR
shows or the goldens that hold them. If a film line reads badly inside a sentence, change the sentence
around the quote, never the quote.

## Change notes

Every change a player could notice gets a change note: a small file in `.changes/unreleased/`, kept with
[changie](https://changie.dev). A release collects the notes into `CHANGELOG.md`, the GitHub Release, and the
`.deb` and `.rpm` changelogs.

- **Add one** with `go tool -modfile=tools/release/go.mod changie new`. It asks for the kind and the text.
  `--kind Fixed --body "..."` skips the questions. The kinds are Keep a Changelog's: Added, Changed,
  Deprecated, Removed, Fixed and Security. Write one note per change. To fix a note, edit the file.
- **Write it for players**, at an 8th-grade reading level:
  - Use short, plain words, on one line of at most 76 characters (changie and CI enforce the length).
  - Say what changed. Do not explain why, or how the code does it.
  - For a deep technical fix, say only what the player saw that is now fixed: "Fixed a crash in some
    cases", "Fixed an issue with the Chess game".
  - Start a fix with "Fixed". Name games as `LIST GAMES` does. Put commands and flags in backticks
    (`` `--seed` ``). Do not put links in a note.
  - For example: "`WOPR_SEED` sets the seed, as `--seed` does." or "The chess board is now square."
- **When a note is not needed.** Tests, CI, tooling, refactoring and contributor docs need none. CI's
  `relnotes -since origin/main` asks for a note when a branch changes a file under `cmd/` or `internal/`,
  unless the file is a test, test data, or in a test-only package (`internal/tools`, `archtest`, `e2e`,
  `golden`, `games/testkit`, `games/gamestest`). It also asks for one when a branch changes a file that
  decides what the downloads hold: `.goreleaser.yaml`, `packaging/` or the manual page in `docs/man/`. If
  players will notice nothing, add `Changelog: none` as a trailer to a commit message on the branch. A
  trailer is a line in the message's last paragraph, next to any `Co-Authored-By:` lines. A line higher
  up in the message is not a trailer.
- **Never edit `CHANGELOG.md` by hand.** It is `changie merge`'s output. The `change-notes` hook
  (`relnotes -check`) fails in any of these cases:
  - `CHANGELOG.md` differs from that output.
  - A note does not parse, or is too long.
  - A release tag has no `.changes/vX.Y.Z.md`.

## Releasing

**Start to finish:**

1. On a new branch from `main` that holds nothing else, run the three commands below. Open a pull request
   titled `vX.Y.Z`.
2. Read the notes as a player would. To fix one, edit `.changes/vX.Y.Z.md` and run `changie merge` again.
   Wait for `ci-ok`, and keep the branch up to date with `main`. If `main` gains new notes meanwhile,
   start again from step 1.
3. The owner merges the pull request.
4. The owner tags the merge commit `vX.Y.Z` and pushes the tag.
5. `release.yml` waits for `main`'s CI. Then it builds and checks every file, signs the Linux packages and
   `checksums.txt`, and attests every file, the installers included. If all of that passes, it publishes
   the GitHub Release with the notes, starts its discussion and publishes the docs site again. Then it
   packs and tests the snaps.
6. Do the checks in `docs/PLAN.md` §14, such as installing from the release page by hand.

Only the merge, the tag and any store upload are the owner's. The details follow.

1. **The release pull request**, titled `vX.Y.Z` (a milestone's version is in `docs/PLAN.md` §15), batches
   the notes and rebuilds the changelog and the manual page:

   ```sh
   go tool -modfile=tools/release/go.mod changie batch vX.Y.Z
   go tool -modfile=tools/release/go.mod changie merge
   go run ./internal/tools/manpage
   ```

   The notes move from `.changes/unreleased/` into `.changes/vX.Y.Z.md`. `CHANGELOG.md` gains the
   version, and the manual page's header names it (CI fails until it does). Read the notes once more as
   a player would. To change one, edit `.changes/vX.Y.Z.md` and run `changie merge` again. Every CI
   run's `build` job shows, in its summary, the notes the next release would get.
2. **The owner merges it**, then tags its merge commit by the commit's hash, and pushes the tag:
   `git fetch origin && git tag vX.Y.Z <merge commit> && git push origin vX.Y.Z`. Tag that commit even
   if other pull requests merged after it. Their notes are not in this release's notes, so their changes
   wait for the next one.
3. **`release.yml` does the rest.** It waits for `main`'s CI. It builds and checks every file (the `.deb`
   and `.rpm` included). It builds the Windows installer and the macOS disk image from those files on
   their own runners, and tests them. It adds both to the release files and `checksums.txt` (`collect`).
   It signs the Linux packages and `checksums.txt` (its `sign` job, in the `release` environment). Then it
   publishes the GitHub Release, with every file attested and `.changes/vX.Y.Z.md` as its notes
   (`internal/tools/relnotes` adds a footer). It starts a discussion of the release in the Discussions
   category Announcements. The packages' changelogs carry the same notes. Then it publishes the docs
   site again (`docs.yml`), so that its download commands name the new release. It packs and tests the
   snaps. They go to the Snap Store only when the repository variable `SNAP_PUBLISH` is `true` and the
   owner approves the `snap-store` deployment. The run keeps the MSIX packages (artifact `msix`) and the
   snaps (`snaps`) for 90 days. They are never release files. To submit WOPR to the Microsoft Store,
   upload the `msix` artifact's `.msixbundle` in Partner Center. Do this once the Store identity is set,
   and from v1.0.0: the Store refuses a version whose first number is 0.

`relnotes` refuses a tag that does not name the batched version, and nothing is published. Remove that
tag and push the right one. `relnotes` warns when the tagged commit has notes the release does not list.
The `.deb` and `.rpm` changelogs date a release by its tagged commit, and `CHANGELOG.md` dates it by the
day it was batched. If the pull request merges on a later day (UTC), they differ, and `relnotes` warns. To
keep one date, set the date in `.changes/vX.Y.Z.md`'s header to the merge day before merging. Then run
`changie merge` and the manpage tool again.

If the pull request did not batch the notes, the release still gets them. The workflow batches
`.changes/unreleased/` itself, dated by the tagged commit, and warns. `main` then lags behind the release.
CI fails on every later commit (the tagged one still passes) until one pull request runs
`go run ./internal/tools/relnotes -catch-up`. That command writes the missing `.changes/vX.Y.Z.md` from
the tag, removes those notes from `.changes/unreleased/`, and merges `CHANGELOG.md`. Run
`go run ./internal/tools/manpage` too, and commit the result. That release's manual page names its own
version, as every build's does, but keeps the previous release's date. Only `vX.Y.Z` tags are releases:
`release.yml` publishes no other, and `relnotes` ignores the rest, such as `v0.3.0-rc.1`.

## Docs site

`site/` is the documentation site: Hugo with the Hextra theme. `docs.yml` publishes it to
<https://ghostofgoes.github.io/WOPR/> on every push to `main`, so a change to the site goes live when its
pull request merges. Its download commands and package instructions name the latest published release,
which `docs.yml` finds. (A release pull request puts its version in `CHANGELOG.md` before the release
exists.) `release.yml` starts `docs.yml` again after it publishes a release. To publish again by hand
(after changing the Pages settings, say), run `docs.yml` on `main`. CI's `docs` job builds the site on
every push with `--panicOnWarning`, so a deprecated setting, a broken internal link or a missing
screenshot fails the build.

- Pages are Markdown in `site/content/`, written for players in plain, direct prose. Usage has two sub-pages,
  `usage/accessibility.md` and `usage/troubleshooting.md`. A page that moves keeps its old address with an
  `aliases` entry in its front matter.
- The install instructions are written once, for both Quickstart and Installation: one Markdown file per tab
  in `site/assets/install/`. The `install-tabs` shortcode puts them in synced tabs. With no argument, it
  draws the download tabs (Windows, macOS, Linux, from `windows-installer.md`, `macos-app.md` and
  `linux-packages.md`). Each tab holds a download button, then each click and warning in order, with no
  command line. With `"command-line"`, it draws the Installation page's Command line install methods
  (Windows, macOS, Linux (any), Linux (apt), Linux (RPM), Go). Each is a line to paste into a terminal,
  which a person who has never used one can follow. Each line uses only the tools its system installs by
  default (the Go tab needs Go). The macOS and Linux (any) lines put the program in `~/.local/bin` without
  `sudo`, over HTTPS only. They add that folder to the shell's `PATH` when it is missing. Only the package
  lines need root. In these files, `@VERSION@` becomes the latest release's version (shortcodes do not run
  there). `@DOWNLOAD-INSTALLER@`, `@DOWNLOAD-APP@`, `@DOWNLOAD-DEB@` or `@DOWNLOAD-RPM@`, alone in a
  paragraph, becomes two download buttons, side by side, drawn by the shortcode (styled in
  `site/assets/css/custom.css`). The buttons are "Intel/AMD" and "ARM" on Windows and Linux, and "Apple
  Silicon" and "Intel" on macOS. Each is a plain link to its release file, with no file name under it. The
  Windows installer and the Mac app are one file for both kinds of computer, so both of their buttons link
  to it. Until the latest release has the installers (the `has-installers` partial: every release after
  v0.4.0), the Windows and macOS download tabs show `download-later.md` instead. The packages' menu entry
  first ships in the same release (v0.4.0's packages have none). So `@APP-MENU@` in `linux-packages.md`,
  `linux-apt.md` and both RPM files becomes the sentence in `app-menu.md` only when `has-installers` is
  true, and nothing before. The `has-signatures` partial says whether the latest release is signed:
  `docs.yml` passes whether the release has `checksums.txt.asc`, and other builds look for
  `packaging/wopr-signing-key.fingerprint`. Once the latest release is signed, `@SIGNATURES@` in
  `linux-packages.md` becomes `packages-signed.md` (before, nothing). The Linux (RPM) command line is then
  `linux-rpm-signed.md` in place of `linux-rpm.md`. In it, `@KEYURL@` becomes the signing key's address on
  the site, and `@FINGERPRINT@` becomes its fingerprint. Put text about the signatures in pages inside
  `{{% if-signed %}}`, which works as `if-installers` does. The fingerprint comes from the
  `signing-fingerprint` shortcode. Every build reads each file of a set, both RPM files and
  `packages-signed.md` included, whichever it shows. It fails if one lacks its placeholder, or if a tab
  still has any `@NAME@` placeholder. On those two pages, `site/assets/js/install-platform.js` (loaded by
  `layouts/_partials/custom/head-end.html`) picks the reader's system in each set they have not picked
  before, and saves nothing. It hides the download button that does not fit the reader's processor when
  the browser says it for certain (client hints, or Firefox's user agent on Linux). Otherwise both buttons
  stay. It picks a tab with Hextra's own `updateGroup`, which `head-end.html` copies from Hextra's `tabs.js`
  at build time, so a Hextra that changes it fails the build. How to check an attestation belongs in the
  Installation page's "Verifying binaries" section, which keeps the anchor `#verifying-binaries-attestation`
  that releases link. Development builds are its last section. The uninstall steps also remove the debug
  log.
- `site/content/games/_content.gotmpl` builds each game's page from `site/data/games/<slug>.json`, and it
  documents the schema. The manual page reads the same files. Change a game's text there.
- Nothing is copied into the site. `site/hugo.yaml` mounts `CHANGELOG.md`, `CODE_OF_CONDUCT.md`,
  `NOTICE.md` and this file's Commands and Pull requests sections, and the `repo-file` shortcode renders
  them. It also mounts the README's screenshots and the manual page.
- Add a screenshot with the `screenshot` shortcode and a caption that says what the screen shows. The
  caption is also its alt text, and it shows in the lightbox. PNGs are 960×564, made smaller with
  `optipng -o2`. One that shows the big board's map gets the map's credit line. Wrap several in the
  `screenshots` shortcode to open them as one set, as each game's page does.
- The lightbox is Hextra's, PhotoSwipe, and the search is FlexSearch. Their files and licences are in
  `site/assets/vendor/`, exactly as their npm packages publish them. Never edit them: `.gitattributes`
  and `prek.toml` skip them. `site/hugo.yaml` names them. The credits page shows the licences, so the
  build needs no network beyond the Go module proxy.
- Download commands in pages use the `version` shortcode, and those in `site/assets/install/` use
  `@VERSION@`. Both are the latest published release in `docs.yml`'s builds, and the latest in
  `CHANGELOG.md` in others. Put text about the Windows installer, the Mac app, or WOPR in the Linux app
  menu (the packages' menu entry) inside `{{% if-installers %}}`. It shows the text only once the latest
  release has them (every release after v0.4.0, the `has-installers` partial), so no build of the site
  names a file that does not exist yet. Put text for the releases before them inside
  `{{% if-installers "not" %}}`. Its content is Markdown only: a shortcode inside it that writes HTML, such
  as `tabs`, is dropped. Inside a Hextra `tab`, write it as `{{< if-installers >}}`: the tab renders the
  Markdown, and the `%` form fails the build there. Text about the `.deb`, the `.rpm` or the archives'
  `wopr.6` needs no gate: every release since v0.4.0 has them. So `if-packages`, which still wraps some of
  it, always shows it. Nothing names the snap until it is published.
- Tests in `internal/cli` check that the Usage page lists every option and environment variable, and the
  Movie scenes page every scene.

## Pull requests

- `main` is PR-only. The required check is `ci-ok`. Merges are squash merges.
- A change players can notice adds a change note ([Change notes](#change-notes)).
- CI (`ci.yml`) runs on every branch push, so a branch is checked before its PR. That branch's push run
  checks a PR from a branch here (its `pull_request` run skips every job). A PR from a fork runs in full.
  The push run tests the branch as it is, not merged with `main`, so the ruleset requires branches to be
  up to date. Updating one is a push, which tests the result.
- Run `prek run --all-files` and `go test ./...` before pushing.
- **Code review.** A maintainer (`GOVERNANCE.md`) reviews every pull request in GitHub's review tools
  before it merges. The reviewer reads the whole diff and checks these things:
  - The change does what its description says, and the project needs it (`ROADMAP.md`, `docs/PLAN.md`).
  - New behaviour has tests, and a bug fix has a regression test.
  - The change keeps the rules in this file, and film text stays as the film shows it.
  - The change moves no trust boundary, or it updates `SECURITY.md` and `docs/ASSURANCE.md`.
  - Players get a change note when they will notice the change.
  - Documentation that the change makes wrong is updated.

  A pull request is acceptable when `ci-ok` passes on its latest commit, the branch is up to date with
  `main`, and every review comment is answered or resolved. The maintainer reviews pull requests that
  an AI agent wrote, and the backup maintainer reviews the maintainer's own when possible.
- OpenSSF Best Practices compliance is a project goal, so check each change against the
  [criteria](https://www.bestpractices.dev/en/criteria/0) before you open a PR.
- Never re-tag a release. Fix a bad release with the next patch version and a `retract` in `go.mod`.

## Secret scanning

The `secrets` job runs gitleaks over the checked-out history (the pushed branch, or a fork's pull request
merged with `main`), with `--no-color` so the job can detect gitleaks's `ERR` lines. It fails on a finding, on
any error, and on an empty scan. Handle a finding as `SECURITY.md` describes: rotate the secret, then add
its fingerprint to `.gitleaksignore` by pull request.

To prove the job (the M0b canary):

1. Push a throwaway branch that contains a fake secret that only gitleaks recognises (not a
   GitHub-supported token pattern, which push protection blocks).
2. Check that its CI run's `secrets` log says `leaks found` rather than an error.
3. Remove the branch.

## Takedown runbook

If a rights holder asks you to remove material:

1. Find every copy:
   - The provenance tags name each line's source (`internal/wopr/lines.go`, `internal/assets/`,
     `internal/movie/scenes/`, and the games' film text).
   - The README's screenshots in `docs/screenshots/` and the docs site's in `site/static/img/` (game
     screenshots in `site/static/img/games/`) show some of it.
   - The docs site's pages (`site/content/`, `site/data/games/`) quote some, which the manual page
     (`docs/man/wopr.6`) repeats.
   - The packages' AppStream metadata (`packaging/linux/io.github.ghostofgoes.wopr.metainfo.xml.in`)
     links some of the site's screenshots.
   - The `.deb`'s copyright file (`packaging/debian/copyright`) names the files. Regenerate it
     afterwards.
2. Remove or replace it in one pull request. Release a patch version. Merging the pull request
   republishes the docs site (`docs.yml`). Once WOPR is in the Snap Store or the Microsoft Store, its
   release there needs the patch too, or the listing needs to be unpublished until then.
3. Add a `retract` directive to `go.mod` for the affected versions. Remove the affected GitHub releases
   (an immutable release can be removed, not edited, and its tag cannot be reused).
4. Reply to the requester and say what you did. Copies remain in git history and in the Go module mirror,
   which has no documented removal process for this case.

## Repository settings (owner, once)

These live in GitHub settings, not in files. Check them at each milestone:

- **Branch ruleset on `main`:** require a pull request. Require the `ci-ok` check from GitHub Actions, with
  "Require branches to be up to date before merging". Block force pushes and deletion. Allow squash merges
  only (merge commits and rebase merging off). `gh api 'repos/{owner}/{repo}/rules/branches/main'` should
  list `pull_request` and `required_status_checks` rules.
- **Collaborators:** the backup maintainer in `GOVERNANCE.md` has write access. Keep the emergency access
  that its Continuity section describes current.
- **Tag ruleset on `v*`:** restrict creation, update and deletion to the owner. Enable immutable releases.
- **Security:** enable secret scanning with push protection, and private vulnerability reporting.
- **Release signing key** (`docs/PLAN.md` §8, "Signatures"). Create an environment named `release`
  (Settings → Environments), with deployment tags limited to `v*`. It holds the secret `WOPR_SIGNING_KEY`
  (the armored secret key) and, if the key has one, `WOPR_SIGNING_PASSPHRASE`. Only `release.yml`'s `sign`
  job uses the environment. The public key and its fingerprint are committed: `release.yml` stops at
  `sign` without them. To make the key, on a trusted computer:

  ```sh
  export GNUPGHOME="$(mktemp -d)"
  gpg --batch --pinentry-mode loopback --passphrase '' \
    --quick-gen-key 'Christopher Goes (wopr release signing) <ghostofgoes@gmail.com>' rsa4096 sign 5y
  gpg --armor --export > packaging/wopr-signing-key.asc
  gpg --with-colons --list-keys | awk -F: '$1 == "fpr" { print $10; exit }' > packaging/wopr-signing-key.fingerprint
  gpg --armor --export-secret-keys   # paste into the WOPR_SIGNING_KEY secret
  ```

  Keep an offline copy of the secret key, then remove `$GNUPGHOME`. Before the key expires, extend it
  (`gpg --quick-set-expire`) and commit the new `wopr-signing-key.asc`, which keeps the fingerprint. A
  new key needs both files changed in one pull request. The docs site then shows the new fingerprint.
- **Issues and Discussions:** issues on, with the bug report and feature request forms in
  `.github/ISSUE_TEMPLATE/` (blank issues off). **Discussions on** (Settings → General → Features), since
  the forms, the docs site and the README send questions there. Discussions keeps its default
  **Announcements** category, where `release.yml` starts each release's discussion.
- **Actions:**
  - Allow only `actions/*`, `j178/prek-action`, `ossf/scorecard-action` and
    `github/codeql-action/upload-sarif` (both for `scorecard.yml`). Require full-length commit-SHA pinning.
  - Set the default `GITHUB_TOKEN` to read-only. Workflows may not create or approve pull requests.
  - Require approval for workflow runs from all external contributors.
- **Dependabot:** off (owner decision). The weekly `scheduled.yml` report covers updates and
  vulnerabilities.
- **Installers:** the Windows installer and the Mac app need no secret, variable or environment. The
  Linux packages need only the release signing key above. Each store needs a secret, a variable or an
  environment, and only when the owner enables that store:
  - **Snap Store**, when the name is registered. With an Ubuntu One account (two-factor on), run
    `snapcraft register wopr`. A person at the Store reviews new names, which can take days or weeks.
    Create the environment `snap-store` with the owner as required reviewer, and deployments from tags
    `v*` only (no branches). Put the secret `SNAPCRAFT_STORE_CREDENTIALS` in it alone. The secret is the
    file that
    `snapcraft export-login --snaps=wopr --channels=stable --acls=package_access,package_push,package_update,package_release --expires=<date under a year away> creds.txt`
    writes (`gh secret set SNAPCRAFT_STORE_CREDENTIALS --env snap-store < creds.txt`, then remove the
    file). Then set the repository variable `SNAP_PUBLISH` to `true`. Any other value keeps `snap.yml`'s
    publish job skipped. The next `v*` tag uploads both snaps to the stable channel once the owner
    approves the deployment. Export a new login before the old one expires.
  - **Microsoft Store**, when the name is reserved as a game in Partner Center: set the repository
    variables `MSIX_IDENTITY_NAME`, `MSIX_PUBLISHER` (`CN=…`) and `MSIX_PUBLISHER_DISPLAY_NAME`, copied
    from Product identity. They are not secret. Until they are set, the MSIX packages carry placeholders
    and the Store would refuse them.
- **Pages:** Settings → Pages → Build and deployment → Source: **GitHub Actions**. `docs.yml` then
  deploys the docs site to <https://ghostofgoes.github.io/WOPR/> through the `github-pages` environment,
  which GitHub creates and limits to deployments from `main`.

## Milestone checklist

At every milestone boundary:

1. Update dependencies to versions at least 14 days old ([Dependency cooldown](#rules-that-tests-enforce)).
   Security fixes are the exception, at any time.
   `go list -m -u -json all` shows each update and its `Time`. Update direct dependencies one by one with
   `go get <module>@<version>`. Never use `go get -u ./...`, which also lifts indirect modules to their
   newest untagged commits. Then run `go mod tidy`. Update the tool modules the same way, with
   `go get -tool <tool>@<version>` in `tools/`, `tools/lint/`, `tools/release/` and `tools/docs/`. Update
   the docs theme with `go -C site get github.com/imfing/hextra@<version>`. Run
   `go run ./internal/tools/cooldown` before pushing. Re-vendor PhotoSwipe (its newest 5.x, the major
   version Hextra's script is written for) and FlexSearch (the version the new Hextra defaults to, in its
   `layouts/_partials/scripts/search.html`):
   - Download each npm tarball from `https://registry.npmjs.org/<name>/-/<name>-<version>.tgz`.
   - Check its SHA-512 against the registry's `dist.integrity`.
   - Copy the same files and `LICENSE` into `site/assets/vendor/<name>/`.
   - Update the versions and hashes in `site/hugo.yaml`.

   Build the docs site: a new Hugo can deprecate a setting, and `--panicOnWarning` makes that an error.
2. Run `prek update`. Keep golangci-lint and gitleaks in step between `prek.toml` and their tool modules.
3. Bump action SHAs from their release tags, each at least 14 days old. Bump the Fedora image digest in
   `smoke.yml` to the newest Fedora release's (`registry.fedoraproject.org/fedora:<N>`).
4. Bump the installers' tools:
   - **Inno Setup**, if <https://jrsoftware.org/isdl.php> lists a newer one: in
     `packaging/windows/build-installer.ps1`, change `$InnoUrl`, `$InnoSha256` (the `file-hash` in the
     release's `.issig` file) and `$InnoSigner` together, with the version in `-InnoDir`'s default and the
     download's name. Read the release notes for changes to scripts.
   - **The Windows SDK**: check that the `windows-2025` and `windows-11-arm` images still have one
     (`WindowsSdk.psm1` takes the newest). Set `MaxVersionTested` in
     `packaging/windows/AppxManifest.xml` to the newest Windows build.
   - **snapcraft**: if `snap info snapcraft` shows a newer major track than `snap.yml`'s `9.x/stable`, move
     to it. Check that core24 is still a supported base.
   - If `LICENSE`'s year changed, change the copyright in `packaging/windows/wopr.iss`,
     `packaging/windows/winres.json`, `internal/tools/macapp` and the docs site's footer
     (`site/i18n/en.yaml`) to match.
5. Regenerate the notices and the icons (`go run ./internal/tools/icons`, which rewrites only the files
   that changed).
6. Check the hosted runner labels in `.github/workflows` against GitHub's announcements.
7. Re-enable `scheduled.yml` if GitHub disabled it after 60 quiet days.
8. Once the snap is published, export a new Snap Store login before the old one expires (Repository
   settings).
9. Update `docs/PLAN.md` and this file.
