package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestSudoersCrossCheck validates that every deploy.Sudo() call site
// uses a binary that has a corresponding sudoers rule.
func TestSudoersCrossCheck(t *testing.T) {
	// Generate rules with dummy binary paths to get the set of binaries with rules
	dummyBinaries := map[string]string{
		"rm":          "/usr/bin/rm",
		"chown":       "/usr/bin/chown",
		"chmod":       "/usr/bin/chmod",
		"mv":          "/usr/bin/mv",
		"mkdir":       "/usr/bin/mkdir",
		"tar":         "/usr/bin/tar",
		"tee":         "/usr/bin/tee",
		"ln":          "/usr/bin/ln",
		"systemd-run": "/usr/bin/systemd-run",
		"journalctl":  "/usr/bin/journalctl",
		"systemctl":   "/usr/bin/systemctl",
		"btrfs":       "/usr/bin/btrfs",
	}
	rules := generateSudoersRules(dummyBinaries)

	// Extract the set of binaries that have rules
	binariesWithRules := make(map[string]bool)
	for _, rule := range rules {
		if strings.HasPrefix(rule, "#") || strings.TrimSpace(rule) == "" {
			continue
		}
		// Rule format: %lxspawn ALL=(root) NOPASSWD: /path/to/binary ^args...
		// Extract binary name from path
		parts := strings.Fields(rule)
		if len(parts) < 5 {
			continue
		}
		binPath := parts[4]
		binName := filepath.Base(binPath)
		binariesWithRules[binName] = true
	}

	// Find all Go files in the project
	rootDir := ".."
	goFiles, err := findGoFiles(rootDir)
	if err != nil {
		t.Fatalf("failed to find Go files: %v", err)
	}

	// Files/directories to skip (disabled backends)
	skipPatterns := []string{
		"internal/snapshot/lvm.go",
		"internal/snapshot/zfs.go",
		"internal/storage/lvm_thin.go",
		"internal/storage/detector.go",
	}

	// Track findings
	var sudoCalls []sudoCall
	var bypassCalls []bypassCall

	for _, file := range goFiles {
		// Check if file should be skipped
		relPath, _ := filepath.Rel(rootDir, file)
		skip := false
		for _, pattern := range skipPatterns {
			if strings.Contains(relPath, pattern) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		calls, bypasses, err := analyzeFile(file)
		if err != nil {
			t.Errorf("failed to analyze %s: %v", file, err)
			continue
		}
		sudoCalls = append(sudoCalls, calls...)
		bypassCalls = append(bypassCalls, bypasses...)
	}

	// Check that every deploy.Sudo() call uses a binary with a rule
	for _, call := range sudoCalls {
		if !binariesWithRules[call.binary] {
			t.Errorf("%s:%d: deploy.Sudo(%q, ...) has no sudoers rule", call.file, call.line, call.binary)
		}
	}

	// Whitelist for exec.Command("sudo", ...) bypasses
	// These are either safe (not privileged) or have been migrated to deploy.Sudo()
	bypassWhitelist := map[string]bool{
		// setup.go: sudo -V for version check (not a privileged operation)
		"internal/deploy/filesystem.go": false, // deploy.Sudo itself
	}

	// Check bypass calls
	for _, bypass := range bypassCalls {
		// Skip if in whitelist
		relPath, _ := filepath.Rel(rootDir, bypass.file)
		if bypassWhitelist[relPath] {
			continue
		}
		// setup.go getSudoVersion is safe (sudo -V for version check)
		if strings.Contains(relPath, "setup.go") && bypass.context == "-V" {
			continue
		}
		t.Errorf("%s:%d: exec.Command(\"sudo\", ...) bypass detected - use deploy.Sudo() instead: %s",
			bypass.file, bypass.line, bypass.context)
	}
}

type sudoCall struct {
	file   string
	line   int
	binary string
}

type bypassCall struct {
	file    string
	line    int
	context string
}

func findGoFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip vendor and test directories
			if info.Name() == "vendor" || info.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func analyzeFile(filename string) ([]sudoCall, []bypassCall, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	var sudoCalls []sudoCall
	var bypassCalls []bypassCall

	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check for deploy.Sudo(...)
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if ident.Name == "deploy" && sel.Sel.Name == "Sudo" {
					// Extract first string literal argument
					if len(call.Args) > 0 {
						if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							binary := strings.Trim(lit.Value, "\"")
							pos := fset.Position(call.Pos())
							sudoCalls = append(sudoCalls, sudoCall{
								file:   filename,
								line:   pos.Line,
								binary: binary,
							})
						}
					}
				}
			}
		}

		// Check for exec.Command("sudo", ...)
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if ident.Name == "exec" && sel.Sel.Name == "Command" {
					if len(call.Args) > 0 {
						if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							cmd := strings.Trim(lit.Value, "\"")
							if cmd == "sudo" {
								pos := fset.Position(call.Pos())
								// Get some context
								context := "sudo call"
								if len(call.Args) > 1 {
									if lit2, ok := call.Args[1].(*ast.BasicLit); ok && lit2.Kind == token.STRING {
										context = strings.Trim(lit2.Value, "\"")
									}
								}
								bypassCalls = append(bypassCalls, bypassCall{
									file:    filename,
									line:    pos.Line,
									context: context,
								})
							}
						}
					}
				}
			}
		}

		// Check for sudoOrDirectForRootfs calls (wrapper that may call deploy.Sudo)
		if ident, ok := call.Fun.(*ast.Ident); ok {
			if ident.Name == "sudoOrDirectForRootfs" {
				// Second argument is the binary name (first is rootfsPath)
				if len(call.Args) > 1 {
					if lit, ok := call.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						binary := strings.Trim(lit.Value, "\"")
						pos := fset.Position(call.Pos())
						sudoCalls = append(sudoCalls, sudoCall{
							file:   filename,
							line:   pos.Line,
							binary: binary,
						})
					}
				}
			}
		}

		return true
	})

	return sudoCalls, bypassCalls, nil
}

