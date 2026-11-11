package vim

import (
	"strings"
	"unicode"
)

func charClass(r rune, bigWord bool) int {
	if unicode.IsSpace(r) {
		return 0
	}
	if bigWord {
		return 1
	}
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		return 1
	}
	return 2
}

func firstNonSpace(line []rune) int {
	for i, r := range line {
		if !unicode.IsSpace(r) {
			return i
		}
	}
	return 0
}

func (e *Engine) stepForward(p Position) (Position, bool) {
	if p.Row >= len(e.Lines) {
		return p, false
	}
	lineLen := len(e.Lines[p.Row])
	if p.Col+1 < lineLen {
		return Position{Row: p.Row, Col: p.Col + 1}, true
	}
	if p.Row+1 < len(e.Lines) {
		return Position{Row: p.Row + 1, Col: 0}, true
	}
	return p, false
}

func (e *Engine) stepBackward(p Position) (Position, bool) {
	if p.Col > 0 {
		return Position{Row: p.Row, Col: p.Col - 1}, true
	}
	if p.Row > 0 {
		prevLen := len(e.Lines[p.Row-1])
		col := 0
		if prevLen > 0 {
			col = prevLen - 1
		}
		return Position{Row: p.Row - 1, Col: col}, true
	}
	return p, false
}

// moveWordForward implements `w` and `W`.
func (e *Engine) moveWordForward(from Position, count int, bigWord bool) Position {
	p := from
	for c := 0; c < count; c++ {
		if p.Row >= len(e.Lines) {
			break
		}
		line := e.Lines[p.Row]
		if len(line) == 0 {
			if p.Row+1 < len(e.Lines) {
				p = Position{Row: p.Row + 1, Col: firstNonSpace(e.Lines[p.Row+1])}
			}
			continue
		}
		startClass := charClass(line[p.Col], bigWord)
		movedLine := false
		if startClass != 0 {
			for p.Col < len(line) && charClass(line[p.Col], bigWord) == startClass {
				p.Col++
			}
		}
		// Skip whitespace, crossing lines as needed
		for {
			line = e.Lines[p.Row]
			for p.Col < len(line) && unicode.IsSpace(line[p.Col]) {
				p.Col++
			}
			if p.Col < len(line) {
				break
			}
			if p.Row+1 >= len(e.Lines) {
				// Reached end of buffer
				if len(line) > 0 {
					p.Col = len(line) - 1
				} else {
					p.Col = 0
				}
				break
			}
			p.Row++
			p.Col = 0
			movedLine = true
			if len(e.Lines[p.Row]) == 0 && movedLine {
				// Empty line counts as a word stop in Vim
				break
			}
		}
	}
	return p
}

// moveWordEnd implements `e` and `E`.
func (e *Engine) moveWordEnd(from Position, count int, bigWord bool) Position {
	p := from
	for c := 0; c < count; c++ {
		next, ok := e.stepForward(p)
		if !ok {
			break
		}
		p = next
		// Skip whitespace across lines
		for {
			line := e.Lines[p.Row]
			if len(line) == 0 {
				if p.Row+1 < len(e.Lines) {
					p = Position{Row: p.Row + 1, Col: 0}
					continue
				}
				break
			}
			for p.Col < len(line) && unicode.IsSpace(line[p.Col]) {
				p.Col++
			}
			if p.Col < len(line) {
				break
			}
			if p.Row+1 >= len(e.Lines) {
				p.Col = len(line) - 1
				return p
			}
			p = Position{Row: p.Row + 1, Col: 0}
		}
		line := e.Lines[p.Row]
		if len(line) == 0 {
			continue
		}
		cls := charClass(line[p.Col], bigWord)
		for p.Col+1 < len(line) && charClass(line[p.Col+1], bigWord) == cls {
			p.Col++
		}
	}
	return p
}

// moveWordBackward implements `b` and `B`.
func (e *Engine) moveWordBackward(from Position, count int, bigWord bool) Position {
	p := from
	for c := 0; c < count; c++ {
		prev, ok := e.stepBackward(p)
		if !ok {
			break
		}
		p = prev
		// Skip whitespace backward
		for {
			line := e.Lines[p.Row]
			if len(line) == 0 {
				if p.Row == 0 {
					break
				}
				// Stopping on empty line if we came from another line
				break
			}
			for p.Col >= 0 && unicode.IsSpace(line[p.Col]) {
				p.Col--
			}
			if p.Col >= 0 {
				break
			}
			if p.Row == 0 {
				p.Col = 0
				return p
			}
			p.Row--
			if len(e.Lines[p.Row]) == 0 {
				p.Col = 0
				break
			}
			p.Col = len(e.Lines[p.Row]) - 1
		}
		line := e.Lines[p.Row]
		if len(line) == 0 {
			continue
		}
		cls := charClass(line[p.Col], bigWord)
		for p.Col > 0 && charClass(line[p.Col-1], bigWord) == cls {
			p.Col--
		}
	}
	return p
}

