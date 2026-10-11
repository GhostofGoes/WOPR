---
title: Troubleshooting
weight: 2
description: Fixes for common problems, the debug log, and where to ask for help or report a bug.
next: /movie
---

## Common problems

**"standard output is not a terminal".** Run `wopr` directly in a terminal, not through a pipe or a
redirect. On Windows, use Windows Terminal. mintty without ConPTY is not supported.

**"TERM=dumb".** The terminal says that it cannot draw a full screen. Use another terminal. `wopr --games`
and the other printing options still work.

**"TERMINAL TOO SMALL".** WOPR needs at least 80 columns and 24 rows. Make the window bigger. The session
carries on where it was.

**`wopr` is not found.** Open a new terminal window: the one you installed from may not know about the new
program yet. The [install lines](/install#command-line-install-methods) put the program in
`%LOCALAPPDATA%\Programs\wopr` on Windows, and in `~/.local/bin` on macOS and Linux. When that folder is
not on your `PATH`, the lines add it: in `~/.zprofile` or `~/.bash_profile` on macOS, and in `~/.bashrc`
or `~/.zshrc` on Linux. `go install` puts the program in Go's `bin` folder.{{% if-packages %}} The `.deb`
puts it in `/usr/games`, which is not on root's `PATH`. As root, or in a small container, run
`/usr/games/wopr`. The `.rpm` puts it in `/usr/bin`.{{% /if-packages %}}
{{% if-installers %}} The Windows installer uses the same folder as the install line. It puts that folder
on your `PATH` only if its **Add WOPR to PATH** box was ticked. Run the installer again to tick it. The
Mac app adds no `wopr` command. Open WOPR from Applications, or install the command with the Terminal
line in [Command line install methods](/install#command-line-install-methods).{{% /if-installers %}}

**Windows protected your PC.** Microsoft Defender SmartScreen says this about a program from the
internet that it does not recognise, such as `wopr.exe`{{% if-installers %}} or its installer{{% /if-installers %}}
downloaded with a web browser. Click **More info**. It names the file, and it names the publisher as
**Unknown publisher**, because `wopr` is not signed yet. Then click **Run anyway**.

**Windows says Smart App Control blocked `wopr`{{% if-installers %}} or its installer{{% /if-installers %}}.**
When Smart App Control is on, it blocks every program that is not signed. `wopr` is not signed
yet{{% if-installers %}}, and neither is its installer{{% /if-installers %}}. Smart App Control has no
exception for one program. The only way to run `wopr` is to turn Smart App Control off: open Windows
Security, then **App & browser control**, then **Smart App Control settings**. Recent Windows updates let
you turn it on again later. Before them, only reinstalling Windows could. Most PCs have it off, or in
evaluation mode, which blocks nothing.

{{% if-installers %}}
**macOS says "WOPR" Not Opened, or that Apple could not verify WOPR.** macOS says this about an app that
Apple did not check, and Apple has not signed WOPR yet. Click **Done**. Then open **System Settings**,
then **Privacy & Security**. Scroll down to **Security**, and click **Open Anyway** beside the line about
WOPR. The button stays there for about an hour after you tried to open WOPR. If it is gone, open WOPR
again first. macOS asks once more: click **Open Anyway** (or **Open**), and enter your password.
Since macOS 15 Sequoia, Control-clicking WOPR and choosing **Open** no longer works. If macOS then blocks `wopr` again when
Terminal starts it, allow that the same way.

**macOS says WOPR "is damaged and can't be opened".** macOS also says this about some apps that Apple did
not sign. Check the `.dmg` you downloaded, as
[Verifying binaries](/install#verifying-binaries-attestation) says, or download it again. Then drag WOPR
to Applications, and clear its download mark in Terminal:
`xattr -dr com.apple.quarantine /Applications/WOPR.app`. That turns off macOS's malware check for WOPR
alone, so do it only for a file you checked. Please [report it](#getting-help) too, with your macOS
version.

**On a Mac, opening WOPR does nothing, or Terminal shows an error.** Drag WOPR to Applications in Finder,
and open it from there. When a downloaded app was not moved with Finder, macOS runs it from a hidden
copy, which can disappear while WOPR hands itself to Terminal.

**WOPR always opens in Terminal on a Mac.** The app opens Apple's Terminal, whichever terminal you use.
To play in another terminal, such as iTerm2, run `/Applications/WOPR.app/Contents/MacOS/wopr` in it. Or
install the `wopr` command with the Terminal line in
[Command line install methods](/install#command-line-install-methods).
{{% /if-installers %}}
{{% if-installers %}}
**WOPR is not in the app menu after installing the `.deb` or the `.rpm`.** Log out and back in. Some
desktops notice a new app only then. The **Linux (any)** line adds no menu entry. Only the packages do.
{{% /if-installers %}}

**`sudo` says you are not in the sudoers file.** On Debian, when a root password was set during
installation, your account cannot use `sudo` until it is in the `sudo` group. Add it with the root
password: `su -c "adduser $USER sudo"`. Then log out and back in, and paste the install line again.

**An old version still starts after you update.** Another copy comes first on your `PATH`. On Linux and
macOS, `type -a wopr` lists every copy. In PowerShell, `Get-Command -All wopr` does. Delete the copies
you do not want. An earlier version of these pages put `wopr` in `/usr/local/bin`, and
`sudo rm /usr/local/bin/wopr` deletes that copy.

**Strange colours.** Try another theme (`--theme green`), or turn colour off with `NO_COLOR=1`.

**The mouse wheel changes what I typed.** Some terminals (GNOME Terminal and others built on VTE) turn the
wheel into the Up and Down keys on a full screen. Those keys step through the lines you typed. Scroll
back with PgUp and PgDn. `wopr` leaves mouse reporting off, so that you can still select and copy text.

**Text types too slowly.** Press any key to show the rest at once, or run with `--instant`.

**Something else.** Run with `WOPR_DEBUG=1`. Then [report it](#getting-help) with the
[debug log](#debug-log).

## Debug log

If something goes wrong, run `wopr` with `WOPR_DEBUG` set to `1`:

```sh
WOPR_DEBUG=1 wopr                                          # Linux and macOS
```

```powershell
$env:WOPR_DEBUG = 1; wopr; Remove-Item Env:WOPR_DEBUG     # Windows PowerShell
```

{{% if-installers %}}
With the Mac app, which adds no `wopr` command, run this in Terminal:
`WOPR_DEBUG=1 /Applications/WOPR.app/Contents/MacOS/wopr`.
{{% /if-installers %}}

`wopr` then writes a debug log, and it prints the log's path when it exits. The log records the session's
seed, the terminal's size and the slow work. It never records what you type. `wopr` keeps the log in your
user cache folder:

| System | Debug log |
|---|---|
| Linux | `~/.cache/wopr/debug.log` (or `$XDG_CACHE_HOME/wopr/debug.log`) |
| macOS | `~/Library/Caches/wopr/debug.log` |
| Windows | `%LOCALAPPDATA%\wopr\debug.log` |

Each session adds to the end of the log, starting with a line that names the version and the seed. So the
newest session is at the end. If `wopr` stops with a message before its screen appears, it writes no log.
The message is what to report.

## Getting help

- **Questions:** ask in [GitHub Discussions](https://github.com/GhostofGoes/WOPR/discussions/categories/q-a).
- **Something is broken:** open a
  [bug report](https://github.com/GhostofGoes/WOPR/issues/new?template=bug_report.yml). Say what you did,
  what happened, and what `wopr --version` prints (or that it would not install). Attach the
  [debug log](#debug-log), with the seed if you used one.
- **A new feature, or something that differs from the film:** open a
  [feature request](https://github.com/GhostofGoes/WOPR/issues/new?template=feature_request.yml), or
  discuss the idea first in [Ideas](https://github.com/GhostofGoes/WOPR/discussions/categories/ideas).
- **A security problem:** report it privately, as the
  [security policy](https://github.com/GhostofGoes/WOPR/security/policy) says, not in a public issue.

Search the [issues](https://github.com/GhostofGoes/WOPR/issues?q=is%3Aissue) first: someone may have
reported it already.
