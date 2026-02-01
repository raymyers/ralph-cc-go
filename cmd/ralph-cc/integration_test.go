package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

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
	Name   string   `yaml:"name"`
	Input  string   `yaml:"input"`
	Expect []string `yaml:"expect"` // Strings that must appear in output
	Skip   string   `yaml:"skip,omitempty"`
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
				if !strings.Contains(output, exp) {
					t.Errorf("expected output to contain %q\nGot:\n%s", exp, output)
				}
			}
		})
	}
}
