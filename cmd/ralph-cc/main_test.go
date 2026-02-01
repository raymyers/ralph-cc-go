package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if version == "" {
		t.Error("version should not be empty")
	}
}

func TestDebugFlagsExist(t *testing.T) {
	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)

	expectedFlags := []string{"dparse", "dc", "dasm", "dclight", "dcsharpminor", "dcminor", "drtl", "dltl", "dmach"}
	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("expected flag --%s to exist", flagName)
		}
	}
}

func TestDebugFlagsWarnAndExit(t *testing.T) {
	// These flags are still unimplemented
	// Note: dclight, dcsharpminor, dcminor, drtl were removed as they're now implemented
	testCases := []struct {
		flagName string
		wantMsg  string
	}{
		{"dc", "dc"},
		{"dasm", "dasm"},
		{"dltl", "dltl"},
		{"dmach", "dmach"},
	}

	for _, tc := range testCases {
		t.Run(tc.flagName, func(t *testing.T) {
			// Reset all flags before each test
			resetDebugFlags()

			var out, errOut bytes.Buffer
			cmd := newRootCmd(&out, &errOut)
			cmd.SetArgs([]string{"--" + tc.flagName, "test.c"})
			err := cmd.Execute()

			// Should return an error
			if err == nil {
				t.Errorf("expected error for flag --%s, got nil", tc.flagName)
			}
			if !errors.Is(err, ErrNotImplemented) {
				t.Errorf("expected ErrNotImplemented, got %v", err)
			}

			output := errOut.String()
			if !strings.Contains(output, tc.wantMsg) {
				t.Errorf("expected output to contain %q, got %q", tc.wantMsg, output)
			}
			if !strings.Contains(output, "not yet implemented") {
				t.Errorf("expected output to contain 'not yet implemented', got %q", output)
			}
		})
	}
}

func TestNoDebugFlagsNoError(t *testing.T) {
	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"test.c"})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("expected no error without debug flags, got %v", err)
	}
}

func TestDParseFlag(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.c")
	content := `int main() { return 0; }`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--dparse", testFile})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("expected no error for -dparse, got %v", err)
	}

	output := out.String()
	// Check that it contains expected AST output
	if !strings.Contains(output, "int main()") {
		t.Errorf("expected output to contain 'int main()', got %q", output)
	}
	if !strings.Contains(output, "return 0") {
		t.Errorf("expected output to contain 'return 0', got %q", output)
	}
}

func TestDParseFlagMultipleFunctions(t *testing.T) {
	// Create a temporary test file with multiple functions
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "multi.c")
	content := `int add(int a, int b) { return a + b; }
int main() { return add(1, 2); }`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--dparse", testFile})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("expected no error for -dparse, got %v", err)
	}

	output := out.String()
	// Check that it contains both functions
	if !strings.Contains(output, "int add(") {
		t.Errorf("expected output to contain 'int add(', got %q", output)
	}
	if !strings.Contains(output, "int main()") {
		t.Errorf("expected output to contain 'int main()', got %q", output)
	}
}

