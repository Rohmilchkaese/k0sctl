# k0sctl PR Submission Plan

## Tier 1: Fixes existing open issues (very likely to merge)

Submit these first — they directly close reported issues, are small, low-risk, and have tests.

| # | Branch | Issue(s) | What it fixes |
|---|--------|----------|---------------|
| 1 | `claude/fix-395-nil-panic-4YOuT` | #395 | Nil panic when `spec.k0s` is empty |
| 2 | `claude/fix-659-x20-escaping-4YOuT` | #659 | Hex escapes in shell Unquote (systemd) |
| 3 | `claude/fix-947-restore-guard-4YOuT` | #947 | Error out on restore against running cluster |
| 4 | `claude/fix-950-warn-no-lb-4YOuT` | #950, #602 | Warn when multi-controller lacks LB config |

## Tier 2: Found by code review — bug/security fixes (likely to merge)

No existing issues, but these are clear bugs/security problems. Easy to justify.

| # | Branch | What it fixes |
|---|--------|---------------|
| 5 | `claude/fix-apply-manifests-silent-fail-4YOuT` | `kubectl apply` errors silently swallowed |
| 6 | `claude/fix-configurer-injection-4YOuT` | Shell injection via unescaped paths in sed |
| 7 | `claude/fix-http-timeout-4YOuT` | No timeout on HTTP downloads (can hang forever) |

## Tier 3: Found by code review — code quality (might need discussion)

These are improvements rather than fixes. Maintainers may have opinions on approach.

| # | Branch | What it does |
|---|--------|--------------|
| 8 | `claude/fix-error-wrapping-4YOuT` | Use `%w` instead of `%s`/`%v` for errors |
| 9 | `claude/fix-temp-file-dedup-4YOuT` | Extract shared temp file path helper |
| 10 | `claude/add-phase-timing-4YOuT` | Log elapsed time per phase |
| 11 | `claude/add-validate-command-4YOuT` | New `k0sctl validate` CLI command |

## Status

- [x] All branches created and pushed
- [x] All branches build cleanly (`go build ./...`)
- [x] All tests pass (`go test ./...`)
- [x] No vet warnings (`go vet ./...`)
- [x] All branches merge cleanly together (tested on uber branch)
- [ ] PRs submitted upstream
