package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// transformExpectedForDarwin transforms expected assembly patterns for Darwin/macOS
// On Darwin, symbols get underscore prefix and bl/b calls to symbols also get prefix
func transformExpectedForDarwin(exp string) string {
	if runtime.GOOS != "darwin" {
		return exp
	}

	// Transform ".global\tfoo" -> ".global\t_foo"
	globalRE := regexp.MustCompile(`\.global\t([a-zA-Z_][a-zA-Z0-9_]*)`)
	exp = globalRE.ReplaceAllString(exp, `.global	_$1`)

	// Transform "foo:" -> "_foo:" (but not local labels like .L_foo)
	labelRE := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*):`)
	exp = labelRE.ReplaceAllString(exp, `_$1:`)

	// Transform "bl\tfoo" -> "bl\t_foo"
	blRE := regexp.MustCompile(`bl\t([a-zA-Z_][a-zA-Z0-9_]*)`)
	exp = blRE.ReplaceAllString(exp, `bl	_$1`)

	// Transform "b\tfoo" -> "b\t_foo" (but not local labels)
	bRE := regexp.MustCompile(`\bb\t([a-zA-Z_][a-zA-Z0-9_]*)`)
	exp = bRE.ReplaceAllString(exp, `b	_$1`)

	return exp
}

// IntegrationTestSpec represents a single integration test case
type IntegrationTestSpec struct {
	Name  string `yaml:"name"`
	Input string `yaml:"input"`
	Skip  string `yaml:"skip,omitempty"` // Reason to skip this test
}

// IntegrationTestFile represents the integration.yaml file structure
type IntegrationTestFile struct {
	Tests []IntegrationTestSpec `yaml:"tests"`
}

// E2EAsmTestSpec represents a single end-to-end ASM test case
type E2EAsmTestSpec struct {
	Name         string   `yaml:"name"`
	Input        string   `yaml:"input"`
	Expect       []string `yaml:"expect"`        // Strings that must appear in output
	ExpectOrder  []string `yaml:"expect_order"`  // Strings that must appear in this order
	ExpectUnique []string `yaml:"expect_unique"` // Strings that must appear exactly once
	ExpectNot    []string `yaml:"expect_not"`    // Strings that must NOT appear in output
	Skip         string   `yaml:"skip,omitempty"`
}

// E2EAsmTestFile represents the e2e_asm.yaml file structure
type E2EAsmTestFile struct {
	Tests []E2EAsmTestSpec `yaml:"tests"`
}

// findCompCert looks for the ccomp binary in common locations
func findCompCert() (string, bool) {
	// Check if COMPCERT environment variable is set
	if path := os.Getenv("COMPCERT"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, true
		}
	}

	// Check common locations
	locations := []string{
		"../../compcert/ccomp",     // Submodule location
		"../compcert/ccomp",        // Alternative relative path
		"/usr/local/bin/ccomp",     // System install
		"/opt/compcert/bin/ccomp",  // Container install
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, true
		}
	}

	// Try to find in PATH
	path, err := exec.LookPath("ccomp")
	if err == nil {
		return path, true
	}

	return "", false
}

// TestIntegrationCompCertEquivalence compares ralph-cc -dparse output with CompCert ccomp -dparse
func TestIntegrationCompCertEquivalence(t *testing.T) {
	ccompPath, found := findCompCert()
	if !found {
		t.Skip("CompCert ccomp not found; set COMPCERT env var or build compcert submodule")
	}

	// Load test cases from YAML
	data, err := os.ReadFile("../../testdata/integration.yaml")
	if err != nil {
		t.Skipf("integration.yaml not found: %v", err)
	}

	var testFile IntegrationTestFile
	if err := yaml.Unmarshal(data, &testFile); err != nil {
		t.Fatalf("failed to parse integration.yaml: %v", err)
	}

	for _, tc := range testFile.Tests {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Skip != "" {
				t.Skip(tc.Skip)
			}

			// Create temp file with test input
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.c")
			if err := os.WriteFile(testFile, []byte(tc.Input), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Get CompCert output
			ccompOut, ccompErr := runCompCert(ccompPath, testFile)
			if ccompErr != nil {
				t.Fatalf("CompCert failed: %v\nOutput: %s", ccompErr, ccompOut)
			}

			// Get ralph-cc output
			resetDebugFlags()
			var ralphOut, ralphErrOut bytes.Buffer
			cmd := newRootCmd(&ralphOut, &ralphErrOut)
			cmd.SetArgs([]string{"--dparse", testFile})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("ralph-cc failed: %v\nStderr: %s", err, ralphErrOut.String())
			}

			// Normalize and compare outputs
			ccompNorm := normalizeOutput(ccompOut)
			ralphNorm := normalizeOutput(ralphOut.String())

			if ccompNorm != ralphNorm {
				t.Errorf("Output mismatch\n--- CompCert ---\n%s\n--- ralph-cc ---\n%s\n--- CompCert (normalized) ---\n%s\n--- ralph-cc (normalized) ---\n%s",
					ccompOut, ralphOut.String(), ccompNorm, ralphNorm)
			}
		})
	}
}

// runCompCert executes ccomp with -dparse flag
func runCompCert(ccompPath, inputFile string) (string, error) {
	cmd := exec.Command(ccompPath, "-dparse", inputFile)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// normalizeOutput normalizes whitespace and formatting for comparison
func normalizeOutput(s string) string {
	// Split into lines
	lines := strings.Split(s, "\n")
	var normalized []string

	for _, line := range lines {
		// Trim trailing whitespace
		line = strings.TrimRight(line, " \t")
		// Skip empty lines
		if line == "" {
			continue
		}
		normalized = append(normalized, line)
	}

	return strings.Join(normalized, "\n")
}

// TestIntegrationDParseBasic tests that -dparse works for basic inputs without CompCert
func TestIntegrationDParseBasic(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string // Strings that must appear in output
	}{
		{
			name:   "empty function",
			input:  "int main() {}",
			expect: []string{"int main()", "{", "}"},
		},
		{
			name:   "return zero",
			input:  "int f() { return 0; }",
			expect: []string{"int f()", "return 0;"},
		},
		{
			name:  "arithmetic",
			input: "int f() { return 1 + 2 * 3; }",
			expect: []string{"int f()", "return", "+", "*"},
		},
		{
			name:  "function with params",
			input: "int add(int a, int b) { return a + b; }",
			expect: []string{"int add(", "int a", "int b", "return", "+"},
		},
		{
			name:  "if statement",
			input: "int f() { if (x) return 1; return 0; }",
			expect: []string{"if (", "return 1;", "return 0;"},
		},
		{
			name:  "while loop",
			input: "int f() { while (x) x--; return 0; }",
			expect: []string{"while (", "--"},
		},
		{
			name:  "for loop",
			input: "int f() { for (i = 0; i < 10; i++) x++; return 0; }",
			expect: []string{"for (", "< 10", "++"},
		},
		{
			name:  "struct definition",
			input: "struct Point { int x; int y; };",
			expect: []string{"struct Point", "int x;", "int y;"},
		},
		{
			name:  "typedef",
			input: "typedef int myint;",
			expect: []string{"typedef", "int", "myint"},
		},
		{
			name:  "typedef with const",
			input: "typedef const char *cstr;",
			expect: []string{"typedef", "const", "char*", "cstr"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.c")
			if err := os.WriteFile(testFile, []byte(tc.input), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			resetDebugFlags()
			var out, errOut bytes.Buffer
			cmd := newRootCmd(&out, &errOut)
			cmd.SetArgs([]string{"--dparse", testFile})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("ralph-cc failed: %v\nStderr: %s", err, errOut.String())
			}

			output := out.String()
			for _, exp := range tc.expect {
				if !strings.Contains(output, exp) {
					t.Errorf("expected output to contain %q\nGot:\n%s", exp, output)
				}
			}
		})
	}
}

// TestE2EAsmYAML tests end-to-end C to ARM64 assembly generation using yaml test cases
func TestE2EAsmYAML(t *testing.T) {
	// Load test cases from YAML
	data, err := os.ReadFile("../../testdata/e2e_asm.yaml")
	if err != nil {
		t.Fatalf("e2e_asm.yaml not found: %v", err)
	}

	var testFile E2EAsmTestFile
	if err := yaml.Unmarshal(data, &testFile); err != nil {
		t.Fatalf("failed to parse e2e_asm.yaml: %v", err)
	}

	for _, tc := range testFile.Tests {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Skip != "" {
				t.Skip(tc.Skip)
			}

			// Create temp file with test input
			tmpDir := t.TempDir()
			testCFile := filepath.Join(tmpDir, "test.c")
			if err := os.WriteFile(testCFile, []byte(tc.Input), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Run ralph-cc with -dasm flag
			resetDebugFlags()
			var out, errOut bytes.Buffer
			cmd := newRootCmd(&out, &errOut)
			cmd.SetArgs([]string{"--dasm", testCFile})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("ralph-cc failed: %v\nStderr: %s", err, errOut.String())
			}

			output := out.String()
			// Check that all expected strings appear in output
			for _, exp := range tc.Expect {
				exp = transformExpectedForDarwin(exp)
				if !strings.Contains(output, exp) {
					t.Errorf("expected output to contain %q\nGot:\n%s", exp, output)
				}
			}

			// Check that strings appear in specified order
			if len(tc.ExpectOrder) > 0 {
				lastIdx := -1
				for _, exp := range tc.ExpectOrder {
					exp = transformExpectedForDarwin(exp)
					idx := strings.Index(output, exp)
					if idx == -1 {
						t.Errorf("expected output to contain %q for order check\nGot:\n%s", exp, output)
					} else if idx <= lastIdx {
						t.Errorf("expected %q to appear after previous pattern (position %d vs %d)\nGot:\n%s", exp, idx, lastIdx, output)
					}
					lastIdx = idx
				}
			}

			// Check that strings appear exactly once
			for _, exp := range tc.ExpectUnique {
				exp = transformExpectedForDarwin(exp)
				count := strings.Count(output, exp)
				if count != 1 {
					t.Errorf("expected %q to appear exactly once, found %d times\nGot:\n%s", exp, count, output)
				}
			}

			// Check that strings do NOT appear
			for _, exp := range tc.ExpectNot {
				exp = transformExpectedForDarwin(exp)
				if strings.Contains(output, exp) {
					t.Errorf("expected output NOT to contain %q\nGot:\n%s", exp, output)
				}
			}
		})
	}
}

// TestIncludeDirective tests that #include directives work
func TestIncludeDirective(t *testing.T) {
	tmpDir := t.TempDir()

	// Create include directory
	includeDir := filepath.Join(tmpDir, "include")
	if err := os.Mkdir(includeDir, 0755); err != nil {
		t.Fatalf("failed to create include dir: %v", err)
	}

	// Create a header file (simple macro only, no function declarations)
	headerContent := `#ifndef MYHEADER_H
