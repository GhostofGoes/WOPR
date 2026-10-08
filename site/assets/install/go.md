With [Go](https://go.dev/dl/) 1.21 or newer installed, open a terminal, paste this line, and press Enter (Go
downloads the version this project needs the first time):

```sh
go install github.com/GhostofGoes/WOPR/cmd/wopr@latest
```

It builds the latest release into Go's `bin` folder: `~/go/bin` on Linux and macOS, `%USERPROFILE%\go\bin`
on Windows. If `wopr` is not found, add that folder to your `PATH`.

If `go` says `unknown directive: toolchain`, your Go is older than 1.21: install the newest from
[go.dev](https://go.dev/dl/) and try again. If it says that the module needs a newer Go and mentions
`GOTOOLCHAIN=local`, run `go env -w GOTOOLCHAIN=auto` once and try again.
