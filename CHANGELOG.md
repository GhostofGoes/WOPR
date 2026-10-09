# Changelog

All notable changes to wopr, newest first. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

[changie](https://changie.dev) generates this file from the notes in the repository's `.changes/` folder.
Edit those notes, not this file.

## v0.4.0 - 2026-10-08

### Added

- Each release lists what changed, on its page and in `CHANGELOG.md`.
- Linux and macOS downloads now include a manual page with tips for each game.
- A new website shows how to install wopr and how to play every game.
- You can now install wopr on Linux with a .deb or .rpm package.

## v0.3.0 - 2026-10-07

### Added

- `WOPR_SEED` sets the seed, as `--seed` does.
- `wopr -m 3 --only` plays just one movie scene, then stops.
- Hearts and Bridge say the cards already in the trick before you play.
- The README now has screenshots and a section on accessibility.

### Changed

- The card that wins a trick now has a border around it.
- Tic-tac-toe underlines the last mark played.
- The DEFCON ladder points at the current level with a `>`.
- Gin Rummy's panel now names your melds.
- `wopr --help` now says what `--reduce-motion` does.

### Fixed

- Fixed the ending speeding up when `--reduce-motion` is on.
- Fixed Black's checkers kings looking like White's men without colour.

## v0.2.0 - 2026-10-07

### Added

- Movie mode: `wopr --movie` replays the film's scenes at WOPR's terminal.
- `wopr --scenes` lists the scenes, and `wopr -m 3` starts from the third.
- Each release now has the bare program for each platform, ready to run.
- Global Thermonuclear War knows 85 more targets and today's city names.
- Type `list` in Global Thermonuclear War to see the targets you can pick.
- Checkers moves can be typed without the dash, like `c3d4`.

### Changed

- The chess board is now square.
- Whole cards show their rank and suit in both corners, like real cards.
- Some cities sit in better places on the big board.

### Fixed

- Fixed Global Thermonuclear War hitting the wrong city for some names.
- Fixed old text showing under a game's board when the game starts.
- Fixed Gin Rummy showing the score twice when you leave.
- Fixed a label in Fighter Combat's score table.

## v0.1.0 - 2026-10-07

### Added

- The first release: the WOPR computer from WarGames, in your terminal.
- Dial in, get past the LOGON: prompt, and talk with WOPR as in the film.
- Every game on WOPR's list can be played, and tic-tac-toe too.
- Global Thermonuclear War: pick a side and targets, then watch the board.
- The film's ending: WOPR plays tic-tac-toe against itself and learns.
- Chess and Checkers against WOPR.
- Card games: Black Jack, Poker, Gin Rummy, Hearts and a simple Bridge.
- War games: Desert Warfare, Guerrilla Engagement and two Theaterwide games.
- Air games: Fighter Combat and Air-to-Ground Actions.
- Falken's Maze: find the exit while WOPR moves the walls you have not seen.
- Every game has its own ASCII art.
- Four colour themes: white (the default), green, amber and NORAD.
- `wopr chess` or `--play` starts a game at once; `--games` lists them.
- `--instant` turns off the slow typing.
- `--seed` makes a run play out the same way every time.
- `--reduce-motion` stops the blinking, and `NO_COLOR` turns colour off.
- Downloads for Linux, macOS and Windows, on both amd64 and arm64.
- You can check that a download is real with `gh attestation verify`.
