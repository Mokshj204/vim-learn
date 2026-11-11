# vimlearn

An interactive ANSI terminal game for learning and mastering Vim keybindings, built in pure Go (zero external dependencies) for both **Windows PowerShell** and **POSIX** terminals (Linux, macOS, BSD).

## Quick Install

### Linux & macOS (POSIX)
```bash
curl -fsSL https://raw.githubusercontent.com/Mokshj204/vim-learn/main/install.sh | bash
```
Or download the binary directly with `curl` (replace `linux-amd64` with `linux-arm64`, `darwin-amd64`, or `darwin-arm64`):
```bash
mkdir -p ~/.local/bin && curl -fsSL https://github.com/Mokshj204/vim-learn/releases/latest/download/vimlearn-linux-amd64 -o ~/.local/bin/vimlearn && chmod +x ~/.local/bin/vimlearn
```

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/Mokshj204/vim-learn/main/install.ps1 | iex
```
Or download the `.exe` directly with `curl.exe` in PowerShell:
```powershell
New-Item -ItemType Directory -Path "$env:USERPROFILE\.local\bin" -Force | Out-Null; curl.exe -fsSL -L "https://github.com/Mokshj204/vim-learn/releases/latest/download/vimlearn-windows-amd64.exe" -o "$env:USERPROFILE\.local\bin\vimlearn.exe"
```

After installing, launch from **any directory** with a single command (minimum terminal size `80x24`):
```text
vimlearn
```

---

## Features
- **Interactive Vim Primer (`[T] Tutorial`)**: 4 structured lessons explaining modal editing (`Normal`, `Insert`, `Visual`, `Command`), fast navigation, Vim's `Operator + Count + Motion` grammar, and power tools (dot repeat, search, marks, macros).
- **5 Difficulties × 5 Selectable Levels (25 Levels)**:
  - **Beginner**: `h j k l`, `w b`, `x`, `i a o`, `dd`, `u`
  - **Intermediate**: `0 ^ $`, `I A`, `f t ;`, `dw cw`, `yy p`, `.`
  - **Hard**: counts (`3j`, `3dd`), text objects (`ci"`, `da"`, `ci(`, `di[`), `gg G % J`, `/search n N`
  - **Expert**: Visual mode (`v V ~ r`), `ct`/`dt`, block indent (`>>`, `V>`), marks (`ma`, `` `a ``)
  - **Impossible** *(Timed)*: macros (`qa...q`, `@a`), named registers (`"a`), nested text objects, search + dot blitz
- **Choose Any Difficulty & Any Level**: Jump straight to any difficulty or level, view saved star ratings (`★★★`) and best keystroke counts, and replay for Par.
- **In-Level Guidance**: Every level displays a **What To Do** briefing, key hints (`Tab` to expand full lesson/keys, `Ctrl+L` for walkthrough & hint), live side-by-side **Target Buffer** diff (`✓` / `≠`), and a live **Keystrokes** tape.

## CLI Shortcuts
```text
vimlearn                         # Open interactive menu (difficulties, levels & tutorials)
vimlearn --tutorial              # Open interactive Vim primer tutorial
vimlearn --list                  # List all difficulties and levels
vimlearn --play <diff> <num>     # Jump straight into a level (e.g. --play hard 2)
```

## Build From Source
```sh
go install .
```

## Creating a GitHub Release
Push a semantic version tag starting with `v` to trigger `.github/workflows/release.yml`:
```sh
git tag v1.0.0
git push origin v1.0.0
```
GitHub Actions will run `go test -v ./...`, build binaries for Windows (`amd64`/`arm64`), Linux (`amd64`/`arm64`), and macOS (`amd64`/`arm64`), generate `checksums.txt`, and publish the GitHub Release automatically.
