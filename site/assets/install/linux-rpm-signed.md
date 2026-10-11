For Fedora, RHEL, CentOS Stream, Rocky Linux, AlmaLinux and their relatives. Open a terminal, paste this
line, and press Enter. It asks for your password, to trust wopr's signing key and install the package:

```sh
sudo rpm --import @KEYURL@ && sudo dnf install -y --disablerepo='*' --setopt=localpkg_gpgcheck=1 "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr-@VERSION@-1.$(uname -m).rpm" && wopr
```

`dnf` checks the package's signature against the key, and it refuses a package that does not match. The
key's fingerprint is `@FINGERPRINT@`. To play again, type `wopr`.@APP-MENU@ The package puts the program
in `/usr/bin`, and it also installs the manual page, `man wopr`. To remove it, run `sudo dnf remove wopr`.
On Fedora Silverblue and the other atomic desktops, use the **Linux (any)** tab.

On openSUSE, use `zypper`, which checks the signature too:

```sh
sudo rpm --import @KEYURL@ && sudo zypper install -y "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr-@VERSION@-1.$(uname -m).rpm" && wopr
```
