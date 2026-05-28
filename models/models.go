// Package models provides the whisper.cpp model catalogue, path resolution,
// and streaming HTTP download with progress reporting.
package models

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

const downloadBase = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/"

// Model describes a single whisper.cpp ggml model.
type Model struct {
	Name    string // e.g. "base.en"
	File    string // e.g. "ggml-base.en.bin"
	SizeStr string // human-readable size for display
}

// All is the full ordered catalogue of supported models.
var All = []Model{
	{Name: "base.en", File: "ggml-base.en.bin", SizeStr: "~142 MB"},
	{Name: "small.en", File: "ggml-small.en.bin", SizeStr: "~466 MB"},
	{Name: "medium.en", File: "ggml-medium.en.bin", SizeStr: "~1.5 GB"},
	{Name: "large-v3", File: "ggml-large-v3.bin", SizeStr: "~3.1 GB"},
}

// Dir returns the directory where models are stored.
func Dir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "go-talkback", "models")
}

// Path returns the full filesystem path for model m.
func Path(m Model) string {
	return filepath.Join(Dir(), m.File)
}

// IsDownloaded reports whether model m exists on disk.
func IsDownloaded(m Model) bool {
	_, err := os.Stat(Path(m))
	return err == nil
}

// Find returns a pointer to the model with the given name, or nil if not found.
func Find(name string) *Model {
	for i := range All {
		if All[i].Name == name {
			return &All[i]
		}
	}
	return nil
}

// Download streams model m from HuggingFace to disk. progress is called with
// values 0–100 as bytes arrive. The file is written to a .tmp path and renamed
// on success; the .tmp file is removed on failure.
func Download(m Model, progress func(pct int)) error {
	dest := Path(m)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}

	cleanup := func() {
		f.Close()
		os.Remove(tmp)
	}

	resp, err := http.Get(downloadBase + m.File)
	if err != nil {
		cleanup()
		return fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cleanup()
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 32*1024)
	lastPct := -1

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				cleanup()
				return fmt.Errorf("write: %w", writeErr)
			}
			written += int64(n)
			if total > 0 && progress != nil {
				pct := int(written * 100 / total)
				if pct != lastPct {
					progress(pct)
					lastPct = pct
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			cleanup()
			return fmt.Errorf("read: %w", readErr)
		}
	}

	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close: %w", err)
	}

	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}

	if progress != nil {
		progress(100)
	}
	return nil
}
