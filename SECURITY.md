# Security policy

## Reporting a vulnerability

Please report vulnerabilities privately through GitHub's
[private vulnerability reporting](https://github.com/GhostofGoes/WOPR/security/advisories/new).
Do not report them in a public issue. Include the version (`wopr --version`), your OS, and the steps
to reproduce the problem.

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
