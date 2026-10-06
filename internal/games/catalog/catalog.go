// Package catalog is the one place where game constructors are wired into a registry.
// It imports every game package; nothing else does.
package catalog

import (
	"github.com/GhostofGoes/WOPR/internal/games"
	"github.com/GhostofGoes/WOPR/internal/games/blackjack"
	"github.com/GhostofGoes/WOPR/internal/games/checkers"
	"github.com/GhostofGoes/WOPR/internal/games/chess"
	"github.com/GhostofGoes/WOPR/internal/games/ending"
	"github.com/GhostofGoes/WOPR/internal/games/gtw"
	"github.com/GhostofGoes/WOPR/internal/games/poker"
	"github.com/GhostofGoes/WOPR/internal/games/tictactoe"
	"github.com/GhostofGoes/WOPR/internal/proto"
)

// entries lists every game in the film's LIST GAMES order, then the unlisted ones.
// Blurbs are original text. A game becomes Playable when its package lands.
func entries() []games.Entry {
	return []games.Entry{
		{Info: games.Info{
			Number: 1, Listed: true, Name: "FALKEN'S MAZE", Slug: "falkens-maze", Aliases: []string{"maze"},
			Layout: proto.LayoutPanel, PanelRows: 12, Blurb: "Find the exit while WOPR learns your habits and moves the walls.",
		}},
		{Info: games.Info{
			Number: 2, Listed: true, Name: "BLACK JACK", Slug: "black-jack", Aliases: []string{"blackjack"},
			Layout: proto.LayoutConsole, Status: games.Playable, Blurb: "WOPR deals; get closer to 21 than the dealer without going over.",
		}, New: blackjack.New},
		{Info: games.Info{
			Number: 3, Listed: true, Name: "GIN RUMMY", Slug: "gin-rummy", Aliases: []string{"gin", "rummy"},
			Layout: proto.LayoutPanel, PanelRows: 8, Blurb: "Form melds, knock or go gin against WOPR.",
		}},
		{Info: games.Info{
			Number: 4, Listed: true, Name: "HEARTS", Slug: "hearts",
			Layout: proto.LayoutPanel, PanelRows: 10, Blurb: "Four hands, you against three WOPR seats; avoid hearts and the queen of spades.",
		}},
		{Info: games.Info{
			Number: 5, Listed: true, Name: "BRIDGE", Slug: "bridge",
			Layout: proto.LayoutPanel, PanelRows: 10, Blurb: "WOPR bids all four seats; you play the contract as declarer.",
		}},
		{Info: games.Info{
			Number: 6, Listed: true, Name: "CHECKERS", Slug: "checkers", Aliases: []string{"draughts"},
			Layout: proto.LayoutPanel, PanelRows: 12, Status: games.Playable, Blurb: "8x8 checkers with forced captures and kings.",
		}, New: checkers.New},
		{Info: games.Info{
			Number: 7, Listed: true, Name: "CHESS", Slug: "chess",
			Layout: proto.LayoutPanel, PanelRows: 12, Status: games.Playable, Blurb: "Type your moves (e2e4 or Nf3); WOPR answers.",
		}, New: chess.New},
		{Info: games.Info{
			Number: 8, Listed: true, Name: "POKER", Slug: "poker",
			Layout: proto.LayoutConsole, Status: games.Playable, Blurb: "Heads-up five-card draw for chips.",
		}, New: poker.New},
		{Info: games.Info{
			Number: 9, Listed: true, Name: "FIGHTER COMBAT", Slug: "fighter-combat", Aliases: []string{"dogfight"},
			Layout: proto.LayoutConsole, Blurb: "A turn-based dogfight: altitude, energy and aspect.",
		}},
		{Info: games.Info{
			Number: 10, Listed: true, Name: "GUERRILLA ENGAGEMENT", Slug: "guerrilla-engagement", Aliases: []string{"guerrilla"},
			Layout: proto.LayoutConsole, Blurb: "An asymmetric campaign of raids, patrols and support.",
		}},
		{Info: games.Info{
			Number: 11, Listed: true, Name: "DESERT WARFARE", Slug: "desert-warfare", Aliases: []string{"desert"},
			Layout: proto.LayoutConsole, Blurb: "Armour against armour across a desert front, with supply lines.",
		}},
		{Info: games.Info{
			Number: 12, Listed: true, Name: "AIR-TO-GROUND ACTIONS", Slug: "air-to-ground-actions", Aliases: []string{"air-to-ground"},
			Layout: proto.LayoutConsole, Blurb: "Plan strike packages against defended targets.",
		}},
		{Info: games.Info{
			Number: 13, Listed: true, Name: "THEATERWIDE TACTICAL WARFARE", Slug: "theaterwide-tactical-warfare", Aliases: []string{"tactical"},
			Layout: proto.LayoutConsole, Blurb: "Corps-level moves on a European front, with an escalation ladder.",
		}},
		{Info: games.Info{
			Number: 14, Listed: true, Name: "THEATERWIDE BIOTOXIC AND CHEMICAL WARFARE", Slug: "theaterwide-biotoxic-and-chemical-warfare", Aliases: []string{"biotoxic", "chemical"},
			Layout: proto.LayoutConsole, Blurb: "Contamination spreads; nobody wins.",
		}},
		{Info: games.Info{
			Number: 15, Listed: true, Name: "GLOBAL THERMONUCLEAR WAR", Slug: gtw.Slug, Aliases: []string{"gtw", "thermonuclear"},
			Layout: proto.LayoutConsole, Status: games.Playable, Blurb: "Choose a side, list your targets, and watch the big board.",
		}, New: gtw.New},
		{Info: games.Info{
			Name: "TIC-TAC-TOE", Slug: "tic-tac-toe", Aliases: []string{"tictactoe", "ttt", "noughts and crosses"},
			Layout: proto.LayoutPanel, PanelRows: 9, Status: games.Playable, Blurb: "Perfect play from WOPR. Try zero players.",
		}, New: tictactoe.New},
		// The climax after zero players: internal, reached only by a hand-off.
		{Info: games.Info{
			Name: "ENDING", Slug: ending.Slug, Layout: proto.LayoutFull, Status: games.Playable, Internal: true,
		}, New: ending.New},
	}
}

// Registry returns the validated registry of every game.
func Registry() *games.Registry {
	r, err := games.NewRegistry(entries()...)
	if err != nil {
		panic("catalog: " + err.Error()) // static data; TestRegistry keeps it valid
	}
	return r
}
