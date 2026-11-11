package game

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"vimlearn/internal/levels"
	"vimlearn/internal/store"
	"vimlearn/internal/term"
	"vimlearn/internal/vim"
)

// Minimum terminal dimensions required to start and play vimlearn.
const (
	MinCols = 80
	MinRows = 24
)

// Screen identifies the current UI screen.
type Screen int

const (
	ScreenHome Screen = iota
	ScreenTutorial
	ScreenLevels
	ScreenEndlessMenu
	ScreenPlay
	ScreenWin
	ScreenTimeout
)

// App coordinates the interactive TUI screens, Vim engine, timer, and persistence.
type App struct {
	term     term.Terminal
	progress *store.Progress

	screen         Screen
	homeIdx        int // 0 = Tutorial, 1..5 = Difficulties
	tutIdx         int
	selDiff        levels.DifficultyID
	levelIdx       int
	endlessMenuIdx int

	currentLevel  levels.Level
	eng           *vim.Engine
	checkpointIdx int
	showLesson    bool
	showHint      bool
	showSolution  bool
	levelStart    time.Time

	endlessMode         bool
	endlessSelectedMode levels.EndlessMode
	endlessActiveMode   levels.EndlessMode
	endlessBag          []levels.EndlessMode
	endlessStreak       int
	rapidStep           int
	rapidBurstDone      int
	rapidBurstTotal     int
	rapidTotalKeys      int
	rapidStart          time.Time
	rng                 *rand.Rand
}

