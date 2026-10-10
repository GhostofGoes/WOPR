# AGENTS.md

Instructions for anyone, human or AI, changing this repository. They are the single source for commands
and conventions. The design and milestones are in [docs/PLAN.md](docs/PLAN.md); this file wins where the
two disagree on day-to-day practice. Update both in the same PR when a convention changes.

## What this is

`wopr` is a Go terminal-UI recreation of the WOPR computer from *WarGames* (1983). It is a single static
binary: Bubble Tea v2 renders it, Lip Gloss v2 styles it, and there is no network code. It is built in the
milestones of `docs/PLAN.md` §15.

## Commands

All commands run from the repository root and work on Linux, macOS and Windows, unless a row names a system.
There is no Makefile.

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
| Dependency cooldown (no module newer than 14 days) | `go run ./internal/tools/cooldown` |
| Secret scan (full history) | `go tool -modfile=tools/go.mod gitleaks git --redact .` |
| Third-party notices and the `.deb`'s copyright file | `go run ./internal/tools/notices` (writes `THIRD_PARTY_NOTICES.txt` and `packaging/debian/copyright`); CI runs it with `-check` |
| Manual page (`docs/man/wopr.6`) | `go run ./internal/tools/manpage`; CI runs it with `-check`. Lint it with `mandoc -T lint -W all docs/man/wopr.6` |
| Icons (`packaging/icons/`, the docs site's favicons) | `go run ./internal/tools/icons` (draws every icon file from `packaging/icons/src/`; `packaging/icons/README.md` says how to change the icon); CI runs it with `-check` |
| Add a change note | `go tool -modfile=tools/release/go.mod changie new` (see [Change notes](#change-notes)) |
| Check the change notes and `CHANGELOG.md` | `go run ./internal/tools/relnotes -check` (a prek hook) |
| A release's notes | `go run ./internal/tools/relnotes -version X.Y.Z -out build/notes` (or `-snapshot`) |
| Release build (local dry run) | `go run ./internal/tools/relnotes -snapshot -out build/notes`, then, with `WOPR_NOTES_DIR=build/notes` in the environment, `go tool -modfile=tools/release/go.mod goreleaser release --snapshot --clean` |
| Size gate | `go run ./internal/tools/sizegate -expect 6 -packages 4`; `-files <file>...` gates the installers (the `collect` jobs) |
| Stage binaries and e2e tests | `go run ./internal/tools/stage` (`-archives -assets dist/release` also checks the archives and the Linux packages, and collects every release file into `dist/release`; `-merge -assets dist/release <file>...` adds the installers to them and to `checksums.txt`, sorted as GoReleaser sorts it) |
| Lint the Linux packages (after a release build; Linux, tools not pinned) | `lintian --pedantic dist/*.deb` and, with Fedora's rpmlint configuration, `rpmlint -r packaging/rpmlintrc dist/*.rpm`; see `docs/PLAN.md` §8 for what they report. List an `.rpm`'s files with `rpm -qlvp`, or unpack it with `bsdtar -xf X.rpm -C dir`; never pipe `rpm2cpio` into a plain `cpio -idm`, which writes into `/` (the payload's paths are absolute) |
| Validate the menu entries and the AppStream metadata (after a release build; Linux, tools not pinned) | `desktop-file-validate build/pkg/deb/*.desktop build/pkg/rpm/*.desktop` (prints nothing when they pass), `appstreamcli validate --no-net --pedantic build/pkg/*.metainfo.xml` and `appstream-util validate-relax --nonet build/pkg/*.metainfo.xml`; the `desktop-files` hook runs them on what `pkgdocs` writes (its `TestValidators`, which skips one that is not installed unless `WOPR_DESKTOP_VALIDATORS=1`), and the smoke jobs on the installed `.deb` |
| Windows installer and MSIX packages (Windows, PowerShell 7, after a release build and `stage -archives -assets dist/release`) | `pwsh -File packaging/windows/check-release-files.ps1 -Version <V> -BinDir dist/release`, then `pwsh -File packaging/windows/build-installer.ps1 -Version <V> -BinDir dist/release -OutputDir build/installer` (downloads the pinned Inno Setup) and `pwsh -File packaging/windows/build-msix.ps1 -Version <V> -BinDir dist/release -OutputDir build/msix` (needs the Windows SDK) |
| Test them (Windows; only on a machine without WOPR, such as a CI runner) | `pwsh -File packaging/windows/test-installer.ps1 -Installer build/installer/wopr_<V>_windows_setup.exe -Version <V>` (changes the user's `PATH` while it runs) and, as an administrator, `pwsh -File packaging/windows/test-msix.ps1 -PackageDir build/msix -Version <V>` (trusts a throwaway certificate while it runs) |
| `wopr.exe`'s icon, manifest and version details by hand | `go tool -modfile=tools/release/go.mod go-winres make --in packaging/windows/winres.json --out cmd/wopr/rsrc --arch amd64,arm64` (GoReleaser's before hook runs it; the `.syso` files it writes are gitignored) |
| macOS app bundle (any OS) | `go run ./internal/tools/macapp -binary <darwin program> -version <V> -out build/WOPR.app` (`-notices dist/release` takes the notices from the release files) |
| macOS disk image (macOS, after a release build and `stage -archives -assets dist/release`) | `packaging/macos/build-dmg.sh dist/release <V> build/dmg`, then `packaging/macos/test-dmg.sh build/dmg/wopr_<V>_macos.dmg <V> [stage/darwin_<arch>/e2e.test]`. `packaging/macos/test-launch.sh build/dmg/wopr_<V>_macos.dmg` opens the app as Finder does and stops Terminal afterwards: run it on a CI runner, or a Mac where that does not matter |
| Snaps (Linux with snapd, after a release build and `stage -archives -assets dist/release`) | `go run ./internal/tools/snapdir -dist dist/release -version <V> -arch amd64 -out build/snap/amd64`, then `snap pack build/snap/amd64 build/snaps` (without snapd, `mksquashfs build/snap/amd64 build/snaps/wopr_<V>_amd64.snap -noappend -comp xz -no-fragments -all-root -no-xattrs` packs it as snapd does). The Store's checks: `sudo snap install review-tools`, then `review-tools.snap-review <snap>` |
| Test a snap (Ubuntu, sudo; installs and removes it) | `packaging/snap/test-snap.sh build/snaps/wopr_<V>_amd64.snap <V> [stage/linux_amd64/e2e.test]` |
| Docs site: preview (localhost:1313/WOPR/) | `go tool -modfile=tools/docs/go.mod hugo server --source site` |
| Docs site: build as CI does | `go tool -modfile=tools/docs/go.mod hugo --source site --panicOnWarning --printPathWarnings --minify` (into `site/public/`) |

`<V>` is the build's version as GoReleaser writes it in `dist/metadata.json`: `1.2.3`, or
`1.2.4-snapshot.abc1234` for a snapshot. The installers' scripts run only on their own systems; CI runs
them (`windows.yml`, `macos.yml`, `snap.yml`).

Go 1.27.1 is pinned in `go.mod` (`toolchain go1.27.1`); `GOTOOLCHAIN=auto` fetches it, and a newer local Go is
fine (CI checks the exact version). Tools are pinned in four modules, never `go run …@latest`:
`tools/go.mod` (gitleaks, govulncheck), `tools/lint/go.mod` (golangci-lint), `tools/release/go.mod`
(GoReleaser, changie, go-winres) and `tools/docs/go.mod` (Hugo, the standard edition). They are separate
because their dependency graphs conflict. The docs site's theme, Hextra, is a Hugo module pinned in
`site/go.mod`. No tool module's `go` line, nor `site/go.mod`'s, may be newer than the root `toolchain` line (a
test checks): bump the toolchain first, then the tool. Inno Setup is not a Go tool: `build-installer.ps1`
pins it by URL, SHA-256 and Authenticode signer. `snap.yml` installs snapcraft from its `9.x/stable`
channel, and review-tools from `latest/stable`.

## Rules that tests enforce

- **Import DAG.** `internal/archtest` checks every import, test imports included, against the table in
  `internal/archtest/archtest_test.go`, which mirrors `docs/PLAN.md` §4.1. A new package needs a row in
  both. Only `internal/ui` imports Bubble Tea. Only `internal/ui` and `internal/theme` import
  `charm.land/*`. `games` and `wopr` never touch the terminal. archtest lists the imports for linux, darwin
  and windows, since some files build for one only. `os/exec` is fenced to `cmd/wopr` (its macOS build
  reopens the app in Terminal), `internal/tools/...`, `internal/archtest` and `internal/e2e`.
- **One clock.** Only `internal/ui/clock.go` may call `tea.Tick`, `tea.Every` or `time.Sleep`-style
  timers (tests may use `time`'s); `tea.Sequence` is banned everywhere (forbidigo).
- **Seeded randomness.** Use `proto.NewRand(seed, streamID)` with a stream from the domains in
  `internal/proto/rand.go`. The top-level `math/rand/v2` functions are banned.
- **Lint rules must fire.** `internal/archtest/testdata/lintfixture` breaks each custom rule on purpose,
  and the self-test requires every rule to report it. Tools pinned both in `prek.toml` and in a tool module
  (golangci-lint, gitleaks) must have the same version.
- **Dependency cooldown.** No module in any build list (wopr's, the four tool modules' and `site/go.mod`'s)
  may be less than 14 days old at the commit being checked, by the time the module proxy gives its version:
  `go run ./internal/tools/cooldown` (CI's `lint` job, and a prek hook on `go.mod` and `go.sum`). A fix that
  cannot wait goes in `tools/cooldown-exceptions.txt` with its reason, until it is old enough.
- **Notices.** `THIRD_PARTY_NOTICES.txt` and `packaging/debian/copyright` (the `.deb`'s machine-readable
  copyright file) must match `go run ./internal/tools/notices`. The copyright file lists the files that
  quote the film from their provenance tags, so regenerate it when film text moves. It files each linked
  module under Expat or Go's BSD-3-clause only when the module's licence has the same words, so a module
  under any other licence stops the tool until it is added there.
- **Linux packages.** `.goreleaser.yaml`'s `nfpms` build a `.deb` (Debian Policy: the program in
  `/usr/games`, the manual page in section 6, the documents in `/usr/share/doc/wopr`) and an `.rpm`
  (Fedora: `/usr/bin`, `%license`, `%doc`) for each Linux build (`docs/PLAN.md` §8). `stage -archives`
  checks what each installs. `relnotes`'s tests check that the packages' maintainer and release match the
  changelogs it writes, so change both together. `pkgdocs` writes the packages' documents and their
  `changelog.yml` into `build/pkg`; its tests hold `packaging/description.txt`, both packages'
  description, to both formats' rules. Both packages also put WOPR in the desktop's menu, in files named
  by the app ID `io.github.ghostofgoes.wopr`: a menu entry that opens a terminal running `wopr`, the icon
  in every hicolor size, and AppStream metadata. `pkgdocs` writes the
  menu entry for each package and the metadata from their templates in `packaging/linux/`; its tests
  hold them to the Desktop Entry Specification and AppStream's rules, and check that the screenshots and
  pages they link exist in `site/`. `stage -archives` checks that the packages hold `build/pkg`'s files
  (by digest), that each menu entry starts that package's own program, and every icon size. The sizes
  are also listed in the `.rpm`'s directories in `.goreleaser.yaml` and in `smoke.yml`: change all three
  together. A release signs the packages (`packaging/sign-packages.sh`: a signature inside each `.rpm`, a
  `.deb.asc` beside each `.deb`, and `checksums.txt.asc`) with the key in `packaging/wopr-signing-key.asc`;
  CI signs with a throwaway key. The smoke jobs install, run and remove both packages, with signature
  checks on in CI, and validate the installed menu entry and metadata; `pkgdocs`'s `TestValidators` runs
  the same validators on its own files wherever they are installed, and CI's lint job installs them, so
  none skips there.
- **Icons.** Everything in `packaging/icons/` except `src/`, and the docs site's `favicon.svg` (also its
  navbar logo), `favicon.ico` and `apple-touch-icon.png`, is drawn by `internal/tools/icons` from the SVG
  sources in `packaging/icons/src/`. Its tests and `-check` fail when a file is stale; pictures are
  compared pixel by pixel within 2/255, because floating point differs between CPUs, and the `.ico` and
  `.icns` around them must be byte for byte what the tool writes from those pixels. The tool refuses a
  source it cannot draw or would never use, and a design that makes any file larger than 512 KB;
  `packaging/icons/README.md` lists the sources and the SVG subset. The design, front-panel lamps, is the
  owner's choice (`docs/PLAN.md` §8).
- **Windows installer and MSIX.** `windows.yml` checks the release files against `checksums.txt`, builds
  `wopr_<V>_windows_setup.exe` with Inno Setup (`packaging/windows/wopr.iss`: per user, no administrator
  prompt, both architectures in one file) and the MSIX packages for the Microsoft Store, then installs,
  runs and removes both on x64 and Arm64 Windows (`test-installer.ps1`, `test-msix.ps1`). Never change
  the installer's `AppId`: upgrades and uninstalling find WOPR by it. go-winres puts the icon, a
  manifest and version details into `wopr.exe` (`packaging/windows/winres.json`), and the `.syso` files
  it writes come out the same on every run, so `repro` still covers the program.
- **macOS app.** `macos.yml` checks the darwin programs against `checksums.txt`, joins them with `lipo`,
  writes `WOPR.app` with `internal/tools/macapp` (whose tests check the bundle on any OS), signs it ad hoc
  and makes `wopr_<V>_macos.dmg` (`packaging/macos/build-dmg.sh`). On Apple silicon and on Intel it then
  checks the image and runs the e2e tests against the app's program (`test-dmg.sh`); on Apple silicon it
  also opens the app as Finder does (`test-launch.sh`: the Intel runner's Terminal runs nothing it is
  given). Started from Finder, with no terminal, `wopr` reopens itself in
  Terminal (`cmd/wopr/launch*.go`, with table tests); started from a terminal, it runs there.
- **Snap.** `snap.yml` prepares each architecture's snap with `internal/tools/snapdir`, which checks the
  program and notices against `checksums.txt`, the metadata against snapd's limits, and turns the Linux
  menu entry into the snap's; `snap pack` packs it. The Store's `review-tools` must pass it, and
  `packaging/snap/test-snap.sh` installs, runs and removes it on both architectures. Publishing is off
  until the owner turns it on (Repository settings).
- **Game pages and the manual page.** Every game in the catalog has `site/data/games/<slug>.json` (the slug
  `wopr --games` shows): summary, how to play, controls, at least three tips, and screenshots in
  `site/static/img/games/`. The docs site and the manual page are built from these files, so nothing else
  repeats them. `internal/tools/manpage`'s tests check each file against the catalog, and check that
  `docs/man/wopr.6` is what the tool generates. The page takes its version and date from the newest
  `.changes/vX.Y.Z.md`, so regenerate it after batching a release. The archives and packages carry it
  with the version being built in its header instead (`pkgdocs`).
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
  position as a FEN string. An Update or View that runs for 10 s (`internal/ui/watchdog.go`), or is still
  running a second after SIGINT or SIGTERM, is taken for a hang: wopr restores the terminal and exits 1.
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
drawn and exempt from the capitals rule), or `original`. Do not copy text or art from other projects
without a licence and a `NOTICE.md` entry that credits the source. Lines adapted from the brother's prompt (the
`prompt` tag) are not committed at all until he grants written permission and a licence (`docs/PLAN.md` §2.1).

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
  `games/testkit`, `games/gamestest`), or a file that decides what the downloads hold: `.goreleaser.yaml`,
  `packaging/` or the manual page in `docs/man/`. If players will notice nothing, add `Changelog: none`
  as a trailer to a commit message on the branch: a line in the message's last paragraph, next to any
  `Co-Authored-By:` lines. A line higher up in the message is not a trailer.
- **Never edit `CHANGELOG.md` by hand.** It is `changie merge`'s output, and the `change-notes` hook
  (`relnotes -check`) fails when it differs, when a note does not parse or is too long, or when a release
  tag has no `.changes/vX.Y.Z.md`.

## Releasing

**Start to finish:** (1) On a new branch from `main` that holds nothing else, run the three commands
below and open a pull request titled `vX.Y.Z`. (2) Read the notes as a player would. To fix one, edit
`.changes/vX.Y.Z.md` and run `changie merge` again. Wait for `ci-ok`, and keep the branch up to date with
`main`; if `main` gains new notes meanwhile, start again from (1). (3) The owner merges the pull request,
then (4) tags the merge commit `vX.Y.Z` and pushes the tag. (5) `release.yml` waits for `main`'s CI, then
builds and checks every file, signs the Linux packages and `checksums.txt`, and attests every file, the
installers included. If all of that passes, it publishes the GitHub Release with the notes, starts its
discussion and publishes the docs site again; then it packs and tests the snaps. (6) Do the checks in
`docs/PLAN.md` §14, such as installing from the release page by hand. Only the merge, the tag and any store
upload are the owner's; the details follow.

1. **The release pull request**, titled `vX.Y.Z` (a milestone's version is in `docs/PLAN.md` §15), batches
   the notes and rebuilds the changelog and the manual page:

   ```sh
   go tool -modfile=tools/release/go.mod changie batch vX.Y.Z
   go tool -modfile=tools/release/go.mod changie merge
   go run ./internal/tools/manpage
   ```

   The notes move from `.changes/unreleased/` into `.changes/vX.Y.Z.md`, `CHANGELOG.md` gains the
   version, and the manual page's header names it (CI fails until it does). Read the notes once more as
   a player would; to change one, edit `.changes/vX.Y.Z.md` and run `changie merge` again. Every CI
   run's `build` job shows the notes the next release would get, in its summary.
2. **The owner merges it**, then tags its merge commit, by the commit's hash, and pushes the tag:
   `git fetch origin && git tag vX.Y.Z <merge commit> && git push origin vX.Y.Z`. Tag that commit even
   if other pull requests merged after it: their notes are not in this release's notes, so their changes
   wait for the next one.
3. **`release.yml` does the rest.** It waits for `main`'s CI, builds and checks every file (the `.deb` and
   `.rpm` included), builds the Windows installer and the macOS disk image from those files on their own
   runners and tests them, and adds both to the release files and `checksums.txt` (`collect`). It signs
   the Linux packages and `checksums.txt` (its `sign` job, in the `release` environment). Then it
   publishes the GitHub Release with every file attested and `.changes/vX.Y.Z.md` as its notes
   (`internal/tools/relnotes` adds a footer), and starts a discussion of it in the Discussions category
   Announcements. The packages' changelogs carry the same notes. Then it publishes the docs site again
   (`docs.yml`), so that its download commands name the new release, and packs and tests the snaps. They
   go to the Snap Store only when the repository variable `SNAP_PUBLISH` is `true` and the owner approves
   the `snap-store` deployment. The run keeps the MSIX packages (artifact `msix`) and the snaps (`snaps`)
   for 90 days; they are never release files. To submit WOPR to the Microsoft Store, upload the `msix`
   artifact's `.msixbundle` in Partner Center (once the Store identity is set, and from v1.0.0: the Store
   refuses a version whose first number is 0).

`relnotes` refuses a tag that does not name the batched version, and nothing is published: delete that
tag and push the right one. It warns when the tagged commit has notes the release does not list. The `.deb` and `.rpm`
changelogs date a release by its tagged commit, and `CHANGELOG.md` by the day it was batched. If the pull
request merges on a later day (UTC), they differ, and `relnotes` warns; to keep one date, set the date in
`.changes/vX.Y.Z.md`'s header to the merge day before merging, and run `changie merge` and the manpage tool
again.

If the pull request did not batch the notes, the release still gets them: the workflow batches
`.changes/unreleased/` itself, dated by the tagged commit, and warns. `main` then lags behind the release,
so CI fails on every later commit (the tagged one still passes) until one pull request runs
`go run ./internal/tools/relnotes -catch-up`. It writes the missing `.changes/vX.Y.Z.md` from the tag,
removes those notes from `.changes/unreleased/`, and merges `CHANGELOG.md`; run
`go run ./internal/tools/manpage` too, and commit the result. That release's manual page names its own
version, as every build's does, but keeps the previous release's date. Only `vX.Y.Z` tags are releases:
`release.yml` publishes no other, and `relnotes` ignores the rest, such as `v0.3.0-rc.1`.

## Docs site

`site/` is the documentation site: Hugo with the Hextra theme, published to
<https://ghostofgoes.github.io/WOPR/> by `docs.yml` on every push to `main`, so a change to the site goes
live when its pull request merges. Its download commands and package instructions name the latest
published release, which `docs.yml` looks up (a release pull request puts its version in `CHANGELOG.md`
before the release exists); `release.yml` starts `docs.yml` again after it publishes a release. To publish
again by hand (after changing the Pages settings, say), run `docs.yml` on `main`. CI's `docs` job builds the
site on every push with `--panicOnWarning`, so a deprecated setting, a broken internal link or a missing
screenshot fails the build.

- Pages are Markdown in `site/content/`, written for players in plain, direct prose. Usage has two sub-pages,
  `usage/accessibility.md` and `usage/troubleshooting.md`; a page that moves keeps its old address with an
  `aliases` entry in its front matter.
- The install instructions are written once, for both Quickstart and Installation: one Markdown file per tab
  in `site/assets/install/`, which the `install-tabs` shortcode puts in synced tabs. With no argument, it
  draws the download tabs (Windows, macOS, Linux, from `windows-installer.md`, `macos-app.md` and
  `linux-packages.md`): a download button, then each click and warning in order, with no command line. With
  `"command-line"`, it draws the Installation page's Command line install methods (Windows, macOS,
  Linux (any), Linux (apt), Linux (RPM), Go): each a line to paste into a terminal, which a person who has
  never used one can follow, using only the tools each system installs by default (the Go tab needs Go).
  The macOS and Linux (any) lines put the program in `~/.local/bin` without `sudo`, over HTTPS only, and
  add that folder to the shell's `PATH` when it is missing; only the package lines need root. In
  these files, `@VERSION@` becomes the latest release's version (shortcodes do not run there), and
  `@DOWNLOAD-INSTALLER@`, `@DOWNLOAD-APP@`, `@DOWNLOAD-DEB@` or `@DOWNLOAD-RPM@`, alone in a paragraph,
  becomes a download button: a link to that release file, drawn by the shortcode (styled in
  `site/assets/css/custom.css`), with the file's name under it and, for a package, a link to its Arm file.
  Until the latest release has the installers (the `has-installers` partial: every release after v0.4.0), the
  Windows and macOS download tabs show `download-later.md` instead. The packages' menu entry first ships in
  the same release (v0.4.0's packages have none), so `@APP-MENU@` in `linux-packages.md`, `linux-apt.md` and
  both RPM files becomes the sentence in `app-menu.md` only when `has-installers` is true, and nothing before.
  Once the latest release is signed (the `has-signatures` partial: `docs.yml` passes whether the release has
  `checksums.txt.asc`; other builds look for `packaging/wopr-signing-key.fingerprint`), `@SIGNATURES@` in
  `linux-packages.md` becomes `packages-signed.md` (before, nothing), and the Linux (RPM) command line is
  `linux-rpm-signed.md` in place of `linux-rpm.md`, where `@KEYURL@` becomes the signing key's address on
  the site and `@FINGERPRINT@` its fingerprint. Text about the signatures in pages goes inside
  `{{% if-signed %}}`, which works as `if-installers` does, and the fingerprint comes from the
  `signing-fingerprint` shortcode. Every build reads each file of a set, both RPM files and
  `packages-signed.md` included, whichever it shows, and fails if one lacks its placeholder or a tab is
  left with any `@NAME@` placeholder. On those two pages, `site/assets/js/install-platform.js` (loaded by
  `layouts/_partials/custom/head-end.html`) picks the reader's system in each set they have not picked before,
  saving nothing, and shows the packages' Arm buttons in place of the others on an Arm Linux computer. It
  picks a tab with Hextra's own `updateGroup`, which `head-end.html` copies from Hextra's `tabs.js` at build
  time, so a Hextra that changes it fails the build. Verifying an attestation is for the Installation page's
  last section.
- Each game's page is built from `site/data/games/<slug>.json` by `site/content/games/_content.gotmpl`,
  which documents the schema; the manual page reads the same files. Change a game's text there.
- Nothing is copied into the site. `CHANGELOG.md`, `CODE_OF_CONDUCT.md`, `NOTICE.md` and this file's
  Commands and Pull requests sections are mounted and rendered by the `repo-file` shortcode; the README's
  screenshots and the manual page are mounted too (`site/hugo.yaml`).
- A screenshot goes in with the `screenshot` shortcode and a caption that says what the screen shows; the
  caption is also its alt text and shows in the lightbox. PNGs are 960×564, made smaller with
  `optipng -o2`. One that shows the big board's map gets the map's credit line. Wrap several in the
  `screenshots` shortcode to open them as one set, as each game's page does.
- The lightbox is Hextra's, PhotoSwipe, and the search is FlexSearch. Their files and licences are in
  `site/assets/vendor/`, exactly as their npm packages publish them (never edit them; `.gitattributes`
  and `prek.toml` leave them alone), and `site/hugo.yaml` names them. The credits page shows the
  licences, so the build needs no network beyond the Go module proxy.
- Download commands in pages use the `version` shortcode, and those in `site/assets/install/` use
  `@VERSION@`: the latest published release in `docs.yml`'s builds, the latest in `CHANGELOG.md` in
  others. Text about the Windows installer, the Mac app, or WOPR in the Linux app menu (the packages' menu
  entry) goes inside `{{% if-installers %}}`, which shows it only once the latest release has them (every
  release after v0.4.0; the `has-installers` partial), so no build of the site names a file that does not
  exist yet; text for the releases before them goes inside `{{% if-installers "not" %}}`. Its content is
  Markdown only: a shortcode inside it that writes HTML, such as `tabs`, is dropped. Inside a Hextra
  `tab`, write it as `{{< if-installers >}}`: the tab renders the Markdown, and the `%` form fails the
  build there. Text about the `.deb`, the `.rpm` or the archives' `wopr.6` needs no gate: every release
  since v0.4.0 has them, so `if-packages`, which still wraps some of it, always shows it. Nothing names the
  snap until it is published.
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

To prove the job (the M0b canary), push a throwaway branch containing a fake secret that only gitleaks
recognises (not a GitHub-supported token pattern, which push protection blocks), check that its CI run's
`secrets` log says `leaks found` rather than an error, then delete the branch.

## Takedown runbook

If a rights holder asks for material to be removed:

1. Find every copy: the provenance tags name each line's source (`internal/wopr/lines.go`,
   `internal/assets/`, `internal/movie/scenes/`, and the games' film text); the README's screenshots in
   `docs/screenshots/` and the docs site's in `site/static/img/` (game screenshots in
   `site/static/img/games/`) show some of it, and the docs site's pages (`site/content/`,
   `site/data/games/`) quote some, which the manual page (`docs/man/wopr.6`) repeats. The packages'
   AppStream metadata (`packaging/linux/io.github.ghostofgoes.wopr.metainfo.xml.in`) links some of the
   site's screenshots. The `.deb`'s copyright file (`packaging/debian/copyright`) names the files;
   regenerate it afterwards.
