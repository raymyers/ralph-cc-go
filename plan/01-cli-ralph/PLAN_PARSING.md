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
- [ ] Compound: `+=`, `-=`, `*=`, `/=`, `%=`, `&=`, `|=`, `^=`, `<<=`, `>>=` (add to lexer)
- [x] Add tests for `x = 1`, `x += 2`

### M2.7: Other Expressions
- [ ] Comma operator
- [x] Conditional (ternary): `?:`
- [ ] Prefix increment/decrement: `++x`, `--x` (add to lexer)
- [ ] Postfix increment/decrement: `x++`, `x--`
- [x] Add tests for ternary, increment/decrement

### M2.8: Function Calls
- [x] Add Call AST node
- [x] Parse function call: `f()`, `f(a, b)`
- [x] Add tests for calls

### M2.9: Array/Member Access
- [x] Add Index AST node (array subscript)
- [ ] Add Member AST node (`.` and `->`)
- [x] Parse `a[i]`, `s.x`, `p->y`
- [x] Add tests for access expressions

### M2.10: Address/Dereference
- [ ] Unary `&` (address-of)
- [ ] Unary `*` (dereference)
- [ ] Add tests for `&x`, `*p`

### M2.11: Sizeof
- [ ] Parse `sizeof expr` and `sizeof(type)` (add to lexer)
- [ ] Add Sizeof AST node
- [ ] Add tests for sizeof

### M2.12: Cast Expressions
- [ ] Parse `(type)expr`
- [ ] Add Cast AST node
- [ ] Add tests for casts

---

## M3: Statements

### M3.1: Expression Statements
- [ ] Parse expression followed by `;`
- [ ] Add Computation (ExprStmt) AST node
- [ ] Add tests for `f(); x = 1;`

### M3.2: If/Else Statements
- [ ] Add If AST node
- [ ] Parse `if (cond) stmt`
- [ ] Parse `if (cond) stmt else stmt`
- [ ] Add tests including dangling else

### M3.3: While/Do-While Loops
- [ ] Add While, DoWhile AST nodes
- [ ] Parse `while (cond) stmt`
- [ ] Parse `do stmt while (cond);` (add `do` to lexer)
- [ ] Add tests for loops

### M3.4: For Loops
- [ ] Add For AST node
- [ ] Parse `for (init; cond; step) stmt`
- [ ] Handle optional parts (e.g., `for (;;)`)
- [ ] Add tests for for loops

### M3.5: Switch Statements
- [ ] Add Switch, Case, Default AST nodes
- [ ] Parse `switch (expr) { case x: ... default: ... }` (add keywords to lexer)
- [ ] Add tests for switch

### M3.6: Break/Continue
- [ ] Add Break, Continue AST nodes
- [ ] Parse `break;`, `continue;` (add keywords to lexer)
- [ ] Add tests

### M3.7: Goto/Labels
- [ ] Add Goto, Label AST nodes
- [ ] Parse `goto label;`, `label:`
- [ ] Add tests for goto/labels

---

## M4: Declarations

### M4.1: Variable Declarations
- [ ] Add DecDef (declaration definition) AST node
- [ ] Parse `int x;`, `int x, y;`
- [ ] Parse with initializers: `int x = 1;`
- [ ] Add tests for variable declarations

### M4.2: Function Parameters
- [ ] Parse function parameters: `int f(int a, int b)`
- [ ] Update FunDef AST to include parameter list
- [ ] Add tests for functions with parameters

### M4.3: Typedef Tracking
- [ ] Track typedef names in parser state
- [ ] Parse `typedef int myint;`
- [ ] Resolve ambiguity: `T * x;` as declaration vs multiplication
- [ ] Add tests for typedef

### M4.4: Storage Class Specifiers
- [ ] Parse `static`, `extern`, `auto`, `register` (add keywords)
- [ ] Add to AST specifier list
- [ ] Add tests

### M4.5: Type Qualifiers
- [ ] Parse `const`, `volatile`, `restrict` (add keywords)
- [ ] Add to AST
- [ ] Add tests

---

## M5: Types

### M5.1: Pointer Types
- [ ] Parse pointer declarators: `int *p;`
- [ ] Multiple indirection: `int **pp;`
- [ ] Pointer to function syntax
- [ ] Add tests

### M5.2: Array Types
- [ ] Parse array declarators: `int a[10];`
- [ ] Multi-dimensional: `int a[2][3];`
- [ ] Variable-length arrays (VLA)
- [ ] Add tests

### M5.3: Struct Types
- [ ] Parse `struct name { members };`
- [ ] Anonymous structs
- [ ] Add Struct AST node
- [ ] Add tests

### M5.4: Union Types
- [ ] Parse `union name { members };` (add keyword)
- [ ] Add Union AST node
- [ ] Add tests

### M5.5: Enum Types
- [ ] Parse `enum name { a, b = 1, c };` (add keyword)
- [ ] Add Enum AST node
- [ ] Add tests

### M5.6: Function Pointer Types
- [ ] Parse `int (*fp)(int, int);`
- [ ] Complex declarators
- [ ] Add tests

---

## M6: Full Grammar

### M6.1: Translation Unit
- [ ] Parse multiple top-level definitions
- [ ] Handle forward declarations
- [ ] Add tests for multi-function files

### M6.2: Preprocessor Artifacts
- [ ] Handle `#line` directives for source locations
- [ ] (Preprocessing itself done externally)

### M6.3: CLI Integration
- [ ] Wire parser to `-dparse` flag
- [ ] Output AST in format matching CompCert's `-dparse`
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
