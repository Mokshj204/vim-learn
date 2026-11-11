package vim

import (
	"strconv"
	"strings"
	"unicode"

	"vimlearn/internal/term"
)

// HandleKey processes one key event through the Vim state machine.
func (e *Engine) HandleKey(k term.KeyEvent) {
	if k.Kind == term.KeyNone {
		return
	}

	if e.macroDepth == 0 && !e.replayingDot {
		e.KeyCount++
		if label := k.String(); label != "" {
			e.Keystrokes = append(e.Keystrokes, label)
		}
	}

	// If recording a macro and user hits 'q' in clean Normal mode, stop recording before pushing 'q'.
	if e.RecordingMacro != 0 && e.Mode == ModeNormal && e.WaitingSub == "" && e.Operator == 0 && k.Kind == term.KeyRune && k.Rune == 'q' {
		reg := e.RecordingMacro
		copied := make([]term.KeyEvent, len(e.macroBuf))
		copy(copied, e.macroBuf)
		e.Macros[reg] = copied
		e.LastMacro = reg
		e.RecordingMacro = 0
		e.macroBuf = nil
		e.StatusMsg = "Recorded macro @" + string(reg)
		return
	}
	if e.RecordingMacro != 0 && e.macroDepth == 0 {
		e.macroBuf = append(e.macroBuf, k)
	}

	if e.isRecordingChange && !e.replayingDot {
		e.recordingChange = append(e.recordingChange, k)
	}

	switch e.Mode {
	case ModeCommand:
		e.handleCommand(k)
	case ModeInsert:
		e.handleInsert(k)
	case ModeVisual, ModeVisualLine:
		e.handleVisual(k)
	default:
		e.handleNormal(k)
	}
}

func (e *Engine) startRecordingChange(initial ...term.KeyEvent) {
	if e.replayingDot {
		return
	}
	e.isRecordingChange = true
	e.recordingChange = append([]term.KeyEvent{}, initial...)
}

func (e *Engine) finishRecordingChange() {
	if e.replayingDot || !e.isRecordingChange {
		return
	}
	e.isRecordingChange = false
	if len(e.recordingChange) > 0 {
		e.LastChange = make([]term.KeyEvent, len(e.recordingChange))
		copy(e.LastChange, e.recordingChange)
	}
	e.recordingChange = nil
}

func (e *Engine) abortRecordingChange() {
	if e.replayingDot {
		return
	}
	e.isRecordingChange = false
	e.recordingChange = nil
}

func (e *Engine) effectiveCount() int {
	c := e.PendingCount
	if e.OpCount > 0 {
		if c > 0 {
			c = e.OpCount * c
		} else {
			c = e.OpCount
		}
	}
	if c <= 0 {
		return 1
	}
	return c
}

func (e *Engine) clearPending() {
	e.PendingCount = 0
	e.OpCount = 0
	e.Operator = 0
	e.WaitingSub = ""
	e.PendingReg = '"'
}

func (e *Engine) handleCommand(k term.KeyEvent) {
	switch k.Kind {
	case term.KeyEsc:
		e.Mode = ModeNormal
		e.CmdInput = ""
		e.StatusMsg = ""
	case term.KeyCtrl:
		if k.Ctrl == 'c' || k.Ctrl == '[' {
			e.Mode = ModeNormal
			e.CmdInput = ""
			e.StatusMsg = ""
		}
	case term.KeyBackspace:
		runes := []rune(e.CmdInput)
		if len(runes) == 0 {
			e.Mode = ModeNormal
			e.StatusMsg = ""
			return
		}
		e.CmdInput = string(runes[:len(runes)-1])
	case term.KeyEnter:
		cmd := e.CmdInput
		prefix := e.CmdPrefix
		e.Mode = ModeNormal
		e.CmdInput = ""
		e.executeCommand(prefix, cmd)
	case term.KeyRune:
		e.CmdInput += string(k.Rune)
	}
}

func (e *Engine) executeCommand(prefix rune, cmd string) {
	if prefix == '/' || prefix == '?' {
		forward := prefix == '/'
		e.LastSearch = searchState{pattern: cmd, forward: forward}
		pos, ok := e.searchPattern(cmd, forward, 1)
		if !ok {
			e.StatusMsg = "Pattern not found: " + cmd
			return
		}
		e.Cursor = pos
		e.clampCursorNormal()
		e.StatusMsg = string(prefix) + cmd
		return
	}

	trimmed := strings.TrimSpace(cmd)
	if trimmed == "" {
		return
	}
	// Numeric line jump like :5
	if n, err := strconv.Atoi(trimmed); err == nil && n > 0 {
		row := n - 1
		if row >= len(e.Lines) {
			row = len(e.Lines) - 1
		}
		e.Cursor = Position{Row: row, Col: firstNonSpace(e.Lines[row])}
		e.clampCursorNormal()
		return
	}

	switch trimmed {
	case "q", "q!", "quit":
		e.QuitRequested = true
		e.StatusMsg = "Quit requested"
	case "w", "write":
		e.SaveRequested = true
		e.StatusMsg = "Buffer saved"
	case "wq", "x", "wq!":
		e.SaveRequested = true
		e.QuitRequested = true
		e.StatusMsg = "Saved and quit"
	case "e!", "reset":
		e.ResetRequested = true
		e.StatusMsg = "Level reset"
	default:
		e.StatusMsg = "Unknown command: :" + trimmed
	}
}

