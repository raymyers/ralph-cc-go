// preprocess.go implements the main preprocessor driver with include processing.
package cpp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Preprocessor is the main driver for C preprocessing.
type Preprocessor struct {
	macros       *MacroTable
	conditional  *ConditionalProcessor
	expander     *Expander
	resolver     *IncludeResolver
	opts         PreprocessorOptions
	includeGuards map[string]string // file path -> guard macro name
}

// PreprocessorOptions configures the preprocessor.
type PreprocessorOptions struct {
	Defines       []string // -D definitions
	Undefines     []string // -U undefinitions
	IncludePaths  []string // -I directories
	SystemPaths   []string // -isystem directories
	KeepComments  bool     // Preserve comments in output
	LineMarkers   bool     // Generate #line markers
}

// NewPreprocessor creates a new preprocessor instance.
func NewPreprocessor(opts PreprocessorOptions) *Preprocessor {
	macros := NewMacroTable()
	
	// Apply command line defines/undefines
	macros.ApplyCmdlineDefines(opts.Defines, opts.Undefines)
	
	resolver := NewIncludeResolver()
	for _, p := range opts.IncludePaths {
		resolver.AddUserPath(p)
	}
	for _, p := range opts.SystemPaths {
		resolver.AddSystemPath(p)
	}
	
	return &Preprocessor{
		macros:        macros,
		conditional:   NewConditionalProcessor(macros),
		expander:      NewExpander(macros),
		resolver:      resolver,
		opts:          opts,
		includeGuards: make(map[string]string),
	}
}

// PreprocessFile preprocesses a file and returns the result.
func (p *Preprocessor) PreprocessFile(filename string) (string, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		absPath = filename
	}
	
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", filename, err)
	}
	
	p.resolver.SetCurrentFile(absPath)
	if err := p.resolver.PushFile(absPath); err != nil {
		return "", err
	}
	defer p.resolver.PopFile()
	
	return p.preprocessContent(string(content), absPath)
}

// PreprocessString preprocesses a string with a given filename for error messages.
func (p *Preprocessor) PreprocessString(source, filename string) (string, error) {
	return p.preprocessContent(source, filename)
}

// preprocessContent is the main preprocessing loop.
func (p *Preprocessor) preprocessContent(source, filename string) (string, error) {
	lex := NewLexer(source, filename)
	var output strings.Builder
	var lineTokens []Token
	currentLine := 1
	
	if p.opts.LineMarkers {
		output.WriteString(fmt.Sprintf("# 1 \"%s\"\n", filename))
	}
	
	for {
		tok := lex.NextToken()
		
		if tok.Type == PP_EOF {
			// Process any remaining tokens on the line
			if len(lineTokens) > 0 {
				result, err := p.processLine(lineTokens, filename)
				if err != nil {
					return "", fmt.Errorf("%s:%d: %w", filename, currentLine, err)
				}
				output.WriteString(result)
			}
			break
		}
		
		if tok.Type == PP_NEWLINE {
			lineTokens = append(lineTokens, tok)
			result, err := p.processLine(lineTokens, filename)
			if err != nil {
				return "", fmt.Errorf("%s:%d: %w", filename, currentLine, err)
			}
			output.WriteString(result)
			lineTokens = nil
			currentLine = tok.Loc.Line + 1
			continue
		}
		
		lineTokens = append(lineTokens, tok)
	}
	
	// Check for unbalanced conditionals
	if err := p.conditional.CheckBalanced(); err != nil {
		return "", fmt.Errorf("%s: %w", filename, err)
	}
	
	return output.String(), nil
}

// processLine processes a single line of tokens.
func (p *Preprocessor) processLine(tokens []Token, filename string) (string, error) {
	if len(tokens) == 0 {
		return "", nil
	}
	
	// Check if line starts with # (directive)
	firstNonWS := 0
	for firstNonWS < len(tokens) && tokens[firstNonWS].Type == PP_WHITESPACE {
		firstNonWS++
	}
	
	if firstNonWS < len(tokens) && tokens[firstNonWS].Type == PP_HASH {
		return p.processDirective(tokens[firstNonWS:], filename)
	}
	
	// Regular line - only output if active
	if !p.conditional.IsActive() {
		return "", nil
	}
	
	// Expand macros
	expanded, err := p.expander.ExpandWithLoc(tokens, SourceLoc{File: filename, Line: tokens[0].Loc.Line})
	if err != nil {
		return "", err
	}
	
	return TokensToString(expanded), nil
}

