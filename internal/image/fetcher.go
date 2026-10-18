package image

import (
	"archive/tar"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Inoriol/lxspawn/internal/kernel"
	"github.com/Inoriol/lxspawn/internal/logger"
)

const lxcImageBaseURL = "https://images.linuxcontainers.org/images"

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

var downloadClient = &http.Client{
	Timeout: 30 * time.Minute,
}

type Fetcher struct {
	MachineName string
	Distro      string
	Release     string
	Arch        string
	TargetDir   string
}

func NewFetcher(machineName, distro, targetDir string) (*Fetcher, error) {
	parts := strings.SplitN(distro, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("distro must be in format 'distro:release' (e.g., debian:bookworm), got %q", distro)
	}

	arch := archToLXC(runtime.GOARCH)

	return &Fetcher{
		MachineName: machineName,
		Distro:      parts[0],
		Release:     parts[1],
		Arch:        arch,
		TargetDir:   targetDir,
	}, nil
}

func (f *Fetcher) Fetch() error {
	url, err := f.buildURL()
	if err != nil {
		return fmt.Errorf("failed to build image URL: %w", err)
	}
	logger.Action(fmt.Sprintf("Downloading rootfs from %s", url))

	tmpFile, err := f.download(url)
	if err != nil {
		return fmt.Errorf("failed to download rootfs: %w", err)
	}
	defer os.Remove(tmpFile)

	logger.Action(fmt.Sprintf("Extracting rootfs to %s", f.TargetDir))
	
	if err := f.prepareTargetDir(); err != nil {
		return fmt.Errorf("failed to prepare target directory: %w", err)
	}

	if err := extractTarXz(tmpFile, f.TargetDir); err != nil {
		os.RemoveAll(f.TargetDir)
		return fmt.Errorf("failed to extract rootfs: %w", err)
	}

	if err := ValidateInit(f.TargetDir); err != nil {
		os.RemoveAll(f.TargetDir)
		return err
	}

	return nil
}

func (f *Fetcher) prepareTargetDir() error {
	// Check if directory already exists (created by storage backend)
	if _, err := os.Stat(f.TargetDir); err == nil {
		// Directory exists, chown to current user so extraction works
		cmd := exec.Command("sudo", "chown", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()), f.TargetDir)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("chown failed: %w (%s)", err, string(output))
		}
		return nil
	}

	parentDir := filepath.Dir(f.TargetDir)
	
	if kernel.IsBtrfs(parentDir) {
		logger.Info("Detected btrfs filesystem, creating subvolume")
		if err := createBtrfsSubvolume(f.TargetDir); err != nil {
			logger.Warn(fmt.Sprintf("Failed to create btrfs subvolume, falling back to regular directory: %v", err))
			return os.MkdirAll(f.TargetDir, 0755)
		}
		return nil
	}
	
	return os.MkdirAll(f.TargetDir, 0755)
}

func createBtrfsSubvolume(path string) error {
	cmd := exec.Command("sudo", "btrfs", "subvolume", "create", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("btrfs subvolume create failed: %w (%s)", err, string(output))
	}
	
	cmd = exec.Command("sudo", "chown", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()), path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chown failed: %w (%s)", err, string(output))
	}
	
	return nil
}

func fixRootfsOwnership(rootfsPath string) error {
	cmd := exec.Command("sudo", "chown", "-R", "0:0", rootfsPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("chown failed: %w (%s)", err, string(output))
	}
	return nil
}

func (f *Fetcher) buildURL() (string, error) {
	date, err := f.getLatestDate()
	if err != nil {
		return "", fmt.Errorf("failed to get latest date: %w", err)
	}
	return fmt.Sprintf("%s/%s/%s/%s/default/%s/rootfs.tar.xz",
		lxcImageBaseURL, f.Distro, f.Release, f.Arch, date), nil
}

func (f *Fetcher) getLatestDate() (string, error) {
	dirURL := fmt.Sprintf("%s/%s/%s/%s/default/",
		lxcImageBaseURL, f.Distro, f.Release, f.Arch)
	
	resp, err := httpClient.Get(dirURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch directory listing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	html := string(body)
	datePattern := regexp.MustCompile(`href="(\d{8}_\d{2}%3A\d{2})/?`)
	matches := datePattern.FindAllStringSubmatch(html, -1)

	var dates []string
	for _, match := range matches {
		if len(match) > 1 {
			date := strings.ReplaceAll(match[1], "%3A", ":")
			dates = append(dates, date)
		}
	}

	if len(dates) == 0 {
		return "", fmt.Errorf("no dated directories found")
	}

	sort.Strings(dates)
	return dates[len(dates)-1], nil
}

func (f *Fetcher) download(url string) (string, error) {
	resp, err := downloadClient.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "lxspawn-rootfs-*.tar.xz")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write download: %w", err)
	}

	logger.Info(fmt.Sprintf("Downloaded %d bytes", written))
	return tmpFile.Name(), nil
}

func extractTarXz(tarXzPath, targetDir string) error {
	f, err := os.Open(tarXzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	xzReader, err := newXZReader(f)
	if err != nil {
		return fmt.Errorf("failed to create xz reader: %w", err)
	}
	defer xzReader.Close()

	tarReader := tar.NewReader(xzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		target := filepath.Join(targetDir, header.Name)

		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(targetDir)+string(os.PathSeparator)) && filepath.Clean(target) != filepath.Clean(targetDir) {
			return fmt.Errorf("tar entry escapes target directory: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)|0700); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		case tar.TypeSymlink:
			parentDir := filepath.Dir(target)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
			}
			linkTarget := header.Linkname
			if filepath.IsAbs(linkTarget) {
				linkTarget = filepath.Join(targetDir, linkTarget)
			} else {
				linkTarget = filepath.Join(filepath.Dir(target), linkTarget)
			}
			cleanTarget := filepath.Clean(linkTarget)
			if !strings.HasPrefix(cleanTarget, filepath.Clean(targetDir)+string(os.PathSeparator)) && cleanTarget != filepath.Clean(targetDir) {
				return fmt.Errorf("symlink target escapes destination directory: %s -> %s", header.Name, header.Linkname)
			}
			os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("failed to create symlink %s -> %s: %w", target, header.Linkname, err)
			}
		case tar.TypeLink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			linkTarget := filepath.Join(targetDir, header.Linkname)
			cleanLinkTarget := filepath.Clean(linkTarget)
			if !strings.HasPrefix(cleanLinkTarget, filepath.Clean(targetDir)+string(os.PathSeparator)) && cleanLinkTarget != filepath.Clean(targetDir) {
				return fmt.Errorf("hardlink target escapes destination directory: %s -> %s", header.Name, header.Linkname)
			}
			os.Remove(target)
			if err := os.Link(linkTarget, target); err != nil {
				return err
			}
		}
	}

	return nil
}

type nopReadCloser struct {
	io.Reader
}

func (n *nopReadCloser) Close() error {
	return nil
}

func newXZReader(r io.Reader) (io.ReadCloser, error) {
	return newXZExecReader(r)
}

func archToLXC(goarch string) string {
	switch goarch {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "arm":
		return "armhf"
	default:
		return goarch
	}
}