func (e *Engine) handleInsert(k term.KeyEvent) {
	switch k.Kind {
	case term.KeyEsc:
		e.Mode = ModeNormal
		if e.Cursor.Col > 0 {
			e.Cursor.Col--
		}
		e.clampCursorNormal()
		e.pushSnapshot()
		e.finishRecordingChange()
		e.StatusMsg = ""
	case term.KeyCtrl:
		if k.Ctrl == 'c' || k.Ctrl == '[' {
			e.Mode = ModeNormal
			if e.Cursor.Col > 0 {
				e.Cursor.Col--
			}
			e.clampCursorNormal()
			e.pushSnapshot()
			e.finishRecordingChange()
			e.StatusMsg = ""
		}
	case term.KeyLeft:
		if e.Cursor.Col > 0 {
			e.Cursor.Col--
		}
	case term.KeyRight:
		if e.Cursor.Col < len(e.Lines[e.Cursor.Row]) {
			e.Cursor.Col++
		}
	case term.KeyUp:
		if e.Cursor.Row > 0 {
			e.Cursor.Row--
			e.clampCursorInsert()
		}
	case term.KeyDown:
		if e.Cursor.Row+1 < len(e.Lines) {
			e.Cursor.Row++
			e.clampCursorInsert()
		}
	case term.KeyHome:
		e.Cursor.Col = 0
	case term.KeyEnd:
		e.Cursor.Col = len(e.Lines[e.Cursor.Row])
	case term.KeyBackspace:
		if e.Cursor.Col > 0 {
			line := e.Lines[e.Cursor.Row]
			newLine := make([]rune, 0, len(line)-1)
			newLine = append(newLine, line[:e.Cursor.Col-1]...)
			newLine = append(newLine, line[e.Cursor.Col:]...)
			e.Lines[e.Cursor.Row] = newLine
			e.Cursor.Col--
		} else if e.Cursor.Row > 0 {
			prevLen := len(e.Lines[e.Cursor.Row-1])
			merged := append(e.Lines[e.Cursor.Row-1], e.Lines[e.Cursor.Row]...)
			e.Lines[e.Cursor.Row-1] = merged
			e.Lines = append(e.Lines[:e.Cursor.Row], e.Lines[e.Cursor.Row+1:]...)
			e.Cursor.Row--
			e.Cursor.Col = prevLen
		}
	case term.KeyDelete:
		line := e.Lines[e.Cursor.Row]
		if e.Cursor.Col < len(line) {
			newLine := make([]rune, 0, len(line)-1)
			newLine = append(newLine, line[:e.Cursor.Col]...)
			newLine = append(newLine, line[e.Cursor.Col+1:]...)
			e.Lines[e.Cursor.Row] = newLine
		} else if e.Cursor.Row+1 < len(e.Lines) {
			e.Lines[e.Cursor.Row] = append(line, e.Lines[e.Cursor.Row+1]...)
			e.Lines = append(e.Lines[:e.Cursor.Row+1], e.Lines[e.Cursor.Row+2:]...)
		}
	case term.KeyEnter:
		line := e.Lines[e.Cursor.Row]
		left := make([]rune, e.Cursor.Col)
		copy(left, line[:e.Cursor.Col])
		right := make([]rune, len(line)-e.Cursor.Col)
		copy(right, line[e.Cursor.Col:])
		e.Lines[e.Cursor.Row] = left
		newLines := make([][]rune, 0, len(e.Lines)+1)
		newLines = append(newLines, e.Lines[:e.Cursor.Row+1]...)
		newLines = append(newLines, right)
		newLines = append(newLines, e.Lines[e.Cursor.Row+1:]...)
		e.Lines = newLines
		e.Cursor.Row++
		e.Cursor.Col = 0
	case term.KeyTab:
		e.insertRunes([]rune("  "))
	case term.KeyRune:
		e.insertRunes([]rune{k.Rune})
	}
}

func (e *Engine) insertRunes(rs []rune) {
	line := e.Lines[e.Cursor.Row]
	col := e.Cursor.Col
	if col > len(line) {
		col = len(line)
	}
	newLine := make([]rune, 0, len(line)+len(rs))
	newLine = append(newLine, line[:col]...)
	newLine = append(newLine, rs...)
	newLine = append(newLine, line[col:]...)
	e.Lines[e.Cursor.Row] = newLine
	e.Cursor.Col = col + len(rs)
}

func (e *Engine) handleVisual(k term.KeyEvent) {
	if k.Kind == term.KeyEsc || (k.Kind == term.KeyCtrl && (k.Ctrl == 'c' || k.Ctrl == '[')) {
		e.Mode = ModeNormal
		e.clearPending()
		e.clampCursorNormal()
		e.StatusMsg = ""
		return
	}

	// Waiting sub-states inside Visual mode (e.g. f<c>, g, i/a text objects, r<c>)
	if e.WaitingSub != "" {
		sub := e.WaitingSub
		e.WaitingSub = ""
		if k.Kind != term.KeyRune {
			return
		}
		ch := k.Rune
		switch sub {
		case "f", "F", "t", "T":
			cnt := e.effectiveCount()
			e.PendingCount = 0
			e.LastFind = findState{kind: rune(sub[0]), char: ch}
			if pos, ok := e.findCharOnLine(rune(sub[0]), ch, cnt); ok {
				e.Cursor = pos
			}
		case "g":
			if ch == 'g' {
				e.Cursor = Position{Row: 0, Col: firstNonSpace(e.Lines[0])}
				e.PendingCount = 0
			}
		case "i", "a":
			if rng, ok := e.textObjectRange(ch, sub == "i"); ok {
				e.Mode = ModeVisual
				e.VisualAnchor = rng.Start
				e.Cursor = rng.End
				e.clampCursorNormal()
			}
		case "r":
			rng := e.visualRange()
			e.replaceRangeWithChar(rng, ch)
			e.Mode = ModeNormal
			e.Cursor = rng.Start
			e.clampCursorNormal()
			e.pushSnapshot()
		}
		return
	}

	if k.Kind != term.KeyRune {
		e.applyArrowMotion(k, 1)
		return
	}

	ch := k.Rune
	if ch >= '1' && ch <= '9' || (ch == '0' && e.PendingCount > 0) {
		e.PendingCount = e.PendingCount*10 + int(ch-'0')
		return
	}

	cnt := e.effectiveCount()

	switch ch {
	case 'v':
		if e.Mode == ModeVisual {
			e.Mode = ModeNormal
			e.StatusMsg = ""
		} else {
			e.Mode = ModeVisual
		}
		e.PendingCount = 0
	case 'V':
		if e.Mode == ModeVisualLine {
			e.Mode = ModeNormal
			e.StatusMsg = ""
		} else {
			e.Mode = ModeVisualLine
		}
		e.PendingCount = 0
	case 'o':
		e.VisualAnchor, e.Cursor = e.Cursor, e.VisualAnchor
		e.clampCursorNormal()
		e.PendingCount = 0
	case 'i', 'a':
		e.WaitingSub = string(ch)
	case 'f', 'F', 't', 'T', 'g', 'r':
		e.WaitingSub = string(ch)
	case 'd', 'x':
		rng := e.visualRange()
		e.yankRange(rng)
		e.deleteRange(rng)
		e.Mode = ModeNormal
		e.clearPending()
		e.clampCursorNormal()
		e.pushSnapshot()
	case 'c', 's':
		rng := e.visualRange()
		e.yankRange(rng)
		wasLine := rng.Linewise
		e.deleteRange(rng)
		if wasLine {
			// Leave an empty line to type into
			row := e.Cursor.Row
			if row >= len(e.Lines) {
				e.Lines = append(e.Lines, []rune{})
				row = len(e.Lines) - 1
			} else {
				newLines := make([][]rune, 0, len(e.Lines)+1)
				newLines = append(newLines, e.Lines[:row]...)
				newLines = append(newLines, []rune{})
				newLines = append(newLines, e.Lines[row:]...)
				e.Lines = newLines
			}
			e.Cursor = Position{Row: row, Col: 0}
		}
		e.Mode = ModeInsert
		e.clearPending()
		e.clampCursorInsert()
	case 'y':
		rng := e.visualRange()
		e.yankRange(rng)
		e.Cursor = rng.Start
		e.Mode = ModeNormal
		e.clearPending()
		e.clampCursorNormal()
		e.StatusMsg = "Yanked selection"
	case '~':
		rng := e.visualRange()
		e.toggleCaseRange(rng)
		e.Mode = ModeNormal
		e.Cursor = rng.Start
		e.clearPending()
		e.clampCursorNormal()
		e.pushSnapshot()
	case '>', '<':
		rng := e.visualRange()
		e.indentLines(rng.Start.Row, rng.End.Row, ch == '>', 1)
		e.Mode = ModeNormal
		e.Cursor = Position{Row: rng.Start.Row, Col: firstNonSpace(e.Lines[rng.Start.Row])}
		e.clearPending()
		e.pushSnapshot()
	case 'J':
		rng := e.visualRange()
		if rng.End.Row > rng.Start.Row {
			e.joinLines(rng.Start.Row, rng.End.Row-rng.Start.Row)
		} else {
			e.joinLines(rng.Start.Row, 1)
		}
		e.Mode = ModeNormal
		e.clearPending()
		e.pushSnapshot()
	default:
		e.PendingCount = 0
		_, _ = e.applyMotionKey(ch, cnt)
	}
}

