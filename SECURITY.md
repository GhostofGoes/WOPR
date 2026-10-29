# Security policy

## Reporting a vulnerability

Please report vulnerabilities privately through GitHub's
[private vulnerability reporting](https://github.com/GhostofGoes/WOPR/security/advisories/new).
Do not report them in a public issue. Include the version (`wopr --version`), your OS, and the steps
to reproduce the problem.

## How reports are handled

1. We acknowledge your report within 14 days.
2. We check the report and tell you if we can reproduce the problem and how severe we think it is.
3. We make the fix in private, in the advisory's temporary private fork.
4. We publish a patch release with the fix. Then we publish the advisory, with a CVE ID when the problem
   needs one. The release notes name the advisory.
5. The advisory and the release notes give credit to you, unless you ask us not to name you.

We keep you informed at each step. If we do not agree that a report is a vulnerability, we tell you why.

## What to expect

`wopr` is a game that runs in your terminal. These are its security properties:

- It has no network code. It does not send or receive data, and it collects no data.
- It reads only its command-line options and these environment variables: `WOPR_THEME`, `WOPR_SEED`,
  `WOPR_INSTANT`, `WOPR_REDUCE_MOTION`, `WOPR_PANEL`, `WOPR_DEBUG`, `NO_COLOR` and `TERM`. It rejects
  values that are not valid. Its terminal libraries also read `COLORTERM`, `CLICOLOR`, `CLICOLOR_FORCE`,
  `TERM_PROGRAM`, `SSH_TTY` and `WT_SESSION` to choose colours and screen features.
- It removes `TMUX`, `TEA_TRACE`, `TEA_DEBUG` and `UV_DEBUG` from its environment when it starts, before
  its libraries can read them. With them, the libraries would run `tmux info`, or write a trace of what
  you type or a log file. Inside tmux, `wopr` uses the colours that `TERM` and `COLORTERM` give.
- It writes only one file: the debug log, and only when `WOPR_DEBUG` is set. The log is
  `wopr/debug.log` in your user cache folder. The folder is readable only by you (0700) and the file too
  (0600). `wopr` does not follow a symbolic link at that path, and it does not use a folder that belongs
  to a different user. The log holds the version, the seed, the terminal size, the theme and timings. It
  does not hold your environment or what you type. When the log is larger than 5 MB, `wopr` empties it
  before it writes, so the log cannot fill your disk.
- It removes escape sequences and control characters from what you type or paste, so the text cannot
  control your terminal.
- It starts no other program, with one exception. On macOS, when you open the app from Finder, `wopr`
  opens itself again in Terminal.
- Each release file is attested and checksummed, and the Linux packages and `checksums.txt` are signed.
  See [How releases are protected](#how-releases-are-protected).

`wopr` cannot protect you from these:

- A computer or terminal that an attacker already controls.
- A copy of `wopr` that you did not download from this project's releases, its packages or
  `go install`, or that you did not verify.
- Problems in your terminal program itself.

`wopr` is not a sandbox. It has the same permissions as your user account. The
[assurance case](docs/ASSURANCE.md) explains why these properties hold.

## Supported versions

Only the latest release receives fixes.

## How releases are protected

- `.github/workflows/release.yml` builds the release binaries from a commit on `main` that passed CI.
  It builds them a second time to check that they are reproducible. It publishes them with a
  build-provenance attestation that covers every file in the release. Check a download before you run
  it:

  ```sh
  gh attestation verify wopr_<version>_<os>_<arch> --repo GhostofGoes/WOPR \
    --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml \
    --source-ref refs/tags/v<version> --deny-self-hosted-runners
  ```

- The release workflow signs the Linux packages with wopr's OpenPGP key, `packaging/wopr-signing-key.asc`.
  Its fingerprint is in `packaging/wopr-signing-key.fingerprint`. Each `.rpm` carries its signature.
  `dnf`, `zypper` and `rpm -K` check that signature once you import the key. Each `.deb` and
  `checksums.txt` has a detached `.asc` signature for `gpg --verify`. Only the release workflow's signing
  job can use the secret key. The key is in a GitHub environment limited to release tags.
- We never move or replace tags and releases. The next patch version fixes a bad release, and `go.mod`
  gains a `retract` directive for the bad one.

## Leaked secrets

CI scans the git history with gitleaks on every branch push and every pull request from a fork. If the
scan finds a real secret, rotate or revoke it first. We never rewrite the history on `main`. Then, in a
pull request, add the finding's fingerprint (from the gitleaks log) to `.gitleaksignore`. Add a comment
that says when and why. Handle a false positive the same way.

## Response rule

A reachable `govulncheck` finding leads to a toolchain bump and a patch release within 14 days. A Go
security release that affects the standard library leads to the same.
