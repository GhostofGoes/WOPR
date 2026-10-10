---
title: Installation
weight: 2
description: Install wopr on Windows, macOS or Linux, remove it, or check that a download is genuine.
tabs:
  sync: true
next: /usage
---

## Install

{{% if-installers %}}
Pick your system, click the download button, and follow the steps under it. To install from a terminal
instead, or with Go, see [Command line install methods](#command-line-install-methods).
{{% /if-installers %}}
{{% if-installers "not" %}}
Pick your system. On Linux, click a download button and follow the steps under it. The downloads for
Windows and macOS come with the next release; until then, and to install with Go, see
[Command line install methods](#command-line-install-methods).
{{% /if-installers %}}

{{< install-tabs >}}

`wopr` is one program with nothing else to install. It runs on Windows 11, macOS 26 Tahoe and Linux, on
amd64 (x86-64) or arm64 computers, in a terminal at least 80 columns wide and 24 rows tall. It never uses
the network and collects no data.

## Uninstall

`wopr` keeps no settings. The only file it leaves behind is a
[debug log](/usage/troubleshooting#debug-log), if you ever made one, and the steps below delete it too.

If you installed WOPR with a download button:

{{< tabs >}}

{{< tab name="Windows" >}}
{{< if-installers >}}
Open **Settings**, then **Apps**, then **Installed apps**. Click **...** next to **WOPR**, then
**Uninstall**, **Uninstall** again, and **Yes**. On Windows 10, it is **Settings**, then **Apps**, then
**Apps & features**, then **WOPR**, then **Uninstall**. This removes the Start menu entry and the debug
log too, and takes WOPR's folder off your `PATH`, even if the PowerShell line put it there.
{{< /if-installers >}}
{{< if-installers "not" >}}
The Windows installer comes with the next release. Until then, WOPR installs from PowerShell: see below.
{{< /if-installers >}}
{{< /tab >}}

{{< tab name="macOS" >}}
{{< if-installers >}}
Drag **WOPR** from **Applications** to the Trash, and empty the Trash. Then delete the debug log, if
there is one, in Terminal:

```sh
rm -rf ~/Library/Caches/wopr
```

{{< /if-installers >}}
{{< if-installers "not" >}}
The Mac app comes with the next release. Until then, WOPR installs from Terminal: see below.
{{< /if-installers >}}
{{< /tab >}}

{{< tab name="Linux" >}}
Open a terminal, and remove the package and the debug log. For the `.deb`:

```sh
sudo apt remove wopr; rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/wopr"
```

For the `.rpm`:

```sh
sudo dnf remove wopr; rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/wopr"
```

{{< if-installers >}}
This also takes WOPR out of your app menu.
{{< /if-installers >}}
{{< /tab >}}

{{< /tabs >}}

If you installed it from the [command line](#command-line-install-methods):

{{< tabs >}}

{{< tab name="Windows" >}}
In PowerShell, delete its folder and its debug log, and take the folder off your `PATH`:

```powershell
$d = "$env:LOCALAPPDATA\Programs\wopr"; Remove-Item -Recurse -Force $d -ErrorAction SilentlyContinue; $k = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment'); $p = $k.GetValue('Path', '', 'DoNotExpandEnvironmentNames'); $k.SetValue('Path', ((@($p -split ';') | Where-Object { $_ -and $_ -ne $d }) -join ';'), 'ExpandString'); $k.Close(); [Environment]::SetEnvironmentVariable('WOPR_PATH_REFRESH', $null, 'User'); Remove-Item -Recurse -Force "$env:LOCALAPPDATA\wopr" -ErrorAction SilentlyContinue
```

{{< /tab >}}

{{< tab name="macOS" >}}

```sh
rm -f ~/.local/bin/wopr; rm -rf ~/Library/Caches/wopr
```

The `PATH` line the install line added to `~/.zprofile` (or `~/.bash_profile`) is harmless; delete it with
a text editor if you like. An earlier version of this page put `wopr` in `/usr/local/bin`; remove that
copy with `sudo rm -f /usr/local/bin/wopr`.
{{< /tab >}}

{{< tab name="Linux (any)" >}}

```sh
rm -f ~/.local/bin/wopr; rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/wopr"
```

The `PATH` line the install line may have added to `~/.bashrc` (or `~/.zshrc`) is harmless; delete it with a
text editor if you like. An earlier version of this page put `wopr` in `/usr/local/bin`; remove that copy
with `sudo rm -f /usr/local/bin/wopr`.
{{< /tab >}}

{{< tab name="Linux (apt)" >}}

```sh
sudo apt remove wopr; rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/wopr"
```

{{< if-installers >}}
This also takes WOPR out of your app menu.
{{< /if-installers >}}
{{< /tab >}}

{{< tab name="Linux (RPM)" >}}

```sh
sudo dnf remove wopr; rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/wopr"
```

On openSUSE, use `sudo zypper remove wopr` in place of `sudo dnf remove wopr`.{{< if-installers >}} This also takes WOPR out of your app menu.{{< /if-installers >}}
{{< /tab >}}

{{< tab name="Go" >}}
Delete `wopr` from Go's `bin` folder, `~/go/bin` on Linux and macOS:

```sh
rm "$(go env GOPATH)/bin/wopr"
```

Then delete the debug log: `rm -rf ~/Library/Caches/wopr` on macOS, or
`rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/wopr"` on Linux.

On Windows, in PowerShell, this deletes both:

```powershell
Remove-Item "$(go env GOPATH)\bin\wopr.exe"; Remove-Item -Recurse -Force "$env:LOCALAPPDATA\wopr" -ErrorAction SilentlyContinue
```

{{< /tab >}}

{{< /tabs >}}

## Command line install methods

Each tab has a line to paste into a terminal. It downloads the latest release, installs it, and starts
WOPR; Go's line only installs it. Use these if you would rather use a terminal, to get the `wopr` command
on a Mac, or on a Linux that the download buttons do not cover.

{{< install-tabs "command-line" >}}

## Verifying binaries {#verifying-binaries-attestation}

This is for people who want proof that a file came from this project before they trust it. You do not
need it to install `wopr`.

Every file in a [release](https://github.com/GhostofGoes/WOPR/releases) carries a build-provenance
attestation: a signed record, kept by GitHub, of the commit and the workflow that built it. To check one:

1. Install the [GitHub CLI](https://cli.github.com/), `gh`, version 2.68 or newer, and sign in once with
   `gh auth login` (a free GitHub account is enough). On Debian and Ubuntu, use the package repository
   that cli.github.com describes: the `gh` in the distributions' own repositories is too old.
2. Run `gh attestation verify` on the file. The command below names the latest release; for an older file,
   put in its version: the one in the file's name, or what `wopr --version` prints for the program you
   installed.

   ```sh
   gh attestation verify FILE --repo GhostofGoes/WOPR \
     --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml \
     --source-ref refs/tags/v{{< version >}} --deny-self-hosted-runners
   ```

   `FILE` is a file you downloaded, such as `wopr_{{< version >}}_linux_amd64`, or the program you
   installed: `"$(command -v wopr)"` on Linux and macOS, `(Get-Command wopr).Source` in PowerShell. The
   check compares the file's contents, so its name does not matter. A program installed with
   `go install` was built on your computer, not by the release workflow, so it has no attestation and this
   check fails; Go has already checked its source against the Go checksum database.{{% if-installers %}}
   Nor does the program inside the Mac app pass it, since it is not one of the release's files (see the
   list below): check the `.dmg` you downloaded instead.{{% /if-installers %}}

`Verification succeeded!` means the file was built from this repository's tagged source by its release
workflow, on GitHub's own machines. For a file from a release, anything else means: do not run it.

{{< screenshot src="img/install-verify.png" caption="`gh attestation verify` on a Linux archive: the repository, the workflow and the tag all match." >}}

To check a file before you run it at all, download it by hand from the
[latest release](https://github.com/GhostofGoes/WOPR/releases/latest), verify it, and only then install
it. Each release has, for each system:

- `wopr_<version>_<os>_<arch>` (`.exe` on Windows): the program itself. `<os>` is `linux`, `darwin`
  (macOS) or `windows`; `<arch>` is `amd64` or `arm64`. On Linux and macOS, make it executable with
  `chmod +x`.
- `wopr_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows): the program with the README, the licence and
  the notices.{{% if-packages %}} On Linux and macOS it also holds the manual page, `wopr.6`.{{% /if-packages %}}
{{% if-installers %}}
- `wopr_<version>_windows_setup.exe`: the Windows installer. It holds both Windows programs and installs
  the one that fits the computer as `wopr.exe`, unchanged, so the check passes for the installed
  `wopr.exe` too.
- `wopr_<version>_macos.dmg`: the Mac app, `WOPR.app`, in a disk image. Its program joins the two macOS
  programs into one file, so it matches neither of them: check the `.dmg` itself, before you open it.
{{% /if-installers %}}
{{% if-packages %}}
- `wopr_<version>-1_<arch>.deb` (`<arch>` as above: `amd64` or `arm64`) and `wopr-<version>-1.<arch>.rpm`
  (here `<arch>` is `x86_64` or `aarch64`, as `uname -m` prints): the Linux packages. Install a file you
  downloaded with `sudo apt install ./FILE` or `sudo dnf install ./FILE`.
{{% /if-packages %}}
- `checksums.txt`: the SHA-256 checksum of every file.{{% if-installers %}} The installer and the `.dmg` are
  listed and attested like the rest.{{% /if-installers %}}
{{% if-signed %}}
- `checksums.txt.asc` and `wopr_<version>-1_<arch>.deb.asc`: OpenPGP signatures of `checksums.txt` and
  the `.deb` files, made with wopr's [signing key](/wopr-signing-key.asc). The `.rpm` files carry their
  signature inside.
{{% /if-signed %}}

{{% if-signed %}}
To check a signature, download the [signing key](/wopr-signing-key.asc) and check that its fingerprint
is `{{< signing-fingerprint >}}`, then run `gpg --import wopr-signing-key.asc` and
`gpg --verify checksums.txt.asc checksums.txt` (or `FILE.deb.asc FILE.deb`). `Good signature` means the
file is as the release workflow signed it. `sha256sum --ignore-missing -c checksums.txt` then checks the
files you downloaded against a list you know is genuine. For an `.rpm`, `rpm --import` the key and run
`rpm -K FILE.rpm`, which prints `digests signatures OK`.
{{% /if-signed %}}

A program downloaded with a web browser is marked as coming from the internet, and the system may stop it.
Once you have verified it:

- **macOS** says it cannot check the program for malware. Clear the mark with
  `xattr -d com.apple.quarantine FILE`.{{% if-installers %}} For the Mac app, follow the steps in the
  **macOS** tab under [Install](#install), or drag it to Applications and clear the mark from all of it
  with `xattr -dr com.apple.quarantine /Applications/WOPR.app` before you open it.{{% /if-installers %}}
- **Windows** SmartScreen may say it does not recognise the program. Choose *More info*, then
  *Run anyway*, or clear the mark with `Unblock-File FILE` in PowerShell.

The lines in [Command line install methods](#command-line-install-methods) download with `curl`, `wget`
or `dnf`, which do not mark files, so this does not come up for them{{% if-installers %}}; for the
installer and the Mac app, the **Windows** and **macOS** tabs under [Install](#install) say what to
click{{% /if-installers %}}. Windows' Smart App Control is different: when it is on, it blocks every
program that is not signed, however it was downloaded ([Troubleshooting](/usage/troubleshooting) says
more).

## Development builds

Every CI run on `main` builds all six targets. Open a
[CI run on main](https://github.com/GhostofGoes/WOPR/actions/workflows/ci.yml?query=branch%3Amain+event%3Apush)
and download your platform's file under **Artifacts**. It is the bare program: `chmod +x` it on Linux and
macOS. The run also has the Windows installer, `installer-windows`, and the Mac app, `installer-macos`.
Development builds are for testing; they carry no attestation.
