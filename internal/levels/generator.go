package levels

import (
	"fmt"
	"math/rand"
	"strings"

	"vimlearn/internal/vim"
)

// GenerateEndless creates a fresh procedural puzzle (defaults to Multi-Edit Mix).
func GenerateEndless(diff DifficultyID, round int, rng *rand.Rand) Level {
	return GenerateEndlessForMode(diff, EndlessMultiEdit, round, rng)
}

// GenerateEndlessForMode creates a procedural puzzle for the specified Endless sub-mode.
func GenerateEndlessForMode(diff DifficultyID, mode EndlessMode, round int, rng *rand.Rand) Level {
	switch mode {
	case EndlessCodeHunt:
		return genCodeHunt(diff, round, rng)
	case EndlessMultiEdit:
		return genMultiEdit(diff, round, rng)
	default:
		return genMultiEdit(diff, round, rng)
	}
}

func keyCountForSeq(seq string) int {
	return len(vim.ParseKeySequence(seq))
}

type snippetTheme struct {
	name  string
	lines []string
}

func snippetThemes() []snippetTheme {
	return []snippetTheme{
		{
			name: "Go Handler",
			lines: []string{
				"func handleHealth(w Response, r *Request) {",
				"  status := \"ok\"",
				"  latency := 12",
				"  region := \"us-east\"",
				"  limit := 250",
				"  logMetric(region, latency)",
				"  writeJSON(w, status, limit)",
				"}",
			},
		},
		{
			name: "JSON Config",
			lines: []string{
				"{",
				"  \"service\": \"auth-api\",",
				"  \"port\": 8080,",
				"  \"workers\": 4,",
				"  \"cache\": \"redis\",",
				"  \"retry\": 3,",
				"  \"tls\": true",
				"}",
			},
		},
		{
			name: "Shell Deploy",
			lines: []string{
				"#!/usr/bin/env sh",
				"APP_NAME=\"vimlearn\"",
				"TARGET_ENV=\"staging\"",
				"BUILD_DIR=\"./dist\"",
				"echo \"building $APP_NAME\"",
				"go test ./...",
				"cp \"$BUILD_DIR/app\" /usr/local/bin/",
			},
		},
		{
			name: "Python Worker",
			lines: []string{
				"def process_batch(items, timeout=30):",
				"    total = len(items)",
				"    batch_size = 64",
				"    queue = \"default\"",
				"    retries = 2",
				"    emit_count(queue, total)",
				"    return total",
			},
		},
		{
			name: "Env Settings",
			lines: []string{
				"APP_HOST=127.0.0.1",
				"APP_PORT=3000",
				"LOG_LEVEL=info",
				"DB_POOL=16",
				"CACHE_TTL=120",
				"RATE_LIMIT=500",
			},
		},
		{
			name: "JS Router",
			lines: []string{
				"const router = createRouter({",
				"  basePath: \"/api/v1\",",
				"  timeoutMs: 1500,",
				"  maxRetries: 3,",
				"  enableCors: true,",
				"  format: \"json\"",
				"})",
			},
		},
	}
}

type lineMutation struct {
	startLines    []string
	targetLines   []string
	solveFromCol0 string
	objective     string
	keys          []KeyHint
}

