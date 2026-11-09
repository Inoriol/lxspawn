package image

import (
	"archive/tar"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFetcherValid(t *testing.T) {
	f, err := NewFetcher("test", "debian:bookworm", "/tmp/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.MachineName != "test" {
		t.Errorf("expected MachineName 'test', got %q", f.MachineName)
	}
	if f.Distro != "debian" {
		t.Errorf("expected Distro 'debian', got %q", f.Distro)
	}
	if f.Release != "bookworm" {
		t.Errorf("expected Release 'bookworm', got %q", f.Release)
	}
	if f.TargetDir != "/tmp/test" {
		t.Errorf("expected TargetDir '/tmp/test', got %q", f.TargetDir)
	}
}

func TestNewFetcherInvalidDistro(t *testing.T) {
	tests := []struct {
		name   string
		distro string
	}{
		{"no colon", "debian"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFetcher("test", tt.distro, "/tmp/test")
			if err == nil {
				t.Error("expected error for invalid distro")
			}
		})
	}
}

func TestNewFetcherMultipleColons(t *testing.T) {
	f, err := NewFetcher("test", "debian:bookworm:extra", "/tmp/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Distro != "debian" {
		t.Errorf("expected Distro 'debian', got %q", f.Distro)
	}
	if f.Release != "bookworm:extra" {
		t.Errorf("expected Release 'bookworm:extra', got %q", f.Release)
	}
}

func TestArchToLXC(t *testing.T) {
	tests := []struct {
		goarch   string
		expected string
	}{
		{"amd64", "amd64"},
		{"arm64", "arm64"},
		{"arm", "armhf"},
		{"386", "386"},
		{"riscv64", "riscv64"},
	}

	for _, tt := range tests {
		t.Run(tt.goarch, func(t *testing.T) {
			result := archToLXC(tt.goarch)
			if result != tt.expected {
				t.Errorf("archToLXC(%q) = %q, want %q", tt.goarch, result, tt.expected)
			}
		})
	}
}

func TestExtractTarXz(t *testing.T) {
	if _, err := exec.LookPath("xz"); err != nil {
		t.Skip("xz not available, skipping")
	}

	tmpDir := t.TempDir()
	tarXzPath := filepath.Join(tmpDir, "test.tar.xz")
	createTestTarXz(t, tarXzPath)

	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatalf("failed to create extract dir: %v", err)
	}

	if err := extractTarXz(tarXzPath, extractDir); err != nil {
		t.Fatalf("extractTarXz failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(extractDir, "testdir")); err != nil {
		t.Errorf("expected testdir to exist: %v", err)
	}

	if _, err := os.Stat(filepath.Join(extractDir, "testfile.txt")); err != nil {
		t.Errorf("expected testfile.txt to exist: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(extractDir, "testfile.txt"))
	if err != nil {
		t.Fatalf("failed to read testfile.txt: %v", err)
	}
	if string(content) != "hello world" {
		t.Errorf("expected 'hello world', got %q", string(content))
	}

	link, err := os.Readlink(filepath.Join(extractDir, "testlink"))
	if err != nil {
		t.Fatalf("expected testlink to exist: %v", err)
	}
	if link != "testfile.txt" {
		t.Errorf("expected symlink to point to 'testfile.txt', got %q", link)
	}
}

func TestExtractTarXzPathTraversal(t *testing.T) {
	if _, err := exec.LookPath("xz"); err != nil {
		t.Skip("xz not available, skipping")
	}

	tmpDir := t.TempDir()
	tarXzPath := filepath.Join(tmpDir, "evil.tar.xz")
	createEvilTarXz(t, tarXzPath)

	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatalf("failed to create extract dir: %v", err)
	}

	err := extractTarXz(tarXzPath, extractDir)
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Errorf("expected 'escapes' in error message, got %q", err.Error())
	}
}

