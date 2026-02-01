# Parser Plan for Ralph-CC

This document outlines the parsing strategy for the Go CLI, designed to produce output equivalent to CompCert's Cabs AST.

## CompCert Parser Architecture

CompCert uses a sophisticated two-stage parsing approach:

### Stage 1: Pre-Parser (`pre_parser.mly`)
- Menhir grammar handling the **lexer hack** (typedef vs identifier disambiguation)
- Tracks typedef names in scope during parsing
- Classifies identifiers as `VAR_NAME`, `TYPEDEF_NAME`, or `OTHER_NAME`
- Emits two tokens per identifier: `PRE_NAME` followed by classified token

### Stage 2: Main Parser (`Parser.vy`)
- Coq grammar (extracted to OCaml) 
- Builds the Cabs abstract syntax tree
- Receives pre-classified tokens

### The Lexer Hack

C's grammar is not context-free due to typedef:
```c
T * x;  // Pointer declaration if T is a typedef, multiplication if T is a variable
```

CompCert solves this by tracking typedef names during parsing and reclassifying identifiers on-the-fly.

## Recommended Approach: Hand-Written Recursive Descent

### Rationale

| Option | Pros | Cons |
|--------|------|------|
| Parser Generator (goyacc, participle) | Grammar-based, similar to Menhir | Lexer hack awkward, rigid |
| Hand-Written Recursive Descent | Full context control, clean lexer hack, good errors | More code |
| PEG Parser (pigeon) | No left-recursion issues | Different semantics, backtracking |

**Chosen: Hand-written recursive descent** because:
1. Full control over typedef context tracking
2. Excellent error messages possible
3. Easy to debug and maintain
4. Natural fit for C's operator precedence (Pratt parsing)
5. Can match CompCert's output precisely

### Parser Components

```
┌─────────────────────────────────────────────────────────┐
│                      Parser                              │
├──────────────┬──────────────┬───────────────────────────┤
│   Lexer      │  Parser      │  AST                      │
│              │              │                           │
│  - Tokens    │  - Recursive │  - CabsExpr              │
│  - Position  │    descent   │  - CabsStmt              │
│  - No hack   │  - Typedef   │  - CabsDecl              │
│              │    tracking  │  - CabsType              │
│              │  - Pratt for │                           │
│              │    expr      │                           │
└──────────────┴──────────────┴───────────────────────────┘
```

### Module Structure

```
pkg/
├── cabs/           # AST definitions (mirrors Cabs.v)
│   ├── expr.go     # Expression nodes
│   ├── stmt.go     # Statement nodes
│   ├── decl.go     # Declaration nodes
│   └── types.go    # Type specifiers, declarators
├── lexer/          # Context-free tokenizer
│   ├── lexer.go
│   ├── token.go
│   └── lexer_test.go
└── parser/         # Recursive descent parser
    ├── parser.go   # Main entry, typedef tracking
    ├── expr.go     # Expression parsing (Pratt)
    ├── stmt.go     # Statement parsing
    ├── decl.go     # Declaration parsing
    └── parser_test.go
```

## Implementation Strategy

### Phase 1: Lexer

Simple context-free tokenizer. No typedef/identifier distinction.

**Tokens**:
- Keywords: `int`, `void`, `if`, `while`, `return`, `typedef`, `struct`, ...
- Operators: `+`, `-`, `*`, `/`, `==`, `!=`, `&&`, `||`, `++`, `--`, ...
- Delimiters: `(`, `)`, `{`, `}`, `[`, `]`, `;`, `,`
- Literals: integers, floats, chars, strings
- Identifiers: `[a-zA-Z_][a-zA-Z0-9_]*`

### Phase 2: Expression Parser (Pratt Parsing)

Pratt parsing handles operator precedence elegantly:

```go
func (p *Parser) parseExpr(precedence int) CabsExpr {
    left := p.parsePrefix()  // unary, literals, parens
    
    for precedence < p.currentPrecedence() {
        left = p.parseInfix(left)  // binary ops, postfix
    }
    return left
}
```

**Precedence levels** (lowest to highest):
1. Comma (`,`)
2. Assignment (`=`, `+=`, etc.)
3. Conditional (`?:`)
4. Logical OR (`||`)
5. Logical AND (`&&`)
6. Bitwise OR (`|`)
7. Bitwise XOR (`^`)
8. Bitwise AND (`&`)
9. Equality (`==`, `!=`)
10. Relational (`<`, `<=`, `>`, `>=`)
11. Shift (`<<`, `>>`)
12. Additive (`+`, `-`)
13. Multiplicative (`*`, `/`, `%`)
14. Unary (`!`, `~`, `-`, `*`, `&`, `++`, `--`, `sizeof`)
15. Postfix (`[]`, `()`, `.`, `->`, `++`, `--`)

