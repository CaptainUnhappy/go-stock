package appdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareMigratesLargestLegacyDatabase(t *testing.T) {
	workspace := t.TempDir()
	binDir := filepath.Join(workspace, "build", "bin")
	rootDataDir := filepath.Join(workspace, "data")
	binDataDir := filepath.Join(binDir, "data")
	homeDir := filepath.Join(workspace, "home")

	mustWriteFile(t, filepath.Join(rootDataDir, "stock.db"), []byte("small"))
	mustWriteFile(t, filepath.Join(binDataDir, "stock.db"), []byte("larger legacy database"))
	mustWriteFile(t, filepath.Join(binDataDir, "stock.db-wal"), []byte("wal"))

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(binDir); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvHome, homeDir)

	paths, migration, err := Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if !migration.Migrated {
		t.Fatal("expected legacy database migration")
	}
	if migration.SourceDir != binDataDir {
		t.Fatalf("expected source %q, got %q", binDataDir, migration.SourceDir)
	}

	got, err := os.ReadFile(paths.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "larger legacy database" {
		t.Fatalf("unexpected migrated database content: %q", string(got))
	}
	if _, err := os.Stat(filepath.Join(paths.DataDir, "stock.db-wal")); err != nil {
		t.Fatalf("expected WAL file to migrate: %v", err)
	}
}

func TestPrepareDoesNotOverwriteExistingDatabase(t *testing.T) {
	workspace := t.TempDir()
	homeDir := filepath.Join(workspace, "home")
	targetDB := filepath.Join(homeDir, "data", "stock.db")
	legacyDB := filepath.Join(workspace, "data", "stock.db")

	mustWriteFile(t, targetDB, []byte("existing"))
	mustWriteFile(t, legacyDB, []byte("larger legacy database"))

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})
	if err := os.Chdir(workspace); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvHome, homeDir)

	paths, migration, err := Prepare()
	if err != nil {
		t.Fatal(err)
	}
	if migration.Migrated {
		t.Fatal("did not expect migration when target database exists")
	}

	got, err := os.ReadFile(paths.DBPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "existing" {
		t.Fatalf("existing database was overwritten: %q", string(got))
	}
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
