# Governance

`wopr` is a single-maintainer project. This document says how decisions are made, who has which role, and
what happens if the maintainer cannot continue.

## How decisions are made

The maintainer makes the decisions about the project: its scope, its design, what to merge and when to
release. Anyone can take part in the discussion before a decision:

- Propose a change or report a bug in an [issue](https://github.com/GhostofGoes/WOPR/issues).
- Ask questions and discuss ideas in [Discussions](https://github.com/GhostofGoes/WOPR/discussions).
- Discuss a specific change in its pull request.

The maintainer records the decisions that change the design or the scope in `docs/PLAN.md`, with the date.
The day-to-day rules are in [AGENTS.md](AGENTS.md). The plans for the next year are in
[ROADMAP.md](ROADMAP.md).

## Roles

| Role | Who | Responsibilities |
|---|---|---|
| Maintainer | Christopher Goes ([@GhostofGoes](https://github.com/GhostofGoes)) | Decides the scope and the design. Reviews and merges pull requests. Answers issues and Discussions. Tags releases and checks them (`AGENTS.md`, "Releasing"). Holds the repository's owner rights and the offline copy of the release signing key. Keeps the repository settings in `AGENTS.md` up to date. |
| Backup maintainer | Ryan Adams ([@adamsrnmsu](https://github.com/adamsrnmsu)) | Reviews and merges pull requests, and handles issues, when the maintainer asks. Takes over as maintainer if the maintainer cannot continue ([Continuity](#continuity)). |
| Security contact | The maintainer | Receives private vulnerability reports and handles them as [SECURITY.md](SECURITY.md#how-reports-are-handled) says. |
| Code of conduct contact | The maintainer | Receives and handles reports under the [Code of Conduct](CODE_OF_CONDUCT.md). |
| Contributor | Anyone who opens an issue, a discussion or a pull request | Follows [CONTRIBUTING.md](CONTRIBUTING.md) and the Code of Conduct. |

AI agents that change the repository are contributors. They follow the same rules and
[AGENTS.md](AGENTS.md), and the maintainer reviews their changes before they merge.

## Access

Everyone with write access to the repository, or access to private vulnerability reports, must turn on
two-factor authentication in GitHub. Use a passkey, a security key or an authenticator app. Do not use
SMS (text messages) as the only second factor. The maintainer checks this before giving a person access.

## Continuity

The project must be able to continue within a week if the maintainer cannot continue.

The backup maintainer is Ryan Adams ([@adamsrnmsu](https://github.com/adamsrnmsu)). Ryan is a
collaborator on the repository, so Ryan can already do these things:

- Create, label and close issues, and answer Discussions.
- Review and merge pull requests that pass `ci-ok`.

Two things need the maintainer's own access: tagging a release (the tag ruleset limits `v*` tags to the
owner) and the offline copy of the release signing key. The maintainer keeps the credentials for both in
emergency access that Ryan can open if the maintainer cannot continue. With them, Ryan can tag a release,
and `release.yml` builds, signs and publishes it.
