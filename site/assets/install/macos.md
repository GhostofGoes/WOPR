Open **Terminal** (in *Applications*, then *Utilities*), paste this line, and press Enter. It asks for
your password, to put `wopr` in `/usr/local/bin`:

```sh
curl -fLo wopr "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr_@VERSION@_darwin_$(uname -m | sed s/x86_64/amd64/)" && sudo install -d /usr/local/bin && sudo install -m 755 wopr /usr/local/bin/ && rm wopr && wopr
```

To play again, type `wopr`.
