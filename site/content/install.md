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
Pick your system. Click the download button, and follow the steps under it. To install from a terminal
instead, or with Go, see [Command line install methods](#command-line-install-methods).
{{% /if-installers %}}
{{% if-installers "not" %}}
Pick your system. On Linux, click a download button and follow the steps under it. The downloads for
Windows and macOS come with the next release. Until then, and to install with Go, see
[Command line install methods](#command-line-install-methods).
{{% /if-installers %}}

{{< install-tabs >}}

`wopr` is one program, with nothing else to install. It runs on Windows 11, macOS 26 Tahoe and Linux, on
amd64 (x86-64) or arm64 computers, in a terminal at least 80 columns wide and 24 rows tall. It never uses
the network, and it collects no data.

## Uninstall

`wopr` keeps no settings. The only file it leaves behind is a
[debug log](/usage/troubleshooting#debug-log), if you ever made one. The steps below delete that too.

If you installed WOPR with a download button:

{{< tabs >}}

{{< tab name="Windows" >}}
{{< if-installers >}}

1. Open **Settings**, then **Apps**, then **Installed apps**.
2. Click **...** next to **WOPR**, then **Uninstall**.
3. Click **Uninstall** again, then **Yes**.

On Windows 10, open **Settings**, then **Apps**, then **Apps & features**. Click **WOPR**, then
**Uninstall**. This removes the Start menu entry and the debug log too. It also removes WOPR's folder
from your `PATH`, even if the PowerShell line put it there.
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
This also removes WOPR from your app menu.
{{< /if-installers >}}
{{< /tab >}}

{{< /tabs >}}

If you installed it from the [command line](#command-line-install-methods):

{{< tabs >}}

{{< tab name="Windows" >}}
In PowerShell, this line deletes WOPR's folder and its debug log, and removes the folder from your `PATH`:

```powershell
$d = "$env:LOCALAPPDATA\Programs\wopr"; Remove-Item -Recurse -Force $d -ErrorAction SilentlyContinue; $k = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment'); $p = $k.GetValue('Path', '', 'DoNotExpandEnvironmentNames'); $k.SetValue('Path', ((@($p -split ';') | Where-Object { $_ -and $_ -ne $d }) -join ';'), 'ExpandString'); $k.Close(); [Environment]::SetEnvironmentVariable('WOPR_PATH_REFRESH', $null, 'User'); Remove-Item -Recurse -Force "$env:LOCALAPPDATA\wopr" -ErrorAction SilentlyContinue
```

{{< /tab >}}

{{< tab name="macOS" >}}

```sh
rm -f ~/.local/bin/wopr; rm -rf ~/Library/Caches/wopr
```

If the install line added a `PATH` line to `~/.zprofile` (or `~/.bash_profile`), that line is harmless.
Delete it with a text editor if you like. An earlier version of this page put `wopr` in `/usr/local/bin`.
Remove that copy with `sudo rm -f /usr/local/bin/wopr`.
{{< /tab >}}

{{< tab name="Linux (any)" >}}

```sh
rm -f ~/.local/bin/wopr; rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/wopr"
```

If the install line added a `PATH` line to `~/.bashrc` (or `~/.zshrc`), that line is harmless. Delete it
with a text editor if you like. An earlier version of this page put `wopr` in `/usr/local/bin`. Remove
that copy with `sudo rm -f /usr/local/bin/wopr`.
{{< /tab >}}

{{< tab name="Linux (apt)" >}}

```sh
sudo apt remove wopr; rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/wopr"
```

{{< if-installers >}}
This also removes WOPR from your app menu.
{{< /if-installers >}}
{{< /tab >}}

{{< tab name="Linux (RPM)" >}}

```sh
sudo dnf remove wopr; rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/wopr"
```

On openSUSE, use `sudo zypper remove wopr` in place of `sudo dnf remove wopr`.{{< if-installers >}} This also removes WOPR from your app menu.{{< /if-installers >}}
{{< /tab >}}

{{< tab name="Go" >}}
Delete `wopr` from Go's `bin` folder, `~/go/bin` on Linux and macOS:

```sh
rm "$(go env GOPATH)/bin/wopr"
```

Then delete the debug log: `rm -rf ~/Library/Caches/wopr` on macOS, or
`rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/wopr"` on Linux.

On Windows, in PowerShell, this line deletes both:

```powershell
Remove-Item "$(go env GOPATH)\bin\wopr.exe"; Remove-Item -Recurse -Force "$env:LOCALAPPDATA\wopr" -ErrorAction SilentlyContinue
```

{{< /tab >}}

{{< /tabs >}}

## Command line install methods

Each tab has a line to paste into a terminal. The line downloads the latest release, installs it, and
starts WOPR. Go's line only installs it. Use these lines if you prefer a terminal, if you want the `wopr`
command on a Mac, or on a Linux that the download buttons do not cover.

{{< install-tabs "command-line" >}}

## Verifying binaries {#verifying-binaries-attestation}

This section is for people who want proof that a file came from this project before they trust it. You do
not need it to install `wopr`.

Every file in a [release](https://github.com/GhostofGoes/WOPR/releases) has a build-provenance
attestation: a signed record, kept by GitHub, of the commit and the workflow that built the file. To check
a file:

1. Install the [GitHub CLI](https://cli.github.com/), `gh`, version 2.68 or newer. Sign in once with
   `gh auth login`. A free GitHub account is enough. On Debian and Ubuntu, use the package repository that
   cli.github.com describes: the `gh` in the distributions' own repositories is too old.
2. Run `gh attestation verify` on the file. The command below names the latest release. For an older
   file, put in its version: the one in the file's name, or the one that `wopr --version` prints for the
   program you installed.

   ```sh
   gh attestation verify FILE --repo GhostofGoes/WOPR \
     --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml \
     --source-ref refs/tags/v{{< version >}} --deny-self-hosted-runners
   ```

   `FILE` is a file you downloaded, such as `wopr_{{< version >}}_linux_amd64`, or the program you
   installed: `"$(command -v wopr)"` on Linux and macOS, `(Get-Command wopr).Source` in PowerShell. The
   check compares the file's contents, so the file's name does not matter. A program installed with
   `go install` was built on your computer, not by the release workflow. It has no attestation, so this
   check fails for it. Go already checked its source against the Go checksum database.{{% if-installers %}}
   The program inside the Mac app does not pass the check either, because it is not one of the release's
   files (see the list below). Check the `.dmg` you downloaded instead.{{% /if-installers %}}

`Verification succeeded!` means that this repository's release workflow built the file from the tagged
source, on GitHub's own machines. For a file from a release, any other result means: do not run it.

{{< screenshot src="img/install-verify.png" caption="`gh attestation verify` on a Linux archive: the repository, the workflow and the tag all match." >}}

To check a file before you run it at all, download it by hand from the
[latest release](https://github.com/GhostofGoes/WOPR/releases/latest), check it, and only then install
it. Each release has, for each system:

- `wopr_<version>_<os>_<arch>` (`.exe` on Windows): the program itself. `<os>` is `linux`, `darwin`
  (macOS) or `windows`. `<arch>` is `amd64` or `arm64`. On Linux and macOS, make it executable with
  `chmod +x`.
- `wopr_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows): the program with the README, the licence and
  the notices.{{% if-packages %}} On Linux and macOS it also holds the manual page, `wopr.6`.{{% /if-packages %}}
{{% if-installers %}}
- `wopr_<version>_windows_setup.exe`: the Windows installer. It holds both Windows programs, and it
  installs the one that fits the computer as `wopr.exe`, unchanged. So the check passes for the installed
  `wopr.exe` too.
- `wopr_<version>_macos.dmg`: the Mac app, `WOPR.app`, in a disk image. Its program joins the two macOS
  programs into one file, so it matches neither of them. Check the `.dmg` itself, before you open it.
{{% /if-installers %}}
{{% if-packages %}}
- `wopr_<version>-1_<arch>.deb` (`<arch>` as above: `amd64` or `arm64`) and `wopr-<version>-1.<arch>.rpm`
  (here `<arch>` is `x86_64` or `aarch64`, as `uname -m` prints it): the Linux packages. Install a file
  you downloaded with `sudo apt install ./FILE` or `sudo dnf install ./FILE`.
{{% /if-packages %}}
- `checksums.txt`: the SHA-256 checksum of every file.{{% if-installers %}} The installer and the `.dmg` are
  listed and attested like the rest.{{% /if-installers %}}
{{% if-signed %}}
- `checksums.txt.asc` and `wopr_<version>-1_<arch>.deb.asc`: OpenPGP signatures of `checksums.txt` and
  the `.deb` files, made with wopr's [signing key](/wopr-signing-key.asc). The `.rpm` files carry their
  signature inside.
{{% /if-signed %}}

{{% if-signed %}}
To check a signature:

1. Download the [signing key](/wopr-signing-key.asc), and check that its fingerprint is
   `{{< signing-fingerprint >}}`.
2. Run `gpg --import wopr-signing-key.asc`.
3. Run `gpg --verify checksums.txt.asc checksums.txt` (or `gpg --verify FILE.deb.asc FILE.deb`).
   `Good signature` means that the file is as the release workflow signed it.
4. Run `sha256sum --ignore-missing -c checksums.txt`. It checks the files you downloaded against a list
   that you now know is genuine.

For an `.rpm`, import the key with `rpm --import`, then run `rpm -K FILE.rpm`. It prints
`digests signatures OK`.
{{% /if-signed %}}

A program downloaded with a web browser carries a mark that says it came from the internet, and the
system may stop it. After you check the file:

- **macOS** says that it cannot check the program for malware. Clear the mark with
  `xattr -d com.apple.quarantine FILE`.{{% if-installers %}} For the Mac app, follow the steps in the
  **macOS** tab under [Install](#install). Or drag the app to Applications, and clear the mark from all
  of it with `xattr -dr com.apple.quarantine /Applications/WOPR.app` before you open it.{{% /if-installers %}}
- **Windows** SmartScreen may say that it does not recognise the program. Choose *More info*, then
  *Run anyway*. Or clear the mark with `Unblock-File FILE` in PowerShell.

The lines in [Command line install methods](#command-line-install-methods) download with `curl`, `wget`
or `dnf`, which do not mark files, so these warnings do not appear for them.{{% if-installers %}} For the
installer and the Mac app, the **Windows** and **macOS** tabs under [Install](#install) say what to
click.{{% /if-installers %}} Windows' Smart App Control is different: when it is on, it blocks every
program that is not signed, however the program was downloaded. [Troubleshooting](/usage/troubleshooting)
says more.

## Development builds

Every CI run on `main` builds all six targets. Open a
[CI run on main](https://github.com/GhostofGoes/WOPR/actions/workflows/ci.yml?query=branch%3Amain+event%3Apush)
and download your platform's file under **Artifacts**. It is the bare program: on Linux and macOS, make
it executable with `chmod +x`. The run also has the Windows installer, `installer-windows`, and the Mac
app, `installer-macos`. Development builds are for testing. They carry no attestation.
