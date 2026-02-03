   - PARTIAL FIX - no longer crashes, but output is still wrong
    - Progress history:
      - Implemented caller-saved register handling
        - Added LiveAcrossCalls tracking in interference graph (pkg/regalloc/interference.go)
        - Registers live across function calls now assigned to callee-saved registers (X19-X28)
        - Added FirstCalleeSavedColor constant to conventions.go
        - Fixed coalescing to propagate LiveAcrossCalls constraint (pkg/regalloc/irc.go)
        - Added move from X0 to destination after function calls (pkg/regalloc/transform.go)
      - Fixed frame layout bug (root cause of bus error):
        - Problem: callee-save stores at [FP+0..+24] overwrote saved FP/LR at [FP] and [FP+8]
        - After `stp x29, x30, [sp, #-48]!; mov x29, sp`, FP points to saved FP/LR
        - Callee-saves were writing `str x19, [x29]` which clobbered saved FP
        - Solution: Changed CalleeSaveOffset from 0 to 16 in pkg/stacking/layout.go
        - Now callee-saves start at FP+16, after the 16-byte FP/LR save area
        - Also fixed LocalOffset and OutgoingOffset to account for FP/LR area
      - Updated tests in pkg/stacking/*_test.go to expect positive offsets
    - All existing runtime tests pass (make check succeeds)
    - Build artifacts added to .gitignore (testdata/example-c/fib, *.o)
    
    - REMAINING ISSUES:
      - fib.c runs but outputs wrong values (garbage: 6132904688 repeated)
      - Register allocation is non-deterministic (Go map iteration order)
      - Output varies between runs due to different register assignments
      - Need to investigate:
        1. Make register allocation deterministic
        2. Check if there's a liveness/interference bug for variables across calls
        3. The garbage value (0x16d8cb2f0) looks like a memory address
        4. Possible type confusion (int vs long long) in printf call
