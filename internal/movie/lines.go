package movie

import "github.com/GhostofGoes/WOPR/internal/script"

// The director's own text: the scene menu and the pause notice (original).
var (
	lineMenuTitle = script.Orig("MOVIE MODE", "", "THE FILM'S WOPR TERMINAL SCENES:", "")
	lineMenuKeys  = script.Orig(
		"",
		"SPACE PAUSES. N OR RIGHT ARROW: NEXT SCENE. P OR LEFT ARROW: PREVIOUS SCENE.",
		"ESC COMES BACK HERE, BUT ONCE THE CLIMAX'S GAMES BEGIN THEY PLAY TO THE END.",
		"A SCENE PLAYS ON TO THE END OF THE LIST. CTRL+C QUITS AT ANY TIME.",
		"",
		"TYPE A SCENE'S NUMBER OR NAME, OR Q TO LEAVE.",
	)
	promptScene   = script.Orig("SCENE: ")
	lineNoScene   = script.Orig("NO SUCH SCENE.")
	lineAmbiguous = script.Orig("THAT NAME FITS MORE THAN ONE SCENE:")
	linePaused    = script.Orig("** PAUSED **   SPACE RESUMES. ESC FOR THE MENU.")
)

// Lines is every block of the director's text, for the provenance test.
var Lines = []script.Ls{lineMenuTitle, lineMenuKeys, promptScene, lineNoScene, lineAmbiguous, linePaused}
