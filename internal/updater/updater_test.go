package updater

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		cur, lat string
		want     bool
	}{
		{"dev", "v1.0.0", false},
		{"", "v1.0.0", false},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.0", "v1.1.0", true},
		{"v1.2.0", "v2.0.0", true},
		{"v2.0.0", "v1.9.9", false},
		{"1.0.0", "v1.0.2", true},
	}
	for _, tc := range cases {
		if got := IsNewer(tc.cur, tc.lat); got != tc.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tc.cur, tc.lat, got, tc.want)
		}
	}
}

func TestCheckLatestAndApplyToPath(t *testing.T) {
	origVer := Version
	Version = "v1.0.0"
	defer func() { Version = origVer }()

	assetName := AssetNameForTarget(runtime.GOOS, runtime.GOARCH)
	newPayload := []byte("NEW-VIMLEARN-BINARY")
	relJSON, _ := json.Marshal(ghRelease{
		TagName: "v1.1.0",
		Assets: []ghAsset{
			{Name: assetName, BrowserDownloadURL: "https://mock.local/download/" + assetName},
		},
	})

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			switch r.URL.Path {
			case "/repos/Mokshj204/vim-learn/releases/latest":
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(relJSON)),
					Header:     make(http.Header),
				}, nil
			case "/download/" + assetName:
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(newPayload)),
					Header:     make(http.Header),
				}, nil
			default:
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewReader(nil)),
					Header:     make(http.Header),
				}, nil
			}
		}),
	}

	info, err := CheckLatestWithClient(context.Background(), client, "https://mock.local", DefaultRepo, false)
	if err != nil {
		t.Fatalf("CheckLatestWithClient failed: %v", err)
	}
	if info == nil || info.LatestVersion != "v1.1.0" || info.AssetName != assetName {
		t.Fatalf("unexpected ReleaseInfo: %+v", info)
	}

	tmpDir := t.TempDir()
	targetExe := filepath.Join(tmpDir, assetName)
	if err := os.WriteFile(targetExe, []byte("OLD-BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := ApplyToPathWithClient(context.Background(), client, info, targetExe); err != nil {
		t.Fatalf("ApplyToPathWithClient failed: %v", err)
	}

	gotBytes, err := os.ReadFile(targetExe)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBytes) != string(newPayload) {
		t.Fatalf("expected updated payload %q, got %q", newPayload, gotBytes)
	}
}
