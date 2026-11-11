package levels

// Tutorials returns the step-by-step Vim & Game Primer lessons shown in the Tutorial Hub.
func Tutorials() []TutorialLesson {
	return []TutorialLesson{
		{
			ID:      "modes",
			Title:   "1. Modal Editing & How This Game Works",
			Summary: "Understand Vim's modes (Normal, Insert, Visual, Command) and how to play.",
			Body: []string{
				"Unlike regular editors where typing always inserts text, Vim is MODAL.",
				"You spend most of your time in NORMAL mode, where keys are commands (move, delete, yank).",
				"Pressing i/a/o enters INSERT mode to type text. Press <Esc> to return to NORMAL mode.",
				"",
				"How vimlearn works:",
				"  - Choose a Difficulty (Beginner -> Impossible) and pick ANY Level inside it.",
				"  - Each level shows your LIVE BUFFER on the left and the TARGET on the right.",
				"  - Read the Level Briefing panel at the top/bottom to see what needs to be done.",
				"  - Beat the Par keystroke count to earn 3 Stars (★★★)!",
				"  - Notation tip: <Esc> = Escape, <CR> = Enter, <C-r> = Ctrl+R.",
			},
			Keys: []KeyHint{
				{Key: "Esc", Desc: "Return to NORMAL mode from anywhere (your home base)"},
				{Key: "i / a", Desc: "Enter INSERT mode before (i) or after (a) the cursor"},
				{Key: "v / V", Desc: "Enter VISUAL (char) or VISUAL LINE selection mode"},
				{Key: ":", Desc: "Enter COMMAND mode (:reset to restart, :q to leave level)"},
			},
			Example: "Workflow: Move in NORMAL mode -> press i -> type text -> press <Esc> -> move again!",
		},
		{
			ID:      "motions",
			Title:   "2. Fast Navigation (Stop Mashing Arrow Keys)",
			Summary: "Jump by characters, words, lines, brackets, and exact characters.",
			Body: []string{
				"In NORMAL mode, your fingers never leave the home row:",
				"  - h j k l move Left, Down, Up, Right.",
				"  - w jumps to the start of the next word; b jumps back; e jumps to word end.",
				"  - 0 goes to column 0; ^ jumps to first non-space; $ jumps to end of line.",
				"  - f<char> snipes forward to <char> on the current line; t<char> stops just before it.",
				"  - gg jumps to the first line; G jumps to the last line; % jumps to matching () [] {}.",
			},
			Keys: []KeyHint{
				{Key: "h / j / k / l", Desc: "Left / Down / Up / Right"},
				{Key: "w / b / e", Desc: "Next word / Previous word / End of word"},
				{Key: "0 / ^ / $", Desc: "Line start / First non-blank / Line end"},
				{Key: "f<c> / t<c> / ;", Desc: "Find <c> / Till <c> / Repeat last find"},
				{Key: "gg / G / %", Desc: "Top of file / Bottom of file / Matching bracket"},
			},
			Example: "Example: Instead of pressing 'l' 20 times to reach '(', just press f( !",
		},
		{
			ID:      "grammar",
			Title:   "3. The Vim Grammar: Operator + Count + Motion",
			Summary: "Combine verbs (d, c, y) with nouns (motions & text objects).",
			Body: []string{
				"Vim commands form sentences: [Operator] + [Count] + [Motion or Text Object].",
				"Once you learn 3 operators and a handful of motions, you know dozens of commands!",
				"",
				"Operators (Verbs):",
				"  d = delete    c = change (delete + enter Insert)    y = yank (copy)",
				"",
				"Text Objects (Nouns that work from ANYWHERE inside them!):",
				"  iw / aw = inner word / a word      i\" / a\" = inside / around double quotes",
				"  i( / a( = inside / around (...)    i{ / a{ = inside / around {...}",
			},
			Keys: []KeyHint{
				{Key: "dw / cw", Desc: "Delete word / Change word"},
				{Key: "dd / yy / p", Desc: "Delete line / Yank line / Paste below (P = above)"},
				{Key: "ci\" / da(", Desc: "Change inside \"...\" / Delete around (...)"},
				{Key: "3j / 2dd", Desc: "Count prefix: move 3 lines down / delete 2 lines"},
			},
			Example: "Example: Cursor anywhere inside \"hello world\" -> ci\"bye<Esc> -> \"bye\" !",
		},
		{
			ID:      "power",
			Title:   "4. Power Tools: Dot Repeat, Search, Marks & Macros",
			Summary: "Multiply your speed in Hard, Expert, and Impossible levels.",
			Body: []string{
				"Great Vim users never do the same edit twice manually:",
				"  - . (dot) repeats your last change in a single keystroke!",
				"  - /word<Enter> searches forward; n jumps to next match; N jumps to previous.",
				"  - ma sets mark 'a' at cursor; 'a teleports back to it from anywhere in the file.",
				"  - qa ... q records a macro into register 'a'; @a replays it (5@a replays 5x!).",
				"  - \"ayy yanks into named register 'a' so another delete won't overwrite it.",
			},
			Keys: []KeyHint{
				{Key: ". (dot)", Desc: "Repeat last editing command"},
				{Key: "/text + n / N", Desc: "Search forward + Next / Previous match"},
				{Key: "ma / 'a", Desc: "Set mark 'a' / Jump back to mark 'a'"},
				{Key: "qa ... q / @a", Desc: "Record macro 'a' / Play macro 'a'"},
				{Key: "\"ay / \"ap", Desc: "Yank to / Paste from named register 'a'"},
			},
			Example: "Example: Fix 5 lines at once with qaA;<Esc>j0q then 4@a !",
		},
	}
}
