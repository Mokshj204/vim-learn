package levels

import "vimlearn/internal/vim"

// DifficultyID identifies one of the five game difficulty tiers.
type DifficultyID string

const (
	DiffBeginner     DifficultyID = "beginner"
	DiffIntermediate DifficultyID = "intermediate"
	DiffHard         DifficultyID = "hard"
	DiffExpert       DifficultyID = "expert"
	DiffImpossible   DifficultyID = "impossible"
)

// DifficultyInfo describes a difficulty tier shown in the selector.
type DifficultyInfo struct {
	ID          DifficultyID
	Name        string
	Tagline     string
	Description string
	KeysSummary string
	ColorCode   int // ANSI 256-color index
}

// GoalType specifies how a level is won.
type GoalType int

const (
	GoalMatchBuffer GoalType = iota
	GoalReachCursor
)

// KeyHint pairs a Vim keybinding with a concise explanation.
type KeyHint struct {
	Key  string
	Desc string
}

// TutorialCard explains what needs to be done and which keys solve the level.
type TutorialCard struct {
	Objective   string
	Explanation string
	Keys        []KeyHint
	Walkthrough string
}

// Level defines a playable Vim learning challenge inside a difficulty tier.
type Level struct {
	ID           string
	Difficulty   DifficultyID
	Number       int
	Title        string
	GoalType     GoalType
	StartText    string
	StartCursor  vim.Position
	TargetText   string
	Checkpoints  []vim.Position // Used when GoalType == GoalReachCursor
	Par          int
	TimeLimitSec int // 0 = untimed; >0 = countdown timer in seconds
	Tutorial     TutorialCard
	Hint         string
	OptimalSeq   string // Verified optimal solution used in tests & post-win reveal
}

// TutorialLesson represents a standalone interactive primer lesson in the Tutorial Hub.
type TutorialLesson struct {
	ID      string
	Title   string
	Summary string
	Body    []string
	Keys    []KeyHint
	Example string
}
