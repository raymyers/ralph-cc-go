package cpp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIncludeResolver_Resolve_QuotedInCurrentDir(t *testing.T) {
	// Create a temp directory with a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.h")
	if err := os.WriteFile(testFile, []byte("// test"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIncludeResolver()
	r.SetCurrentFile(filepath.Join(tmpDir, "main.c"))

	path, err := r.Resolve("test.h", IncludeQuoted)
	if err != nil {
		t.Fatalf("expected to find test.h, got error: %v", err)
	}
	if filepath.Base(path) != "test.h" {
		t.Errorf("expected test.h, got %s", path)
	}
}

func TestIncludeResolver_Resolve_AngledNotInCurrentDir(t *testing.T) {
	// Create a temp directory with a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.h")
	if err := os.WriteFile(testFile, []byte("// test"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIncludeResolver()
	r.SetCurrentFile(filepath.Join(tmpDir, "main.c"))
	// Don't add tmpDir to system paths - angled includes shouldn't find it in current dir

	_, err := r.Resolve("test.h", IncludeAngled)
	if err == nil {
		t.Fatal("expected error for angled include not finding file in current dir")
	}
}

func TestIncludeResolver_Resolve_UserPath(t *testing.T) {
	// Create temp directories
	userIncDir := t.TempDir()
	testFile := filepath.Join(userIncDir, "myheader.h")
	if err := os.WriteFile(testFile, []byte("// user header"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIncludeResolver()
	r.AddUserPath(userIncDir)

	// Both quoted and angled should find it via -I path
	for _, kind := range []IncludeKind{IncludeQuoted, IncludeAngled} {
		path, err := r.Resolve("myheader.h", kind)
		if err != nil {
			t.Fatalf("kind %v: expected to find myheader.h, got error: %v", kind, err)
		}
		if filepath.Base(path) != "myheader.h" {
			t.Errorf("kind %v: expected myheader.h, got %s", kind, path)
		}
	}
}

func TestIncludeResolver_Resolve_SystemPath(t *testing.T) {
	// Create temp directory for system headers
	sysIncDir := t.TempDir()
	testFile := filepath.Join(sysIncDir, "sysheader.h")
	if err := os.WriteFile(testFile, []byte("// system header"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIncludeResolver()
	r.systemDetected = true // Skip auto-detection
	r.AddSystemPath(sysIncDir)

	path, err := r.Resolve("sysheader.h", IncludeAngled)
	if err != nil {
		t.Fatalf("expected to find sysheader.h, got error: %v", err)
	}
	if filepath.Base(path) != "sysheader.h" {
		t.Errorf("expected sysheader.h, got %s", path)
	}
}

func TestIncludeResolver_Resolve_SearchOrder(t *testing.T) {
	// Create multiple directories with same-named file
	currentDir := t.TempDir()
	userDir := t.TempDir()
	systemDir := t.TempDir()

	// Create test.h in each with different content (use file content to distinguish)
	if err := os.WriteFile(filepath.Join(currentDir, "test.h"), []byte("current"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "test.h"), []byte("user"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(systemDir, "test.h"), []byte("system"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIncludeResolver()
	r.systemDetected = true // Skip auto-detection
	r.SetCurrentFile(filepath.Join(currentDir, "main.c"))
	r.AddUserPath(userDir)
	r.AddSystemPath(systemDir)

	// Quoted form should find current directory first
	path, err := r.Resolve("test.h", IncludeQuoted)
	if err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(path)
	if string(content) != "current" {
		t.Errorf("quoted include should find current dir first, got %s", content)
	}

	// Angled form should find user path first (skips current dir)
	path, err = r.Resolve("test.h", IncludeAngled)
	if err != nil {
		t.Fatal(err)
	}
	content, _ = os.ReadFile(path)
	if string(content) != "user" {
		t.Errorf("angled include should find user path first, got %s", content)
	}
}

func TestIncludeResolver_CircularInclude(t *testing.T) {
	r := NewIncludeResolver()

	if err := r.PushFile("/a.h"); err != nil {
		t.Fatal(err)
	}
	if err := r.PushFile("/b.h"); err != nil {
		t.Fatal(err)
	}
	if err := r.PushFile("/c.h"); err != nil {
		t.Fatal(err)
	}

	// Now trying to include /a.h again should fail
	err := r.PushFile("/a.h")
	if err == nil {
		t.Fatal("expected circular include error")
	}
	if _, ok := err.(*CircularIncludeError); !ok {
		t.Errorf("expected *CircularIncludeError, got %T", err)
	}
}

func TestIncludeResolver_PragmaOnce(t *testing.T) {
	r := NewIncludeResolver()

	// First include is not marked
	if r.IsAlreadyIncluded("/test.h") {
		t.Error("file should not be marked as included yet")
	}

	// Mark as pragma once
	r.MarkPragmaOnce("/test.h")

	// Now should be marked
	if !r.IsAlreadyIncluded("/test.h") {
		t.Error("file should be marked as already included")
	}
}

func TestIncludeResolver_IncludeDepth(t *testing.T) {
	r := NewIncludeResolver()

	if r.IncludeDepth() != 0 {
		t.Error("initial depth should be 0")
	}

	r.PushFile("/a.h")
	if r.IncludeDepth() != 1 {
		t.Error("depth should be 1")
	}

	r.PushFile("/b.h")
	if r.IncludeDepth() != 2 {
		t.Error("depth should be 2")
	}

	r.PopFile()
	if r.IncludeDepth() != 1 {
		t.Error("depth should be 1 after pop")
	}

	r.PopFile()
	if r.IncludeDepth() != 0 {
		t.Error("depth should be 0 after pop")
	}
}

func TestIncludeResolver_DetectSystemPaths(t *testing.T) {
	r := NewIncludeResolver()
	r.DetectSystemPaths()

	// Should have detected at least some paths (unless on a very unusual system)
	if len(r.SystemPaths) == 0 {
		t.Log("Warning: no system paths detected, this may be expected in some environments")
	}

	// Should only detect once
	originalLen := len(r.SystemPaths)
	r.DetectSystemPaths()
	if len(r.SystemPaths) != originalLen {
		t.Error("DetectSystemPaths should only run once")
	}
}

func TestIncludeResolver_Resolve_NotFound(t *testing.T) {
	r := NewIncludeResolver()
	r.systemDetected = true // Skip auto-detection

	_, err := r.Resolve("nonexistent.h", IncludeQuoted)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}

	incErr, ok := err.(*IncludeError)
	if !ok {
		t.Fatalf("expected *IncludeError, got %T", err)
	}
	if incErr.Filename != "nonexistent.h" {
		t.Errorf("expected filename nonexistent.h, got %s", incErr.Filename)
	}
}

func TestIncludeResolver_Resolve_Subdirectory(t *testing.T) {
	// Create nested directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(subDir, "nested.h")
	if err := os.WriteFile(testFile, []byte("// nested"), 0644); err != nil {
		t.Fatal(err)
	}

	r := NewIncludeResolver()
	r.AddUserPath(tmpDir)

	path, err := r.Resolve("subdir/nested.h", IncludeQuoted)
	if err != nil {
		t.Fatalf("expected to find subdir/nested.h, got error: %v", err)
	}
	if filepath.Base(path) != "nested.h" {
		t.Errorf("expected nested.h, got %s", path)
	}
}

func TestParseCompilerOutput(t *testing.T) {
	// Create a temp directory to simulate paths that exist
	tmpDir := t.TempDir()
	existingPath1 := filepath.Join(tmpDir, "include1")
	existingPath2 := filepath.Join(tmpDir, "include2")
	os.MkdirAll(existingPath1, 0755)
	os.MkdirAll(existingPath2, 0755)

	// Test output similar to what gcc produces
	output := `Using built-in specs.
COLLECT_GCC=gcc
Target: aarch64-linux-gnu
#include "..." search starts here:
#include <...> search starts here:
 ` + existingPath1 + `
 ` + existingPath2 + `
 /nonexistent/path/that/should/be/filtered
End of search list.
`

	paths := parseCompilerOutput(output)

	// Should have found the two existing paths but not the nonexistent one
	if len(paths) != 2 {
		t.Errorf("expected 2 paths, got %d: %v", len(paths), paths)
	}

	// Check that the existing paths were found
	foundPath1, foundPath2 := false, false
	for _, p := range paths {
		if p == existingPath1 {
			foundPath1 = true
		}
		if p == existingPath2 {
			foundPath2 = true
		}
	}
	if !foundPath1 {
		t.Errorf("expected to find %s in paths", existingPath1)
	}
	if !foundPath2 {
		t.Errorf("expected to find %s in paths", existingPath2)
	}
}

func TestIncludeError(t *testing.T) {
	err := &IncludeError{Filename: "test.h", Kind: IncludeQuoted}
	msg := err.Error()
	if !contains(msg, "test.h") {
		t.Errorf("error message should contain filename: %s", msg)
	}
	if !contains(msg, "quoted") {
		t.Errorf("error message should contain kind: %s", msg)
	}

	err2 := &IncludeError{Filename: "sys.h", Kind: IncludeAngled}
	msg2 := err2.Error()
	if !contains(msg2, "angled") {
		t.Errorf("error message should contain kind: %s", msg2)
	}
}

func TestCircularIncludeError(t *testing.T) {
	err := &CircularIncludeError{
		Path:  "/c.h",
		Stack: []string{"/a.h", "/b.h"},
	}
	msg := err.Error()
	if !contains(msg, "circular") {
		t.Errorf("error message should mention circular: %s", msg)
	}
	if !contains(msg, "c.h") {
		t.Errorf("error message should contain the path: %s", msg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