func genMultiEdit(diff DifficultyID, round int, rng *rand.Rand) Level {
	// Impossible also rotates in global macro/register/search blitz challenges.
	if diff == DiffImpossible && rng.Intn(2) == 0 {
		return genImpossibleGlobal(round, rng)
	}
	// Hard occasionally runs a full-file search blitz.
	if diff == DiffHard && rng.Intn(3) == 0 {
		return genHardSearchBlitz(round, rng)
	}

	// Escalate mutation count with streak
	mutCount := 2
	if round <= 2 {
		mutCount = 1 + rng.Intn(2)
	} else if round >= 7 {
		mutCount = 2 + rng.Intn(2)
	}

	pool := mutationPoolForDifficulty(diff, rng)
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if mutCount > len(pool) {
		mutCount = len(pool)
	}
	chosen := pool[:mutCount]

	themes := snippetThemes()
	theme := themes[rng.Intn(len(themes))]

	var startLines []string
	var targetLines []string
	type placedMut struct {
		mut      lineMutation
		startRow int
	}
	var placed []placedMut

	ctxIdx := 0
	addCtx := func() {
		if ctxIdx < len(theme.lines) {
			l := theme.lines[ctxIdx]
			startLines = append(startLines, l)
			targetLines = append(targetLines, l)
			ctxIdx++
		}
	}

	addCtx()
	for _, m := range chosen {
		placed = append(placed, placedMut{mut: m, startRow: len(startLines)})
		startLines = append(startLines, m.startLines...)
		targetLines = append(targetLines, m.targetLines...)
		addCtx()
	}
	if len(startLines) < 4 {
		addCtx()
	}

	startText := strings.Join(startLines, "\n")
	targetText := strings.Join(targetLines, "\n")

	// Randomize starting cursor row/col (avoiding trivially always starting at 0,0)
	startRow := rng.Intn(len(startLines))
	startCol := 0
	if len(startLines[startRow]) > 1 {
		startCol = rng.Intn(len(startLines[startRow]))
	}
	startCursor := vim.Position{Row: startRow, Col: startCol}

	// Synthesize exact optimal sequence by driving a scratch vim.Engine
	sim := vim.NewEngine(startText, startCursor)
	var opt strings.Builder
	rowDelta := 0
	for _, pm := range placed {
		targetRow := pm.startRow + rowDelta
		if targetRow > sim.Cursor.Row {
			d := targetRow - sim.Cursor.Row
			var mv string
			if d == 1 {
				mv = "j"
			} else {
				mv = fmt.Sprintf("%dj", d)
			}
			opt.WriteString(mv)
			sim.Feed(mv)
		} else if targetRow < sim.Cursor.Row {
			d := sim.Cursor.Row - targetRow
			var mv string
			if d == 1 {
				mv = "k"
			} else {
				mv = fmt.Sprintf("%dk", d)
			}
			opt.WriteString(mv)
			sim.Feed(mv)
		}
		if sim.Cursor.Col != 0 {
			opt.WriteString("0")
			sim.Feed("0")
		}
		opt.WriteString(pm.mut.solveFromCol0)
		sim.Feed(pm.mut.solveFromCol0)
		rowDelta += len(pm.mut.targetLines) - len(pm.mut.startLines)
	}

	optSeq := opt.String()
	par := sim.KeyCount
	if diff == DiffBeginner {
		par += 3
	} else if diff == DiffIntermediate {
		par += 2
	}

	var objParts []string
	var hints []KeyHint
	seenKeys := map[string]bool{}
	for _, pm := range placed {
		objParts = append(objParts, pm.mut.objective)
		for _, kh := range pm.mut.keys {
			if !seenKeys[kh.Key] {
				seenKeys[kh.Key] = true
				hints = append(hints, kh)
			}
		}
	}

	timeLimit := 0
	if diff == DiffImpossible {
		timeLimit = 50
	}

	return Level{
		ID:           fmt.Sprintf("endless-%s-%d", diff, round),
		Difficulty:   diff,
		Number:       round,
		Title:        fmt.Sprintf("%s Drill #%d", theme.name, round),
		GoalType:     GoalMatchBuffer,
		TimeLimitSec: timeLimit,
		StartText:    startText,
		StartCursor:  startCursor,
		TargetText:   targetText,
		Par:          par,
		Tutorial: TutorialCard{
			Objective:   strings.Join(objParts, " + "),
			Explanation: "Combine the target edits in any order using your Vim motions and operators.",
			Keys:        hints,
			Walkthrough: "Move to each changed line, apply the edit, and match the Target Buffer.",
		},
		Hint:       "Compare the ≠ lines on the Target Buffer and plan your shortest path!",
		OptimalSeq: optSeq,
	}
}

