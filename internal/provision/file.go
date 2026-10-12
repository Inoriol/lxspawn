package provision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultDownloadTimeout = 10 * time.Minute
	maxDownloadSize        = 100 * 1024 * 1024 // 100 MB
)

type FileHandler struct {
	CacheDir string
}

func NewFileHandler() (*FileHandler, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(home, ".cache", "lxspawn", "scripts")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return nil, err
	}

	return &FileHandler{CacheDir: cacheDir}, nil
}

func (f *FileHandler) ResolveFileRef(url string, digest string, baseDir string) (string, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		// Relative path, resolve against baseDir
		url = filepath.Join(baseDir, url)
	}

	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return f.downloadAndCache(url, digest)
	}

	// Local file
	if err := validateDigest(url, digest); err != nil {
		return "", err
	}

	return url, nil
}

func (f *FileHandler) downloadAndCache(url string, digest string) (string, error) {
	cacheKey := fmt.Sprintf("%x", sha256.Sum256([]byte(url)))
	cachePath := filepath.Join(f.CacheDir, cacheKey)

	if _, err := os.Stat(cachePath); err == nil {
		if digest != "" {
			if err := validateDigest(cachePath, digest); err != nil {
				os.Remove(cachePath)
			} else {
				return cachePath, nil
			}
		} else {
			return cachePath, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	if resp.ContentLength > maxDownloadSize {
		return "", fmt.Errorf("file size %d exceeds maximum %d bytes", resp.ContentLength, maxDownloadSize)
	}

	file, err := os.Create(cachePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	limitedReader := io.LimitReader(resp.Body, maxDownloadSize)
	if _, err := io.Copy(file, limitedReader); err != nil {
		os.Remove(cachePath)
		return "", err
	}

	if digest != "" {
		if err := validateDigest(cachePath, digest); err != nil {
			os.Remove(cachePath)
			return "", err
		}
	}

	return cachePath, nil
}

func validateDigest(path string, expected string) error {
	if expected == "" {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	actual := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return fmt.Errorf("digest mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}