func (e *Engine) findCharOnLine(kind, ch rune, count int) (Position, bool) {
	line := e.Lines[e.Cursor.Row]
	col := e.Cursor.Col
	forward := kind == 'f' || kind == 't'
	till := kind == 't' || kind == 'T'

	if forward {
		found := -1
		matches := 0
		for i := col + 1; i < len(line); i++ {
			if line[i] == ch {
				matches++
				if matches == count {
					found = i
					break
				}
			}
		}
		if found == -1 {
			return e.Cursor, false
		}
		if till {
			found--
		}
		return Position{Row: e.Cursor.Row, Col: found}, true
	}

	found := -1
	matches := 0
	for i := col - 1; i >= 0; i-- {
		if line[i] == ch {
			matches++
			if matches == count {
				found = i
				break
			}
		}
	}
	if found == -1 {
		return e.Cursor, false
	}
	if till {
		found++
	}
	return Position{Row: e.Cursor.Row, Col: found}, true
}

func matchingPair(r rune) (open rune, close rune, isOpen bool, ok bool) {
	switch r {
	case '(':
		return '(', ')', true, true
	case ')':
		return '(', ')', false, true
	case '[':
		return '[', ']', true, true
	case ']':
		return '[', ']', false, true
	case '{':
		return '{', '}', true, true
	case '}':
		return '{', '}', false, true
	case '<':
		return '<', '>', true, true
	case '>':
		return '<', '>', false, true
	default:
		return 0, 0, false, false
	}
}

// matchBracket implements `%`.
func (e *Engine) matchBracket(from Position) (Position, bool) {
	if from.Row >= len(e.Lines) {
		return from, false
	}
	line := e.Lines[from.Row]
	if len(line) == 0 {
		return from, false
	}
	col := from.Col
	if col >= len(line) {
		col = len(line) - 1
	}
	open, closeCh, isOpen, ok := matchingPair(line[col])
	if !ok {
		// Scan forward on current line for the first bracket (Vim behavior)
		found := false
		for i := col + 1; i < len(line); i++ {
			if o, c, io, yes := matchingPair(line[i]); yes && o != '<' {
				open, closeCh, isOpen = o, c, io
				col = i
				found = true
				break
			}
		}
		if !found {
			return from, false
		}
	}

	depth := 0
	p := Position{Row: from.Row, Col: col}
	if isOpen {
		for {
			r := e.Lines[p.Row][p.Col]
			if r == open {
				depth++
			} else if r == closeCh {
				depth--
				if depth == 0 {
					return p, true
				}
			}
			next, hasNext := e.stepForward(p)
			if !hasNext {
				return from, false
			}
			// skip empty lines
			for len(e.Lines[next.Row]) == 0 {
				n2, hasN2 := e.stepForward(next)
				if !hasN2 {
					return from, false
				}
				next = n2
			}
			p = next
		}
	} else {
		for {
			r := e.Lines[p.Row][p.Col]
			if r == closeCh {
				depth++
			} else if r == open {
				depth--
				if depth == 0 {
					return p, true
				}
			}
			prev, hasPrev := e.stepBackward(p)
			if !hasPrev {
				return from, false
			}
			for len(e.Lines[prev.Row]) == 0 {
				p2, hasP2 := e.stepBackward(prev)
				if !hasP2 {
					return from, false
				}
				prev = p2
			}
			p = prev
		}
	}
}

func (e *Engine) paragraphPrev(from Position, count int) Position {
	r := from.Row
	for c := 0; c < count; c++ {
		for r > 0 && len(strings.TrimSpace(string(e.Lines[r]))) == 0 {
			r--
		}
		for r > 0 && len(strings.TrimSpace(string(e.Lines[r-1]))) != 0 {
			r--
		}
		if r > 0 {
			r--
		}
	}
	return Position{Row: r, Col: 0}
}