func mutationPoolForDifficulty(diff DifficultyID, rng *rand.Rand) []lineMutation {
	switch diff {
	case DiffBeginner:
		return beginnerMutations(rng)
	case DiffIntermediate:
		return intermediateMutations(rng)
	case DiffHard:
		return hardMutations(rng)
	case DiffExpert:
		return expertMutations(rng)
	case DiffImpossible:
		return expertMutations(rng)
	default:
		return beginnerMutations(rng)
	}
}

func beginnerMutations(rng *rand.Rand) []lineMutation {
	type typoPair struct {
		bad    string
		good   string
		delCol int
	}
	typos := []typoPair{
		{"portt", "port", 4},
		{"hosst", "host", 2},
		{"modde", "mode", 3},
		{"codde", "code", 3},
		{"flagg", "flag", 4},
		{"timme", "time", 3},
	}
	tp := typos[rng.Intn(len(typos))]
	val := 10 + rng.Intn(89)

	type missingChar struct {
		bad   string
		good  string
		col   int
		ch    string
		after bool
	}
	missing := []missingChar{
		{"prt = 80", "port = 80", 1, "o", false},
		{"hst = 1", "host = 1", 1, "o", false},
		{"ok = fals", "ok = false", 8, "e", true},
		{"run = tru", "run = true", 8, "e", true},
	}
	mc := missing[rng.Intn(len(missing))]
	insCmd := fmt.Sprintf("%dli%s<Esc>", mc.col, mc.ch)
	if mc.after {
		insCmd = fmt.Sprintf("%dla%s<Esc>", mc.col, mc.ch)
	}

	debugLines := []string{
		"  // DEBUG: delete this line",
		"  temp_unused = 999",
		"  # REMOVE_ME",
	}
	dbg := debugLines[rng.Intn(len(debugLines))]

	addLines := []string{"  ready = true", "  # verified", "  retries = 3"}
	addLine := addLines[rng.Intn(len(addLines))]
	anchorLines := []string{"  step = 1", "  state = init", "  zone = local"}
	anchor := anchorLines[rng.Intn(len(anchorLines))]

	return []lineMutation{
		{
			startLines:    []string{fmt.Sprintf("  %s = %d", tp.bad, val)},
			targetLines:   []string{fmt.Sprintf("  %s = %d", tp.good, val)},
			solveFromCol0: fmt.Sprintf("%dlx", tp.delCol+2),
			objective:     fmt.Sprintf("Fix '%s' -> '%s'", tp.bad, tp.good),
			keys:          []KeyHint{{Key: "x", Desc: "Delete char under cursor"}},
		},
		{
			startLines:    []string{"  " + mc.bad},
			targetLines:   []string{"  " + mc.good},
			solveFromCol0: fmt.Sprintf("2l%s", insCmd),
			objective:     fmt.Sprintf("Fix '%s' -> '%s'", mc.bad, mc.good),
			keys:          []KeyHint{{Key: "i / a", Desc: "Insert before / after cursor"}},
		},
		{
			startLines:    []string{dbg},
			targetLines:   nil,
			solveFromCol0: "dd",
			objective:     "Delete stray line",
			keys:          []KeyHint{{Key: "dd", Desc: "Delete entire line"}},
		},
		{
			startLines:    []string{anchor},
			targetLines:   []string{anchor, addLine},
			solveFromCol0: fmt.Sprintf("o%s<Esc>", addLine),
			objective:     fmt.Sprintf("Add '%s'", strings.TrimSpace(addLine)),
			keys:          []KeyHint{{Key: "o ... <Esc>", Desc: "Open new line below"}},
		},
	}
}

