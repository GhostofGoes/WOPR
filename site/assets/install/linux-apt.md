For Debian, Ubuntu, Linux Mint and their relatives. Open a terminal, paste this line, and press Enter.
It asks for your password, to install the package:

```sh
f=$(mktemp --suffix=.deb) && wget -O "$f" "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr_@VERSION@-1_$(dpkg --print-architecture).deb" && chmod 644 "$f" && sudo apt install -y "$f" && rm "$f" && wopr
```

To play again, type `wopr`. The package also installs the manual page, `man wopr`. To remove it:
`sudo apt remove wopr`.
