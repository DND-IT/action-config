# TODOS

## Core

### v4: Make dimension fully optional

**What:** Remove the `"service"` fallback from main.go and make dimension truly optional — when unset, expand all dimensions without a primary dimension.

**Why:** Users with non-service dimension names (e.g., `services`, `app`, `component`) shouldn't need to declare `settings.dimension` for basic expansion. The fallback was kept in v3 for backward compatibility.

**Context:** In v3.x, the dimension priority chain is: explicit input > config `settings.dimension` > `"service"` fallback (main.go). To make dimension optional in v4:
1. Remove the `"service"` fallback in main.go (the `else if optsCfg.Dimension == ""` branch)
2. Add error checks: return an error when `target` or `change_detection` is used but no dimension is set (these features require a primary dimension to operate on)
3. Update action.yaml description to reflect that dimension is optional
4. Verify `addDirectoryField` gracefully falls back to `base_dir` only (already works)
5. Bump to v4.0.0 (breaking: users without `settings.dimension` who relied on implicit `"service"` must add it)

**Effort:** S
**Priority:** P2
**Depends on:** Ship v3.x with fallback chain first

## Completed
