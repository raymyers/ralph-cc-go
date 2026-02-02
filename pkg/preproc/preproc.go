// Package preproc handles C preprocessing.
// It provides both an internal preprocessor implementation and fallback
// to an external system preprocessor (cc -E).
package preproc

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/raymyers/ralph-cc/pkg/cpp"
)

// Options configures the preprocessing step
type Options struct {
	IncludePaths []string          // -I directories
	SystemPaths  []string          // -isystem directories
	Defines      map[string]string // -D macros (name -> value, empty string for simple define)
	Undefines    []string          // -U macros
	UseExternal  bool              // Force use of external preprocessor
	LineMarkers  bool              // Generate #line markers
}

// Preprocess runs the C preprocessor on the given source file and returns
// the preprocessed source code as a string.
// By default, it uses the internal preprocessor. Set UseExternal option
// to force use of the system preprocessor.
func Preprocess(filename string, opts *Options) (string, error) {
	if opts != nil && opts.UseExternal {
		return preprocessExternal(filename, opts)
	}
	return preprocessInternal(filename, opts)
}

// preprocessInternal uses our internal pkg/cpp preprocessor
func preprocessInternal(filename string, opts *Options) (string, error) {
	ppOpts := cpp.PreprocessorOptions{
		LineMarkers: opts != nil && opts.LineMarkers,
	}

	if opts != nil {
		ppOpts.IncludePaths = opts.IncludePaths
		ppOpts.SystemPaths = opts.SystemPaths
		ppOpts.Undefines = opts.Undefines

		// Convert defines map to slice format expected by cpp package
		for name, value := range opts.Defines {
			if value == "" {
				ppOpts.Defines = append(ppOpts.Defines, name)
			} else {
				ppOpts.Defines = append(ppOpts.Defines, name+"="+value)
			}
		}
	}

	pp := cpp.NewPreprocessor(ppOpts)
	return pp.PreprocessFile(filename)
}

// preprocessExternal uses the system C preprocessor (cc -E)
func preprocessExternal(filename string, opts *Options) (string, error) {
	// Build the command arguments
	args := []string{"-E"} // Preprocess only

	if opts != nil {
		// Add include paths
		for _, path := range opts.IncludePaths {
			args = append(args, "-I"+path)
		}
		// Add system include paths
		for _, path := range opts.SystemPaths {
			args = append(args, "-isystem", path)
		}
		// Add defines
		for name, value := range opts.Defines {
			if value == "" {
				args = append(args, "-D"+name)
			} else {
				args = append(args, "-D"+name+"="+value)
			}
		}
		// Add undefines
		for _, name := range opts.Undefines {
			args = append(args, "-U"+name)
		}
	}

	// Add the input file
	args = append(args, filename)

	// Find the preprocessor command
	cppCmd := findPreprocessor()
	if cppCmd == "" {
		return "", fmt.Errorf("no C preprocessor found (tried: cc, gcc, clang)")
	}

	cmd := exec.Command(cppCmd, args...)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set the working directory to the file's directory for relative includes
	cmd.Dir = filepath.Dir(filename)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("preprocessing failed: %v\n%s", err, stderr.String())
	}

	return stdout.String(), nil
}

// PreprocessString preprocesses C source code provided as a string.
// It writes the source to a temporary file, preprocesses it, then cleans up.
func PreprocessString(source, filename string, opts *Options) (string, error) {
	// Create a temporary file for the source
	tmpDir := os.TempDir()
	baseName := filepath.Base(filename)
	if baseName == "" {
		baseName = "source.c"
	}
	tmpFile := filepath.Join(tmpDir, "ralph-cc-"+baseName)

	if err := os.WriteFile(tmpFile, []byte(source), 0644); err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	return Preprocess(tmpFile, opts)
}

// NeedsPreprocessing returns true if the file might need preprocessing.
// Files ending in .i or .p are considered already preprocessed.
func NeedsPreprocessing(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	// .i and .p files are considered already preprocessed (CompCert convention)
	return ext != ".i" && ext != ".p"
}

// findPreprocessor searches for a C preprocessor on the system
func findPreprocessor() string {
	// Try common preprocessor commands
	candidates := []string{"cc", "gcc", "clang"}

	for _, cmd := range candidates {
		if path, err := exec.LookPath(cmd); err == nil {
			return path
		}
	}
	return ""
}