2. Remove or replace it in one pull request, and release a patch version. Merging it republishes the
   docs site (`docs.yml`). Once WOPR is in the Snap Store or the Microsoft Store, its release there needs
   the patch too, or the listing unpublished until then.
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
- **Tag ruleset on `v*`:** restrict creation, update and deletion to the owner. Turn on immutable releases.
- **Security:** secret scanning with push protection, private vulnerability reporting.
- **Release signing key** (`docs/PLAN.md` §8, "Signatures"). An environment named `release`
  (Settings → Environments) with deployment tags limited to `v*`, holding the secret `WOPR_SIGNING_KEY`
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

  Keep an offline copy of the secret key, then delete `$GNUPGHOME`. Before the key expires, extend it
  (`gpg --quick-set-expire`) and commit the new `wopr-signing-key.asc`, which keeps the fingerprint. A
  new key needs both files changed in one pull request, and the docs site then shows the new fingerprint.
- **Issues and Discussions:** issues on, with the bug report and feature request forms in
  `.github/ISSUE_TEMPLATE/` (blank issues off); **Discussions on** (Settings → General → Features), since
  the forms, the docs site and the README send questions there, with its default **Announcements**
  category, where `release.yml` starts each release's discussion.
- **Actions:**
  - Allow only `actions/*` and `j178/prek-action`, and require full-length commit-SHA pinning.
  - Set the default `GITHUB_TOKEN` to read-only. Workflows may not create or approve pull requests.
  - Require approval for workflow runs from all external contributors.
