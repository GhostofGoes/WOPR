For Fedora, RHEL, CentOS Stream, Rocky Linux, AlmaLinux and their relatives. Open a terminal, paste this
line, and press Enter. It asks for your password, to install the package:

```sh
sudo dnf install -y --disablerepo='*' "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr-@VERSION@-1.$(uname -m).rpm" && wopr
```

`dnf` may say it skipped the OpenPGP checks for one package: that is expected, since the package has no
GPG signature. To play again, type `wopr`, or click **WOPR** in your app menu, which opens it in a
terminal window. The package puts the program in `/usr/bin` and also installs the manual page,
`man wopr`. To remove it: `sudo dnf remove wopr`. On Fedora Silverblue and the other atomic
desktops, use the **Linux** tab.

Or download `wopr-@VERSION@-1.x86_64.rpm` from the
[release](https://github.com/GhostofGoes/WOPR/releases/tag/v@VERSION@) (on an Arm computer,
`wopr-@VERSION@-1.aarch64.rpm`) and open it: your desktop's software app, such as Software on Fedora,
offers to install it. If it will not, use the line above.

On openSUSE, use `zypper`. The package has no GPG signature, so `zypper` needs `--allow-unsigned-rpm`; to
check the package first, see [Verifying binaries](/install#verifying-binaries-attestation).

```sh
sudo zypper install -y --allow-unsigned-rpm "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr-@VERSION@-1.$(uname -m).rpm" && wopr
```
