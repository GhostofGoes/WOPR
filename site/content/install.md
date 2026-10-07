---
title: Installation
weight: 2
description: Download wopr for Linux, macOS or Windows, verify it, and put it on your PATH. Or build it from source.
tabs:
  sync: true
---

`wopr` is one program with nothing else to install. It runs on:

- **Linux**, any distribution, on amd64 (x86-64) or arm64.
- **macOS** 26 Tahoe, on Apple silicon (arm64) or Intel (amd64).
- **Windows** 11, on amd64 or arm64, in Windows Terminal.

It needs a terminal at least 80 columns wide and 24 rows tall. It never uses the network and collects no
data.

## What to download

Every [GitHub Release](https://github.com/GhostofGoes/WOPR/releases) has these files for each platform.
The latest release is **v{{< version >}}**.

| File | What it is |
|---|---|
| `wopr_<version>_<os>_<arch>` (`.exe` on Windows) | The program itself, ready to run. |
| `wopr_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows) | An archive of the program with the README, the licence and the notices.{{% if-packages %}} On Linux and macOS it also holds the manual page, `wopr.6`.{{% /if-packages %}} |
| `checksums.txt` | The SHA-256 checksum of every file. |

`<os>` is `linux`, `darwin` (macOS) or `windows`, and `<arch>` is `amd64` or `arm64`. Not sure which you
have? Run `uname -m` on Linux or macOS (`x86_64` is amd64; `aarch64` or `arm64` is arm64), or
`$env:PROCESSOR_ARCHITECTURE` in PowerShell.
{{% if-packages %}}

Linux also has [packages](#linux-packages): `wopr_<version>-1_<arch>.deb` for Debian, Ubuntu and their
relatives (`<arch>` is `amd64` or `arm64`), and `wopr-<version>-1.<arch>.rpm` for Fedora, RHEL and their
relatives (`<arch>` is `x86_64` or `aarch64`).
{{% /if-packages %}}
{{% if-packages "not" %}}

The next release adds Linux packages: a `.deb` for Debian and Ubuntu, and an `.rpm` for Fedora and
RHEL.
{{% /if-packages %}}

## Verify before you run

Every file in a release carries a build-provenance attestation: a signed record of the commit and the
GitHub workflow that built it. Check it with the [GitHub CLI](https://cli.github.com/) before you run
anything you downloaded. Log in first with `gh auth login` if you have not.

```sh
gh attestation verify wopr_{{< version >}}_linux_amd64 --repo GhostofGoes/WOPR \
  --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml \
  --source-ref refs/tags/v{{< version >}} --deny-self-hosted-runners
```

Use the name of the file you downloaded. `Verification succeeded!` means the file was built from this
repository's tagged source by its release workflow, on GitHub's own machines. Anything else means: do not
run it.

{{< screenshot src="img/install-verify.png" caption="`gh attestation verify` on a Linux archive: the repository, the workflow and the tag all match." >}}

## Install

{{% if-packages %}}

On Debian, Ubuntu, Fedora or RHEL, the [Linux packages](#linux-packages) are easier, and they install
the manual page too.
{{% /if-packages %}}

{{< tabs >}}

{{< tab name="Linux" >}}
Download, verify, and install the program as `wopr` in `~/.local/bin` (use `arm64` on an ARM computer):

```sh
VERSION={{< version >}}
curl -LO "https://github.com/GhostofGoes/WOPR/releases/download/v${VERSION}/wopr_${VERSION}_linux_amd64"
gh attestation verify "wopr_${VERSION}_linux_amd64" --repo GhostofGoes/WOPR \
  --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml \
  --source-ref "refs/tags/v${VERSION}" --deny-self-hosted-runners
mkdir -p ~/.local/bin
install -m 755 "wopr_${VERSION}_linux_amd64" ~/.local/bin/wopr
wopr --version
```

Most distributions put `~/.local/bin` on your `PATH` once it exists; you may need to open a new terminal.
If `wopr` is still not found, add `export PATH="$HOME/.local/bin:$PATH"` to your `~/.bashrc` or
`~/.zshrc`. To install it for every user instead, use `sudo install -m 755 ... /usr/local/bin/wopr`.

A downloaded program is not executable until `install` or `chmod +x` makes it so. The archive keeps the
executable bit: `tar xzf wopr_${VERSION}_linux_amd64.tar.gz` and run `./wopr`.
{{< /tab >}}

{{< tab name="macOS" >}}
Download, verify, and install the program as `wopr` in `/usr/local/bin` (use `amd64` on an Intel Mac):

```sh
VERSION={{< version >}}
curl -LO "https://github.com/GhostofGoes/WOPR/releases/download/v${VERSION}/wopr_${VERSION}_darwin_arm64"
gh attestation verify "wopr_${VERSION}_darwin_arm64" --repo GhostofGoes/WOPR \
  --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml \
  --source-ref "refs/tags/v${VERSION}" --deny-self-hosted-runners
sudo mkdir -p /usr/local/bin
sudo install -m 755 "wopr_${VERSION}_darwin_arm64" /usr/local/bin/wopr
wopr --version
```

**Gatekeeper.** The program is not signed by Apple. A file you download with a web browser is
quarantined, and macOS refuses to run it. After you have verified it, clear the quarantine:

```sh
xattr -d com.apple.quarantine /usr/local/bin/wopr
```

Files downloaded with `curl`, as above, are not quarantined.
{{< /tab >}}

{{< tab name="Windows" >}}
In PowerShell, download, verify, and install the program as `wopr.exe` in a folder of your own (use
`arm64` on an ARM computer):

```powershell
$VERSION = "{{< version >}}"
curl.exe -LO "https://github.com/GhostofGoes/WOPR/releases/download/v$VERSION/wopr_${VERSION}_windows_amd64.exe"
gh attestation verify "wopr_${VERSION}_windows_amd64.exe" --repo GhostofGoes/WOPR `
  --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml `
  --source-ref "refs/tags/v$VERSION" --deny-self-hosted-runners
$dir = "$env:LOCALAPPDATA\Programs\wopr"
New-Item -ItemType Directory -Force $dir | Out-Null
Move-Item "wopr_${VERSION}_windows_amd64.exe" "$dir\wopr.exe"
```

Then add that folder to your `PATH`, once, and open a new terminal:

```powershell
[Environment]::SetEnvironmentVariable("Path", [Environment]::GetEnvironmentVariable("Path", "User") + ";$env:LOCALAPPDATA\Programs\wopr", "User")
```

Run `wopr` in **Windows Terminal**. mintty (Git Bash's default terminal) without ConPTY is not supported.

**SmartScreen.** The program is not signed, so Windows may warn that it does not recognise it. After you
have verified the file, choose *More info*, then *Run anyway*. A file downloaded with a web browser can
also be cleared with `Unblock-File "$env:LOCALAPPDATA\Programs\wopr\wopr.exe"`.
{{< /tab >}}

{{< /tabs >}}
{{% if-packages %}}

## Linux packages

Each release has a package for Debian, Ubuntu and their relatives (`.deb`), and one for Fedora, RHEL
and their relatives (`.rpm`), for amd64 and arm64 computers. A package puts `wopr` on your `PATH` and
installs the manual page, so `man wopr` works. It needs no other package, and removing it removes every
file it added. It also holds the changelog, the licences and the notices.

### Debian and Ubuntu

Download, verify, and install the `.deb` with `apt`:

```sh
VERSION={{< version >}}
ARCH=$(dpkg --print-architecture)   # amd64 or arm64
curl -LO "https://github.com/GhostofGoes/WOPR/releases/download/v${VERSION}/wopr_${VERSION}-1_${ARCH}.deb"
gh attestation verify "wopr_${VERSION}-1_${ARCH}.deb" --repo GhostofGoes/WOPR \
  --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml \
  --source-ref "refs/tags/v${VERSION}" --deny-self-hosted-runners
sudo apt install "./wopr_${VERSION}-1_${ARCH}.deb"
wopr --version
```

The program goes in `/usr/games`, where Debian keeps its games, and Debian and Ubuntu put that folder on
each user's `PATH` when they log in. If your shell cannot find `wopr` (as root, or in a small
container), run `/usr/games/wopr`. To remove it: `sudo apt remove wopr`.

### Fedora and RHEL

Download, verify, and install the `.rpm` with `dnf`:

```sh
VERSION={{< version >}}
ARCH=$(uname -m)   # x86_64 or aarch64
curl -LO "https://github.com/GhostofGoes/WOPR/releases/download/v${VERSION}/wopr-${VERSION}-1.${ARCH}.rpm"
gh attestation verify "wopr-${VERSION}-1.${ARCH}.rpm" --repo GhostofGoes/WOPR \
  --signer-workflow GhostofGoes/WOPR/.github/workflows/release.yml \
  --source-ref "refs/tags/v${VERSION}" --deny-self-hosted-runners
sudo dnf install "./wopr-${VERSION}-1.${ARCH}.rpm"
wopr --version
```

The program goes in `/usr/bin`. To remove it: `sudo dnf remove wopr`.

On openSUSE, download and verify the `.rpm` the same way, then install it with `zypper`, which refuses a
package with no GPG signature unless you tell it not to. Do that only after `gh attestation verify` has
passed:

```sh
sudo zypper install --allow-unsigned-rpm "./wopr-${VERSION}-1.${ARCH}.rpm"
```

The packages carry no GPG signature. Check them with `gh attestation verify`, as above, like every other
file in a release.
{{% /if-packages %}}

## From source

With [Go](https://go.dev/dl/) installed (any version from 1.21 on downloads the right Go for you), build
and install the latest release:

```sh
go install github.com/GhostofGoes/WOPR/cmd/wopr@latest
```

If `go` says that the module needs a newer Go and mentions `GOTOOLCHAIN=local`, your system's Go is set
not to download one, as some Linux distributions set it. Run `go env -w GOTOOLCHAIN=auto` once and try
again, or install Go from [go.dev](https://go.dev/dl/).

The program goes into Go's `bin` folder: `~/go/bin` on Linux and macOS, `%USERPROFILE%\go\bin` on
Windows (`go env GOPATH` prints the folder above it). Add that folder to your `PATH` if `wopr` is not
found. [Contributing](/contributing) explains how to build from a clone of the repository.

## The manual page

On Linux and macOS, `man wopr` can show every option, every game with how to play it and tips, and
movie mode.{{% if-packages %}} The [Linux packages](#linux-packages) install it for you.{{% /if-packages %}}
Download {{< man-page >}}{{% if-packages %}}, or take it from the archive you downloaded{{% /if-packages %}}.
Read it in place with `man ./wopr.6`, or install it so that `man wopr` finds it:

```sh
# for everyone
sudo install -d /usr/local/share/man/man6
sudo install -m 644 wopr.6 /usr/local/share/man/man6/
# or just for you
mkdir -p ~/.local/share/man/man6
cp wopr.6 ~/.local/share/man/man6/
```

`man` looks in `~/.local/share/man` when `~/.local/bin` is on your `PATH`. If it does not, add this line
to your `~/.bashrc` or `~/.zshrc`; the colon at the end keeps the system's manual pages too:

```sh
export MANPATH="$HOME/.local/share/man:"
```

## Development builds

Every CI run on `main` builds all six targets. Open a
[CI run on main](https://github.com/GhostofGoes/WOPR/actions/workflows/ci.yml?query=branch%3Amain+event%3Apush)
and download your platform's file under **Artifacts**. It is the bare program: `chmod +x` it on Linux and
macOS. Development builds are for testing; they carry no attestation.

## Uninstall

Delete the program (`~/.local/bin/wopr`, `/usr/local/bin/wopr`, or the `wopr` folder under
`%LOCALAPPDATA%\Programs` on Windows) and the manual page if you installed it.{{% if-packages %}} If you
installed a Linux package, remove it with `sudo apt remove wopr` or `sudo dnf remove wopr` instead.{{% /if-packages %}}
`wopr` keeps no settings.
If you ever ran it with `WOPR_DEBUG`, also delete its debug log; [Usage](/usage#debug-log) says where it
is.
