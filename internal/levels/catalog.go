package levels

import "vimlearn/internal/vim"

// Difficulties returns the 5 difficulty tiers in progression order.
func Difficulties() []DifficultyInfo {
	return []DifficultyInfo{
		{
			ID:          DiffBeginner,
			Name:        "Beginner",
			Tagline:     "Foundations of Motion & Editing",
			Description: "Learn core home-row movement, entering/exiting Insert mode, and deleting characters or lines.",
			KeysSummary: "h j k l  w b  x  i a o  dd  u",
			ColorCode:   46, // bright green
		},
		{
			ID:          DiffIntermediate,
			Name:        "Intermediate",
			Tagline:     "Line Anchors, Word Operators & Repeat",
			Description: "Stop stepping char-by-char. Snap to line bounds, snipe characters with f/t, yank/paste, and repeat with '.'",
			KeysSummary: "0 ^ $  A I  f t ;  dw cw  yy p  .",
			ColorCode:   39, // bright cyan/blue
		},
		{
			ID:          DiffHard,
			Name:        "Hard",
			Tagline:     "Counts, Text Objects & Search",
			Description: "Edit inside quotes and brackets from anywhere, jump with counts, match braces, and hunt with /search.",
			KeysSummary: "3j  ci\" da( vi[  gg G %  J  / n N",
			ColorCode:   220, // gold/yellow
		},
		{
			ID:          DiffExpert,
			Name:        "Expert",
			Tagline:     "Surgical Refactoring",
			Description: "Combine Visual modes, till-motions, block text objects, indenting, and marks for crisp multi-line edits.",
			KeysSummary: "v V ~ r  dt) ct;  ci{ >> <<  ma 'a",
			ColorCode:   208, // orange
		},
		{
			ID:          DiffImpossible,
			Name:        "Impossible",
			Tagline:     "Timed Vim Grandmaster Challenges",
			Description: "Beat strict timers and razor-thin Par limits using macros, named registers, nested text objects, and dot-blitz.",
			KeysSummary: "qa...q @a  \"ay \"ap  ci( da\"  / n .",
			ColorCode:   196, // crimson red
		},
	}
}

