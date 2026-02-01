# Frontend Issues

Issues discovered during frontend review.

## Issue 1: SimplLocals address-taken analysis doesn't handle `*cabs.Block`

**Severity:** Critical  
**Symptom:** Variables whose address is taken (e.g., `&x`) are incorrectly promoted to temporaries, causing panic in Csharpminor generation: "cannot take address of expression"

**Root Cause:**  
In `pkg/simpllocals/transform.go`, `AnalyzeStmt()` has a case for `cabs.Block` (value type), but `FunDef.Body` is `*cabs.Block` (pointer type). The switch case doesn't match, so the function body is never analyzed for address-taken variables.

**Location:** `pkg/simpllocals/transform.go:157`

**Fix:**  
Add a case for `*cabs.Block` in `AnalyzeStmt`:
```go
case *cabs.Block:
    for _, item := range stmt.Items {
        t.AnalyzeStmt(item)
    }
```

**Test Case:**
```c
int test() {
    int x;
    int *p;
    p = &x;  // x's address is taken
    return *p;
}
```
Should work with `-dcsharpminor` without panic. Variable `x` should remain a local (not promoted to temp).

---

## Issue 2: Clight printer shows wrong variable name in address-of

**Severity:** Medium (cosmetic but confusing)  
**Symptom:** After SimplLocals, the Clight output shows `$1 = &$1;` instead of `$1 = &x;`

**Root Cause:**  
Related to Issue 1. Once Issue 1 is fixed, `x` should remain as `Evar{Name: "x"}` and not be promoted, making the output correct.

**Test Case:**
After fixing Issue 1:
```bash
bin/ralph-cc /tmp/addr_test.c -dclight
```
Should show `p = &x;` (with `x` as a local, not a temp).

---

## Issue 3: all.c crashes on `-dcsharpminor` and `-dcminor`

**Severity:** Critical  
**Symptom:** Running `bin/ralph-cc testdata/example-c/all.c -dcsharpminor` panics

**Root Cause:**  
Same as Issue 1. The `pointerOps` function takes `&x`:
```c
int pointerOps(int *p, int **pp) {
    int x;
    int *localp;
    ...
    localp = &x;
    ...
}
```

**Fix:** Same as Issue 1.

---

## Notes

- CminorSel is documented as an internal phase with no CompCert dump flag, so no `-dcminorsel` CLI flag is needed
- The `pkg/selection` package is complete but wired only via `pkg/cminorsel` (internal use)
