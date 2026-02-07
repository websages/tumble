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
| High | 4 | 2 fixed, 2 pending |
| Medium | 4 | All pending |
| Low | 3 | 1 fixed, 2 pending |

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
**Status:** PENDING
**File:** `internal/handler/irclink.go:206-213`

**Issue:** The redirect endpoint trusts database values without validation:
```go
redirectURL, err := h.Store.GetIRCLinkURL(ctx, id)
http.Redirect(w, r, redirectURL, http.StatusFound)
```

**Recommendation:** Validate redirect URLs are within acceptable domains or use URL allowlist.

---

### 5. XSS Vulnerabilities in Templates
**Severity:** HIGH
**Status:** PENDING
**File:** `internal/templates/views/index.html` (lines 56, 122, 171, 220, 267, 272, 281-282, 314, 352)

**Issues:**
1. Direct HTML injection in onerror handlers with `.URL`
2. User-controlled content in href attributes (`poster` parameter)
3. Unsafe OEmbed HTML insertion via `innerHTML`

**Recommendations:**
- Use `textContent` instead of `innerHTML` where possible
- Proper HTML entity encoding for all template variables
- Implement Content Security Policy headers
- Validate and sanitize HTML from external sources

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
**Status:** PENDING
**File:** `conf/config.yaml:3-4`

**Issue:** Configuration file contains plaintext database credentials.

**Recommendation:**
- Use environment variables for sensitive values
- Use `.env` files with `.env.example` template
- Implement secret management system

---

### 8. No Rate Limiting
**Severity:** MEDIUM
**Status:** PENDING

**Issues:**
- No rate limiting on any endpoints
- `/search` endpoint reads all matching results without pagination limits
- No request size limits on form submissions

**Recommendation:** Implement middleware for:
- Request rate limiting (e.g., `golang.org/x/time/rate`)
- Request size limits
- Query timeout limits

---

### 9. Inadequate Input Validation
**Severity:** MEDIUM
**Status:** PENDING
**File:** `internal/handler/handlers.go:159,160,168,374-376`

**Issues:**
- Poster name accepted without validation and embedded in HTML
- Filter type used in database queries without validation
- URL parameters not length-checked

**Recommendation:**
- Implement allowlist validation for all string inputs
- Set maximum length limits (100 chars for username, 500 for URLs)
- Use regex validation for expected formats

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
**Status:** PENDING
**File:** `internal/templates/views/index.html:313-331`

**Issue:** OEmbed HTML from external providers is inserted and scripts are executed without inspection.

**Recommendation:**
- Use iframe sandboxing instead of direct HTML injection
- Parse and validate HTML before insertion
- Implement Content Security Policy to restrict script execution

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
5. **HIGH:** Sanitize all user inputs before template rendering
6. **HIGH:** Implement rate limiting middleware
7. **HIGH:** Use iframe sandboxing for OEmbed content
8. **MEDIUM:** Move credentials to environment variables
9. **MEDIUM:** Add input validation for all parameters
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
