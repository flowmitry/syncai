package util

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func EnsureDir(dir string) error {
	if dir == "" || dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return nil
}

func WriteIfChanged(path string, data []byte) error {
	if b, err := os.ReadFile(path); err == nil {
		if bytes.Equal(b, data) {
			return nil // no change, skip writing
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func FileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	sum := h.Sum(nil)
	return hex.EncodeToString(sum), nil
}

func ReadFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open source: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}

	return data, nil
}
