package main

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestBuiltCLI(t *testing.T) {
	tempDir := t.TempDir()
	binaryName := "llmschema"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tempDir, binaryName)

	build := exec.Command("go", "build", "-ldflags", "-X main.version=1.2.3-test", "-o", binaryPath, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("failed to build CLI: %v\n%s", err, output)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open SQLite fixture: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY,
			email TEXT NOT NULL UNIQUE
		)
	`); err != nil {
		_ = db.Close()
		t.Fatalf("failed to create SQLite fixture: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("failed to close SQLite fixture: %v", err)
	}

	t.Run("reports linked version", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "--version")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI failed: %v\n%s", err, output)
		}
		if got, want := string(output), "llmschema version 1.2.3-test\n"; got != want {
			t.Fatalf("version output = %q, want %q", got, want)
		}
	})

	t.Run("extracts SQLite schema to stdout", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "--db-url", "sqlite://"+dbPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI failed: %v\n%s", err, output)
		}
		for _, want := range []string{"# Database Schema", "## users", "| email | TEXT NOT NULL UNIQUE |"} {
			if !strings.Contains(string(output), want) {
				t.Errorf("CLI output missing %q:\n%s", want, output)
			}
		}
	})

	t.Run("writes SQLite schema to file", func(t *testing.T) {
		outputPath := filepath.Join(tempDir, "schema.md")
		cmd := exec.Command(binaryPath, "--db-url", "sqlite://"+dbPath, "--output", outputPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("CLI failed: %v\n%s", err, output)
		}
		if len(output) != 0 {
			t.Errorf("CLI wrote unexpected output: %q", output)
		}
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("failed to read CLI output: %v", err)
		}
		if !strings.Contains(string(content), "## users") {
			t.Fatalf("file output missing users table:\n%s", content)
		}
	})

	t.Run("returns nonzero for invalid invocation", func(t *testing.T) {
		cmd := exec.Command(binaryPath)
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("CLI succeeded unexpectedly:\n%s", output)
		}
		if !strings.Contains(string(output), "--db-url is required") {
			t.Fatalf("CLI error missing database URL guidance:\n%s", output)
		}
	})
}
