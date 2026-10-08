For Fedora, RHEL, CentOS Stream, Rocky Linux, AlmaLinux and their relatives. Open a terminal, paste this
line, and press Enter. It asks for your password, to install the package:

```sh
sudo dnf install -y "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr-@VERSION@-1.$(uname -m).rpm" && wopr
```

To play again, type `wopr`. The package also installs the manual page, `man wopr`. To remove it:
`sudo dnf remove wopr`.

On openSUSE, use `zypper`, which installs a package without a GPG signature only when told to:

```sh
sudo zypper install --allow-unsigned-rpm "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr-@VERSION@-1.$(uname -m).rpm"
```
