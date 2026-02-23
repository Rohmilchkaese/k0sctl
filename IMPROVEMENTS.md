# k0sctl Codebase Improvements

Findings from comprehensive static analysis of the entire codebase.

---

## Tier A: High-Impact Bug Fixes (very likely to get accepted)

These fix real bugs that affect users today.

### A1. Wrong log path in reset cleanup (copy-paste bug)
- **Files**: `phase/reset_controllers.go:146`, `phase/reset_leader.go:86`, `phase/reset_workers.go:137`
- **Issue**: Logs `K0sConfigPath()` when it should log `K0sBinaryPath()` — misleading error messages
- **Fix**: One-line fix per file

### A2. Missing `strings.TrimSpace()` on shell output
- **Files**: `configurer/linux.go` — `Arch()` (line 89), `SystemTime()` (line 294), `MachineID()` (line 289), `TempFile()`/`TempDir()` (line 120)
- **Issue**: Shell output includes trailing newline. `Arch()` returns `"x86_64\n"` which won't match the switch, `SystemTime()` fails `strconv.ParseInt`
- **Fix**: Add `strings.TrimSpace()` before processing

### A3. Nil pointer in `ResolveConfigurer()` error path
- **File**: `pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster/host.go:248-255`
- **Issue**: If type assertion fails, code still calls `h.Configurer.SetPath()` on nil Configurer before returning error
- **Fix**: Move `SetPath` call inside the success branch

### A4. HTTP response body leak in GitHub client
- **File**: `integration/github/github.go:92-112`
- **Issue**: If `io.ReadAll()` fails, response body is never closed
- **Fix**: `defer resp.Body.Close()` immediately after successful Get()

### A5. Silent failure in daemon reload
- **File**: `phase/daemon_reload.go:26-33`
- **Issue**: Failed daemon reload only logs warning, returns nil — subsequent service operations fail with confusing errors
- **Fix**: Return the error

### A6. Unsafe type assertions (panic risk)
- **Files**: `cmd/apply.go:126`, `cmd/backup.go:76,81`, `cmd/reset.go:31,36`, `cmd/kubeconfig.go:46,53`, `cmd/config_edit.go:25`
- **Issue**: `ctx.Context.Value(...).(type)` without comma-ok check — panics if context value missing
- **Fix**: Add `ok` check with fallback

---

## Tier B: Security Fixes (likely to get accepted)

### B1. Command injection in `FileContains()`
- **File**: `configurer/linux.go:162-165`
- **Issue**: `grep -q "%s" "%s"` — double quotes don't protect against `$(...)` or backticks
- **Fix**: Use `shellescape.Quote()` for both parameters

### B2. Unquoted interface name in `PrivateAddress()`
- **File**: `configurer/linux.go:230-256`
- **Issue**: `iface` parameter interpolated directly into shell command
- **Fix**: Use `shellescape.Quote(iface)`

### B3. Ignored survey confirmation error in reset
- **File**: `action/reset.go:32`
- **Issue**: `_ = survey.AskOne(prompt, &confirmed)` — if prompt fails, destructive reset may proceed without confirmation
- **Fix**: Check and return the error

### B4. Unchecked file permission parsing
- **File**: `phase/uploadfiles.go:188`
- **Issue**: `strconv.ParseUint(f.PermString, 8, 32)` error ignored — files created with wrong permissions
- **Fix**: Return error on invalid permission string

---

## Tier C: Upgrade Safety (high impact, medium acceptance — may need RFC/discussion)

### C1. Kubernetes version skew validation
- **Issue**: No check that controllers and workers stay within K8s N-2 minor version skew policy during rolling upgrades
- **Current**: Only checks target > current, no cross-node version comparison
- **Fix**: Add validation in `ValidateFacts` or `UpgradeControllers` that all nodes remain within acceptable skew
- **Closes**: Would address a class of silent cluster breakage

### C2. No health check between controller upgrade batches
- **File**: `phase/upgrade_controllers.go:162-189`
- **Issue**: Individual node readiness is checked, but no cluster-wide health validation (etcd quorum, API server) between batches
- **Fix**: Add etcd member health check and API server availability check between batches

### C3. No rollback on failed upgrade
- **Issue**: If upgrade fails after binary replacement, old binary is gone. No automatic recovery.
- **Fix**: Keep backup of old binary, restore on failed health check

### C4. Reset phases silently swallow critical errors
- **Files**: `phase/reset_controllers.go:89-91,102-103,109-110`, `phase/reset_leader.go:71-75`, `phase/reset_workers.go:124-126`
- **Issue**: Failed drain, etcd leave, k0s reset, node delete — all only log warnings, return nil
- **Fix**: Return errors for critical operations, allow non-critical cleanup to warn

---

## Tier D: Robustness Improvements (good quality PRs)

### D1. Missing retry logic for HTTP binary downloads
- **File**: `phase/download_binaries.go:148-194`
- **Issue**: Single HTTP request with no retry on transient failures
- **Fix**: Add retry with exponential backoff

### D2. Cleanup errors silently ignored everywhere
- **Files**: `phase/download_k0s.go:98`, `phase/upload_k0s.go:99`, `phase/install_binaries.go:88`, `phase/disconnect.go:23`
- **Issue**: `_ = h.Configurer.DeleteFile(...)` — temp files accumulate silently
- **Fix**: Log warnings on cleanup failures

### D3. `context.Background()` used instead of passed context
- **Files**: `phase/backup.go:37,45`, `phase/initialize_k0s.go:42,50`, `phase/disconnect.go:28`
- **Issue**: Operations can't be cancelled during graceful shutdown
- **Fix**: Pass actual context through

### D4. Lock file race condition
- **File**: `phase/lock.go:70-99`
- **Issue**: `p.cfs` slice accessed by goroutines while `Cancel()` may be called
- **Fix**: Use `sync.WaitGroup` and ensure goroutines stop before Cancel returns

### D5. Race condition on `numRunning` counter
- **File**: `phase/install_controllers.go:170,204`
- **Issue**: `p.numRunning` incremented without atomic operations
- **Fix**: Use `atomic.AddInt32` or mutex

### D6. Port number not validated in `init` command
- **File**: `cmd/init.go:85-91`
- **Issue**: No range check (1-65535), invalid port silently falls back to default
- **Fix**: Validate and return clear error

### D7. Controller count not validated
- **File**: `cmd/init.go:166-170`
- **Issue**: Accepts negative numbers or unreasonably large values
- **Fix**: Validate > 0

---

## Tier E: Nice-to-Have / Code Quality

### E1. Missing tests for `internal/shell/split.go`
### E2. Inconsistent error wrapping across configurer/ (linux vs windows)
### E3. String-based SSH host key error detection (`phase/connect.go:30`) — fragile
### E4. Kubeconfig written to stdout without sensitivity warning
### E5. Backup flow lacks intermediate progress logging
### E6. Token data not securely cleared from memory after use
### E7. Evict-taint format validation is incomplete (`cmd/apply.go:98-108`)
### E8. Stdin reading error handling dead code in `cmd/init.go:192-200`

---

## Summary

| Tier | Count | Theme |
|------|-------|-------|
| A | 6 | Real bugs affecting users today |
| B | 4 | Security vulnerabilities |
| C | 4 | Upgrade safety gaps |
| D | 7 | Robustness and reliability |
| E | 8 | Code quality and polish |
| **Total** | **29** | |