func intermediateMutations(rng *rand.Rand) []lineMutation {
	vars := []string{"alpha", "beta", "gamma", "delta"}
	v := vars[rng.Intn(len(vars))]
	n := 1 + rng.Intn(50)

	ports := []struct{ old, nw string }{{"8080", "443"}, {"9000", "3000"}, {"7070", "5000"}}
	p := ports[rng.Intn(len(ports))]

	junkWords := []string{"broken", "stale", "unused"}
	jw := junkWords[rng.Intn(len(junkWords))]

	oldW := []string{"oldMode", "tmpFlag", "badKey"}
	newW := []string{"newMode", "appFlag", "apiKey"}
	idx := rng.Intn(len(oldW))

	cloneItems := []string{"  worker = 1", "  route = ok", "  shard = 0"}
	ci := cloneItems[rng.Intn(len(cloneItems))]

	return []lineMutation{
		{
			startLines:    []string{fmt.Sprintf("  %s = %d", v, n)},
			targetLines:   []string{fmt.Sprintf("  let %s = %d;", v, n)},
			solveFromCol0: "Ilet <Esc>A;<Esc>",
			objective:     fmt.Sprintf("Wrap '%s' with 'let ' and ';'", v),
			keys:          []KeyHint{{Key: "I / A", Desc: "Insert at non-blank / Append at EOL"}},
		},
		{
			startLines:    []string{fmt.Sprintf("  bind(host, %s)", p.old)},
			targetLines:   []string{fmt.Sprintf("  bind(host, %s)", p.nw)},
			solveFromCol0: fmt.Sprintf("f%ccw%s<Esc>", p.old[0], p.nw),
			objective:     fmt.Sprintf("Change port %s -> %s", p.old, p.nw),
			keys:          []KeyHint{{Key: "f<c> + cw", Desc: "Snipe char & change word"}},
		},
		{
			startLines:    []string{fmt.Sprintf("  %s = %s sync()", oldW[idx], jw)},
			targetLines:   []string{fmt.Sprintf("  %s = sync()", newW[idx])},
			solveFromCol0: fmt.Sprintf("wcw%s<Esc>2wdw", newW[idx]),
			objective:     fmt.Sprintf("Rename '%s' -> '%s' & drop '%s'", oldW[idx], newW[idx], jw),
			keys:          []KeyHint{{Key: "cw / dw", Desc: "Change word / Delete word"}},
		},
		{
			startLines:    []string{ci},
			targetLines:   []string{ci, ci},
			solveFromCol0: "yyp",
			objective:     fmt.Sprintf("Duplicate '%s'", strings.TrimSpace(ci)),
			keys:          []KeyHint{{Key: "yyp", Desc: "Yank & paste line below"}},
		},
	}
}

func hardMutations(rng *rand.Rand) []lineMutation {
	titles := []string{"Ready", "Active", "Online", "Final"}
	nt := titles[rng.Intn(len(titles))]
	drops := []string{"drop-me", "temp", "stale"}
	dr := drops[rng.Intn(len(drops))]
	dropCnt := 2 + rng.Intn(2)
	var dropLines []string
	for i := 0; i < dropCnt; i++ {
		dropLines = append(dropLines, fmt.Sprintf("  drop_%d()", i+1))
	}
	ret := 10 + rng.Intn(89)

	return []lineMutation{
		{
			startLines:    []string{"  title = \"Draft\""},
			targetLines:   []string{fmt.Sprintf("  title = \"%s\"", nt)},
			solveFromCol0: fmt.Sprintf("ci\"%s<Esc>", nt),
			objective:     fmt.Sprintf("Change \"Draft\" -> \"%s\"", nt),
			keys:          []KeyHint{{Key: "ci\"", Desc: "Change inside double quotes"}},
		},
		{
			startLines:    []string{fmt.Sprintf("  tag = \"%s\" + \"v2\"", dr)},
			targetLines:   []string{"  tag =  + \"v2\""},
			solveFromCol0: "da\"",
			objective:     fmt.Sprintf("Delete \"%s\" with quotes", dr),
			keys:          []KeyHint{{Key: "da\"", Desc: "Delete around double quotes"}},
		},
		{
			startLines:    []string{"  sum(10, 20, 30)"},
			targetLines:   []string{"  sum(x, y)"},
			solveFromCol0: "f(ci(x, y<Esc>",
			objective:     "Replace sum(...) args with x, y",
			keys:          []KeyHint{{Key: "f( + ci(", Desc: "Jump to '(' & change inside"}},
		},
		{
			startLines:    []string{"  list = [1, 2, 3, 4]"},
			targetLines:   []string{"  list = []"},
			solveFromCol0: "f[di[",
			objective:     "Empty list [...]",
			keys:          []KeyHint{{Key: "f[ + di[", Desc: "Jump to '[' & delete inside"}},
		},
		{
			startLines:    dropLines,
			targetLines:   nil,
			solveFromCol0: fmt.Sprintf("%ddd", dropCnt),
			objective:     fmt.Sprintf("Delete %d drop lines", dropCnt),
			keys:          []KeyHint{{Key: "<N>dd", Desc: "Delete N lines"}},
		},
		{
			startLines:    []string{"  return", fmt.Sprintf("    %d", ret)},
			targetLines:   []string{fmt.Sprintf("  return %d", ret)},
			solveFromCol0: "J",
			objective:     fmt.Sprintf("Join 'return' and '%d'", ret),
			keys:          []KeyHint{{Key: "J", Desc: "Join next line"}},
		},
	}
}

