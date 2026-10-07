package proto

// Key is a key a program can receive in key mode. Esc is not a Key: the host owns it.
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
	default:
		return "unknown"
	}
}
