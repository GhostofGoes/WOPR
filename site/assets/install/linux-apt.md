For Debian, Ubuntu, Linux Mint and their relatives. Open a terminal, paste this line, and press Enter.
It asks for your password, to install the package:

```sh
f=$(mktemp --suffix=.deb) && u="https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr_@VERSION@-1_$(dpkg --print-architecture).deb" && { if command -v wget >/dev/null; then wget -O "$f" "$u"; else curl -fL -o "$f" "$u"; fi; } && chmod 644 "$f" && sudo apt install -y "$f" && rm "$f" && wopr
```

To play again, type `wopr`.@APP-MENU@ The package puts the program in `/usr/games`; as root, or in a
small container, where that folder is not on the `PATH`, run `/usr/games/wopr`. It also installs the
manual page, `man wopr`. To remove it: `sudo apt remove wopr`.

Or download `wopr_@VERSION@-1_amd64.deb` from the
[release](https://github.com/GhostofGoes/WOPR/releases/tag/v@VERSION@) (on an Arm computer,
`wopr_@VERSION@-1_arm64.deb`) and open it. On Ubuntu, App Center opens, warns that the package comes
from a third party, and installs it when you click **Install**.
