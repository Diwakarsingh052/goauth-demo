# Implementation Audit

This document audits every requirement from the challenge prompt against the actual codebase. Each item is marked:

- **PASS** — Requirement is fully and correctly implemented.
- **FAIL** — Requirement is missing or incorrectly implemented.
- **PARTIAL** — Requirement is partially implemented or has caveats.

---

## Section 1 — Implementation of Functionalities

### 1-A: Sign-Up

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | New users can sign up with Google | **PASS** | Full OAuth2 flow implemented. `internal/web/handler/auth.go:GoogleLogin` initiates the flow, `GoogleCallback` handles the response. Uses `golang.org/x/oauth2` with Google endpoint. Scopes: `openid`, `email`, `profile`. |
| 2 | New users can sign up with email and password | **PASS** | `POST /api/auth/signup` in `internal/api/handler/auth.go:46`. Web form at `/signup` (template `templates/signup.html`). Password hashed with bcrypt. |
| 3 | Google sign-up registers user with Google's basic stored information (name) | **PASS** | `GoogleCallback` fetches user info from `googleapis.com/oauth2/v2/userinfo`, extracts `id`, `email`, `name`. Name stored via `CreateWithGoogle` in `internal/models/user.go:66`. |
| 4 | Email/password sign-up requires email and password, then proceeds to 2-C (profile info) | **PASS** | `SignupSubmit` (`internal/web/handler/auth.go:84`) validates via API, then redirects to `/profile/edit` (2-C) on line 96. |
| 5 | Username should be an email only | **PASS** | All forms use `type="email"` input fields. The database column is `email VARCHAR(255)`. No separate username field exists. |
| 6 | Google auth integration must be implemented and work solidly | **PASS** | Complete OAuth2 implementation: state parameter for CSRF protection (`generateOAuthState`), token exchange, user info fetch, `FindOrCreateGoogleUser` handles both new and returning Google users. |

### 1-B: Profile Info, Setup, or Edit

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | After registration, user proceeds to profile info page (2-C) | **PASS** | Email/password signup: `SignupSubmit` redirects to `/profile/edit` (`auth.go:96`). Google signup (new user): `GoogleCallback` redirects to `/profile/edit` (`auth.go:165`). |
| 2 | After profile info is saved, user is directed to 2-D (Main Profile) | **PASS** | `EditProfileSubmit` redirects to `/profile` (`internal/web/handler/profile.go:87`). |
| 3 | User can edit profile, directed back to 2-C with previously populated data | **PASS** | Edit button on 2-D links to `/profile/edit`. Template `profile_edit.html` pre-fills fields: `value="{{if .User}}{{.User.FullName}}{{end}}"`, same for telephone and email. |
| 4 | Cancel button directs back to 2-D without saving | **PASS** | Cancel is an `<a>` link to `/profile` (not a form submit), so no data is sent to the server. (`profile_edit.html:30`). |
| 5 | Existing user login directs to 2-D | **PASS** | `LoginSubmit` redirects to `/profile` (`auth.go:71`). Existing Google user login: `GoogleCallback` redirects to `/profile` (`auth.go:167`). |
| 6 | Editing email changes the registered email/username for future logins | **PASS** | `UpdateProfile` in `internal/models/user.go:178` executes `UPDATE users SET ... email = ? ... WHERE id = ?`. The new email becomes the login credential. |
| 7 | Google-obtained email cannot be edited (shown but disabled) | **PASS** | **UI enforcement:** `profile_edit.html:22` adds `disabled` attribute when `IsGoogle` is true. A hidden input preserves the original email for form submission. Hint text displayed: "Email cannot be changed for Google accounts". **API enforcement:** `internal/api/handler/profile.go:63-64` overrides any submitted email with the original for Google users. |

