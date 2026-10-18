// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import Testing

@testable import WOPR

@Suite("Tic-tac-toe")
struct TicTacToeTests {
    /// A board with X on xs and O on os.
    private func board(x xs: [Int], o os: [Int]) -> Board {
        var board = Board()
        for square in xs {
            let placed = board.place(.x, at: square)
            #expect(placed)
        }
        for square in os {
            let placed = board.place(.o, at: square)
            #expect(placed)
        }
        return board
    }

    @Test("Every line of three wins", arguments: Board.lines)
    func lineWins(line: [Int]) {
        #expect(board(x: line, o: []).winner == .x)
    }

    @Test("A full board with no line is a draw")
    func draw() {
        let full = board(x: [0, 1, 5, 6, 8], o: [2, 3, 4, 7])
        #expect(full.winner == nil)
        #expect(full.isOver)
        #expect(full.emptySquares.isEmpty)
    }

    @Test("A square cannot be taken twice or off the board")
    func illegalPlaces() {
        var board = Board()
        let results = [
            board.place(.x, at: 4), board.place(.o, at: 4), board.place(.o, at: -1), board.place(.o, at: 9),
        ]
        #expect(results == [true, false, false, false])
        #expect(board.squares[4] == .x)
    }

    @Test("WOPR takes a win when it has one")
    func takesWin() {
        let position = board(x: [0, 1, 8], o: [3, 4])
        #expect(Minimax.bestMove(on: position, for: .o) == 5)
    }

    @Test("WOPR blocks a line of two")
    func blocks() {
        let position = board(x: [0, 1], o: [4])
        #expect(Minimax.bestMove(on: position, for: .o) == 2)
    }

    @Test("WOPR has no move on a finished board")
    func noMoveWhenOver() {
        #expect(Minimax.bestMove(on: board(x: [0, 1, 2], o: [3, 4]), for: .o) == nil)
    }

    /// Plays every game that X can play against WOPR, and checks that X never wins.
    @Test("WOPR never loses, whatever X plays")
    func neverLoses() {
        var games = 0
        func play(_ board: Board) {
            for square in board.emptySquares {
                var next = board
                next.place(.x, at: square)
                if !next.isOver, let reply = Minimax.bestMove(on: next, for: .o) {
                    next.place(.o, at: reply)
                }
                if next.isOver {
                    games += 1
                    #expect(next.winner != .x, "X won: \(next.rows)")
                } else {
                    play(next)
                }
            }
        }
        play(Board())
        #expect(games > 100)
    }

    @Test("The board draws as three rows with the empty squares' numbers")
    func rows() {
        #expect(board(x: [0], o: [4]).rows == [" X | 2 | 3 ", " 4 | O | 6 ", " 7 | 8 | 9 "])
    }
}
