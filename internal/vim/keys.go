package vim

import (
	"unicode/utf8"

	"vimlearn/internal/term"
)

// ParseKeySequence parses a Vim-notation string (e.g. "ciwfoo<Esc>j") into KeyEvents.
func ParseKeySequence(seq string) []term.KeyEvent {
	var out []term.KeyEvent
	for i := 0; i < len(seq); {
		if seq[i] == '<' {
			end := -1
			for j := i + 1; j < len(seq) && j <= i+12; j++ {
				if seq[j] == '>' {
					end = j
					break
				}
			}
			if end != -1 {
				tok := seq[i+1 : end]
				if ev, ok := specialToken(tok); ok {
					out = append(out, ev)
					i = end + 1
					continue
				}
			}
		}
		r, w := utf8.DecodeRuneInString(seq[i:])
		out = append(out, term.KeyEvent{Kind: term.KeyRune, Rune: r})
		i += w
	}
	return out
}

func specialToken(tok string) (term.KeyEvent, bool) {
	switch tok {
	case "Esc", "ESC":
		return term.KeyEvent{Kind: term.KeyEsc}, true
	case "CR", "Enter":
		return term.KeyEvent{Kind: term.KeyEnter}, true
	case "BS":
		return term.KeyEvent{Kind: term.KeyBackspace}, true
	case "Tab":
		return term.KeyEvent{Kind: term.KeyTab}, true
	case "Space":
		return term.KeyEvent{Kind: term.KeyRune, Rune: ' '}, true
	case "Up":
		return term.KeyEvent{Kind: term.KeyUp}, true
	case "Down":
		return term.KeyEvent{Kind: term.KeyDown}, true
	case "Left":
		return term.KeyEvent{Kind: term.KeyLeft}, true
	case "Right":
		return term.KeyEvent{Kind: term.KeyRight}, true
	case "C-r":
		return term.KeyEvent{Kind: term.KeyCtrl, Ctrl: 'r'}, true
	case "C-c":
		return term.KeyEvent{Kind: term.KeyCtrl, Ctrl: 'c'}, true
	}
	return term.KeyEvent{}, false
}

// Feed executes every key event parsed from a Vim-notation sequence.
func (e *Engine) Feed(seq string) {
	for _, ev := range ParseKeySequence(seq) {
		e.HandleKey(ev)
	}
}
