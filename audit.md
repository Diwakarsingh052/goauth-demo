# Project Audit

Audit of the codebase against every requirement in the challenge specification.

---

## 1 — Functionalities

### 1-A. Sign-Up

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | Sign up with Google | PASS | `web/handler/auth.go` GoogleLogin + GoogleCallback, OAuth2 flow with `mode=signup` |
| 2 | Sign up with email and password | PASS | `web/handler/auth.go` SignupSubmit → API `/api/auth/signup` |
| 3 | Google auth registers with Google's basic info (name) | PASS | GoogleCallback reads `name` from Google userinfo endpoint, passes to `CreateGoogleUser` which stores it as `full_name` |
| 4 | Username is email only | PASS | All forms use email field, DB schema uses `email VARCHAR(255) UNIQUE` |
| 5 | Google auth integration works solidly | PASS | Full OAuth2 flow with CSRF state validation, token exchange, userinfo fetch, and proper error handling on every step |
| 6 | After Google signup → proceed to 2-C | PASS | GoogleCallback redirects to `/profile/edit` when `mode=signup` |
| 7 | After email signup → proceed to 2-C | PASS | SignupSubmit redirects to `/profile/edit` |

### 1-B. Profile Info, Setup, or Edit

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | After registration, user proceeds to enter profile info (2-C) | PASS | Both signup paths redirect to `/profile/edit` |
| 2 | After saving profile info, directed to main profile (2-D) | PASS | EditProfileSubmit redirects to `/profile` |
| 3 | Edit button on 2-D directs to 2-C | PASS | `profile_view.html` has Edit link to `/profile/edit` |
| 4 | Previously populated data fetched into fields | PASS | EditProfile handler fetches profile via API, template uses `value="{{.User.FullName}}"` etc. |
| 5 | Cancel button directs back to 2-D | PASS | `profile_edit.html` Cancel links to `/profile` |
| 6 | Existing user login directs to 2-D | PASS | LoginSubmit redirects to `/profile` |
| 7 | Editing email replaces the registered email/username | PASS | `UpdateProfile` API handler updates email in DB; future login requires new email |
| 8 | Google auth email cannot be edited (shown but disabled) | PASS | Template disables input with `{{if .IsGoogle}}disabled{{end}}`, hidden input preserves value, server-side enforced in `profile.go:86` overriding any tampered request |

### 1-C. Login/Authentication

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | Login with Google account | PASS | GoogleLogin handler with `mode=login` |
| 2 | Login with email and password | PASS | LoginSubmit handler → API `/api/auth/login` |
| 3 | After login, directed to 2-D | PASS | LoginSubmit and GoogleCallback (login mode) redirect to `/profile` |
| 4 | "username entered does not exist" message | PASS | `ErrUserNotFound` = `"username entered does not exist"`, API returns this exact string |
| 5 | "password is incorrect" message | PASS | `ErrInvalidPassword` = `"password is incorrect"`, API returns this exact string |
| 6 | Logout expires session, directed to 2-A | PASS | Logout handler clears session (MaxAge = -1), redirects to `/login` |
| 7 | Google user cannot login with email/password | PASS | `Authenticate` checks `AuthProvider == "google"` and returns `ErrGoogleAccount` before password check |
| 8 | Email/password user cannot login with Google | PASS | `GoogleLogin` API checks `AuthProvider == "local"` and returns `ErrLocalAccount` |

---

## 2 — UI / Pages, Links, and Buttons

### 2-A. Login Page

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | Login page is the home/starting page | PASS | Root `/` redirects to `/login` |
| 2 | Username (email) and password fields shown | PASS | `login.html` has email and password inputs |
| 3 | Google login button | PASS | "Sign in with Google" button with Google SVG icon |
| 4 | Link to sign-up for new users | PASS | "New user? Sign Up" link |

### 2-B. Sign-Up Page

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | Username (email) and password fields | PASS | `signup.html` has email and password inputs |
| 2 | Google sign-up button | PASS | "Sign up with Google" button |
| 3 | "Existing User Login" link back to 2-A | PASS | "Existing user? Login" link to `/login` |

### 2-C. Enter Profile Information

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | Three fields: full name, telephone, email | PASS | `profile_edit.html` has all three fields |
| 2 | Single entry line/field each | PASS | All use `<input>` elements (single line) |
| 3 | "Save & Continue" button | PASS | `Save & Continue` submit button |
| 4 | "Cancel" button | PASS | Cancel link styled as button |
| 5 | Save directs to 2-D | PASS | EditProfileSubmit redirects to `/profile` |
| 6 | Cancel directs to 2-D | PASS | Cancel links to `/profile` |

### 2-D. Main Profile

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | Displays user's contact/entries in view/display mode | PASS | `profile_view.html` shows full name, telephone, email as read-only text |
| 2 | "Edit" button → 2-C | PASS | Edit link to `/profile/edit` |
| 3 | "Logout" button → session expires → 2-A | PASS | Logout form POSTs to `/logout`, which clears session and redirects to `/login` |

---