func expertMutations(rng *rand.Rand) []lineMutation {
	toks := []string{"ok_ok", "up_up", "go_go", "hi_hi"}
	tok := toks[rng.Intn(len(toks))]
	upper := strings.ReplaceAll(strings.ToUpper(tok), "_", "-")

	nums := []string{"9999", "8888", "7777"}
	newNums := []string{"80", "443", "200"}
	ni := rng.Intn(len(nums))

	return []lineMutation{
		{
			startLines:    []string{fmt.Sprintf("  state = %s", tok)},
			targetLines:   []string{fmt.Sprintf("  state = %s", upper)},
			solveFromCol0: fmt.Sprintf("f%cv$~f_r-", tok[0]),
			objective:     fmt.Sprintf("Convert '%s' -> '%s'", tok, upper),
			keys:          []KeyHint{{Key: "v$~ / r-", Desc: "Visual case flip & replace char"}},
		},
		{
			startLines:    []string{"  send(old_a, old_b, old_c);"},
			targetLines:   []string{"  send(payload);"},
			solveFromCol0: "f(lct)payload<Esc>",
			objective:     "Replace send(...) args using ct)",
			keys:          []KeyHint{{Key: "ct)", Desc: "Change till ')'"}},
		},
		{
			startLines:    []string{fmt.Sprintf("  port: %s;", nums[ni])},
			targetLines:   []string{fmt.Sprintf("  port: %s;", newNums[ni])},
			solveFromCol0: fmt.Sprintf("f%cct;%s<Esc>", nums[ni][0], newNums[ni]),
			objective:     fmt.Sprintf("Change %s -> %s using ct;", nums[ni], newNums[ni]),
			keys:          []KeyHint{{Key: "ct;", Desc: "Change till ';'"}},
		},
		{
			startLines:    []string{"if ok {", "emit()", "flush()", "}"},
			targetLines:   []string{"if ok {", "  emit()", "  flush()", "}"},
			solveFromCol0: "jVj>",
			objective:     "Indent emit() & flush()",
			keys:          []KeyHint{{Key: "Vj>", Desc: "Visual-line indent block"}},
		},
		{
			startLines:    []string{"  run(x) { old() }"},
			targetLines:   []string{"  run(x) { ok() }"},
			solveFromCol0: "f{ci{ ok() <Esc>",
			objective:     "Swap body inside {...} to ok()",
			keys:          []KeyHint{{Key: "ci{", Desc: "Change inside curly braces"}},
		},
	}
}

