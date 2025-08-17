package selfupdate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	release_url = "https://api.github.com/repos/flowmitry/syncai/releases/latest"
)

type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type releaseResponse struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

// Run downloads the latest release binary for the current platform and replaces the current executable.
func Run() error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Determine expected asset name (raw binary, not archived)
	assetBase := fmt.Sprintf("syncai_%s_%s", goos, goarch)
	wantedAsset := assetBase
	if goos == "windows" {
		wantedAsset += ".exe"
	}

	rel, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("fetch latest release: %w", err)
	}

	var url string
	for _, a := range rel.Assets {
		if a.Name == wantedAsset {
			url = a.BrowserDownloadURL
			break
		}
	}
	if url == "" {
		return fmt.Errorf("no asset found for %s/%s (expected %s)", goos, goarch, wantedAsset)
	}

	// Download binary to temp file
	tmpDir := os.TempDir()
	tmpBin := filepath.Join(tmpDir, fmt.Sprintf("%s-upd-%d", "syncai", time.Now().UnixNano()))
	if goos == "windows" {
		tmpBin += ".exe"
	}
	if err := downloadFile(url, tmpBin); err != nil {
		return fmt.Errorf("download binary: %w", err)
	}

	// Ensure executable permissions on Unix
	if goos != "windows" {
		_ = os.Chmod(tmpBin, 0o755)
	}
	// Ensure temporary file is cleaned up
	defer os.Remove(tmpBin)

	// Replace current executable
	curExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate current executable: %w", err)
	}
	curExe, _ = filepath.EvalSymlinks(curExe)

	// Try to rename over
	if goos == "windows" {
		// On Windows, replacing a running executable often fails. Try best-effort.
		_ = os.Remove(curExe)
		if err := os.Rename(tmpBin, curExe); err != nil {
			// Fallback: leave alongside as .new
			newPath := curExe + ".new"
			_ = os.Remove(newPath)
			if copyErr := copyFile(tmpBin, newPath); copyErr != nil {
				return fmt.Errorf("unable to place new binary (%v / %v)", err, copyErr)
			}
			return fmt.Errorf("updated binary saved to %s; please replace the running executable after exit", newPath)
		}
		return nil
	}

	// Unix: attempt atomic replace in place; if permission denied, fall back to saving in temp and instruct user.
	targetDir := filepath.Dir(curExe)
	finalTmp := filepath.Join(targetDir, ".syncai-update.tmp")
	if err := copyFile(tmpBin, finalTmp); err != nil {
		// Fallback: cannot write into install dir (likely permission denied). Save in temp and instruct user.
		fallbackPath := filepath.Join(os.TempDir(), filepath.Base(curExe)+".new")
		_ = os.Remove(fallbackPath)
		if copyErr := copyFile(tmpBin, fallbackPath); copyErr != nil {
			return fmt.Errorf("prepare updated binary: %v; fallback failed: %v", err, copyErr)
		}
		_ = os.Chmod(fallbackPath, 0o755)
		return fmt.Errorf("updated binary saved to %s; install it with: sudo mv %s %s", fallbackPath, fallbackPath, curExe)
	}
	if err := os.Chmod(finalTmp, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	if err := os.Rename(finalTmp, curExe); err != nil {
		_ = os.Remove(finalTmp)
		// Fallback on rename failure (e.g., permissions): save to temp and instruct user.
		fallbackPath := filepath.Join(os.TempDir(), filepath.Base(curExe)+".new")
		_ = os.Remove(fallbackPath)
		if copyErr := copyFile(tmpBin, fallbackPath); copyErr != nil {
			return fmt.Errorf("replace executable: %v; fallback failed: %v", err, copyErr)
		}
		_ = os.Chmod(fallbackPath, 0o755)
		return fmt.Errorf("updated binary saved to %s; install it with: sudo mv %s %s", fallbackPath, fallbackPath, curExe)
	}
	return nil
}

func fetchLatestRelease() (*releaseResponse, error) {
	req, err := http.NewRequest("GET", release_url, nil)
	if err != nil {
		return nil, err
	}
	// Set UA to avoid 403s
	req.Header.Set("User-Agent", "SyncAI")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("github api: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var rel releaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func downloadFile(url, path string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "SyncAI")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return err
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return nil
}
