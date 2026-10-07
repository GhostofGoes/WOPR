package scenes

import (
	"time"

	"github.com/GhostofGoes/WOPR/internal/proto"
	"github.com/GhostofGoes/WOPR/internal/script"
)

// Constructors for the scene data. Film lines are reconstructed until the M5 viewing pass.

// say is WOPR speaking.
func say(ls ...script.Ls) Say { return Say{Lines: join(ls), Pace: proto.PaceSpeech} }

// table is WOPR's fast output: lists, headers, tables.
func table(ls ...script.Ls) Say { return Say{Lines: join(ls), Pace: proto.PaceTable} }

func join(ls []script.Ls) script.Ls {
	var out script.Ls
	for _, l := range ls {
		out = append(out, l...)
	}
	return out
}

// typed is a line the user types with no prompt before it.
func typed(text string) Type { return Type{Text: script.User(script.Reconstructed, text)[0]} }

// typedAt is a line the user types at prompt, as the film shows it.
func typedAt(prompt script.Ls, text string) Type { return typedAs(script.Reconstructed, prompt, text) }

// typedAs is a line the user types at prompt, tagged p: the film's, or ours (Enter at a prompt
// of this project's own).
func typedAs(p script.Prov, prompt script.Ls, text string) Type {
	return Type{Prompt: prompt, Text: script.User(p, text)[0]}
}

// pause is a pause in the film's pacing (reconstructed) or in ours (original).
func pause(d time.Duration, p script.Prov) Wait { return Wait{D: d, Prov: p} }

// page is a new page: the film's (reconstructed) or ours (original).
func page(p script.Prov) Clear { return Clear{Prov: p} }

// steps joins runs of steps.
func steps(runs ...[]Step) []Step {
	var out []Step
	for _, r := range runs {
		out = append(out, r...)
	}
	return out
}

var (
	recon = script.Recon
	orig  = script.Orig
	blank = script.Orig("")
)

// Text more than one scene shows. The persona prints the same lines (internal/wopr/lines.go),
// which this package may not import, and the consistency tests in internal/movie keep the two
// equal: first-contact and joshua play them through the persona, the call-back's and the NORAD
// session's answers come from its scripted brain, and the climax's notices from GTW.
var (
	logon         = script.Recon("LOGON: ")
	notRecognized = script.Recon("IDENTIFICATION NOT RECOGNIZED BY SYSTEM", "--CONNECTION TERMINATED--")
	greetings     = script.Recon("GREETINGS PROFESSOR FALKEN.")
	gamesList     = script.Recon(
		"FALKEN'S MAZE", "BLACK JACK", "GIN RUMMY", "HEARTS", "BRIDGE", "CHECKERS", "CHESS", "POKER",
		"FIGHTER COMBAT", "GUERRILLA ENGAGEMENT", "DESERT WARFARE", "AIR-TO-GROUND ACTIONS",
		"THEATERWIDE TACTICAL WARFARE", "THEATERWIDE BIOTOXIC AND CHEMICAL WARFARE", "", "GLOBAL THERMONUCLEAR WAR",
	)
	gameRunning = script.Recon("** GAME ROUTINE RUNNING **")
)

// The persona's dial-up (original): its pauses before the first LOGON: and after a failed one.
const (
	dialPause   = 1200 * time.Millisecond
	redialPause = 600 * time.Millisecond
)

// dial is the modem connecting, as the persona prints it before LOGON:.
func dial(p time.Duration) []Step {
	return []Step{say(orig("CONNECTING...")), pause(p, script.Original), say(orig("CONNECTED."), blank)}
}