func genHardSearchBlitz(round int, rng *rand.Rand) Level {
	pairs := [][2]string{
		{"BUG", "FIX"},
		{"TODO", "DONE"},
		{"STUB", "IMPL"},
		{"HACK", "SAFE"},
	}
	p := pairs[rng.Intn(len(pairs))]
	hits := 2 + rng.Intn(2)
	var startLines, targetLines []string
	hitPos := map[int]bool{}
	for len(hitPos) < hits {
		hitPos[1+rng.Intn(5)] = true
	}
	for r := 0; r < 7; r++ {
		if hitPos[r] {
			startLines = append(startLines, fmt.Sprintf("%s()", p[0]))
			targetLines = append(targetLines, fmt.Sprintf("%s()", p[1]))
		} else {
			startLines = append(startLines, fmt.Sprintf("ok_%d()", r+1))
			targetLines = append(targetLines, fmt.Sprintf("ok_%d()", r+1))
		}
	}
	opt := fmt.Sprintf("/%s<CR>ciw%s<Esc>%s", p[0], p[1], strings.Repeat("n.", hits-1))
	return Level{
		ID:          fmt.Sprintf("endless-hard-%d", round),
		Difficulty:  DiffHard,
		Number:      round,
		Title:       fmt.Sprintf("Search Blitz #%d", round),
		GoalType:    GoalMatchBuffer,
		StartText:   strings.Join(startLines, "\n"),
		StartCursor: vim.Position{Row: 0, Col: 0},
		TargetText:  strings.Join(targetLines, "\n"),
		Par:         keyCountForSeq(opt),
		Tutorial: TutorialCard{
			Objective:   fmt.Sprintf("Find all %d '%s()' calls using / and replace '%s' -> '%s'.", hits, p[0], p[0], p[1]),
			Explanation: "Search with /pattern<Enter>, edit once with ciw, then hop and repeat with n and '.'.",
			Keys: []KeyHint{
				{Key: "/text<CR>", Desc: "Search forward"},
				{Key: "n / .", Desc: "Next match / Repeat edit"},
			},
			Walkthrough: fmt.Sprintf("/%s<CR>ciw%s<Esc> then repeat n. for remaining matches.", p[0], p[1]),
		},
		Hint:       "Search once with /, edit with ciw, then alternate n and .",
		OptimalSeq: opt,
	}
}

