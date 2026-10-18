// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import Foundation
import Testing

@testable import WOPR

/// The checks that internal/ui/access_test.go makes of the Go program's screens, made of the
/// app's text. archtest's TestIOSFilmLines checks the film's lines against the Go goldens.
@Suite("Script")
struct ScriptTests {
    @Test("Every line is printable ASCII", arguments: Script.all)
    func printableASCII(lines: Lines) {
        for text in lines.texts {
            #expect(text.unicodeScalars.allSatisfy { $0.value >= 0x20 && $0.value < 0x7F }, "\(text)")
        }
    }

    @Test("Every line fits 80 columns", arguments: Script.all)
    func fits(lines: Lines) {
        for text in lines.texts {
            #expect(text.count <= 80, "\(text)")
        }
    }

    @Test("Every prompt ends in a colon and a space or a question mark")
    func prompts() {
        for prompt in [Script.logon, Script.yourMove, Script.playAgain] {
            let text = prompt.texts[0].trimmingCharacters(in: .whitespaces)
            #expect(text.hasSuffix(":") || text.hasSuffix("?"), "\(text)")
        }
    }

    @Test("Every block has a line")
    func blocks() {
        #expect(Script.all.allSatisfy { !$0.texts.isEmpty })
    }
}
