# The RALPH Technique in Practice: Building a C Compiler with Autonomous Agents

_A deep-dive case study for coding agent developers_

---

## 1. Introduction: Why This Article Exists

If you've worked on coding agents—systems like OpenHands, Devin, or the software-agent-sdk—you've grappled with the long-horizon problem. How do you get an AI to complete tasks that require hundreds of steps, span multiple files, and take hours instead of minutes? How do you prevent context degradation, scope creep, and the accumulation of half-broken implementations?

In mid-2025, Geoffrey Huntley introduced a technique called "Ralph" that takes a radically different approach: instead of sophisticated orchestration, just run a dumb loop. Feed a prompt to an AI coding agent, let it work, and when it exits, do it again. And again. Forever.

In its purest form, Ralph is five words of bash:

```bash
while :; do cat PROMPT.md | claude-code ; done
```

That's it. No state machine. No safety rails. No memory system. Just a loop that reads a prompt file and pipes it to an AI agent.

This sounds absurd. And yet, Ralph has been used to build real software—including entire programming languages and commercial application clones. The technique went viral. Anthropic created an official plugin. Y Combinator startups adopted it.

This article takes a different approach than the typical Ralph explainer. Instead of describing the philosophy in the abstract, we're going to walk through a concrete implementation: [ralph-cc-go](https://github.com/raymyers/ralph-cc-go), an experiment in building a C compiler using the Ralph technique with the OpenHands agent. This project was created by [Ray Myers](https://github.com/raymyers), who began the experiment on February 1, 2026—four days before Anthropic published their own parallel exploration, ["Building a C compiler with a team of parallel Claudes"](https://www.anthropic.com/engineering/building-c-compiler). While both projects share the ambitious goal of AI-driven compiler construction, they represent independent approaches: Anthropic's work used 16 parallel Claude agents and $20,000 in API costs to build a 100,000-line Rust compiler, while Myers' ralph-cc-go explores the same territory using the simpler Ralph loop technique with OpenHands.

We'll examine the actual commits. We'll see what the RALPH.md prompts looked like and how they evolved. We'll watch the agent tackle 126 parser tasks in three hours, create its own implementation plans, and systematically debug issues discovered through fuzz testing.

By the end, you'll have the experience of having worked with Ralph—not just understood its theory.

---

## 2. The RALPH Technique: Core Concepts

### The Philosophy of Naive Persistence

Ralph is named after Ralph Wiggum, the lovably dim but eternally persistent character from _The Simpsons_. Like its namesake, Ralph doesn't know it's failing. It doesn't get discouraged. It just keeps trying.

> "The technique is deterministically bad in an undeterministic world. It's better to fail predictably than succeed unpredictably."
> — Geoffrey Huntley

The key insight is that most AI coding failures come from **context degradation**. In a long conversation, the model's context window fills up with unsuccessful attempts, dead ends, and accumulated confusion. By the time you're on attempt #7, the model is swimming through a sea of its own wreckage.

Ralph solves this by **throwing away the context and starting fresh**. Each iteration:

1. The AI gets a **fresh context window** (starts clean)
2. It reads your `PROMPT.md` and any `AGENTS.md` files
3. It sees its _previous work_ through the filesystem—git commits, modified files, test results
4. It picks up where it left off, but without the "context rot" of a long conversation

Progress persists in **files and git history**, not in LLM memory.

### Stateless Resampling

When practitioners talk about Ralph, they emphasize **stateless resampling**. Each loop iteration is statistically independent—a fresh sample from the model's capability distribution. Bad implementations get overwritten. Good patterns compound. Given enough iterations and well-tuned prompts, the system converges on working software.

This is fundamentally different from the traditional "chat until it works" approach. Instead of accumulating context, you're taking fresh shots at the problem with the benefit of all previous artifacts.

### Tune It Like a Guitar

Huntley describes refining Ralph prompts with a specific metaphor:

> "Each time Ralph does something bad, Ralph gets tuned—like a guitar."

The approach is **observational and reactive**:

1. Run Ralph
2. Watch what it does
3. When it fails in specific ways, add guardrails
4. Run it again

These guardrails—Huntley calls them "signs"—aren't just prompt text. They're anything Ralph can discover:

- Explicit instructions ("don't assume not implemented")
- Operational learnings in `AGENTS.md`
- Utility functions and patterns in your codebase that Ralph will mimic
- Tests that reject invalid work (what Huntley calls "backpressure")

### What Ralph Is NOT

As Ralph went viral, various formalizations appeared—and Huntley pushed back against many of them.

The Anthropic plugin, for instance, maintains a growing conversation history rather than starting fresh. It uses a "Stop Hook" that intercepts when Claude tries to exit, keeping the same session going. This is **fundamentally different** from the original approach.

One practitioner, Dex Horthy, tested the official plugin and reported:

> "It misses the key point of ralph which is not 'run forever' but in 'carve off small bits of work into independent context windows.'"

The power of original Ralph wasn't just the looping—it was the **rawness**. The AI confronts its broken builds, its failed tests, its half-implemented features without any protective formatting. It has to figure out what went wrong by reading its own wreckage.

---

## 3. The ralph-cc-go Experiment: Setup and Context

### What the Project Aimed to Build

