package image

import (
	"archive/tar"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Inoriol/lxspawn/internal/deploy"
	"github.com/Inoriol/lxspawn/internal/kernel"
	"github.com/Inoriol/lxspawn/internal/logger"
)

const lxcImageBaseURL = "https://images.linuxcontainers.org/images"

const maxDownloadSize = 2 * 1024 * 1024 * 1024 // 2 GB
const maxDecompressedSize = 10 * 1024 * 1024 * 1024 // 10 GB

func unixModeToFileMode(mode int64) os.FileMode {
	fm := os.FileMode(mode & 0777)
	if mode&syscall.S_ISUID != 0 {
		fm |= os.ModeSetuid
	}
	if mode&syscall.S_ISGID != 0 {
		fm |= os.ModeSetgid
	}
	if mode&syscall.S_ISVTX != 0 {
		fm |= os.ModeSticky
	}
	return fm
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme == "http" {
			return fmt.Errorf("refusing to follow redirect to insecure HTTP: %s", req.URL)
		}
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

func getDownloadTimeout() time.Duration {
	if v := os.Getenv("LXSPAWN_DOWNLOAD_TIMEOUT"); v != "" {
		if minutes, err := strconv.Atoi(v); err == nil && minutes > 0 {
			return time.Duration(minutes) * time.Minute
		}
	}
	return 30 * time.Minute
}

func newDownloadClient() *http.Client {
	return &http.Client{
		Timeout: getDownloadTimeout(),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme == "http" {
				return fmt.Errorf("refusing to follow redirect to insecure HTTP: %s", req.URL)
			}
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
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

	if err := f.verifyDownload(tmpFile, url); err != nil {
		logger.Warn(fmt.Sprintf("image verification failed: %v", err))
	}

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
		cmd := deploy.Sudo("chown", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()), f.TargetDir)
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
	cmd := deploy.Sudo("btrfs", "subvolume", "create", path)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("btrfs subvolume create failed: %w (%s)", err, string(output))
	}
	
	cmd = deploy.Sudo("chown", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()), path)
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
	if strings.HasPrefix(url, "http://") {
		return "", fmt.Errorf("insecure HTTP downloads are not allowed, use HTTPS: %s", url)
	}

	resp, err := newDownloadClient().Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	cacheDir := filepath.Join(deploy.LxspawnDataDirPath(), "cache")
	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	tmpFile, err := os.CreateTemp(cacheDir, "rootfs-*.tar.xz")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	limitedReader := io.LimitReader(resp.Body, maxDownloadSize+1)
	written, err := io.Copy(tmpFile, limitedReader)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write download: %w", err)
	}

	if written > maxDownloadSize {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("downloaded file exceeds maximum size of %d bytes", maxDownloadSize)
	}

	logger.Info(fmt.Sprintf("Downloaded %d bytes", written))
	return tmpFile.Name(), nil
}

