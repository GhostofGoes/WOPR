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

## Games

Playable now (the rest of `LIST GAMES` answers `GAME ROUTINE NOT AVAILABLE` until its release):

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
- **Theaterwide Tactical Warfare** (`tactical`): corps and air wings on a European front. Any unit may
  `escalate`; each rung adds to every attack, WOPR answers in kind, and the top rung ends everything. Hold
  five regions to win.
- **Theaterwide Biotoxic and Chemical Warfare** (`biotoxic`): `release 4` puts an agent on a region, `decon`
  cleans a little; it spreads with the wind. Nobody wins.
- **Global Thermonuclear War** (`gtw`): choose a side, list target cities (an empty line ends the list), and
  watch the big board. Then try to stop it.
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
