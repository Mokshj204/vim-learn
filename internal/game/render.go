package game

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"vimlearn/internal/levels"
	"vimlearn/internal/store"
	"vimlearn/internal/term"
	"vimlearn/internal/vim"
)

func fitWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	rs := []rune(s)
	if len(rs) > width {
		if width > 1 {
			return string(rs[:width-1]) + "…"
		}
		return string(rs[:width])
	}
	if len(rs) < width {
		return s + strings.Repeat(" ", width-len(rs))
	}
	return s
}

func wrapText(s string, width int) []string {
	if width <= 4 {
		return []string{fitWidth(s, width)}
	}
	if s == "" {
		return []string{""}
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if utf8.RuneCountInString(cur)+1+utf8.RuneCountInString(w) <= width {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	lines = append(lines, cur)
	return lines
}

func starsString(stars int) string {
	var b strings.Builder
	for i := 0; i < 3; i++ {
		if i < stars {
			b.WriteString(term.FgBrightYellow + "★" + term.Reset)
		} else {
			b.WriteString(term.FgBrightBlack + "☆" + term.Reset)
		}
	}
	return b.String()
}

func starsPlain(stars int) string {
	var b strings.Builder
	for i := 0; i < 3; i++ {
		if i < stars {
			b.WriteRune('★')
		} else {
			b.WriteRune('☆')
		}
	}
	return b.String()
}

type screenCanvas struct {
	b    strings.Builder
	cols int
	rows int
	row  int
}

func newCanvas(cols, rows int) *screenCanvas {
	return &screenCanvas{cols: cols, rows: rows, row: 1}
}

func (c *screenCanvas) putRow(r int, s string) {
	if r < 1 || r > c.rows {
		return
	}
	c.b.WriteString(term.MoveTo(r, 1))
	c.b.WriteString(s)
	c.b.WriteString(term.Reset)
	c.b.WriteString(term.ClearLine)
}

func (c *screenCanvas) next(s string) {
	if c.row <= c.rows {
		c.putRow(c.row, s)
		c.row++
	}
}

func (c *screenCanvas) fillTo(targetRow int) {
	for c.row < targetRow && c.row <= c.rows {
		c.putRow(c.row, "")
		c.row++
	}
}

func (c *screenCanvas) finish() string {
	for c.row <= c.rows {
		c.putRow(c.row, "")
		c.row++
	}
	return c.b.String()
}

func boxTop(width int, title string) string {
	inner := width - 2
	if inner < 2 {
		return "+"
	}
	if title == "" {
		return "┌" + strings.Repeat("─", inner) + "┐"
	}
	t := " " + title + " "
	tr := utf8.RuneCountInString(t)
	if tr > inner-2 {
		t = fitWidth(t, inner-2)
		tr = utf8.RuneCountInString(t)
	}
	right := inner - 1 - tr
	if right < 0 {
		right = 0
	}
	return "┌─" + t + strings.Repeat("─", right) + "┐"
}

func boxBottom(width int) string {
	inner := width - 2
	if inner < 2 {
		return "+"
	}
	return "└" + strings.Repeat("─", inner) + "┘"
}

func (a *App) render() string {
	cols, rows, err := a.term.Size()
	if err != nil {
		cols = 80
		rows = 24
	}
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}

	c := newCanvas(cols, rows)
	if cols < MinCols || rows < MinRows {
		a.renderTooSmall(c)
		return c.finish()
	}

	switch a.screen {
	case ScreenHome:
		a.renderHome(c)
	case ScreenTutorial:
		a.renderTutorial(c)
	case ScreenLevels:
		a.renderLevelSelect(c)
	case ScreenPlay, ScreenWin, ScreenTimeout:
		a.renderPlay(c)
	}
	return c.finish()
}

func (a *App) renderTooSmall(c *screenCanvas) {
	mid := c.rows / 2
	if mid < 2 {
		mid = 1
	}
	c.fillTo(mid - 1)
	c.next(term.Bold + term.FgBrightRed + fitWidth(" Terminal Too Small ", c.cols) + term.Reset)
	c.next(term.FgBrightWhite + fitWidth(fmt.Sprintf(" Current: %dx%d  |  Required: %dx%d", c.cols, c.rows, MinCols, MinRows), c.cols) + term.Reset)
	c.next(term.Dim + fitWidth(" Resize your terminal window to continue (or press q / Ctrl-C to quit).", c.cols) + term.Reset)
}

func renderTopBar(cols int, left, right string) string {
	lLen := utf8.RuneCountInString(left)
	rLen := utf8.RuneCountInString(right)
	gap := cols - lLen - rLen - 2
	if gap < 1 {
		gap = 1
	}
	bar := " " + left + strings.Repeat(" ", gap) + right + " "
	return term.Bg256(236) + term.Bold + term.FgBrightCyan + fitWidth(bar, cols) + term.Reset
}

func renderFooterBar(cols int, text string) string {
	return term.Bg256(235) + term.FgBrightWhite + " " + fitWidth(text, cols-1) + term.Reset
}

func (a *App) renderHome(c *screenCanvas) {
	cols, rows := c.cols, c.rows

	totalStars := 0
	completed := 0
	allLevels := levels.All()
	for _, lv := range allLevels {
		rec := a.progress.Levels[lv.ID]
		totalStars += rec.Stars
		if rec.Completed {
			completed++
		}
	}

	c.next(renderTopBar(cols,
		"VIMLEARN — Interactive Vim Keybindings Trainer",
		fmt.Sprintf("Progress: %d/%d  |  %d/%d ★", completed, len(allLevels), totalStars, len(allLevels)*3),
	))
	c.next("")
	c.next(term.Bold + term.FgBrightCyan + "  ╦  ╦╦╔╦╗╦  ╔═╗╔═╗╦═╗╔╗╔   " + term.Reset + term.Dim + "POSIX & PowerShell ANSI Edition" + term.Reset)
	c.next(term.Bold + term.FgCyan + "  ╚╗╔╝║║║║║  ║╣ ╠═╣╠╦╝║║║   " + term.Reset + term.FgBrightBlack + fmt.Sprintf("Terminal: %dx%d", cols, rows) + term.Reset)
	c.next("")

	boxW := cols - 4
	if boxW < 30 {
		boxW = cols
	}
	innerW := boxW - 4

	// Tutorial Hub Entry
	tutSelected := a.homeIdx == 0
	tutText := "[T] Interactive Vim Tutorial — Learn Modes, Motions, Grammar & Power Tools"
	if tutSelected {
		c.next("  " + term.Bold + term.Bg256(24) + term.FgBrightWhite + " ▶ " + fitWidth(tutText, boxW-3) + term.Reset)
	} else {
		c.next("  " + term.FgBrightCyan + "   " + fitWidth(tutText, boxW-3) + term.Reset)
	}
	c.next("")

	c.next("  " + term.Bold + term.FgBrightWhite + boxTop(boxW, "Choose a Difficulty (Press 1-5 or Enter)") + term.Reset)
	diffs := levels.Difficulties()
	for i, d := range diffs {
		idx := i + 1
		dLevels := levels.ByDifficulty(d.ID)
		dDone, dStars := 0, 0
		for _, lv := range dLevels {
			rec := a.progress.Levels[lv.ID]
			if rec.Completed {
				dDone++
			}
			dStars += rec.Stars
		}

		selected := a.homeIdx == idx
		prefix := "   "
		rowStyle := ""
		if selected {
			prefix = term.Bold + term.FgBrightYellow + " ▶ "
			rowStyle = term.Bg256(237)
		}

		leftPart := fmt.Sprintf("[%d] %-13s  %s", i+1, d.Name, d.Tagline)
		rightPart := fmt.Sprintf("%d/%d done  %2d/%2d ★", dDone, len(dLevels), dStars, len(dLevels)*3)
		availLeft := innerW - 3 - utf8.RuneCountInString(rightPart) - 2
		if availLeft < 16 {
			availLeft = 16
		}
		rowText := fitWidth(leftPart, availLeft) + "  " + rightPart
		c.next("  │ " + prefix + rowStyle + term.Bold + term.Fg256(d.ColorCode) + fitWidth(rowText, innerW-3) + term.Reset + " │")

		if selected {
			for _, dl := range wrapText(d.Description, innerW-6) {
				c.next("  │ " + strings.Repeat(" ", 6) + term.Dim + fitWidth(dl, innerW-6) + term.Reset + " │")
			}
			c.next("  │ " + strings.Repeat(" ", 6) + term.Fg256(d.ColorCode) + fitWidth("Keys: "+d.KeysSummary, innerW-6) + term.Reset + " │")
		}
	}
	c.next("  " + term.FgBrightWhite + boxBottom(boxW) + term.Reset)

	// Fill remaining space before footer
	c.fillTo(rows)
	c.putRow(rows, renderFooterBar(cols, "j/k or ↑/↓: Move   Enter: Open Difficulty   1-5: Quick Jump   t: Tutorial   q: Quit"))
}

func (a *App) renderTutorial(c *screenCanvas) {
	cols, rows := c.cols, c.rows
	lessons := levels.Tutorials()
	if a.tutIdx >= len(lessons) {
		a.tutIdx = 0
	}
	cur := lessons[a.tutIdx]

	c.next(renderTopBar(cols, "VIM PRIMER & GAME TUTORIAL", fmt.Sprintf("Lesson %d/%d", a.tutIdx+1, len(lessons))))
	c.next("")

	var tabs strings.Builder
	tabs.WriteString("  ")
	for i, l := range lessons {
		if i == a.tutIdx {
			tabs.WriteString(term.Bold + term.Bg256(31) + term.FgBrightWhite + fmt.Sprintf(" %d: %s ", i+1, l.ID) + term.Reset + " ")
		} else {
			tabs.WriteString(term.Bg256(236) + term.FgWhite + fmt.Sprintf(" %d: %s ", i+1, l.ID) + term.Reset + " ")
		}
	}
	c.next(tabs.String())
	c.next("")

	boxW := cols - 4
	innerW := boxW - 4
	c.next("  " + term.FgBrightCyan + boxTop(boxW, cur.Title) + term.Reset)
	c.next("  │ " + term.Italic + term.FgCyan + fitWidth(cur.Summary, innerW) + term.Reset + " │")
	c.next("  │ " + strings.Repeat(" ", innerW) + " │")

	for _, rawLine := range cur.Body {
		if rawLine == "" {
			c.next("  │ " + strings.Repeat(" ", innerW) + " │")
			continue
		}
		for _, wrapped := range wrapText(rawLine, innerW) {
			c.next("  │ " + fitWidth(wrapped, innerW) + " │")
		}
	}
	c.next("  │ " + strings.Repeat(" ", innerW) + " │")
	c.next("  │ " + term.Bold + term.FgBrightGreen + fitWidth("Key Cheat Sheet:", innerW) + term.Reset + " │")
	for _, kh := range cur.Keys {
		descW := innerW - 22
		if descW < 10 {
			descW = 10
		}
		c.next("  │ " + fmt.Sprintf("  %s%-18s%s  %s", term.Bold+term.FgBrightYellow, fitWidth(kh.Key, 18), term.Reset, fitWidth(kh.Desc, descW)) + " │")
	}

	// Fill inside the box down to near the bottom so the tutorial card stretches cleanly
	boxBottomRow := rows - 4
	for c.row < boxBottomRow {
		c.next("  │ " + strings.Repeat(" ", innerW) + " │")
	}
	c.next("  │ " + term.Bg256(236) + term.FgBrightWhite + fitWidth(" "+cur.Example, innerW) + term.Reset + " │")
	c.next("  " + term.FgBrightCyan + boxBottom(boxW) + term.Reset)

	c.fillTo(rows)
	c.putRow(rows, renderFooterBar(cols, "h/l or ←/→: Prev/Next Lesson   1-4: Jump Lesson   Enter: Next   Esc/q: Back to Menu"))
}

func (a *App) renderLevelSelect(c *screenCanvas) {
	cols, rows := c.cols, c.rows
	diff, _ := levels.DifficultyByID(a.selDiff)
	dLevels := levels.ByDifficulty(a.selDiff)
	if a.levelIdx >= len(dLevels) {
		a.levelIdx = 0
	}

	c.next(renderTopBar(cols, strings.ToUpper(diff.Name)+" — "+diff.Tagline, fmt.Sprintf("%d Levels", len(dLevels))))
	c.next("  " + term.Dim + fitWidth(diff.Description, cols-4) + term.Reset)
	c.next("")

	boxW := cols - 4
	innerW := boxW - 4
	c.next("  " + term.Fg256(diff.ColorCode) + boxTop(boxW, "Select Any Level") + term.Reset)

	for i, lv := range dLevels {
		rec := a.progress.Levels[lv.ID]
		selected := i == a.levelIdx
		prefix := "   "
		rowStyle := ""
		if selected {
			prefix = term.Bold + term.FgBrightYellow + " ▶ "
			rowStyle = term.Bg256(237) + term.FgBrightWhite
		}
		bestStr := " -- "
		if rec.Completed && rec.BestKeys > 0 {
			bestStr = fmt.Sprintf("%4d", rec.BestKeys)
		}
		timerStr := "    "
		if lv.TimeLimitSec > 0 {
			timerStr = fmt.Sprintf(" %2ds", lv.TimeLimitSec)
		}
		rightStats := fmt.Sprintf("Par:%2d%s  Best:%s  %s", lv.Par, timerStr, bestStr, starsPlain(rec.Stars))
		titleW := innerW - 3 - utf8.RuneCountInString(rightStats) - 2
		if titleW < 20 {
			titleW = 20
		}
		rowLine := fmt.Sprintf("[%d] %-*s  %s", lv.Number, titleW-4, fitWidth(lv.Title, titleW-4), rightStats)
		c.next("  │ " + prefix + rowStyle + fitWidth(rowLine, innerW-3) + term.Reset + " │")
	}
	c.next("  " + term.Fg256(diff.ColorCode) + boxBottom(boxW) + term.Reset)
	c.next("")

	if len(dLevels) > 0 {
		lv := dLevels[a.levelIdx]
		c.next("  " + term.FgBrightCyan + boxTop(boxW, fmt.Sprintf("Level %d Briefing: %s", lv.Number, lv.Title)) + term.Reset)
		for _, line := range wrapText("Goal: "+lv.Tutorial.Objective, innerW) {
			c.next("  │ " + term.Bold + fitWidth(line, innerW) + term.Reset + " │")
		}
		c.next("  │ " + strings.Repeat(" ", innerW) + " │")
		for _, line := range wrapText(lv.Tutorial.Explanation, innerW) {
			c.next("  │ " + term.FgCyan + fitWidth(line, innerW) + term.Reset + " │")
		}
		c.next("  │ " + strings.Repeat(" ", innerW) + " │")
		var keyList []string
		for _, kh := range lv.Tutorial.Keys {
			keyList = append(keyList, kh.Key+" ("+kh.Desc+")")
		}
		for _, line := range wrapText("Keys: "+strings.Join(keyList, "  |  "), innerW) {
			c.next("  │ " + term.FgBrightYellow + fitWidth(line, innerW) + term.Reset + " │")
		}
		for c.row < rows-2 {
			c.next("  │ " + strings.Repeat(" ", innerW) + " │")
		}
		c.next("  " + term.FgBrightCyan + boxBottom(boxW) + term.Reset)
	}

	c.fillTo(rows)
	c.putRow(rows, renderFooterBar(cols, "j/k or ↑/↓: Select   Enter: Start Level   1-5: Quick Play   t: Tutorial   Esc/b: Difficulties"))
}

func (a *App) renderPlay(c *screenCanvas) {
	cols, rows := c.cols, c.rows
	lv := a.currentLevel
	diff, _ := levels.DifficultyByID(lv.Difficulty)

	// Top Bar spanning full terminal width
	modeBadge := modeColor(a.eng.Mode) + " " + a.eng.Mode.String() + " " + term.Reset
	parColor := term.FgBrightGreen
	if a.eng.KeyCount > lv.Par {
		parColor = term.FgBrightYellow
	}
	timerText := ""
	if lv.TimeLimitSec > 0 {
		rem := a.remainingSeconds()
		tCol := term.FgBrightCyan
		if rem <= 10 {
			tCol = term.Bold + term.FgBrightRed
		}
		timerText = fmt.Sprintf("  |  %s⏱ %ds%s", tCol, rem, term.Reset)
	}
	macroBadge := ""
	if a.eng.RecordingMacro != 0 {
		macroBadge = term.Bold + term.BgRed + term.FgBrightWhite + " REC @" + string(a.eng.RecordingMacro) + " " + term.Reset + " "
	}
	headerLeft := fmt.Sprintf(" %s %s%s L%d: %s%s", modeBadge, term.Bold+term.Fg256(diff.ColorCode), diff.Name, lv.Number, lv.Title, term.Reset)
	headerRight := fmt.Sprintf("%sKeys: %s%d%s / Par %d%s ", macroBadge, parColor+term.Bold, a.eng.KeyCount, term.Reset, lv.Par, timerText)
	c.next(headerLeft + "   " + headerRight)

	// Briefing Box spanning full terminal width
	boxW := cols - 4
	innerW := boxW - 4
	c.next("  " + term.FgBrightCyan + boxTop(boxW, "What To Do") + term.Reset)
	for _, line := range wrapText(lv.Tutorial.Objective, innerW) {
		c.next("  │ " + term.Bold + term.FgBrightWhite + fitWidth(line, innerW) + term.Reset + " │")
	}
	if a.showLesson || a.showHint {
		for _, line := range wrapText(lv.Tutorial.Explanation, innerW) {
			c.next("  │ " + term.FgCyan + fitWidth(line, innerW) + term.Reset + " │")
		}
		for _, kh := range lv.Tutorial.Keys {
			descW := innerW - 18
			if descW < 10 {
				descW = 10
			}
			c.next("  │ " + fmt.Sprintf("  %s%-14s%s  %s", term.Bold+term.FgBrightYellow, fitWidth(kh.Key, 14), term.Reset, fitWidth(kh.Desc, descW)) + " │")
		}
		if a.showHint {
			for _, line := range wrapText("Walkthrough: "+lv.Tutorial.Walkthrough, innerW) {
				c.next("  │ " + term.FgBrightGreen + fitWidth(line, innerW) + term.Reset + " │")
			}
			for _, line := range wrapText("Hint: "+lv.Hint, innerW) {
				c.next("  │ " + term.FgBrightMagenta + fitWidth(line, innerW) + term.Reset + " │")
			}
		}
	} else {
		var keysCompact []string
		for _, kh := range lv.Tutorial.Keys {
			keysCompact = append(keysCompact, kh.Key+" ("+kh.Desc+")")
		}
		c.next("  │ " + term.FgBrightYellow + fitWidth("Keys: "+strings.Join(keysCompact, " · "), innerW) + term.Reset + " │")
	}
	c.next("  " + term.FgBrightCyan + boxBottom(boxW) + term.Reset)

	// Reserve bottom rows for status, keystrokes, win/timeout modal, and footer
	bottomReserved := 3 // status/cmd + keystrokes + footer
	if a.screen == ScreenWin || a.screen == ScreenTimeout {
		bottomReserved += 5
		if a.showSolution {
			bottomReserved++
		}
	}
	availBufRows := rows - c.row - bottomReserved - 1 // -2 for top/bottom box border +1 inclusive
	if availBufRows < 4 {
		availBufRows = 4
	}

	// Render Editor + Target spanning full width AND full remaining height!
	bufLines := a.eng.LineStrings()
	if lv.GoalType == levels.GoalReachCursor {
		a.renderCursorPuzzle(c, boxW, availBufRows, bufLines)
	} else {
		targetLines := strings.Split(lv.TargetText, "\n")
		if cols >= 72 {
			a.renderSplitBuffers(c, cols, availBufRows, bufLines, targetLines)
		} else {
			a.renderStackedBuffers(c, boxW, availBufRows, bufLines, targetLines)
		}
	}

	// Status / Command line + Keystroke tape + Win/Timeout Banner + Footer pinned to bottom
	statusRow := rows - bottomReserved + 1
	c.fillTo(statusRow)

	if a.eng.Mode == vim.ModeCommand {
		c.putRow(statusRow, "  "+term.Bold+term.FgBrightWhite+string(a.eng.CmdPrefix)+a.eng.CmdInput+term.Reverse+" "+term.Reset)
	} else {
		pending := a.eng.PendingSummary()
		status := a.eng.StatusMsg
		if pending != "" {
			status = "Pending: " + pending + "   " + status
		}
		c.putRow(statusRow, "  "+term.FgBrightYellow+fitWidth(status, cols-4)+term.Reset)
	}

	tape := a.recentKeyTape(cols - 16)
	c.putRow(statusRow+1, "  "+term.Dim+"Keystrokes: "+term.Reset+term.FgBrightWhite+tape+term.Reset)

	switch a.screen {
	case ScreenWin:
		a.renderWinBanner(c, statusRow+2, boxW)
		c.putRow(rows, renderFooterBar(cols, "Enter / n: Next Level   r: Retry for Par   s: Toggle Optimal Solution   Esc / l: Level Select"))
	case ScreenTimeout:
		a.renderTimeoutBanner(c, statusRow+2, boxW)
		c.putRow(rows, renderFooterBar(cols, "Enter / r: Retry Level   Esc / l: Level Select"))
	default:
		c.putRow(rows, renderFooterBar(cols, "Tab: Expand Lesson/Keys   Ctrl+L: Walkthrough/Hint   :reset: Restart   :q: Level Select"))
	}
	c.row = rows + 1
}

func (a *App) renderCursorPuzzle(c *screenCanvas, boxW, contentRows int, bufLines []string) {
	lv := a.currentLevel
	cpLabel := fmt.Sprintf("Live Buffer (Checkpoint %d/%d)", a.checkpointIdx, len(lv.Checkpoints))
	c.next("  " + term.FgBrightWhite + boxTop(boxW, cpLabel) + term.Reset)

	var targetPos vim.Position
	hasTarget := a.checkpointIdx < len(lv.Checkpoints)
	if hasTarget {
		targetPos = lv.Checkpoints[a.checkpointIdx]
	}

	innerW := boxW - 7 // "│ " (2) + "2d " (3) + " │" (2)
	for r := 0; r < contentRows; r++ {
		if r < len(bufLines) {
			rendered := a.renderLineWithCursorAndTarget(r, []rune(bufLines[r]), targetPos, hasTarget, innerW)
			c.next(fmt.Sprintf("  │ %s%2d%s %s │", term.Dim, r+1, term.Reset, rendered))
		} else {
			c.next(fmt.Sprintf("  │ %s~ %s %s │", term.FgBrightBlack, term.Reset, strings.Repeat(" ", innerW)))
		}
	}
	c.next("  " + term.FgBrightWhite + boxBottom(boxW) + term.Reset)
}

func (a *App) renderSplitBuffers(c *screenCanvas, cols, contentRows int, bufLines, targetLines []string) {
	// Two equal panes filling full width: "  " + paneW + "  " + paneW + "  "
	paneW := (cols - 6) / 2
	innerW := paneW - 7 // "│ " (2) + "2d " (3) + " │" (2)
	c.next("  " + term.FgBrightWhite + boxTop(paneW, "Your Buffer") + term.Reset + "  " + term.FgBrightGreen + boxTop(paneW, "Target Buffer") + term.Reset)

	for r := 0; r < contentRows; r++ {
		var leftRendered string
		if r < len(bufLines) {
			leftRendered = a.renderLineWithCursorAndTarget(r, []rune(bufLines[r]), vim.Position{}, false, innerW)
			leftRendered = fmt.Sprintf("%s%2d%s %s", term.Dim, r+1, term.Reset, leftRendered)
		} else {
			leftRendered = term.FgBrightBlack + " ~ " + term.Reset + strings.Repeat(" ", innerW)
		}

		var rightRendered string
		if r < len(targetLines) {
			matchMark := term.FgRed + "≠" + term.Reset
			if r < len(bufLines) && bufLines[r] == targetLines[r] {
				matchMark = term.FgBrightGreen + "✓" + term.Reset
			}
			rightRendered = fmt.Sprintf("%s %s%2d%s %s", matchMark, term.Dim, r+1, term.Reset, fitWidth(targetLines[r], innerW-2))
		} else {
			rightRendered = term.FgBrightBlack + "   ~" + term.Reset + strings.Repeat(" ", maxZero(innerW-1))
		}
		c.next("  │ " + leftRendered + " │  │ " + rightRendered + " │")
	}
	c.next("  " + term.FgBrightWhite + boxBottom(paneW) + term.Reset + "  " + term.FgBrightGreen + boxBottom(paneW) + term.Reset)
}

func (a *App) renderStackedBuffers(c *screenCanvas, boxW, contentRows int, bufLines, targetLines []string) {
	innerW := boxW - 7
	half := contentRows / 2
	if half < 2 {
		half = 2
	}
	c.next("  " + term.FgBrightWhite + boxTop(boxW, "Your Buffer") + term.Reset)
	for r := 0; r < half; r++ {
		if r < len(bufLines) {
			rendered := a.renderLineWithCursorAndTarget(r, []rune(bufLines[r]), vim.Position{}, false, innerW)
			c.next(fmt.Sprintf("  │ %s%2d%s %s │", term.Dim, r+1, term.Reset, rendered))
		} else {
			c.next(fmt.Sprintf("  │ %s ~ %s%s │", term.FgBrightBlack, term.Reset, strings.Repeat(" ", innerW)))
		}
	}
	c.next("  " + term.FgBrightWhite + boxBottom(boxW) + term.Reset)

	c.next("  " + term.FgBrightGreen + boxTop(boxW, "Target Buffer") + term.Reset)
	for r := 0; r < half; r++ {
		if r < len(targetLines) {
			mark := term.FgRed + "≠" + term.Reset
			if r < len(bufLines) && bufLines[r] == targetLines[r] {
				mark = term.FgBrightGreen + "✓" + term.Reset
			}
			c.next(fmt.Sprintf("  │ %s %s%2d%s %s │", mark, term.Dim, r+1, term.Reset, fitWidth(targetLines[r], innerW-2)))
		} else {
			c.next(fmt.Sprintf("  │ %s   ~%s%s │", term.FgBrightBlack, term.Reset, strings.Repeat(" ", maxZero(innerW-1))))
		}
	}
	c.next("  " + term.FgBrightGreen + boxBottom(boxW) + term.Reset)
}

func (a *App) renderLineWithCursorAndTarget(row int, rs []rune, target vim.Position, hasTarget bool, width int) string {
	var out strings.Builder
	cur := a.eng.Cursor
	visStart, visEnd, hasVis := a.visualSelectionBounds()

	displayCols := width
	if displayCols < 1 {
		displayCols = 1
	}

	printed := 0
	for col := 0; col <= len(rs) && printed < displayCols; col++ {
		isCursor := row == cur.Row && col == cur.Col
		isTarget := hasTarget && row == target.Row && col == target.Col

		if col == len(rs) {
			if isCursor {
				if a.eng.Mode == vim.ModeInsert {
					out.WriteString(term.BgGreen + term.FgBlack + "▏" + term.Reset)
				} else {
					out.WriteString(term.Reverse + " " + term.Reset)
				}
				printed++
			}
			break
		}

		chStr := string(rs[col])
		pos := vim.Position{Row: row, Col: col}
		inVis := hasVis && isPosInVisual(pos, visStart, visEnd, a.eng.Mode == vim.ModeVisualLine)

		switch {
		case isCursor && a.eng.Mode == vim.ModeInsert:
			out.WriteString(term.BgBrightGreen + term.FgBlack + term.Bold + chStr + term.Reset)
		case isCursor:
			out.WriteString(term.BgBrightYellow + term.FgBlack + term.Bold + chStr + term.Reset)
		case isTarget:
			out.WriteString(term.BgCyan + term.FgBrightWhite + term.Bold + chStr + term.Reset)
		case inVis:
			out.WriteString(term.Bg256(60) + term.FgBrightWhite + chStr + term.Reset)
		default:
			out.WriteString(chStr)
		}
		printed++
	}
	if printed < displayCols {
		out.WriteString(strings.Repeat(" ", displayCols-printed))
	}
	return out.String()
}

func (a *App) visualSelectionBounds() (vim.Position, vim.Position, bool) {
	if a.eng.Mode != vim.ModeVisual && a.eng.Mode != vim.ModeVisualLine {
		return vim.Position{}, vim.Position{}, false
	}
	s, e := a.eng.VisualAnchor, a.eng.Cursor
	if s.Row > e.Row || (s.Row == e.Row && s.Col > e.Col) {
		s, e = e, s
	}
	return s, e, true
}

func isPosInVisual(p, s, e vim.Position, linewise bool) bool {
	if linewise {
		return p.Row >= s.Row && p.Row <= e.Row
	}
	if p.Row < s.Row || p.Row > e.Row {
		return false
	}
	if p.Row == s.Row && p.Col < s.Col {
		return false
	}
	if p.Row == e.Row && p.Col > e.Col {
		return false
	}
	return true
}

func (a *App) renderWinBanner(c *screenCanvas, startRow, boxW int) {
	lv := a.currentLevel
	stars := store.StarsForScore(a.eng.KeyCount, lv.Par)
	innerW := boxW - 4
	plainMsg := fmt.Sprintf("Nice work!  Keys: %d (Par %d)   Rating: %s", a.eng.KeyCount, lv.Par, starsPlain(stars))
	pad := maxZero(innerW - utf8.RuneCountInString(plainMsg))
	c.putRow(startRow, "  "+term.Bold+term.FgBrightGreen+boxTop(boxW, "LEVEL COMPLETE!")+term.Reset)
	c.putRow(startRow+1, fmt.Sprintf("  │ %s  Keys: %s%d%s (Par %d)   Rating: %s%s │",
		term.Bold+term.FgBrightWhite+"Nice work!"+term.Reset,
		term.Bold+term.FgBrightCyan, a.eng.KeyCount, term.Reset,
		lv.Par,
		starsString(stars),
		strings.Repeat(" ", pad),
	))
	row := startRow + 2
	if a.showSolution {
		solLine := "Optimal Solution: " + lv.OptimalSeq
		c.putRow(row, "  │ "+term.FgBrightYellow+fitWidth(solLine, innerW)+term.Reset+" │")
		row++
	}
	c.putRow(row, "  "+term.Bold+term.FgBrightGreen+boxBottom(boxW)+term.Reset)
}

func (a *App) renderTimeoutBanner(c *screenCanvas, startRow, boxW int) {
	innerW := boxW - 4
	c.putRow(startRow, "  "+term.Bold+term.FgBrightRed+boxTop(boxW, "TIME'S UP!")+term.Reset)
	c.putRow(startRow+1, "  │ "+term.FgBrightWhite+fitWidth("The Impossible countdown reached 0s. Try again!", innerW)+term.Reset+" │")
	c.putRow(startRow+2, "  "+term.Bold+term.FgBrightRed+boxBottom(boxW)+term.Reset)
}

func (a *App) recentKeyTape(maxW int) string {
	if len(a.eng.Keystrokes) == 0 {
		return "(start typing Vim commands)"
	}
	joined := strings.Join(a.eng.Keystrokes, " ")
	rs := []rune(joined)
	if len(rs) > maxW && maxW > 3 {
		return "…" + string(rs[len(rs)-(maxW-1):])
	}
	return joined
}

func modeColor(m vim.Mode) string {
	switch m {
	case vim.ModeInsert:
		return term.Bold + term.BgGreen + term.FgBlack
	case vim.ModeVisual, vim.ModeVisualLine:
		return term.Bold + term.BgMagenta + term.FgBrightWhite
	case vim.ModeCommand:
		return term.Bold + term.BgYellow + term.FgBlack
	default:
		return term.Bold + term.BgBlue + term.FgBrightWhite
	}
}

func maxZero(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
