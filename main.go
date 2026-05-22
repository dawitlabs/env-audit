package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const version = "0.1.0"

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
)

type envFile struct {
	path    string
	rel     string
	vars    map[string]string // key → value
	tracked bool              // committed to git
}

type duplicate struct {
	key      string
	value    string
	files    []string
}

var skipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"vendor":       true,
	".next":        true,
	"dist":         true,
	"build":        true,
	".svelte-kit":  true,
}

var sourceExts = map[string]bool{
	".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
	".py": true, ".rb": true, ".rs": true, ".svelte": true, ".vue": true,
	".sh": true, ".fish": true, ".yaml": true, ".yml": true, ".toml": true,
}

func isEnvFile(name string) bool {
	if name == ".env" {
		return true
	}
	return strings.HasPrefix(name, ".env.")
}

func parseEnv(path string) map[string]string {
	vars := map[string]string{}
	f, err := os.Open(path)
	if err != nil {
		return vars
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// strip surrounding quotes
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') ||
			(val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		vars[key] = val
	}
	return vars
}

func isGitTracked(path string) bool {
	dir := filepath.Dir(path)
	cmd := exec.Command("git", "-C", dir, "ls-files", "--error-unmatch", path)
	cmd.Stderr = nil
	return cmd.Run() == nil
}

func keyUsedInSource(root, key string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !sourceExts[filepath.Ext(path)] {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1024*1024), 1024*1024)
		for sc.Scan() {
			if strings.Contains(sc.Text(), key) {
				found = true
				return fmt.Errorf("done") // early exit
			}
		}
		return nil
	})
	return found
}

func projectRoot(envPath string) string {
	dir := filepath.Dir(envPath)
	for {
		if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return filepath.Dir(envPath)
		}
		dir = parent
	}
}

func section(title, sub string) {
	fmt.Printf("\n%s%s%s  %s%s%s\n", bold, title, reset, dim, sub, reset)
	fmt.Println(strings.Repeat("━", 54))
}

func shorten(path, root string) string {
	home, _ := os.UserHomeDir()
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	if strings.HasPrefix(rel, "..") {
		rel = strings.Replace(path, home, "~", 1)
	}
	return rel
}

func main() {
	defaultRoot := filepath.Join(os.Getenv("HOME"), "projects")
	root := flag.String("root", defaultRoot, "root directory to scan")
	flag.Parse()

	absRoot, _ := filepath.Abs(*root)
	if _, err := os.Stat(absRoot); err != nil {
		fmt.Fprintf(os.Stderr, "directory not found: %s\n", absRoot)
		os.Exit(1)
	}

	fmt.Printf("\n%s╔══════════════════════════════════════════════╗%s\n", cyan, reset)
	fmt.Printf("%s║%s  %senv-audit%s v%s  ·  %s%-23s%s  %s║%s\n",
		cyan, reset, bold, reset, version, dim, shorten(absRoot, filepath.Dir(absRoot)), reset, cyan, reset)
	fmt.Printf("%s╚══════════════════════════════════════════════╝%s\n", cyan, reset)

	fmt.Printf("\n%sscanning…%s\n", dim, reset)

	// Walk and collect env files
	var envFiles []envFile
	_ = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !isEnvFile(d.Name()) {
			return nil
		}
		vars := parseEnv(path)
		tracked := isGitTracked(path)
		envFiles = append(envFiles, envFile{
			path:    path,
			rel:     shorten(path, absRoot),
			vars:    vars,
			tracked: tracked,
		})
		return nil
	})

	if len(envFiles) == 0 {
		fmt.Printf("\n  %sno .env files found in %s%s\n\n", yellow, absRoot, reset)
		return
	}

	// ── Exposed in git ──────────────────────────────────────────────
	var exposed []envFile
	for _, ef := range envFiles {
		if ef.tracked {
			exposed = append(exposed, ef)
		}
	}

	section("EXPOSED IN GIT", fmt.Sprintf("%d files", len(exposed)))
	if len(exposed) == 0 {
		fmt.Printf("  %s✓ none%s\n", green, reset)
	} else {
		for _, ef := range exposed {
			fmt.Printf("  %s%s%-40s%s  %s%d vars  ← tracked in git!%s\n",
				bold+red, reset, ef.rel, reset, yellow, len(ef.vars), reset)
		}
	}

	// ── Duplicate secrets ───────────────────────────────────────────
	// key:value → list of files
	type kv struct{ k, v string }
	kvMap := map[kv][]string{}
	for _, ef := range envFiles {
		for k, v := range ef.vars {
			if v == "" {
				continue
			}
			key := kv{k, v}
			kvMap[key] = append(kvMap[key], ef.rel)
		}
	}

	var dups []duplicate
	for kv, files := range kvMap {
		if len(files) > 1 {
			dups = append(dups, duplicate{key: kv.k, value: kv.v, files: files})
		}
	}
	sort.Slice(dups, func(i, j int) bool { return len(dups[i].files) > len(dups[j].files) })

	section("DUPLICATE SECRETS", fmt.Sprintf("%d pairs", len(dups)))
	if len(dups) == 0 {
		fmt.Printf("  %s✓ none%s\n", green, reset)
	} else {
		for _, d := range dups {
			masked := d.value
			if len(masked) > 6 {
				masked = masked[:3] + strings.Repeat("*", len(masked)-3)
			}
			fmt.Printf("  %s%-32s%s %s(%s)%s\n", bold, d.key, reset, dim, masked, reset)
			for _, f := range d.files {
				fmt.Printf("    %s%s%s\n", dim, f, reset)
			}
		}
	}

	// ── Unreferenced vars ───────────────────────────────────────────
	type unreffed struct {
		file string
		key  string
	}
	var unused []unreffed

	for _, ef := range envFiles {
		root := projectRoot(ef.path)
		for key := range ef.vars {
			if !keyUsedInSource(root, key) {
				unused = append(unused, unreffed{ef.rel, key})
			}
		}
	}

	section("UNREFERENCED VARS", fmt.Sprintf("%d vars", len(unused)))
	if len(unused) == 0 {
		fmt.Printf("  %s✓ none%s\n", green, reset)
	} else {
		// group by file
		byFile := map[string][]string{}
		var fileOrder []string
		seen := map[string]bool{}
		for _, u := range unused {
			if !seen[u.file] {
				fileOrder = append(fileOrder, u.file)
				seen[u.file] = true
			}
			byFile[u.file] = append(byFile[u.file], u.key)
		}
		for _, f := range fileOrder {
			fmt.Printf("  %s%s%s\n", bold, f, reset)
			for _, k := range byFile[f] {
				fmt.Printf("    %s%-32s%s  not found in source\n", yellow, k, reset)
			}
		}
	}

	// ── Summary ─────────────────────────────────────────────────────
	totalVars := 0
	for _, ef := range envFiles {
		totalVars += len(ef.vars)
	}

	section("SUMMARY", "")
	fmt.Printf("  %d .env files  ·  %d vars  ·  %s%d exposed%s  ·  %s%d duplicate secrets%s  ·  %s%d unreferenced%s\n\n",
		len(envFiles), totalVars,
		colorCount(len(exposed)), len(exposed), reset,
		colorCount(len(dups)), len(dups), reset,
		colorCount(len(unused)), len(unused), reset,
	)
}

func colorCount(n int) string {
	if n == 0 {
		return green
	}
	return red
}
