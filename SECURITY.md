# Security policy

## Reporting a vulnerability

Please report vulnerabilities privately through GitHub's
[private vulnerability reporting](https://github.com/GhostofGoes/WOPR/security/advisories/new)
rather than in a public issue. Include the version (`wopr --version`), your OS, and steps to
reproduce.

## Supported versions

Only the latest release receives fixes.

## How releases are protected

- Release binaries are built by `.github/workflows/release.yml` from a commit on `main` that passed CI,
  rebuilt a second time to check they are reproducible, and published with a build-provenance
  attestation that covers every file in the release. Verify a download before running it:

  ```sh
  gh attestation verify wopr_<version>_<os>_<arch> --repo GhostofGoes/WOPR \
    --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml \
    --source-ref refs/tags/v<version> --deny-self-hosted-runners
  ```

- The Linux packages are signed with wopr's OpenPGP key, `packaging/wopr-signing-key.asc` (fingerprint
  in `packaging/wopr-signing-key.fingerprint`): each `.rpm` carries its signature, which `dnf`, `zypper` and
  `rpm -K` check once the key is imported, and each `.deb` and `checksums.txt` has a detached `.asc`
  signature for `gpg --verify`. The secret key is only available to the release workflow's signing job,
  in a GitHub environment limited to release tags.
- Tags and releases are never moved or replaced. A bad release is fixed by the next patch version, and
  `go.mod` gains a `retract` directive for the bad one.

## Leaked secrets

CI scans the git history with gitleaks on every branch push and every pull request from a fork. If a real
secret is found, rotate or revoke it first; history on `main` is never rewritten. Then, in a pull request,
add the finding's fingerprint (from the gitleaks log) to `.gitleaksignore` with a comment saying when and
why. False positives are handled the same way.

## Response rule

A reachable `govulncheck` finding, or a Go security release that affects the standard library, leads to a
toolchain bump and a patch release within 14 days.