### Phase 3: Statement Parser

Straightforward recursive descent:

```go
func (p *Parser) parseStmt() CabsStmt {
    switch p.current().Type {
    case TokenIf:
        return p.parseIfStmt()
    case TokenWhile:
        return p.parseWhileStmt()
    case TokenFor:
        return p.parseForStmt()
    case TokenReturn:
        return p.parseReturnStmt()
    case TokenLBrace:
        return p.parseBlock()
    // ...
    default:
        return p.parseExprStmt()
    }
}
```

### Phase 4: Declaration Parser with Typedef Tracking

The parser maintains a set of typedef names:

```go
type Parser struct {
    lexer    *Lexer
    typedefs map[string]bool  // Names introduced by typedef
    // ...
}

func (p *Parser) isTypeName(name string) bool {
    // Check builtins, then user typedefs
    return isBuiltinType(name) || p.typedefs[name]
}
```

When parsing declarations:
1. Parse specifiers (storage class, type specifiers)
2. If `typedef` seen, remember it
3. Parse declarator to get the name
4. If this was a typedef declaration, add name to `typedefs`

This resolves the ambiguity:
```go
func (p *Parser) parseStmtOrDecl() CabsNode {
    // Look ahead: is this a type specifier or expression?
    if p.startsDeclaration() {
        return p.parseDeclaration()
    }
    return p.parseStmt()
}

func (p *Parser) startsDeclaration() bool {
    tok := p.current()
    if isTypeKeyword(tok) {
        return true
    }
    if tok.Type == TokenIdent && p.isTypeName(tok.Value) {
        return true
    }
    return false
}
```

## Cabs AST Structure (Go)

Based on CompCert's `Cabs.v`:

### Expressions
```go
type CabsExpr interface{ implCabsExpr() }

type Unary struct {
    Op   UnaryOp
    Expr CabsExpr
}
type Binary struct {
    Op    BinaryOp
    Left  CabsExpr
    Right CabsExpr
}
type Variable struct {
    Name string
}
type Constant struct {
    Value CabsConstant
}
type Call struct {
    Func CabsExpr
    Args []CabsExpr
}
type Index struct {
    Array CabsExpr
    Index CabsExpr
}
// ... etc
```

### Statements
```go
type CabsStmt interface{ implCabsStmt() }

type Computation struct {
    Expr CabsExpr
    Loc  Location
}
type Block struct {
    Items []CabsStmt
    Loc   Location
}
type If struct {
    Cond CabsExpr
    Then CabsStmt
    Else CabsStmt  // may be nil
    Loc  Location
}
// ... etc
```

### Declarations
```go
type Definition interface{ implDefinition() }

type FunDef struct {
    Specs []SpecElem
    Name  Name
    Decls []Definition
    Body  CabsStmt
    Loc   Location
}
type DecDef struct {
    Specs []SpecElem
    Inits []InitName
    Loc   Location
}
```

## Testing Strategy

### Input/Output Tests (`testdata/parse.yaml`)

```yaml
tests:
  - name: "empty function"
    input: |
      int main() {}
    ast:
      kind: FunDef
      specs: [{kind: SpecType, type: Tint}]
      name: {name: main, decl: PROTO}
      body: {kind: Block, items: []}

  - name: "return statement"
    input: |
      int f() { return 42; }
    ast:
      kind: FunDef
      body:
        kind: Block
        items:
          - kind: Return
            expr: {kind: Constant, value: "42"}
```

### Equivalence Testing

Compare output with CompCert's `-dparse`:
```bash
# Generate reference
ccomp -dparse test.c  # produces test.parsed.c

# Compare ASTs (via a comparison tool)
ralph-cc -dparse test.c
diff test.parsed.c test.ralph.parsed.c
```

## Milestones

1. **M1: Minimal Parser** - Parse `int main() { return 0; }`
2. **M2: Expressions** - Full expression precedence, binary/unary ops
3. **M3: Statements** - if/while/for/switch/goto
4. **M4: Declarations** - Variables, functions, typedef
5. **M5: Types** - Structs, unions, enums, pointers, arrays
6. **M6: Full Grammar** - All C syntax CompCert supports

## References

- [Pratt Parsing Made Easy](https://journal.stuffwithstuff.com/2011/03/19/pratt-parsers-expression-parsing-made-easy/) - Bob Nystrom
- [CompCert C Parser](https://github.com/AbsInt/CompCert/tree/master/cparser)
- [C11 Standard Grammar](https://port70.net/~nsz/c/c11/n1570.html#A)
