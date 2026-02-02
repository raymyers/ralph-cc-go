// Include path handling for the C preprocessor.
package cpp

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// IncludeKind distinguishes between <file> and "file" includes.
type IncludeKind int

const (
	IncludeQuoted IncludeKind = iota // "file" form
	IncludeAngled                    // <file> form
)

// IncludeResolver handles include path resolution.
type IncludeResolver struct {
	UserPaths      []string        // -I directories
	SystemPaths    []string        // -isystem directories
	CurrentDir     string          // Directory of file currently being processed
	includeStack   []string        // Stack of included files for cycle detection
	includedOnce   map[string]bool // Files with #pragma once
	systemDetected bool            // Have we detected system paths?
}

// NewIncludeResolver creates a new include resolver.
func NewIncludeResolver() *IncludeResolver {
	return &IncludeResolver{
		UserPaths:    []string{},
		SystemPaths:  []string{},
		includedOnce: make(map[string]bool),
	}
}

// AddUserPath adds a -I include directory.
func (r *IncludeResolver) AddUserPath(path string) {
	r.UserPaths = append(r.UserPaths, path)
}

// AddSystemPath adds a -isystem include directory.
func (r *IncludeResolver) AddSystemPath(path string) {
	r.SystemPaths = append(r.SystemPaths, path)
}

// SetCurrentFile sets the current file being processed (for relative includes).
func (r *IncludeResolver) SetCurrentFile(filename string) {
	r.CurrentDir = filepath.Dir(filename)
}

// DetectSystemPaths attempts to detect system include paths.
func (r *IncludeResolver) DetectSystemPaths() {
	if r.systemDetected {
		return
	}
	r.systemDetected = true

	// Try to query the compiler for include paths
	paths := queryCompilerIncludePaths()
	if len(paths) > 0 {
		r.SystemPaths = append(r.SystemPaths, paths...)
		return
	}

	// Fall back to default paths
	r.SystemPaths = append(r.SystemPaths, getDefaultSystemPaths()...)
}

// Resolve attempts to find the include file.
// Returns the absolute path to the file, or an error if not found.
func (r *IncludeResolver) Resolve(filename string, kind IncludeKind) (string, error) {
	// Ensure system paths are detected
	r.DetectSystemPaths()

	var searchPaths []string

	if kind == IncludeQuoted {
		// For "file": current directory first, then -I paths, then system paths
		if r.CurrentDir != "" {
			searchPaths = append(searchPaths, r.CurrentDir)
		}
	}

	// Add -I paths
	searchPaths = append(searchPaths, r.UserPaths...)

	// Add system paths
	searchPaths = append(searchPaths, r.SystemPaths...)

	// Search for the file
	for _, dir := range searchPaths {
		fullPath := filepath.Join(dir, filename)
		if _, err := os.Stat(fullPath); err == nil {
			absPath, err := filepath.Abs(fullPath)
			if err != nil {
				absPath = fullPath
			}
			return absPath, nil
		}
	}

	return "", &IncludeError{Filename: filename, Kind: kind}
}

// PushFile marks a file as being included and pushes it onto the include stack.
// Returns an error if the file is already in the stack (circular include).
func (r *IncludeResolver) PushFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Check for circular includes
	for _, f := range r.includeStack {
		if f == absPath {
			return &CircularIncludeError{Path: absPath, Stack: r.includeStack}
		}
	}

	r.includeStack = append(r.includeStack, absPath)
	return nil
}

// PopFile removes the current file from the include stack.
func (r *IncludeResolver) PopFile() {
	if len(r.includeStack) > 0 {
		r.includeStack = r.includeStack[:len(r.includeStack)-1]
	}
}

// IncludeStack returns the current include stack for error messages.
func (r *IncludeResolver) IncludeStack() []string {
	return r.includeStack
}

// MarkPragmaOnce marks the current file as having #pragma once.
func (r *IncludeResolver) MarkPragmaOnce(path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	r.includedOnce[absPath] = true
}

