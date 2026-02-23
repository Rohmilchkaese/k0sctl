# k0sctl PR Submission Plan

## Tier 1: Fixes existing open issues (very likely to merge)

Submit these first — they directly close reported issues, are small, low-risk, and have tests.

| # | Branch | Issue(s) | What it fixes |
|---|--------|----------|---------------|
| 1 | `claude/fix-395-nil-panic-4YOuT` | #395 | Nil panic when `spec.k0s` is empty |
| 2 | `claude/fix-659-x20-escaping-4YOuT` | #659 | Hex escapes in shell Unquote (systemd) |
| 3 | `claude/fix-947-restore-guard-4YOuT` | #947 | Error out on restore against running cluster |
| 4 | `claude/fix-950-warn-no-lb-4YOuT` | #950, #602 | Warn when multi-controller lacks LB config |

## Tier 2: Bug fixes found by code review (likely to merge)

Clear bugs that affect users. Easy to justify — no behavior change debate.

| # | Branch | What it fixes |
|---|--------|---------------|
| 5 | `claude/fix-apply-manifests-silent-fail-4YOuT` | `kubectl apply` errors silently swallowed |
| 6 | `claude/fix-reset-log-path-4YOuT` | Copy-paste bug: wrong path logged in reset cleanup |
| 7 | `claude/fix-trim-shell-output-4YOuT` | Missing TrimSpace on shell output (arch, time, etc.) |
| 8 | `claude/fix-resolve-configurer-nil-4YOuT` | Nil pointer in ResolveConfigurer error path |
| 9 | `claude/fix-github-body-leak-4YOuT` | HTTP response body leak in GitHub client |
| 10 | `claude/fix-daemon-reload-error-4YOuT` | Silent failure in daemon reload (returns nil on error) |
| 11 | `claude/fix-unsafe-type-assertions-4YOuT` | Unsafe type assertions (panic risk) in cmd/ |
| 12 | `claude/fix-stdin-dead-code-4YOuT` | Dead code in init command stdin error handling |
| 39 | `claude/fix-internal-addr-nil-4YOuT` | Nil pointer in clusterInternalAddress() when no controllers |
| 40 | `claude/fix-isemptyk0s-logic-4YOuT` | Unreachable code / logic bug in isEmptyK0s() |
| 41 | `claude/fix-init-cleanup-nil-4YOuT` | Nil pointer in InitializeK0s.CleanUp() when leader is nil |

## Tier 3: Security fixes (likely to merge)

Shell injection and input validation fixes.

| # | Branch | What it fixes |
|---|--------|---------------|
| 13 | `claude/fix-configurer-injection-4YOuT` | Shell injection via unescaped paths in sed |
| 14 | `claude/fix-filecontains-injection-4YOuT` | Command injection in FileContains() grep |
| 15 | `claude/fix-privateaddr-injection-4YOuT` | Unquoted interface name in PrivateAddress() |
| 16 | `claude/fix-reset-confirm-error-4YOuT` | Ignored survey confirmation error in reset |
| 17 | `claude/fix-perm-parse-error-4YOuT` | Unchecked file permission parsing |

## Tier 4: Upgrade safety improvements (high impact, may need discussion)

| # | Branch | What it does |
|---|--------|--------------|
| 18 | `claude/add-version-skew-check-4YOuT` | Warn on K8s N-2 version skew during upgrades |
| 19 | `claude/add-upgrade-health-check-4YOuT` | Log health after controller upgrade readiness check |
| 20 | `claude/add-upgrade-rollback-4YOuT` | Backup binary before replacement for rollback |
| 21 | `claude/fix-reset-silent-errors-4YOuT` | Return error from k0s reset instead of swallowing it |

## Tier 5: Robustness improvements (good quality PRs)

| # | Branch | What it does |
|---|--------|--------------|
| 22 | `claude/fix-http-timeout-4YOuT` | Add timeout on HTTP downloads |
| 23 | `claude/add-download-retry-4YOuT` | Add retry with exponential backoff for binary downloads |
| 24 | `claude/fix-cleanup-logging-4YOuT` | Log warnings on cleanup failures instead of ignoring |
| 25 | `claude/fix-context-propagation-4YOuT` | context.TODO → context.Background in disconnect |
| 26 | `claude/fix-lock-race-4YOuT` | Add ticker.Stop() in lock file goroutine |
| 27 | `claude/fix-port-validation-4YOuT` | Port range and controller count validation in init |
| 28 | `claude/add-reset-knownhosts-cleanup-4YOuT` | Remove SSH known_hosts entries after node reset |

## Tier 6: New features (may need RFC/discussion)

| # | Branch | What it does |
|---|--------|--------------|
| 38 | `claude/add-status-command-4YOuT` | New `k0sctl status` command — read-only cluster health reporting |

## Tier 7: Code quality (might need discussion)

| # | Branch | What it does |
|---|--------|--------------|
| 29 | `claude/fix-error-wrapping-4YOuT` | Use `%w` instead of `%s`/`%v` for errors |
| 30 | `claude/fix-temp-file-dedup-4YOuT` | Extract shared temp file path helper |
| 31 | `claude/add-phase-timing-4YOuT` | Log elapsed time per phase |
| 32 | `claude/add-validate-command-4YOuT` | New `k0sctl validate` CLI command |
| 33 | `claude/add-shell-split-tests-4YOuT` | Add tests for internal/shell/split.go |
| 34 | `claude/fix-ssh-hostkey-detection-4YOuT` | Add comment explaining SSH host key error detection |
| 35 | `claude/add-kubeconfig-warning-4YOuT` | Warn when kubeconfig written to stdout (sensitive) |
| 36 | `claude/add-backup-logging-4YOuT` | Add intermediate progress logging to backup phase |
| 37 | `claude/fix-evict-taint-validation-4YOuT` | Validate evict-taint effect from CLI flag |

## Status

- [x] All 41 branches created and pushed
- [x] All branches build cleanly (`go build ./...`)
- [x] All tests pass (`go test ./...`)
- [x] No vet warnings (`go vet ./...`)
- [ ] Integration test all branches together (uber branch)
- [ ] PRs submitted upstream
