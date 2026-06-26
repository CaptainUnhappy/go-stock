package appdata

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	EnvHome = "GO_STOCK_HOME"
	AppName = "go-stock"
)

type Paths struct {
	BaseDir      string
	DataDir      string
	LogsDir      string
	DBPath       string
	WailsLogPath string
}

type MigrationResult struct {
	Migrated  bool
	SourceDir string
	TargetDir string
	Files     []string
}

func Resolve() (Paths, error) {
	baseDir := strings.TrimSpace(os.Getenv(EnvHome))
	if baseDir == "" {
		configDir, err := os.UserConfigDir()
		if err != nil || strings.TrimSpace(configDir) == "" {
			homeDir, homeErr := os.UserHomeDir()
			if homeErr != nil {
				if err != nil {
					return Paths{}, err
				}
				return Paths{}, homeErr
			}
			configDir = homeDir
		}
		baseDir = filepath.Join(configDir, AppName)
	}

	baseDir = filepath.Clean(baseDir)
	dataDir := filepath.Join(baseDir, "data")
	logsDir := filepath.Join(baseDir, "logs")

	return Paths{
		BaseDir:      baseDir,
		DataDir:      dataDir,
		LogsDir:      logsDir,
		DBPath:       filepath.Join(dataDir, "stock.db"),
		WailsLogPath: filepath.Join(logsDir, "wails.log"),
	}, nil
}

func Prepare() (Paths, MigrationResult, error) {
	paths, err := Resolve()
	if err != nil {
		return Paths{}, MigrationResult{}, err
	}
	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		return Paths{}, MigrationResult{}, err
	}
	if err := os.MkdirAll(paths.LogsDir, 0o755); err != nil {
		return Paths{}, MigrationResult{}, err
	}

	migration, err := migrateLegacyDatabase(paths)
	if err != nil {
		return Paths{}, migration, err
	}
	return paths, migration, nil
}

func DefaultDBPath() (string, error) {
	paths, _, err := Prepare()
	if err != nil {
		return "", err
	}
	return paths.DBPath, nil
}

type legacyCandidate struct {
	dir      string
	total    int64
	modified time.Time
}

func migrateLegacyDatabase(paths Paths) (MigrationResult, error) {
	if fileExists(paths.DBPath) {
		return MigrationResult{TargetDir: paths.DataDir}, nil
	}

	candidates := findLegacyCandidates(paths.DataDir)
	if len(candidates) == 0 {
		return MigrationResult{TargetDir: paths.DataDir}, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].total != candidates[j].total {
			return candidates[i].total > candidates[j].total
		}
		return candidates[i].modified.After(candidates[j].modified)
	})

	sourceDir := candidates[0].dir
	result := MigrationResult{
		Migrated:  true,
		SourceDir: sourceDir,
		TargetDir: paths.DataDir,
	}

	for _, name := range []string{"stock.db", "stock.db-wal", "stock.db-shm"} {
		src := filepath.Join(sourceDir, name)
		if !fileExists(src) {
			continue
		}
		dst := filepath.Join(paths.DataDir, name)
		if err := copyFile(src, dst); err != nil {
			return result, fmt.Errorf("migrate %s: %w", name, err)
		}
		result.Files = append(result.Files, name)
	}

	return result, nil
}

func findLegacyCandidates(targetDataDir string) []legacyCandidate {
	dirs := legacyDataDirs(targetDataDir)
	candidates := make([]legacyCandidate, 0, len(dirs))
	for _, dir := range dirs {
		dbPath := filepath.Join(dir, "stock.db")
		if !fileExists(dbPath) {
			continue
		}

		var total int64
		var modified time.Time
		for _, name := range []string{"stock.db", "stock.db-wal", "stock.db-shm"} {
			info, err := os.Stat(filepath.Join(dir, name))
			if err != nil {
				continue
			}
			total += info.Size()
			if info.ModTime().After(modified) {
				modified = info.ModTime()
			}
		}
		candidates = append(candidates, legacyCandidate{
			dir:      dir,
			total:    total,
			modified: modified,
		})
	}
	return candidates
}

func legacyDataDirs(targetDataDir string) []string {
	target, _ := filepath.Abs(targetDataDir)
	seen := map[string]bool{}
	dirs := make([]string, 0)

	add := func(dir string) {
		if strings.TrimSpace(dir) == "" {
			return
		}
		abs, err := filepath.Abs(filepath.Clean(dir))
		if err != nil {
			return
		}
		if samePath(abs, target) || seen[strings.ToLower(abs)] {
			return
		}
		seen[strings.ToLower(abs)] = true
		dirs = append(dirs, abs)
	}

	addRelativeDataDirs := func(base string) {
		if strings.TrimSpace(base) == "" {
			return
		}
		add(filepath.Join(base, "data"))
		add(filepath.Join(base, "..", "data"))
		add(filepath.Join(base, "..", "..", "data"))
	}

	if cwd, err := os.Getwd(); err == nil {
		addRelativeDataDirs(cwd)
	}
	if exe, err := os.Executable(); err == nil {
		addRelativeDataDirs(filepath.Dir(exe))
	}

	return dirs
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dst)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dst)
		return closeErr
	}

	if info, err := os.Stat(src); err == nil {
		_ = os.Chtimes(dst, info.ModTime(), info.ModTime())
	}
	return nil
}
