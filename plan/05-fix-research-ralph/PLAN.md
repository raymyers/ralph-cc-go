# Fix Research Plan

## Csmith Findings to Research

- [x] 20260205-225448 - first fuzzing run, multiple findings

## Findings by Type

### Output Mismatches (semantic bugs)
- [x] mismatch_263236830 - global init with Paren expression lost
- [x] mismatch_2487828851 - logical AND/OR (&&/||) compiled as bitwise operators

### Compilation Failures
- [x] fail_compile_130805769 - stack slot spill handling incomplete
- [ ] incomplete_logical_ops - missing helper methods break build (HIGH PRIORITY)

### Runtime Crashes
- [x] crash_55976753 - callee-save offset sign error (same as COMMON_CAUSES)
- [x] crash_785411410 - callee-save offset sign error (same root cause as 20260205-225448)
- [x] crash_3051214201 - callee-save offset sign error (verified duplicate)
- [x] crash_2690612573 - callee-save offset sign error (verified duplicate)
- [x] crash_184567722 - callee-save offset sign error (verified duplicate)
- [x] crash_1176020246 - callee-save offset sign error (verified duplicate, batch verification)
- [x] crash_1870324845 - callee-save offset sign error (verified duplicate, batch verification)
- [x] crash_145691413 - callee-save offset sign error (verified duplicate, batch verification)
- [x] crash_928313049 - callee-save offset sign error (verified duplicate, batch verification)
- [x] crash_1093823871 - callee-save offset sign error (verified duplicate, batch verification)
- [x] crash_853828320 - callee-save offset sign error (verified duplicate, batch verification)
- [x] crash_1950716464 - callee-save offset sign error (verified duplicate, batch verification)
- [x] crash_23463107 - callee-save offset sign error (verified duplicate, batch verification)
- [x] crash_3253220824 - callee-save offset sign error (verified duplicate, batch verification)
- [x] crash_114128045 - callee-save offset sign error (verified duplicate, batch verification)
