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
| Security contact | The maintainer | Receives private vulnerability reports and handles them as [SECURITY.md](SECURITY.md#how-reports-are-handled) says. |
| Code of conduct contact | The maintainer | Receives and handles reports under the [Code of Conduct](CODE_OF_CONDUCT.md). |
| Contributor | Anyone who opens an issue, a discussion or a pull request | Follows [CONTRIBUTING.md](CONTRIBUTING.md), [AGENTS.md](AGENTS.md) and the Code of Conduct. |

AI agents that change the repository are contributors. They follow the same rules, and the maintainer
reviews their changes before they merge.

## Continuity

The project must be able to continue within a week if the maintainer cannot continue.

<!-- The maintainer completes this section: name the person and say what they can reach. -->
**To be decided by the maintainer.** This section will name the person who can take over, and how they get
the access that they need: the owner rights to the repository, the `release` environment and its signing
key secret, and the offline copy of the signing key.

With that access, the person can create and close issues, merge pull requests, and tag a release, which
`release.yml` builds and publishes.
