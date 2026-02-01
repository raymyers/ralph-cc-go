# Parser Implementation Plan

Detailed task list to implement the parsing plan from docs/PARSING.md. The goal is full C parsing with `-dparse` output equivalent to CompCert.

## Current State
- [x] Lexer: Basic tokens (keywords, operators, identifiers, integers)
- [x] Parser: Minimal function definitions with `return <int>` statements
- [x] Tests: YAML-driven tests in `testdata/parse.yaml`

---

## M1: Minimal Parser (Complete)
- [x] Parse `int main() {}` - empty function
- [x] Parse `int f() { return 0; }` - return with integer literal
- [x] YAML test framework for AST verification

---

## M2: Expressions

### M2.1: Pratt Parser Infrastructure
- [x] Add precedence constants for all C operators
- [x] Implement Pratt parser skeleton with `parseExpr(precedence int)`
- [x] Add prefix/infix parsing function maps

### M2.2: Literals and Primary Expressions
- [x] Parse identifier expressions (Variable node)
- [x] Parse parenthesized expressions
- [x] Add tests for `(42)`, `x`, `(x)`

### M2.3: Arithmetic Operators
- [x] Binary: `+`, `-`, `*`, `/`, `%`
- [x] Unary prefix: `-` (negation)
- [x] Add tests for `1+2`, `3*4+5`, `-x`

### M2.4: Comparison and Logical Operators
- [x] Comparison: `<`, `<=`, `>`, `>=`, `==`, `!=`
- [x] Logical: `&&`, `||`, `!`
- [x] Add tests for `a < b && c > d`, `!x`

### M2.5: Bitwise Operators
- [x] Binary: `&`, `|`, `^`
- [x] Unary: `~`
- [x] Shift: `<<`, `>>` (add to lexer)
- [x] Add tests for `a & b | c`, `~x`

### M2.6: Assignment Operators
- [x] Simple: `=`
- [x] Compound: `+=`, `-=`, `*=`, `/=`, `%=`, `&=`, `|=`, `^=`, `<<=`, `>>=` (add to lexer)
- [x] Add tests for `x = 1`, `x += 2`

### M2.7: Other Expressions
- [x] Comma operator
- [x] Conditional (ternary): `?:`
- [x] Prefix increment/decrement: `++x`, `--x` (add to lexer)
- [x] Postfix increment/decrement: `x++`, `x--`
- [x] Add tests for ternary, increment/decrement

### M2.8: Function Calls
- [x] Add Call AST node
- [x] Parse function call: `f()`, `f(a, b)`
- [x] Add tests for calls

### M2.9: Array/Member Access
- [x] Add Index AST node (array subscript)
- [x] Add Member AST node (`.` and `->`)
- [x] Parse `a[i]`, `s.x`, `p->y`
- [x] Add tests for access expressions

### M2.10: Address/Dereference
- [x] Unary `&` (address-of)
- [x] Unary `*` (dereference)
- [x] Add tests for `&x`, `*p`

### M2.11: Sizeof
- [x] Parse `sizeof expr` and `sizeof(type)` (add to lexer)
- [x] Add Sizeof AST node
- [x] Add tests for sizeof

### M2.12: Cast Expressions
- [x] Parse `(type)expr`
- [x] Add Cast AST node
- [x] Add tests for casts

---

## M3: Statements

### M3.1: Expression Statements
- [x] Parse expression followed by `;`
- [x] Add Computation (ExprStmt) AST node
- [x] Add tests for `f(); x = 1;`

### M3.2: If/Else Statements
- [x] Add If AST node
- [x] Parse `if (cond) stmt`
- [x] Parse `if (cond) stmt else stmt`
- [x] Add tests including dangling else

### M3.3: While/Do-While Loops
- [x] Add While, DoWhile AST nodes
- [x] Parse `while (cond) stmt`
- [x] Parse `do stmt while (cond);` (add `do` to lexer)
- [x] Add tests for loops

