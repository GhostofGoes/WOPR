# Assurance case

This document explains why `wopr` meets the security properties that [SECURITY.md](../SECURITY.md#what-to-expect)
states. It gives the threat model, the trust boundaries, the secure design principles that the project
applies, and how the project counters common weaknesses. Each claim names the code, test or check that
supports it. The security review of 2026-10-09 is the source of most of the evidence.

Update this document when a change moves a trust boundary, for example when a change adds network access,
reads a new file or environment variable, or starts a different program.

## Scope

The case covers the `wopr` program, the release files (archives, binaries, Linux packages, the Windows
installer, the Mac app) and the process that builds them. It does not cover the docs site, which is
static pages with no scripts from other sites, and no forms or accounts.

## Threat model

`wopr` is a game for one user, in that user's terminal, with that user's permissions. It keeps no secrets
and no accounts. So the assets to protect are these:

- The user's terminal and account: input to `wopr` must not make it change other files, run programs or
  send escape sequences to the terminal.
- The availability of the terminal: no input may leave the terminal in raw mode or make `wopr` stop
  responding.
- The integrity of the release files: a user must get the program that the source on `main` builds.

The attackers that the project considers:

| Attacker | What they can do |
|---|---|
| Text that the user pastes | Put any bytes into the input, such as escape sequences, control characters or very long or large numbers |
| A local user on a shared computer | Make files and symbolic links in shared folders, such as a cache folder in `/tmp` |
| An attacker on the network | Change downloads that use plain HTTP, or redirect HTTPS to HTTP |
| A compromised dependency or CI action | Put code into the build |
| A contributor | Send a pull request that changes the code or the workflows |

Out of scope: an attacker who already controls the user's account or terminal, a terminal program's own
defects, and a copy of `wopr` that the user did not get from this project.

## Trust boundaries

| # | Boundary | What crosses it | Control |
|---|---|---|---|
| B1 | Keyboard and paste to `wopr` | Keys and pasted text | `console.SanitizeInput` and `console.SanitizeText` (`internal/ui/console/sanitize.go`) remove escape sequences and control characters. Each game parses only its own commands and caps numbers. |
| B2 | Environment and command line to `wopr` | Options and eight environment variables | `internal/cli` and `cmd/wopr` parse them strictly: a fixed list of themes, a number for the seed, a fixed set of boolean values. Their tests cover each one. |
| B3 | `wopr` to the file system | The debug log only, when `WOPR_DEBUG` is set | `internal/debuglog`: a 0700 folder that the user owns, a 0600 file, `O_NOFOLLOW`, and `Lstat` checks. |
| B4 | `wopr` to the terminal | Screen output | All output is printable ASCII drawn by the renderer. Text from the user goes through B1 before it is shown. |
| B5 | `wopr` to other programs | Only on macOS, when Finder opens the app | `cmd/wopr/launch_darwin.go` opens `wopr` again in Terminal with fixed arguments, and no shell. |
| B6 | Source to release files | The build | `release.yml`, below. |
| B7 | Release files to the user | Downloads | HTTPS only, attestations, OpenPGP signatures. |

## Secure design principles

The principles of Saltzer and Schroeder, as `wopr` applies them:

- **Economy of mechanism.** One static Go binary with no network code, no plug-ins and no configuration
  files. The import rules in `internal/archtest` keep the terminal code in `internal/ui` only, and keep
  `os/exec` out of every package except `cmd/wopr` and the build-time tools.
- **Fail-safe defaults.** An environment variable with a value that is not valid is rejected, not
  guessed. The debug log is off unless `WOPR_DEBUG` is set. If the debug log cannot be opened safely,
  `wopr` runs without it.
- **Least privilege.** `wopr` needs no administrator rights. The command-line installers for macOS and
  Linux install to `~/.local/bin` without `sudo`, and the Windows installer is per user. CI workflows
  have `permissions: {}` by default and give each job only what it needs. Only the release workflow's
  `sign` job can use the signing key, in an environment limited to release tags.
- **Complete mediation.** All input goes through the host's input path (B1), so no game sees raw
  terminal bytes. The host owns Esc and Ctrl+C.
- **Open design.** The source, the workflows and this case are public. The security comes from the
  checks, not from secrecy.
- **Separation of privilege.** `main` accepts only pull requests that pass the `ci-ok` check. Only the
  owner can make release tags.
- **Least common mechanism.** `wopr` shares no state between users and keeps nothing between runs except
  the optional debug log.
- **Psychological acceptability.** The safe way to install is the documented way, and the docs show how
  to verify a download.

## Common weaknesses and how they are countered

| Weakness (CWE) | Where it could occur | Counter | Evidence |
|---|---|---|---|
| Terminal escape injection (CWE-150) | Pasted text shown on the screen | B1 sanitizers | Sanitizer tests and fuzz targets in `internal/ui/console`; the 2026-10-09 review fuzzed the UI model with pasted escape sequences |
| Uncaught panic, out-of-range index (CWE-248, CWE-129) | Game parsers | Go's bounds checks; fixes and regression tests for every crash found | All fuzz targets run in CI; the 2026-10-09 review fuzzed every game through the real host |
| Integer overflow, uncontrolled resource use (CWE-190, CWE-400) | Typed counts in games | Counts capped at 999 | Regression tests for the AIR-TO-GROUND ACTIONS fix |
| Infinite loop that blocks the UI (CWE-835) | A bug in a game | The watchdog (`internal/ui/watchdog.go`) restores the terminal and exits | Watchdog tests |
| Symbolic link following, insecure permissions (CWE-59, CWE-732) | The debug log | `O_NOFOLLOW`, `Lstat`, owner check, 0700 and 0600 | `internal/debuglog` tests |
| OS command injection (CWE-78) | The macOS relaunch | Fixed arguments, no shell | Table tests in `cmd/wopr` |
| Use of a weak random generator for security (CWE-338) | None: the seeded generator is for games and replays only | No security decision uses randomness | `proto.NewRand`; gosec's finding is annotated |
| Download of code without an integrity check (CWE-494) | Installs | HTTPS only (`--proto '=https'`, `--https-only`), attestations, signatures | `SECURITY.md`; the install pages |
| Vulnerable or malicious dependency (CWE-1395) | Go modules, CI actions | `govulncheck` on every push and weekly; a 14-day dependency cooldown; actions pinned by commit SHA; zizmor | `ci.yml`, `scheduled.yml`, `internal/tools/cooldown` |
| Leaked credentials (CWE-798) | The repository | gitleaks on the full history in CI; secrets only in GitHub environments | The `secrets` job |
| Tampered build (CWE-506) | CI | Builds are done twice on two architectures and must match; every file is attested; checksums are signed | `release.yml`'s `repro`, `attest` and `sign` jobs |

`golangci-lint` with gosec, staticcheck and go vet runs on every push, and CI fails on any finding. A false
positive from gosec carries a `//nolint:gosec` comment with its reason.

## Residual risks

- One person holds the owner rights and the release signing key. The backup maintainer can reach them
  through emergency access ([GOVERNANCE.md](../GOVERNANCE.md#continuity)).
- The Windows installer and the Mac app are not signed by Microsoft or Apple yet (issues #10 and #14),
  so the systems warn about them. The attestation still proves where they came from.
- Optional milestone M7 (`docs/PLAN.md` §4.7) would add network access for an opt-in LLM brain. That
  changes B1 to B7, and this case must be updated before it ships.
