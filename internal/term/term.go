package term

import (
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"
)

// KeyKind identifies the type of terminal key event.
type KeyKind int

const (
	KeyNone KeyKind = iota
	KeyRune
	KeyEsc
	KeyEnter
	KeyBackspace
	KeyTab
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyDelete
	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDn
	KeyCtrl
)

// KeyEvent represents a single decoded key press from the terminal.
type KeyEvent struct {
	Kind KeyKind
	Rune rune
	Ctrl byte // lowercase letter for KeyCtrl (e.g. 'r' for Ctrl-R, 'c' for Ctrl-C)
}

// String returns a human-readable representation of the keystroke.
func (k KeyEvent) String() string {
	switch k.Kind {
	case KeyRune:
		if k.Rune == ' ' {
			return "<Space>"
		}
		return string(k.Rune)
	case KeyEsc:
		return "<Esc>"
	case KeyEnter:
		return "<CR>"
	case KeyBackspace:
		return "<BS>"
	case KeyTab:
		return "<Tab>"
	case KeyUp:
		return "<Up>"
	case KeyDown:
		return "<Down>"
	case KeyLeft:
		return "<Left>"
	case KeyRight:
		return "<Right>"
	case KeyDelete:
		return "<Del>"
	case KeyHome:
		return "<Home>"
	case KeyEnd:
		return "<End>"
	case KeyPgUp:
		return "<PgUp>"
	case KeyPgDn:
		return "<PgDn>"
	case KeyCtrl:
		return fmt.Sprintf("<C-%c>", k.Ctrl)
	default:
		return ""
	}
}

// Terminal abstracts raw mode, terminal dimensions, and ANSI screen control.
type Terminal interface {
	EnableRaw() error
	Restore() error
	Size() (cols, rows int, err error)
	Reader() io.Reader
	Writer() io.Writer
	ResizeNotify() <-chan struct{}
}

// KeyReader decodes raw bytes from a terminal into KeyEvents, distinguishing
// standalone Esc from ANSI CSI/SS3 escape sequences without dropping fast keys.
type KeyReader struct {
	ch      chan byte
	errCh   chan error
	pending []byte
}

// NewKeyReader starts a background byte reader over r.
func NewKeyReader(r io.Reader) *KeyReader {
	kr := &KeyReader{
		ch:    make(chan byte, 256),
		errCh: make(chan error, 1),
	}
	go func() {
		buf := make([]byte, 64)
		for {
			n, err := r.Read(buf)
			for i := 0; i < n; i++ {
				kr.ch <- buf[i]
			}
			if err != nil {
				kr.errCh <- err
				return
			}
		}
	}()
	return kr
}

func (kr *KeyReader) nextByteTimeout(d time.Duration) (byte, bool, error) {
	if len(kr.pending) > 0 {
		b := kr.pending[0]
		kr.pending = kr.pending[1:]
		return b, true, nil
	}
	if d <= 0 {
		select {
		case b := <-kr.ch:
			return b, true, nil
		case err := <-kr.errCh:
			return 0, false, err
		}
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case b := <-kr.ch:
		return b, true, nil
	case err := <-kr.errCh:
		return 0, false, err
	case <-timer.C:
		return 0, false, nil
	}
}

func (kr *KeyReader) unread(b ...byte) {
	kr.pending = append(b, kr.pending...)
}

// ReadKey blocks until a KeyEvent is available or an error occurs.
func (kr *KeyReader) ReadKey() (KeyEvent, error) {
	b, ok, err := kr.nextByteTimeout(0)
	if err != nil {
		return KeyEvent{}, err
	}
	if !ok {
		return KeyEvent{}, io.EOF
	}

	switch b {
	case 0x1b:
		return kr.readEscape()
	case '\r', '\n':
		return KeyEvent{Kind: KeyEnter}, nil
	case 0x08, 0x7f:
		return KeyEvent{Kind: KeyBackspace}, nil
	case '\t':
		return KeyEvent{Kind: KeyTab}, nil
	}

	if b >= 0x01 && b <= 0x1a {
		return KeyEvent{Kind: KeyCtrl, Ctrl: 'a' + (b - 1)}, nil
	}

	// Single-byte ASCII printable
	if b < 0x80 {
		return KeyEvent{Kind: KeyRune, Rune: rune(b)}, nil
	}

	// Multi-byte UTF-8 rune
	buf := []byte{b}
	for !utf8.FullRune(buf) && len(buf) < utf8.UTFMax {
		nb, got, err := kr.nextByteTimeout(20 * time.Millisecond)
		if err != nil || !got {
			break
		}
		buf = append(buf, nb)
	}
	r, _ := utf8.DecodeRune(buf)
	if r == utf8.RuneError {
		return KeyEvent{Kind: KeyRune, Rune: rune(b)}, nil
	}
	return KeyEvent{Kind: KeyRune, Rune: r}, nil
}

