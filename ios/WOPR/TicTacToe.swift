// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

/// A mark on the tic-tac-toe board.
enum Mark: Character, Sendable, Equatable {
    case x = "X"
    case o = "O"

    /// The other side's mark.
    var opponent: Mark { self == .x ? .o : .x }
}

/// A tic-tac-toe position: squares 0 to 8, row by row, as in internal/games/tictactoe.
struct Board: Sendable, Equatable {
    /// The squares that make three in a row.
    static let lines: [[Int]] = [
        [0, 1, 2], [3, 4, 5], [6, 7, 8], [0, 3, 6], [1, 4, 7], [2, 5, 8], [0, 4, 8], [2, 4, 6],
    ]

    private(set) var squares: [Mark?] = Array(repeating: nil, count: 9)

    /// The side with three in a row, if any.
    var winner: Mark? {
        for line in Self.lines {
            if let mark = squares[line[0]], squares[line[1]] == mark, squares[line[2]] == mark {
                return mark
            }
        }
        return nil
    }

    /// The empty squares, in order.
    var emptySquares: [Int] { squares.indices.filter { squares[$0] == nil } }

    /// Whether the game is over: a side won or the board is full.
    var isOver: Bool { winner != nil || emptySquares.isEmpty }

    /// Puts mark on square, and returns false if the square is not empty or does not exist.
    @discardableResult
    mutating func place(_ mark: Mark, at square: Int) -> Bool {
        guard squares.indices.contains(square), squares[square] == nil else { return false }
        squares[square] = mark
        return true
    }

    /// The board drawn as text, with the numbers of the empty squares.
    var rows: [String] {
        (0..<3).map { row in
            (0..<3).map { col in
                let square = row * 3 + col
                return " \(squares[square].map { String($0.rawValue) } ?? String(square + 1)) "
            }.joined(separator: "|")
        }
    }
}

/// WOPR's player: a full minimax search, which never loses. The board allows fewer than 9! move
/// orders, so the search is quick and needs no background thread. Ties go to the lowest square, so
/// the same position always gives the same move.
enum Minimax {
    /// The best square for mark to take, or nil when the game is over.
    static func bestMove(on board: Board, for mark: Mark) -> Int? {
        guard !board.isOver else { return nil }
        var best: (square: Int, score: Int)?
        for square in board.emptySquares {
            var next = board
            next.place(mark, at: square)
            let score = -value(of: next, toMove: mark.opponent, depth: 1)
            if let current = best, score <= current.score {
                continue
            }
            best = (square, score)
        }
        return best?.square
    }

    /// The position's value for the side to move: positive if it wins, negative if it loses, 0
    /// for a draw. Quicker wins and slower losses score further from 0.
    private static func value(of board: Board, toMove mark: Mark, depth: Int) -> Int {
        if let winner = board.winner {
            return winner == mark ? 10 - depth : depth - 10
        }
        if board.emptySquares.isEmpty {
            return 0
        }
        var best = Int.min
        for square in board.emptySquares {
            var next = board
            next.place(mark, at: square)
            best = max(best, -value(of: next, toMove: mark.opponent, depth: depth + 1))
        }
        return best
    }
}
