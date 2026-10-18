// SPDX-FileCopyrightText: 2026 Christopher Goes
// SPDX-License-Identifier: MIT

/// Where a block of WOPR's screen text comes from (docs/PLAN.md §2.1).
enum Provenance: Sendable, Equatable {
    /// The film's own screen text, character for character. Never change it (AGENTS.md, "Film
    /// text and provenance"). archtest's TestIOSFilmLines checks each one against the Go goldens.
    case film
    /// From one transcription of the film only. These are the Go game's lines.
    case reconstructed
    /// This project's own text, in WOPR's voice.
    case original
}

/// A block of screen lines that share one provenance.
struct Lines: Sendable, Equatable {
    let texts: [String]
    let provenance: Provenance
}

private func film(_ texts: String...) -> Lines { Lines(texts: texts, provenance: .film) }
private func recon(_ texts: String...) -> Lines { Lines(texts: texts, provenance: .reconstructed) }
private func orig(_ texts: String...) -> Lines { Lines(texts: texts, provenance: .original) }

/// The demo's screen text. The texts are the Go program's (internal/wopr/lines.go and
/// internal/games/tictactoe), so the app says what `wopr` says.
enum Script {
    // The film's lines below are its screen text, character for character: two spaces after the
    // colon and after each full stop, and the system's own misspelling of IDENTIFICATION.
    static let logon = film("LOGON:  ")
    static let notRecognized = film("INDENTIFICATION NOT RECOGNIZED BY SYSTEM", "--CONNECTION TERMINATED--")
    static let helpNotAvailable = film("HELP NOT AVAILABLE")
    static let helpGames = film(
        "'GAMES' REFERS TO MODELS, SIMULATIONS AND GAMES",
        "WHICH HAVE TACTICAL AND STRATEGIC APPLICATIONS."
    )
    static let greetings = film("GREETINGS PROFESSOR FALKEN.")
    static let howFeeling = film("HOW ARE YOU FEELING TODAY?")
    static let excellent = film(
        "EXCELLENT.  IT'S BEEN A LONG TIME.  CAN YOU EXPLAIN",
        "THE REMOVAL OF YOUR USER ACCOUNT NUMBER ON 6/23/73?"
    )
    static let yesTheyDo = film("YES THEY DO.  SHALL WE PLAY A GAME?")
    static let improper = film("** IMPROPER REQUEST **")

    static let stalemate = recon("STALEMATE.", "WANT TO PLAY AGAIN?")

    static let dialing = orig("CONNECTING...")
    static let connected = orig("CONNECTED.")
    static let logonHint = orig("** HINT: THE BACKDOOR IS THE NAME OF FALKEN'S SON **")
    static let shallWe = orig("SHALL WE PLAY A GAME?")
    static let gameList = orig("TIC-TAC-TOE")
    static let oneGame = orig("THIS DEMO HAS ONE GAME: TIC-TAC-TOE.")
    static let yourMarkIsX = orig("YOU ARE X. WOPR IS O.")
    static let pickSquare = orig("CHOOSE AN EMPTY SQUARE, 1 TO 9.")
    static let yourMove = orig("YOUR MOVE: ")
    static let woprMove = orig("WOPR: ")  // followed by the square WOPR took
    static let woprWins = orig("WINNER: WOPR.", "WANT TO PLAY AGAIN?")
    static let playAgain = orig("Y OR N: ")

    /// Every block, for the tests.
    static let all: [Lines] = [
        logon, notRecognized, helpNotAvailable, helpGames, greetings, howFeeling, excellent,
        yesTheyDo, improper, stalemate, dialing, connected, logonHint, shallWe, gameList, oneGame,
        yourMarkIsX, pickSquare, yourMove, woprMove, woprWins, playAgain,
    ]
}