func (e *Engine) paragraphNext(from Position, count int) Position {
	r := from.Row
	for c := 0; c < count; c++ {
		for r < len(e.Lines)-1 && len(strings.TrimSpace(string(e.Lines[r]))) == 0 {
			r++
		}
		for r < len(e.Lines)-1 && len(strings.TrimSpace(string(e.Lines[r+1]))) != 0 {
			r++
		}
		if r < len(e.Lines)-1 {
			r++
		}
	}
	return Position{Row: r, Col: 0}
}

func (e *Engine) searchPattern(pattern string, forward bool, count int) (Position, bool) {
	if pattern == "" {
		return e.Cursor, false
	}
	patRunes := []rune(pattern)
	cur := e.Cursor
	nLines := len(e.Lines)
	if nLines == 0 {
		return cur, false
	}

	foundPos := cur
	foundAny := false
	for c := 0; c < count; c++ {
		hit := false
		if forward {
			startRow := cur.Row
			startCol := cur.Col + 1
			for offset := 0; offset <= nLines; offset++ {
				r := (startRow + offset) % nLines
				line := e.Lines[r]
				colBegin := 0
				if offset == 0 {
					colBegin = startCol
				}
				for col := colBegin; col+len(patRunes) <= len(line); col++ {
					if runeSliceEqual(line[col:col+len(patRunes)], patRunes) {
						cur = Position{Row: r, Col: col}
						hit = true
						break
					}
				}
				if hit {
					break
				}
			}
		} else {
			startRow := cur.Row
			startCol := cur.Col - 1
			for offset := 0; offset <= nLines; offset++ {
				r := (startRow - offset + nLines) % nLines
				line := e.Lines[r]
				colMax := len(line) - len(patRunes)
				if offset == 0 && startCol < colMax {
					colMax = startCol
				}
				for col := colMax; col >= 0; col-- {
					if runeSliceEqual(line[col:col+len(patRunes)], patRunes) {
						cur = Position{Row: r, Col: col}
						hit = true
						break
					}
				}
				if hit {
					break
				}
			}
		}
		if !hit {
			return foundPos, foundAny
		}
		foundPos = cur
		foundAny = true
	}
	return foundPos, foundAny
}

func runeSliceEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (e *Engine) wordUnderCursor() string {
	line := e.Lines[e.Cursor.Row]
	if len(line) == 0 || e.Cursor.Col >= len(line) {
		return ""
	}
	col := e.Cursor.Col
	if charClass(line[col], false) != 1 {
		return ""
	}
	start, end := col, col
	for start > 0 && charClass(line[start-1], false) == 1 {
		start--
	}
	for end+1 < len(line) && charClass(line[end+1], false) == 1 {
		end++
	}
	return string(line[start : end+1])
}

