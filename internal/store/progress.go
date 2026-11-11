package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// LevelRecord tracks the player's best performance on a level.
type LevelRecord struct {
	Completed bool `json:"completed"`
	BestKeys  int  `json:"best_keys"`
	Stars     int  `json:"stars"`
}

// Progress holds saved progress across all levels and endless mode high scores.
type Progress struct {
	Levels      map[string]LevelRecord `json:"levels"`
	EndlessBest map[string]int         `json:"endless_best,omitempty"`
}

// StarsForScore computes a 1-3 star rating relative to Par.
func StarsForScore(keys, par int) int {
	if keys <= par {
		return 3
	}
	twoStarLimit := par + par/3
	if twoStarLimit < par+4 {
		twoStarLimit = par + 4
	}
	if keys <= twoStarLimit {
		return 2
	}
	return 1
}

func progressPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vimlearn", "progress.json"), nil
}

// Load reads saved progress from disk, returning empty Progress if missing.
func Load() *Progress {
	p := &Progress{
		Levels:      make(map[string]LevelRecord),
		EndlessBest: make(map[string]int),
	}
	path, err := progressPath()
	if err != nil {
		return p
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return p
	}
	_ = json.Unmarshal(data, p)
	if p.Levels == nil {
		p.Levels = make(map[string]LevelRecord)
	}
	if p.EndlessBest == nil {
		p.EndlessBest = make(map[string]int)
	}
	return p
}

// RecordEndlessStreak updates the best endless streak for a difficulty if beaten.
func (p *Progress) RecordEndlessStreak(diff string, streak int) {
	if streak > p.EndlessBest[diff] {
		p.EndlessBest[diff] = streak
		_ = p.Save()
	}
}

// RecordWin updates best score and stars for levelID and persists to disk.
func (p *Progress) RecordWin(levelID string, keys, par int) {
	stars := StarsForScore(keys, par)
	prev := p.Levels[levelID]
	if !prev.Completed || keys < prev.BestKeys || prev.BestKeys == 0 {
		prev.BestKeys = keys
	}
	if stars > prev.Stars {
		prev.Stars = stars
	}
	prev.Completed = true
	p.Levels[levelID] = prev
	_ = p.Save()
}

// Save writes progress to disk.
func (p *Progress) Save() error {
	path, err := progressPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