func (e *Engine) visualRange() Range {
	a, b := e.VisualAnchor, e.Cursor
	if comparePos(a, b) > 0 {
		a, b = b, a
	}
	if e.Mode == ModeVisualLine {
		endCol := 0
		if len(e.Lines[b.Row]) > 0 {
			endCol = len(e.Lines[b.Row]) - 1
		}
		return Range{
			Start:     Position{Row: a.Row, Col: 0},
			End:       Position{Row: b.Row, Col: endCol},
			Inclusive: true,
			Linewise:  true,
		}
	}
	return Range{Start: a, End: b, Inclusive: true, Linewise: false}
}

func (e *Engine) handleNormal(k term.KeyEvent) {
	if k.Kind == term.KeyEsc {
		e.clearPending()
		e.abortRecordingChange()
		e.StatusMsg = ""
		return
	}
	if k.Kind == term.KeyCtrl {
		switch k.Ctrl {
		case 'c', '[':
			e.clearPending()
			e.abortRecordingChange()
			e.StatusMsg = ""
		case 'r':
			e.clearPending()
			e.redo()
		}
		return
	}

	if e.WaitingSub != "" {
		e.handleNormalSub(k)
		return
	}

	if k.Kind != term.KeyRune {
		switch k.Kind {
		case term.KeyBackspace:
			e.applyMotionKey('h', e.effectiveCount())
			e.clearPending()
		default:
			e.applyArrowMotion(k, e.effectiveCount())
			e.clearPending()
		}
		return
	}

	ch := k.Rune

	// Numeric counts (0 is a motion unless already building a count)
	if (ch >= '1' && ch <= '9') || (ch == '0' && e.PendingCount > 0) {
		e.PendingCount = e.PendingCount*10 + int(ch-'0')
		return
	}

	// Operator-pending mode
	if e.Operator != 0 {
		e.handleOperatorMotion(ch, k)
		return
	}

	cnt := e.effectiveCount()

	switch ch {
	case '"':
		e.WaitingSub = "\""
	case 'q':
		e.WaitingSub = "q"
	case '@':
		e.WaitingSub = "@"
	case 'm':
		e.WaitingSub = "m"
	case '\'':
		e.WaitingSub = "'"
	case '`':
		e.WaitingSub = "`"
	case 'f', 'F', 't', 'T', 'r', 'g':
		e.WaitingSub = string(ch)
	case ':':
		e.clearPending()
		e.Mode = ModeCommand
		e.CmdPrefix = ':'
		e.CmdInput = ""
	case '/':
		e.clearPending()
		e.Mode = ModeCommand
		e.CmdPrefix = '/'
		e.CmdInput = ""
	case '?':
		e.clearPending()
		e.Mode = ModeCommand
		e.CmdPrefix = '?'
		e.CmdInput = ""
	case 'v':
		e.clearPending()
		e.Mode = ModeVisual
		e.VisualAnchor = e.Cursor
		e.StatusMsg = "-- VISUAL --"
	case 'V':
		e.clearPending()
		e.Mode = ModeVisualLine
		e.VisualAnchor = e.Cursor
		e.StatusMsg = "-- VISUAL LINE --"
	case 'd', 'c', 'y', '>', '<':
		e.Operator = ch
		e.OpCount = e.PendingCount
		e.PendingCount = 0
		if ch == 'd' || ch == 'c' || ch == '>' || ch == '<' {
			e.startRecordingChange(k)
		}
	case 'D':
		e.startRecordingChange(k)
		line := e.Lines[e.Cursor.Row]
		if len(line) > 0 && e.Cursor.Col < len(line) {
			rng := Range{
				Start:     e.Cursor,
				End:       Position{Row: e.Cursor.Row, Col: len(line) - 1},
				Inclusive: true,
			}
			e.yankRange(rng)
			e.deleteRange(rng)
			e.clampCursorNormal()
			e.pushSnapshot()
		}
		e.finishRecordingChange()
		e.clearPending()
	case 'C':
		e.startRecordingChange(k)
		line := e.Lines[e.Cursor.Row]
		if len(line) > 0 && e.Cursor.Col < len(line) {
			rng := Range{
				Start:     e.Cursor,
				End:       Position{Row: e.Cursor.Row, Col: len(line) - 1},
				Inclusive: true,
			}
			e.yankRange(rng)
			e.deleteRange(rng)
		}
		e.Mode = ModeInsert
		e.clampCursorInsert()
		e.clearPending()
	case 'S':
		e.startRecordingChange(k)
		e.changeLines(cnt)
		e.clearPending()
	case 'Y':
		endRow := e.Cursor.Row + cnt - 1
		if endRow >= len(e.Lines) {
			endRow = len(e.Lines) - 1
		}
		rng := Range{
			Start:    Position{Row: e.Cursor.Row, Col: 0},
			End:      Position{Row: endRow, Col: 0},
			Linewise: true,
		}
		e.yankRange(rng)
		e.StatusMsg = "Yanked line"
		e.clearPending()
	case 'x':
		e.startRecordingChange(k)
		line := e.Lines[e.Cursor.Row]
		if len(line) > 0 {
			endCol := e.Cursor.Col + cnt - 1
			if endCol >= len(line) {
				endCol = len(line) - 1
			}
			rng := Range{
				Start:     e.Cursor,
				End:       Position{Row: e.Cursor.Row, Col: endCol},
				Inclusive: true,
			}
			e.yankRange(rng)
			e.deleteRange(rng)
			e.clampCursorNormal()
			e.pushSnapshot()
		}
		e.finishRecordingChange()
		e.clearPending()
	case 'X':
		e.startRecordingChange(k)
		if e.Cursor.Col > 0 {
			startCol := e.Cursor.Col - cnt
			if startCol < 0 {
				startCol = 0
			}
			rng := Range{
				Start:     Position{Row: e.Cursor.Row, Col: startCol},
				End:       Position{Row: e.Cursor.Row, Col: e.Cursor.Col - 1},
				Inclusive: true,
			}
			e.yankRange(rng)
			e.deleteRange(rng)
			e.Cursor.Col = startCol
			e.clampCursorNormal()
			e.pushSnapshot()
		}
		e.finishRecordingChange()
		e.clearPending()
	case 's':
		e.startRecordingChange(k)
		line := e.Lines[e.Cursor.Row]
		if len(line) > 0 {
			endCol := e.Cursor.Col + cnt - 1
			if endCol >= len(line) {
				endCol = len(line) - 1
			}
			rng := Range{
				Start:     e.Cursor,
				End:       Position{Row: e.Cursor.Row, Col: endCol},
				Inclusive: true,
			}
			e.yankRange(rng)
			e.deleteRange(rng)
		}
		e.Mode = ModeInsert
		e.clampCursorInsert()
		e.clearPending()
	case '~':
		e.startRecordingChange(k)
		line := e.Lines[e.Cursor.Row]
		if len(line) > 0 {
			endCol := e.Cursor.Col + cnt - 1
			if endCol >= len(line) {
				endCol = len(line) - 1
			}
			e.toggleCaseRange(Range{
				Start:     e.Cursor,
				End:       Position{Row: e.Cursor.Row, Col: endCol},
				Inclusive: true,
			})
			e.Cursor.Col = endCol + 1
			e.clampCursorNormal()
			e.pushSnapshot()
		}
		e.finishRecordingChange()
		e.clearPending()
	case 'J':
		e.startRecordingChange(k)
		joins := cnt
		if joins == 1 {
			joins = 1
		}
		e.joinLines(e.Cursor.Row, joins)
		e.pushSnapshot()
		e.finishRecordingChange()
		e.clearPending()
	case 'i':
		e.startRecordingChange(k)
		e.Mode = ModeInsert
		e.clampCursorInsert()
		e.clearPending()
	case 'I':
		e.startRecordingChange(k)
		e.Cursor.Col = firstNonSpace(e.Lines[e.Cursor.Row])
		e.Mode = ModeInsert
		e.clampCursorInsert()
		e.clearPending()
	case 'a':
		e.startRecordingChange(k)
		if len(e.Lines[e.Cursor.Row]) > 0 {
			e.Cursor.Col++
		}
		e.Mode = ModeInsert
		e.clampCursorInsert()
		e.clearPending()
	case 'A':
		e.startRecordingChange(k)
		e.Cursor.Col = len(e.Lines[e.Cursor.Row])
		e.Mode = ModeInsert
		e.clampCursorInsert()
		e.clearPending()
	case 'o':
		e.startRecordingChange(k)
		row := e.Cursor.Row + 1
		newLines := make([][]rune, 0, len(e.Lines)+1)
		newLines = append(newLines, e.Lines[:row]...)
		newLines = append(newLines, []rune{})
		newLines = append(newLines, e.Lines[row:]...)
		e.Lines = newLines
		e.Cursor = Position{Row: row, Col: 0}
		e.Mode = ModeInsert
		e.clearPending()
	case 'O':
		e.startRecordingChange(k)
		row := e.Cursor.Row
		newLines := make([][]rune, 0, len(e.Lines)+1)
		newLines = append(newLines, e.Lines[:row]...)
		newLines = append(newLines, []rune{})
		newLines = append(newLines, e.Lines[row:]...)
		e.Lines = newLines
		e.Cursor = Position{Row: row, Col: 0}
		e.Mode = ModeInsert
		e.clearPending()
	case 'p':
		e.startRecordingChange(k)
		e.paste(true, cnt)
		e.pushSnapshot()
		e.finishRecordingChange()
		e.clearPending()
	case 'P':
		e.startRecordingChange(k)
		e.paste(false, cnt)
		e.pushSnapshot()
		e.finishRecordingChange()
		e.clearPending()
	case 'u':
		for i := 0; i < cnt; i++ {
			e.undo()
		}
		e.clearPending()
	case '.':
		e.replayDot(cnt)
		e.clearPending()
	default:
		_, _ = e.applyMotionKey(ch, cnt)
		e.clearPending()
	}
}

