// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import SwiftUI

/// WOPR's screen: the lines so far, the board during a game, and the prompt with the line the
/// player is typing.
struct TerminalView: View {
    @State private var model = TerminalModel()
    @FocusState private var typing: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            transcript
            if model.session.phase == .playing {
                BoardView(board: model.session.board) { square in
                    model.tap(square: square)
                }
            }
            inputLine
        }
        .padding()
        .font(.system(.body, design: .monospaced))
        .foregroundStyle(Color.phosphor)
        .background(Color.black, ignoresSafeAreaEdges: .all)
        .onAppear { typing = true }
        .onChange(of: model.session.screen) { _, screen in
            // VoiceOver reads WOPR's answer as it appears, as a sighted player sees it.
            if let line = screen.last(where: { !$0.trimmingCharacters(in: .whitespaces).isEmpty }) {
                AccessibilityNotification.Announcement(line).post()
            }
        }
    }

    private var transcript: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 2) {
                    ForEach(Array(model.session.screen.enumerated()), id: \.offset) { index, line in
                        Text(verbatim: line.isEmpty ? " " : line)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .id(index)
                    }
                }
            }
            .defaultScrollAnchor(.bottom)
            .onChange(of: model.session.screen.count) { _, count in
                proxy.scrollTo(count - 1, anchor: .bottom)
            }
        }
    }

    private var inputLine: some View {
        HStack(spacing: 0) {
            Text(verbatim: model.session.prompt)
                .accessibilityHidden(true)
            TextField("", text: $model.draft)
                .focused($typing)
                .textInputAutocapitalization(.characters)
                .autocorrectionDisabled()
                .keyboardType(.asciiCapable)
                .submitLabel(.send)
                .onSubmit {
                    model.submit()
                    typing = true
                }
                .accessibilityLabel(
                    model.session.prompt.isEmpty ? Text("Reply to WOPR") : Text(verbatim: model.session.prompt)
                )
        }
    }
}

extension Color {
    /// The green of WOPR's screen. On black it has a contrast ratio above 14:1.
    static let phosphor = Color(red: 0.2, green: 1.0, blue: 0.4)
}

#Preview {
    TerminalView()
}
