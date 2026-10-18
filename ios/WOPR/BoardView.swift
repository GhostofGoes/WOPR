// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import SwiftUI

/// The tic-tac-toe board as nine buttons, for players who would rather tap than type. Each
/// square shows its mark or its number, so its meaning never depends on colour.
struct BoardView: View {
    let board: Board
    let onTap: (Int) -> Void

    var body: some View {
        Grid(horizontalSpacing: 8, verticalSpacing: 8) {
            ForEach(0..<3, id: \.self) { row in
                GridRow {
                    ForEach(0..<3, id: \.self) { col in
                        square(row * 3 + col)
                    }
                }
            }
        }
        .frame(maxWidth: 320)
        .frame(maxWidth: .infinity)
    }

    private func square(_ index: Int) -> some View {
        let mark = board.squares[index]
        return Button {
            onTap(index)
        } label: {
            Text(verbatim: mark.map { String($0.rawValue) } ?? String(index + 1))
                .font(.system(.title, design: .monospaced).bold())
                .opacity(mark == nil ? 0.5 : 1)
                .frame(maxWidth: .infinity, minHeight: 64)
                .overlay(RoundedRectangle(cornerRadius: 6).stroke(Color.phosphor, lineWidth: 2))
        }
        .buttonStyle(.plain)
        .disabled(mark != nil || board.isOver)
        .accessibilityLabel(Self.label(for: index, mark: mark))
    }

    /// What VoiceOver says for a square: its place on the board and what is in it.
    static func label(for index: Int, mark: Mark?) -> String {
        let place = "Row \(index / 3 + 1), column \(index % 3 + 1)"
        guard let mark else { return "\(place), empty" }
        return "\(place), \(mark.rawValue)"
    }
}