func (e *Engine) handleNormalSub(k term.KeyEvent) {
	sub := e.WaitingSub
	e.WaitingSub = ""

	if k.Kind != term.KeyRune {
		e.clearPending()
		e.abortRecordingChange()
		return
	}
	ch := k.Rune

	switch sub {
	case "\"":
		e.PendingReg = ch
		return
	case "q":
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			e.RecordingMacro = ch
			e.macroBuf = nil
			e.StatusMsg = "Recording @" + string(ch)
		}
		e.PendingCount = 0
		return
	case "@":
		cnt := e.effectiveCount()
		e.clearPending()
		reg := ch
		if reg == '@' {
			reg = e.LastMacro
		}
		e.playMacro(reg, cnt)
		return
	case "m":
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			e.Marks[ch] = e.Cursor
			e.StatusMsg = "Mark '" + string(ch) + "' set"
		}
		e.clearPending()
		return
	case "'", "`":
		if pos, ok := e.Marks[ch]; ok {
			if pos.Row >= len(e.Lines) {
				pos.Row = len(e.Lines) - 1
			}
			if sub == "'" {
				pos.Col = firstNonSpace(e.Lines[pos.Row])
			}
			e.Cursor = pos
			e.clampCursorNormal()
		} else {
			e.StatusMsg = "Mark not set: " + string(ch)
		}
		e.clearPending()
		return
	case "r":
		cnt := e.effectiveCount()
		e.startRecordingChange(term.KeyEvent{Kind: term.KeyRune, Rune: 'r'}, k)
		line := e.Lines[e.Cursor.Row]
		if len(line) > 0 && e.Cursor.Col+cnt <= len(line) {
			for i := 0; i < cnt; i++ {
				line[e.Cursor.Col+i] = ch
			}
			if cnt > 1 {
				e.Cursor.Col += cnt - 1
			}
			e.pushSnapshot()
		}
		e.finishRecordingChange()
		e.clearPending()
		return
	case "g":
		if ch == 'g' {
			targetRow := 0
			if e.PendingCount > 0 {
				targetRow = e.PendingCount - 1
				if targetRow >= len(e.Lines) {
					targetRow = len(e.Lines) - 1
				}
			}
			target := Position{Row: targetRow, Col: firstNonSpace(e.Lines[targetRow])}
			if e.Operator != 0 {
				e.applyOperatorLineRange(e.Cursor.Row, target.Row)
				e.clearPending()
				return
			}
			e.Cursor = target
			e.clampCursorNormal()
		}
		e.clearPending()
		return
	case "f", "F", "t", "T":
		kind := rune(sub[0])
		e.LastFind = findState{kind: kind, char: ch}
		cnt := e.effectiveCount()
		pos, ok := e.findCharOnLine(kind, ch, cnt)
		if e.Operator != 0 {
			if ok {
				e.applyOperatorFind(kind, pos)
			} else {
				e.abortRecordingChange()
			}
			e.clearPending()
			return
		}
		if ok {
			e.Cursor = pos
		}
		e.clearPending()
		return
	case "i", "a":
		if e.Operator == 0 {
			e.clearPending()
			return
		}
		rng, ok := e.textObjectRange(ch, sub == "i")
		if !ok {
			e.abortRecordingChange()
			e.clearPending()
			return
		}
		e.applyOperatorOnRange(rng)
		return
	}
}