- **Dependabot:** off (owner decision). The weekly `scheduled.yml` report covers updates and
  vulnerabilities.
- **Installers:** the Windows installer and the Mac app need no secret, variable or environment, and the
  Linux packages only the release signing key above. The stores do, and only when the owner turns them on:
  - **Snap Store**, when the name is registered. With an Ubuntu One account (two-factor on), run
    `snapcraft register wopr`; a person at the Store reviews new names, which can take days or weeks.
    Create the environment `snap-store` with the owner as required reviewer, deployments from tags `v*`
    only (no branches), and the secret `SNAPCRAFT_STORE_CREDENTIALS` in it alone: the file that
    `snapcraft export-login --snaps=wopr --channels=stable --acls=package_access,package_push,package_update,package_release --expires=<date under a year away> creds.txt`
    writes (`gh secret set SNAPCRAFT_STORE_CREDENTIALS --env snap-store < creds.txt`, then delete the
    file). Then set the repository variable `SNAP_PUBLISH` to `true`; any other value keeps `snap.yml`'s
    publish job skipped. The next `v*` tag uploads both snaps to the stable channel once the owner
    approves the deployment. Export a new login before the old one expires.
  - **Microsoft Store**, when the name is reserved as a game in Partner Center: the repository variables
    `MSIX_IDENTITY_NAME`, `MSIX_PUBLISHER` (`CN=…`) and `MSIX_PUBLISHER_DISPLAY_NAME`, copied from
    Product identity. They are not secret. Until they are set, the MSIX packages carry placeholders and the
    Store would refuse them.