// NewApp constructs a new game application.
func NewApp(t term.Terminal) *App {
	return &App{
		term:     t,
		progress: store.Load(),
		screen:   ScreenHome,
		homeIdx:  1,
		selDiff:  levels.DiffBeginner,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// OpenLevel starts directly on a specific difficulty and 1-based level number.
func (a *App) OpenLevel(diff levels.DifficultyID, num int) error {
	lv, ok := levels.Find(diff, num)
	if !ok {
		return fmt.Errorf("level %s #%d not found", diff, num)
	}
	a.endlessMode = false
	a.selDiff = diff
	a.levelIdx = num - 1
	a.startLevel(lv)
	return nil
}

// StartEndless begins an indefinite procedural muscle-memory run in Random Mix mode.
func (a *App) StartEndless(diff levels.DifficultyID) error {
	return a.StartEndlessMode(diff, levels.EndlessRandom)
}

// StartEndlessMode begins an indefinite run for a difficulty and Endless sub-mode.
func (a *App) StartEndlessMode(diff levels.DifficultyID, mode levels.EndlessMode) error {
	if _, ok := levels.DifficultyByID(diff); !ok {
		return fmt.Errorf("unknown difficulty %q", diff)
	}
	switch mode {
	case "", levels.EndlessRandom:
		mode = levels.EndlessRandom
	case levels.EndlessMultiEdit, levels.EndlessCodeHunt, levels.EndlessRapid:
	default:
		return fmt.Errorf("unknown endless mode %q (use random, multi, hunt, or rapid)", mode)
	}
	a.selDiff = diff
	a.endlessMode = true
	a.endlessSelectedMode = mode
	a.endlessActiveMode = ""
	a.endlessBag = nil
	a.endlessStreak = 0
	a.rapidStep = 0
	a.rapidTotalKeys = 0
	a.rapidStart = time.Now()
	a.nextEndlessRound()
	return nil
}

func (a *App) drawBagMode() levels.EndlessMode {
	if len(a.endlessBag) == 0 {
		a.endlessBag = []levels.EndlessMode{
			levels.EndlessMultiEdit,
			levels.EndlessCodeHunt,
			levels.EndlessRapid,
		}
		a.rng.Shuffle(len(a.endlessBag), func(i, j int) {
			a.endlessBag[i], a.endlessBag[j] = a.endlessBag[j], a.endlessBag[i]
		})
		// Avoid repeating the previous mode across bag boundary
		if a.endlessActiveMode != "" && a.endlessBag[0] == a.endlessActiveMode && len(a.endlessBag) > 1 {
			a.endlessBag[0], a.endlessBag[1] = a.endlessBag[1], a.endlessBag[0]
		}
	}
	picked := a.endlessBag[0]
	a.endlessBag = a.endlessBag[1:]
	return picked
}

func (a *App) nextEndlessRound() {
	mode := a.endlessSelectedMode
	if mode == levels.EndlessRandom {
		mode = a.drawBagMode()
	}
	a.endlessActiveMode = mode

	if mode == levels.EndlessRapid {
		a.rapidStep = 1
		a.rapidBurstDone = 0
		if a.endlessSelectedMode == levels.EndlessRandom {
			a.rapidBurstTotal = 1
		} else {
			a.rapidBurstTotal = 0 // infinite continuous chain
		}
		lv := levels.GenerateRapidStep(a.selDiff, a.rapidStep, "", vim.Position{}, a.rng)
		a.startLevel(lv)
		return
	}

	lv := levels.GenerateEndlessForMode(a.selDiff, mode, a.endlessStreak+1, a.rng)
	a.startLevel(lv)
}

func (a *App) applyNextRapidStep(flashMsg string) {
	a.rapidStep++
	curText := a.eng.Text()
	curCursor := a.eng.Cursor
	lv := levels.GenerateRapidStep(a.selDiff, a.rapidStep, curText, curCursor, a.rng)
	a.startLevel(lv)
	a.eng.StatusMsg = flashMsg
}

func (a *App) rapidKPS() float64 {
	elapsed := time.Since(a.rapidStart).Seconds()
	if elapsed < 0.5 {
		elapsed = 0.5
	}
	total := a.rapidTotalKeys
	if a.eng != nil {
		total += a.eng.KeyCount
	}
	return float64(total) / elapsed
}

// OpenTutorial starts on the interactive Vim tutorial screen.
func (a *App) OpenTutorial() {
	a.screen = ScreenTutorial
	a.tutIdx = 0
}

func (a *App) startLevel(lv levels.Level) {
	a.currentLevel = lv
	a.eng = vim.NewEngine(lv.StartText, lv.StartCursor)
	a.checkpointIdx = 0
	a.showLesson = false
	a.showHint = false
	a.showSolution = false
	a.levelStart = time.Now()
	a.screen = ScreenPlay
}

func (a *App) remainingSeconds() int {
	if a.currentLevel.TimeLimitSec <= 0 {
		return 0
	}
	elapsed := int(time.Since(a.levelStart).Seconds())
	rem := a.currentLevel.TimeLimitSec - elapsed
	if rem < 0 {
		return 0
	}
	return rem
}

type keyMsg struct {
	ev  term.KeyEvent
	err error
}

// Run enters raw ANSI terminal mode and runs the interactive game loop.
func (a *App) Run() error {
	if cols, rows, err := a.term.Size(); err == nil && (cols < MinCols || rows < MinRows) {
		return fmt.Errorf("terminal size %dx%d is too small; minimum required is %dx%d — please resize your terminal and try again", cols, rows, MinCols, MinRows)
	}

	if err := a.term.EnableRaw(); err != nil {
		return err
	}
	defer func() {
		_, _ = io.WriteString(a.term.Writer(), term.Reset+term.CursorShow+term.AltScreenOff)
		_ = a.term.Restore()
	}()

	_, _ = io.WriteString(a.term.Writer(), term.AltScreenOn+term.CursorHide+term.ClearScreen)

	kr := term.NewKeyReader(a.term.Reader())
	keyCh := make(chan keyMsg, 32)
	go func() {
		for {
			ev, err := kr.ReadKey()
			keyCh <- keyMsg{ev: ev, err: err}
			if err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	a.draw()

	for {
		select {
		case msg := <-keyCh:
			if msg.err != nil {
				if msg.err == io.EOF {
					return nil
				}
				return msg.err
			}
			if a.handleEvent(msg.ev) {
				return nil
			}
			a.draw()

		case <-a.term.ResizeNotify():
			a.draw()

		case <-ticker.C:
			if a.screen == ScreenPlay && a.currentLevel.TimeLimitSec > 0 {
				if a.remainingSeconds() == 0 {
					a.screen = ScreenTimeout
				}
				a.draw()
			}
		}
	}
}

func (a *App) draw() {
	frame := a.render()
	_, _ = io.WriteString(a.term.Writer(), frame)
}

func (a *App) handleEvent(ev term.KeyEvent) (quit bool) {
	if cols, rows, err := a.term.Size(); err == nil && (cols < MinCols || rows < MinRows) {
		if ev.Kind == term.KeyCtrl && ev.Ctrl == 'c' {
			return true
		}
		if ev.Kind == term.KeyRune && ev.Rune == 'q' {
			return true
		}
		return false
	}

	switch a.screen {
	case ScreenHome:
		return a.handleHomeKey(ev)
	case ScreenTutorial:
		a.handleTutorialKey(ev)
	case ScreenLevels:
		a.handleLevelSelectKey(ev)
	case ScreenEndlessMenu:
		a.handleEndlessMenuKey(ev)
	case ScreenPlay:
		a.handlePlayKey(ev)
	case ScreenWin:
		a.handleWinKey(ev)
	case ScreenTimeout:
		a.handleTimeoutKey(ev)
	}
	return false
}

func (a *App) handleHomeKey(ev term.KeyEvent) bool {
	maxIdx := len(levels.Difficulties()) // 0 = tutorial, 1..5 = difficulties
	switch ev.Kind {
	case term.KeyUp:
		if a.homeIdx > 0 {
			a.homeIdx--
		}
	case term.KeyDown:
		if a.homeIdx < maxIdx {
			a.homeIdx++
		}
	case term.KeyEnter:
		if a.homeIdx == 0 {
			a.screen = ScreenTutorial
			return false
		}
		diffs := levels.Difficulties()
		a.selDiff = diffs[a.homeIdx-1].ID
		a.levelIdx = 0
		a.screen = ScreenLevels
	case term.KeyCtrl:
		if ev.Ctrl == 'c' {
			return true
		}
	case term.KeyRune:
		switch ev.Rune {
		case 'q':
			return true
		case 'k':
			if a.homeIdx > 0 {
				a.homeIdx--
			}
		case 'j':
			if a.homeIdx < maxIdx {
				a.homeIdx++
			}
		case 't', 'T':
			a.screen = ScreenTutorial
		case 'e', 'E':
			if a.homeIdx >= 1 {
				diffs := levels.Difficulties()
				a.selDiff = diffs[a.homeIdx-1].ID
				a.endlessMenuIdx = 0
				a.screen = ScreenEndlessMenu
			}
		case '1', '2', '3', '4', '5':
			idx := int(ev.Rune - '1')
			diffs := levels.Difficulties()
			if idx >= 0 && idx < len(diffs) {
				a.homeIdx = idx + 1
				a.selDiff = diffs[idx].ID
				a.levelIdx = 0
				a.screen = ScreenLevels
			}
		}
	}
	return false
}

func (a *App) handleTutorialKey(ev term.KeyEvent) {
	lessons := levels.Tutorials()
	switch ev.Kind {
	case term.KeyEsc:
		a.screen = ScreenHome
	case term.KeyLeft:
		if a.tutIdx > 0 {
			a.tutIdx--
		}
	case term.KeyRight, term.KeyEnter, term.KeyTab:
		a.tutIdx = (a.tutIdx + 1) % len(lessons)
	case term.KeyCtrl:
		if ev.Ctrl == 'c' {
			a.screen = ScreenHome
		}
	case term.KeyRune:
		switch ev.Rune {
		case 'q', 'b':
			a.screen = ScreenHome
		case 'h', 'k':
			if a.tutIdx > 0 {
				a.tutIdx--
			}
		case 'l', 'j':
			if a.tutIdx < len(lessons)-1 {
				a.tutIdx++
			}
		case '1', '2', '3', '4':
			idx := int(ev.Rune - '1')
			if idx >= 0 && idx < len(lessons) {
				a.tutIdx = idx
			}
		}
	}
}

func (a *App) handleLevelSelectKey(ev term.KeyEvent) {
	dLevels := levels.ByDifficulty(a.selDiff)
	switch ev.Kind {
	case term.KeyEsc:
		a.screen = ScreenHome
	case term.KeyUp:
		if a.levelIdx > 0 {
			a.levelIdx--
		}
	case term.KeyDown:
		if a.levelIdx < len(dLevels)-1 {
			a.levelIdx++
		}
	case term.KeyEnter:
		if a.levelIdx >= 0 && a.levelIdx < len(dLevels) {
			a.endlessMode = false
			a.startLevel(dLevels[a.levelIdx])
		}
	case term.KeyCtrl:
		if ev.Ctrl == 'c' {
			a.screen = ScreenHome
		}
	case term.KeyRune:
		switch ev.Rune {
		case 'q', 'b':
			a.screen = ScreenHome
		case 'k':
			if a.levelIdx > 0 {
				a.levelIdx--
			}
		case 'j':
			if a.levelIdx < len(dLevels)-1 {
				a.levelIdx++
			}
		case 't', 'T':
			a.screen = ScreenTutorial
		case 'e', 'E':
			a.endlessMenuIdx = 0
			a.screen = ScreenEndlessMenu
		case '1', '2', '3', '4', '5':
			idx := int(ev.Rune - '1')
			if idx >= 0 && idx < len(dLevels) {
				a.endlessMode = false
				a.levelIdx = idx
				a.startLevel(dLevels[idx])
			}
		}
	}
}

func (a *App) handleEndlessMenuKey(ev term.KeyEvent) {
	modes := levels.EndlessModes()
	switch ev.Kind {
	case term.KeyEsc:
		a.screen = ScreenLevels
	case term.KeyUp:
		if a.endlessMenuIdx > 0 {
			a.endlessMenuIdx--
		}
	case term.KeyDown:
		if a.endlessMenuIdx < len(modes)-1 {
			a.endlessMenuIdx++
		}
	case term.KeyEnter:
		if a.endlessMenuIdx >= 0 && a.endlessMenuIdx < len(modes) {
			_ = a.StartEndlessMode(a.selDiff, modes[a.endlessMenuIdx].ID)
		}
	case term.KeyCtrl:
		if ev.Ctrl == 'c' {
			a.screen = ScreenLevels
		}
	case term.KeyRune:
		switch ev.Rune {
		case 'q', 'b':
			a.screen = ScreenLevels
		case 'k':
			if a.endlessMenuIdx > 0 {
				a.endlessMenuIdx--
			}
		case 'j':
			if a.endlessMenuIdx < len(modes)-1 {
				a.endlessMenuIdx++
			}
		case '1', '2', '3', '4':
			idx := int(ev.Rune - '1')
			if idx >= 0 && idx < len(modes) {
				a.endlessMenuIdx = idx
				_ = a.StartEndlessMode(a.selDiff, modes[idx].ID)
			}
		}
	}
}

func (a *App) handlePlayKey(ev term.KeyEvent) {
	// Non-colliding UI toggles
	if ev.Kind == term.KeyTab && a.eng.Mode != vim.ModeInsert {
		a.showLesson = !a.showLesson
		return
	}
	if ev.Kind == term.KeyCtrl {
		if ev.Ctrl == 'l' {
			a.showLesson = true
			a.showHint = !a.showHint
			return
		}
		if ev.Ctrl == 'c' && a.eng.Mode == vim.ModeNormal && a.eng.Operator == 0 && a.eng.WaitingSub == "" {
			a.eng.HandleKey(ev)
			a.eng.StatusMsg = "In Normal mode — type :q<Enter> to exit level, or :reset<Enter> to restart"
			return
		}
	}

	a.eng.HandleKey(ev)

	if a.eng.QuitRequested {
		a.endlessMode = false
		a.screen = ScreenLevels
		return
	}
	if a.eng.ResetRequested {
		a.startLevel(a.currentLevel)
		return
	}

	// Check win conditions
	if a.currentLevel.GoalType == levels.GoalReachCursor {
		if a.checkpointIdx < len(a.currentLevel.Checkpoints) &&
			a.eng.Cursor == a.currentLevel.Checkpoints[a.checkpointIdx] {
			a.checkpointIdx++
			if a.checkpointIdx >= len(a.currentLevel.Checkpoints) {
				a.recordWin()
			} else {
				a.eng.StatusMsg = fmt.Sprintf("Checkpoint %d/%d reached!", a.checkpointIdx, len(a.currentLevel.Checkpoints))
			}
		}
	} else if a.eng.Text() == a.currentLevel.TargetText {
		a.recordWin()
	}
}

func (a *App) recordWin() {
	if a.endlessMode {
		a.endlessStreak++
		a.progress.RecordEndlessStreak(string(a.selDiff), a.endlessStreak)
		a.progress.RecordEndlessStreak(string(a.selDiff)+":"+string(a.endlessSelectedMode), a.endlessStreak)

		if a.endlessActiveMode == levels.EndlessRapid {
			a.rapidTotalKeys += a.eng.KeyCount
			if a.rapidBurstTotal > 0 {
				a.rapidBurstDone++
				if a.rapidBurstDone < a.rapidBurstTotal {
					a.applyNextRapidStep(fmt.Sprintf("✓ Burst %d/%d cleared! Next target...", a.rapidBurstDone, a.rapidBurstTotal))
					return
				}
			} else {
				a.applyNextRapidStep(fmt.Sprintf("✓ +1! Chain: %d  (%.1f keys/s)", a.endlessStreak, a.rapidKPS()))
				return
			}
		}
	} else {
		a.progress.RecordWin(a.currentLevel.ID, a.eng.KeyCount, a.currentLevel.Par)
	}
	a.screen = ScreenWin
}

func (a *App) handleWinKey(ev term.KeyEvent) {
	switch ev.Kind {
	case term.KeyEsc:
		a.endlessMode = false
		a.screen = ScreenLevels
	case term.KeyEnter:
		a.advanceNextLevel()
	case term.KeyRune:
		switch ev.Rune {
		case 'n', ' ':
			a.advanceNextLevel()
		case 'r':
			a.startLevel(a.currentLevel)
		case 's':
			a.showSolution = !a.showSolution
		case 'l', 'q':
			a.endlessMode = false
			a.screen = ScreenLevels
		}
	}
}

func (a *App) handleTimeoutKey(ev term.KeyEvent) {
	switch ev.Kind {
	case term.KeyEsc:
		a.endlessMode = false
		a.screen = ScreenLevels
	case term.KeyEnter:
		if a.endlessMode {
			_ = a.StartEndlessMode(a.selDiff, a.endlessSelectedMode)
		} else {
			a.startLevel(a.currentLevel)
		}
	case term.KeyRune:
		switch ev.Rune {
		case 'r':
			if a.endlessMode {
				_ = a.StartEndlessMode(a.selDiff, a.endlessSelectedMode)
			} else {
				a.startLevel(a.currentLevel)
			}
		case 'l', 'q':
			a.endlessMode = false
			a.screen = ScreenLevels
		}
	}
}

func (a *App) advanceNextLevel() {
	if a.endlessMode {
		a.nextEndlessRound()
		return
	}
	dLevels := levels.ByDifficulty(a.selDiff)
	if a.levelIdx+1 < len(dLevels) {
		a.levelIdx++
		a.startLevel(dLevels[a.levelIdx])
		return
	}
	// Advance to next difficulty if available
	diffs := levels.Difficulties()
	for i, d := range diffs {
		if d.ID == a.selDiff && i+1 < len(diffs) {
			a.selDiff = diffs[i+1].ID
			a.levelIdx = 0
			nextLevels := levels.ByDifficulty(a.selDiff)
			if len(nextLevels) > 0 {
				a.startLevel(nextLevels[0])
				return
			}
		}
	}
	a.screen = ScreenLevels
}
