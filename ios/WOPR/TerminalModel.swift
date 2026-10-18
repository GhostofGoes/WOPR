// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import Observation

/// The view's state: the session, and the line the player is typing. It runs on the main actor,
/// as SwiftUI's views do. The session itself is a plain value that the tests use directly.
@MainActor
@Observable
final class TerminalModel {
    private(set) var session = Session()

    /// The line the player is typing. It is filtered as it changes, so the field never holds a
    /// character that the session would remove.
    var draft: String {
        get { typed }
        set { typed = InputFilter.clean(newValue) }
    }

    private var typed = ""

    /// Sends the draft to the session and clears it.
    func submit() {
        session.enter(draft)
        draft = ""
    }

    /// Takes a square on the board for the player.
    func tap(square: Int) {
        session.tap(square: square)
    }
}
