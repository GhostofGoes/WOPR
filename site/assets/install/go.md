You need [Go](https://go.dev/dl/) 1.21 or newer. Open a terminal, paste this line, and press Enter. The
first time, Go downloads the Go version that this project needs:

```sh
go install github.com/GhostofGoes/WOPR/cmd/wopr@latest
```

The line builds the latest release into Go's `bin` folder: `~/go/bin` on Linux and macOS,
`%USERPROFILE%\go\bin` on Windows. If your terminal cannot find `wopr`, add that folder to your `PATH`.

If `go` says `unknown directive: toolchain`, your Go is older than 1.21. Install the newest Go from
[go.dev](https://go.dev/dl/), and try again. If `go` says that the module needs a newer Go, and mentions
`GOTOOLCHAIN=local`, run `go env -w GOTOOLCHAIN=auto` once, and try again.
