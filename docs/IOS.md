# An iOS app: the options

This document compares the ways to make WOPR an iPhone and iPad app. It explains why the demo in
[`ios/`](../ios/README.md) is a native SwiftUI app, and what a full app would need next.

## Requirements

- iOS 26 and iOS 27, the last two major versions, on iPhone and iPad.
- The rules in [AGENTS.md](../AGENTS.md): film text exactly as the film shows it, no network code,
  seeded randomness, and accessibility.
- Apple's guidelines: a privacy manifest, no permissions that the app does not need, Dynamic Type and
  VoiceOver, and touch input that does not need a keyboard.
- The project's supply chain rules: pinned tools, a 14-day cooldown for dependencies, and CI actions
  from the allowlist only.

## The options

| Option | How it works | For | Against |
|---|---|---|---|
| **1. Native SwiftUI** (the demo) | The app's screens are SwiftUI views. The game logic is in Swift. | Apple's own tools only, with no new dependency. The best accessibility, touch input and look. Small app. Easy to review. | Each game must be written again in Swift, and the film's text is in two places. A Go test checks the film's lines, but not the rest of the behaviour. |
| **2. SwiftUI with the Go core** (gomobile) | `gomobile bind` builds the Go programs (`internal/proto`) into an XCFramework. SwiftUI shows their `proto.Output` and sends them `proto.Event`s. | One copy of every game, its AI and the film's text. The Go tests cover the app's behaviour. | A new dependency, `golang.org/x/mobile`, with few tagged versions. The import DAG needs a new package with an exported API. The app is larger, because it holds the Go runtime. cgo and Xcode in the Go build. |
| **3. SwiftUI with the Go core** (`-buildmode=c-archive`) | As option 2, but `GOOS=ios go build -buildmode=c-archive` builds a C library, and a small C header is the bridge. | No `golang.org/x/mobile`. Full control of the API. | More code to write and to keep: the C bridge, memory ownership and the XCFramework. cgo, as in option 2. |
| **4. A terminal emulator** | A terminal view (for example SwiftTerm) runs the Bubble Tea UI in the app, through pipes. | The whole program, unchanged, on iOS. | A third-party dependency. A terminal is hard to use with touch and with VoiceOver. Bubble Tea assumes a terminal that iOS does not have. Apple can refuse an app that is a terminal around other code. |
| **5. Cross-platform frameworks** (Flutter, React Native, Kotlin Multiplatform, .NET MAUI) | One UI framework for iOS and Android. | One app for two systems. | A new language, a new toolchain and many new dependencies. No reuse of the Go code, so the same work as option 1, with less native accessibility. |
| **6. A web app** (Go to WebAssembly, in Safari) | The Go program runs in the browser. A web page draws the screen. | Also gives the roadmap's "Play in a web browser" ([#15](https://github.com/GhostofGoes/WOPR/issues/15)). No store review. | Not an app from the App Store. Large WebAssembly files. A web terminal has the same touch and VoiceOver problems as option 4. |

## The choice

The demo uses option 1. It needs only Apple's tools, so the CI needs no new action or dependency, and
it shows what a native app can do with touch and VoiceOver. Its logic is small: the LOGON gate, the
greeting and tic-tac-toe. Its lines are the Go program's lines.

For a full app with all of WOPR's games, the recommendation is option 2 or option 3. The Go programs
already receive events and return outputs (`internal/proto`), and they never access the terminal. So a
SwiftUI host can draw them as the Bubble Tea host does. Option 3 adds no dependency. Option 2 needs
less code. Each needs the owner's decision, because each adds cgo to an iOS build and changes the
import DAG (`docs/PLAN.md` §4.1).

## Next steps for a full app

1. Choose option 2 or 3. Add a Go package with an exported API over `proto.Program`.
2. Make a SwiftUI host for `proto.Output`: lines, prompts, keys, the canvas and the themes.
3. Sign the app with the owner's Apple Developer account, in a `release` environment in GitHub.
4. Add the app to App Store Connect, with its privacy details: no data collected.
5. Add the iOS app to `SECURITY.md`, `docs/ASSURANCE.md` and the release process.