### M3.4: For Loops
- [x] Add For AST node
- [x] Parse `for (init; cond; step) stmt`
- [x] Handle optional parts (e.g., `for (;;)`)
- [x] Add tests for for loops

### M3.5: Switch Statements
- [x] Add Switch, Case, Default AST nodes
- [x] Parse `switch (expr) { case x: ... default: ... }` (add keywords to lexer)
- [x] Add tests for switch

### M3.6: Break/Continue
- [x] Add Break, Continue AST nodes
- [x] Parse `break;`, `continue;` (add keywords to lexer)
- [x] Add tests

### M3.7: Goto/Labels
- [x] Add Goto, Label AST nodes
- [x] Parse `goto label;`, `label:`
- [x] Add tests for goto/labels

---

## M4: Declarations

### M4.1: Variable Declarations
- [x] Add DecDef (declaration definition) AST node
- [x] Parse `int x;`, `int x, y;`
- [x] Parse with initializers: `int x = 1;`
- [x] Add tests for variable declarations

### M4.2: Function Parameters
- [x] Parse function parameters: `int f(int a, int b)`
- [x] Update FunDef AST to include parameter list
- [x] Add tests for functions with parameters

### M4.3: Typedef Tracking
- [x] Track typedef names in parser state
- [x] Parse `typedef int myint;`
- [x] Resolve ambiguity: `T * x;` as declaration vs multiplication
- [x] Add tests for typedef

### M4.4: Storage Class Specifiers
- [x] Parse `static`, `extern`, `auto`, `register` (add keywords)
- [x] Add to AST specifier list
- [x] Add tests

### M4.5: Type Qualifiers
- [x] Parse `const`, `volatile`, `restrict` (add keywords)
- [x] Add to AST
- [x] Add tests

---

## M5: Types

### M5.1: Pointer Types
- [x] Parse pointer declarators: `int *p;`
- [x] Multiple indirection: `int **pp;`
- [ ] Pointer to function syntax
- [x] Add tests

### M5.2: Array Types
- [x] Parse array declarators: `int a[10];`
- [ ] Multi-dimensional: `int a[2][3];`
- [ ] Variable-length arrays (VLA)
- [x] Add tests

### M5.3: Struct Types
- [x] Parse `struct name { members };`
- [x] Anonymous structs
- [x] Add Struct AST node
- [x] Add tests

### M5.4: Union Types
- [x] Parse `union name { members };` (add keyword)
- [x] Add Union AST node
- [x] Add tests

### M5.5: Enum Types
- [x] Parse `enum name { a, b = 1, c };` (add keyword)
- [x] Add Enum AST node
- [x] Add tests

### M5.6: Function Pointer Types
- [ ] Parse `int (*fp)(int, int);`
- [ ] Complex declarators
- [ ] Add tests

---

## M6: Full Grammar

### M6.1: Translation Unit
- [x] Parse multiple top-level definitions
- [x] Handle forward declarations
- [x] Add tests for multi-function files

### M6.2: Preprocessor Artifacts
- [ ] Handle `#line` directives for source locations
- [ ] (Preprocessing itself done externally)

### M6.3: CLI Integration
- [x] Wire parser to `-dparse` flag
- [x] Output AST in format matching CompCert's `-dparse`
- [ ] Add integration tests comparing with CompCert output

### M6.4: Error Recovery
- [ ] Implement panic-mode recovery
- [ ] Continue parsing after errors
- [ ] Report multiple errors per file

---

## Equivalence Testing

### E1: Test Infrastructure
- [ ] Script to run CompCert `-dparse` on test files
- [ ] Comparison tool for AST output
- [ ] Add to CI

### E2: Equivalence Test Suite
- [ ] Expressions equivalence tests
- [ ] Statements equivalence tests  
- [ ] Declarations equivalence tests
- [ ] Full programs equivalence tests
