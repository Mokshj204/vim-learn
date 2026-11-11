package game

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"vimlearn/internal/levels"
	"vimlearn/internal/term"
	"vimlearn/internal/vim"
)

type mockTerm struct {
	in   *bytes.Buffer
	out  *bytes.Buffer
	ch   chan struct{}
	cols int
	rows int
}

func newMockTerm() *mockTerm {
	return &mockTerm{
		in:   &bytes.Buffer{},
		out:  &bytes.Buffer{},
		ch:   make(chan struct{}),
		cols: 100,
		rows: 32,
	}
}

func (m *mockTerm) EnableRaw() error            { return nil }
func (m *mockTerm) Restore() error              { return nil }
func (m *mockTerm) Size() (int, int, error)     { return m.cols, m.rows, nil }
func (m *mockTerm) Reader() io.Reader           { return m.in }
func (m *mockTerm) Writer() io.Writer           { return m.out }
func (m *mockTerm) ResizeNotify() <-chan struct{} { return m.ch }

func TestAppNavigationAndGameplay(t *testing.T) {
	mt := newMockTerm()
	app := NewApp(mt)

	// Home renders cleanly
	homeOut := app.render()
	if !strings.Contains(homeOut, "Beginner") || !strings.Contains(homeOut, "Impossible") || !strings.Contains(homeOut, "Press 'e'") {
		t.Fatalf("home screen missing difficulties or endless mode hint")
	}

	// Open Tutorial
	app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: 't'})
	if app.screen != ScreenTutorial {
		t.Fatalf("expected ScreenTutorial, got %v", app.screen)
	}
	tutOut := app.render()
	if !strings.Contains(tutOut, "Modal Editing") {
		t.Fatalf("tutorial screen missing lesson title")
	}

	// Return home and open Hard difficulty (key '3')
	app.handleEvent(term.KeyEvent{Kind: term.KeyEsc})
	if app.screen != ScreenHome {
		t.Fatalf("expected ScreenHome")
	}
	app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: '3'})
	if app.screen != ScreenLevels || app.selDiff != levels.DiffHard {
		t.Fatalf("expected Hard level select, got %v %v", app.screen, app.selDiff)
	}
	lvlOut := app.render()
	if !strings.Contains(lvlOut, "Inside the Quotes") {
		t.Fatalf("hard level select missing expected level")
	}

	// Start Hard Level 4 (Brace Jump & Join) and solve it!
	if err := app.OpenLevel(levels.DiffHard, 4); err != nil {
		t.Fatal(err)
	}
	playOut := app.render()
	if !strings.Contains(playOut, "What To Do") {
		t.Fatalf("play screen missing briefing")
	}

	// Toggle lesson drawer with Tab
	app.handleEvent(term.KeyEvent{Kind: term.KeyTab})
	if !app.showLesson {
		t.Fatalf("expected Tab to toggle showLesson")
	}

	// Execute optimal solution
	app.eng.Feed("ddggjJ")
	if app.eng.Text() != app.currentLevel.TargetText {
		t.Fatalf("expected target match, got %q", app.eng.Text())
	}

	// Verify handlePlayKey transitions to ScreenWin
	if err := app.OpenLevel(levels.DiffHard, 4); err != nil {
		t.Fatal(err)
	}
	for _, k := range []rune{'d', 'd', 'g', 'g', 'j', 'J'} {
		app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: k})
	}
	if app.screen != ScreenWin {
		t.Fatalf("expected ScreenWin after solving level, got %v", app.screen)
	}
	winOut := app.render()
	if !strings.Contains(winOut, "LEVEL COMPLETE!") {
		t.Fatalf("win banner missing")
	}
}

func TestResponsiveTerminalSizing(t *testing.T) {
	mt := newMockTerm()
	mt.cols = 148
	mt.rows = 42
	app := NewApp(mt)

	// Home screen fills down to row 42
	if out := app.render(); !strings.Contains(out, "\x1b[42;1H") {
		t.Fatalf("expected home screen to render footer at row 42")
	}

	// Tutorial stretches down to row 42 and uses full width (box border > 120 dashes)
	app.OpenTutorial()
	tutOut := app.render()
	if !strings.Contains(tutOut, "\x1b[42;1H") {
		t.Fatalf("expected tutorial screen to fill to row 42")
	}
	if !strings.Contains(tutOut, strings.Repeat("─", 100)) {
		t.Fatalf("expected tutorial box to expand beyond 100 cols")
	}

	// Gameplay stretches down to row 42
	if err := app.OpenLevel(levels.DiffBeginner, 3); err != nil {
		t.Fatal(err)
	}
	playOut := app.render()
	if !strings.Contains(playOut, "\x1b[42;1H") {
		t.Fatalf("expected play screen to render footer at row 42")
	}
}

