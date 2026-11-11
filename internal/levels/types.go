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

// EndlessMode identifies a gameplay sub-mode inside Endless Muscle-Memory Mode.
type EndlessMode string

const (
	EndlessMultiEdit EndlessMode = "multi"
	EndlessCodeHunt  EndlessMode = "hunt"
	EndlessRapid     EndlessMode = "rapid"
	EndlessRandom    EndlessMode = "random"
)

// EndlessModeInfo describes an Endless Mode option in the mode picker.
type EndlessModeInfo struct {
	ID          EndlessMode
	Name        string
	Tagline     string
	Description string
	ColorCode   int
}

// EndlessModes returns all selectable Endless sub-modes in display order.
func EndlessModes() []EndlessModeInfo {
	return []EndlessModeInfo{
		{
			ID:          EndlessRandom,
			Name:        "Random Mix",
			Tagline:     "Shuffled game modes every round (Recommended)",
			Description: "Alternates between Multi-Edit Mix, Code Hunt, and Rapid-Fire Bursts using an anti-repeat bag so no two rounds feel the same.",
			ColorCode:   213,
		},
		{
			ID:          EndlessMultiEdit,
			Name:        "Multi-Edit Mix",
			Tagline:     "Composable multi-step edits on real code & configs",
			Description: "Combines 1 to 3+ random mutations across realistic snippets with varied cursor starts and streak escalation.",
			ColorCode:   39,
		},
		{
			ID:          EndlessCodeHunt,
			Name:        "Code Hunt",
			Tagline:     "High-speed navigation across real code landmarks",
			Description: "Jump between highlighted identifiers and delimiters inside realistic code blocks instead of blank grids.",
			ColorCode:   46,
		},
		{
			ID:          EndlessRapid,
			Name:        "Rapid-Fire Chain",
			Tagline:     "Continuous Monkeytype-style micro-drills on a live buffer",
			Description: "Hit one target and the next lights up immediately on the same buffer. Tracks chain streak and Keys/sec!",
			ColorCode:   220,
		},
	}
}
