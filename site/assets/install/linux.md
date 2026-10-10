For any Linux. Open a terminal, paste this line, and press Enter. It puts `wopr` in `~/.local/bin`, in
your home folder, so it needs no password:

```sh
f=$(mktemp) && u="https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr_@VERSION@_linux_$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')" && { if command -v curl >/dev/null; then curl -fL --proto '=https' -o "$f" "$u"; else wget --https-only -O "$f" "$u"; fi; } && mkdir -p ~/.local/bin && install -m 755 "$f" ~/.local/bin/wopr && rm "$f" && { case ":$PATH:" in *":$HOME/.local/bin:"*) ;; *) r=~/.bashrc; [ "${SHELL##*/}" = zsh ] && r=~/.zshrc; grep -qs '/.local/bin' "$r" || echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$r" ;; esac; } && ~/.local/bin/wopr
```

To play again, open a new terminal and type `wopr`. If `~/.local/bin` was not on your `PATH`, the line
adds it in `~/.bashrc` (or `~/.zshrc` if you use zsh). This works on Fedora Silverblue and the other atomic desktops too, where
`dnf install` does not.