func TestMinimumTerminalSizeEnforcement(t *testing.T) {
	mt := newMockTerm()
	mt.cols = 60
	mt.rows = 18
	app := NewApp(mt)

	// Refuses to start below 80x24
	if err := app.Run(); err == nil || !strings.Contains(err.Error(), "80x24") {
		t.Fatalf("expected minimum terminal size error at 60x18, got %v", err)
	}

	// If shrunk while running, renders resize warning and blocks gameplay inputs
	if out := app.render(); !strings.Contains(out, "Terminal Too Small") {
		t.Fatalf("expected Terminal Too Small warning, got %q", out)
	}
	if quit := app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: '1'}); quit {
		t.Fatalf("did not expect non-quit key to quit while too small")
	}
	if app.screen != ScreenHome {
		t.Fatalf("expected screen to stay ScreenHome while below minimum size")
	}
	if quit := app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: 'q'}); !quit {
		t.Fatalf("expected 'q' to quit while terminal is too small")
	}
}

func TestEndlessModeFlow(t *testing.T) {
	mt := newMockTerm()
	app := NewApp(mt)

	// Open Hard level select and press 'e' for Endless Game Mode Picker
	app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: '3'})
	app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: 'e'})
	if app.screen != ScreenEndlessMenu {
		t.Fatalf("expected ScreenEndlessMenu, got %v", app.screen)
	}
	menuOut := app.render()
	if !strings.Contains(menuOut, "Random Mix") || !strings.Contains(menuOut, "Multi-Edit Mix") ||
		!strings.Contains(menuOut, "Code Hunt") || !strings.Contains(menuOut, "Rapid-Fire Chain") {
		t.Fatalf("endless mode picker missing modes")
	}

	// Start Multi-Edit Mix explicitly
	if err := app.StartEndlessMode(levels.DiffHard, levels.EndlessMultiEdit); err != nil {
		t.Fatal(err)
	}
	if !app.endlessMode || app.endlessStreak != 0 || app.screen != ScreenPlay {
		t.Fatalf("unexpected initial endless state: mode=%v streak=%d screen=%v", app.endlessMode, app.endlessStreak, app.screen)
	}
	if out := app.render(); !strings.Contains(out, "MULTI-EDIT") {
		t.Fatalf("expected MULTI-EDIT badge on play screen")
	}

	// Solve round 1 using its generated OptimalSeq
	for _, ev := range vimKeys(app.currentLevel.OptimalSeq) {
		app.handleEvent(ev)
	}
	if app.screen != ScreenWin || app.endlessStreak != 1 {
		t.Fatalf("expected ScreenWin and streak=1, got screen=%v streak=%d", app.screen, app.endlessStreak)
	}
	if winOut := app.render(); !strings.Contains(winOut, "ROUND CLEARED!") {
		t.Fatalf("expected ROUND CLEARED! banner")
	}

	// Advance to round 2 with Enter
	app.handleEvent(term.KeyEvent{Kind: term.KeyEnter})
	if app.screen != ScreenPlay {
		t.Fatalf("expected round 2 on ScreenPlay, got screen=%v", app.screen)
	}

	// Test Rapid-Fire Chain continuous stepping without ScreenWin
	if err := app.StartEndlessMode(levels.DiffBeginner, levels.EndlessRapid); err != nil {
		t.Fatal(err)
	}
	if out := app.render(); !strings.Contains(out, "RAPID-FIRE") {
		t.Fatalf("expected RAPID-FIRE badge on play screen")
	}
	for _, ev := range vimKeys(app.currentLevel.OptimalSeq) {
		app.handleEvent(ev)
	}
	if app.screen != ScreenPlay || app.endlessStreak != 1 || app.rapidStep != 2 {
		t.Fatalf("expected Rapid-Fire to stay on ScreenPlay and advance to step 2, got screen=%v streak=%d step=%d", app.screen, app.endlessStreak, app.rapidStep)
	}

	// Test Random Mix anti-repeat bag never picks same mode twice in a row,
	// and switches immediately after solving each round (including Rapid-Fire).
	if err := app.StartEndlessMode(levels.DiffIntermediate, levels.EndlessRandom); err != nil {
		t.Fatal(err)
	}
	prev := app.endlessActiveMode
	for i := 0; i < 6; i++ {
		for _, ev := range vimKeys(app.currentLevel.OptimalSeq) {
			app.handleEvent(ev)
		}
		if app.screen != ScreenWin {
			t.Fatalf("Random Mix round %d (%v) did not reach ScreenWin after solving", i+1, prev)
		}
		app.handleEvent(term.KeyEvent{Kind: term.KeyEnter})
		if app.screen != ScreenPlay || app.endlessActiveMode == prev {
			t.Fatalf("Random Mix repeated mode %v or failed to advance at round %d", prev, i+1)
		}
		prev = app.endlessActiveMode
	}

	// Quit endless mode back to Level Select
	app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: ':'})
	app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: 'q'})
	app.handleEvent(term.KeyEvent{Kind: term.KeyEnter})
	if app.endlessMode || app.screen != ScreenLevels {
		t.Fatalf("expected :q to exit endless mode to ScreenLevels, got mode=%v screen=%v", app.endlessMode, app.screen)
	}
}

func vimKeys(seq string) []term.KeyEvent {
	return vim.ParseKeySequence(seq)
}
