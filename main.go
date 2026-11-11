package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"vimlearn/internal/game"
	"vimlearn/internal/levels"
	"vimlearn/internal/term"
)

func main() {
	listFlag := flag.Bool("list", false, "List all difficulties and levels")
	tutorialFlag := flag.Bool("tutorial", false, "Start directly in the interactive Vim tutorial")
	playFlag := flag.Bool("play", false, "Jump straight to a difficulty and level: vimlearn --play <difficulty> <level>")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "vimlearn — Interactive ANSI Terminal Vim Learning Game\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  vimlearn                         Open interactive menu (difficulties, levels & tutorials)\n")
		fmt.Fprintf(os.Stderr, "  vimlearn --tutorial              Open interactive Vim primer tutorial\n")
		fmt.Fprintf(os.Stderr, "  vimlearn --list                  List all difficulties and levels\n")
		fmt.Fprintf(os.Stderr, "  vimlearn --play <diff> <num>     Start a specific level (e.g. --play intermediate 2)\n\n")
		fmt.Fprintf(os.Stderr, "Difficulties: beginner, intermediate, hard, expert, impossible\n")
	}
	flag.Parse()

	if *listFlag {
		printCatalog()
		return
	}

	t := term.New()
	app := game.NewApp(t)

	if *tutorialFlag {
		app.OpenTutorial()
	} else if *playFlag {
		args := flag.Args()
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "error: --play requires <difficulty> <level_number>")
			os.Exit(2)
		}
		diff := levels.DifficultyID(strings.ToLower(args[0]))
		num, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid level number %q\n", args[1])
			os.Exit(2)
		}
		if err := app.OpenLevel(diff, num); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "vimlearn error: %v\n", err)
		os.Exit(1)
	}
}

func printCatalog() {
	for _, d := range levels.Difficulties() {
		fmt.Printf("[%s] %s — %s\n", d.ID, d.Name, d.Tagline)
		for _, lv := range levels.ByDifficulty(d.ID) {
			timer := ""
			if lv.TimeLimitSec > 0 {
				timer = fmt.Sprintf(" (%ds)", lv.TimeLimitSec)
			}
			fmt.Printf("  %d. %-32s Par %2d%s  — %s\n", lv.Number, lv.Title, lv.Par, timer, lv.Tutorial.Objective)
		}
		fmt.Println()
	}
}