func (f *Fetcher) verifyDownload(filePath, imageURL string) error {
	shaURL := imageURL[:strings.LastIndex(imageURL, "/")] + "/SHA256SUMS"

	resp, err := httpClient.Get(shaURL)
	if err != nil {
		return fmt.Errorf("failed to download SHA256SUMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SHA256SUMS not available (HTTP %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read SHA256SUMS: %w", err)
	}

	expectedHash := ""
	filename := imageURL[strings.LastIndex(imageURL, "/")+1:]
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == filename {
			expectedHash = parts[0]
			break
		}
	}

	if expectedHash == "" {
		return fmt.Errorf("SHA256SUMS does not contain hash for %s", filename)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return err
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	if actualHash != expectedHash {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	logger.Info(fmt.Sprintf("SHA256 verified: %s", actualHash))
	return nil
}

func extractTarXz(tarXzPath, targetDir string) (err error) {
	f, err := os.Open(tarXzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	xzReader, err := newXZReader(f)
	if err != nil {
		return fmt.Errorf("failed to create xz reader: %w", err)
	}
	defer func() {
		if closeErr := xzReader.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("xz close error: %w", closeErr)
		}
	}()

	limitedReader := io.LimitReader(xzReader, maxDecompressedSize+1)
	tarReader := tar.NewReader(limitedReader)

	cleanTargetDir := filepath.Clean(targetDir)
	var totalDecompressed int64

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read error: %w", err)
		}

		totalDecompressed += header.Size
		if totalDecompressed > maxDecompressedSize {
			return fmt.Errorf("decompressed size exceeds maximum of %d bytes", maxDecompressedSize)
		}

		name := header.Name
		if strings.HasPrefix(name, "./") {
			name = name[2:]
		}
		name = strings.TrimPrefix(name, "/")

		if name == "" {
			continue
		}

		target := filepath.Join(targetDir, name)

		if !strings.HasPrefix(filepath.Clean(target)+string(os.PathSeparator), cleanTargetDir+string(os.PathSeparator)) && filepath.Clean(target) != cleanTargetDir {
			return fmt.Errorf("tar entry escapes target directory: %s", header.Name)
		}

		if err := checkNoSymlinkInPath(target, cleanTargetDir); err != nil {
			return fmt.Errorf("refusing to extract %s: %w", header.Name, err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, unixModeToFileMode(header.Mode)|0700); err != nil {
				return err
			}
			if os.Geteuid() == 0 {
				if err := os.Lchown(target, int(header.Uid), int(header.Gid)); err != nil {
					return fmt.Errorf("failed to chown directory %s: %w", target, err)
				}
				if err := os.Chmod(target, unixModeToFileMode(header.Mode)); err != nil {
					return fmt.Errorf("failed to chmod directory %s: %w", target, err)
				}
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|syscall.O_NOFOLLOW, unixModeToFileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			if os.Geteuid() == 0 {
				if err := os.Lchown(target, int(header.Uid), int(header.Gid)); err != nil {
					return fmt.Errorf("failed to chown file %s: %w", target, err)
				}
				if err := os.Chmod(target, unixModeToFileMode(header.Mode)); err != nil {
					return fmt.Errorf("failed to chmod file %s: %w", target, err)
				}
			}
		case tar.TypeSymlink:
			parentDir := filepath.Dir(target)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
			}
			linkTarget := header.Linkname
			var resolvedLinkTarget string
			if filepath.IsAbs(linkTarget) {
				resolvedLinkTarget = filepath.Join(targetDir, linkTarget)
			} else {
				resolvedLinkTarget = filepath.Join(filepath.Dir(target), linkTarget)
			}
			cleanLinkTarget := filepath.Clean(resolvedLinkTarget)
			if !strings.HasPrefix(cleanLinkTarget+string(os.PathSeparator), cleanTargetDir+string(os.PathSeparator)) && cleanLinkTarget != cleanTargetDir {
				return fmt.Errorf("symlink target escapes destination directory: %s -> %s", header.Name, header.Linkname)
			}
			os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("failed to create symlink %s -> %s: %w", target, header.Linkname, err)
			}
			if os.Geteuid() == 0 {
				if err := os.Lchown(target, int(header.Uid), int(header.Gid)); err != nil {
					return fmt.Errorf("failed to lchown symlink %s: %w", target, err)
				}
			}
		case tar.TypeLink:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			linkTarget := filepath.Join(targetDir, header.Linkname)
			cleanLinkTarget := filepath.Clean(linkTarget)
			if !strings.HasPrefix(cleanLinkTarget+string(os.PathSeparator), cleanTargetDir+string(os.PathSeparator)) && cleanLinkTarget != cleanTargetDir {
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

func checkNoSymlinkInPath(path, root string) error {
	cleanRoot := filepath.Clean(root)
	cleanPath := filepath.Clean(path)

	relPath, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return err
	}

	parts := strings.Split(relPath, string(os.PathSeparator))
	current := cleanRoot

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		next := filepath.Join(current, part)

		fi, err := os.Lstat(next)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path component %s is a symlink", next)
		}

		current = next
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
