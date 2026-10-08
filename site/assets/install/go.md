With [Go](https://go.dev/dl/) installed, open a terminal, paste this line, and press Enter:

```sh
go install github.com/GhostofGoes/WOPR/cmd/wopr@latest
```

It builds the latest release into Go's `bin` folder: `~/go/bin` on Linux and macOS, `%USERPROFILE%\go\bin`
on Windows. If `wopr` is not found, add that folder to your `PATH`.

If `go` says that the module needs a newer Go and mentions `GOTOOLCHAIN=local`, run
`go env -w GOTOOLCHAIN=auto` once and try again.
