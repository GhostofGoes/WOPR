package proto

// Key is a key a program can receive in key mode. Esc belongs to the host: only a root program
// that captures keys (AwaitKeys.Capture, movie mode) receives KeyEsc.
type Key uint8

// Keys.
const (
	KeyRune Key = iota // a printable character; see KeyEvent.Rune (Space arrives as ' ')
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyEnter
	KeyBackspace
	KeyEsc // only for a capturing root program
)

func (k Key) String() string {
	switch k {
	case KeyRune:
		return "rune"
	case KeyUp:
		return "up"
	case KeyDown:
		return "down"
	case KeyLeft:
		return "left"
	case KeyRight:
		return "right"
	case KeyEnter:
		return "enter"
	case KeyBackspace:
		return "backspace"
	case KeyEsc:
		return "esc"
	default:
		return "unknown"
	}
}
