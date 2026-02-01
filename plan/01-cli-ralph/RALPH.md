Execute these steps.

1. From the unfinished tasks in plan/01-cli-ralph/PLAN.md, choose a logical one to do next.
2. Do ONLY that task, and related automated tests.
3. Verify (including `make check`).
4. If complete: update PLAN to mark complete, commit.

## Tech Guidelines

We are building a C compiler frontend CLI, optimized for testing of the compilation passes rather than practical use.

Our CLI is in Go lang, but following the compcert design with goal of equivalent output on each IR. Optimizations are not required (compare with -O0).

Makefile should have test, lint and check (doing both).

### AST

Use this pattern for ASTs, aiming for type-safe ADT style. Suppose we have an IR called `XX`:

```go
// XXNode is the base interface for all AST nodes
type XXNode interface {
	implXXNode()
}

// Expression variants
type Number struct {
	Value int
}
type Add struct {
	Left, Right XXNode
}
type Multiply struct {
	Left, Right XXNode
}

// Marker methods for interface implementation
func (Number) implXXNode()   {}
func (Add) implXXNode()      {}
func (Multiply) implXXNode() {}

// Evaluate recursively
func Eval(n XXNode) int {
	switch t := n.(type) {
	case Number:
		return t.Value
	case Add:
		return Eval(t.Left) + Eval(t.Right)
	case Multiply:
		return Eval(t.Left) * Eval(t.Right)
	}
	panic("unhandled node type")
}
```