// textObjectRange resolves Vim text objects: w, W, ", ', `, (, ), b, [, ], {, }, B, <, >.
func (e *Engine) textObjectRange(kind rune, inner bool) (Range, bool) {
	switch kind {
	case 'w', 'W':
		big := kind == 'W'
		line := e.Lines[e.Cursor.Row]
		if len(line) == 0 {
			return Range{}, false
		}
		col := e.Cursor.Col
		if col >= len(line) {
			col = len(line) - 1
		}
		cls := charClass(line[col], big)
		start, end := col, col
		for start > 0 && charClass(line[start-1], big) == cls {
			start--
		}
		for end+1 < len(line) && charClass(line[end+1], big) == cls {
			end++
		}
		if !inner {
			if cls != 0 {
				if end+1 < len(line) && unicode.IsSpace(line[end+1]) {
					for end+1 < len(line) && unicode.IsSpace(line[end+1]) {
						end++
					}
				} else {
					for start > 0 && unicode.IsSpace(line[start-1]) {
						start--
					}
				}
			}
		}
		return Range{
			Start:     Position{Row: e.Cursor.Row, Col: start},
			End:       Position{Row: e.Cursor.Row, Col: end},
			Inclusive: true,
		}, true

	case '"', '\'', '`':
		line := e.Lines[e.Cursor.Row]
		var positions []int
		for i, r := range line {
			if r == kind && (i == 0 || line[i-1] != '\\') {
				positions = append(positions, i)
			}
		}
		if len(positions) < 2 {
			return Range{}, false
		}
		// Find pair enclosing cursor, or first pair starting at/after cursor.
		openIdx, closeIdx := -1, -1
		for i := 0; i+1 < len(positions); i += 2 {
			a, b := positions[i], positions[i+1]
			if e.Cursor.Col >= a && e.Cursor.Col <= b {
				openIdx, closeIdx = a, b
				break
			}
		}
		if openIdx == -1 {
			for i := 0; i+1 < len(positions); i += 2 {
				if positions[i] >= e.Cursor.Col {
					openIdx, closeIdx = positions[i], positions[i+1]
					break
				}
			}
		}
		if openIdx == -1 {
			return Range{}, false
		}
		if inner {
			if closeIdx-openIdx <= 1 {
				// Empty quotes: nothing inside, still return empty range at openIdx+1 for `ci"`
				return Range{
					Start:     Position{Row: e.Cursor.Row, Col: openIdx + 1},
					End:       Position{Row: e.Cursor.Row, Col: openIdx},
					Inclusive: true,
				}, true
			}
			return Range{
				Start:     Position{Row: e.Cursor.Row, Col: openIdx + 1},
				End:       Position{Row: e.Cursor.Row, Col: closeIdx - 1},
				Inclusive: true,
			}, true
		}
		return Range{
			Start:     Position{Row: e.Cursor.Row, Col: openIdx},
			End:       Position{Row: e.Cursor.Row, Col: closeIdx},
			Inclusive: true,
		}, true

	case '(', ')', 'b', '[', ']', '{', '}', 'B', '<', '>':
		var openCh, closeCh rune
		switch kind {
		case '(', ')', 'b':
			openCh, closeCh = '(', ')'
		case '[', ']':
			openCh, closeCh = '[', ']'
		case '{', '}', 'B':
			openCh, closeCh = '{', '}'
		case '<', '>':
			openCh, closeCh = '<', '>'
		}
		openPos, closePos, ok := e.findEnclosingPair(openCh, closeCh)
		if !ok {
			return Range{}, false
		}
		if inner {
			s, ok1 := e.stepForward(openPos)
			t, ok2 := e.stepBackward(closePos)
			if !ok1 || !ok2 || comparePos(s, t) > 0 {
				return Range{
					Start:     s,
					End:       openPos,
					Inclusive: true,
				}, true
			}
			return Range{Start: s, End: t, Inclusive: true}, true
		}
		return Range{Start: openPos, End: closePos, Inclusive: true}, true
	}
	return Range{}, false
}

func (e *Engine) findEnclosingPair(openCh, closeCh rune) (Position, Position, bool) {
	cur := e.Cursor
	// Walk backward from cursor to find unmatched openCh
	depth := 0
	p := cur
	if len(e.Lines[cur.Row]) > 0 && cur.Col < len(e.Lines[cur.Row]) {
		if e.Lines[cur.Row][cur.Col] == openCh {
			if match, ok := e.findForwardMatch(cur, openCh, closeCh); ok {
				return cur, match, true
			}
		}
	}

	var openPos Position
	foundOpen := false
	for {
		if len(e.Lines[p.Row]) > 0 && p.Col < len(e.Lines[p.Row]) {
			r := e.Lines[p.Row][p.Col]
			if r == closeCh && (p.Row != cur.Row || p.Col != cur.Col) {
				depth++
			} else if r == openCh {
				if depth == 0 {
					openPos = p
					foundOpen = true
					break
				}
				depth--
			}
		}
		prev, hasPrev := e.stepBackward(p)
		if !hasPrev {
			break
		}
		p = prev
	}

	if !foundOpen {
		// Scan forward on current line for an opening bracket
		line := e.Lines[cur.Row]
		for i := cur.Col; i < len(line); i++ {
			if line[i] == openCh {
				openPos = Position{Row: cur.Row, Col: i}
				foundOpen = true
				break
			}
		}
	}
	if !foundOpen {
		return Position{}, Position{}, false
	}
	closePos, ok := e.findForwardMatch(openPos, openCh, closeCh)
	return openPos, closePos, ok
}

func (e *Engine) findForwardMatch(openPos Position, openCh, closeCh rune) (Position, bool) {
	depth := 0
	p := openPos
	for {
		if len(e.Lines[p.Row]) > 0 && p.Col < len(e.Lines[p.Row]) {
			r := e.Lines[p.Row][p.Col]
			if r == openCh {
				depth++
			} else if r == closeCh {
				depth--
				if depth == 0 {
					return p, true
				}
			}
		}
		next, hasNext := e.stepForward(p)
		if !hasNext {
			return Position{}, false
		}
		p = next
	}
}

func comparePos(a, b Position) int {
	if a.Row != b.Row {
		if a.Row < b.Row {
			return -1
		}
		return 1
	}
	if a.Col != b.Col {
		if a.Col < b.Col {
			return -1
		}
		return 1
	}
	return 0
}