- **Pages:** Settings → Pages → Build and deployment → Source: **GitHub Actions**. `docs.yml` then
  deploys the docs site to <https://ghostofgoes.github.io/WOPR/> through the `github-pages` environment,
  which GitHub creates and limits to deployments from `main`.

## Milestone checklist

At every milestone boundary:

1. Update dependencies to versions at least 14 days old ([Dependency cooldown](#rules-that-tests-enforce)):
   `go list -m -u -json all` shows each update and its `Time`. Update direct dependencies one by one with
   `go get <module>@<version>`, never `go get -u ./...`, which also lifts indirect modules to their newest
   untagged commits; then `go mod tidy`. Update the tool modules the same way, with
   `go get -tool <tool>@<version>` in `tools/`, `tools/lint/`, `tools/release/` and `tools/docs/`, and the
   docs theme with `go -C site get github.com/imfing/hextra@<version>`. Run
   `go run ./internal/tools/cooldown` before pushing. Re-vendor PhotoSwipe (its newest
   5.x, the major version Hextra's script is written for) and FlexSearch (the version the new Hextra
   defaults to, in its `layouts/_partials/scripts/search.html`): download each npm tarball from
   `https://registry.npmjs.org/<name>/-/<name>-<version>.tgz`, check its SHA-512 against the
   registry's `dist.integrity`, copy the same files and `LICENSE` into `site/assets/vendor/<name>/`,
   and update the versions and hashes in `site/hugo.yaml`. Build the docs site: a new Hugo can
   deprecate a setting, which `--panicOnWarning` turns into an error.
2. Run `prek update`, and keep golangci-lint and gitleaks in step between `prek.toml` and their tool modules.
3. Bump action SHAs from their release tags, each at least 14 days old, and the Fedora image digest in
   `smoke.yml` to the newest Fedora release's (`registry.fedoraproject.org/fedora:<N>`).
4. Bump the installers' tools:
   - **Inno Setup**, if <https://jrsoftware.org/isdl.php> lists a newer one: in
     `packaging/windows/build-installer.ps1`, change `$InnoUrl`, `$InnoSha256` (the `file-hash` in the
     release's `.issig` file) and `$InnoSigner` together, with the version in `-InnoDir`'s default and the
     download's name, and read the release notes for changes to scripts.
   - **The Windows SDK**: check that the `windows-2025` and `windows-11-arm` images still have one
     (`WindowsSdk.psm1` takes the newest), and set `MaxVersionTested` in
     `packaging/windows/AppxManifest.xml` to the newest Windows build.
   - **snapcraft**: if `snap info snapcraft` shows a newer major track than `snap.yml`'s `9.x/stable`, move
     to it; check that core24 is still a supported base.
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