func genImpossibleGlobal(round int, rng *rand.Rand) Level {
	switch rng.Intn(3) {
	case 0:
		words := []string{"alpha", "beta", "gamma", "delta", "omega", "sigma", "theta", "kappa"}
		rng.Shuffle(len(words), func(i, j int) { words[i], words[j] = words[j], words[i] })
		count := 4 + rng.Intn(3)
		chosen := words[:count]
		var targetLines []string
		for _, w := range chosen {
			targetLines = append(targetLines, fmt.Sprintf("- \"%s\",", w))
		}
		opt := fmt.Sprintf("qaI- \"<Esc>A\",<Esc>j0q%d@a", count-1)
		return Level{
			ID:           fmt.Sprintf("endless-impossible-%d", round),
			Difficulty:   DiffImpossible,
			Number:       round,
			Title:        fmt.Sprintf("Macro Storm #%d", round),
			GoalType:     GoalMatchBuffer,
			TimeLimitSec: 55,
			StartText:    strings.Join(chosen, "\n"),
			StartCursor:  vim.Position{Row: 0, Col: 0},
			TargetText:   strings.Join(targetLines, "\n"),
			Par:          keyCountForSeq(opt),
			Tutorial: TutorialCard{
				Objective:   fmt.Sprintf("Format all %d lines into '- \"<word>\",' under 55s using a macro!", count),
				Explanation: "Record line 1 + j0 into @a with qa...q, then replay across the rest with <N>@a.",
				Keys: []KeyHint{
					{Key: "qa ... q", Desc: "Record macro 'a'"},
					{Key: "<N>@a", Desc: "Replay macro 'a' N times"},
				},
				Walkthrough: fmt.Sprintf("qaI- \"<Esc>A\",<Esc>j0q then %d@a.", count-1),
			},
			Hint:       "Remember to include j0 inside your macro before pressing q!",
			OptimalSeq: opt,
		}

	case 1:
		tags := []string{"X", "Y", "Z", "1", "2", "3", "P", "Q"}
		rng.Shuffle(len(tags), func(i, j int) { tags[i], tags[j] = tags[j], tags[i] })
		l1 := fmt.Sprintf("first: %s", tags[0])
		l2 := fmt.Sprintf("second: %s", tags[1])
		l3 := fmt.Sprintf("third: %s", tags[2])
		opt := "\"addddp\"ap"
		return Level{
			ID:           fmt.Sprintf("endless-impossible-%d", round),
			Difficulty:   DiffImpossible,
			Number:       round,
			Title:        fmt.Sprintf("Register Juggler #%d", round),
			GoalType:     GoalMatchBuffer,
			TimeLimitSec: 45,
			StartText:    strings.Join([]string{l1, l2, l3}, "\n"),
			StartCursor:  vim.Position{Row: 0, Col: 0},
			TargetText:   strings.Join([]string{l3, l2, l1}, "\n"),
			Par:          keyCountForSeq(opt),
			Tutorial: TutorialCard{
				Objective:   "Reverse all 3 lines under 45s using named registers.",
				Explanation: "Save line 1 into register 'a' (\"add) before deleting line 2 (dd) so p and \"ap reorder all 3 lines.",
				Keys: []KeyHint{
					{Key: "\"add", Desc: "Delete line into register 'a'"},
					{Key: "\"ap", Desc: "Paste from register 'a'"},
				},
				Walkthrough: "\"add -> dd -> p -> \"ap.",
			},
			Hint:       "\"add saves line 1 safely into register a.",
			OptimalSeq: opt,
		}

	default:
		pairs := [][2]string{
			{"foo", "bar"},
			{"tmp", "out"},
			{"val", "res"},
			{"src", "dst"},
		}
		p := pairs[rng.Intn(len(pairs))]
		start := fmt.Sprintf("let %s = 1\nlet x = %s + 2\nreturn %s * %s", p[0], p[0], p[0], p[0])
		target := fmt.Sprintf("let %s = 1\nlet x = %s + 2\nreturn %s * %s", p[1], p[1], p[1], p[1])
		opt := fmt.Sprintf("/%s<CR>ciw%s<Esc>n.n.n.", p[0], p[1])
		return Level{
			ID:           fmt.Sprintf("endless-impossible-%d", round),
			Difficulty:   DiffImpossible,
			Number:       round,
			Title:        fmt.Sprintf("Dot Blitz #%d", round),
			GoalType:     GoalMatchBuffer,
			TimeLimitSec: 45,
			StartText:    start,
			StartCursor:  vim.Position{Row: 0, Col: 0},
			TargetText:   target,
			Par:          keyCountForSeq(opt),
			Tutorial: TutorialCard{
				Objective:   fmt.Sprintf("Rename all 4 occurrences of '%s' -> '%s' under 45s!", p[0], p[1]),
				Explanation: "Search once, ciw once, then hammer n. n. n.",
				Keys: []KeyHint{
					{Key: "/word<CR>", Desc: "Search"},
					{Key: "ciw + n.", Desc: "Change word + jump & repeat"},
				},
				Walkthrough: fmt.Sprintf("/%s<CR>ciw%s<Esc>n.n.n.", p[0], p[1]),
			},
			Hint:       "Never re-type the replacement—press n. after the first edit!",
			OptimalSeq: opt,
		}
	}
}