func (e *Engine) handleOperatorMotion(ch rune, k term.KeyEvent) {
	// Text object introducers inside operator-pending mode (e.g. ciw, da", vi()
	if ch == 'i' || ch == 'a' {
		e.WaitingSub = string(ch)
		return
	}
	if ch == 'f' || ch == 'F' || ch == 't' || ch == 'T' || ch == 'g' {
		e.WaitingSub = string(ch)
		return
	}

	cnt := e.effectiveCount()

	// Doubled operator (dd, cc, yy, >>, <<)
	if ch == e.Operator {
		endRow := e.Cursor.Row + cnt - 1
		if endRow >= len(e.Lines) {
			endRow = len(e.Lines) - 1
		}
		e.applyOperatorLineRange(e.Cursor.Row, endRow)
		e.clearPending()
		return
	}

	switch ch {
	case 'j':
		endRow := e.Cursor.Row + cnt
		if endRow >= len(e.Lines) {
			endRow = len(e.Lines) - 1
		}
		e.applyOperatorLineRange(e.Cursor.Row, endRow)
		e.clearPending()
		return
	case 'k':
		startRow := e.Cursor.Row - cnt
		if startRow < 0 {
			startRow = 0
		}
		e.applyOperatorLineRange(startRow, e.Cursor.Row)
		e.clearPending()
		return
	case 'G':
		targetRow := len(e.Lines) - 1
		if e.PendingCount > 0 {
			targetRow = e.PendingCount - 1
			if targetRow >= len(e.Lines) {
				targetRow = len(e.Lines) - 1
			}
		}
		e.applyOperatorLineRange(e.Cursor.Row, targetRow)
		e.clearPending()
		return
	case '{', '}':
		var dest Position
		if ch == '{' {
			dest = e.paragraphPrev(e.Cursor, cnt)
		} else {
			dest = e.paragraphNext(e.Cursor, cnt)
		}
		e.applyOperatorLineRange(e.Cursor.Row, dest.Row)
		e.clearPending()
		return
	case ';', ',':
		if e.LastFind.char == 0 {
			e.abortRecordingChange()
			e.clearPending()
			return
		}
		kind := e.LastFind.kind
		if ch == ',' {
			kind = flipFind(kind)
		}
		if pos, ok := e.findCharOnLine(kind, e.LastFind.char, cnt); ok {
			e.applyOperatorFind(kind, pos)
		} else {
			e.abortRecordingChange()
		}
		e.clearPending()
		return
	}

	// Characterwise motions for operators
	start := e.Cursor
	var end Position
	inclusive := false
	valid := true

	switch ch {
	case 'h':
		end = start
		end.Col -= cnt
		if end.Col < 0 {
			end.Col = 0
		}
		if end.Col == start.Col {
			valid = false
		} else {
			// Exclusive of start
			rng := Range{Start: end, End: Position{Row: start.Row, Col: start.Col - 1}, Inclusive: true}
			e.applyOperatorOnRange(rng)
			return
		}
	case 'l', ' ':
		line := e.Lines[start.Row]
		if len(line) == 0 {
			valid = false
		} else {
			endCol := start.Col + cnt - 1
			if endCol >= len(line) {
				endCol = len(line) - 1
			}
			end = Position{Row: start.Row, Col: endCol}
			inclusive = true
		}
	case '0':
		if start.Col == 0 {
			valid = false
		} else {
			rng := Range{Start: Position{Row: start.Row, Col: 0}, End: Position{Row: start.Row, Col: start.Col - 1}, Inclusive: true}
			e.applyOperatorOnRange(rng)
			return
		}
	case '^', '_':
		col := firstNonSpace(e.Lines[start.Row])
		if col < start.Col {
			rng := Range{Start: Position{Row: start.Row, Col: col}, End: Position{Row: start.Row, Col: start.Col - 1}, Inclusive: true}
			e.applyOperatorOnRange(rng)
			return
		} else if col > start.Col {
			rng := Range{Start: start, End: Position{Row: start.Row, Col: col - 1}, Inclusive: true}
			e.applyOperatorOnRange(rng)
			return
		}
		valid = false
	case '$':
		line := e.Lines[start.Row]
		if len(line) == 0 {
			valid = false
		} else {
			end = Position{Row: start.Row, Col: len(line) - 1}
			inclusive = true
		}
	case 'w', 'W':
		big := ch == 'W'
		// Vim nuance: `cw`/`cW` on non-whitespace behaves like `ce`/`cE` (keeps trailing spaces).
		if e.Operator == 'c' && start.Row < len(e.Lines) && start.Col < len(e.Lines[start.Row]) && !unicode.IsSpace(e.Lines[start.Row][start.Col]) {
			// If on a single-char word, `ce` would move to next word; handle single-char word cleanly.
			line := e.Lines[start.Row]
			cls := charClass(line[start.Col], big)
			if start.Col+1 >= len(line) || charClass(line[start.Col+1], big) != cls {
				if cnt == 1 {
					end = start
					inclusive = true
				} else {
					end = e.moveWordEnd(start, cnt-1, big)
					inclusive = true
				}
			} else {
				end = e.moveWordEnd(start, cnt, big)
				inclusive = true
			}
		} else {
			next := e.moveWordForward(start, cnt, big)
			if comparePos(next, start) == 0 {
				// At last word on line: delete to end of line
				line := e.Lines[start.Row]
				if len(line) > 0 {
					end = Position{Row: start.Row, Col: len(line) - 1}
					inclusive = true
				} else {
					valid = false
				}
			} else if next.Row > start.Row {
				// Don't cross onto next line for `dw` on last word of line
				line := e.Lines[start.Row]
				if len(line) > 0 {
					end = Position{Row: start.Row, Col: len(line) - 1}
					inclusive = true
				} else {
					valid = false
				}
			} else {
				end = Position{Row: next.Row, Col: next.Col - 1}
				inclusive = true
			}
		}
	case 'e', 'E':
		end = e.moveWordEnd(start, cnt, ch == 'E')
		inclusive = true
	case 'b', 'B':
		prev := e.moveWordBackward(start, cnt, ch == 'B')
		if comparePos(prev, start) >= 0 {
			valid = false
		} else {
			if prev.Row == start.Row {
				rng := Range{Start: prev, End: Position{Row: start.Row, Col: start.Col - 1}, Inclusive: true}
				e.applyOperatorOnRange(rng)
				return
			}
			// Cross-line backward
			prevStep, ok := e.stepBackward(start)
			if !ok {
				valid = false
			} else {
				e.applyOperatorOnRange(Range{Start: prev, End: prevStep, Inclusive: true})
				return
			}
		}
	case '%':
		match, ok := e.matchBracket(start)
		if !ok {
			valid = false
		} else {
			if comparePos(match, start) < 0 {
				e.applyOperatorOnRange(Range{Start: match, End: start, Inclusive: true})
				return
			}
			end = match
			inclusive = true
		}
	default:
		valid = false
	}

	if !valid {
		e.abortRecordingChange()
		e.clearPending()
		return
	}
	e.applyOperatorOnRange(Range{Start: start, End: end, Inclusive: inclusive})
}

