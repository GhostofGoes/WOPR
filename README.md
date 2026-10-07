# wopr

> SHALL WE PLAY A GAME?

`wopr` is a terminal recreation of the WOPR (War Operation Plan Response) computer from the 1983 film
*WarGames*. You dial in, find your way past the `LOGON:` prompt, talk to WOPR, and play the games on its
list. That includes Global Thermonuclear War, which cannot be won.

It is a single static binary for Linux, macOS and Windows. It needs no network connection and collects
no data.

> **Status: under construction.** Every game on the list is playable as of v0.1.0. Movie mode
> (`--movie`) and a pass over the film's text against the film are still to come. See
> [docs/PLAN.md](docs/PLAN.md) for the plan and milestones.

## Install

Each [GitHub Release](https://github.com/GhostofGoes/WOPR/releases) has builds for Linux, macOS (26 Tahoe)
and Windows 11, on amd64 and arm64. For each platform there is the bare binary, such as
`wopr_<version>_linux_amd64` or `wopr_<version>_windows_amd64.exe`, and an archive of it with the README,
licence and notices (`.tar.gz`, or `.zip` for Windows). `LICENSE`, `README.md`, `NOTICE.md`,
`THIRD_PARTY_NOTICES.txt` and `checksums.txt` are attached too. (v0.1.0 has the archives only.)

**Verify before you run.** Every file in a release carries a build-provenance attestation. With the
[GitHub CLI](https://cli.github.com/):

```sh
gh attestation verify wopr_<version>_<os>_<arch> --repo GhostofGoes/WOPR \
  --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml \
  --source-ref refs/tags/v<version> --deny-self-hosted-runners
```

Then run it. On Linux and macOS a downloaded binary is not executable yet:
`chmod +x wopr_<version>_<os>_<arch>`, then `./wopr_<version>_<os>_<arch>` (rename it to `wopr` if you
like). An archive keeps the executable bit: unpack it and run `./wopr`.

- **macOS:** the binary is not signed, so Gatekeeper quarantines it. After verifying it, run
  `xattr -d com.apple.quarantine` on it.
- **Windows:** SmartScreen may warn about an unrecognised app. After verifying the file, choose
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

## Games

Every game in `LIST GAMES` is playable, and each has its own ASCII art: title pieces, card faces and trick
tables, framed boards, a picture of the front in the war games, and Global Thermonuclear War's outlines of the
two nations (drawn from [Natural Earth](https://www.naturalearthdata.com/) data). Its big board uses Matthew
Thomas's ASCII world map; see [NOTICE.md](NOTICE.md).

- **Falken's Maze** (`maze`): find the exit with the arrow keys (or WASD); `q` gives up. WOPR watches which
  way you turn and moves walls you have not seen yet. The exit always stays reachable.
- **Desert Warfare** (`desert`): order each unit in turn: `move 4` or `move sollum`, `attack 5`, `hold`;
  `status` shows the map, `help` the orders, `end` holds the rest. Take Benghazi, or hold more of the road
  after ten turns. Units cut off from their depot attack at half.
- **Fighter Combat** (`dogfight`): each turn pick a manoeuvre (`press`, `extend`, `climb`, `break`, `fire`)
  while WOPR picks its own. Missiles at range, guns close in from behind; two hits bring an aircraft down.
  `disengage` at long range ends it even.
- **Guerrilla Engagement** (`guerrilla`): your cells hide (`hide`), ambush from hiding at double strength
  (`attack 4`), and `recruit` with support. Patrols find cells near them. Survive twelve turns, reach
  support 10, or take the capital.
- **Air-to-Ground Actions** (`air-to-ground`): six sorties against six defended targets. Set a package
  (`target radar strike 4 sead 2 escort 2`) and `go`. SEAD fights the target's SAM sites, escorts tie up
  interceptors, and WOPR hides a mobile SAM battery where it expects you. Ten points win.
- **Theaterwide Tactical Warfare** (`tactical`): corps and air wings on a European front. Any unit may
  `escalate`; each rung adds to every attack, WOPR answers in kind, and the top rung ends everything. Hold
  five regions to win.
- **Theaterwide Biotoxic and Chemical Warfare** (`biotoxic`): `release 4` puts an agent on a region, `decon`
  cleans a little; it spreads with the wind. Nobody wins.
- **Global Thermonuclear War** (`gtw`): choose a side and list target cities (an empty line ends the list);
  the first strike flies at once. Then order two more strikes as percentages of your ICBMs, SLBMs and bombers
  (`50 25 100`, `icbm 50`, `hold`; Enter carries out WOPR's war plan, `auto` hands it the rest) while DEFCON
  falls to 1. WOPR holds a quarter more of everything. Then try to stop it.
- **Chess**: you are White. Type moves as `e2e4` or `Nf3`; `resign` ends the game.
- **Checkers**: you are Black and move first. Type `c3-d4`, or `c3xe5` to jump (`c3xe5xg7` to jump twice);
  jumps are compulsory.
- **Black Jack** (`blackjack`): you sit down with $100. Bet $1 to $25, then `h` hit, `s` stand, `d` double
  down, `p` split a pair. The dealer stands on soft 17; black jack pays 3 to 2. `leave` cashes out.
- **Poker**: heads-up five-card draw, 100 chips each. `c` checks or calls, `b` bets or raises, `f` folds. At
  the draw, name up to three cards to throw away by position (`1 3`) or by name (`7h`). WOPR bluffs.
- **Gin Rummy** (`gin`): first to 100. `s` draws from the stock, `d` takes the discard; then discard by name
  or position. `knock 7h` discards the 7H and knocks (10 or less deadwood); with none, it is gin.
- **Hearts**: you are South against three WOPR seats. Pass three cards (`qs ah 2d`, or positions), then play
  a card a trick. Hearts are a point each, the queen of spades 13; the lowest score at 100 wins.
- **Bridge** (minimal): WOPR bids all four hands by point count and your side always declares. You play
  both your hand and dummy's (`7h`); WOPR defends. Make the contract to win.
- **Tic-tac-toe** (`ttt`, not on the list): squares are numbered 1 to 9. WOPR never loses. Try zero players.

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
- **Reporting a bug**: run with `WOPR_DEBUG=1`. wopr writes a debug log (never what you type) and prints its
  path when it exits; attach it to the issue, with `--seed` if you used one.

## Development builds

Every CI run builds all six targets and keeps one download per platform. Open a
[CI run on main](https://github.com/GhostofGoes/WOPR/actions/workflows/ci.yml?query=branch%3Amain+event%3Apush)
and, under **Artifacts**, download the one for your platform, for example
`wopr_0.0.1-snapshot.1a2b3c4_linux_amd64` or `..._windows_amd64.exe` (about 6 MB). It is the bare binary:
`chmod +x` it on Linux and macOS, and `wopr --licenses` prints its licence and notices. Downloads from
`main` are kept for 30 days; builds from other branches (7 days) and pull requests (3 days) are for
testing only.

## Building and contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Commands and conventions are in [AGENTS.md](AGENTS.md).

## Credits and licence

The code is MIT licensed; see [LICENSE](LICENSE). Film quotations and third-party text are not covered
by the MIT grant; [NOTICE.md](NOTICE.md) explains what they are and who they belong to. This is a fan-made
homage. It is not affiliated with or endorsed by the makers or owners of *WarGames*.
