package levels

import (
	"math/rand"
	"testing"

	"vimlearn/internal/vim"
)

func verifyGeneratedLevel(t *testing.T, lv Level) {
	t.Helper()
	eng := vim.NewEngine(lv.StartText, lv.StartCursor)
	cpIdx := 0

	for _, ev := range vim.ParseKeySequence(lv.OptimalSeq) {
		eng.HandleKey(ev)
		if lv.GoalType == GoalReachCursor && cpIdx < len(lv.Checkpoints) {
			if eng.Cursor == lv.Checkpoints[cpIdx] {
				cpIdx++
			}
		}
	}

	switch lv.GoalType {
	case GoalReachCursor:
		if cpIdx != len(lv.Checkpoints) {
			t.Fatalf("%s (%s): reached %d/%d checkpoints; seq=%q", lv.ID, lv.Title, cpIdx, len(lv.Checkpoints), lv.OptimalSeq)
		}
	case GoalMatchBuffer:
		if eng.Text() != lv.TargetText {
			t.Fatalf("%s (%s): buffer mismatch\nseq=%q\ngot:\n%q\nwant:\n%q", lv.ID, lv.Title, lv.OptimalSeq, eng.Text(), lv.TargetText)
		}
	}

	if eng.KeyCount > lv.Par {
		t.Fatalf("%s (%s): seq %q took %d keys > Par %d", lv.ID, lv.Title, lv.OptimalSeq, eng.KeyCount, lv.Par)
	}
}

func TestEndlessGeneratorSolvableWithinPar(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for _, d := range Difficulties() {
		for round := 1; round <= 60; round++ {
			verifyGeneratedLevel(t, GenerateEndlessForMode(d.ID, EndlessMultiEdit, round, rng))
			verifyGeneratedLevel(t, GenerateEndlessForMode(d.ID, EndlessCodeHunt, round, rng))
		}
	}
}

func TestRapidFireChainSolvableWithinPar(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	for _, d := range Difficulties() {
		curText := ""
		curCursor := vim.Position{Row: 0, Col: 0}
		for step := 1; step <= 50; step++ {
			lv := GenerateRapidStep(d.ID, step, curText, curCursor, rng)
			verifyGeneratedLevel(t, lv)
			eng := vim.NewEngine(lv.StartText, lv.StartCursor)
			eng.Feed(lv.OptimalSeq)
			curText = eng.Text()
			curCursor = eng.Cursor
		}
	}
}
