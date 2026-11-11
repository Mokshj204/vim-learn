package levels

import (
	"fmt"
	"math/rand"
	"strings"

	"vimlearn/internal/vim"
)

// GenerateRapidStep generates a single Monkeytype-style Rapid-Fire micro-challenge
// directly on top of the current live buffer & cursor (or seeds a fresh snippet when needed).
func GenerateRapidStep(diff DifficultyID, step int, curText string, curCursor vim.Position, rng *rand.Rand) Level {
	var baseLines []string
	curLines := strings.Split(curText, "\n")
	if curText == "" || len(curLines) < 5 || step%6 == 1 {
		themes := snippetThemes()
		theme := themes[rng.Intn(len(themes))]
		baseLines = append([]string(nil), theme.lines...)
		if curText == "" {
			curCursor = vim.Position{Row: 0, Col: 0}
		}
	} else {
		baseLines = curLines
	}

	if curCursor.Row < 0 {
		curCursor.Row = 0
	}
	if curCursor.Row >= len(baseLines) {
		curCursor.Row = len(baseLines) - 1
	}
	if len(baseLines[curCursor.Row]) == 0 {
		curCursor.Col = 0
	} else if curCursor.Col >= len(baseLines[curCursor.Row]) {
		curCursor.Col = len(baseLines[curCursor.Row]) - 1
	}

	timeLimit := 0
	if diff == DiffImpossible {
		timeLimit = 20
	}

	// Alternate between fast navigation snipes and single-line micro-edits
	if rng.Intn(2) == 0 {
		if lv, ok := rapidNavStep(diff, step, baseLines, curCursor, timeLimit, rng); ok {
			return lv
		}
	}
	return rapidEditStep(diff, step, baseLines, curCursor, timeLimit, rng)
}

func rapidNavStep(diff DifficultyID, step int, baseLines []string, curCursor vim.Position, timeLimit int, rng *rand.Rand) (Level, bool) {
	type cand struct {
		pos   vim.Position
		token string
	}
	var cands []cand
	for r, line := range baseLines {
		if r == curCursor.Row {
			continue
		}
		rs := []rune(line)
		for c := 0; c < len(rs); c++ {
			if (c == 0 || rs[c-1] == ' ' || rs[c-1] == '(' || rs[c-1] == '{' || rs[c-1] == '"' || rs[c-1] == '=') &&
				rs[c] != ' ' {
				end := c
				for end < len(rs) && rs[end] != ' ' && rs[end] != ',' && rs[end] != ')' && rs[end] != ';' {
					end++
				}
				tok := string(rs[c:end])
				if len(tok) >= 2 {
					cands = append(cands, cand{pos: vim.Position{Row: r, Col: c}, token: tok})
				}
			}
		}
	}
	if len(cands) == 0 {
		return Level{}, false
	}
	pick := cands[rng.Intn(len(cands))]

	startText := strings.Join(baseLines, "\n")
	sim := vim.NewEngine(startText, curCursor)
	var opt strings.Builder
	if pick.pos.Row > sim.Cursor.Row {
		d := pick.pos.Row - sim.Cursor.Row
		mv := "j"
		if d > 1 {
			mv = fmt.Sprintf("%dj", d)
		}
		opt.WriteString(mv)
		sim.Feed(mv)
	} else if pick.pos.Row < sim.Cursor.Row {
		d := sim.Cursor.Row - pick.pos.Row
		mv := "k"
		if d > 1 {
			mv = fmt.Sprintf("%dk", d)
		}
		opt.WriteString(mv)
		sim.Feed(mv)
	}
	if pick.pos.Col > sim.Cursor.Col {
		d := pick.pos.Col - sim.Cursor.Col
		mv := "l"
		if d > 1 {
			mv = fmt.Sprintf("%dl", d)
		}
		opt.WriteString(mv)
		sim.Feed(mv)
	} else if pick.pos.Col < sim.Cursor.Col {
		d := sim.Cursor.Col - pick.pos.Col
		mv := "h"
		if d > 1 {
			mv = fmt.Sprintf("%dh", d)
		}
		opt.WriteString(mv)
		sim.Feed(mv)
	}
	optSeq := opt.String()
	return Level{
		ID:           fmt.Sprintf("rapid-%s-%d", diff, step),
		Difficulty:   diff,
		Number:       step,
		Title:        fmt.Sprintf("Rapid Target #%d", step),
		GoalType:     GoalReachCursor,
		TimeLimitSec: timeLimit,
		StartText:    startText,
		StartCursor:  curCursor,
		Checkpoints:  []vim.Position{pick.pos},
		Par:          sim.KeyCount + 4,
		Tutorial: TutorialCard{
			Objective:   fmt.Sprintf("Jump to '%s' on line %d.", pick.token, pick.pos.Row+1),
			Explanation: "Rapid-Fire: snap onto the cyan highlight as fast as possible!",
			Keys: []KeyHint{
				{Key: "j / k", Desc: "Jump lines"},
				{Key: "w / b / f<c>", Desc: "Snap to target word/char"},
			},
			Walkthrough: "Move to the target line and land on the highlighted token.",
		},
		Hint:       fmt.Sprintf("Line %d, column %d (%s).", pick.pos.Row+1, pick.pos.Col+1, pick.token),
		OptimalSeq: optSeq,
	}, true
}

