# Fix Research Plan

## Csmith Findings to Research

- [x] 20260205-225448 - first fuzzing run, multiple findings

## Findings by Type

### Output Mismatches (semantic bugs)
- [x] mismatch_263236830 - global init with Paren expression lost

### Compilation Failures
- [x] fail_compile_130805769 - stack slot spill handling incomplete

### Runtime Crashes
- [x] crash_55976753 - callee-save offset sign error (same as COMMON_CAUSES)
- [x] crash_785411410 - callee-save offset sign error (same root cause as 20260205-225448)
- [ ] crash_3051214201
- [ ] crash_2690612573
- [ ] crash_184567722
- [ ] crash_1176020246
- [ ] crash_1870324845
- [ ] crash_145691413
- [ ] crash_928313049
- [ ] crash_1093823871
- [ ] crash_853828320
- [ ] crash_1950716464
- [ ] crash_23463107
- [ ] crash_3253220824
- [ ] crash_114128045