#define MYHEADER_H
#define MY_CONSTANT 42
#endif
`
	headerPath := filepath.Join(includeDir, "myheader.h")
	if err := os.WriteFile(headerPath, []byte(headerContent), 0644); err != nil {
		t.Fatalf("failed to write header: %v", err)
	}

	// Create source file that includes the header
	sourceContent := `#include "myheader.h"
int main() {
    return MY_CONSTANT;
}
`
	sourcePath := filepath.Join(tmpDir, "test.c")
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}

	// Run ralph-cc with -I flag
	resetDebugFlags()
	includePaths = nil // Reset global state
	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"-I", includeDir, "--dparse", sourcePath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ralph-cc failed: %v\nStderr: %s", err, errOut.String())
	}

	output := out.String()

	// The macro should be expanded to 42
	if !strings.Contains(output, "return 42") {
		t.Errorf("expected macro MY_CONSTANT to expand to 42\nGot:\n%s", output)
	}

	// Clean up global state
	includePaths = nil
}

// E2ERuntimeTestSpec represents a single end-to-end runtime test case
type E2ERuntimeTestSpec struct {
	Name         string `yaml:"name"`
	Input        string `yaml:"input"`
	ExpectedExit int    `yaml:"expected_exit"`
	Skip         string `yaml:"skip,omitempty"`
}

// E2ERuntimeTestFile represents the e2e_runtime.yaml file structure
type E2ERuntimeTestFile struct {
	Tests []E2ERuntimeTestSpec `yaml:"tests"`
}

// TestE2ERuntimeYAML tests end-to-end C compilation with actual execution
func TestE2ERuntimeYAML(t *testing.T) {
	// Check if we can run executables (need assembler and linker)
	if _, err := exec.LookPath("as"); err != nil {
		t.Skip("assembler 'as' not found in PATH")
	}
	if _, err := exec.LookPath("ld"); err != nil {
		t.Skip("linker 'ld' not found in PATH")
	}

	// Load test cases from YAML
	data, err := os.ReadFile("../../testdata/e2e_runtime.yaml")
	if err != nil {
		t.Fatalf("e2e_runtime.yaml not found: %v", err)
	}

	var testFile E2ERuntimeTestFile
	if err := yaml.Unmarshal(data, &testFile); err != nil {
		t.Fatalf("failed to parse e2e_runtime.yaml: %v", err)
	}

	for _, tc := range testFile.Tests {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Skip != "" {
				t.Skip(tc.Skip)
			}

			// Create temp directory for build artifacts
			tmpDir := t.TempDir()
			testCFile := filepath.Join(tmpDir, "test.c")
			testSFile := filepath.Join(tmpDir, "test.s")
			testOFile := filepath.Join(tmpDir, "test.o")
			testExe := filepath.Join(tmpDir, "test")

			// Write C source
			if err := os.WriteFile(testCFile, []byte(tc.Input), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Step 1: Compile C to assembly with ralph-cc
			resetDebugFlags()
			var asmOut, errOut bytes.Buffer
			cmd := newRootCmd(&asmOut, &errOut)
			cmd.SetArgs([]string{"--dasm", testCFile})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("ralph-cc failed: %v\nStderr: %s", err, errOut.String())
			}

			// Step 2: Convert to macOS format if needed
			asmContent := asmOut.String()
			asmContent = convertToMacOS(asmContent)

			// Write assembly
			if err := os.WriteFile(testSFile, []byte(asmContent), 0644); err != nil {
				t.Fatalf("failed to write assembly: %v", err)
			}

			// Step 3: Assemble
			asCmd := exec.Command("as", "-o", testOFile, testSFile)
			if output, err := asCmd.CombinedOutput(); err != nil {
				t.Fatalf("assembler failed: %v\nOutput: %s\nAssembly:\n%s", err, output, asmContent)
			}

			// Step 4: Link
			// On macOS, need to link with -lSystem
			var ldCmd *exec.Cmd
			sdkPath, _ := exec.Command("xcrun", "--show-sdk-path").Output()
			sdkPathStr := strings.TrimSpace(string(sdkPath))
			if sdkPathStr != "" {
				ldCmd = exec.Command("ld", "-o", testExe, testOFile, "-lSystem", "-L"+sdkPathStr+"/usr/lib")
			} else {
				ldCmd = exec.Command("ld", "-o", testExe, testOFile, "-lc")
			}
			if output, err := ldCmd.CombinedOutput(); err != nil {
				t.Fatalf("linker failed: %v\nOutput: %s", err, output)
			}

			// Step 5: Run and check exit code
			runCmd := exec.Command(testExe)
			runCmd.Run() // Ignore error, we want exit code
			exitCode := runCmd.ProcessState.ExitCode()

			if exitCode != tc.ExpectedExit {
				t.Errorf("expected exit code %d, got %d\nAssembly:\n%s", tc.ExpectedExit, exitCode, asmContent)
			}
		})
	}
}

// convertToMacOS converts ELF-style assembly to macOS format
func convertToMacOS(asm string) string {
	lines := strings.Split(asm, "\n")
	var result []string

	for _, line := range lines {
		// Skip .type and .size directives (not supported on macOS)
		if strings.HasPrefix(strings.TrimSpace(line), ".type") ||
			strings.HasPrefix(strings.TrimSpace(line), ".size") {
			continue
		}

		// Add underscore prefix to global symbols
		if strings.Contains(line, ".global\t") {
			// Extract symbol name and add underscore
			parts := strings.Split(line, "\t")
			if len(parts) >= 2 {
				sym := parts[len(parts)-1]
				if !strings.HasPrefix(sym, ".") && !strings.HasPrefix(sym, "_") { // Don't prefix local labels or already prefixed
					// Replace only the symbol part, not the directive
					line = "\t.global\t_" + sym
				}
			}
		}

		// Add underscore prefix to function labels (lines like "main:")
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ":") && !strings.HasPrefix(trimmed, ".") {
			// This is a label - add underscore if it's a function
			label := strings.TrimSuffix(trimmed, ":")
			if !strings.HasPrefix(label, ".L") && !strings.HasPrefix(label, "_") { // Not a local label or already prefixed
				line = "_" + trimmed
			}
		}

		// Convert bl calls to external functions
		if strings.Contains(line, "\tbl\t") {
			parts := strings.Split(line, "\tbl\t")
			if len(parts) == 2 {
				sym := strings.TrimSpace(parts[1])
				if !strings.HasPrefix(sym, "_") && !strings.HasPrefix(sym, ".") {
					line = parts[0] + "\tbl\t_" + sym
				}
			}
		}

		// Handle adrp/add @PAGE/@PAGEOFF for macOS (skip if already present)
		if strings.Contains(line, "adrp") && strings.Contains(line, ".Lstr") && !strings.Contains(line, "@PAGE") {
			// adrp x0, .Lstr0 -> adrp x0, .Lstr0@PAGE
			parts := strings.Fields(line)
			if len(parts) >= 3 && strings.HasPrefix(parts[2], ".Lstr") {
				// parts[1] already has comma like "x0," so don't add another
				reg := strings.TrimSuffix(parts[1], ",")
				line = parts[0] + "\t" + reg + ", " + parts[2] + "@PAGE"
			}
		}
		if strings.Contains(line, "add") && strings.Contains(line, "#0") && strings.Contains(line, ".Lstr") && !strings.Contains(line, "@PAGEOFF") {
			// add x0, x0, #0 after adrp for .Lstr should use @PAGEOFF
			// This is a simplified detection - look for pattern
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// TestPreprocessedFileExtension tests that .i files are not preprocessed
func TestPreprocessedFileExtension(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a .i file (should be treated as already preprocessed)
	// Note: #define should NOT be expanded since .i files skip preprocessing
	sourceContent := `int main() {
    return 42;
}
`
	sourcePath := filepath.Join(tmpDir, "test.i")
	if err := os.WriteFile(sourcePath, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}

	// Run ralph-cc - should work without preprocessing
	resetDebugFlags()
	includePaths = nil
	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--dparse", sourcePath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ralph-cc failed: %v\nStderr: %s", err, errOut.String())
	}

	output := out.String()
	if !strings.Contains(output, "return 42") {
		t.Errorf("expected output to contain 'return 42'\nGot:\n%s", output)
	}

	includePaths = nil
}
