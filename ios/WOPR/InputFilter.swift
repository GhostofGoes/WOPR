// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

/// The app's one input boundary, as console.SanitizeInput is the Go program's
/// (docs/ASSURANCE.md, B1). Everything the player types or pastes goes through it before the
/// session reads it or the screen shows it.
enum InputFilter {
    /// The longest line the session reads. WOPR's commands are short, and a cap keeps a large
    /// paste from growing the screen.
    static let maxLength = 64

    /// The input with every character that is not printable ASCII removed (control characters,
    /// escape sequences, emoji and other scripts), cut to maxLength characters.
    static func clean(_ input: String) -> String {
        let printable = input.unicodeScalars.filter { $0.value >= 0x20 && $0.value < 0x7F }
        return String(String.UnicodeScalarView(printable.prefix(maxLength)))
    }

    /// The input as a command: cleaned, in capitals, with single spaces between words.
    static func normalize(_ input: String) -> String {
        clean(input).uppercased().split(separator: " ").joined(separator: " ")
    }
}
