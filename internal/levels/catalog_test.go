package levels

import (
	"testing"

	"vimlearn/internal/vim"
)

func TestAllLevelsSolvableWithinPar(t *testing.T) {
	all := All()
	if len(all) != 25 {
		t.Fatalf("expected 25 levels (5 per difficulty), got %d", len(all))
	}

	for _, lv := range all {
		t.Run(lv.ID, func(t *testing.T) {
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
					t.Fatalf("%s: reached %d/%d checkpoints; cursor at %+v", lv.ID, cpIdx, len(lv.Checkpoints), eng.Cursor)
				}
			case GoalMatchBuffer:
				if eng.Text() != lv.TargetText {
					t.Fatalf("%s: buffer mismatch\ngot:\n%q\nwant:\n%q", lv.ID, eng.Text(), lv.TargetText)
				}
			}

			if eng.KeyCount > lv.Par {
				t.Fatalf("%s: optimal sequence took %d keys, exceeds Par %d", lv.ID, eng.KeyCount, lv.Par)
			}
		})
	}
}