// processDirective handles a preprocessing directive.
func (p *Preprocessor) processDirective(tokens []Token, filename string) (string, error) {
	if len(tokens) == 0 {
		return "", nil
	}
	
	// Get location from the # token
	loc := tokens[0].Loc
	
	// Parse the directive (skip the # token)
	var directiveTokens []Token
	for i := 1; i < len(tokens); i++ {
		directiveTokens = append(directiveTokens, tokens[i])
	}
	
	dir, err := ParseDirectiveFromTokens(directiveTokens, loc)
	if err != nil {
		// In inactive blocks, silently ignore unknown directives
		if !p.conditional.IsActive() {
			return "", nil
		}
		return "", err
	}
	
	// Handle conditional directives even in inactive blocks
	switch dir.Type {
	case DIR_IF:
		return "", p.conditional.ProcessIf(dir.Expression)
	case DIR_IFDEF:
		return "", p.conditional.ProcessIfdef(dir.Identifier)
	case DIR_IFNDEF:
		return "", p.conditional.ProcessIfndef(dir.Identifier)
	case DIR_ELIF:
		return "", p.conditional.ProcessElif(dir.Expression)
	case DIR_ELSE:
		return "", p.conditional.ProcessElse()
	case DIR_ENDIF:
		return "", p.conditional.ProcessEndif()
	}
	
	// Other directives are only processed in active blocks
	if !p.conditional.IsActive() {
		return "", nil
	}
	
	switch dir.Type {
	case DIR_INCLUDE:
		return p.processInclude(dir, filename)
	case DIR_DEFINE:
		return "", p.macros.DefineFromDirective(dir)
	case DIR_UNDEF:
		p.macros.Undefine(dir.Identifier)
		return "", nil
	case DIR_LINE:
		// Output the line directive
		if dir.FileName != "" {
			return fmt.Sprintf("# %d \"%s\"\n", dir.LineNum, dir.FileName), nil
		}
		return fmt.Sprintf("# %d\n", dir.LineNum), nil
	case DIR_LINEMARKER:
		// Pass through GCC line markers
		return TokensToString(tokens) + "\n", nil
	case DIR_ERROR:
		return "", fmt.Errorf("#error %s", dir.Message)
	case DIR_WARNING:
		// Warnings are typically printed to stderr and not fatal
		fmt.Fprintf(os.Stderr, "%s:%d: warning: %s\n", loc.File, loc.Line, dir.Message)
		return "", nil
	case DIR_PRAGMA:
		return p.processPragma(dir, filename)
	case DIR_EMPTY:
		return "", nil
	default:
		return "", fmt.Errorf("unhandled directive type: %v", dir.Type)
	}
}

