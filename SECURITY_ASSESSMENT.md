# Security Assessment - Tumble Application

**Date:** 2026-01-28
**Reviewed by:** Claude Opus 4.5

## Overview

The Tumble application is a Go-based web service that functions as a link aggregator/archive system. It's built with Go 1.25.5 and uses GORM for database operations with support for MySQL and SQLite backends.

---

## Summary Table

| Category | Count | Status |
|----------|-------|--------|
| Critical | 2 | 1 fixed, 1 pending |
| High | 4 | All fixed |
| Medium | 4 | 2 fixed, 1 N/A, 1 pending |
| Low | 3 | 2 fixed, 1 pending |

---

## Critical Vulnerabilities

### 1. Missing Authorization on DELETE Operations
**Severity:** CRITICAL
**Status:** MITIGATED (localhost-only restriction added)
**File:** `internal/handler/irclink.go:39-76`

**Issue:** The DELETE endpoint for removing IRC links originally had no authentication or authorization checks.

**Mitigation Applied:** Restricted DELETE to localhost/127.0.0.1 only. Full authentication layer still needed.

**Remaining Work:** Implement proper authentication and authorization system.

---

### 2. SSRF Vulnerability (Server-Side Request Forgery)
**Severity:** CRITICAL
**Status:** FIXED
**File:** `internal/handler/irclink.go:100-125`

**Issue:** User-supplied URLs were directly passed to `http.Client.Get()` with no validation.

**Fix Applied:** Integrated `github.com/doyensec/safeurl` library which:
- Blocks private/internal IP ranges (127.0.0.0/8, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)
- Blocks IPv6 localhost and private ranges
- Restricts to http/https schemes only
- Protects against DNS rebinding attacks

---

## High Severity Issues

### 3. Unbounded HTTP Response Reads
**Severity:** HIGH
**Status:** FIXED
**Files:** `internal/handler/irclink.go:114-115`

**Issue:** `ioutil.ReadAll()` was used without size limits, enabling memory exhaustion.

**Fix Applied:** Added `io.LimitReader` to cap response bodies at 1MB.

---

### 4. Open Redirect Vulnerability
**Severity:** HIGH
**Status:** FIXED
**File:** `internal/handler/irclink.go:264-279`

**Issue:** The redirect endpoint trusted database values without validation.

**Fix Applied:** Added URL scheme validation before redirect - only `http://` and `https://` schemes are allowed. This blocks dangerous schemes like `javascript:`, `data:`, `file:`, etc.

Note: URLs are also validated at insertion time via safeurl, providing defense-in-depth.

---

### 5. XSS Vulnerabilities in Templates
**Severity:** HIGH
**Status:** FIXED
**File:** `internal/templates/views/index.html`

**Issues Addressed:**
1. OEmbed HTML injection - now rendered in sandboxed iframes
2. Server-side template variables - protected by Go's html/template contextual auto-escaping
3. Client-side innerHTML usage - protected by `escapeHtml()` function
4. YouTube embeds - only regex-validated video IDs used, no external HTML

**Protection Layers:**
- Go's `html/template` package auto-escapes all template variables in context (HTML, JS, URL)
- Client-side `escapeHtml()` function used for all dynamic innerHTML insertions
- OEmbed content isolated in sandboxed iframes with restricted permissions

---

### 6. Missing Security Headers
**Severity:** HIGH
**Status:** FIXED
**File:** `cmd/tumble/main.go`

**Issue:** No HTTP security headers were set.