// All returns all 25 handcrafted levels across the 5 difficulties.
func All() []Level {
	return []Level{
		// ==================== BEGINNER (5 Levels) ====================
		{
			ID:          "beginner-1",
			Difficulty:  DiffBeginner,
			Number:      1,
			Title:       "Home Row Compass",
			GoalType:    GoalReachCursor,
			StartText:   "start . . . [1]\n. . . . . . . .\n[2] . . . . . .\n. . . . [3] . .",
			StartCursor: vim.Position{Row: 0, Col: 0},
			Checkpoints: []vim.Position{
				{Row: 0, Col: 13},
				{Row: 2, Col: 1},
				{Row: 3, Col: 9},
			},
			Par: 34,
			Tutorial: TutorialCard{
				Objective:   "Visit all 3 numbered checkpoints [1] -> [2] -> [3] in order using h, j, k, l.",
				Explanation: "In Vim Normal mode, your right hand stays on the home row: h (left), j (down), k (up), l (right). Watch the highlighted target checkpoint in the buffer!",
				Keys: []KeyHint{
					{Key: "h / l", Desc: "Move cursor Left / Right by 1 character"},
					{Key: "j / k", Desc: "Move cursor Down / Up by 1 line"},
				},
				Walkthrough: "Press l 13 times to hit [1], then j j and h 12 times for [2], then j and l 8 times for [3].",
			},
			Hint:       "Tip: You can also hold/tap counts like 13l or step with h j k l!",
			OptimalSeq: "13l2j12hj8l",
		},
		{
			ID:          "beginner-2",
			Difficulty:  DiffBeginner,
			Number:      2,
			Title:       "Word Hopper",
			GoalType:    GoalReachCursor,
			StartText:   "let speed = fast + turbo\nrun jump [GOAL] dash",
			StartCursor: vim.Position{Row: 0, Col: 0},
			Checkpoints: []vim.Position{
				{Row: 0, Col: 19},
				{Row: 1, Col: 10},
			},
			Par: 10,
			Tutorial: TutorialCard{
				Objective:   "Jump onto 'turbo' on line 1, then hop to 'GOAL' inside [GOAL] on line 2.",
				Explanation: "Stepping one letter at a time with 'l' is slow. Use 'w' to hop forward word-by-word and 'b' to hop backward!",
				Keys: []KeyHint{
					{Key: "w", Desc: "Jump forward to start of next word"},
					{Key: "b", Desc: "Jump backward to start of previous word"},
					{Key: "j", Desc: "Move down 1 line"},
				},
				Walkthrough: "Press w 5 times to land on 'turbo', then j to drop a line and use w/b to land on 'G' in [GOAL].",
			},
			Hint:       "Notice punctuation like '[' counts as its own word boundary for w/b!",
			OptimalSeq: "5wjbbb",
		},
		{
			ID:          "beginner-3",
			Difficulty:  DiffBeginner,
			Number:      3,
			Title:       "First Typo Surgery",
			GoalType:    GoalMatchBuffer,
			StartText:   "helo wrld\nprinnt(\"ok\")",
			StartCursor: vim.Position{Row: 0, Col: 0},
			TargetText:  "hello world\nprint(\"ok\")",
			Par:         16,
			Tutorial: TutorialCard{
				Objective:   "Fix the 3 typos: 'helo' -> 'hello', 'wrld' -> 'world', and 'prinnt' -> 'print'.",
				Explanation: "Use 'x' in Normal mode to delete a single character under the cursor. Use 'i' (insert before) or 'a' (append after) to type missing letters, then press <Esc> to return to Normal mode.",
				Keys: []KeyHint{
					{Key: "x", Desc: "Delete character under cursor (Normal mode)"},
					{Key: "i / a", Desc: "Enter Insert mode before / after cursor"},
					{Key: "Esc", Desc: "Return from Insert mode to Normal mode"},
				},
				Walkthrough: "Move to 'l' in helo, press al<Esc>, hop to 'w' in wrld, ao<Esc>, drop down to extra 'n' in prinnt and press x.",
			},
			Hint:       "Remember to press <Esc> after inserting letters so j/l/w move instead of typing!",
			OptimalSeq: "wbi<Right><Right>l<Esc>wao<Esc>j^3lx",
		},
		{
			ID:          "beginner-4",
			Difficulty:  DiffBeginner,
			Number:      4,
			Title:       "Line Cleaner",
			GoalType:    GoalMatchBuffer,
			StartText:   "func main() {\n  DEBUG_REMOVE_ME()\n}",
			StartCursor: vim.Position{Row: 0, Col: 0},
			TargetText:  "func main() {\n  runApp()\n}",
			Par:         16,
			Tutorial: TutorialCard{
				Objective:   "Delete the DEBUG_REMOVE_ME() line and insert '  runApp()' in its place.",
				Explanation: "'dd' deletes the entire current line in one swift stroke! Then 'o' opens a fresh line below and puts you straight into Insert mode.",
				Keys: []KeyHint{
					{Key: "dd", Desc: "Delete entire current line"},
					{Key: "o", Desc: "Open new line below cursor and enter Insert mode"},
					{Key: "u", Desc: "Undo last change if you make a mistake"},
				},
				Walkthrough: "Press j to move to line 2, press dd to delete it (cursor moves to '}'), wait—better: press k back to line 1, press o, type '  runApp()', then <Esc>!",
			},
			Hint:       "Delete line 2 with j then dd, go up with k, then open line below with o!",
			OptimalSeq: "jddko  runApp()<Esc>",
		},
		{
			ID:          "beginner-5",
			Difficulty:  DiffBeginner,
			Number:      5,
			Title:       "Config Cleanup (Beginner Boss)",
			GoalType:    GoalMatchBuffer,
			StartText:   "host = localhost\nportt = 8080\ntemp = delete_this\nmode = prod",
			StartCursor: vim.Position{Row: 0, Col: 0},
			TargetText:  "host = localhost\nport = 8080\nmode = prod\n# ready",
			Par:         18,
			Tutorial: TutorialCard{
				Objective:   "Fix 'portt' -> 'port', delete the 'temp = delete_this' line, and add '# ready' at the bottom.",
				Explanation: "Combine everything from Beginner: move with j/w, delete extra char with x, delete a line with dd, and add a bottom line with o.",
				Keys: []KeyHint{
					{Key: "j / w", Desc: "Navigate lines and words"},
					{Key: "x / dd", Desc: "Delete char / Delete line"},
					{Key: "o ... <Esc>", Desc: "Open line below, type, exit Insert"},
				},
				Walkthrough: "j -> 4lx (fix portt) -> j dd (remove temp line) -> o# ready<Esc>.",
			},
			Hint:       "After dd on line 3, 'mode = prod' becomes line 3—press o right there!",
			OptimalSeq: "j4lxjddo# ready<Esc>",
		},

		// ==================== INTERMEDIATE (5 Levels) ====================
		{
			ID:          "intermediate-1",
			Difficulty:  DiffIntermediate,
			Number:      1,
			Title:       "Line Anchors",
			GoalType:    GoalMatchBuffer,
			StartText:   "  alpha = 1\n  beta = 2\n  gamma = 3",
			StartCursor: vim.Position{Row: 0, Col: 5},
			TargetText:  "  let alpha = 1;\n  let beta = 2;\n  let gamma = 3;",
			Par:         29,
			Tutorial: TutorialCard{
				Objective:   "Prepend 'let ' before each variable name and append ';' at the end of each line.",
				Explanation: "'I' jumps to the first non-whitespace character and enters Insert mode. 'A' jumps to the end of the line and enters Insert mode. Much faster than 0/$/i/a!",
				Keys: []KeyHint{
					{Key: "0 / ^ / $", Desc: "Column 0 / First non-blank / End of line"},
					{Key: "I / A", Desc: "Insert at first non-blank / Append at end of line"},
				},
				Walkthrough: "On line 1: Ilet <Esc>A;<Esc>. Move down with j and repeat for lines 2 and 3!",
			},
			Hint:       "Use I for the front (skips indentation!) and A for the end of each line.",
			OptimalSeq: "Ilet <Esc>A;<Esc>jIlet <Esc>A;<Esc>jIlet <Esc>A;<Esc>",
		},
		{
			ID:          "intermediate-2",
			Difficulty:  DiffIntermediate,
			Number:      2,
			Title:       "Character Sniper (f / t)",
			GoalType:    GoalMatchBuffer,
			StartText:   "connect(host, 8080, false)",
			StartCursor: vim.Position{Row: 0, Col: 0},
			TargetText:  "connect(host, 443, true)",
			Par:         21,
			Tutorial: TutorialCard{
				Objective:   "Change port 8080 -> 443 and false -> true.",
				Explanation: "'f<char>' jumps straight onto the next <char> on the line! 't<char>' stops just before <char>. ';' repeats your last f/t jump.",
				Keys: []KeyHint{
					{Key: "f<c>", Desc: "Jump forward ONTO character <c>"},
					{Key: "t<c>", Desc: "Jump forward TILL (just before) <c>"},
					{Key: "; / ,", Desc: "Repeat last f/t forward / backward"},
					{Key: "cw", Desc: "Change word under cursor"},
				},
				Walkthrough: "Press f8 to jump straight onto 8080, cw443<Esc>, then ff to jump to 'false' and cwtrue<Esc>!",
			},
			Hint:       "f8 lands right on '8' in 8080; then cw443<Esc>, then ff lands on 'f' in false!",
			OptimalSeq: "f8cw443<Esc>ffcwtrue<Esc>",
		},
		{
			ID:          "intermediate-3",
			Difficulty:  DiffIntermediate,
			Number:      3,
			Title:       "Word Surgery (dw / cw)",
			GoalType:    GoalMatchBuffer,
			StartText:   "var oldName = broken getData()\nreturn bad result",
			StartCursor: vim.Position{Row: 0, Col: 0},
			TargetText:  "const newName = getData()\nreturn good result",
			Par:         33,
			Tutorial: TutorialCard{
				Objective:   "Change 'var'->'const', 'oldName'->'newName', delete 'broken ', and change 'bad'->'good'.",
				Explanation: "Combine operators with 'w': 'cw' changes a word and enters Insert mode (keeping trailing spaces!), while 'dw' deletes a word AND its trailing space.",
				Keys: []KeyHint{
					{Key: "cw", Desc: "Change word (replaces word, keeps trailing space)"},
					{Key: "dw", Desc: "Delete word (removes word + trailing space)"},
				},
				Walkthrough: "cwconst<Esc> w cwnewName<Esc> 2w dw j w cwgood<Esc>.",
			},
			Hint:       "Use cw when replacing a word and dw when removing a word completely!",
			OptimalSeq: "cwconst<Esc>wcwnewName<Esc>2wdwj^wcwgood<Esc>",
		},
		{
			ID:          "intermediate-4",
			Difficulty:  DiffIntermediate,
			Number:      4,
			Title:       "Clone & Dot-Repeat (yy p .)",
			GoalType:    GoalMatchBuffer,
			StartText:   "item = 0\nitem = 0\nitem = 0",
			StartCursor: vim.Position{Row: 0, Col: 0},
			TargetText:  "item = 0;\nitem = 0;\nitem = 0;\nitem = 0;",
			Par:         10,
			Tutorial: TutorialCard{
				Objective:   "Duplicate one line so there are 4 lines, and append ';' to every line.",
				Explanation: "'yy' yanks (copies) the current line; 'p' pastes it below. Even better: '.' (dot) repeats your last editing command (like A;<Esc>) in ONE keystroke!",
				Keys: []KeyHint{
					{Key: "yy / p", Desc: "Yank current line / Paste below (P = above)"},
					{Key: ". (dot)", Desc: "Repeat your last editing command"},
				},
				Walkthrough: "yyp adds the 4th line. Then A;<Esc> on line 1, and just j. down the remaining lines!",
			},
			Hint:       "Do A;<Esc> once, then use j. to repeat it down each line—and yyp to clone a line!",
			OptimalSeq: "A;<Esc>j.j.yyp",
		},
		{
			ID:          "intermediate-5",
			Difficulty:  DiffIntermediate,
			Number:      5,
			Title:       "Struct Refactor (Intermediate Boss)",
			GoalType:    GoalMatchBuffer,
			StartText:   "type User struct {\n  Name string\n  Temp int\n  Role string\n}",
			StartCursor: vim.Position{Row: 0, Col: 0},
			TargetText:  "type Account struct {\n  Name string\n  Role string\n  Role string\n}",
			Par:         18,
			Tutorial: TutorialCard{
				Objective:   "Rename 'User' -> 'Account', remove 'Temp int', and duplicate 'Role string'.",
				Explanation: "Chain Intermediate skills smoothly: word jumps, cw, dd, and yyp.",
				Keys: []KeyHint{
					{Key: "w / cw", Desc: "Jump to word & change it"},
					{Key: "dd / yyp", Desc: "Delete line / Yank & paste line"},
				},
				Walkthrough: "wcwAccount<Esc> on line 1, 2j dd to remove Temp, then yyp to clone Role!",
			},
			Hint:       "wcwAccount<Esc>2jddyyp!",
			OptimalSeq: "wcwAccount<Esc>2jddyyp",
		},

		// ==================== HARD (5 Levels) ====================
		{
			ID:          "hard-1",
			Difficulty:  DiffHard,
			Number:      1,
			Title:       "Count Multiplier",
			GoalType:    GoalMatchBuffer,
			StartText:   "keep 1\nkeep 2\ndrop A\ndrop B\ndrop C\nkeep 3",
			StartCursor: vim.Position{Row: 0, Col: 0},
			TargetText:  "keep 1\nkeep 2\nkeep 3\nkeep 3",
			Par:         8,
			Tutorial: TutorialCard{
				Objective:   "Delete all 3 'drop' lines at once and duplicate 'keep 3'.",
				Explanation: "Almost every Vim motion and operator accepts a numeric count prefix! '2j' moves down 2 lines; '3dd' deletes 3 lines in two tokens.",
				Keys: []KeyHint{
					{Key: "<N>j / <N>k", Desc: "Move N lines down / up"},
					{Key: "<N>dd", Desc: "Delete N lines starting from cursor"},
					{Key: "yyp", Desc: "Yank & paste line"},
				},
				Walkthrough: "2j jumps straight to 'drop A', 3dd deletes all 3 drop lines, then yyp duplicates 'keep 3'!",
			},
			Hint:       "2j then 3dd then yyp!",
			OptimalSeq: "2j3ddyyp",
		},
		{
			ID:          "hard-2",
			Difficulty:  DiffHard,
			Number:      2,
			Title:       "Inside the Quotes (ci\" / da\")",
			GoalType:    GoalMatchBuffer,
			StartText:   "const title = \"Draft Title\"\nconst tag = \"remove-me\" + \"v2\"",
			StartCursor: vim.Position{Row: 0, Col: 0},
			TargetText:  "const title = \"Final Release\"\nconst tag =  + \"v2\"",
			Par:         23,
			Tutorial: TutorialCard{
				Objective:   "Change \"Draft Title\" to \"Final Release\", and completely delete \"remove-me\" (including its quotes).",
				Explanation: "Text objects let you act on quoted strings from anywhere on the line! 'ci\"' changes INSIDE quotes; 'da\"' deletes AROUND quotes (including the quote marks).",
				Keys: []KeyHint{
					{Key: "ci\"", Desc: "Change inside double quotes"},
					{Key: "da\"", Desc: "Delete around double quotes (includes quotes)"},
					{Key: "ci' / di'", Desc: "Same for single quotes"},
				},
				Walkthrough: "On line 1, press ci\"Final Release<Esc>. On line 2, press j0 to go to column 0, then da\".",
			},
			Hint:       "ci\"Final Release<Esc>j0da\"",
			OptimalSeq: "ci\"Final Release<Esc>j0da\"",
		},
		{
			ID:          "hard-3",
			Difficulty:  DiffHard,
			Number:      3,
			Title:       "Bracket Text Objects",
			GoalType:    GoalMatchBuffer,
			StartText:   "sum(10, 20, 30, 40)\narr = [1, 2, 3]",
			StartCursor: vim.Position{Row: 0, Col: 6},
			TargetText:  "sum(x, y)\narr = []",
			Par:         12,
			Tutorial: TutorialCard{
				Objective:   "Replace the args inside sum(...) with 'x, y', and empty the array [...].",
				Explanation: "Bracket text objects 'i(' and 'i[' select everything strictly inside matching parentheses or square brackets.",
				Keys: []KeyHint{
					{Key: "ci(", Desc: "Change inside (...)"},
					{Key: "di[", Desc: "Delete inside [...]"},
					{Key: "da{ / ci{", Desc: "Delete around / Change inside {...}"},
				},
				Walkthrough: "ci(x, y<Esc> replaces everything inside sum(...), then jdi[ clears inside [...]!",
			},
			Hint:       "ci(x, y<Esc>jdi[",
			OptimalSeq: "ci(x, y<Esc>jdi[",
		},
		{
			ID:          "hard-4",
			Difficulty:  DiffHard,
			Number:      4,
			Title:       "Brace Jump & Join (gg G % J)",
			GoalType:    GoalMatchBuffer,
			StartText:   "func calc() {\n  return\n    42\n}\n// footer",
			StartCursor: vim.Position{Row: 4, Col: 0},
			TargetText:  "func calc() {\n  return 42\n}",
			Par:         8,
			Tutorial: TutorialCard{
				Objective:   "Delete '// footer', jump to the top, and join 'return' and '42' onto one line.",
				Explanation: "'gg' jumps to line 1, 'G' jumps to the bottom, '%' jumps between matching () [] {}, and 'J' joins the next line onto the current line with one space!",
				Keys: []KeyHint{
					{Key: "gg / G", Desc: "Jump to top / bottom of buffer"},
					{Key: "%", Desc: "Jump to matching bracket"},
					{Key: "J", Desc: "Join current line with next line"},
				},
				Walkthrough: "dd deletes '// footer', gg jumps to line 1, j moves to 'return', and J joins '42' onto it!",
			},
			Hint:       "dd gg j J (only 5 keystrokes!)",
			OptimalSeq: "ddggjJ",
		},
		{
			ID:          "hard-5",
			Difficulty:  DiffHard,
			Number:      5,
			Title:       "Search & Destroy (/ n N)",
			GoalType:    GoalMatchBuffer,
			StartText:   "step1()\nstep2()\nBUG()\nstep3()\nstep4()\nBUG()\nstep5()",
			StartCursor: vim.Position{Row: 0, Col: 0},
			TargetText:  "step1()\nstep2()\nFIX()\nstep3()\nstep4()\nFIX()\nstep5()",
			Par:         14,
			Tutorial: TutorialCard{
				Objective:   "Use /BUG<Enter> to jump straight to each BUG() call and change BUG -> FIX.",
				Explanation: "Press '/' followed by text and <CR> to search forward. Press 'n' to jump to the next match and '.' to repeat your edit!",
				Keys: []KeyHint{
					{Key: "/text<CR>", Desc: "Search forward for 'text'"},
					{Key: "n / N", Desc: "Next / Previous search match"},
					{Key: "ciw / .", Desc: "Change inner word / Repeat last edit"},
				},
				Walkthrough: "/BUG<CR> lands on the first BUG, ciwFIX<Esc> changes it, n jumps to the second BUG, and . repeats!",
			},
			Hint:       "/BUG<CR> ciwFIX<Esc> n .",
			OptimalSeq: "/BUG<CR>ciwFIX<Esc>n.",
		},

		// ==================== EXPERT (5 Levels) ====================
		{
			ID:          "expert-1",
			Difficulty:  DiffExpert,
			Number:      1,
			Title:       "Visual Alchemy (v V ~ r)",
			GoalType:    GoalMatchBuffer,
			StartText:   "status = ok_ok\nlegacy = DELETE\nother = fine",
			StartCursor: vim.Position{Row: 0, Col: 9},
			TargetText:  "status = OK-OK\nother = fine",
			Par:         12,
			Tutorial: TutorialCard{
				Objective:   "Flip 'ok_ok' to uppercase 'OK_OK', replace '_' with '-', and delete the 'legacy' line.",
				Explanation: "In Visual mode ('v'), motions extend a live selection! Press '~' to toggle case of the entire selection, 'r<c>' to replace chars, or 'V' for whole-line Visual mode.",
				Keys: []KeyHint{
					{Key: "v / V", Desc: "Visual char / Visual line selection"},
					{Key: "~", Desc: "Toggle UPPER/lower case"},
					{Key: "r<c>", Desc: "Replace char under cursor with <c>"},
				},
				Walkthrough: "v$~ turns 'ok_ok' into 'OK_OK', f_r- swaps '_' for '-', and jdd deletes the legacy line!",
			},
			Hint:       "v$~ toggles case to end of line; f_r- fixes the underscore; jdd deletes line 2.",
			OptimalSeq: "v$~f_r-jdd",
		},
		{
			ID:          "expert-2",
			Difficulty:  DiffExpert,
			Number:      2,
			Title:       "Till the Delimiter (dt / ct)",
			GoalType:    GoalMatchBuffer,
			StartText:   "send(old_a, old_b, old_c);\nport: 9999;",
			StartCursor: vim.Position{Row: 0, Col: 5},
			TargetText:  "send(payload);\nport: 80;",
			Par:         21,
			Tutorial: TutorialCard{
				Objective:   "Replace all args inside send(...) using ct) and change 9999 -> 80 using ct;.",
				Explanation: "Combining 'c' or 'd' with 't<char>' deletes/changes everything UP TO (but keeping!) <char>.",
				Keys: []KeyHint{
					{Key: "ct<c>", Desc: "Change till character <c> (keeps <c>)"},
					{Key: "dt<c>", Desc: "Delete till character <c> (keeps <c>)"},
					{Key: "df<c>", Desc: "Delete through character <c> (removes <c>)"},
				},
				Walkthrough: "ct)payload<Esc> replaces 'old_a, old_b, old_c' up to ')'. Then j0f9ct;80<Esc>!",
			},
			Hint:       "ct)payload<Esc>j0f9ct;80<Esc>",
			OptimalSeq: "ct)payload<Esc>j0f9ct;80<Esc>",
		},
		{
			ID:          "expert-3",
			Difficulty:  DiffExpert,
			Number:      3,
			Title:       "Block Indent & Body Swap",
			GoalType:    GoalMatchBuffer,
			StartText:   "if ready {\nreturn false\nlog()\n}",
			StartCursor: vim.Position{Row: 1, Col: 0},
			TargetText:  "if ready {\n  return true\n  log()\n}",
			Par:         12,
			Tutorial: TutorialCard{
				Objective:   "Indent lines 2 and 3 by two spaces, and change 'false' -> 'true'.",
				Explanation: "Select multiple lines with 'Vj' and press '>' to indent them together! Or use '2>>' from Normal mode.",
				Keys: []KeyHint{
					{Key: ">> / <<", Desc: "Indent / Dedent current line"},
					{Key: "V ... >", Desc: "Visual-line select and indent block"},
				},
				Walkthrough: "Vj> indents both inner lines, then $bcwtrue<Esc> (or ffcwtrue<Esc>) changes false -> true!",
			},
			Hint:       "Vj> indents lines 2-3; then ffcwtrue<Esc>!",
			OptimalSeq: "Vj>ffcwtrue<Esc>",
		},
		{
			ID:          "expert-4",
			Difficulty:  DiffExpert,
			Number:      4,
			Title:       "Marks & Teleport (ma / 'a)",
			GoalType:    GoalMatchBuffer,
			StartText:   "// anchor: EDIT_HERE\nline 2\nline 3\nline 4\nline 5\n// tail: REMOVE",
			StartCursor: vim.Position{Row: 0, Col: 11},
			TargetText:  "// anchor: DONE\nline 2\nline 3\nline 4\nline 5",
			Par:         15,
			Tutorial: TutorialCard{
				Objective:   "Mark line 1 with 'ma', jump to bottom with G to delete '// tail: REMOVE', then teleport back with 'a and change EDIT_HERE -> DONE.",
				Explanation: "Marks ('m' + letter) let you bookmark any spot in a file and snap straight back with ' + letter (line) or ` + letter (exact col).",
				Keys: []KeyHint{
					{Key: "m<a>", Desc: "Set bookmark 'a' at cursor"},
					{Key: "'<a> / `<a>", Desc: "Jump to line / exact pos of mark 'a'"},
				},
				Walkthrough: "ma G dd `a cwDONE<Esc>!",
			},
			Hint:       "Set mark with ma, jump down with G, dd, teleport with `a, then cwDONE<Esc>!",
			OptimalSeq: "maGdd`acwDONE<Esc>",
		},
		{
			ID:          "expert-5",
			Difficulty:  DiffExpert,
			Number:      5,
			Title:       "Surgical Refactor (Expert Boss)",
			GoalType:    GoalMatchBuffer,
			StartText:   "func greet(a, b) {\n  old()\n  old()\n}\n// end",
			StartCursor: vim.Position{Row: 0, Col: 0},
			TargetText:  "func greet(name) { return \"hi\" }",
			Par:         32,
			Tutorial: TutorialCard{
				Objective:   "Collapse the multi-line function into: func greet(name) { return \"hi\" }",
				Explanation: "Combine text objects (ci(, ci{), line deletion, and line joining (J) for surgical refactoring.",
				Keys: []KeyHint{
					{Key: "ci( / ci{", Desc: "Change inside (...) / {...}"},
					{Key: "J / dd", Desc: "Join lines / Delete line"},
				},
				Walkthrough: "Gdd deletes '// end', gg jumps top, ci(name<Esc> fixes params, 3J collapses the block onto 1 line, then ci{ return \"hi\" <Esc> replaces body!",
			},
			Hint:       "Gdd gg ci(name<Esc> 3J ci{ return \"hi\" <Esc>",
			OptimalSeq: "Gddggci(name<Esc>3Jci{ return \"hi\" <Esc>",
		},

		// ==================== IMPOSSIBLE (5 Timed Levels) ====================
		{
			ID:           "impossible-1",
			Difficulty:   DiffImpossible,
			Number:       1,
			Title:        "Macro Storm (qa ... q)",
			GoalType:     GoalMatchBuffer,
			TimeLimitSec: 60,
			StartText:    "alpha\nbeta\ngamma\ndelta\nepsilon",
			StartCursor:  vim.Position{Row: 0, Col: 0},
			TargetText:   "- \"alpha\",\n- \"beta\",\n- \"gamma\",\n- \"delta\",\n- \"epsilon\",",
			Par:          17,
			Tutorial: TutorialCard{
				Objective:   "Transform all 5 lines into '- \"<word>\",' within 60s and Par 17 using a macro!",
				Explanation: "Press 'qa' to start recording macro 'a', transform line 1 and move to line 2 with j0, press 'q' to stop, then run '3@a' (or 4@a) to replay! Macro playback only counts the @a keystrokes!",
				Keys: []KeyHint{
					{Key: "qa ... q", Desc: "Record macro into register 'a'"},
					{Key: "@a / 3@a", Desc: "Execute macro 'a' once / 3 times"},
				},
				Walkthrough: "qaI- \"<Esc>A\",<Esc>j0q transforms line 1 and moves to line 2; then 4@a finishes all remaining lines!",
			},
			Hint:       "Record: qa I- \"<Esc> A\",<Esc> j0 q   Then replay: 4@a",
			OptimalSeq: "qaI- \"<Esc>A\",<Esc>j0q4@a",
		},
		{
			ID:           "impossible-2",
			Difficulty:   DiffImpossible,
			Number:       2,
			Title:        "Register Juggler (\"a)",
			GoalType:     GoalMatchBuffer,
			TimeLimitSec: 60,
			StartText:    "first: A\nsecond: B\nthird: C",
			StartCursor:  vim.Position{Row: 0, Col: 0},
			TargetText:   "third: C\nsecond: B\nfirst: A",
			Par:          15,
			Tutorial: TutorialCard{
				Objective:   "Reverse the 3 lines in 60s without losing your yanked text to a later delete!",
				Explanation: "Deleting or yanking normally overwrites the unnamed register \"\". Prefix with '\"a' (or '\"b') to store into dedicated named registers a-z!",
				Keys: []KeyHint{
					{Key: "\"add", Desc: "Delete line into register 'a'"},
					{Key: "\"ap / \"aP", Desc: "Paste from register 'a' below / above"},
				},
				Walkthrough: "\"add deletes 'first: A' into reg a; dd deletes 'second: B' into unnamed reg; p pastes 'second: B' below 'third: C'; \"ap pastes 'first: A' at the bottom!",
			},
			Hint:       "\"add dd p \"ap  (Only 10 keystrokes!)",
			OptimalSeq: "\"addddp\"ap",
		},
		{
			ID:           "impossible-3",
			Difficulty:   DiffImpossible,
			Number:       3,
			Title:        "Nested Surgery",
			GoalType:     GoalMatchBuffer,
			TimeLimitSec: 55,
			StartText:    "call(wrap(\"stale\", [1, 2, 3]), \"drop\")",
			StartCursor:  vim.Position{Row: 0, Col: 0},
			TargetText:   "call(wrap(\"fresh\", []), \"keep\")",
			Par:          25,
			Tutorial: TutorialCard{
				Objective:   "Update \"stale\" -> \"fresh\", empty [1, 2, 3] -> [], and \"drop\" -> \"keep\" in one sweep!",
				Explanation: "Jump cleanly between nested delimiters with f + text objects (ci\", di[, f\" + ci\").",
				Keys: []KeyHint{
					{Key: "ci\"", Desc: "Change inside next quotes"},
					{Key: "di[", Desc: "Delete inside [...]"},
					{Key: "f\"", Desc: "Jump forward to quote"},
				},
				Walkthrough: "ci\"fresh<Esc> clears stale; f[di[ empties array; f\"ci\"keep<Esc> updates drop!",
			},
			Hint:       "ci\"fresh<Esc>f[di[2f\"ci\"keep<Esc>",
			OptimalSeq: "ci\"fresh<Esc>f[di[2f\"ci\"keep<Esc>",
		},
		{
			ID:           "impossible-4",
			Difficulty:   DiffImpossible,
			Number:       4,
			Title:        "Search & Dot Blitz",
			GoalType:     GoalMatchBuffer,
			TimeLimitSec: 50,
			StartText:    "let foo = 1\nlet x = foo + 2\nreturn foo * foo",
			StartCursor:  vim.Position{Row: 0, Col: 0},
			TargetText:   "let bar = 1\nlet x = bar + 2\nreturn bar * bar",
			Par:          19,
			Tutorial: TutorialCard{
				Objective:   "Rename all 4 occurrences of 'foo' to 'bar' in under 50s and Par 19.",
				Explanation: "Search once with /foo<CR>, replace with ciwbar<Esc>, then hammer n. n. n. at lightning speed!",
				Keys: []KeyHint{
					{Key: "/foo<CR>", Desc: "Find 'foo'"},
					{Key: "ciwbar<Esc>", Desc: "Change word to 'bar'"},
					{Key: "n .", Desc: "Next match & repeat!"},
				},
				Walkthrough: "/foo<CR> ciwbar<Esc> n. n. n.",
			},
			Hint:       "/foo<CR>ciwbar<Esc>n.n.n.",
			OptimalSeq: "/foo<CR>ciwbar<Esc>n.n.n.",
		},
		{
			ID:           "impossible-5",
			Difficulty:   DiffImpossible,
			Number:       5,
			Title:        "Grandmaster Finale",
			GoalType:     GoalMatchBuffer,
			TimeLimitSec: 60,
			StartText:    "fn(a, b)\nfn(c, d)\nfn(e, f)\n// junk",
			StartCursor:  vim.Position{Row: 0, Col: 0},
			TargetText:   "fn(ok);\nfn(ok);\nfn(ok);",
			Par:          22,
			Tutorial: TutorialCard{
				Objective:   "Replace every fn(...) arg list with (ok); and delete '// junk' in 60s!",
				Explanation: "Final exam: combine macro recording (qa ... q), text objects (ci(), line append (A;), count replay (2@a), and Gdd.",
				Keys: []KeyHint{
					{Key: "qa ... q / 2@a", Desc: "Record & replay macro twice"},
					{Key: "ci( / A;", Desc: "Change inside (...) & append ';'"},
					{Key: "G dd", Desc: "Jump to bottom & delete line"},
				},
				Walkthrough: "Gdd removes '// junk', gg goes top, qa ci(ok<Esc>A;<Esc>j0q records line transform, then 2@a finishes!",
			},
			Hint:       "Gddggqaci(ok<Esc>A;<Esc>j0q2@a",
			OptimalSeq: "Gddggqaci(ok<Esc>A;<Esc>j0q2@a",
		},
	}
}

// ByDifficulty returns all levels belonging to the requested difficulty.
func ByDifficulty(id DifficultyID) []Level {
	var out []Level
	for _, lv := range All() {
		if lv.Difficulty == id {
			out = append(out, lv)
		}
	}
	return out
}

// Find looks up a level by difficulty and 1-based number (or by ID).
func Find(diff DifficultyID, num int) (Level, bool) {
	for _, lv := range All() {
		if lv.Difficulty == diff && lv.Number == num {
			return lv, true
		}
	}
	return Level{}, false
}

// DifficultyByID returns metadata for a DifficultyID.
func DifficultyByID(id DifficultyID) (DifficultyInfo, bool) {
	for _, d := range Difficulties() {
		if d.ID == id {
			return d, true
		}
	}
	return DifficultyInfo{}, false
}