func (e *Engine) applyOperatorFind(kind rune, pos Position) {
	forward := kind == 'f' || kind == 't'
	if forward {
		e.applyOperatorOnRange(Range{Start: e.Cursor, End: pos, Inclusive: true})
		return
	}
	if pos.Col >= e.Cursor.Col {
		e.abortRecordingChange()
		return
	}
	e.applyOperatorOnRange(Range{
		Start:     pos,
		End:       Position{Row: e.Cursor.Row, Col: e.Cursor.Col - 1},
		Inclusive: true,
	})
}

func (e *Engine) applyOperatorLineRange(r1, r2 int) {
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	if r1 < 0 {
		r1 = 0
	}
	if r2 >= len(e.Lines) {
		r2 = len(e.Lines) - 1
	}
	rng := Range{
		Start:    Position{Row: r1, Col: 0},
		End:      Position{Row: r2, Col: 0},
		Linewise: true,
	}
	switch e.Operator {
	case 'y':
		e.yankRange(rng)
		e.Cursor = Position{Row: r1, Col: e.Cursor.Col}
		e.clampCursorNormal()
		e.StatusMsg = "Yanked lines"
	case 'd':
		e.yankRange(rng)
		e.deleteRange(rng)
		e.clampCursorNormal()
		e.pushSnapshot()
		e.finishRecordingChange()
	case 'c':
		e.yankRange(rng)
		e.changeLinesRange(r1, r2)
	case '>':
		e.indentLines(r1, r2, true, 1)
		e.Cursor = Position{Row: r1, Col: firstNonSpace(e.Lines[r1])}
		e.pushSnapshot()
		e.finishRecordingChange()
	case '<':
		e.indentLines(r1, r2, false, 1)
		e.Cursor = Position{Row: r1, Col: firstNonSpace(e.Lines[r1])}
		e.pushSnapshot()
		e.finishRecordingChange()
	}
}

func (e *Engine) changeLines(cnt int) {
	endRow := e.Cursor.Row + cnt - 1
	if endRow >= len(e.Lines) {
		endRow = len(e.Lines) - 1
	}
	e.changeLinesRange(e.Cursor.Row, endRow)
}

func (e *Engine) changeLinesRange(r1, r2 int) {
	e.Lines[r1] = []rune{}
	if r2 > r1 {
		e.Lines = append(e.Lines[:r1+1], e.Lines[r2+1:]...)
	}
	e.Cursor = Position{Row: r1, Col: 0}
	e.Mode = ModeInsert
}

func (e *Engine) applyOperatorOnRange(rng Range) {
	op := e.Operator
	e.clearPending()

	switch op {
	case 'y':
		e.yankRange(rng)
		e.Cursor = rng.Start
		e.clampCursorNormal()
		e.StatusMsg = "Yanked"
	case 'd':
		e.yankRange(rng)
		e.deleteRange(rng)
		e.clampCursorNormal()
		e.pushSnapshot()
		e.finishRecordingChange()
	case 'c':
		e.yankRange(rng)
		e.deleteRange(rng)
		e.Mode = ModeInsert
		e.clampCursorInsert()
	case '>':
		e.indentLines(rng.Start.Row, rng.End.Row, true, 1)
		e.pushSnapshot()
		e.finishRecordingChange()
	case '<':
		e.indentLines(rng.Start.Row, rng.End.Row, false, 1)
		e.pushSnapshot()
		e.finishRecordingChange()
	}
}