func TestDClightFlag(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.c")
	content := `int main() {
	int x = 5;
	x = x + 1;
	return x;
}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--dclight", testFile})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("expected no error for -dclight, got %v", err)
	}

	output := out.String()
	// Check that it contains Clight function output
	if !strings.Contains(output, "int main()") {
		t.Errorf("expected output to contain 'int main()', got %q", output)
	}
	// Check for some Clight-specific output (temps or return)
	if !strings.Contains(output, "return") {
		t.Errorf("expected output to contain 'return', got %q", output)
	}
}

func TestDClightCreatesOutputFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.c")
	content := "int main() { return 0; }"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--dclight", testFile})
	err := cmd.Execute()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check that the .light.c file was created
	outputFile := filepath.Join(tmpDir, "test.light.c")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("expected output file %s to be created", outputFile)
	}
}

func TestClightOutputFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"test.c", "test.light.c"},
		{"path/to/file.c", "path/to/file.light.c"},
		{"noext", "noext.light.c"},
	}

	for _, tt := range tests {
		got := clightOutputFilename(tt.input)
		if got != tt.want {
			t.Errorf("clightOutputFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDCsharpminorFlag(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.c")
	content := `int add(int a, int b) { return a + b; }`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--dcsharpminor", testFile})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("expected no error for -dcsharpminor, got %v", err)
	}

	output := out.String()
	// Check that it contains Csharpminor function output
	if !strings.Contains(output, "int add(a, b)") {
		t.Errorf("expected output to contain 'int add(a, b)', got %q", output)
	}
	// Check for Csharpminor-specific output (typed add operation)
	if !strings.Contains(output, "add(") {
		t.Errorf("expected output to contain 'add(' (typed add operation), got %q", output)
	}
}

func TestDCsharpminorCreatesOutputFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.c")
	content := "int main() { return 0; }"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--dcsharpminor", testFile})
	err := cmd.Execute()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check that the .csharpminor file was created
	outputFile := filepath.Join(tmpDir, "test.csharpminor")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("expected output file %s to be created", outputFile)
	}
}

func TestCsharpminorOutputFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"test.c", "test.csharpminor"},
		{"path/to/file.c", "path/to/file.csharpminor"},
		{"noext", "noext.csharpminor"},
	}

	for _, tt := range tests {
		got := csharpminorOutputFilename(tt.input)
		if got != tt.want {
			t.Errorf("csharpminorOutputFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDCminorFlag(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.c")
	content := `int add(int a, int b) { return a + b; }`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--dcminor", testFile})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("expected no error for -dcminor, got %v", err)
	}

	output := out.String()
	// Check that it contains Cminor function output - quoted function name
	if !strings.Contains(output, `"add"(`) {
		t.Errorf("expected output to contain '\"add\"(', got %q", output)
	}
	// Check for Cminor-specific output (return statement)
	if !strings.Contains(output, "return") {
		t.Errorf("expected output to contain 'return', got %q", output)
	}
}

func TestDCminorCreatesOutputFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.c")
	content := "int main() { return 0; }"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--dcminor", testFile})
	err := cmd.Execute()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check that the .cminor file was created
	outputFile := filepath.Join(tmpDir, "test.cminor")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("expected output file %s to be created", outputFile)
	}
}

func TestCminorOutputFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"test.c", "test.cminor"},
		{"path/to/file.c", "path/to/file.cminor"},
		{"noext", "noext.cminor"},
	}

	for _, tt := range tests {
		got := cminorOutputFilename(tt.input)
		if got != tt.want {
			t.Errorf("cminorOutputFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDRTLFlag(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.c")
	content := `int add(int a, int b) { return a + b; }`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--drtl", testFile})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("expected no error for -drtl, got %v", err)
	}

	output := out.String()
	// Check that it contains RTL function output
	if !strings.Contains(output, "add(") {
		t.Errorf("expected output to contain 'add(', got %q", output)
	}
	// Check for RTL-specific output (entry point)
	if !strings.Contains(output, "entry:") {
		t.Errorf("expected output to contain 'entry:', got %q", output)
	}
}

func TestDRTLCreatesOutputFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.c")
	content := "int main() { return 0; }"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--drtl", testFile})
	err := cmd.Execute()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Check that the .rtl.0 file was created
	outputFile := filepath.Join(tmpDir, "test.rtl.0")
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Errorf("expected output file %s to be created", outputFile)
	}
}

func TestRTLOutputFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"test.c", "test.rtl.0"},
		{"path/to/file.c", "path/to/file.rtl.0"},
		{"noext", "noext.rtl.0"},
	}

	for _, tt := range tests {
		got := rtlOutputFilename(tt.input)
		if got != tt.want {
			t.Errorf("rtlOutputFilename(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDParseFlagFileNotFound(t *testing.T) {
	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--dparse", "nonexistent.c"})
	err := cmd.Execute()

	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestDParseCreatesOutputFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.c")
	content := `int main() { return 42; }`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	expectedOutputFile := filepath.Join(tmpDir, "test.parsed.c")

	resetDebugFlags()

	var out, errOut bytes.Buffer
	cmd := newRootCmd(&out, &errOut)
	cmd.SetArgs([]string{"--dparse", testFile})
	err := cmd.Execute()

	if err != nil {
		t.Errorf("expected no error for -dparse, got %v", err)
	}

	// Check that output file was created
	if _, err := os.Stat(expectedOutputFile); os.IsNotExist(err) {
		t.Errorf("expected output file %s to be created", expectedOutputFile)
	}

	// Check output file contents match stdout
	fileContent, err := os.ReadFile(expectedOutputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if out.String() != string(fileContent) {
		t.Errorf("output file content doesn't match stdout\nStdout:\n%s\nFile:\n%s", out.String(), string(fileContent))
	}

	// Verify content looks correct
	if !strings.Contains(string(fileContent), "int main()") {
		t.Errorf("expected output file to contain 'int main()'")
	}
	if !strings.Contains(string(fileContent), "return 42") {
		t.Errorf("expected output file to contain 'return 42'")
	}
}

func TestParsedOutputFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test.c", "test.parsed.c"},
		{"path/to/file.c", "path/to/file.parsed.c"},
		{"/absolute/path.c", "/absolute/path.parsed.c"},
		{"no_extension", "no_extension.parsed.c"},
		{"multiple.dots.c", "multiple.dots.parsed.c"},
	}

	for _, tc := range tests {
		result := parsedOutputFilename(tc.input)
		if result != tc.expected {
			t.Errorf("parsedOutputFilename(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func resetDebugFlags() {
	dParse = false
	dC = false
	dAsm = false
	dClight = false
	dCsharpminor = false
	dCminor = false
	dRTL = false
	dLTL = false
	dMach = false
}

func TestNormalizeFlags(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "single-dash dparse",
			input:    []string{"-dparse", "test.c"},
			expected: []string{"--dparse", "test.c"},
		},
		{
			name:     "double-dash dparse unchanged",
			input:    []string{"--dparse", "test.c"},
			expected: []string{"--dparse", "test.c"},
		},
		{
			name:     "single-dash dc",
			input:    []string{"-dc", "test.c"},
			expected: []string{"--dc", "test.c"},
		},
		{
			name:     "mixed flags",
			input:    []string{"test.c", "-dparse", "-dc"},
			expected: []string{"test.c", "--dparse", "--dc"},
		},
		{
			name:     "no flags",
			input:    []string{"test.c"},
			expected: []string{"test.c"},
		},
		{
			name:     "other flags unchanged",
			input:    []string{"-o", "output.o", "test.c"},
			expected: []string{"-o", "output.o", "test.c"},
		},
		{
			name:     "all debug flags",
			input:    []string{"-dparse", "-dc", "-dasm", "-dclight", "-dcsharpminor", "-dcminor", "-drtl", "-dltl", "-dmach"},
			expected: []string{"--dparse", "--dc", "--dasm", "--dclight", "--dcsharpminor", "--dcminor", "--drtl", "--dltl", "--dmach"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeFlags(tc.input)
			if len(result) != len(tc.expected) {
				t.Errorf("normalizeFlags(%v) = %v, want %v", tc.input, result, tc.expected)
				return
			}
			for i := range result {
				if result[i] != tc.expected[i] {
					t.Errorf("normalizeFlags(%v) = %v, want %v", tc.input, result, tc.expected)
					return
				}
			}
		})
	}
}