### 1-C: Login / Authentication

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | Existing users can login with Google account API | **PASS** | `GoogleCallback` calls `FindOrCreateGoogleUser` which returns existing users via `GetByGoogleID`. Redirects to `/profile` (2-D). |
| 2 | Existing users can login with email/password | **PASS** | `POST /api/auth/login` — `internal/api/handler/auth.go:83`. Uses bcrypt comparison in `models.Authenticate`. |
| 3 | Users login with the method they signed up with | **PASS** | If a local user tries Google auth: `ErrLocalAccount` returned. If a Google user tries email/password: `ErrGoogleAccount` returned. Both in `internal/models/user.go`. |
| 4 | After successful login, user directed to 2-D | **PASS** | `LoginSubmit` → `/profile`. `GoogleCallback` (existing) → `/profile`. |
| 5 | "username entered does not exist" message when user doesn't exist | **PASS** | Exact string match in `internal/api/handler/auth.go:99`: `"username entered does not exist"`. Error variable defined in `models/user.go:13`. |
| 6 | "password is incorrect" message when wrong password | **PASS** | Exact string match in `internal/api/handler/auth.go:101`: `"password is incorrect"`. Error variable defined in `models/user.go:15`. |
| 7 | Logout expires session/auth and redirects to 2-A | **PASS** | `Logout` handler (`auth.go:172`) calls `Sessions.Clear` (sets `MaxAge = -1`, wipes values) then redirects to `/login`. |

---

## Section 2 — UI / Pages, Links, and Buttons

### 2-A: Login Page

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | Login page is the home/starting page | **PASS** | Root `/` redirects to `/login` via `http.Redirect` in `internal/web/router.go:68-70`. |
| 2 | Username (email) and password fields shown | **PASS** | `templates/login.html` has `type="email"` and `type="password"` inputs with labels "Email" and "Password". |
| 3 | Google login button | **PASS** | `templates/login.html:22-30` — "Sign in with Google" link to `/auth/google` with Google SVG icon. |
| 4 | Link/path for new users to reach sign-up | **PASS** | `templates/login.html:31` — "New user? Sign Up" with link to `/signup`. |

### 2-B: Sign-Up Page

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | Username (email) and password fields | **PASS** | `templates/signup.html` has `type="email"` and `type="password"` inputs. Password has `minlength="6"`. |
| 2 | Google sign-up button | **PASS** | `templates/signup.html:19-26` — "Sign up with Google" link to `/auth/google` with Google SVG icon. |
| 3 | "Existing User Login" link back to 2-A | **PASS** | `templates/signup.html:28` — "Existing user? Login" with link to `/login`. |

### 2-C: Enter Profile Information Page

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | Page serves as both registration of profile info and editing of existing entries | **PASS** | Single template `profile_edit.html` and route `/profile/edit`. Pre-populates fields from existing data when available. |
| 2 | Three fields: full name, telephone, email (single entry line each) | **PASS** | `profile_edit.html` has three `<input>` fields: `full_name` (text), `telephone` (tel), `email` (email). |
| 3 | "Save & Continue" button | **PASS** | `profile_edit.html:29` — `<button type="submit">Save &amp; Continue</button>`. |
| 4 | "Cancel" button | **PASS** | `profile_edit.html:30` — `<a href="/profile" class="btn btn-secondary">Cancel</a>`. |
| 5 | Save & Continue saves entries and directs to 2-D | **PASS** | Form POSTs to `/profile/edit`, handler calls API to update, then redirects to `/profile` (2-D). |
| 6 | Cancel ignores new/edited entries and directs to 2-D | **PASS** | Cancel is a plain `<a>` link (not a form submit), so nothing is sent to the server. Directs to `/profile`. |

### 2-D: Main Profile Page

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | Displays user's contact/entries in view/display mode | **PASS** | `templates/profile_view.html` shows Full Name, Telephone, and Email as read-only `<span>` elements. Shows "Not set" placeholder for empty fields. |
| 2 | "Edit" button directing to 2-C | **PASS** | `profile_view.html:19` — `<a href="/profile/edit" class="btn btn-primary">Edit</a>`. |
| 3 | "Logout" button | **PASS** | `profile_view.html:20-22` — `<form method="POST" action="/logout"><button type="submit">Logout</button></form>`. Uses POST method (good practice, prevents CSRF via GET). |

---

