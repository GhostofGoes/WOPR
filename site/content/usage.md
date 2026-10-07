---
title: Usage
weight: 3
description: Every option and environment variable, the logon, what WOPR understands, the keys, the themes, seeds, the debug log, and troubleshooting.
---

```text
wopr [flags] [game]
wopr --movie [--only] [scene]
```

With no game, `wopr` dials in and waits at `LOGON:`. With a game, it starts that game at once, without the
dial, the logon and the greeting: `wopr chess`, `wopr 7` and `wopr --play chess` all do the same.

`wopr --help` prints a short summary of the options. On Linux and macOS, `man wopr` shows the manual
page: every option, every game with how to play it and tips, and movie mode.
[Installation](/install#the-manual-page) says how to install it.

## Options

Every option has a one-letter form, and options can come before or after the game.

| Option | What it does |
|---|---|
| `-h`, `--help` | Print a summary of the options and exit. |
| `-v`, `--version` | Print the version, the commit and the build date, and exit. |
| `-g`, `--games` | List the games, with the shortest name each one answers to, and exit. |
| `-S`, `--scenes` | List movie mode's scenes and exit. |
| `-L`, `--licenses` | Print the licence, the notices and the licences of the code built into `wopr`, and exit. |
| `-p`, `--play <game>` | Start a game at once. Name it by its number, its name, an alias, or the start of a name that fits only one game. |
| `-m`, `--movie [scene]` | Replay the film's scenes at WOPR's terminal, from the scene named to the end. Without a scene, open the scene menu. See [Movie scenes](/movie). |
| `-o`, `--only` | With `--movie`, play just the chosen scene, then stop. |
| `-t`, `--theme <name>` | The colours: `imsai` (the default), `green`, `amber` or `norad`. See [Themes](#themes). |
| `-i`, `--instant` | Show text at once instead of typing it out. |
| `-s`, `--seed <n>` | Play a repeatable session. See [Seeds](#seeds). |
| `-r`, `--reduce-motion` | Nothing blinks, the front panel's lights stand still, and the ending keeps a steady pace. |

## Environment variables

| Variable | What it does |
|---|---|
| `WOPR_THEME` | The theme, as `--theme` sets it. The option wins if you give both. |
| `WOPR_INSTANT` | Set to `1` for `--instant`. |
| `WOPR_REDUCE_MOTION` | Set to `1` for `--reduce-motion`. |
| `WOPR_SEED` | The seed, as `--seed` sets it. The option wins if you give both. |
| `WOPR_PANEL` | `1` shows the front panel, a status line along the bottom, in any theme; `0` hides it. Unset or empty, only the `norad` theme shows it. |
| `WOPR_DEBUG` | Set to `1` to write a [debug log](#debug-log). |
| `NO_COLOR` | Set to any value but an empty one to turn colour off. See [Accessibility](/accessibility). |

For the on and off variables, `0`, `false`, `no` and `off` mean off, and any other value means on. An
empty value counts as unset.

`wopr` exits with 0 after `LOGOFF`, 130 after Ctrl+C, 2 for a mistake on the command line or a terminal it
cannot use, and 1 for any other error.

## Logging on

`wopr` dials in, connects, and asks `LOGON:`. The film's backdoor works: `Joshua`. A name WOPR does not
know drops the line, and WOPR dials again; every third failed try, it gives you a hint. At `LOGON:` you can
also type `HELP LOGON`, `HELP GAMES` and `LIST GAMES`, as David does in the film.

{{< screenshot src="img/logon-2.png" caption="`HELP GAMES` and `LIST GAMES` at the `LOGON:` prompt, before logging on." >}}

## Talking to WOPR

After the logon, WOPR greets you as Professor Falken and follows the film's conversation. There is no
prompt text while you talk: what WOPR said last is the question. Type in any mix of capitals and small
letters.

| You type | WOPR |
|---|---|
| `HELP` | Lists the commands. |
| `HELP GAMES` | Says what its games are. |
| `LIST GAMES` | Lists the games. Then type a number to pick one. |
| `PLAY <game>`, or a game's name | Starts the game. `Let's play chess.` and `How about poker?` work too. |
| `LOGOFF` | Ends the session. `LOG OFF`, `EXIT` and `QUIT` do too, even inside a game. |

WOPR also answers some of the film's questions, such as `Is this a game or is it real?` and
`What is the primary goal?`. It is a scripted program, not a chatbot: when it does not understand, it asks
you to restate your request, calls it an improper request, or offers a game.

## Keys

| Key | What it does |
|---|---|
| Enter | Send the line you typed. |
| Backspace | Delete the last character. Ctrl+U clears the line. |
| Up and Down | Step through the lines you have typed. |
| PgUp and PgDn | Scroll back through earlier text. |
| Any key | While WOPR is typing, show the rest of its text at once. What you type is kept. |
| Esc, twice | Leave a game and go back to WOPR. The first Esc asks you to press it again within three seconds. |
| Ctrl+D | On an empty line, end the session. |
| Ctrl+C | Quit at once. |

Falken's Maze reads single keys instead of lines, and says which keys it uses.

## Themes

The theme sets the colours. Choose one with `--theme` or `WOPR_THEME`:

- `imsai`, the default: white phosphor, as on David's IMSAI terminal in the film.
- `green`: green phosphor.
- `amber`: amber phosphor.
- `norad`: the blue of NORAD's big board, with the front panel along the bottom.

```sh
wopr --theme amber
```

{{< screenshot src="img/themes-green.png" caption="`--theme green`, on the chess board." >}}

{{< screenshot src="img/themes-amber.png" caption="`--theme amber`, as Global Thermonuclear War asks you to pick a side." >}}

{{< screenshot src="img/themes-norad.png" caption="`--theme norad`, with the front panel along the bottom." >}}

{{< screenshot src="img/themes-nocolor.png" caption="`NO_COLOR=1`: no colour at all. Every card still shows its suit letter." >}}

`wopr` uses as many colours as your terminal has: full colour, 256 colours, or the basic 16. With
`NO_COLOR`, or on a terminal without colour, every theme uses bold, dim, underline and reverse instead,
and nothing is lost: see [Accessibility](/accessibility).

## Seeds

Every session starts from a random seed, so the cards, the maze and WOPR's choices differ each time. Give
a seed to play the same session again:

```sh
wopr poker --seed 1983
```

The same seed deals the same cards, builds the same maze and gives the same replies from WOPR: with a
seed, WOPR's searches stop at a fixed depth rather than when time runs out. A seed is a whole number from
0 up. Movie mode ignores seeds: every replay is the same anyway.

## Debug log

If something goes wrong, run `wopr` with `WOPR_DEBUG=1`. It writes a debug log and prints the log's path
when it exits. The log records the session's seed, the terminal's size and the slow work; it never records
what you type. It is kept in your user cache folder:

| System | Debug log |
|---|---|
| Linux | `~/.cache/wopr/debug.log` (or `$XDG_CACHE_HOME/wopr/debug.log`) |
| macOS | `~/Library/Caches/wopr/debug.log` |
| Windows | `%LOCALAPPDATA%\wopr\debug.log` |

Each session adds to the end of the log. Attach it to a
[bug report](https://github.com/GhostofGoes/WOPR/issues), with the seed if you used one.

## Troubleshooting

**"standard output is not a terminal".** Run `wopr` straight in a terminal, not through a pipe or a
redirect. On Windows, use Windows Terminal: mintty without ConPTY is not supported.

**"TERM=dumb".** The terminal says it cannot draw a full screen. Use another terminal. `wopr --games` and
the other printing options still work.

**"TERMINAL TOO SMALL".** WOPR needs at least 80 columns and 24 rows. Make the window bigger, and the
session carries on where it was.

**Strange colours.** Try another theme (`--theme green`), or turn colour off with `NO_COLOR=1`.

**The mouse wheel changes what I typed.** Some terminals (GNOME Terminal and others built on VTE) turn the
wheel into the Up and Down keys on a full screen, and those step through the lines you typed. Scroll back
with PgUp and PgDn. `wopr` leaves mouse reporting off so that you can still select and copy text.

**Text types too slowly.** Press any key to show the rest at once, or run with `--instant`.

**Something else.** Run with `WOPR_DEBUG=1` and [open an issue](https://github.com/GhostofGoes/WOPR/issues)
with the [debug log](#debug-log).