// processInclude handles #include directives.
func (p *Preprocessor) processInclude(dir *Directive, currentFile string) (string, error) {
	// Determine the header name
	headerName := dir.HeaderName
	
	// If we have Expression tokens instead of HeaderName, expand them
	if headerName == "" && len(dir.Expression) > 0 {
		expanded, err := p.expander.Expand(dir.Expression)
		if err != nil {
			return "", fmt.Errorf("expanding include: %w", err)
		}
		headerName = strings.TrimSpace(TokensToString(expanded))
	}
	
	if headerName == "" {
		return "", fmt.Errorf("empty include file name")
	}
	
	// Parse the header name format
	var fileName string
	var kind IncludeKind
	
	if strings.HasPrefix(headerName, "<") && strings.HasSuffix(headerName, ">") {
		fileName = headerName[1 : len(headerName)-1]
		kind = IncludeAngled
	} else if strings.HasPrefix(headerName, "\"") && strings.HasSuffix(headerName, "\"") {
		fileName = headerName[1 : len(headerName)-1]
		kind = IncludeQuoted
	} else {
		// Assume quoted form for unquoted names (shouldn't normally happen)
		fileName = headerName
		kind = IncludeQuoted
	}
	
	// Resolve the include path
	p.resolver.SetCurrentFile(currentFile)
	includePath, err := p.resolver.Resolve(fileName, kind)
	if err != nil {
		return "", fmt.Errorf("#include %s: %w", headerName, err)
	}
	
	// Check for #pragma once
	if p.resolver.IsAlreadyIncluded(includePath) {
		return "", nil
	}
	
	// Check for include guards (optimization)
	if guardMacro, ok := p.includeGuards[includePath]; ok {
		if p.macros.IsDefined(guardMacro) {
			return "", nil
		}
	}
	
	// Check include depth
	if p.resolver.IncludeDepth() >= MaxIncludeDepth {
		return "", fmt.Errorf("#include nested too deeply")
	}
	
	// Push file onto stack
	if err := p.resolver.PushFile(includePath); err != nil {
		return "", err
	}
	defer p.resolver.PopFile()
	
	// Read the include file
	content, err := os.ReadFile(includePath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", includePath, err)
	}
	
	// Detect include guards
	guardMacro := p.detectIncludeGuard(string(content), includePath)
	if guardMacro != "" {
		p.includeGuards[includePath] = guardMacro
	}
	
	// Generate line marker for entering file
	var output strings.Builder
	if p.opts.LineMarkers {
		output.WriteString(fmt.Sprintf("# 1 \"%s\" 1\n", includePath))
	}
	
	// Recursively preprocess the included file
	oldCurrentFile := p.resolver.CurrentDir
	p.resolver.SetCurrentFile(includePath)
	
	result, err := p.preprocessContent(string(content), includePath)
	if err != nil {
		return "", fmt.Errorf("in %s: %w", includePath, err)
	}
	output.WriteString(result)
	
	p.resolver.CurrentDir = oldCurrentFile
	
	// Generate line marker for returning to original file
	if p.opts.LineMarkers {
		output.WriteString(fmt.Sprintf("# %d \"%s\" 2\n", dir.Loc.Line+1, currentFile))
	}
	
	return output.String(), nil
}

// detectIncludeGuard checks if a file has an include guard pattern.
// Returns the guard macro name if found, empty string otherwise.
func (p *Preprocessor) detectIncludeGuard(content, filename string) string {
	lex := NewLexer(content, filename)
	
	// Look for #ifndef or #if !defined pattern at start of file
	var tokens []Token
	for {
		tok := lex.NextToken()
		if tok.Type == PP_EOF {
			break
		}
		// Collect first few meaningful tokens
		if tok.Type != PP_WHITESPACE && tok.Type != PP_NEWLINE {
			tokens = append(tokens, tok)
		}
		if len(tokens) > 10 {
			break
		}
	}
	
	if len(tokens) < 3 {
		return ""
	}
	
	// Check for #ifndef GUARD pattern
	if tokens[0].Type == PP_HASH && tokens[1].Type == PP_IDENTIFIER && tokens[1].Text == "ifndef" {
		if tokens[2].Type == PP_IDENTIFIER {
			// Check if next directive is #define GUARD
			if len(tokens) >= 6 {
				if tokens[3].Type == PP_HASH && tokens[4].Type == PP_IDENTIFIER && tokens[4].Text == "define" {
					if tokens[5].Type == PP_IDENTIFIER && tokens[5].Text == tokens[2].Text {
						return tokens[2].Text
					}
				}
			}
		}
	}
	
	return ""
}

// processPragma handles #pragma directives.
func (p *Preprocessor) processPragma(dir *Directive, filename string) (string, error) {
	if len(dir.PragmaTokens) == 0 {
		return "", nil
	}
	
	// Check for #pragma once
	if dir.PragmaTokens[0].Type == PP_IDENTIFIER && dir.PragmaTokens[0].Text == "once" {
		p.resolver.MarkPragmaOnce(filename)
		return "", nil
	}
	
	// Pass through other pragmas
	var sb strings.Builder
	sb.WriteString("#pragma ")
	sb.WriteString(TokensToString(dir.PragmaTokens))
	sb.WriteString("\n")
	return sb.String(), nil
}

// GetMacros returns the macro table for inspection.
func (p *Preprocessor) GetMacros() *MacroTable {
	return p.macros
}

// SetLineMarkers enables or disables line marker output.
func (p *Preprocessor) SetLineMarkers(enabled bool) {
	p.opts.LineMarkers = enabled
}