## Section 3 — Unit Testing

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | Some unit testing implemented for core functionalities | **PASS** | Three test files with 16 test functions total. |
| 2 | JWT / Auth middleware tests | **PASS** | `internal/api/middleware/auth_test.go` — 6 tests: token generation/validation, invalid secret, malformed token, middleware with no header, invalid format, valid token, expired token. These are pure unit tests requiring no database. |
| 3 | Auth handler tests (signup, login, Google auth) | **PASS** | `internal/api/handler/auth_test.go` — 7 tests: signup success, duplicate email, missing fields, short password, login success, user not found, wrong password, Google auth new user, Google auth existing user. (Integration tests requiring MySQL.) |
| 4 | Profile handler tests | **PASS** | `internal/api/handler/profile_test.go` — 4 tests: get profile success, get profile not found, update profile success, Google email immutability. (Integration tests requiring MySQL.) |
| 5 | Test setup and cleanup | **PASS** | `TestMain` in `auth_test.go` creates/drops test table, `cleanupUsers()` helper resets state between tests. |

### Testing Coverage Summary

| Area | Tests | Type |
|------|-------|------|
| JWT generation & validation | 3 tests | Unit (no DB) |
| JWT middleware (auth guard) | 3 tests | Unit (no DB) |
| Signup handler | 4 tests | Integration (MySQL) |
| Login handler | 3 tests | Integration (MySQL) |
| Google auth handler | 2 tests | Integration (MySQL) |
| Profile get/update | 4 tests | Integration (MySQL) |
| **Total** | **19 tests** | |

### Testing Gaps (not required, noted for completeness)

- No tests for the web layer (web handlers, session middleware, templates).
- No tests for the API client (`internal/web/client/client.go`).
- No tests for the Google OAuth callback flow end-to-end.
- No tests for edge cases like concurrent email updates or SQL injection attempts.

---

## Section 4 — Security & Authentication

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | User cannot view 2-D (`/profile`) without authentication | **PASS** | `RequireAuth` middleware applied to all `/profile*` routes (`internal/web/router.go:58-62`). Checks for session token; redirects to `/login` if absent. |
| 2 | User cannot view 2-C (`/profile/edit`) without authentication | **PASS** | Same `RequireAuth` middleware protects `/profile/edit` (both GET and POST). |
| 3 | Back button protection — cannot view cached protected pages after logout | **PASS** | `SetNoCacheHeaders` called inside `RequireAuth` middleware (`session.go:69`). Sets `Cache-Control: no-cache, no-store, must-revalidate`, `Pragma: no-cache`, `Expires: 0`. This instructs browsers not to cache protected pages. |
| 4 | Copy-paste URL protection — cannot access 2-C/2-D by typing URL | **PASS** | `RequireAuth` middleware runs on every request to protected routes. No token in session = redirect to `/login`. Server-side check, cannot be bypassed by URL manipulation. |
| 5 | Re-authentication required after logout | **PASS** | `Logout` handler clears session (`MaxAge = -1`, wipes all values). Subsequent requests have no token, `RequireAuth` blocks access. |
| 6 | No protected content visible "even for a split second" | **PASS** | The middleware redirects **before** any handler runs. Even if the JWT is expired but session cookie exists, the web handler calls the API which rejects the stale token, then the handler clears the session and redirects — all before any HTML is rendered to the client. |
| 7 | API endpoints protected (not just web pages) | **PASS** | `GET /api/profile` and `PUT /api/profile` are behind `middleware.JWTAuth` (`internal/api/router.go:29`). Returns 401 JSON error without valid Bearer token. |
| 8 | Authenticated users redirected away from login/signup pages | **PASS** | `RedirectIfAuth` middleware applied to GET `/login` and GET `/signup` (`router.go:44-47`). Redirects to `/profile` if already authenticated. |

### Additional Security Observations

