Open **Terminal** (in *Applications*, then *Utilities*), paste this line, and press Enter. It asks for
your password, to put `wopr` in `/usr/local/bin`; nothing shows while you type it.

```sh
f=$(mktemp) && curl -fL -o "$f" "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr_@VERSION@_darwin_$(uname -m | sed s/x86_64/amd64/)" && sudo mkdir -p /usr/local/bin && sudo install -m 755 "$f" /usr/local/bin/wopr && rm "$f" && wopr
```

To play again, type `wopr`. Your account must be an administrator's to use `sudo`.
