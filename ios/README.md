# WOPR for iOS (demo)

This folder holds a demo of WOPR as an iPhone and iPad app. It is a native SwiftUI app. It shows WOPR's
terminal, the LOGON gate, the film's greeting and a game of tic-tac-toe against WOPR. It is not a
release file, and no store has it. [docs/IOS.md](../docs/IOS.md) compares the ways to make a full app
and explains the choice.

The app needs iOS 26 or later. That is the last two major versions of iOS: iOS 26 and iOS 27.

## What it does

1. The app connects and shows `LOGON:`.
2. Type `JOSHUA` to open the backdoor. `HELP LOGON`, `HELP GAMES` and `LIST GAMES` also work, as in
   `wopr`.
3. Answer WOPR's greeting. The film's lines follow, up to `SHALL WE PLAY A GAME?`.
4. Type `TIC-TAC-TOE`. Type a square's number, 1 to 9, or tap the square on the board.

WOPR plays a full search, so it never loses.

## Layout

| Path | What it holds |
|---|---|
| `WOPR.xcodeproj` | The Xcode project. Its folders are synchronized groups, so a new file in `WOPR/` or `WOPRTests/` needs no change to the project. The shared scheme `WOPR` builds the app and runs the tests. |
| `WOPR/` | The app. `Session.swift` is the conversation, `TicTacToe.swift` the game and WOPR's search, `InputFilter.swift` the input filter, `Script.swift` the screen text. The views are `TerminalView.swift` and `BoardView.swift`. |
| `WOPR/PrivacyInfo.xcprivacy` | The privacy manifest. |
| `WOPR/Assets.xcassets` | The app icon and the accent colour. `go run ./internal/tools/icons` draws the icon from `packaging/icons/src/wopr-full.svg`. |
| `WOPRTests/` | The unit tests and the security tests, in Swift Testing. |
| `scripts/` | `simulator.sh` finds a simulator for the tests. `check-app.sh` checks a built app's security properties. |
| `.swift-format` | The settings for swift-format, the formatter and linter in Xcode. |

## Commands

Run these from the repository root, on a Mac with Xcode 26 or later. CI runs the same commands
(`.github/workflows/ios.yml`).

| Task | Command |
|---|---|
| Open the project | `open ios/WOPR.xcodeproj` |
| Lint | `xcrun swift-format lint --strict --recursive --parallel ios` |
| Format | `xcrun swift-format format --in-place --recursive --parallel ios` |
| Test on a simulator | `xcodebuild test -project ios/WOPR.xcodeproj -scheme WOPR -destination "id=$(ios/scripts/simulator.sh)" -enableCodeCoverage YES CODE_SIGNING_ALLOWED=NO` |
| Build for devices | `xcodebuild build -project ios/WOPR.xcodeproj -scheme WOPR -configuration Release -destination generic/platform=iOS -derivedDataPath build/ios CLANG_ENABLE_CODE_COVERAGE=NO CODE_SIGNING_ALLOWED=NO` |
| Security checks of that build | `ios/scripts/check-app.sh build/ios/Build/Products/Release-iphoneos/WOPR.app` |

`ios/scripts/simulator.sh 26` picks a simulator on iOS 26, for a test on the oldest supported version.
To run the app on your own iPhone, set your team under Signing & Capabilities in Xcode. Do not commit
that change.

## Rules

The rules in [AGENTS.md](../AGENTS.md) apply here too:

- **Film text.** A line that `Script.swift` tags `.film` is the film's own text, character for character.
  `TestIOSFilmLines` (`internal/archtest/ios_test.go`) checks each one against the Go program's
  `film_lines.golden` files. The app's lines are the Go program's lines, with the same provenance.
- **Licence headers.** Every Swift file and the privacy manifest start with the `SPDX-` lines.
- **Accessibility.** The tests check that every line is printable ASCII, fits 80 columns, and that each
  prompt ends in `:` or `?`. Text uses Dynamic Type. Each square on the board has a VoiceOver label, and
  shows its mark or its number, so no meaning depends on colour. VoiceOver reads each answer from WOPR.
- **Warnings are errors.** The project builds in the Swift 6 language mode, with complete concurrency
  checks, and treats every compiler warning as an error.

## Security

The app has the same security properties as `wopr` ([SECURITY.md](../SECURITY.md)), and fewer
boundaries:

- It has no network code, and it keeps no data. It does not use the file system, `UserDefaults` or a
  web view.
- It has no entitlements, and it asks for no permission.
- Its Info.plist has no App Transport Security exception, no URL scheme and no background mode.
- Its privacy manifest declares no tracking, no collected data and no API that needs a reason.
- Its one input is the text field. `InputFilter` keeps only printable ASCII and cuts a line to 64
  characters, before the session reads it or the screen shows it. The screen keeps 200 lines.
- The program links only the system's libraries, embeds no framework, and is position independent.
- `ITSAppUsesNonExemptEncryption` is `NO`: the app uses no encryption.

`WOPRTests/SecurityTests.swift` tests the input filter, the Info.plist and the privacy manifest.
`scripts/check-app.sh` checks the sources, the release build's bundle and its program.
