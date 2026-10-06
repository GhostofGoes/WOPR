# wopr

> SHALL WE PLAY A GAME?

`wopr` is a terminal recreation of the WOPR (War Operation Plan Response) computer from the 1983 film
*WarGames*. You dial in, find your way past the `LOGON:` prompt, talk to WOPR, and play the games on its
list. That includes Global Thermonuclear War, which cannot be won.

It is a single static binary for Linux, macOS and Windows. It needs no network connection and collects
no data.

> **Status: under construction.** The first release (v0.1.0) arrives with the film's set pieces:
> Global Thermonuclear War, tic-tac-toe, chess and checkers. The other games follow in later releases. See
> [docs/PLAN.md](docs/PLAN.md) for the plan and milestones.

## Install

Release binaries for Linux, macOS (26 Tahoe) and Windows 11, on amd64 and arm64, will be attached to each
[GitHub Release](https://github.com/GhostofGoes/WOPR/releases).

**Verify before you run.** Every release archive carries a build-provenance attestation. With the
[GitHub CLI](https://cli.github.com/):

```sh
gh attestation verify wopr_<version>_<os>_<arch>.tar.gz --repo GhostofGoes/WOPR \
  --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml \
  --source-ref refs/tags/v<version> --deny-self-hosted-runners
```

Then unpack the archive and run `./wopr`.

- **macOS:** the binary is not signed, so Gatekeeper quarantines it. After verifying it, run
  `xattr -d com.apple.quarantine wopr`.
- **Windows:** SmartScreen may warn about an unrecognised app. After verifying the archive, choose
  *More info → Run anyway*.

With Go installed, you can instead build from source:

```sh
go install github.com/GhostofGoes/WOPR/cmd/wopr@latest
```

## Quick start

```sh
wopr
```

Wait for `LOGON:`. If you have seen the film, you know what to type. If not, `wopr --help` will tell you.

Once logged on, talk to WOPR, type `LIST GAMES`, and pick one by name or by its place in the list. Type
`LOGOFF` to leave; Ctrl+C always quits.

## Command-line flags

| Flag | Meaning |
|---|---|
| `-v`, `--version` | print the version and exit |
| `-h`, `--help` | print help and exit |
| `-g`, `--games` | list the games and exit |
| `-L`, `--licenses` | print licence notices and exit |
| `-p`, `--play <game>` | start a game directly (number, name, alias or unique prefix); `wopr <game>` does the same |
| `-m`, `--movie [scene]` | replay the film's WOPR scenes (arrives in v1.1) |
| `-t`, `--theme <name>` | `imsai` (white phosphor, the default), `green`, `amber` or `norad`; env `WOPR_THEME` |
| `-i`, `--instant` | no typewriter pacing; env `WOPR_INSTANT=1` |
| `-s`, `--seed <n>` | deterministic run |
| `-r`, `--reduce-motion` | no blinking or motion; env `WOPR_REDUCE_MOTION=1` |

`NO_COLOR` (any value) turns colour off.

## Troubleshooting

- **"standard output is not a terminal"**: run `wopr` directly in a terminal, not through a pipe. On
  Windows, use Windows Terminal; mintty without ConPTY is not supported.
- **"TERMINAL TOO SMALL"**: WOPR needs at least 80×24.
- **Strange colours**: try `--theme green`, or set `NO_COLOR=1`.

## Development builds

Every push to `main` builds all six targets. Open a
[CI run on main](https://github.com/GhostofGoes/WOPR/actions/workflows/ci.yml?query=branch%3Amain+event%3Apush),
download the `wopr-dev` artifact under **Artifacts**, unzip it, and run the binary for your platform
(`chmod +x` it first on Linux and macOS). Builds from pull requests are for testing only.

## Building and contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Commands and conventions are in [AGENTS.md](AGENTS.md).

## Credits and licence

The code is MIT licensed; see [LICENSE](LICENSE). Film quotations and third-party text are not covered
by the MIT grant; [NOTICE.md](NOTICE.md) explains what they are and who they belong to. This is a fan-made
homage. It is not affiliated with or endorsed by the makers or owners of *WarGames*.