func (kr *KeyReader) readEscape() (KeyEvent, error) {
	b1, got, err := kr.nextByteTimeout(35 * time.Millisecond)
	if err != nil || !got {
		return KeyEvent{Kind: KeyEsc}, err
	}
	if b1 != '[' && b1 != 'O' {
		// Fast typist pressed Esc followed immediately by a normal key.
		kr.unread(b1)
		return KeyEvent{Kind: KeyEsc}, nil
	}

	if b1 == 'O' {
		b2, got2, err := kr.nextByteTimeout(35 * time.Millisecond)
		if err != nil || !got2 {
			return KeyEvent{Kind: KeyEsc}, err
		}
		switch b2 {
		case 'A':
			return KeyEvent{Kind: KeyUp}, nil
		case 'B':
			return KeyEvent{Kind: KeyDown}, nil
		case 'C':
			return KeyEvent{Kind: KeyRight}, nil
		case 'D':
			return KeyEvent{Kind: KeyLeft}, nil
		case 'H':
			return KeyEvent{Kind: KeyHome}, nil
		case 'F':
			return KeyEvent{Kind: KeyEnd}, nil
		default:
			return KeyEvent{Kind: KeyEsc}, nil
		}
	}

	// CSI sequence: read until final byte in 0x40..0x7E
	var params strings.Builder
	for i := 0; i < 16; i++ {
		c, gotC, err := kr.nextByteTimeout(35 * time.Millisecond)
		if err != nil || !gotC {
			return KeyEvent{Kind: KeyEsc}, err
		}
		if c >= 0x40 && c <= 0x7E {
			return decodeCSI(params.String(), c), nil
		}
		params.WriteByte(c)
	}
	return KeyEvent{Kind: KeyEsc}, nil
}

func decodeCSI(params string, final byte) KeyEvent {
	switch final {
	case 'A':
		return KeyEvent{Kind: KeyUp}
	case 'B':
		return KeyEvent{Kind: KeyDown}
	case 'C':
		return KeyEvent{Kind: KeyRight}
	case 'D':
		return KeyEvent{Kind: KeyLeft}
	case 'H':
		return KeyEvent{Kind: KeyHome}
	case 'F':
		return KeyEvent{Kind: KeyEnd}
	case '~':
		switch params {
		case "1", "7":
			return KeyEvent{Kind: KeyHome}
		case "3":
			return KeyEvent{Kind: KeyDelete}
		case "4", "8":
			return KeyEvent{Kind: KeyEnd}
		case "5":
			return KeyEvent{Kind: KeyPgUp}
		case "6":
			return KeyEvent{Kind: KeyPgDn}
		}
	}
	return KeyEvent{Kind: KeyNone}
}

// ANSI escape sequence helpers.
const (
	AltScreenOn  = "\x1b[?1049h"
	AltScreenOff = "\x1b[?1049l"
	CursorHide   = "\x1b[?25l"
	CursorShow   = "\x1b[?25h"
	CursorBlock  = "\x1b[2 q"
	CursorBar    = "\x1b[6 q"
	ClearScreen  = "\x1b[2J\x1b[H"
	ClearLine    = "\x1b[K"
	ClearBelow   = "\x1b[J"
	Reset        = "\x1b[0m"
	Bold         = "\x1b[1m"
	Dim          = "\x1b[2m"
	Italic       = "\x1b[3m"
	Underline    = "\x1b[4m"
	Reverse      = "\x1b[7m"

	FgBlack   = "\x1b[30m"
	FgRed     = "\x1b[31m"
	FgGreen   = "\x1b[32m"
	FgYellow  = "\x1b[33m"
	FgBlue    = "\x1b[34m"
	FgMagenta = "\x1b[35m"
	FgCyan    = "\x1b[36m"
	FgWhite   = "\x1b[37m"

	FgBrightBlack   = "\x1b[90m"
	FgBrightRed     = "\x1b[91m"
	FgBrightGreen   = "\x1b[92m"
	FgBrightYellow  = "\x1b[93m"
	FgBrightBlue    = "\x1b[94m"
	FgBrightMagenta = "\x1b[95m"
	FgBrightCyan    = "\x1b[96m"
	FgBrightWhite   = "\x1b[97m"

	BgBlue         = "\x1b[44m"
	BgGreen        = "\x1b[42m"
	BgYellow       = "\x1b[43m"
	BgRed          = "\x1b[41m"
	BgMagenta      = "\x1b[45m"
	BgCyan         = "\x1b[46m"
	BgBrightGreen  = "\x1b[102m"
	BgBrightYellow = "\x1b[103m"
)

// MoveTo returns an ANSI cursor-positioning sequence (1-based row, col).
func MoveTo(row, col int) string {
	return fmt.Sprintf("\x1b[%d;%dH", row, col)
}

// Fg256 returns a 256-color foreground ANSI sequence.
func Fg256(code int) string {
	return fmt.Sprintf("\x1b[38;5;%dm", code)
}

// Bg256 returns a 256-color background ANSI sequence.
func Bg256(code int) string {
	return fmt.Sprintf("\x1b[48;5;%dm", code)
}