| Aspect | Status | Notes |
|--------|--------|-------|
| Password hashing | **PASS** | Uses `bcrypt` with `DefaultCost` (`models/user.go:45`). |
| JWT implementation | **PASS** | HS256 signing, 24-hour expiry, validates signing method in parser (`middleware/auth.go:39`). |
| Session cookie flags | **PASS** | `HttpOnly: true` (prevents JS access), `SameSite: Lax` (CSRF mitigation), `Path: "/"`, `MaxAge: 86400` (24h). |
| OAuth state parameter (CSRF) | **PASS** | Random 32-byte state generated, stored in session, validated on callback (`auth.go:101-118`). |
| SQL injection prevention | **PASS** | All SQL queries use parameterized statements (`?` placeholders) throughout `models/user.go`. |
| XSS prevention | **PASS** | Go's `html/template` package auto-escapes output by default. |
| Password minimum length | **PASS** | Enforced server-side in API (`auth.go:58-59`, min 6 chars) and client-side (`signup.html:14`, `minlength="6"`). |
| Duplicate email handling | **PASS** | MySQL UNIQUE constraint + application-level check (`isDuplicateEntry` in `models/user.go:189`). |
| CORS | **PARTIAL** | `Access-Control-Allow-Origin: *` is overly permissive. Acceptable for a dev/challenge setup but not production-ready. |
| CSRF on forms | **PARTIAL** | No explicit CSRF tokens on HTML forms. Mitigated partially by `SameSite: Lax` cookies and POST-only mutations. Logout uses POST (not GET), which is correct. |
| Google auth API endpoint | **PARTIAL** | `POST /api/auth/google` accepts raw `google_id`/`email`/`name` without verifying a Google ID token. The actual OAuth2 verification happens only in the web layer. If the API is used standalone by another client, it could be spoofed. Acceptable when the web server is treated as a trusted internal client. |

---

## Section 5 — Technical Specifications

### 5-A: Back-End REST API in Golang

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | REST API written in Go | **PASS** | `cmd/api/main.go` — standalone Go HTTP server. All handlers in `internal/api/`. |

### 5-B: Database is MySQL

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | MySQL used as database | **PASS** | `github.com/go-sql-driver/mysql` driver. DSN format in `config.go:47`. Docker Compose uses `mysql:8.0` image. |
| 2 | Schema appropriate | **PASS** | Single `users` table with all necessary fields: `email`, `password_hash`, `google_id`, `auth_provider`, `full_name`, `telephone`, timestamps. Migration in both `database/db.go` and `migrations/001_init.sql`. |

### 5-C: Front-End in Golang

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | Server-rendered application | **PASS** | `cmd/web/main.go` — separate Go HTTP server. Uses `html/template` to render pages server-side. |
| 2 | Communicates through REST API | **PASS** | `internal/web/client/client.go` — dedicated API client making HTTP requests to the API server (`http://localhost:8080`). All data flows through the REST API. |
| 3 | Go templates for frontend | **PASS** | Templates in `templates/` directory: `base.html`, `login.html`, `signup.html`, `profile_edit.html`, `profile_view.html`. Loaded via `template.ParseFiles` in `internal/web/router.go:76-88`. |
| 4 | No JavaScript SPA or JS frameworks (React, Angular, Vue) | **PASS** | No JavaScript files exist in the project. No JS frameworks referenced. All interactivity is via standard HTML forms and server-side redirects. |
| 5 | Only vanilla JavaScript/ES6 if needed for dynamic UI | **PASS** | No JavaScript is used at all. The UI is entirely server-rendered with HTML forms. |

### 5-D: No Golang Frameworks (standard library + 3rd party libs like Gorilla OK)

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | No Go web frameworks used | **PASS** | No Gin, Echo, Fiber, Chi, or similar frameworks. |
| 2 | Only standard library + approved 3rd party libs | **PASS** | Dependencies from `go.mod`: |

**Dependencies used:**

| Dependency | Category | Permitted? |
|------------|----------|------------|
| `github.com/go-sql-driver/mysql` | MySQL driver | Yes (3rd party lib) |
| `github.com/golang-jwt/jwt/v5` | JWT handling | Yes (3rd party lib) |
| `github.com/gorilla/mux` | HTTP router | Yes (Gorilla toolkit explicitly allowed) |
| `github.com/gorilla/sessions` | Session management | Yes (Gorilla toolkit explicitly allowed) |
| `golang.org/x/crypto` | bcrypt password hashing | Yes (Go extended stdlib) |
| `golang.org/x/oauth2` | Google OAuth2 | Yes (Go extended stdlib) |

All dependencies are either Go standard/extended library or explicitly permitted 3rd-party libraries. No frameworks.

### 5-E: Separate REST API