func TestExtractTarXzNonexistentFile(t *testing.T) {
	err := extractTarXz("/nonexistent/file.tar.xz", "/tmp/test")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestExtractTarXzSubdirectories(t *testing.T) {
	if _, err := exec.LookPath("xz"); err != nil {
		t.Skip("xz not available, skipping")
	}

	tmpDir := t.TempDir()
	tarXzPath := filepath.Join(tmpDir, "nested.tar.xz")
	createNestedTarXz(t, tarXzPath)

	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatalf("failed to create extract dir: %v", err)
	}

	if err := extractTarXz(tarXzPath, extractDir); err != nil {
		t.Fatalf("extractTarXz failed: %v", err)
	}

	nestedFile := filepath.Join(extractDir, "a", "b", "c", "deep.txt")
	if _, err := os.Stat(nestedFile); err != nil {
		t.Errorf("expected deeply nested file to exist: %v", err)
	}
}

func createTestTarXz(t *testing.T, outputPath string) {
	t.Helper()

	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "test.tar")

	f, err := os.Create(tarPath)
	if err != nil {
		t.Fatalf("failed to create tar: %v", err)
	}

	tw := tar.NewWriter(f)

	tw.WriteHeader(&tar.Header{
		Name:     "testdir/",
		Typeflag: tar.TypeDir,
		Mode:     0755,
	})

	content := []byte("hello world")
	tw.WriteHeader(&tar.Header{
		Name:     "testfile.txt",
		Typeflag: tar.TypeReg,
		Mode:     0644,
		Size:     int64(len(content)),
	})
	tw.Write(content)

	tw.WriteHeader(&tar.Header{
		Name:     "testlink",
		Typeflag: tar.TypeSymlink,
		Linkname: "testfile.txt",
	})

	tw.Close()
	f.Close()

	compressTarToXz(t, tarPath, outputPath)
}

func createEvilTarXz(t *testing.T, outputPath string) {
	t.Helper()

	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "evil.tar")

	f, err := os.Create(tarPath)
	if err != nil {
		t.Fatalf("failed to create tar: %v", err)
	}

	tw := tar.NewWriter(f)

	tw.WriteHeader(&tar.Header{
		Name:     "../../../etc/evil",
		Typeflag: tar.TypeReg,
		Mode:     0644,
		Size:     4,
	})
	tw.Write([]byte("evil"))

	tw.Close()
	f.Close()

	compressTarToXz(t, tarPath, outputPath)
}

func createNestedTarXz(t *testing.T, outputPath string) {
	t.Helper()

	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "nested.tar")

	f, err := os.Create(tarPath)
	if err != nil {
		t.Fatalf("failed to create tar: %v", err)
	}

	tw := tar.NewWriter(f)

	dirs := []string{"a/", "a/b/", "a/b/c/"}
	for _, d := range dirs {
		tw.WriteHeader(&tar.Header{
			Name:     d,
			Typeflag: tar.TypeDir,
			Mode:     0755,
		})
	}

	content := []byte("deep content")
	tw.WriteHeader(&tar.Header{
		Name:     "a/b/c/deep.txt",
		Typeflag: tar.TypeReg,
		Mode:     0644,
		Size:     int64(len(content)),
	})
	tw.Write(content)

	tw.Close()
	f.Close()

	compressTarToXz(t, tarPath, outputPath)
}

func compressTarToXz(t *testing.T, tarPath, outputPath string) {
	t.Helper()

	inFile, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("failed to open tar: %v", err)
	}
	defer inFile.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		t.Fatalf("failed to create xz output: %v", err)
	}
	defer outFile.Close()

	cmd := exec.Command("xz", "-c")
	cmd.Stdin = inFile
	cmd.Stdout = outFile

	if err := cmd.Run(); err != nil {
		t.Fatalf("xz compression failed: %v", err)
	}
}

func TestDownloadRejectsHTTP(t *testing.T) {
	f := &Fetcher{
		MachineName: "test",
		Distro:      "debian",
		Release:     "bookworm",
		Arch:        "amd64",
		TargetDir:   t.TempDir(),
	}

	// Test that HTTP URLs are rejected
	_, err := f.download("http://example.com/rootfs.tar.xz")
	if err == nil {
		t.Error("expected error for HTTP URL, got nil")
	}
	if !strings.Contains(err.Error(), "insecure HTTP") {
		t.Errorf("expected error to mention 'insecure HTTP', got: %v", err)
	}
}
