---
title: Troubleshooting
weight: 2
description: Fixes for common problems, the debug log, and where to ask for help or report a bug.
---

## Common problems

**"standard output is not a terminal".** Run `wopr` straight in a terminal, not through a pipe or a
redirect. On Windows, use Windows Terminal: mintty without ConPTY is not supported.

**"TERM=dumb".** The terminal says it cannot draw a full screen. Use another terminal. `wopr --games` and
the other printing options still work.

**"TERMINAL TOO SMALL".** WOPR needs at least 80 columns and 24 rows. Make the window bigger, and the
session carries on where it was.

**`wopr` is not found.** Open a new terminal window: the one you installed from may not know about the new
program yet. [Installation](/install) says where each way of installing puts it.

**Strange colours.** Try another theme (`--theme green`), or turn colour off with `NO_COLOR=1`.

**The mouse wheel changes what I typed.** Some terminals (GNOME Terminal and others built on VTE) turn the
wheel into the Up and Down keys on a full screen, and those step through the lines you typed. Scroll back
with PgUp and PgDn. `wopr` leaves mouse reporting off so that you can still select and copy text.

**Text types too slowly.** Press any key to show the rest at once, or run with `--instant`.

**Something else.** Run with `WOPR_DEBUG=1`, then [report it](#getting-help) with the
[debug log](#debug-log).

## Debug log

If something goes wrong, run `wopr` with `WOPR_DEBUG=1`. It writes a debug log and prints the log's path
when it exits. The log records the session's seed, the terminal's size and the slow work; it never records
what you type. It is kept in your user cache folder:

| System | Debug log |
|---|---|
| Linux | `~/.cache/wopr/debug.log` (or `$XDG_CACHE_HOME/wopr/debug.log`) |
| macOS | `~/Library/Caches/wopr/debug.log` |
| Windows | `%LOCALAPPDATA%\wopr\debug.log` |

Each session adds to the end of the log. To turn it on:

```sh
WOPR_DEBUG=1 wopr                # Linux and macOS
```

```powershell
$env:WOPR_DEBUG = 1; wopr        # Windows PowerShell
```

## Getting help

- **Questions and ideas:** ask in
  [GitHub Discussions](https://github.com/GhostofGoes/WOPR/discussions).
- **Something is broken:** open a
  [bug report](https://github.com/GhostofGoes/WOPR/issues/new?template=bug_report.yml). Say what you did,
  what happened, and what `wopr --version` prints, and attach the [debug log](#debug-log), with the seed
  if you used one.
- **A new feature, or something that differs from the film:** open a
  [feature request](https://github.com/GhostofGoes/WOPR/issues/new?template=feature_request.yml).
- **A security problem:** report it privately, as the
  [security policy](https://github.com/GhostofGoes/WOPR/security/policy) says, not in a public issue.

Search the [open issues](https://github.com/GhostofGoes/WOPR/issues) first: someone may have reported it
already.