func (e *Engine) yankRange(rng Range) {
	reg := e.PendingReg
	if reg == 0 {
		reg = '"'
	}
	var val RegisterValue
	if rng.Linewise {
		parts := make([]string, 0, rng.End.Row-rng.Start.Row+1)
		for r := rng.Start.Row; r <= rng.End.Row && r < len(e.Lines); r++ {
			parts = append(parts, string(e.Lines[r]))
		}
		val = RegisterValue{Text: strings.Join(parts, "\n"), Linewise: true}
	} else if comparePos(rng.Start, rng.End) > 0 {
		val = RegisterValue{Text: "", Linewise: false}
	} else {
		var b strings.Builder
		for r := rng.Start.Row; r <= rng.End.Row && r < len(e.Lines); r++ {
			line := e.Lines[r]
			cStart := 0
			if r == rng.Start.Row {
				cStart = rng.Start.Col
			}
			cEnd := len(line) - 1
			if r == rng.End.Row {
				cEnd = rng.End.Col
				if !rng.Inclusive {
					cEnd--
				}
			}
			if cStart < 0 {
				cStart = 0
			}
			if cEnd >= len(line) {
				cEnd = len(line) - 1
			}
			if cStart <= cEnd {
				b.WriteString(string(line[cStart : cEnd+1]))
			}
			if r < rng.End.Row {
				b.WriteByte('\n')
			}
		}
		val = RegisterValue{Text: b.String(), Linewise: false}
	}
	e.Registers[reg] = val
	e.Registers['"'] = val
}

func (e *Engine) deleteRange(rng Range) {
	if rng.Linewise {
		r1, r2 := rng.Start.Row, rng.End.Row
		if r1 < 0 {
			r1 = 0
		}
		if r2 >= len(e.Lines) {
			r2 = len(e.Lines) - 1
		}
		newLines := make([][]rune, 0, len(e.Lines)-(r2-r1+1))
		newLines = append(newLines, e.Lines[:r1]...)
		newLines = append(newLines, e.Lines[r2+1:]...)
		if len(newLines) == 0 {
			newLines = [][]rune{{}}
		}
		e.Lines = newLines
		row := r1
		if row >= len(e.Lines) {
			row = len(e.Lines) - 1
		}
		e.Cursor = Position{Row: row, Col: firstNonSpace(e.Lines[row])}
		return
	}

	if comparePos(rng.Start, rng.End) > 0 {
		e.Cursor = rng.Start
		return
	}

	start := rng.Start
	end := rng.End
	if !rng.Inclusive {
		if end.Col > 0 {
			end.Col--
		} else if end.Row > 0 {
			end.Row--
			end.Col = len(e.Lines[end.Row]) - 1
		}
	}

	if start.Row == end.Row {
		line := e.Lines[start.Row]
		if len(line) == 0 {
			e.Cursor = start
			return
		}
		c1 := start.Col
		c2 := end.Col + 1
		if c1 < 0 {
			c1 = 0
		}
		if c2 > len(line) {
			c2 = len(line)
		}
		if c1 < c2 {
			newLine := make([]rune, 0, len(line)-(c2-c1))
			newLine = append(newLine, line[:c1]...)
			newLine = append(newLine, line[c2:]...)
			e.Lines[start.Row] = newLine
		}
		e.Cursor = Position{Row: start.Row, Col: c1}
		return
	}

	// Multi-line characterwise deletion
	firstLine := e.Lines[start.Row]
	lastLine := e.Lines[end.Row]
	prefix := firstLine[:minInt(start.Col, len(firstLine))]
	suffixStart := end.Col + 1
	if suffixStart > len(lastLine) {
		suffixStart = len(lastLine)
	}
	suffix := lastLine[suffixStart:]

	merged := make([]rune, 0, len(prefix)+len(suffix))
	merged = append(merged, prefix...)
	merged = append(merged, suffix...)

	newLines := make([][]rune, 0, len(e.Lines)-(end.Row-start.Row))
	newLines = append(newLines, e.Lines[:start.Row]...)
	newLines = append(newLines, merged)
	newLines = append(newLines, e.Lines[end.Row+1:]...)
	e.Lines = newLines
	e.Cursor = Position{Row: start.Row, Col: len(prefix)}
}

func (e *Engine) paste(after bool, count int) {
	reg := e.PendingReg
	if reg == 0 {
		reg = '"'
	}
	val, ok := e.Registers[reg]
	if !ok || (val.Text == "" && !val.Linewise) {
		return
	}

	if val.Linewise {
		pasted := splitLines(val.Text)
		var all [][]rune
		for c := 0; c < count; c++ {
			all = append(all, cloneLines(pasted)...)
		}
		insertAt := e.Cursor.Row
		if after {
			insertAt++
		}
		newLines := make([][]rune, 0, len(e.Lines)+len(all))
		newLines = append(newLines, e.Lines[:insertAt]...)
		newLines = append(newLines, all...)
		newLines = append(newLines, e.Lines[insertAt:]...)
		e.Lines = newLines
		e.Cursor = Position{Row: insertAt, Col: firstNonSpace(e.Lines[insertAt])}
		e.clampCursorNormal()
		return
	}

	repeated := strings.Repeat(val.Text, count)
	rs := []rune(repeated)
	line := e.Lines[e.Cursor.Row]
	col := e.Cursor.Col
	if after && len(line) > 0 {
		col++
	}
	if col > len(line) {
		col = len(line)
	}
	newLine := make([]rune, 0, len(line)+len(rs))
	newLine = append(newLine, line[:col]...)
	newLine = append(newLine, rs...)
	newLine = append(newLine, line[col:]...)
	e.Lines[e.Cursor.Row] = newLine
	if len(rs) > 0 {
		e.Cursor.Col = col + len(rs) - 1
	} else {
		e.Cursor.Col = col
	}
	e.clampCursorNormal()
}

func (e *Engine) toggleCaseRange(rng Range) {
	for r := rng.Start.Row; r <= rng.End.Row && r < len(e.Lines); r++ {
		line := e.Lines[r]
		c1, c2 := 0, len(line)-1
		if !rng.Linewise {
			if r == rng.Start.Row {
				c1 = rng.Start.Col
			}
			if r == rng.End.Row {
				c2 = rng.End.Col
			}
		}
		for c := c1; c <= c2 && c < len(line); c++ {
			if unicode.IsLower(line[c]) {
				line[c] = unicode.ToUpper(line[c])
			} else if unicode.IsUpper(line[c]) {
				line[c] = unicode.ToLower(line[c])
			}
		}
	}
}

func (e *Engine) replaceRangeWithChar(rng Range, ch rune) {
	for r := rng.Start.Row; r <= rng.End.Row && r < len(e.Lines); r++ {
		line := e.Lines[r]
		c1, c2 := 0, len(line)-1
		if !rng.Linewise {
			if r == rng.Start.Row {
				c1 = rng.Start.Col
			}
			if r == rng.End.Row {
				c2 = rng.End.Col
			}
		}
		for c := c1; c <= c2 && c < len(line); c++ {
			line[c] = ch
		}
	}
}

