package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const DefaultRepo = "Mokshj204/vim-learn"

// Version is set at build time via -ldflags "-X vimlearn/internal/updater.Version=vX.Y.Z".
var Version = "dev"

// ReleaseInfo describes an available update from GitHub Releases.
type ReleaseInfo struct {
	CurrentVersion string
	LatestVersion  string
	AssetName      string
	DownloadURL    string
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// AssetNameForTarget returns the expected release asset filename for goos/goarch.
func AssetNameForTarget(goos, goarch string) string {
	name := fmt.Sprintf("vimlearn-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// IsNewer reports whether latest is a higher semantic version than current.
// Returns false for "dev" or malformed versions during automatic background checks.
func IsNewer(current, latest string) bool {
	if current == "" || current == "dev" {
		return false
	}
	cMaj, cMin, cPat, ok1 := parseSemver(current)
	lMaj, lMin, lPat, ok2 := parseSemver(latest)
	if !ok1 || !ok2 {
		return false
	}
	if lMaj != cMaj {
		return lMaj > cMaj
	}
	if lMin != cMin {
		return lMin > cMin
	}
	return lPat > cPat
}

func parseSemver(v string) (int, int, int, bool) {
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return 0, 0, 0, false
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	pat := 0
	var err3 error
	if len(parts) == 3 {
		pat, err3 = strconv.Atoi(parts[2])
	}
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return maj, min, pat, true
}

// CheckLatest queries GitHub Releases for a newer release for the current OS/arch.
// If force is true, it returns the latest release even when Version is "dev" (unless already equal).
func CheckLatest(ctx context.Context, apiBase, repo string, force bool) (*ReleaseInfo, error) {
	return CheckLatestWithClient(ctx, nil, apiBase, repo, force)
}

// CheckLatestWithClient is like CheckLatest but uses the provided HTTP client when non-nil.
func CheckLatestWithClient(ctx context.Context, client *http.Client, apiBase, repo string, force bool) (*ReleaseInfo, error) {
	if apiBase == "" {
		apiBase = "https://api.github.com"
	}
	if repo == "" {
		repo = DefaultRepo
	}
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	url := fmt.Sprintf("%s/repos/%s/releases/latest", strings.TrimRight(apiBase, "/"), repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "vimlearn/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases returned status %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, nil
	}

	if !force && !IsNewer(Version, rel.TagName) {
		return nil, nil
	}
	if force && Version != "dev" && !IsNewer(Version, rel.TagName) {
		return nil, nil
	}

	wantAsset := AssetNameForTarget(runtime.GOOS, runtime.GOARCH)
	for _, a := range rel.Assets {
		if a.Name == wantAsset && a.BrowserDownloadURL != "" {
			return &ReleaseInfo{
				CurrentVersion: Version,
				LatestVersion:  rel.TagName,
				AssetName:      a.Name,
				DownloadURL:    a.BrowserDownloadURL,
			}, nil
		}
	}
	return nil, nil
}

// CheckLatestAsync performs a background release check without blocking startup.
func CheckLatestAsync() <-chan *ReleaseInfo {
	ch := make(chan *ReleaseInfo, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		info, err := CheckLatest(ctx, "", DefaultRepo, false)
		if err == nil && info != nil {
			ch <- info
			return
		}
		close(ch)
	}()
	return ch
}

// Apply downloads the release asset and atomically replaces the running executable.
func Apply(ctx context.Context, info *ReleaseInfo) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return ApplyToPath(ctx, info, exe)
}

// ApplyToPath downloads the asset from info.DownloadURL and replaces targetPath safely
// (using .old rename on Windows so running executables can be replaced).
func ApplyToPath(ctx context.Context, info *ReleaseInfo, targetPath string) error {
	return ApplyToPathWithClient(ctx, nil, info, targetPath)
}

// ApplyToPathWithClient is like ApplyToPath using the provided HTTP client when non-nil.
func ApplyToPathWithClient(ctx context.Context, client *http.Client, info *ReleaseInfo, targetPath string) error {
	if info == nil || info.DownloadURL == "" {
		return fmt.Errorf("missing download URL")
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.DownloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "vimlearn/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	dir := filepath.Dir(targetPath)
	tmpFile, err := os.CreateTemp(dir, ".vimlearn-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	cleanupTmp := true
	defer func() {
		_ = tmpFile.Close()
		if cleanupTmp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := io.Copy(tmpFile, io.LimitReader(resp.Body, 100<<20)); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return err
	}

	oldPath := targetPath + ".old"
	_ = os.Remove(oldPath)

	if runtime.GOOS == "windows" {
		if _, err := os.Stat(targetPath); err == nil {
			if err := os.Rename(targetPath, oldPath); err != nil {
				return fmt.Errorf("failed to stage old executable: %w", err)
			}
		}
		if err := os.Rename(tmpPath, targetPath); err != nil {
			_ = os.Rename(oldPath, targetPath)
			return fmt.Errorf("failed to install new executable: %w", err)
		}
		cleanupTmp = false
		return nil
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		return err
	}
	cleanupTmp = false
	return nil
}

// CleanupOldBinary removes any leftover .old executable created during a Windows update.
func CleanupOldBinary() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	_ = os.Remove(exe + ".old")
}
