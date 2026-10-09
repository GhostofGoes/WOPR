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
Pick your system. On Windows and macOS, download WOPR and open it as the tab says, or copy the line under
it into a terminal. On Linux, and with Go, copy the line and paste it into a terminal.
{{% /if-installers %}}
{{% if-installers "not" %}}
Pick your system, copy the line, and paste it into a terminal.
{{% /if-installers %}}

{{< install-tabs >}}

`wopr` is one program with nothing else to install. It runs on Windows 11, macOS 26 Tahoe and Linux, on
amd64 (x86-64) or arm64 computers, in a terminal at least 80 columns wide and 24 rows tall. It never uses
the network and collects no data.

## Uninstall

`wopr` keeps no settings: removing the program removes everything except a debug log, if you ever made one
([Troubleshooting](/usage/troubleshooting#debug-log) says where it is; delete its `wopr` folder).

{{< tabs >}}

{{< tab name="Windows" >}}
{{< if-installers >}}
If you used the installer, open **Settings**, then **Apps**, then **Installed apps**. Click **...** next
to **WOPR**, then **Uninstall**, **Uninstall** again, and **Yes**. On Windows 10, it is **Settings**, then
**Apps**, then **Apps & features**, then **WOPR**, then **Uninstall**. This removes the Start menu entry
too, and takes WOPR's folder off your `PATH`, even if the PowerShell line put it there.

If you used the PowerShell line, delete its folder and take the folder off your `PATH`, in PowerShell:
{{< /if-installers >}}
{{< if-installers "not" >}}
In PowerShell, delete its folder and take the folder off your `PATH`:
{{< /if-installers >}}

```powershell
$d = "$env:LOCALAPPDATA\Programs\wopr"; Remove-Item -Recurse -Force $d -ErrorAction SilentlyContinue; $k = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment'); $p = $k.GetValue('Path', '', 'DoNotExpandEnvironmentNames'); $k.SetValue('Path', ((@($p -split ';') | Where-Object { $_ -and $_ -ne $d }) -join ';'), 'ExpandString'); $k.Close(); [Environment]::SetEnvironmentVariable('WOPR_PATH_REFRESH', $null, 'User')
```

{{< /tab >}}

{{< tab name="macOS" >}}
{{< if-installers >}}
If you installed the app, drag **WOPR** from **Applications** to the Trash, and empty the Trash.

If you used the Terminal line:
{{< /if-installers >}}

```sh
sudo rm /usr/local/bin/wopr
```

{{< /tab >}}

{{< tab name="Linux" >}}

```sh
sudo rm -f /usr/local/bin/wopr ~/.local/bin/wopr
```

An earlier version of this page put `wopr` in `~/.local/bin`; this removes that copy too.
{{< /tab >}}

{{< tab name="Linux (apt)" >}}
{{< if-packages >}}

```sh
sudo apt remove wopr
```

This also takes WOPR out of your app menu.
{{< /if-packages >}}
{{< if-packages "not" >}}
There is no package yet; see the **Linux** tab.
{{< /if-packages >}}
{{< /tab >}}

{{< tab name="Linux (RPM)" >}}
{{< if-packages >}}

```sh
sudo dnf remove wopr
```

On openSUSE: `sudo zypper remove wopr`. This also takes WOPR out of your app menu.
{{< /if-packages >}}
{{< if-packages "not" >}}
There is no package yet; see the **Linux** tab.
{{< /if-packages >}}
{{< /tab >}}

{{< tab name="Go" >}}
Delete `wopr` from Go's `bin` folder, `~/go/bin` on Linux and macOS:

```sh
rm "$(go env GOPATH)/bin/wopr"
```

On Windows, in PowerShell:

```powershell
Remove-Item "$(go env GOPATH)\bin\wopr.exe"
```

{{< /tab >}}

{{< /tabs >}}

## Development builds

Every CI run on `main` builds all six targets. Open a
[CI run on main](https://github.com/GhostofGoes/WOPR/actions/workflows/ci.yml?query=branch%3Amain+event%3Apush)
and download your platform's file under **Artifacts**. It is the bare program: `chmod +x` it on Linux and
macOS. The run also has the Windows installer, `installer-windows`, and the Mac app, `installer-macos`.
Development builds are for testing; they carry no attestation.

## Verifying binaries (attestation)

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

A program downloaded with a web browser is marked as coming from the internet, and the system may stop it.
Once you have verified it:

- **macOS** says it cannot check the program for malware. Clear the mark with
  `xattr -d com.apple.quarantine FILE`.{{% if-installers %}} For the Mac app, follow the steps in the
  **macOS** tab, or drag it to Applications and clear the mark from all of it with
  `xattr -dr com.apple.quarantine /Applications/WOPR.app` before you open it.{{% /if-installers %}}
- **Windows** SmartScreen may say it does not recognise the program. Choose *More info*, then
  *Run anyway*, or clear the mark with `Unblock-File FILE` in PowerShell.

The install lines above download with `curl`, `wget` or `dnf`, which do not mark files, so this does not
come up for them{{% if-installers %}}; for the installer and the Mac app, the **Windows** and **macOS** tabs
say what to click{{% /if-installers %}}. Windows' Smart App Control is different: when it is on, it blocks every program that is not
signed, however it was downloaded ([Troubleshooting](/usage/troubleshooting) says more).
