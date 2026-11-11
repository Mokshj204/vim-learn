package vim

import "testing"

func TestBasicMovementAndInsert(t *testing.T) {
	e := NewEngine("hello world\nsecond line", Position{Row: 0, Col: 0})
	e.Feed("w")
	if e.Cursor != (Position{Row: 0, Col: 6}) {
		t.Fatalf("after w, got %+v", e.Cursor)
	}
	e.Feed("b")
	if e.Cursor != (Position{Row: 0, Col: 0}) {
		t.Fatalf("after b, got %+v", e.Cursor)
	}
	e.Feed("e")
	if e.Cursor != (Position{Row: 0, Col: 4}) {
		t.Fatalf("after e, got %+v", e.Cursor)
	}
	e.Feed("a!<Esc>")
	if got := e.Text(); got != "hello! world\nsecond line" {
		t.Fatalf("after a!<Esc>, got %q", got)
	}
}

func TestOperatorsAndTextObjects(t *testing.T) {
	e := NewEngine(`const msg = "old value";`, Position{Row: 0, Col: 0})
	e.Feed(`ci"new text<Esc>`)
	if got := e.Text(); got != `const msg = "new text";` {
		t.Fatalf(`ci" failed: %q`, got)
	}

	e2 := NewEngine("alpha beta gamma", Position{Row: 0, Col: 7})
	e2.Feed("ciwOMEGA<Esc>")
	if got := e2.Text(); got != "alpha OMEGA gamma" {
		t.Fatalf("ciw failed: %q", got)
	}

	e3 := NewEngine("fn(a, b, c) + 1", Position{Row: 0, Col: 5})
	e3.Feed("di(")
	if got := e3.Text(); got != "fn() + 1" {
		t.Fatalf("di( failed: %q", got)
	}

	e4 := NewEngine("foo bar baz", Position{Row: 0, Col: 0})
	e4.Feed("cwqux<Esc>")
	if got := e4.Text(); got != "qux bar baz" {
		t.Fatalf("cw failed (should preserve space): %q", got)
	}
}

func TestUndoRedoAndDotRepeat(t *testing.T) {
	e := NewEngine("one\ntwo\nthree", Position{Row: 0, Col: 0})
	e.Feed("dd")
	if got := e.Text(); got != "two\nthree" {
		t.Fatalf("dd failed: %q", got)
	}
	e.Feed(".")
	if got := e.Text(); got != "three" {
		t.Fatalf("dot repeat dd failed: %q", got)
	}
	e.Feed("u")
	if got := e.Text(); got != "two\nthree" {
		t.Fatalf("undo failed: %q", got)
	}
	e.Feed("<C-r>")
	if got := e.Text(); got != "three" {
		t.Fatalf("redo failed: %q", got)
	}
}

func TestVisualFindAndSearch(t *testing.T) {
	e := NewEngine("let x = foo(1, 2) + bar;", Position{Row: 0, Col: 0})
	e.Feed("f(ldt)i3<Esc>")
	if got := e.Text(); got != "let x = foo(3) + bar;" {
		t.Fatalf("find/dt) failed: %q", got)
	}

	e2 := NewEngine("line1\nline2 target\nline3 target", Position{Row: 0, Col: 0})
	e2.Feed("/target<CR>n")
	if e2.Cursor != (Position{Row: 2, Col: 6}) {
		t.Fatalf("search / + n failed: %+v", e2.Cursor)
	}

	e3 := NewEngine("abc def ghi", Position{Row: 0, Col: 4})
	e3.Feed("ve~")
	if got := e3.Text(); got != "abc DEF ghi" {
		t.Fatalf("visual ve~ failed: %q", got)
	}
}

func TestMacrosRegistersAndJoin(t *testing.T) {
	e := NewEngine("a = 1\nb = 2\nc = 3", Position{Row: 0, Col: 0})
	e.Feed("qaA;<Esc>j0q2@a")
	if got := e.Text(); got != "a = 1;\nb = 2;\nc = 3;" {
		t.Fatalf("macro failed: %q", got)
	}

	e2 := NewEngine("first\n  second", Position{Row: 0, Col: 0})
	e2.Feed("J")
	if got := e2.Text(); got != "first second" {
		t.Fatalf("J failed: %q", got)
	}

	e3 := NewEngine("keep\nswap1\nswap2", Position{Row: 1, Col: 0})
	e3.Feed(`"addP`)
	if got := e3.Text(); got != "keep\nswap1\nswap2" {
		// dd on swap1 put cursor on swap2; P pasted swap1 above swap2 -> swap1, swap2
		t.Fatalf("named register paste failed: %q", got)
	}
}
