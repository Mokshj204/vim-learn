package main

import (
	"bufio"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"vimlearn/internal/game"
	"vimlearn/internal/levels"
	"vimlearn/internal/term"
	"vimlearn/internal/updater"
)

//go:embed docs/vimlearn.1
var manPage string

func main() {
	updater.CleanupOldBinary()

	versionFlag := flag.Bool("version", false, "Print version and exit")
	manFlag := flag.Bool("man", false, "Print the roff manual page to stdout")
	updateFlag := flag.Bool("update", false, "Check GitHub Releases for updates and install")
	listFlag := flag.Bool("list", false, "List all difficulties and levels")
	tutorialFlag := flag.Bool("tutorial", false, "Start directly in the interactive Vim tutorial")
	playFlag := flag.Bool("play", false, "Jump straight to a difficulty and level: vimlearn --play <difficulty> <level>")
	endlessFlag := flag.String("endless", "", "Start Endless mode for a difficulty: --endless <diff> [random|multi|hunt|rapid]")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "vimlearn (%s) — Interactive ANSI Terminal Vim Learning Game\n\n", updater.Version)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  vimlearn                         Open interactive menu (difficulties, levels & tutorials)\n")
		fmt.Fprintf(os.Stderr, "  vimlearn --tutorial              Open interactive Vim primer tutorial\n")
		fmt.Fprintf(os.Stderr, "  vimlearn --list                  List all difficulties and levels\n")
		fmt.Fprintf(os.Stderr, "  vimlearn --play <diff> <num>     Start a specific level (e.g. --play intermediate 2)\n")
		fmt.Fprintf(os.Stderr, "  vimlearn --endless <diff> [mode] Start Endless mode (modes: random, multi, hunt, rapid)\n")
		fmt.Fprintf(os.Stderr, "  vimlearn --update                Check for and install the latest GitHub release\n")
		fmt.Fprintf(os.Stderr, "  vimlearn --man                   Print manual page (pipe to: man -l -)\n")
		fmt.Fprintf(os.Stderr, "  vimlearn --version               Print installed version\n\n")
		fmt.Fprintf(os.Stderr, "Difficulties: beginner, intermediate, hard, expert, impossible\n")
	}
	flag.Parse()

	if *versionFlag {
		fmt.Printf("vimlearn %s\n", updater.Version)
		return
	}

	if *manFlag {
		fmt.Print(manPage)
		return
	}

	if *updateFlag {
		runCLIUpdate()
		return
	}

	if *listFlag {
		printCatalog()
		return
	}

	t := term.New()
	app := game.NewApp(t)

	if *tutorialFlag {
		app.OpenTutorial()
	} else if *endlessFlag != "" {
		diff := levels.DifficultyID(strings.ToLower(*endlessFlag))
		mode := levels.EndlessRandom
		if args := flag.Args(); len(args) > 0 {
			mode = levels.EndlessMode(strings.ToLower(args[0]))
		}
		if err := app.StartEndlessMode(diff, mode); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
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

func runCLIUpdate() {
	fmt.Printf("Current version: %s\nChecking GitHub Releases for updates...\n", updater.Version)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := updater.CheckLatest(ctx, "", updater.DefaultRepo, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Update check failed: %v\n", err)
		os.Exit(1)
	}
	if info == nil {
		fmt.Println("You are already on the latest release!")
		return
	}

	fmt.Printf("Update available: %s -> %s (%s)\nInstall now? [y/N]: ", info.CurrentVersion, info.LatestVersion, info.AssetName)
	reader := bufio.NewReader(os.Stdin)
	ans, _ := reader.ReadString('\n')
	ans = strings.TrimSpace(strings.ToLower(ans))
	if ans != "y" && ans != "yes" {
		fmt.Println("Update cancelled.")
		return
	}

	fmt.Printf("Downloading %s...\n", info.LatestVersion)
	dlCtx, dlCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer dlCancel()
	if err := updater.Apply(dlCtx, info); err != nil {
		fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Successfully updated vimlearn to %s!\n", info.LatestVersion)
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
