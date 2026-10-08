---
title: Installation
weight: 2
description: Install wopr on Windows, macOS or Linux with one line, remove it, or check that a download is genuine.
tabs:
  sync: true
---

## Install

Pick your system, copy the line, and paste it into a terminal.

{{< install-tabs >}}

`wopr` is one program with nothing else to install. It runs on Windows 11, macOS 26 Tahoe and Linux, on
amd64 (x86-64) or arm64 computers, in a terminal at least 80 columns wide and 24 rows tall. It never uses
the network and collects no data.

## Uninstall

`wopr` keeps no settings, so removing the program removes it all.

{{< tabs >}}

{{< tab name="Windows" >}}
In PowerShell, delete its folder and take the folder off your `PATH`:

```powershell
$d = "$env:LOCALAPPDATA\Programs\wopr"; Remove-Item -Recurse -Force $d; [Environment]::SetEnvironmentVariable('Path', ((([Environment]::GetEnvironmentVariable('Path', 'User')) -split ';' | Where-Object { $_ -and $_ -ne $d }) -join ';'), 'User')
```

{{< /tab >}}

{{< tab name="macOS" >}}

```sh
sudo rm /usr/local/bin/wopr
```

{{< /tab >}}

{{< tab name="Linux" >}}

```sh
sudo rm /usr/local/bin/wopr
```

{{< /tab >}}

{{< tab name="Linux (apt)" >}}

```sh
sudo apt remove wopr
```

{{< /tab >}}

{{< tab name="Linux (RPM)" >}}

```sh
sudo dnf remove wopr
```

On openSUSE: `sudo zypper remove wopr`.
{{< /tab >}}

{{< tab name="Go" >}}
Delete `wopr` (`wopr.exe` on Windows) from Go's `bin` folder, which `go env GOPATH` prints the parent of.
On Linux and macOS:

```sh
rm "$(go env GOPATH)/bin/wopr"
```

{{< /tab >}}

{{< /tabs >}}

If you ever ran it with `WOPR_DEBUG`, also delete its debug log;
[Troubleshooting](/usage/troubleshooting#debug-log) says where it is.

## Development builds

Every CI run on `main` builds all six targets. Open a
[CI run on main](https://github.com/GhostofGoes/WOPR/actions/workflows/ci.yml?query=branch%3Amain+event%3Apush)
and download your platform's file under **Artifacts**. It is the bare program: `chmod +x` it on Linux and
macOS. Development builds are for testing; they carry no attestation.

## Verifying binaries (attestation)

This is for people who want proof that a file came from this project before they trust it. You do not
need it to install `wopr`.

Every file in a [release](https://github.com/GhostofGoes/WOPR/releases) carries a build-provenance
attestation: a signed record, kept by GitHub, of the commit and the workflow that built it. To check one:

1. Install the [GitHub CLI](https://cli.github.com/), `gh`, and sign in once with `gh auth login`.
2. Run `gh attestation verify` on the file, with the tag of its version (`wopr --version` prints it):

   ```sh
   gh attestation verify FILE --repo GhostofGoes/WOPR \
     --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml \
     --source-ref refs/tags/v{{< version >}} --deny-self-hosted-runners
   ```

   `FILE` is a file you downloaded, such as `wopr_{{< version >}}_linux_amd64`, or the program you
   installed: `"$(command -v wopr)"` on Linux and macOS, `(Get-Command wopr).Source` in PowerShell. The
   check compares the file's contents, so its name does not matter.

`Verification succeeded!` means the file was built from this repository's tagged source by its release
workflow, on GitHub's own machines. Anything else means: do not run it.

{{< screenshot src="img/install-verify.png" caption="`gh attestation verify` on a Linux archive: the repository, the workflow and the tag all match." >}}

To check a file before you run it at all, download it by hand from the
[latest release](https://github.com/GhostofGoes/WOPR/releases/latest), verify it, and only then install
it. Each release has, for each system:

- `wopr_<version>_<os>_<arch>` (`.exe` on Windows): the program itself. `<os>` is `linux`, `darwin`
  (macOS) or `windows`; `<arch>` is `amd64` or `arm64`. On Linux and macOS, make it executable with
  `chmod +x`.
- `wopr_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows): the program with the README, the licence and
  the notices.{{% if-packages %}} On Linux and macOS it also holds the manual page, `wopr.6`.{{% /if-packages %}}
{{% if-packages %}}
- `wopr_<version>-1_<arch>.deb` and `wopr-<version>-1.<arch>.rpm`: the Linux packages. Install a file
  you downloaded with `sudo apt install ./FILE` or `sudo dnf install ./FILE`.
{{% /if-packages %}}
- `checksums.txt`: the SHA-256 checksum of every file.

A program downloaded with a web browser is marked as coming from the internet, and the system may stop it.
Once you have verified it:

- **macOS** says it cannot check the program for malware. Clear the mark with
  `xattr -d com.apple.quarantine FILE`.
- **Windows** SmartScreen may say it does not recognise the program. Choose *More info*, then
  *Run anyway*, or clear the mark with `Unblock-File FILE` in PowerShell.

The install commands above download with `curl`, `wget`, `dnf` or `apt`, which do not mark files, so this
does not come up.