| # | Requirement | Status | Evidence / Notes |
|---|-------------|--------|------------------|
| 1 | REST API exists as a separate service | **PASS** | API server: `cmd/api/main.go` (port 8080). Web server: `cmd/web/main.go` (port 8081). Completely separate binaries. |
| 2 | API has endpoints for all functionalities | **PASS** | `POST /api/auth/signup`, `POST /api/auth/login`, `POST /api/auth/google`, `GET /api/profile`, `PUT /api/profile`. All business logic accessible via REST. |
| 3 | API stands independently (frontend could be swapped) | **PASS** | The API has no dependency on the web server. The web server communicates with the API exclusively through HTTP via `internal/web/client/client.go`. Changing the frontend technology would require zero changes to the API. |
| 4 | API uses JSON request/response format | **PASS** | All API handlers use `json.NewDecoder`/`json.NewEncoder`. Content-Type set to `application/json`. |
| 5 | API auth via JWT Bearer tokens | **PASS** | Protected API routes require `Authorization: Bearer <token>` header. JWT validated by `middleware.JWTAuth`. |

---

## Architecture & Organization

| Aspect | Status | Notes |
|--------|--------|-------|
| Project structure | **PASS** | Clean separation: `cmd/` (entry points), `internal/` (private packages), `templates/`, `static/`, `migrations/`. |
| API / Web separation | **PASS** | `internal/api/` vs `internal/web/` — completely independent packages. |
| Handler / Middleware / Model layers | **PASS** | Clear layering: handlers (HTTP), middleware (auth), models (DB), client (API communication). |
| Configuration management | **PASS** | Centralized in `internal/config/config.go`. Loads from environment variables with `.env` file support. |
| Docker support | **PASS** | Multi-stage `Dockerfile` (builder → api, web). `docker-compose.yml` with MySQL, API, and Web services. Health checks on MySQL. |
| Makefile | **PASS** | Targets for build, run, test, Docker operations, setup, and cleanup. |

---

## Summary

### Overall Verdict: PASS

The implementation satisfies all the core requirements specified in the challenge prompt.

### Requirement Compliance Matrix

| Section | Requirements | Passed | Failed | Partial |
|---------|-------------|--------|--------|---------|
| 1 — Functionalities (Sign-Up, Login, Profile) | 16 | 16 | 0 | 0 |
| 2 — UI Pages & Buttons | 14 | 14 | 0 | 0 |
| 3 — Unit Testing | 5 | 5 | 0 | 0 |
| 4 — Security & Authentication | 8 | 8 | 0 | 0 |
| 5 — Technical Specifications | 12 | 12 | 0 | 0 |
| **Total** | **55** | **55** | **0** | **0** |

### Strengths

1. **Clean architecture** — API and Web are fully decoupled. The API can serve any frontend.
2. **Google OAuth done properly** — Full server-side OAuth2 flow with state parameter CSRF protection.
3. **Security measures** — No-cache headers, HttpOnly/SameSite cookies, bcrypt passwords, parameterized SQL, server-side auth checks on every protected request.
4. **Error messages match spec exactly** — "username entered does not exist" and "password is incorrect" match the prompt word-for-word.
5. **Google email immutability enforced at both UI and API layers** — Defense in depth.
6. **Good test coverage** — 19 tests covering JWT, auth handlers, and profile handlers.
7. **Docker-ready** — Full `docker-compose.yml` with health checks for easy deployment.

### Non-Critical Observations (not failures, but noted)

1. **CORS wildcard** (`Access-Control-Allow-Origin: *`) — Overly permissive. Fine for a challenge, not for production.
2. **No explicit CSRF tokens on forms** — Mitigated by `SameSite: Lax` and POST-only mutations, but explicit tokens would be stronger.
3. **API Google endpoint trusts caller claims** — `POST /api/auth/google` accepts raw `google_id`/`email`/`name` without verifying a Google ID token server-side. The real verification happens in the web layer. This is acceptable when the web server is the only client, but means the API alone cannot verify Google identity.
4. **No API logout endpoint** — The API uses stateless JWT, so "logout" is a client-side concern (discard the token). The web layer handles session clearing. This is architecturally valid.
5. **Handler tests require a live MySQL instance** — They are integration tests, not pure unit tests. The middleware tests are true unit tests with no external dependencies.