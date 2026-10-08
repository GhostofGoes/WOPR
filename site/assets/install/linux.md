For any Linux. Open a terminal, paste this line, and press Enter. It asks for your password, to put
`wopr` in `/usr/local/bin`:

```sh
f=$(mktemp) && u="https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr_@VERSION@_linux_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')" && { if command -v curl >/dev/null; then curl -fL -o "$f" "$u"; else wget -O "$f" "$u"; fi; } && sudo install -m 755 "$f" /usr/local/bin/wopr && rm "$f" && wopr
```

To play again, type `wopr`. This works on Fedora Silverblue and the other atomic desktops too, where
`dnf install` does not.