// TestGenerateSudoersRulesDeterministic verifies that generateSudoersRules
// produces consistent output for the same input.
func TestGenerateSudoersRulesDeterministic(t *testing.T) {
	binaries := map[string]string{
		"rm":          "/usr/bin/rm",
		"chown":       "/usr/bin/chown",
		"chmod":       "/usr/bin/chmod",
		"systemd-run": "/usr/bin/systemd-run",
	}

	rules1 := generateSudoersRules(binaries)
	rules2 := generateSudoersRules(binaries)

	if len(rules1) != len(rules2) {
		t.Errorf("generateSudoersRules not deterministic: got %d and %d rules", len(rules1), len(rules2))
	}

	for i := range rules1 {
		if rules1[i] != rules2[i] {
			t.Errorf("rule %d differs: %q vs %q", i, rules1[i], rules2[i])
		}
	}
}

// TestSudoersRulesMatchBinaries verifies that each binary in the input
// produces at least one rule.
func TestSudoersRulesMatchBinaries(t *testing.T) {
	binaries := map[string]string{
		"rm":          "/usr/bin/rm",
		"chown":       "/usr/bin/chown",
		"chmod":       "/usr/bin/chmod",
		"mv":          "/usr/bin/mv",
		"mkdir":       "/usr/bin/mkdir",
		"tar":         "/usr/bin/tar",
		"tee":         "/usr/bin/tee",
		"ln":          "/usr/bin/ln",
		"systemd-run": "/usr/bin/systemd-run",
		"journalctl":  "/usr/bin/journalctl",
		"systemctl":   "/usr/bin/systemctl",
		"btrfs":       "/usr/bin/btrfs",
	}

	rules := generateSudoersRules(binaries)
	rulesStr := strings.Join(rules, "\n")

	for binName, binPath := range binaries {
		if !strings.Contains(rulesStr, binPath) {
			t.Errorf("binary %s (%s) not found in any rule", binName, binPath)
		}
	}
}

// TestSudoersRulesNoEmptyBinaries verifies that empty binary map produces
// only comments and blank lines.
func TestSudoersRulesNoEmptyBinaries(t *testing.T) {
	rules := generateSudoersRules(map[string]string{})

	for _, rule := range rules {
		if strings.TrimSpace(rule) != "" && !strings.HasPrefix(rule, "#") {
			t.Errorf("unexpected non-comment rule with empty binaries: %q", rule)
		}
	}
}

// TestSudoersRulesPatterns verifies that generated rules use proper regex patterns.
func TestSudoersRulesPatterns(t *testing.T) {
	binaries := map[string]string{
		"rm":    "/usr/bin/rm",
		"chown": "/usr/bin/chown",
		"btrfs": "/usr/bin/btrfs",
	}

	rules := generateSudoersRules(binaries)

	// Check that machine name pattern is used
	namePat := regexp.MustCompile(`\[a-zA-Z0-9\]\[a-zA-Z0-9_-\]\*`)
	foundNamePat := false
	for _, rule := range rules {
		if namePat.MatchString(rule) {
			foundNamePat = true
			break
		}
	}
	if !foundNamePat {
		t.Error("no rules use machine name pattern")
	}

	// Check that rules start with %lxspawn
	for _, rule := range rules {
		if strings.TrimSpace(rule) == "" || strings.HasPrefix(rule, "#") {
			continue
		}
		if !strings.HasPrefix(rule, "%lxspawn") {
			t.Errorf("rule doesn't start with %%lxspawn: %q", rule)
		}
	}
}