[ralph-cc-go](https://github.com/raymyers/ralph-cc-go) is an experiment in building a C compiler using the Ralph technique. The README poses a simple question:

> "Can we code a C compiler hands-free?"

The compiler follows CompCert's design—a verified C compiler developed by INRIA that transforms C through a series of intermediate representations (IRs). The goal wasn't just to build _a_ compiler, but to build one that follows a proven architecture with multiple transformation passes.

This is a genuinely difficult project. A C compiler involves:

- Lexical analysis and tokenization
- Parsing with precedence handling
- Multiple intermediate representations (Clight, Csharpminor, Cminor, RTL, LTL, Linear, Mach, ASM)
- Code generation for a real architecture (ARM64)
- Calling convention compliance
- Stack frame management

### Ground Rules

The README establishes clear boundaries:

> "Manual edits are allowed only on `README.md` and `plan/*`. The other commits must be by agents prompted using a repeatable `plan/*/RALPH.md`"

This creates a clean experiment: humans can steer via planning documents, but all implementation must come from the agent. It also makes it easy to distinguish human vs. agent commits by examining which files were touched.

### Repository Structure

The project organizes work into numbered plan folders:

```
plan/
├── 01-cli-ralph/      # Initial implementation
├── 02-e2e-ralph/      # End-to-end testing
├── 03-pop-ralph/      # Making real programs work
├── 04-learn/          # (unused)
├── 05-fix-research-ralph/  # Bug research (read-only)
├── 06-regression-ralph/    # Regression testing
├── 07-sqlite-ralph/        # SQLite compilation
└── 08-parallel-sdk-triage/ # Parallel agent orchestration
```

Each plan folder contains:

- `RALPH.md` - The prompt that drives the agent
- `PLAN.md` - The task list with checkboxes
- `progress/` - Files that preserve context across iterations
- Various sub-plans (like `PLAN_PARSING.md`)

### The Initial RALPH.md

Here's the first RALPH.md prompt from `plan/01-cli-ralph/`:

```markdown
Execute these steps.

1. From the unfinished tasks in plan/01-cli-ralph/PLAN.md, choose a logical one to do next.
2. Do ONLY that task, and related automated tests.
3. Verify (including `make check`).
4. If complete: update PLAN to mark complete, commit.

## Environments

Use `container-use` cli if you need to install anything.

## Tech Guidelines

We are building a C compiler frontend CLI, optimized for testing of the compilation passes rather than practical use.

Our CLI is in Go lang, but following the compcert design with goal of equivalent output on each IR. Optimizations are not required (compare with -O0).

Makefile should have test, lint and check (doing both).

### AST

Use this pattern for ASTs, aiming for type-safe ADT style...
```

Note the key instructions:

- **Pick one task from a list** - enables stateless iteration
- **Do ONLY that task** - prevents scope creep
- **Verify with `make check`** - backpressure via tests
- **Mark complete and commit** - progress persists in git

This is the minimal scaffolding that makes Ralph work on long-horizon tasks.

---

## 4. Phase 1: Exploration and Setup (Commits 1-35)

### The Initial Human Setup

**Commit fdd77c0** (Human, 02:45): The first commit creates the project skeleton:

```
- AGENTS.md (8 lines) - Instructions for container-use tool
- README.md (22 lines) - Project description and ground rules
- plan/01-cli-ralph/ - Empty folder structure
- plan/lint.py (111 lines) - Utility script
```

At this point, there's no RALPH.md yet. The human is setting up the experiment framework.

### The Agent Explores CompCert

What follows is fascinating: approximately 28 commits representing the agent exploring the CompCert codebase in a container environment. Most of these are **empty commits**—they record container actions without changing files:

```
98d744c Create environment comic-hookworm: Add CompCert as submodule and build
38bcf75 Clone CompCert repository into compcert directory
879c492 Read configure script header to understand build options
5a08d7b Install basic build dependencies (opam for OCaml/Coq, make, gcc)
...
e131a44 Read the main Compiler.v to understand pass composition
```

The agent:

1. Cloned the CompCert repository
2. Read the configure script
3. Installed dependencies (opam, Coq, Menhir, GCC)
4. Built CompCert (with some trial-and-error on Coq version)
5. Tested with a simple C program
6. Explored ccomp options for viewing intermediate representations
7. Read Compiler.v to understand pass composition

This "exploration first" pattern produced a key artifact: **docs/PHASES.md** (152 lines), documenting all CompCert compilation phases:

```markdown
# CompCert Compilation Phases

CompCert transforms C source code through a series of intermediate representations (IRs), each with a formally verified semantics...

## Intermediate Languages

### Frontend Languages

| Language       | Description                                                            |
| -------------- | ---------------------------------------------------------------------- |
| **CompCert C** | Full C subset accepted by CompCert. Expressions may have side effects. |
| **Clight**     | C without side-effects in expressions. Evaluation order is fixed.      |

...
```

The agent spent significant time understanding the reference implementation before any actual coding. This documentation would guide later implementation.

### The Critical Commit: RALPH.md is Born

**Commit bfdc233** (Human, 04:09): This is where the Ralph technique truly begins.

```
Files Changed:
- README.md - Reduced from 22 to 8 lines
- plan/01-cli-ralph/PLAN.md - Added 7 task items (checkboxes)
- plan/01-cli-ralph/RALPH.md - CREATED (54 lines)
- plan/01-cli-ralph/README.md - Added 7 lines
```

The human creates the RALPH.md prompt we saw earlier, plus an initial PLAN.md:

```markdown
[x] Add compcert submodule and get it to build
[x] Understand compcert phases, summarize in docs/PHASES.md
[ ] Initialize go with bin `ralph-cc`, cobra CLI, Makefile
[ ] Implement placeholders for debug flags: -dparse -dc -dasm -dclight...
[ ] Study Menhir grammar and plan parser approach in docs/PARSING.md
[ ] Try parsing a tiny C subset with YAML-driven tests
[ ] Update docs/PARSING.md with bulleted list of syntax elements needed
```

Note that the first two items are pre-checked—the agent's exploration work is being retroactively tracked.

### First Implementation Commit

**Commit 666cc4b** (Agent, 04:22): "Initialize Go CLI with Cobra and Makefile"

This is the first real implementation by the agent. It has the Co-authored-by trailer that marks agent commits:

```
Files Created/Changed:
- cmd/ralph-cc/main.go (35 lines) - Cobra CLI skeleton
- cmd/ralph-cc/main_test.go (11 lines) - Basic test
- go.mod (9 lines)
- go.sum (10 lines)
- Makefile (23 lines) - test, lint, check targets
- .gitignore (1 line)
- plan/01-cli-ralph/PLAN.md - Updated checkbox [x]
```

The agent followed the RALPH.md pattern exactly: pick one task, implement, test, mark complete, commit. Clean and focused.

---

## 5. Phase 2: Parser Implementation - The 126-Task Sprint (Commits 36-68)

### The Agent Self-Organizes

**Commit 486dbeb** (Agent, 04:58): "Add parser plan: hand-written recursive descent with Pratt parsing"

Before implementing the parser, the agent studied the CompCert Menhir grammar and made an architectural decision:

- **Hand-written recursive descent** (not a parser generator)
- **Pratt parsing for expressions** (precedence climbing)
- **Typedef tracking during parse**

This is documented in **docs/PARSING.md** (323 lines), showing the agent's reasoning process.

**Commit 2ae962e** (Agent, 05:12): "Add PLAN_PARSING.md with detailed parser implementation tasks"

The agent created its own sub-plan with **126 tasks** organized into milestones:

```markdown
## M1: Minimal Parser (Complete)

- [x] Parse `int main() {}` - empty function
- [x] Parse `int f() { return 0; }` - return with integer literal

## M2: Expressions

### M2.1: Pratt Parser Infrastructure

- [x] Add precedence constants for all C operators
- [x] Implement Pratt parser skeleton with `parseExpr(precedence int)`
      ...

### M2.11: Sizeof

- [ ] Parse `sizeof expr` and `sizeof(type)`
- [ ] Add Sizeof AST node

## M3: Statements

### M3.1: Expression Statements

...

## M6: Post-MVP Features

...
```

This self-organizing behavior is remarkable. The agent recognized that the task "implement a parser" was too large and decomposed it into 126 trackable sub-tasks across 6 milestones.

### Human "Guitar Tuning"

**Commit d2a6022** (Human, 05:14): "Add RALPH note"

The human adds 4 lines to RALPH.md:

```markdown
## Environments

Use `container-use` cli if you need to install anything.
```

This is the "guitar tuning" Huntley describes. The human observed that the agent might need to install things and added guidance. A small adjustment based on observation.

### The Sprint

What follows is rapid implementation:

| Time  | Commit  | Description                                   |
| ----- | ------- | --------------------------------------------- |
| 05:24 | 42027ab | Pratt parser with expressions (25% milestone) |
| 05:31 | 8c8d74a | Compound assignment, inc/dec, member access   |
| 05:33 | 0ad72e6 | sizeof expression                             |
| 05:35 | 465ad81 | Cast expressions                              |
| 05:37 | 4cbcaf6 | Expression statements                         |
| 05:39 | 438cc2f | if/else statements                            |
| 05:40 | 64a81e1 | while/do-while loops                          |
| 05:41 | 8f7e7c4 | for loops                                     |
| 05:42 | 981bd7f | break/continue                                |
| 05:42 | 43effff | **50% milestone reached**                     |

In roughly 20 minutes, the agent went from 25% to 50% on the parser. Each commit is one logical task—focused, testable, verifiable.

The sprint continues:

| Time  | Commit  | Description                                 |
| ----- | ------- | ------------------------------------------- |
| 05:45 | 6ba7503 | switch/goto/labels                          |
| 05:52 | 627dc2a | Declarations, structs, unions, enums        |
| 06:07 | 4353dfb | ParseProgram and -dparse CLI flag           |
| 06:19 | fe67c9a | Function pointers, multi-dimensional arrays |
| 06:24 | 7ece12c | VLA support                                 |
| ...   | ...     | ...                                         |
| 08:18 | 6de5f99 | **100% parser complete**                    |

**126 tasks completed in approximately 3 hours.**

Human intervention during this phase? Just two "Update plan" commits touching only PLAN.md—minor steering adjustments, not implementation.

### What Made This Work

Several factors enabled this productivity:

1. **Granular task breakdown** - 126 small, testable tasks
2. **One task at a time** - Prevented scope creep
3. **Verification gate** - `make check` caught issues before commit
4. **Checkbox tracking** - Agent knew exactly what was done vs. pending
5. **Fresh context each iteration** - No accumulated confusion

---

## 6. Phase 3: Backend Pipeline - Self-Organized Planning (Commits 69-115)

### Agent Creates Phase Plans

**Commit 3b597be** (Agent, 13:29): "Add implementation plans for all compilation phases"

The agent created **10 PLAN*PHASE*\*.md files** totaling **1,737 lines** of implementation plans:

| Plan File                 | Phase                   | Key Content                  |
| ------------------------- | ----------------------- | ---------------------------- |
| PLAN_PHASE_CLIGHT.md      | SimplExpr + SimplLocals | Cabs → Clight transformation |
| PLAN_PHASE_CSHARPMINOR.md | Cshmgen                 | Type-dependent operations    |
| PLAN_PHASE_CMINOR.md      | Cminorgen               | Stack allocation             |
| PLAN_PHASE_CMINORSEL.md   | Selection               | ARM64 instruction selection  |
| PLAN_PHASE_RTL.md         | RTLgen                  | CFG generation               |
| PLAN_PHASE_RTL_OPTS.md    | Optimizations           | (optional passes)            |
| PLAN_PHASE_LTL.md         | Register allocation     | Liveness analysis            |
| PLAN_PHASE_LINEAR.md      | Linearization           | CFG → linear code            |
| PLAN_PHASE_MACH.md        | Frame layout            | Concrete stack frames        |
| PLAN_PHASE_ASM.md         | Asmgen                  | ARM64 assembly               |

Each plan includes:

- Key CompCert files to study
- Milestone sections with task bullets
- AST definitions needed
- CLI integration steps
- Testing strategies

Here's an excerpt from **PLAN_PHASE_RTL.md** showing the level of detail:

```markdown
# Phase: RTL Generation (RTLgen)

**Transformation:** CminorSel → RTL
**Prereqs:** CminorSel generation (PLAN_PHASE_CMINORSEL.md)

## Key CompCert Files to Study

| File                  | Purpose                       |
| --------------------- | ----------------------------- |
| `backend/RTL.v`       | RTL AST definition            |
| `backend/RTLgen.v`    | Transformation from CminorSel |
| `backend/Registers.v` | Pseudo-register definitions   |

## Milestone 1: RTL AST Definition ✅

- [x] Create `pkg/rtl/ast.go` with node interfaces
- [x] Define registers: Pseudo-register type (infinite supply)
- [x] Define RTL instructions:
  - [x] `Inop` (no operation, jump to successor)
  - [x] `Iop` (operation: `rd = op(rs...)`)
  - [x] `Iload` (memory load: `rd = Mem[addr]`)
        ...
```

This shows the agent's ability to **decompose a large problem autonomously**. Nobody told it to create 10 detailed phase plans—it recognized the need and did it.

### Systematic Implementation

Over the next 4 hours, the agent implemented the entire backend:

| Time  | Commit  | Phase       | Key Work                                  |
| ----- | ------- | ----------- | ----------------------------------------- |
| 13:43 | 9bfba86 | Clight      | pkg/ctypes, pkg/clight AST                |
| 14:03 | 5140ad7 | Clight      | SimplExpr + SimplLocals transformations   |
| 14:39 | a19dc86 | Csharpminor | Operator/expr/stmt translation            |
| 14:46 | c63955b | Refactor    | Extracted clightgen into separate package |
| 15:18 | 3217e71 | Cminor      | Stack frame, variable transform           |
| 15:49 | d00f30a | CminorSel   | Instruction selection for ARM64           |
| 16:21 | 021b6ef | RTL         | CFG construction, register allocation     |
| 16:33 | 670a36e | LTL         | Full register allocation pass             |
| 16:48 | 3d14611 | Linear      | Linearization and tunneling               |
| 17:06 | ecda3fe | Mach        | Stack frame layout                        |
| 17:24 | 5cf6473 | **ASM**     | ARM64 assembly generation                 |
| 17:27 | 4852ea4 | Tests       | 11 end-to-end YAML tests                  |

**Full compilation pipeline (C → ARM64 assembly) in ~4 hours.**

### Emergent Behaviors

Several behaviors emerged without explicit instruction:

**Self-Review (Commit dbd1e7c, 15:53)**:

> "Review frontend, document SimplLocals address-taken bug in PLAN_FRONTEND_ISSUES.md"

The agent reviewed its own work and found a bug—the SimplLocals pass wasn't handling a certain case correctly.

**License Addition (Commit e794121, 16:50)**:

> "Add LICENSE file with CompCert non-commercial use notice"

Without being asked, the agent added an appropriate license noting the CompCert derivation and its implications.

**Refactoring (Commit c63955b, 14:46)**:

> "refactor: Extract Clight generation into pkg/clightgen package"

The agent recognized that main.go was getting too large (670 lines) and refactored to improve organization.

These behaviors show the agent developing **domain awareness** and making **judgment calls** beyond the task list.

---

## 7. Phase 4: Making Real Programs Work (Commits 116-145)

### The hello.c Journey

With the pipeline complete, the next goal was compiling real C programs. This required:

1. **Preprocessing** - #include, macros
2. **System header compatibility** - stdio.h
3. **Calling conventions** - extern functions like printf

**Commit d74398e** (19:15): "feat: Add #include directive support via external preprocessor"

Initially, the agent used an external preprocessor (like CompCert does):

```bash
cc -E input.c  # Let system cc preprocess
```

But then the human created **PLAN_PREPROCESSOR.md** to guide building an internal preprocessor.

**Commits ceb5017 - c17782e** (19:38 - 20:36): The agent built a complete C preprocessor with 11 milestones:

1. Preprocessor lexer
2. Include path resolution
3. Directive parser
4. Macro definition/storage
5. Macro expansion
6. Conditional compilation (#ifdef, etc.)
7. Include processing
8. CLI integration
9. Token pasting (## operator)
   10-11. Final polish

### Progress Files for Context Handoff

This phase introduces a key evolution: **progress files**.

In **plan/03-pop-ralph/RALPH.md**, we see new instructions:

```markdown
2. If a progress file is specified in the task, study it.
   Otherwise create one in `plan/03-pop-ralph/progress/` and reference in the plan file task.
   ...
   The audience will be a coding agent that needs to continue your task with little help,
   or understand the history of the execution.
```

Progress files serve as **structured memory** across iterations. Here's `FIB_SEGFAULT.md`:

````markdown
# Fib Segfault Debug

## Issue

`testdata/example-c/fib_fn.c` segfaulted when run.

## Root Cause

Two related bugs in assembly generation (asmgen) and stack layout (stacking):

### Bug 1: Prologue Generated Wrong Frame Layout

The asmgen/transform.go prologue was:

```asm
stp x29, x30, [sp, #-framesize]!   ; save FP/LR, pre-decrement SP
```
````

But the Mach IR expects:

```asm
sub sp, sp, #framesize              ; allocate frame
stp x29, x30, [sp, #fpOffset]       ; save FP/LR at TOP of frame
```

## Fix

Changed generatePrologue() to emit...

## Verification

- fib_fn.c now runs correctly
- All unit tests pass

````

This is **explicit context transfer**. When the next iteration picks up this task, it doesn't start from scratch—it reads the progress file to understand what was tried, what worked, and what remains.

### hello.c Finally Works

**Commit 737c509** (17:40): "Fix external function calls and verify hello.c works"

The agent had to add many language features for real-world headers:
- `restrict` keyword (C99)
- Compound type specifiers (signed char, long long)
- Variadic function declarations (...)
- `__attribute__`, `__asm` in declarations
- `inline` and `__inline` keywords
- `typedef struct { ... } name;` pattern
- `__builtin_va_list`

After all this work: **hello.c compiles and runs correctly!**

---

## 8. Phase 5: Bug Fixing and Fuzzing (Commits 146-220)

### Usability Assessment

**Commit 7b75688** (17:59): "Add usability assessment: E2E runtime tests"

The agent created **PLAN_USABLE.md** with:
- Feature matrix for C language coverage
- 60+ executable test cases
- Assessment: **"NOT YET USABLE"** due to broken comparison codegen

Honest self-assessment based on systematic testing.

### Csmith Integration

**Commit acc7516** (22:46): "Add csmith fuzzing automation for compiler testing"

The agent created `scripts/csmith-fuzz.sh`:

```bash
#!/bin/bash
# Compares ralph-cc output against gcc on randomly generated C programs.
# Reports mismatches with reproducible seeds.

for i in $(seq 1 $ITERATIONS); do
    seed=$RANDOM$RANDOM

    # Generate C program with csmith
    csmith --seed $seed ...

    # Compile with both compilers
    gcc -O0 -w -o gcc_out test.c
    ralph-cc -dasm test.c && as -o test.o test.s && ld -o ralph_out test.o

    # Compare exit codes
    if [ "$gcc_exit" -eq "$ralph_exit" ]; then
        passed=$((passed + 1))
    else
        # Report mismatch
        cp test.c "$REPORT_DIR/mismatch_$seed.c"
    fi
done
````

This is powerful: **automated differential testing** against a known-correct reference (gcc).

### Read-Only Research Mode

For investigating bugs, a new plan folder with different rules:

**plan/05-fix-research-ralph/RALPH.md**:

```markdown
## Make No Changes

You may run code to investigate but do not change it.
Only record ideas for us to make changes later.
```

This separates **investigation from implementation**. The agent:

1. Studies the failing test case
2. Traces through IR dumps
3. Documents root cause
4. Writes fix plan in `fixes/`

But doesn't actually change code. This prevents hasty, incorrect fixes.

### COMMON_CAUSES.md - A Knowledge Base

The agent built a knowledge base of bug patterns:

```markdown
# Common Causes of Compilation Bugs

## Stack Frame Layout Issues

### Callee-Save Register Offset Sign Error (CONFIRMED)

**Symptom**: Runtime crashes in functions using callee-saved registers
**Cause**: Callee-saved registers stored at positive offsets from FP
**Location**: pkg/stacking/layout.go
**Fix**: Use negative offsets from FP

## Operator Semantics

### Logical AND/OR Missing Short-Circuit (CONFIRMED)

**Symptom**: `(-8 && -8)` returns `-8` instead of `1`
**Cause**: Logical operators mapped to bitwise operators
**Fix**: Convert to short-circuit conditional evaluation
```

This is **accumulated expertise**—the agent learns patterns and documents them for future iterations.

### Deep Dive: A Concrete Fix Research File

To illustrate the depth of the agent's investigation, here's an actual fix research file (`fixes/mismatch_2487828851.md`):

```markdown
# Fix: mismatch_2487828851 - Logical AND/OR operators compiled as bitwise

## Test Case

static int16_t g_2 = (-8);
static uint8_t func_1(void) {
uint8_t l_3 = (g_2 && g_2); // Should be 1 (both non-zero)
...
}

## Expected vs Actual

- `g_2 && g_2` should evaluate to `1` (logical AND of two non-zero values)
- But ralph-cc computes `g_2 & g_2 = -8 & -8 = -8` (bitwise AND)

## Root Cause

In `pkg/simplexpr/transform.go`, the `cabsToBinaryOp` function has placeholder code:
case cabs.OpAnd:
return clight.Oand // placeholder <-- BUG!

## Fix Plan

Transform `a && b` → `a ? (b ? 1 : 0) : 0` with short-circuit evaluation

## Implementation

func (t \*Transformer) transformLogicalAnd(left, right cabs.Expr) TransformResult {
// if (left) { if (right) temp=1 else temp=0 } else { temp=0 }
...
}
```

The agent identified the root cause (placeholder code), understood the correct C semantics (short-circuit evaluation), and planned a concrete fix with implementation code—all without making changes yet. This is the research phase working as intended.

### Systematic Bug Fixing

The bug-fix workflow:

1. **Run csmith** until bugs found
2. **Create fix research** documenting findings
3. **Update COMMON_CAUSES.md** if relevant
4. **Apply fixes** with targeted commits
5. **Add regression test** to prevent recurrence

**Commit 162f722**: "Add csmith regression script"

```bash
# 18 known csmith seeds that previously failed
SEEDS=(
    263236830
    2487828851
    1176020246
    ...
)

for seed in "${SEEDS[@]}"; do
    # Regenerate and test
done
```

**Commit 155bfdd**: "Fix parameter assignment bug causing crashes in csmith tests"

> **All 18 csmith regression seeds now pass**

---

## 9. Phase 6: Advanced Patterns - HANS and the OpenHands SDK

### From Bash Loop to Python SDK

The original Ralph technique uses a simple bash loop. But as the ralph-cc-go project matured, it evolved to use the [OpenHands Software Agent SDK](https://github.com/OpenHands/software-agent-sdk)—a Python library for building AI coding agents. This evolution demonstrates how Ralph's core principles can be implemented programmatically with greater control and parallelization.

The SDK provides the building blocks for agent construction:
- **Agent**: The core abstraction—an LLM with tools and context
- **Conversation**: Manages the interaction loop between user prompts and agent responses
- **Tool**: Capabilities like terminal access, file editing, web browsing
- **Skill**: Domain knowledge injected into the agent's context
- **DelegateTool**: Enables one agent to spawn and coordinate sub-agents

### HANS: A Ralph Implementation in Python

**Commit 7f1d582**: "fix(compiler): HANS STABILIZE - 4 agents"

The project includes `hans.py`—a complete implementation of Ralph-style looping using the OpenHands SDK. The name follows the Ralph tradition of using character names (Hans Moleman, another Simpsons character known for his persistence despite adversity).

Here's the core structure:

```python
#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
#   "openhands-sdk==1.11.1",
#   "openhands-tools==1.11.1",
#   "rich>=13.7.0"
# ]
# ///
"""
Hans - Compiler hardening with OpenHands SDK.
4 parallel agents coordinated by an orchestrator for compiler debugging.
"""

from openhands.sdk import (
    LLM,
    Agent,
    AgentContext,
    Conversation,
    Tool,
)
from openhands.sdk.context import Skill
from openhands.tools.delegate import DelegateTool, register_agent
from openhands.tools.file_editor import FileEditorTool
from openhands.tools.terminal import TerminalTool
```

The script can be run directly with `uv run hans.py` thanks to PEP 723 inline script metadata—no virtual environment setup required.

### Agent Specialization with Skills

HANS defines specialized agent types, each with domain-specific knowledge injected via Skills:

```python
COMPILER_DEBUG_SKILL = Skill(
    name="compiler_debug",
    content="""You are debugging the ralph-cc compiler. Key commands:

BUILD & TEST:
- `make build` (~1s) - build compiler
- `make test` (~2s) - smoke test
- `make check` (~30s) - full test suite

IR DUMPS (find divergence stage):
- `./bin/ralph-cc --dparse  test.c`  - parse tree
- `./bin/ralph-cc --dclight test.c`  - Clight IR
- `./bin/ralph-cc --drtl    test.c`  - RTL IR
- `./bin/ralph-cc --dltl    test.c`  - LTL IR
- `./bin/ralph-cc --dmach   test.c`  - machine code
- `./bin/ralph-cc --dasm    test.c`  - assembly

GCC IS ALWAYS RIGHT. If gcc and ralph-cc disagree, ralph-cc has the bug.

After fixing, ALWAYS run `make test` to verify.""",
)
```

This skill is the "guitar tuning" that Huntley describes—accumulated knowledge about what works, encoded so fresh agent instances can immediately apply it.

### Creating Specialized Agent Types

Each agent type combines tools with specialized skills:

```python
def create_seed_fixer_agent(llm: LLM) -> Agent:
    """Agent specialized in fixing csmith seed failures."""
    skills = [
        COMPILER_DEBUG_SKILL,
        Skill(
            name="seed_fixer",
            content="""You fix csmith-generated test failures.

WORKFLOW:
1. Get the test file from `csmith-reports/crash_<seed>.c` or `mismatch_<seed>.c`
2. Compile with both gcc and ralph-cc, compare outputs
3. Use IR dumps to find where ralph-cc diverges
4. Make minimal fix - one bug, one fix
5. Add seed to regression.sh
6. Verify with `make test`

Return a summary: what was wrong, what you fixed, file(s) changed.""",
        ),
    ]
    return Agent(
        llm=llm,
        tools=[Tool(name=TerminalTool.name), Tool(name=FileEditorTool.name)],
        agent_context=AgentContext(skills=skills),
    )
```

The project defines four agent types:
- **seed_fixer**: Fixes csmith-generated test failures
- **feature_hardener**: Tests and hardens specific compiler features
- **program_porter**: Ports real-world C programs to compile
- **diagnostician**: Deep diagnosis of complex bugs others got stuck on

### Agent Registration and Delegation

Agents are registered so the orchestrator can spawn them by name:

```python
register_agent(
    name="seed_fixer",
    factory_func=create_seed_fixer_agent,
    description="Fixes csmith seed failures by diagnosing and patching compiler bugs",
)
register_agent(
    name="feature_hardener",
    factory_func=create_feature_hardener_agent,
    description="Hardens specific compiler features with edge case testing",
)
register_agent(
    name="program_porter",
    factory_func=create_program_porter_agent,
    description="Ports real-world C programs to compile with ralph-cc",
)
register_agent(
    name="diagnostician",
    factory_func=create_diagnostician_agent,
    description="Deep diagnosis of complex bugs that others got stuck on",
)
```

### Dynamic Task Discovery

HANS automatically discovers what needs to be done based on the current phase:

```python
def discover_tasks_for_phase(phase: str) -> dict[str, str]:
    """Discover available tasks based on current phase."""
    tasks = {}

    if phase in ("STABILIZE", "EXPAND"):
        # Find failing seeds from regression tests
        seeds = discover_failing_seeds(limit=4)
        for i, seed in enumerate(seeds):
            tasks[f"fixer{i + 1}"] = (
                f"Fix the compiler bug causing seed {seed} to fail."
            )

    elif phase == "HARDEN":
        # Parse HARDEN_PLAN.md for unchecked items
        plan_file = Path("plan/08-parallel-sdk-triage/HARDEN_PLAN.md")
        if plan_file.exists():
            content = plan_file.read_text()
            for line in content.splitlines():
                if "- [ ]" in line:
                    feature = line.replace("- [ ]", "").strip()
                    tasks[f"hardener{len(tasks) + 1}"] = f"Harden: {feature}"

    elif phase == "REAL_PROGRAMS":
        for prog in ["jsmn", "miniz", "sqlite", "lua"]:
            tasks[f"porter{len(tasks) + 1}"] = f"Port {prog} to ralph-cc"

    return tasks
```

This is **artifact-driven task discovery**—the agent reads markdown files and test results to determine what to work on, rather than being explicitly told.

### The Orchestrator Pattern

The main orchestrator spawns specialized sub-agents in parallel:

```python
def create_orchestrator_prompt(phase: str, tasks: dict[str, str]) -> str:
    return f"""You are coordinating compiler debugging for ralph-cc.
Current phase: {phase}

STEP 1: Spawn {len(tasks)} sub-agents with these IDs and types:
  IDs: {', '.join(tasks.keys())}

STEP 2: Delegate these tasks in parallel:
{chr(10).join(f'  - {aid}: {task}' for aid, task in tasks.items())}

STEP 3: Wait for all results, then:
  - Summarize what each agent fixed
  - Run `make test` to verify all fixes work together
  - If tests fail, identify which fix broke things

STEP 4: Report final status - how many bugs fixed, any remaining issues.
"""

# Create orchestrator with delegation capability
orchestrator = Agent(
    llm=llm,
    tools=[
        Tool(name="DelegateTool"),
        Tool(name=TerminalTool.name),
        Tool(name=FileEditorTool.name),
    ],
    agent_context=AgentContext(
        skills=[COMPILER_DEBUG_SKILL],
        system_message_suffix="You coordinate multiple sub-agents working in parallel.",
    ),
)

conversation = Conversation(
    agent=orchestrator,
    workspace=os.getcwd(),
    visualizer=DelegationVisualizer(name="HANS"),
)
conversation.send_message(prompt)
conversation.run()
```

### Pre-Flight and Post-Flight Operations

HANS implements the same verification gates as traditional Ralph, but programmatically:

```python
# PRE-FLIGHT: Check for uncommitted changes
if is_dirty():
    if args.reset:
        git_reset()  # Start clean
    elif args.keep:
        pass  # Continue with dirty state (risky)
    else:
        sys.exit(1)  # Cannot start dirty

# Sync with remote
git_pull()

# ... agent work happens ...

# POST-FLIGHT: Verify and commit
if is_dirty():
    if run_tests():  # make test passes
        sh("git add -A")
        commit_msg = generate_commit_message(llm)  # LLM writes commit message
        sh(f'git commit -m "{commit_msg}"')
        sh("git push origin main")
    else:
        console.print("[red]Tests failed - not committing[/red]")
```

The `generate_commit_message` function even uses the LLM to write conventional commit messages from the diff:

```python
def generate_commit_message(llm: LLM) -> str:
    """Use a small agent to generate a commit message from git diff."""
    _, diff = sh("git diff --cached --stat && echo '---' && git diff --cached")
    if len(diff) > 8000:
        diff = diff[:8000] + "\n... (truncated)"

    agent = Agent(llm=llm, tools=[Tool(name=TerminalTool.name)])
    conv = Conversation(agent=agent, workspace=os.getcwd())
    conv.send_message(f"""Generate a git commit message for these changes.
Format: <type>(<scope>): <subject>
Diff:
{diff}
Reply with ONLY the commit message.""")
    conv.run()
    return get_agent_final_response(conv.state.events)
```

### Running HANS

The script supports multiple modes:

```bash
# Auto-discover tasks based on current phase
uv run hans.py --phase stabilize

# Custom task for all agents
uv run hans.py --task "Fix these seeds: 12345, 67890, 11111, 22222"

# Dry run to see what would be done
uv run hans.py --dry-run

# Control parallel agent count
uv run hans.py --agents 8

# Handle dirty state
uv run hans.py --reset  # Discard uncommitted changes
uv run hans.py --keep   # Continue with dirty state
```

### Why SDK Over Bash?

The OpenHands SDK implementation provides several advantages over a raw bash loop:

| Bash Loop | SDK Implementation |
|-----------|-------------------|
| Single agent | Multiple parallel agents |
| No specialization | Agent types with domain skills |
| Manual task selection | Dynamic task discovery |
| External commit logic | Integrated verification gates |
| Text-based prompts | Programmatic prompt construction |
| No cost tracking | Built-in usage metrics |

HANS is essentially a **more sophisticated Ralph**: multiple stateless agents, coordinated by an orchestrator, with the same "fresh context + artifact persistence" philosophy—but with the control and observability that comes from a proper programming interface.

### Key Insight: Ralph Principles are Portable

The ralph-cc-go experiment shows that Ralph's core principles—stateless iteration, verification gates, artifact persistence, one task at a time—can be implemented in multiple ways:

1. **Bash loop** (`while :; do ... done`) - Simplest, zero dependencies
2. **OpenHands SDK** (`hans.py`) - Programmatic control, parallelization
3. **Other agent frameworks** - Same principles, different tooling

The technique is not tied to any specific tool. It's a **pattern for organizing AI work**.

---

## 10. Lessons for Coding Agent Developers

### What Made RALPH Work in This Project

After walking through 257 commits, several patterns emerge as critical:

**1. One Task at a Time**

The "Do ONLY that task" instruction prevented scope creep. Each commit averages 100-300 lines of focused, testable changes. Larger tasks get broken into sub-tasks automatically.

**2. Verification Gates**

`make check` before commit catches issues immediately. Tests reject invalid work before it pollutes the codebase. This is Huntley's "backpressure."

**3. Progress in Artifacts, Not Context**

Everything important gets written to files:

- PLAN.md checkboxes track completion
- Progress files document investigation state
- COMMON_CAUSES.md accumulates expertise
- Git history preserves all work

Fresh context each iteration, but building on concrete artifacts.

**4. Self-Organizing Task Decomposition**

The agent created PLAN*PARSING.md (126 tasks) and 10 PLAN_PHASE*\*.md files (1,737 lines) without being asked. Give an agent the right frame, and it can plan complex work.

**5. Separation of Research and Implementation**

The read-only research mode in plan/05-fix-research-ralph prevents hasty fixes. Investigate thoroughly, document findings, then implement in a separate pass.

**6. Human Tuning, Not Human Directing**

Human commits only touched plan/\* files—task adjustments, new plan folders, prompt refinements. The human's job was designing the loop, not directing each step.

### Patterns to Adopt

For coding agent developers, consider these patterns:

**Stateless Iteration with Checkpoints**

```
Each iteration:
1. Read checkpoint (task list, progress files)
2. Pick next task
3. Execute task
4. Verify (tests, lint)
5. Update checkpoint
6. Commit
```

**Progress Files for Multi-Iteration Tasks**

```markdown
# TASK_NAME.md

## Previous Attempts

- Attempt 1: Tried X, failed because Y
- Attempt 2: Tried Z, partial success

## Current State

The issue is in file.go:123. Root cause is...

## Next Steps

1. Fix the specific function
2. Add test case
```

**Verification as Backpressure**

```bash
# In RALPH.md
Verify (including `make check`).
If not passing, do not commit—fix first or bail.
```

**Read-Only Research Mode**

```markdown
# RALPH.md for research

You may run code to investigate but do not change it.
Only record ideas for us to make changes later.
```

### Potential Improvements

Based on this case study, possible enhancements:

1. **Earlier fuzzing** - Csmith could have caught bugs sooner
2. **Explicit "stuck" handling** - What should the agent do when blocked?
3. **Metric tracking** - Time per task, iteration counts
4. **Parallel task selection** - HANS-style coordination earlier

### The Deeper Insight

Ralph isn't really about the bash loop. It's about a **shift in mindset**:

| Traditional AI Coding   | RALPH Mindset             |
| ----------------------- | ------------------------- |
| One-shot perfection     | Iteration over perfection |
| Failures are setbacks   | Failures are data         |
| Human directs each step | Human designs the loop    |
| Prompt engineering      | Context engineering       |
| In the loop             | On the loop               |

The developer's job has changed. Instead of directing an agent step by step, you:

- Design prompts that converge
- Build backpressure that rejects bad work
- Create artifact structures that preserve progress
- Tune the environment based on observed failures

Then get out of the way and let Ralph ralph.

---

## Final Statistics

| Metric                            | Value                      |
| --------------------------------- | -------------------------- |
| Total commits                     | 257                        |
| Go implementation code            | ~29,770 lines              |
| Test code                         | ~26,103 lines              |
| Plan/progress markdown            | ~6,163 lines               |
| Packages created                  | 26                         |
| Human steering commits            | ~50                        |
| Agent commits with Co-authored-by | 166                        |
| Time span                         | 6 days (Feb 1-6, 2026)     |
| Parser tasks completed            | 126 in ~3 hours            |
| Backend phases                    | 10 implemented in ~4 hours |

---

## Conclusion

The ralph-cc-go experiment demonstrates that the RALPH technique can produce a non-trivial, working software project—a C-to-ARM64 compiler with 26 packages and comprehensive test coverage—with minimal human intervention.

The key mechanisms are:

1. **Stateless resampling** - Fresh context each iteration, progress in files
2. **One task at a time** - Prevents mess accumulation
3. **Verification gates** - Tests reject bad work
4. **Checkpoint persistence** - PLAN.md tracks progress, progress files preserve context
5. **Guitar tuning** - Human observes and adjusts prompts when needed

For coding agent developers, Ralph offers a different paradigm than sophisticated orchestration. Sometimes the dumb thing works surprisingly well. And if a dumb loop can build a compiler, what could a smart version of the same principles achieve?

---

_"Ralph is a Bash loop. A technique that is deterministically bad in an undeterministic world."_
— Geoffrey Huntley

---

## Resources

- [ralph-cc-go Repository](https://github.com/raymyers/ralph-cc-go) - The case study explored in this article, created by Ray Myers
- [Building a C compiler with a team of parallel Claudes](https://www.anthropic.com/engineering/building-c-compiler) - Anthropic's parallel exploration of AI-driven compiler construction
- [Geoffrey Huntley's original blog post](https://ghuntley.com/ralph/) - Origins of the technique
- [A Brief History of Ralph](https://www.humanlayer.dev/blog/brief-history-of-ralph) by Dex Horthy
- [OpenHands](https://github.com/OpenHands/OpenHands) - The coding agent used in this experiment
- [OpenHands SDK](https://github.com/OpenHands/software-agent-sdk) - SDK for building agents like HANS