## 3 — Unit Testing

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | Unit tests present for core functionalities | PASS | 5 test files covering multiple areas |
| 2 | `handler/auth_test.go` | PASS | Tests `jsonResponse` and `jsonError` helpers |
| 3 | `handler/profile_test.go` | PASS | Tests `validatePhone` with valid, invalid, edge-case inputs |
| 4 | `middleware/auth_test.go` | PASS | Tests `GetUserID` context extraction and full `JWTAuth` middleware flow |
| 5 | `auth/token_test.go` | PASS | Tests `NewTokenService`, `GenerateToken`, `ValidateToken` (valid, expired, wrong-secret, malformed) |
| 6 | `users/user_test.go` | PASS | Tests `isDuplicateEntry` error detection |

---

## 4 — Security & Authentication

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | Cannot view 2-D or 2-C without authentication | PASS | `RequireAuth` middleware on all `/profile` and `/profile/edit` routes checks session token |
| 2 | Back-button protection (browser cache) | PASS | `SetNoCacheHeaders` sets `Cache-Control: no-cache, no-store, must-revalidate`, `Pragma: no-cache`, `Expires: 0` on every protected response |
| 3 | Direct URL paste protection | PASS | `RequireAuth` middleware redirects to `/login` if no valid session token |
| 4 | Re-authentication required after logout | PASS | `Clear` sets `MaxAge = -1` which expires the cookie, and wipes all session values |
| 5 | Already-authenticated users redirected away from login/signup | PASS | `RedirectIfAuth` middleware on GET `/login` and GET `/signup` redirects to `/profile` |
| 6 | OAuth CSRF protection | PASS | Random 32-byte state stored in session, validated on callback |
| 7 | Cookies are HttpOnly | PASS | `session.go:25` sets `HttpOnly: true` |
| 8 | SameSite cookie policy | PASS | `session.go:26` sets `SameSite: Lax` |
| 9 | Google email server-side enforcement | PASS | `profile.go:86` overrides email from request with DB email for Google users |

---

## 5 — Technical Specifications

### 5-A. Back-End REST API in Golang

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | REST API implemented in Go | PASS | `cmd/api/main.go` runs a separate HTTP server |
| 2 | RESTful endpoints | PASS | `POST /api/auth/signup`, `POST /api/auth/login`, `POST /api/auth/google/signup`, `POST /api/auth/google/login`, `GET /api/profile`, `PUT /api/profile` |

### 5-B. Database: MySQL

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | MySQL used as the database | PASS | `go-sql-driver/mysql` driver, `docker-compose.yml` runs `mysql:8.0` |
| 2 | Schema created via migration | PASS | `database/migrate.go` creates `users` table with `CREATE TABLE IF NOT EXISTS` |

### 5-C. Front-End: Golang

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | Server-rendered application | PASS | Go templates in `templates/` rendered by `handler.render()` |
| 2 | Communication through REST API | PASS | `web/client/client.go` makes HTTP calls to the API server |
| 3 | Go templates used for frontend | PASS | `html/template` package, `.html` template files |
| 4 | No JavaScript SPA or popular JS frameworks | PASS | No React/Angular/Vue/etc. |
| 5 | Vanilla JS only when needed | PASS | Only minimal inline JS for phone input digit filtering (`oninput` in `profile_edit.html`) |

### 5-D. No Golang Frameworks

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | No Go web frameworks (Gin, Echo, Fiber, etc.) | PASS | Only standard library + allowed 3rd-party libraries |
| 2 | Gorilla toolkit (allowed) | PASS | `gorilla/mux` for routing, `gorilla/sessions` for session management |
| 3 | Other 3rd-party libraries are appropriate | PASS | `golang-jwt/jwt` (JWT), `go-sql-driver/mysql` (DB driver), `joho/godotenv` (.env), `golang.org/x/crypto` (bcrypt), `golang.org/x/oauth2` (OAuth) — all are libraries, not frameworks |

### 5-E. Separate REST API

| # | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 1 | API is independent from frontend | PASS | Separate binary (`cmd/api`), separate Docker container, separate port |
| 2 | Changing frontend technology would not require API changes | PASS | Web server communicates via HTTP client (`web/client/client.go`); API has no knowledge of the frontend |
| 3 | API returns JSON | PASS | All responses use `Content-Type: application/json` |

---

## Issues Found

### FAIL — None

All functional requirements from the specification are implemented and verified.

### Observations (non-blocking, not required by spec)

| # | Observation | Severity | Detail |
|---|------------|----------|--------|
| 1 | `cmd/api/main.go:18` uses `%f` format verb for error | Low | Should be `%v` — `%f` is for floats and will print `%!f(*fs.PathError=...)` instead of the actual error message |
| 2 | No `make test` target in Makefile | Low | Tests exist and pass with `go test ./...` but there is no convenience Makefile target for it |
| 3 | `godotenv.Load` fatally exits if `.env` is missing | Low | In Docker, env vars are provided by docker-compose, but if `.env` file is absent in the container the app will crash on startup. Consider using `godotenv.Load` with a non-fatal fallback |
| 4 | CORS allows all origins (`*`) | Low | `router.go` sets `Access-Control-Allow-Origin: *` — acceptable for a challenge project but would need restricting in production |
| 5 | Session cookie missing `Secure` flag | Low | Cookie is sent over HTTP too — fine for local dev but should be `Secure: true` behind HTTPS in production |