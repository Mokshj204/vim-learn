package game

import (
	"fmt"
	"io"
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
	ScreenPlay
	ScreenWin
	ScreenTimeout
)

// App coordinates the interactive TUI screens, Vim engine, timer, and persistence.
type App struct {
	term     term.Terminal
	progress *store.Progress

	screen   Screen
	homeIdx  int // 0 = Tutorial, 1..5 = Difficulties
	tutIdx   int
	selDiff  levels.DifficultyID
	levelIdx int

	currentLevel  levels.Level
	eng           *vim.Engine
	checkpointIdx int
	showLesson    bool
	showHint      bool
	showSolution  bool
	levelStart    time.Time
}

// NewApp constructs a new game application.
func NewApp(t term.Terminal) *App {
	return &App{
		term:     t,
		progress: store.Load(),
		screen:   ScreenHome,
		homeIdx:  1,
		selDiff:  levels.DiffBeginner,
	}
}

// OpenLevel starts directly on a specific difficulty and 1-based level number.
func (a *App) OpenLevel(diff levels.DifficultyID, num int) error {
	lv, ok := levels.Find(diff, num)
	if !ok {
		return fmt.Errorf("level %s #%d not found", diff, num)
	}
	a.selDiff = diff
	a.levelIdx = num - 1
	a.startLevel(lv)
	return nil
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
		case '1', '2', '3', '4', '5':
			idx := int(ev.Rune - '1')
			if idx >= 0 && idx < len(dLevels) {
				a.levelIdx = idx
				a.startLevel(dLevels[idx])
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
				a.progress.RecordWin(a.currentLevel.ID, a.eng.KeyCount, a.currentLevel.Par)
				a.screen = ScreenWin
			} else {
				a.eng.StatusMsg = fmt.Sprintf("Checkpoint %d/%d reached!", a.checkpointIdx, len(a.currentLevel.Checkpoints))
			}
		}
	} else if a.eng.Text() == a.currentLevel.TargetText {
		a.progress.RecordWin(a.currentLevel.ID, a.eng.KeyCount, a.currentLevel.Par)
		a.screen = ScreenWin
	}
}

func (a *App) handleWinKey(ev term.KeyEvent) {
	switch ev.Kind {
	case term.KeyEsc:
		a.screen = ScreenLevels
	case term.KeyEnter:
		a.advanceNextLevel()
	case term.KeyRune:
		switch ev.Rune {
		case 'n':
			a.advanceNextLevel()
		case 'r':
			a.startLevel(a.currentLevel)
		case 's':
			a.showSolution = !a.showSolution
		case 'l', 'q':
			a.screen = ScreenLevels
		}
	}
}

func (a *App) handleTimeoutKey(ev term.KeyEvent) {
	switch ev.Kind {
	case term.KeyEsc:
		a.screen = ScreenLevels
	case term.KeyEnter:
		a.startLevel(a.currentLevel)
	case term.KeyRune:
		switch ev.Rune {
		case 'r':
			a.startLevel(a.currentLevel)
		case 'l', 'q':
			a.screen = ScreenLevels
		}
	}
}

func (a *App) advanceNextLevel() {
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
