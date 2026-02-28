package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSQLiteDBPath_UsesExplicitEnv(t *testing.T) {
	t.Setenv("DB_PATH", filepath.Join("tmp", "custom.db"))
	t.Setenv("SQLITE_DB_PATH", "")

	got, err := resolveSQLiteDBPath()
	if err != nil {
		t.Fatalf("resolveSQLiteDBPath error: %v", err)
	}

	expect, _ := filepath.Abs(filepath.Join("tmp", "custom.db"))
	assertSamePath(t, got, expect)
}

func TestResolveSQLiteDBPath_PrefersTrackcardServerWhenOutside(t *testing.T) {
	t.Setenv("DB_PATH", "")
	t.Setenv("SQLITE_DB_PATH", "")

	wd, _ := os.Getwd()
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir failed: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()

	serverDir := filepath.Join(tmpDir, "trackcard-server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatalf("mkdir trackcard-server failed: %v", err)
	}

	serverDB := filepath.Join(serverDir, "trackcard.db")
	rootDB := filepath.Join(tmpDir, "trackcard.db")
	if err := os.WriteFile(serverDB, []byte("server"), 0o644); err != nil {
		t.Fatalf("write server db failed: %v", err)
	}
	if err := os.WriteFile(rootDB, []byte("root"), 0o644); err != nil {
		t.Fatalf("write root db failed: %v", err)
	}

	got, err := resolveSQLiteDBPath()
	if err != nil {
		t.Fatalf("resolveSQLiteDBPath error: %v", err)
	}

	expect, _ := filepath.Abs(serverDB)
	assertSamePath(t, got, expect)
}

func TestResolveSQLiteDBPath_UsesCwdWhenInsideTrackcardServer(t *testing.T) {
	t.Setenv("DB_PATH", "")
	t.Setenv("SQLITE_DB_PATH", "")

	wd, _ := os.Getwd()
	tmpDir := t.TempDir()
	serverDir := filepath.Join(tmpDir, "trackcard-server")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatalf("mkdir trackcard-server failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "trackcard.db"), []byte("server"), 0o644); err != nil {
		t.Fatalf("write server db failed: %v", err)
	}

	if err := os.Chdir(serverDir); err != nil {
		t.Fatalf("chdir server dir failed: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()

	got, err := resolveSQLiteDBPath()
	if err != nil {
		t.Fatalf("resolveSQLiteDBPath error: %v", err)
	}

	expect, _ := filepath.Abs(filepath.Join(serverDir, "trackcard.db"))
	assertSamePath(t, got, expect)
}

func assertSamePath(t *testing.T, got, expect string) {
	t.Helper()

	gotInfo, gotErr := os.Stat(got)
	expectInfo, expectErr := os.Stat(expect)
	if gotErr == nil && expectErr == nil {
		if !os.SameFile(gotInfo, expectInfo) {
			t.Fatalf("unexpected file target: got=%s want=%s", got, expect)
		}
		return
	}

	if filepath.Clean(got) != filepath.Clean(expect) {
		t.Fatalf("unexpected path: got=%s want=%s", got, expect)
	}
}
