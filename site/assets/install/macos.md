Open **Terminal** (in *Applications*, then *Utilities*), paste this line, and press Enter. It puts `wopr`
in `~/.local/bin`, in your home folder, so it needs no password:

```sh
f=$(mktemp) && curl -fL --proto '=https' -o "$f" "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr_@VERSION@_darwin_$(uname -m | sed s/x86_64/amd64/)" && mkdir -p ~/.local/bin && install -m 755 "$f" ~/.local/bin/wopr && rm "$f" && { case ":$PATH:" in *":$HOME/.local/bin:"*) ;; *) r=~/.zprofile; [ "${SHELL##*/}" = bash ] && r=~/.bash_profile; grep -qs '/.local/bin' "$r" || echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$r" ;; esac; } && ~/.local/bin/wopr
```

To play again, open a new Terminal window and type `wopr`. The line adds `~/.local/bin` to your `PATH`
in `~/.zprofile` (or `~/.bash_profile` if you use bash), unless it is there already.
