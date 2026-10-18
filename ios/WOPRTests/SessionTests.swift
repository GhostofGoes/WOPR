// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

import Testing

@testable import WOPR

@Suite("Session")
struct SessionTests {
    /// A session past the LOGON gate and the greeting, at the shell.
    private func atShell() -> Session {
        var session = Session()
        for line in ["Joshua", "Hello", "I'm fine", "Love to"] {
            session.enter(line)
        }
        return session
    }

    @Test("A new session dials and asks for LOGON")
    func dials() {
        let session = Session()
        #expect(session.phase == .logon)
        #expect(session.prompt == "LOGON:  ")
        #expect(session.screen.contains("CONNECTED."))
    }

    @Test("A wrong LOGON is not recognized, and the session dials again")
    func wrongLogon() {
        var session = Session()
        session.enter("falken")
        #expect(session.screen.contains("INDENTIFICATION NOT RECOGNIZED BY SYSTEM"))  // the film's spelling
        #expect(session.phase == .logon)
        #expect(session.prompt == "LOGON:  ")
    }

    @Test("Every third wrong LOGON gives the hint")
    func hint() {
        var session = Session()
        session.enter("one")
        session.enter("two")
        #expect(!session.screen.contains(Script.logonHint.texts[0]))
        session.enter("three")
        #expect(session.screen.contains(Script.logonHint.texts[0]))
    }

    @Test("HELP LOGON and HELP GAMES answer as the film does")
    func logonHelp() {
        var session = Session()
        session.enter("help logon")
        #expect(session.screen.contains("HELP NOT AVAILABLE"))
        session.enter("help games")
        #expect(session.screen.contains(Script.helpGames.texts[0]))
        #expect(session.phase == .logon)
    }

    @Test("JOSHUA opens the backdoor, and the greeting follows the film")
    func greeting() {
        var session = Session()
        session.enter("  joshua ")
        #expect(session.screen.first == "GREETINGS PROFESSOR FALKEN.")
        #expect(session.phase == .greeting(step: 0))
        session.enter("Hello")
        #expect(session.screen.contains("HOW ARE YOU FEELING TODAY?"))
        session.enter("I'm fine")
        #expect(session.screen.contains(Script.excellent.texts[1]))
        session.enter("People sometimes make mistakes")
        #expect(session.screen.contains("YES THEY DO.  SHALL WE PLAY A GAME?"))
        #expect(session.phase == .shell)
    }

    @Test("An empty line is not the player's turn")
    func emptyLine() {
        var session = Session()
        session.enter("joshua")
        session.enter("")
        #expect(session.phase == .greeting(step: 0))
    }

    @Test(
        "Tic-tac-toe starts however the player writes it",
        arguments: ["tic-tac-toe", "TIC TAC TOE", "tictactoe", "let's play tic-tac-toe"])
    func startsGame(request: String) {
        var session = atShell()
        session.enter(request)
        #expect(session.phase == .playing)
        #expect(session.prompt == "YOUR MOVE: ")
    }

    @Test("The greeting ends early when the player asks for the game")
    func earlyGame() {
        var session = Session()
        session.enter("joshua")
        session.enter("tic-tac-toe")
        #expect(session.phase == .playing)
    }

    @Test("Another game gets the demo's one-game answer")
    func otherGame() {
        var session = atShell()
        session.enter("global thermonuclear war")
        #expect(session.screen.contains(Script.oneGame.texts[0]))
        #expect(session.phase == .shell)
    }

    @Test("A move that is not an empty square asks again", arguments: ["0", "10", "A", "", "5 5"])
    func badMove(move: String) {
        var session = atShell()
        session.enter("tic-tac-toe")
        session.enter(move)
        #expect(session.board == Board())
        #expect(session.phase == .playing)
    }

    @Test("A move gets WOPR's answer, and a tap is a move")
    func moves() {
        var session = atShell()
        session.enter("tic-tac-toe")
        session.tap(square: 4)
        #expect(session.board.squares[4] == .x)
        #expect(session.board.squares.compactMap { $0 }.count == 2)
        #expect(session.screen.contains { $0.hasPrefix("WOPR: ") })
    }

    @Test("A game ends in a stalemate or WOPR's win, then asks to play again")
    func gameEnds() {
        var session = atShell()
        session.enter("tic-tac-toe")
        while session.phase == .playing, let square = session.board.emptySquares.first {
            session.tap(square: square)
        }
        #expect(session.phase == .playAgain)
        #expect(session.board.winner != .x)
        session.enter("y")
        #expect(session.phase == .playing)
        #expect(session.board == Board())
    }

    @Test("N after a game goes back to the shell")
    func noMoreGames() {
        var session = atShell()
        session.enter("tic-tac-toe")
        while session.phase == .playing, let square = session.board.emptySquares.first {
            session.tap(square: square)
        }
        session.enter("n")
        #expect(session.phase == .shell)
        #expect(session.screen.contains("SHALL WE PLAY A GAME?"))
    }

    @Test("The screen keeps at most its limit of lines")
    func screenLimit() {
        var session = Session()
        for _ in 0..<100 {
            session.enter("wrong")
        }
        #expect(session.screen.count == Session.maxScreenLines)
    }
}
