package game

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"vimlearn/internal/levels"
	"vimlearn/internal/term"
	"vimlearn/internal/updater"
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

	// Test Random Mix across all 5 difficulties: anti-repeat bag never repeats mode,
	// and every round reaches ScreenWin even when OptimalSeq ends with <Esc>.
	for _, d := range levels.Difficulties() {
		if err := app.StartEndlessMode(d.ID, levels.EndlessRandom); err != nil {
			t.Fatal(err)
		}
		prev := app.endlessActiveMode
		for i := 0; i < 20; i++ {
			for _, ev := range vimKeys(app.currentLevel.OptimalSeq) {
				app.handleEvent(ev)
			}
			if app.screen != ScreenWin {
				t.Fatalf("%s Random Mix round %d (%v, seq=%q) did not reach ScreenWin (screen=%v)", d.ID, i+1, prev, app.currentLevel.OptimalSeq, app.screen)
			}
			app.handleEvent(term.KeyEvent{Kind: term.KeyEnter})
			if app.screen != ScreenPlay || app.endlessActiveMode == prev {
				t.Fatalf("%s Random Mix repeated mode %v or failed to advance at round %d", d.ID, prev, i+1)
			}
			prev = app.endlessActiveMode
		}
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

func TestInAppUpdatePrompt(t *testing.T) {
	mt := newMockTerm()
	app := NewApp(mt)
	info := &updater.ReleaseInfo{
		CurrentVersion: "v1.0.0",
		LatestVersion:  "v1.1.0",
		AssetName:      "vimlearn-linux-amd64",
		DownloadURL:    "http://example.invalid/bin",
	}

	// Render shows update prompt with [Y] Yes / [N] No
	app.pendingUpdate = info
	out := app.render()
	if !strings.Contains(out, "Update available: v1.0.0 -> v1.1.0") || !strings.Contains(out, "[Y] Yes") {
		t.Fatalf("expected update prompt on Home screen, got %q", out)
	}

	// Pressing 'n' dismisses prompt
	app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: 'n'})
	if app.pendingUpdate != nil {
		t.Fatalf("expected 'n' to dismiss pendingUpdate")
	}

	// Pressing 'y' runs applyUpdateFn
	called := make(chan struct{}, 1)
	app.pendingUpdate = info
	app.updateResultCh = make(chan error, 1)
	app.applyUpdateFn = func(ctx context.Context, got *updater.ReleaseInfo) error {
		called <- struct{}{}
		return nil
	}
	app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: 'y'})
	<-called
	err := <-app.updateResultCh
	if err != nil {
		t.Fatalf("expected nil update err, got %v", err)
	}
	app.updateBusy = false
	app.updateDone = true
	app.updateSuccess = true
	app.updateStatus = "Updated to v1.1.0! Press any key to exit and relaunch vimlearn."

	if !strings.Contains(app.render(), "Updated to v1.1.0") {
		t.Fatalf("expected updated status banner")
	}
	if quit := app.handleEvent(term.KeyEvent{Kind: term.KeyEnter}); !quit {
		t.Fatalf("expected keypress after successful update to exit cleanly")
	}
}

func TestHelpOverlayAndCommand(t *testing.T) {
	mt := newMockTerm()
	app := NewApp(mt)

	// Press '?' on Home -> opens ScreenHelp -> 'Esc' returns to ScreenHome
	app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: '?'})
	if app.screen != ScreenHelp {
		t.Fatalf("expected ScreenHelp on '?', got %v", app.screen)
	}
	if out := app.render(); !strings.Contains(out, "VIMLEARN HELP") || !strings.Contains(out, "man vimlearn") {
		t.Fatalf("help screen missing expected content")
	}
	app.handleEvent(term.KeyEvent{Kind: term.KeyEsc})
	if app.screen != ScreenHome {
		t.Fatalf("expected return to ScreenHome, got %v", app.screen)
	}

	// Open a level, make an edit, run :help, and verify buffer is preserved on return
	if err := app.OpenLevel(levels.DiffBeginner, 3); err != nil {
		t.Fatal(err)
	}
	app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: 'x'})
	editedText := app.eng.Text()
	for _, ev := range vimKeys(":help<CR>") {
		app.handleEvent(ev)
	}
	if app.screen != ScreenHelp {
		t.Fatalf("expected :help to open ScreenHelp, got %v", app.screen)
	}
	app.handleEvent(term.KeyEvent{Kind: term.KeyRune, Rune: 'q'})
	if app.screen != ScreenPlay || app.eng.Text() != editedText {
		t.Fatalf("expected return to ScreenPlay with buffer preserved")
	}
}