func (e *Engine) indentLines(r1, r2 int, indent bool, count int) {
	if r1 > r2 {
		r1, r2 = r2, r1
	}
	spaces := strings.Repeat("  ", count)
	for r := r1; r <= r2 && r < len(e.Lines); r++ {
		if indent {
			if len(e.Lines[r]) > 0 {
				e.Lines[r] = append([]rune(spaces), e.Lines[r]...)
			}
		} else {
			removed := 0
			line := e.Lines[r]
			for removed < 2*count && len(line) > 0 && line[0] == ' ' {
				line = line[1:]
				removed++
			}
			e.Lines[r] = line
		}
	}
}

func (e *Engine) joinLines(row, count int) {
	for i := 0; i < count; i++ {
		if row+1 >= len(e.Lines) {
			break
		}
		cur := e.Lines[row]
		nextTrimmed := strings.TrimLeft(string(e.Lines[row+1]), " \t")
		joinedCol := len(cur)
		if len(cur) > 0 && len(nextTrimmed) > 0 {
			cur = append(cur, ' ')
		}
		cur = append(cur, []rune(nextTrimmed)...)
		e.Lines[row] = cur
		e.Lines = append(e.Lines[:row+1], e.Lines[row+2:]...)
		e.Cursor = Position{Row: row, Col: joinedCol}
		e.clampCursorNormal()
	}
}

func (e *Engine) applyArrowMotion(k term.KeyEvent, count int) {
	switch k.Kind {
	case term.KeyLeft:
		_, _ = e.applyMotionKey('h', count)
	case term.KeyDown:
		_, _ = e.applyMotionKey('j', count)
	case term.KeyUp:
		_, _ = e.applyMotionKey('k', count)
	case term.KeyRight:
		_, _ = e.applyMotionKey('l', count)
	case term.KeyHome:
		_, _ = e.applyMotionKey('0', 1)
	case term.KeyEnd:
		_, _ = e.applyMotionKey('$', 1)
	}
}

func (e *Engine) applyMotionKey(ch rune, count int) (bool, bool) {
	switch ch {
	case 'h':
		e.Cursor.Col -= count
		e.clampCursorNormal()
		return true, false
	case 'l', ' ':
		e.Cursor.Col += count
		e.clampCursorNormal()
		return true, false
	case 'j':
		e.Cursor.Row += count
		e.clampCursorNormal()
		return true, true
	case 'k':
		e.Cursor.Row -= count
		e.clampCursorNormal()
		return true, true
	case '0':
		e.Cursor.Col = 0
		return true, false
	case '^', '_':
		e.Cursor.Col = firstNonSpace(e.Lines[e.Cursor.Row])
		return true, false
	case '$':
		lineLen := len(e.Lines[e.Cursor.Row])
		if lineLen > 0 {
			e.Cursor.Col = lineLen - 1
		} else {
			e.Cursor.Col = 0
		}
		return true, false
	case 'w', 'W':
		e.Cursor = e.moveWordForward(e.Cursor, count, ch == 'W')
		e.clampCursorNormal()
		return true, false
	case 'b', 'B':
		e.Cursor = e.moveWordBackward(e.Cursor, count, ch == 'B')
		e.clampCursorNormal()
		return true, false
	case 'e', 'E':
		e.Cursor = e.moveWordEnd(e.Cursor, count, ch == 'E')
		e.clampCursorNormal()
		return true, false
	case 'G':
		row := len(e.Lines) - 1
		if e.PendingCount > 0 {
			row = e.PendingCount - 1
			if row >= len(e.Lines) {
				row = len(e.Lines) - 1
			}
		}
		e.Cursor = Position{Row: row, Col: firstNonSpace(e.Lines[row])}
		e.clampCursorNormal()
		return true, true
	case '{':
		e.Cursor = e.paragraphPrev(e.Cursor, count)
		e.clampCursorNormal()
		return true, true
	case '}':
		e.Cursor = e.paragraphNext(e.Cursor, count)
		e.clampCursorNormal()
		return true, true
	case '%':
		if pos, ok := e.matchBracket(e.Cursor); ok {
			e.Cursor = pos
		}
		return true, false
	case ';':
		if e.LastFind.char != 0 {
			if pos, ok := e.findCharOnLine(e.LastFind.kind, e.LastFind.char, count); ok {
				e.Cursor = pos
			}
		}
		return true, false
	case ',':
		if e.LastFind.char != 0 {
			if pos, ok := e.findCharOnLine(flipFind(e.LastFind.kind), e.LastFind.char, count); ok {
				e.Cursor = pos
			}
		}
		return true, false
	case 'n', 'N':
		if e.LastSearch.pattern != "" {
			fwd := e.LastSearch.forward
			if ch == 'N' {
				fwd = !fwd
			}
			if pos, ok := e.searchPattern(e.LastSearch.pattern, fwd, count); ok {
				e.Cursor = pos
				e.clampCursorNormal()
			} else {
				e.StatusMsg = "Pattern not found: " + e.LastSearch.pattern
			}
		}
		return true, false
	case '*', '#':
		word := e.wordUnderCursor()
		if word != "" {
			fwd := ch == '*'
			e.LastSearch = searchState{pattern: word, forward: fwd}
			if pos, ok := e.searchPattern(word, fwd, count); ok {
				e.Cursor = pos
				e.clampCursorNormal()
			}
		}
		return true, false
	}
	return false, false
}

func (e *Engine) playMacro(reg rune, count int) {
	seq, ok := e.Macros[reg]
	if !ok || len(seq) == 0 {
		e.StatusMsg = "Empty macro @" + string(reg)
		return
	}
	if e.macroDepth >= 8 {
		e.StatusMsg = "Macro recursion limit"
		return
	}
	e.LastMacro = reg
	e.macroDepth++
	defer func() { e.macroDepth-- }()
	for c := 0; c < count; c++ {
		for _, ev := range seq {
			e.HandleKey(ev)
		}
	}
}

func (e *Engine) replayDot(count int) {
	if len(e.LastChange) == 0 {
		return
	}
	e.replayingDot = true
	defer func() { e.replayingDot = false }()
	for c := 0; c < count; c++ {
		for _, ev := range e.LastChange {
			e.HandleKey(ev)
		}
	}
}

func flipFind(kind rune) rune {
	switch kind {
	case 'f':
		return 'F'
	case 'F':
		return 'f'
	case 't':
		return 'T'
	case 'T':
		return 't'
	default:
		return kind
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