func rapidEditStep(diff DifficultyID, step int, baseLines []string, curCursor vim.Position, timeLimit int, rng *rand.Rand) Level {
	pool := mutationPoolForDifficulty(diff, rng)
	mut := pool[rng.Intn(len(pool))]

	// Pick an interior row so context lines remain around it
	insertRow := 1 + rng.Intn(len(baseLines)-1)
	if len(baseLines) <= 2 {
		insertRow = 0
	}

	var startLines, targetLines []string
	startLines = append(startLines, baseLines[:insertRow]...)
	targetLines = append(targetLines, baseLines[:insertRow]...)

	startLines = append(startLines, mut.startLines...)
	targetLines = append(targetLines, mut.targetLines...)

	// Replace 1 original line if not a pure line-insertion mutation, keeping buffer compact
	skipOrig := 1
	if len(mut.targetLines) == 0 {
		skipOrig = 0 // dd-style stray line inserted into buffer
	}
	if insertRow+skipOrig < len(baseLines) {
		startLines = append(startLines, baseLines[insertRow+skipOrig:]...)
		targetLines = append(targetLines, baseLines[insertRow+skipOrig:]...)
	}

	startText := strings.Join(startLines, "\n")
	targetText := strings.Join(targetLines, "\n")

	if curCursor.Row >= len(startLines) {
		curCursor.Row = len(startLines) - 1
	}
	if len(startLines[curCursor.Row]) == 0 {
		curCursor.Col = 0
	} else if curCursor.Col >= len(startLines[curCursor.Row]) {
		curCursor.Col = len(startLines[curCursor.Row]) - 1
	}

	sim := vim.NewEngine(startText, curCursor)
	var opt strings.Builder
	if insertRow > sim.Cursor.Row {
		d := insertRow - sim.Cursor.Row
		var mv string
		if d == 1 {
			mv = "j"
		} else {
			mv = fmt.Sprintf("%dj", d)
		}
		opt.WriteString(mv)
		sim.Feed(mv)
	} else if insertRow < sim.Cursor.Row {
		d := sim.Cursor.Row - insertRow
		var mv string
		if d == 1 {
			mv = "k"
		} else {
			mv = fmt.Sprintf("%dk", d)
		}
		opt.WriteString(mv)
		sim.Feed(mv)
	}
	if sim.Cursor.Col != 0 {
		opt.WriteString("0")
		sim.Feed("0")
	}
	opt.WriteString(mut.solveFromCol0)
	sim.Feed(mut.solveFromCol0)

	return Level{
		ID:           fmt.Sprintf("rapid-%s-%d", diff, step),
		Difficulty:   diff,
		Number:       step,
		Title:        fmt.Sprintf("Rapid Edit #%d", step),
		GoalType:     GoalMatchBuffer,
		TimeLimitSec: timeLimit,
		StartText:    startText,
		StartCursor:  curCursor,
		TargetText:   targetText,
		Par:          sim.KeyCount + 2,
		Tutorial: TutorialCard{
			Objective:   fmt.Sprintf("Line %d: %s", insertRow+1, mut.objective),
			Explanation: "Rapid-Fire: complete this edit to chain straight to the next target!",
			Keys:        mut.keys,
			Walkthrough: fmt.Sprintf("Go to line %d and apply the edit.", insertRow+1),
		},
		Hint:       fmt.Sprintf("Target is on line %d.", insertRow+1),
		OptimalSeq: opt.String(),
	}
}
