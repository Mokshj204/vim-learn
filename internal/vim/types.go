package vim

import (
	"strings"

	"vimlearn/internal/term"
)

// Mode represents the active Vim editing mode.
type Mode int

const (
	ModeNormal Mode = iota
	ModeInsert
	ModeVisual
	ModeVisualLine
	ModeCommand
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "NORMAL"
	case ModeInsert:
		return "INSERT"
	case ModeVisual:
		return "VISUAL"
	case ModeVisualLine:
		return "V-LINE"
	case ModeCommand:
		return "COMMAND"
	default:
		return "NORMAL"
	}
}

// Position is a 0-based (Row, Col) coordinate in the text buffer.
type Position struct {
	Row int
	Col int
}

// Range represents an editor span between Start and End.
type Range struct {
	Start     Position
	End       Position
	Inclusive bool
	Linewise  bool
}

// RegisterValue stores yanked or deleted text.
type RegisterValue struct {
	Text     string
	Linewise bool
}

type snapshot struct {
	lines  [][]rune
	cursor Position
}

type findState struct {
	kind rune // 'f', 'F', 't', 'T'
	char rune
}

type searchState struct {
	pattern string
	forward bool
}

// Engine implements a self-contained modal Vim editor for puzzle & tutorial levels.
type Engine struct {
	Lines        [][]rune
	Cursor       Position
	VisualAnchor Position
	Mode         Mode

	PendingCount int
	Operator     rune
	OpCount      int
	WaitingSub   string // "", "f", "F", "t", "T", "r", "g", "m", "'", "`", "\"", "q", "@", "i", "a"
	PendingReg   rune
	Registers    map[rune]RegisterValue
	Marks        map[rune]Position
	LastFind     findState
	LastSearch   searchState

	CmdPrefix rune // ':', '/', '?'
	CmdInput  string

	history    []snapshot
	historyIdx int

	RecordingMacro rune
	macroBuf       []term.KeyEvent
	Macros         map[rune][]term.KeyEvent
	LastMacro      rune
	macroDepth     int

	LastChange        []term.KeyEvent
	recordingChange   []term.KeyEvent
	isRecordingChange bool
	replayingDot      bool

	StatusMsg      string
	Keystrokes     []string
	KeyCount       int
	QuitRequested  bool
	SaveRequested  bool
	ResetRequested bool
	HelpRequested  bool
}

// NewEngine initializes a Vim engine with the given initial lines and cursor position.
func NewEngine(text string, cursor Position) *Engine {
	lines := splitLines(text)
	e := &Engine{
		Lines:      lines,
		Cursor:     cursor,
		Mode:       ModeNormal,
		PendingReg: '"',
		Registers:  make(map[rune]RegisterValue),
		Marks:      make(map[rune]Position),
		Macros:     make(map[rune][]term.KeyEvent),
	}
	e.clampCursorNormal()
	e.pushSnapshot()
	return e
}

func splitLines(text string) [][]rune {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	raw := strings.Split(text, "\n")
	if len(raw) == 0 {
		return [][]rune{{}}
	}
	out := make([][]rune, len(raw))
	for i, s := range raw {
		out[i] = []rune(s)
	}
	return out
}

func cloneLines(lines [][]rune) [][]rune {
	out := make([][]rune, len(lines))
	for i, l := range lines {
		cp := make([]rune, len(l))
		copy(cp, l)
		out[i] = cp
	}
	return out
}

// Text returns the buffer contents joined with '\n'.
func (e *Engine) Text() string {
	parts := make([]string, len(e.Lines))
	for i, l := range e.Lines {
		parts[i] = string(l)
	}
	return strings.Join(parts, "\n")
}

// LineStrings returns the buffer as a slice of strings.
func (e *Engine) LineStrings() []string {
	parts := make([]string, len(e.Lines))
	for i, l := range e.Lines {
		parts[i] = string(l)
	}
	return parts
}

func (e *Engine) pushSnapshot() {
	if e.historyIdx < len(e.history)-1 {
		e.history = e.history[:e.historyIdx+1]
	}
	e.history = append(e.history, snapshot{
		lines:  cloneLines(e.Lines),
		cursor: e.Cursor,
	})
	e.historyIdx = len(e.history) - 1
}

func (e *Engine) undo() {
	if e.historyIdx <= 0 {
		e.StatusMsg = "Already at oldest change"
		return
	}
	e.historyIdx--
	snap := e.history[e.historyIdx]
	e.Lines = cloneLines(snap.lines)
	e.Cursor = snap.cursor
	e.clampCursorNormal()
	e.StatusMsg = "Undo"
}

func (e *Engine) redo() {
	if e.historyIdx >= len(e.history)-1 {
		e.StatusMsg = "Already at newest change"
		return
	}
	e.historyIdx++
	snap := e.history[e.historyIdx]
	e.Lines = cloneLines(snap.lines)
	e.Cursor = snap.cursor
	e.clampCursorNormal()
	e.StatusMsg = "Redo"
}

func (e *Engine) clampCursorNormal() {
	if len(e.Lines) == 0 {
		e.Lines = [][]rune{{}}
	}
	if e.Cursor.Row < 0 {
		e.Cursor.Row = 0
	}
	if e.Cursor.Row >= len(e.Lines) {
		e.Cursor.Row = len(e.Lines) - 1
	}
	lineLen := len(e.Lines[e.Cursor.Row])
	if lineLen == 0 {
		e.Cursor.Col = 0
	} else {
		if e.Cursor.Col < 0 {
			e.Cursor.Col = 0
		}
		if e.Cursor.Col >= lineLen {
			e.Cursor.Col = lineLen - 1
		}
	}
}

func (e *Engine) clampCursorInsert() {
	if len(e.Lines) == 0 {
		e.Lines = [][]rune{{}}
	}
	if e.Cursor.Row < 0 {
		e.Cursor.Row = 0
	}
	if e.Cursor.Row >= len(e.Lines) {
		e.Cursor.Row = len(e.Lines) - 1
	}
	lineLen := len(e.Lines[e.Cursor.Row])
	if e.Cursor.Col < 0 {
		e.Cursor.Col = 0
	}
	if e.Cursor.Col > lineLen {
		e.Cursor.Col = lineLen
	}
}

// PendingSummary returns the partially entered command string (e.g. "3d", "ci").
func (e *Engine) PendingSummary() string {
	var b strings.Builder
	if e.PendingReg != '"' && e.PendingReg != 0 {
		b.WriteByte('"')
		b.WriteRune(e.PendingReg)
	}
	if e.OpCount > 0 {
		b.WriteString(itoa(e.OpCount))
	}
	if e.Operator != 0 {
		b.WriteRune(e.Operator)
	}
	if e.PendingCount > 0 {
		b.WriteString(itoa(e.PendingCount))
	}
	if e.WaitingSub != "" {
		b.WriteString(e.WaitingSub)
	}
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}
