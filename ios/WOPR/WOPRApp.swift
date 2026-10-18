// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import SwiftUI

/// The demo app: one window that shows WOPR's terminal.
@main
struct WOPRApp: App {
    var body: some Scene {
        WindowGroup {
            TerminalView()
                .preferredColorScheme(.dark)
        }
    }
}
