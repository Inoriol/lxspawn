package provision

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sha256sum(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func TestValidateDigestOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	digest := sha256sum(content)
	if err := validateDigest(path, digest); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateDigestMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := validateDigest(path, "sha256:0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Fatal("expected digest mismatch error")
	}
}

func TestValidateDigestEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := validateDigest(path, ""); err != nil {
		t.Errorf("empty digest should skip validation: %v", err)
	}
}

func TestValidateDigestFileNotFound(t *testing.T) {
	if err := validateDigest("/nonexistent/file.txt", "sha256:abc"); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestValidateDigestBinaryContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "binary.bin")
	content := []byte{0x00, 0x01, 0x02, 0xff, 0xfe, 0xfd}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	digest := sha256sum(content)
	if err := validateDigest(path, digest); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateDigestLargeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.bin")
	content := make([]byte, 1024*1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	digest := sha256sum(content)
	if err := validateDigest(path, digest); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestResolveFileRefLocal(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "script.sh")
	content := []byte("#!/bin/bash\necho hello")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatal(err)
	}

	fh := &FileHandler{CacheDir: t.TempDir()}
	got, err := fh.ResolveFileRef("script.sh", "", baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != filePath {
		t.Errorf("got %q, want %q", got, filePath)
	}
}

func TestResolveFileRefLocalWithDigest(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "script.sh")
	content := []byte("#!/bin/bash\necho hello")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatal(err)
	}

	digest := sha256sum(content)
	fh := &FileHandler{CacheDir: t.TempDir()}
	got, err := fh.ResolveFileRef("script.sh", digest, baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != filePath {
		t.Errorf("got %q, want %q", got, filePath)
	}
}

func TestResolveFileRefLocalDigestMismatch(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "script.sh")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	fh := &FileHandler{CacheDir: t.TempDir()}
	_, err := fh.ResolveFileRef("script.sh", "sha256:0000000000000000000000000000000000000000000000000000000000000000", baseDir)
	if err == nil {
		t.Fatal("expected digest mismatch error")
	}
}

func TestResolveFileRefLocalAbsolute(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "absolute.sh")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	fh := &FileHandler{CacheDir: t.TempDir()}
	got, err := fh.ResolveFileRef(filePath, "", "/some/other/dir")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	joined := filepath.Join("/some/other/dir", filePath)
	if got != joined {
		t.Errorf("got %q, want %q", got, joined)
	}
}

func TestResolveFileRefLocalNotFound(t *testing.T) {
	fh := &FileHandler{CacheDir: t.TempDir()}
	got, err := fh.ResolveFileRef("nonexistent.sh", "", t.TempDir())
	if err != nil {
		t.Fatalf("ResolveFileRef should not error for nonexistent local files: %v", err)
	}
	if got == "" {
		t.Error("should return a path even if file doesn't exist")
	}
}

func TestResolveFileRefSubdirectory(t *testing.T) {
	baseDir := t.TempDir()
	subDir := filepath.Join(baseDir, "scripts")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(subDir, "setup.sh")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	fh := &FileHandler{CacheDir: t.TempDir()}
	got, err := fh.ResolveFileRef("scripts/setup.sh", "", baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != filePath {
		t.Errorf("got %q, want %q", got, filePath)
	}
}

func TestResolveFileRefRejectsHTTP(t *testing.T) {
	fh := &FileHandler{CacheDir: t.TempDir()}
	_, err := fh.ResolveFileRef("http://example.com/script.sh", "sha256:abc", t.TempDir())
	if err == nil {
		t.Fatal("expected error for http:// URL")
	}
	if !strings.Contains(err.Error(), "insecure URL rejected") {
		t.Errorf("expected insecure URL error, got: %v", err)
	}
}

func TestResolveFileRefRequiresDigestForHTTPS(t *testing.T) {
	fh := &FileHandler{CacheDir: t.TempDir()}
	_, err := fh.ResolveFileRef("https://example.com/script.sh", "", t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing digest")
	}
	if !strings.Contains(err.Error(), "digest is required") {
		t.Errorf("expected digest required error, got: %v", err)
	}
}

func TestResolveFileRefRejectsPathTraversal(t *testing.T) {
	baseDir := t.TempDir()
	fh := &FileHandler{CacheDir: t.TempDir()}
	
	_, err := fh.ResolveFileRef("../../../etc/passwd", "", baseDir)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
	if !strings.Contains(err.Error(), "escapes base directory") {
		t.Errorf("expected path traversal error, got: %v", err)
	}
}

func TestResolveFileRefAllowsValidRelativePaths(t *testing.T) {
	baseDir := t.TempDir()
	filePath := filepath.Join(baseDir, "scripts", "setup.sh")
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	fh := &FileHandler{CacheDir: t.TempDir()}
	got, err := fh.ResolveFileRef("scripts/setup.sh", "", baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != filePath {
		t.Errorf("got %q, want %q", got, filePath)
	}
}