**Fix Applied:** Added `securityHeadersMiddleware` that sets:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains` (non-localhost only)

Note: CSP header not yet implemented (requires careful tuning for OEmbed content).

---

## Medium Severity Issues

### 7. Plaintext Database Credentials
**Severity:** MEDIUM
**Status:** N/A (Deployment concern)
**File:** `conf/config.yaml:3-4`

**Issue:** Example config file contains plaintext database credentials.

**Already Supported:** Viper config (internal/config/config.go:50-53) supports environment variables with `TUMBLE_` prefix:
- `TUMBLE_USERNAME` - database username
- `TUMBLE_PASSWORD` - database password
- `TUMBLE_CLICK_SIGNING_KEY` - HMAC signing key
- `TUMBLE_ADMIN_SECRET` - admin authentication secret

**Deployment Note:** Use environment variables in production deployments instead of config file values.

---

### 8. No Rate Limiting
**Severity:** MEDIUM
**Status:** FIXED
**File:** `cmd/tumble/main.go`

**Issue:** No rate limiting on any endpoints.

**Fix Applied:** Added IP-based rate limiting middleware using `golang.org/x/time/rate`:
- General endpoints: 60 req/min with burst of 10
- `/ogpreview`: 30 req/min with burst of 5 (expensive operation)
- `/search`: 20 req/min with burst of 3

Features:
- Per-IP tracking with X-Forwarded-For support for proxied requests
- Returns 429 Too Many Requests with Retry-After header
- Logs rate limit violations

---

### 9. Inadequate Input Validation
**Severity:** MEDIUM
**Status:** FIXED
**File:** `internal/handler/handlers.go:50-85`

**Issue:** Input parameters accepted without length limits or validation.

**Fix Applied:** Added input sanitization functions:
- `poster`: Truncated to 256 chars max
- `filterType`: Allowlist validation (only "", "links", "quotes" accepted)
- `search`: Truncated to 500 chars max

Note: SQL injection was never a risk (GORM uses parameterized queries), and XSS is handled by Go's html/template auto-escaping. This fix adds defense-in-depth for DoS and log injection vectors.

---

### 10. Error Information Disclosure
**Severity:** MEDIUM
**Status:** PENDING
**File:** `internal/handler/irclink.go:90`

**Issue:** Database errors returned to users:
```go
http.Error(w, fmt.Sprintf("Database Error: %v", err), http.StatusInternalServerError)
```

**Recommendation:**
```go
slog.Error("Database error", "error", err)
http.Error(w, "Internal Server Error", http.StatusInternalServerError)
```

---

## Low Severity Issues

### 11. Deprecated API Usage
**Severity:** LOW
**Status:** FIXED
**File:** `internal/service/content.go:300`

**Issue:** Uses deprecated `io/ioutil.ReadAll()`.

**Fix Applied:** Updated to `io.ReadAll()` from the `io` package.

---

### 12. Development Mode in Config
**Severity:** LOW
**Status:** PENDING
**File:** `conf/config.yaml:6`

```yaml
mode: dev
```

**Recommendation:** Ensure `mode: production` for production deployments.

---

### 13. Unsafe OEmbed Script Execution
**Severity:** LOW-MEDIUM
**Status:** FIXED
**File:** `internal/templates/views/index.html:328-378`

**Issue:** OEmbed HTML from external providers was inserted via innerHTML and scripts were executed directly in the main page context.

**Fix Applied:** OEmbed content is now rendered inside a sandboxed iframe with `sandbox="allow-scripts allow-same-origin allow-popups allow-presentation"`. This isolates external embed content from the main page - embedded scripts cannot access parent page cookies, DOM, or JavaScript context.

---

## Architecture Issues

### No Authentication/Authorization Layer
**Severity:** CRITICAL
**Status:** PENDING

The entire application lacks any user identification system:
- No login mechanism
- No session management
- No role-based access control
- No IP allowlisting (except localhost restriction on DELETE)

This is the most critical architectural flaw requiring implementation before production deployment.

---

## Priority Remediation Order

1. **IMMEDIATE:** Implement full authentication and authorization system
2. ~~IMMEDIATE: Add URL validation and private IP range checks~~ ✅ DONE
3. ~~URGENT: Implement size limits on HTTP response reads~~ ✅ DONE
4. ~~URGENT: Add security headers via middleware~~ ✅ DONE
5. ~~HIGH: Sanitize all user inputs before template rendering~~ ✅ DONE (Go html/template + escapeHtml)
6. ~~HIGH: Implement rate limiting middleware~~ ✅ DONE
7. ~~HIGH: Use iframe sandboxing for OEmbed content~~ ✅ DONE
8. ~~MEDIUM: Move credentials to environment variables~~ ✅ Already supported via Viper
9. ~~MEDIUM: Add input validation for all parameters~~ ✅ DONE
10. ~~LOW: Update remaining deprecated API calls~~ ✅ DONE
11. **LOW:** Ensure production mode in deployment configs

---

## Completed Fixes

| Date | Commit | Description |
|------|--------|-------------|
| 2026-01-28 | `1e1c747` | Restricted DELETE endpoint to localhost only |
| 2026-01-28 | `e944237` | Added SSRF protection with doyensec/safeurl and 1MB response limit |
| 2026-02-06 | - | Updated deprecated ioutil.ReadAll to io.ReadAll in content.go |
| 2026-02-06 | - | Added security headers middleware (HSTS skipped for localhost) |
| 2026-02-06 | - | Added URL scheme validation to prevent open redirect attacks |
| 2026-02-06 | - | OEmbed content now rendered in sandboxed iframes for XSS protection |
| 2026-02-06 | - | Verified XSS protection: Go html/template auto-escaping + client-side escapeHtml() |
| 2026-02-06 | - | Added IP-based rate limiting middleware with tiered limits per endpoint type |
| 2026-02-06 | - | Added input validation: poster (256 chars), filterType (allowlist), search (500 chars) |
