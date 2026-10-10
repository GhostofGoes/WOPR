---
title: Troubleshooting
weight: 2
description: Fixes for common problems, the debug log, and where to ask for help or report a bug.
next: /movie
---

## Common problems

**"standard output is not a terminal".** Run `wopr` straight in a terminal, not through a pipe or a
redirect. On Windows, use Windows Terminal: mintty without ConPTY is not supported.

**"TERM=dumb".** The terminal says it cannot draw a full screen. Use another terminal. `wopr --games` and
the other printing options still work.

**"TERMINAL TOO SMALL".** WOPR needs at least 80 columns and 24 rows. Make the window bigger, and the
session carries on where it was.

**`wopr` is not found.** Open a new terminal window: the one you installed from may not know about the new
program yet. The [install lines](/install) put it in `%LOCALAPPDATA%\Programs\wopr` on Windows,
`~/.local/bin` on macOS and Linux (adding that folder to your `PATH` in `~/.zprofile`, `~/.bashrc` or
`~/.zshrc` when it is not there), and Go's `bin` folder with `go install`.{{% if-packages %}} The `.deb`
puts it in `/usr/games`, which is not on root's `PATH`: as root, or in a small container, run
`/usr/games/wopr`. The `.rpm` puts it in `/usr/bin`.{{% /if-packages %}}

**Windows says Smart App Control blocked `wopr`.** Smart App Control, when it is on, blocks every program
that is not signed, and `wopr` is not signed yet. It has no exception for one program: the only way to run
`wopr` is to turn Smart App Control off (Windows Security, then App & browser control, then Smart App
Control settings). Most PCs have it off, or in evaluation mode, which blocks nothing.

**`sudo` says you are not in the sudoers file.** On Debian, when a root password was set during
installation, your account cannot use `sudo` until it is in the `sudo` group. Add it with the root
password, `su -c "adduser $USER sudo"`, then log out and back in, and paste the install line again.

**An old version still starts after you update.** Another copy comes first on your `PATH`. On Linux and
macOS, `type -a wopr` lists every copy; in PowerShell, `Get-Command -All wopr`. Delete the ones you do not
want.

**Strange colours.** Try another theme (`--theme green`), or turn colour off with `NO_COLOR=1`.

**The mouse wheel changes what I typed.** Some terminals (GNOME Terminal and others built on VTE) turn the
wheel into the Up and Down keys on a full screen, and those step through the lines you typed. Scroll back
with PgUp and PgDn. `wopr` leaves mouse reporting off so that you can still select and copy text.

**Text types too slowly.** Press any key to show the rest at once, or run with `--instant`.

**Something else.** Run with `WOPR_DEBUG=1`, then [report it](#getting-help) with the
[debug log](#debug-log).

## Debug log

If something goes wrong, run `wopr` with `WOPR_DEBUG` set to `1`:

```sh
WOPR_DEBUG=1 wopr                                          # Linux and macOS
```

```powershell
$env:WOPR_DEBUG = 1; wopr; Remove-Item Env:WOPR_DEBUG     # Windows PowerShell
```

It writes a debug log and prints the log's path when it exits. The log records the session's seed, the
terminal's size and the slow work; it never records what you type. It is kept in your user cache folder:

| System | Debug log |
|---|---|
| Linux | `~/.cache/wopr/debug.log` (or `$XDG_CACHE_HOME/wopr/debug.log`) |
| macOS | `~/Library/Caches/wopr/debug.log` |
| Windows | `%LOCALAPPDATA%\wopr\debug.log` |

Each session adds to the end of the log, starting with a line that names the version and the seed, so the
newest session is at the end. If `wopr` stops with a message before its screen appears, it writes no log:
the message is what to report.

## Getting help

- **Questions:** ask in [GitHub Discussions](https://github.com/GhostofGoes/WOPR/discussions/categories/q-a).
- **Something is broken:** open a
  [bug report](https://github.com/GhostofGoes/WOPR/issues/new?template=bug_report.yml). Say what you did,
  what happened, and what `wopr --version` prints (or that it would not install), and attach the
  [debug log](#debug-log), with the seed if you used one.
- **A new feature, or something that differs from the film:** open a
  [feature request](https://github.com/GhostofGoes/WOPR/issues/new?template=feature_request.yml), or talk
  an idea over first in [Ideas](https://github.com/GhostofGoes/WOPR/discussions/categories/ideas).
- **A security problem:** report it privately, as the
  [security policy](https://github.com/GhostofGoes/WOPR/security/policy) says, not in a public issue.

Search the [issues](https://github.com/GhostofGoes/WOPR/issues?q=is%3Aissue) first: someone may have
reported it already.