func genCodeHunt(diff DifficultyID, round int, rng *rand.Rand) Level {
	themes := snippetThemes()
	theme := themes[rng.Intn(len(themes))]
	lines := append([]string(nil), theme.lines...)

	// Collect candidate token positions on each line (start of words/numbers/quotes/brackets)
	type cand struct {
		pos   vim.Position
		token string
	}
	var perRow [][]cand
	for r, line := range lines {
		var rowCands []cand
		rs := []rune(line)
		for c := 0; c < len(rs); c++ {
			if (c == 0 || rs[c-1] == ' ' || rs[c-1] == '(' || rs[c-1] == '{' || rs[c-1] == '"' || rs[c-1] == '=') &&
				rs[c] != ' ' {
				end := c
				for end < len(rs) && rs[end] != ' ' && rs[end] != ',' && rs[end] != ')' && rs[end] != ';' {
					end++
				}
				tok := string(rs[c:end])
				if len(tok) >= 1 {
					rowCands = append(rowCands, cand{pos: vim.Position{Row: r, Col: c}, token: tok})
				}
			}
		}
		if len(rowCands) > 0 {
			perRow = append(perRow, rowCands)
		}
	}

	cpCount := 3
	if diff == DiffHard || diff == DiffExpert {
		cpCount = 4
	} else if diff == DiffImpossible {
		cpCount = 5
	}
	if round >= 6 && cpCount < 5 {
		cpCount++
	}
	if cpCount > len(perRow) {
		cpCount = len(perRow)
	}

	// Pick random starting position & distinct row targets
	startRow := rng.Intn(len(lines))
	startCol := 0
	if len(lines[startRow]) > 0 {
		startCol = rng.Intn(len(lines[startRow]))
	}
	startPos := vim.Position{Row: startRow, Col: startCol}

	perm := rng.Perm(len(perRow))
	var cps []vim.Position
	var labels []string
	for _, idx := range perm {
		if len(cps) >= cpCount {
			break
		}
		rowList := perRow[idx]
		pick := rowList[rng.Intn(len(rowList))]
		if pick.pos == startPos {
			continue
		}
		cps = append(cps, pick.pos)
		labels = append(labels, pick.token)
	}
	if len(cps) == 0 {
		cps = append(cps, vim.Position{Row: len(lines) - 1, Col: 0})
		labels = append(labels, "end")
	}

	// Compute exact traversal sequence using a scratch Vim engine (accounts for line-length clamping)
	startText := strings.Join(lines, "\n")
	sim := vim.NewEngine(startText, startPos)
	var seq strings.Builder
	for _, cp := range cps {
		if cp.Row > sim.Cursor.Row {
			d := cp.Row - sim.Cursor.Row
			mv := "j"
			if d > 1 {
				mv = fmt.Sprintf("%dj", d)
			}
			seq.WriteString(mv)
			sim.Feed(mv)
		} else if cp.Row < sim.Cursor.Row {
			d := sim.Cursor.Row - cp.Row
			mv := "k"
			if d > 1 {
				mv = fmt.Sprintf("%dk", d)
			}
			seq.WriteString(mv)
			sim.Feed(mv)
		}
		if cp.Col > sim.Cursor.Col {
			d := cp.Col - sim.Cursor.Col
			mv := "l"
			if d > 1 {
				mv = fmt.Sprintf("%dl", d)
			}
			seq.WriteString(mv)
			sim.Feed(mv)
		} else if cp.Col < sim.Cursor.Col {
			d := sim.Cursor.Col - cp.Col
			mv := "h"
			if d > 1 {
				mv = fmt.Sprintf("%dh", d)
			}
			seq.WriteString(mv)
			sim.Feed(mv)
		}
	}
	opt := seq.String()
	par := sim.KeyCount + 8
	if diff == DiffHard || diff == DiffExpert {
		par = sim.KeyCount + 5
	} else if diff == DiffImpossible {
		par = sim.KeyCount + 3
	}

	timeLimit := 0
	if diff == DiffImpossible {
		timeLimit = 30
	}

	return Level{
		ID:           fmt.Sprintf("endless-hunt-%s-%d", diff, round),
		Difficulty:   diff,
		Number:       round,
		Title:        fmt.Sprintf("Code Hunt: %s #%d", theme.name, round),
		GoalType:     GoalReachCursor,
		TimeLimitSec: timeLimit,
		StartText:    strings.Join(lines, "\n"),
		StartCursor:  startPos,
		Checkpoints:  cps,
		Par:          par,
		Tutorial: TutorialCard{
			Objective:   fmt.Sprintf("Land on the %d highlighted code tokens (%s) in order.", len(cps), strings.Join(labels, " -> ")),
			Explanation: "Navigate real code using hjkl, w/b/e, 0/^/$, f<char>, or /search instead of blank grids.",
			Keys: []KeyHint{
				{Key: "w / b / e", Desc: "Jump across words"},
				{Key: "0 / ^ / $", Desc: "Line start / first non-blank / end"},
				{Key: "f<c> / /", Desc: "Snipe char / Search"},
			},
			Walkthrough: "Use line motions (j/k, ^/$) and word/char jumps (w/b/f) to hit each cyan highlight.",
		},
		Hint:       "Use w/b or f<char> once on the target line instead of tapping h/l!",
		OptimalSeq: opt,
	}
}
