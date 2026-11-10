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
	// Reject http:// URLs - only HTTPS is allowed for security
	if strings.HasPrefix(url, "http://") {
		return "", fmt.Errorf("insecure URL rejected: %s (use https://)", url)
	}

	if !strings.HasPrefix(url, "https://") {
		// Relative path, resolve against baseDir
		resolved := filepath.Join(baseDir, url)
		
		// Validate the resolved path doesn't escape the base directory
		absBase, err := filepath.Abs(baseDir)
		if err != nil {
			return "", fmt.Errorf("failed to resolve base directory: %w", err)
		}
		absResolved, err := filepath.Abs(resolved)
		if err != nil {
			return "", fmt.Errorf("failed to resolve path: %w", err)
		}
		if !strings.HasPrefix(absResolved, absBase) {
			return "", fmt.Errorf("path %q escapes base directory", url)
		}
		
		// Local file - digest is optional for local files
		if err := validateDigest(resolved, digest); err != nil {
			return "", err
		}
		return resolved, nil
	}

	// HTTPS URL - digest is required for remote files
	if digest == "" {
		return "", fmt.Errorf("digest is required for remote files: %s", url)
	}

	return f.downloadAndCache(url, digest)
}

func (f *FileHandler) downloadAndCache(url string, digest string) (string, error) {
	// Cache key includes digest to ensure different digests don't share cache
	cacheKey := fmt.Sprintf("%x-%s", sha256.Sum256([]byte(url)), digest[:min(16, len(digest))])
	cachePath := filepath.Join(f.CacheDir, cacheKey)

	// Check if cached file exists and matches digest
	if _, err := os.Stat(cachePath); err == nil {
		if err := validateDigest(cachePath, digest); err == nil {
			return cachePath, nil
		}
		// Cached file doesn't match digest, remove it
		os.Remove(cachePath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Use a client that doesn't follow redirects to HTTP
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Block redirects to HTTP
			if req.URL.Scheme == "http" {
				return fmt.Errorf("redirect to insecure URL blocked")
			}
			// Limit redirect chain
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

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

	// Create temp file first, then rename on success
	tempPath := cachePath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return "", err
	}
	defer func() {
		file.Close()
		os.Remove(tempPath) // Clean up temp file on failure
	}()

	// Use a counting reader to detect truncation
	limitedReader := io.LimitReader(resp.Body, maxDownloadSize+1)
	written, err := io.Copy(file, limitedReader)
	if err != nil {
		return "", err
	}

	// Check if the file was truncated
	if written > maxDownloadSize {
		return "", fmt.Errorf("file exceeds maximum size of %d bytes", maxDownloadSize)
	}

	file.Close()

	// Validate digest before moving to cache
	if err := validateDigest(tempPath, digest); err != nil {
		return "", err
	}

	// Move temp file to cache location
	if err := os.Rename(tempPath, cachePath); err != nil {
		return "", fmt.Errorf("failed to cache file: %w", err)
	}

	return cachePath, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
