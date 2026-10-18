// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

/// The demo's conversation with WOPR: the LOGON gate, the film's greeting, and tic-tac-toe. It
/// follows the Go persona (internal/wopr/persona.go) in a smaller form. It is a value type with no
/// clock and no randomness, so a test can play it line by line.
struct Session: Sendable, Equatable {
    /// Where the conversation is.
    enum Phase: Sendable, Equatable {
        case logon
        case greeting(step: Int)
        case shell
        case playing
        case playAgain
    }

    /// The most lines the screen keeps. Older lines scroll away.
    static let maxScreenLines = 200

    /// Failed LOGON attempts between hints, as in the Go persona.
    static let hintEvery = 3

    private(set) var phase: Phase = .logon
    private(set) var screen: [String] = []
    private(set) var prompt = ""
    private(set) var board = Board()
    private var failures = 0

    init() {
        dial()
    }

    /// Reads one line that the player typed, shows it after the prompt, and answers it.
    mutating func enter(_ input: String) {
        let text = InputFilter.clean(input)
        show(prompt + text)
        let command = InputFilter.normalize(text)
        switch phase {
        case .logon: logon(command)
        case .greeting(let step): greeting(command, step: step)
        case .shell: shell(command)
        case .playing: move(command)
        case .playAgain: playAgain(command)
        }
    }

    /// Takes square (0 to 8) for the player, as typing its number does.
    mutating func tap(square: Int) {
        guard phase == .playing else { return }
        enter(String(square + 1))
    }

    // MARK: - LOGON

    private mutating func dial() {
        phase = .logon
        show(Script.dialing)
        show(Script.connected)
        show("")
        prompt = Script.logon.texts[0]
    }

    private mutating func logon(_ command: String) {
        switch command {
        case "":
            return
        case "HELP LOGON":
            show(Script.helpNotAvailable)
        case "HELP GAMES":
            show(Script.helpGames)
        case "LIST GAMES":
            show(Script.gameList)
        case "JOSHUA":
            screen = []
            show(Script.greetings)
            show("")
            phase = .greeting(step: 0)
            prompt = ""
            return
        default:
            failures += 1
            show(Script.notRecognized)
            show("")
            if failures % Self.hintEvery == 0 {
                show(Script.logonHint)
                show("")
            }
            dial()
            return
        }
        show("")
    }

    // MARK: - The greeting and the shell

    private mutating func greeting(_ command: String, step: Int) {
        if command.isEmpty {
            return
        }
        if Self.asksForTicTacToe(command) {
            startGame()
            return
        }
        switch step {
        case 0:
            respond(Script.howFeeling)
            phase = .greeting(step: 1)
        case 1:
            respond(Script.excellent)
            phase = .greeting(step: 2)
        default:
            respond(Script.yesTheyDo)
            phase = .shell
        }
    }

    private mutating func shell(_ command: String) {
        switch command {
        case "":
            return
        case "HELP", "HELP GAMES":
            respond(Script.helpGames)
        case "LIST GAMES":
            respond(Script.gameList)
        case _ where Self.asksForTicTacToe(command):
            startGame()
        default:
            respond(Script.oneGame)
        }
    }

    /// Whether the command names tic-tac-toe, in any of the ways a player may write it.
    static func asksForTicTacToe(_ command: String) -> Bool {
        let letters = command.filter(\.isLetter)
        return letters.contains("TICTACTOE")
    }

    // MARK: - Tic-tac-toe

    private mutating func startGame() {
        board = Board()
        show("")
        show(Script.yourMarkIsX)
        show(Script.pickSquare)
        showBoard()
        phase = .playing
        prompt = Script.yourMove.texts[0]
    }

    private mutating func move(_ command: String) {
        guard let number = Int(command), (1...9).contains(number), board.place(.x, at: number - 1)
        else {
            show(Script.pickSquare)
            return
        }
        if !board.isOver, let square = Minimax.bestMove(on: board, for: .o) {
            board.place(.o, at: square)
            show(Script.woprMove.texts[0] + String(square + 1))
        }
        showBoard()
        if board.isOver {
            show(board.winner == .o ? Script.woprWins : Script.stalemate)
            phase = .playAgain
            prompt = Script.playAgain.texts[0]
        }
    }

    private mutating func playAgain(_ command: String) {
        switch command {
        case "Y", "YES":
            startGame()
        case "N", "NO":
            respond(Script.shallWe)
            phase = .shell
        default:
            return
        }
    }

    // MARK: - The screen

    private mutating func respond(_ lines: Lines) {
        show(lines)
        show("")
        prompt = ""
    }

    private mutating func showBoard() {
        show("")
        for row in board.rows {
            show(row)
        }
        show("")
    }

    private mutating func show(_ lines: Lines) {
        for line in lines.texts {
            show(line)
        }
    }

    private mutating func show(_ line: String) {
        screen.append(line)
        if screen.count > Self.maxScreenLines {
            screen.removeFirst(screen.count - Self.maxScreenLines)
        }
    }
}
