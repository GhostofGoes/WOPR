For any Linux. Open a terminal, paste this line, and press Enter. It asks for your password, to put
`wopr` in `/usr/local/bin`:

```sh
curl -fLo wopr "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr_@VERSION@_linux_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')" && sudo install -m 755 wopr /usr/local/bin/ && rm wopr && wopr
```

To play again, type `wopr`. No `curl`? Use `wget -O wopr` in its place.