// IsAlreadyIncluded returns true if the file has #pragma once and was already included.
func (r *IncludeResolver) IsAlreadyIncluded(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	return r.includedOnce[absPath]
}

// IncludeDepth returns the current include nesting depth.
func (r *IncludeResolver) IncludeDepth() int {
	return len(r.includeStack)
}

// MaxIncludeDepth is the maximum allowed include nesting.
const MaxIncludeDepth = 200

// IncludeError indicates that an include file was not found.
type IncludeError struct {
	Filename string
	Kind     IncludeKind
}

func (e *IncludeError) Error() string {
	kindStr := "quoted"
	if e.Kind == IncludeAngled {
		kindStr = "angled"
	}
	return "include file not found: " + e.Filename + " (" + kindStr + ")"
}

// CircularIncludeError indicates a circular include dependency.
type CircularIncludeError struct {
	Path  string
	Stack []string
}

func (e *CircularIncludeError) Error() string {
	var sb strings.Builder
	sb.WriteString("circular include detected: ")
	sb.WriteString(e.Path)
	sb.WriteString("\ninclude stack:\n")
	for i, f := range e.Stack {
		sb.WriteString("  ")
		for j := 0; j < i; j++ {
			sb.WriteString("  ")
		}
		sb.WriteString(filepath.Base(f))
		sb.WriteString("\n")
	}
	return sb.String()
}

// queryCompilerIncludePaths queries the system C compiler for include paths.
func queryCompilerIncludePaths() []string {
	// Try cc, gcc, clang in order
	compilers := []string{"cc", "gcc", "clang"}
	for _, compiler := range compilers {
		if path, err := exec.LookPath(compiler); err == nil {
			if paths := queryCompiler(path); len(paths) > 0 {
				return paths
			}
		}
	}
	return nil
}

func queryCompiler(compiler string) []string {
	cmd := exec.Command(compiler, "-v", "-E", "-x", "c", "-")
	cmd.Stdin = strings.NewReader("")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = nil

	_ = cmd.Run() // Ignore errors, we're parsing stderr

	return parseCompilerOutput(stderr.String())
}

func parseCompilerOutput(output string) []string {
	var paths []string
	inSearchList := false

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Look for the start of include path section
		if strings.Contains(line, "#include <...> search starts here:") ||
			strings.Contains(line, "#include \"...\" search starts here:") {
			inSearchList = true
			continue
		}

		// End of include path section
		if strings.Contains(line, "End of search list") {
			inSearchList = false
			continue
		}

		if inSearchList {
			// Path lines are indented with a space
			path := strings.TrimSpace(line)
			// Skip framework paths (macOS specific)
			if strings.HasSuffix(path, " (framework directory)") {
				continue
			}
			if path != "" && dirExists(path) {
				paths = append(paths, path)
			}
		}
	}

	return paths
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func getDefaultSystemPaths() []string {
	var paths []string

	switch runtime.GOOS {
	case "darwin":
		// macOS paths
		candidates := []string{
			"/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/usr/include",
			"/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk/usr/include",
			"/usr/local/include",
		}
		for _, p := range candidates {
			if dirExists(p) {
				paths = append(paths, p)
			}
		}

	case "linux":
		// Linux paths
		candidates := []string{
			"/usr/include",
			"/usr/local/include",
		}
		for _, p := range candidates {
			if dirExists(p) {
				paths = append(paths, p)
			}
		}
		// Add gcc include paths
		gccPaths := findGCCIncludePaths()
		paths = append(paths, gccPaths...)

	default:
		// Generic Unix paths
		candidates := []string{
			"/usr/include",
			"/usr/local/include",
		}
		for _, p := range candidates {
			if dirExists(p) {
				paths = append(paths, p)
			}
		}
	}

	return paths
}

func findGCCIncludePaths() []string {
	var paths []string

	// Look for gcc version directories
	gccBase := "/usr/lib/gcc"
	if !dirExists(gccBase) {
		return paths
	}

	// Walk looking for include directories
	_ = filepath.Walk(gccBase, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && info.Name() == "include" {
			paths = append(paths, path)
		}
		return nil
	})

	return paths
}
