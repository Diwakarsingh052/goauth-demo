# Session & Authentication Audit Report

**Repository:** `challange-go-cyaz`  
**Date:** 2025-07-15  
**Scope:** Complete analysis of session management, authentication layers, middleware pipeline, and security posture.

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [System Architecture Overview](#2-system-architecture-overview)
3. [Authentication Layer 1: JWT (API Server)](#3-authentication-layer-1-jwt-api-server)
4. [Authentication Layer 2: Cookie Sessions (Web Server)](#4-authentication-layer-2-cookie-sessions-web-server)
5. [Authentication Layer 3: Google OAuth 2.0](#5-authentication-layer-3-google-oauth-20)
6. [Middleware Pipeline Analysis](#6-middleware-pipeline-analysis)
7. [Session Lifecycle: Full Walkthrough](#7-session-lifecycle-full-walkthrough)
8. [How Sessions Are Stored](#8-how-sessions-are-stored)
9. [How Tokens Flow Between Layers](#9-how-tokens-flow-between-layers)
10. [Route Protection Matrix](#10-route-protection-matrix)
11. [Security Audit Findings](#11-security-audit-findings)
12. [Configuration & Secrets Management](#12-configuration--secrets-management)
13. [Key File Reference](#13-key-file-reference)
14. [Summary of Concerns and Observations](#14-summary-of-concerns-and-observations)

---

## 1. Executive Summary

This application uses a **dual-layer authentication architecture** across two independent Go servers:

| Layer | Location | Mechanism | Storage |
|---|---|---|---|
| **Layer 1 — JWT** | API Server (`:8080`) | Stateless HMAC-SHA256 signed tokens | None (stateless) |
| **Layer 2 — Cookie Session** | Web Server (`:8081`) | `gorilla/sessions` CookieStore | Encoded in the browser cookie itself |
| **Layer 3 — Google OAuth 2.0** | Web Server (`:8081`) | Standard OAuth2 authorization code flow | OAuth state stored temporarily in session |

The web server acts as a **Backend-for-Frontend (BFF)**. It never talks to the database directly. Instead, it stores the API-issued JWT inside a cookie-based session, then replays that JWT to the API on every request. The browser never sees or handles the JWT directly.

**There is no server-side session store.** Both layers are client-side: the JWT lives inside the cookie, and the cookie itself is the session store (gorilla `CookieStore` serializes all session data into the cookie value). There is no Redis, no database session table, no in-memory session map.

---

## 2. System Architecture Overview

```
 Browser
    │
    │ HTTP (port 8081)
    │ Cookie: session=<encoded JWT + session data>
    │
    ▼
 ┌────────────────────────────────────────────┐
 │           WEB SERVER (cmd/web)             │
 │                                            │
 │  gorilla/sessions.CookieStore              │
 │   ├── reads JWT from cookie                │
 │   ├── stores JWT into cookie               │
 │   └── stores OAuth state in cookie         │
 │                                            │
 │  Middleware:                                │
 │   ├── RequireAuth   (protected routes)     │
 │   └── RedirectIfAuth (public routes)       │
 │                                            │
 │  Handlers:                                 │
 │   ├── AuthWebHandler  (login/signup/oauth) │
 │   └── ProfileWebHandler (profile pages)    │
 └──────────────────┬─────────────────────────┘
                    │
                    │ HTTP/JSON (port 8080)
                    │ Authorization: Bearer <JWT>
                    ▼
 ┌────────────────────────────────────────────┐
 │           API SERVER (cmd/api)             │
 │                                            │
 │  Middleware:                                │
 │   ├── corsMiddleware (all routes)          │
 │   └── JWTAuth       (protected routes)    │
 │                                            │
 │  Handlers:                                 │
 │   ├── AuthHandler    (signup/login/google) │
 │   └── ProfileHandler (get/update profile)  │
 └──────────────────┬─────────────────────────┘
                    │
                    │ MySQL (port 3306)
                    ▼
 ┌────────────────────────────────────────────┐
 │              MySQL Database                │
 │           (users table only)               │
 └────────────────────────────────────────────┘
```

---

## 3. Authentication Layer 1: JWT (API Server)

**Source file:** `internal/api/middleware/auth.go`  
**Library:** `github.com/golang-jwt/jwt/v5`

### 3.1 Token Structure

The JWT contains a custom `Claims` struct:

```
{
  "user_id": 42,          // custom claim — database user ID
  "exp": 1752700800,      // standard claim — 24h from issuance
  "iat": 1752614400       // standard claim — issuance time
}
```

- **Signing algorithm:** HMAC-SHA256 (`jwt.SigningMethodHS256`)
- **Signing key:** `JWT_SECRET` environment variable (default: `"change-me-to-a-secure-secret"`)
- **Expiry:** 24 hours from creation (hardcoded at `auth.go:28`)
- **Payload:** Only `user_id` — no email, no role, no sensitive data

### 3.2 Token Generation

`GenerateToken(userID int, secret string) (string, error)` at `internal/api/middleware/auth.go:24`

Called in three places:
1. `internal/api/handler/auth.go:73` — after successful email/password signup
2. `internal/api/handler/auth.go:110` — after successful email/password login
3. `internal/api/handler/auth.go:142` — after successful Google auth

Every authentication path produces a JWT. The JWT is identical regardless of whether the user signed up with email/password or Google — it always contains just `user_id`.

### 3.3 Token Validation

`ValidateToken(tokenString, secret string) (*Claims, error)` at `internal/api/middleware/auth.go:37`

Validation steps:
1. Parse the JWT string
2. Verify the signing method is `*jwt.SigningMethodHMAC` (prevents algorithm-switching attacks)
3. Verify the HMAC-SHA256 signature using the secret
4. Check that the token is not expired (handled by `jwt.ParseWithClaims` automatically)
5. Extract and return the `Claims`

### 3.4 JWTAuth Middleware

`JWTAuth(secret string) func(http.Handler) http.Handler` at `internal/api/middleware/auth.go:56`

This is the **gateway to all protected API endpoints**. Pipeline:

```
1. Read "Authorization" header
2. If empty → 401 {"error":"authorization header required"}
3. Split on " " → expect ["Bearer", "<token>"]
4. If format wrong → 401 {"error":"invalid authorization format"}
5. Call ValidateToken(token, secret)
6. If invalid/expired → 401 {"error":"invalid or expired token"}
7. Store claims.UserID in request context under key "userID"
8. Call next handler
```

### 3.5 User ID Extraction

`GetUserID(r *http.Request) int` at `internal/api/middleware/auth.go:84`

Retrieves the user ID that `JWTAuth` stored in the context. Returns `0` if missing (which should never happen on protected routes, since the middleware would have rejected the request first).

### 3.6 Statelessness

The API server is **fully stateless**. It does not:
- Store issued tokens anywhere
- Have a token blacklist or revocation mechanism
- Track sessions
- Remember which tokens it has issued

**Implication:** A JWT remains valid until its 24-hour expiry, even after the user logs out on the web server. The web server destroys the cookie, but if the JWT were intercepted before logout, it could still be used against the API directly.

---

## 4. Authentication Layer 2: Cookie Sessions (Web Server)

**Source file:** `internal/web/middleware/session.go`  
**Library:** `github.com/gorilla/sessions` v1.2.2

### 4.1 Session Store Type

The application uses `sessions.CookieStore`, which means **all session data is serialized, signed, encrypted, and stored entirely inside the browser cookie**. There is no server-side session storage.

```go
store := sessions.NewCookieStore([]byte(secret))
```

`NewSessionManager(secret string)` at `internal/web/middleware/session.go:20`

### 4.2 Cookie Configuration

| Property | Value | Source |
|---|---|---|
| **Cookie name** | `"session"` | `SessionName` constant, `session.go:10` |
| **Path** | `"/"` | `session.go:23` |
| **MaxAge** | `86400` (24 hours) | `session.go:24` |
| **HttpOnly** | `true` | `session.go:25` |
| **SameSite** | `Lax` | `session.go:26` |
| **Secure** | `false` (default) | Not explicitly set |
| **Domain** | (not set, defaults to request domain) | Not explicitly set |

### 4.3 What Is Stored in the Session

The session stores exactly **one value** during normal operation and **two values** during the Google OAuth flow:

| Key | Type | When Present | Description |
|---|---|---|---|
| `"token"` | `string` | After login/signup until logout | The JWT issued by the API server |
| `"oauth_state"` | `string` | During Google OAuth flow only | Random 32-byte base64url string for CSRF protection |

### 4.4 Session Operations

#### SetToken
`(sm *SessionManager) SetToken(w, r, token string) error` at `session.go:45`

Called after successful authentication to persist the JWT in the cookie:
- `internal/web/handler/auth.go:70` — after email/password login
- `internal/web/handler/auth.go:95` — after email/password signup
- `internal/web/handler/auth.go:160` — after Google OAuth callback

#### GetToken
`(sm *SessionManager) GetToken(r *http.Request) string` at `session.go:32`

Called by every middleware check and every protected handler to retrieve the JWT:
- `session.go:71` — inside `RequireAuth` middleware
- `session.go:85` — inside `RedirectIfAuth` middleware
- `internal/web/handler/profile.go:33` — `ViewProfile` handler
- `internal/web/handler/profile.go:51` — `EditProfile` handler
- `internal/web/handler/profile.go:75` — `EditProfileSubmit` handler

#### Clear
`(sm *SessionManager) Clear(w, r) error` at `session.go:55`

Called to destroy the session:
- `internal/web/handler/auth.go:173` — logout handler
- `internal/web/handler/profile.go:37` — when API returns error (expired JWT)
- `internal/web/handler/profile.go:54` — when API returns error (expired JWT)

Clear does two things:
1. Wipes all session values: `session.Values = make(map[interface{}]interface{})`
2. Sets `MaxAge = -1` to expire the cookie immediately

### 4.5 Session Expiry Alignment

Both the cookie and the JWT expire after 24 hours, but their clocks start independently:

| Artifact | Lifetime | Clock starts when |
|---|---|---|
| JWT (`exp` claim) | 24 hours | Token is generated by API |
| Cookie (`MaxAge`) | 24 hours | Cookie is set/refreshed by web server |

Since the web server sets the cookie immediately after receiving the JWT, the two expirations are nearly aligned (off by milliseconds of network latency). However, if the API were slow, or if clocks were skewed between servers, the cookie could outlive the JWT or vice versa.

**Consequence:** If the cookie is still valid but the JWT inside it has expired, the `RequireAuth` middleware will let the request through (it only checks if the token string is non-empty), but the API will reject the request with 401. The web handler then clears the session and redirects to login. This is a **graceful degradation** — the user sees "Session expired, please login again".

---

## 5. Authentication Layer 3: Google OAuth 2.0

**Source files:**
- `internal/web/handler/auth.go` (OAuth flow handlers)
- `internal/web/router.go` (OAuth config setup)
- `internal/api/handler/auth.go` (Google user creation on API side)

**Library:** `golang.org/x/oauth2`

### 5.1 OAuth Configuration

Created in `internal/web/router.go:24-29`:

| Field | Source |
|---|---|
| ClientID | `GOOGLE_CLIENT_ID` env var |
| ClientSecret | `GOOGLE_CLIENT_SECRET` env var |
| RedirectURL | `GOOGLE_REDIRECT_URL` (default: `http://localhost:8081/auth/google/callback`) |
| Scopes | `openid`, `email`, `profile` |
| Endpoint | `google.Endpoint` (from `golang.org/x/oauth2/google`) |

### 5.2 OAuth Flow Step by Step

**Step 1 — Initiate (`GET /auth/google`)**  
Handler: `AuthWebHandler.GoogleLogin` at `internal/web/handler/auth.go:100`

1. Generate 32 random bytes via `crypto/rand` → base64url encode → OAuth `state`
2. Save `state` in the session cookie under key `"oauth_state"`
3. Redirect browser to Google's authorization URL with the state

**Step 2 — Google consent screen**  
The user signs in with Google and grants permission. Google redirects back to the callback URL with `?code=...&state=...`.

**Step 3 — Callback (`GET /auth/google/callback`)**  
Handler: `AuthWebHandler.GoogleCallback` at `internal/web/handler/auth.go:111`

1. Read the saved `state` from session
2. Delete `"oauth_state"` from session (single-use)
3. Compare the returned `state` query param with the saved value → reject if mismatch
4. Exchange the `code` for a Google access token via `oauth2.Config.Exchange()`
5. Use the access token to fetch `https://www.googleapis.com/oauth2/v2/userinfo`
6. Extract `{id, email, name}` from Google's response
7. Call `APIClient.GoogleAuth(googleID, email, name)` → which calls `POST /api/auth/google`
8. API server runs `FindOrCreateGoogleUser` → returns JWT + `is_new` flag
9. Store JWT in session via `SetToken`
10. Redirect to `/profile/edit` if new user, `/profile` if returning user

### 5.3 Account Collision Handling

The `FindOrCreateGoogleUser` method at `internal/models/user.go:106` handles the case where a Google login email matches an existing local account:

```
1. Try GetByGoogleID(googleID) → if found, return existing user (not new)
2. Try GetByEmail(email) → if found AND auth_provider=="local", return ErrLocalAccount
3. Otherwise, CreateWithGoogle(email, googleID, fullName) → return new user
```

The reverse is also handled: if a Google user tries to login with email/password, `Authenticate` at `user.go:83` checks `auth_provider == "google"` and returns `ErrGoogleAccount`.

---

## 6. Middleware Pipeline Analysis

### 6.1 API Server Middleware Stack

**Source:** `internal/api/router.go`

```
                    ┌─── corsMiddleware (global, all routes)
                    │
  Public routes ────┤
  /api/auth/*       └─── Handler (no auth check)

                    ┌─── corsMiddleware (global)
                    │
  Protected routes ─┤
  /api/profile      ├─── JWTAuth (validates Bearer token)
                    │
                    └─── Handler (user ID available via context)
```

| Middleware | Applied to | File | Purpose |
|---|---|---|---|
| `corsMiddleware` | All API routes | `router.go:36` | Sets `Access-Control-Allow-Origin: *` and handles OPTIONS preflight |
| `JWTAuth(jwtSecret)` | `/api/profile` (GET, PUT) | `router.go:29` | Validates JWT, extracts user ID into context |

### 6.2 Web Server Middleware Stack

**Source:** `internal/web/router.go`

```
  Static files ──── (no middleware) ──── FileServer
  /static/*

                    ┌─── RedirectIfAuth
  Public pages ─────┤    (if token in session → 302 /profile)
  GET /login        │
  GET /signup       └─── Handler (render page)

  Auth actions ──── (no middleware) ──── Handler
  POST /login
  POST /signup
  GET /auth/google
  GET /auth/google/callback

                    ┌─── RequireAuth
  Protected pages ──┤    (if no token in session → 302 /login)
  GET /profile      │    (sets no-cache headers)
  GET /profile/edit │
  POST /profile/edit└─── Handler (read token, call API)
  POST /logout

  Root ──── (no middleware) ──── 302 /login
  GET /
```

### 6.3 RequireAuth Middleware — Detailed

`(sm *SessionManager) RequireAuth(next http.Handler) http.Handler` at `session.go:67`

```
Request arrives
    │
    ├── Set Cache-Control: no-cache, no-store, must-revalidate
    ├── Set Pragma: no-cache
    ├── Set Expires: 0
    │
    ├── GetToken(r) → reads "token" from session cookie
    │
    ├── if token == "" → redirect to /login (302 See Other)
    │                     request STOPS here
    │
    └── if token != "" → call next handler
                          handler will use token to call API
                          if API rejects token, handler clears session
```

**Critical observation:** `RequireAuth` only checks if the JWT *string* exists in the session. It does **not** validate the JWT itself. Validation happens later when the web handler sends the token to the API server, and the API's `JWTAuth` middleware validates it. This is a deliberate design: the web server treats the JWT as an opaque string and delegates validation to the API.

### 6.4 RedirectIfAuth Middleware — Detailed

`(sm *SessionManager) RedirectIfAuth(next http.Handler) http.Handler` at `session.go:82`

```
Request arrives
    │
    ├── GetToken(r) → reads "token" from session cookie
    │
    ├── if token != "" → redirect to /profile (302 See Other)
    │                     already logged-in users don't see login/signup
    │
    └── if token == "" → call next handler (show login/signup page)
```

### 6.5 CORS Middleware (API)

`corsMiddleware` at `internal/api/router.go:36`

| Header | Value |
|---|---|
| `Access-Control-Allow-Origin` | `*` (all origins) |
| `Access-Control-Allow-Methods` | `GET, POST, PUT, DELETE, OPTIONS` |
| `Access-Control-Allow-Headers` | `Content-Type, Authorization` |

Handles `OPTIONS` preflight requests by returning 200 immediately.

---

## 7. Session Lifecycle: Full Walkthrough

### 7.1 Login (Email/Password)

```
1. Browser → GET /login
   - RedirectIfAuth: no token in cookie → pass through
   - Render login.html

2. Browser → POST /login (email, password)
   - No middleware (POST bypasses RedirectIfAuth group)
   - Web handler calls APIClient.Login(email, password)
   - API handler: validate credentials → bcrypt compare → generate JWT
   - API returns: {"token": "eyJ...", "user": {...}}
   - Web handler: SessionManager.SetToken(w, r, "eyJ...")
     → gorilla/sessions serializes the JWT into the cookie
     → Set-Cookie: session=<encoded data>; Path=/; Max-Age=86400; HttpOnly; SameSite=Lax
   - Redirect → 302 /profile

3. Browser → GET /profile
   - RequireAuth: reads cookie → decodes → finds "token" → non-empty → pass
   - Sets no-cache headers
   - Handler: GetToken(r) → "eyJ..."
   - Handler calls APIClient.GetProfile("eyJ...")
     → HTTP request with Authorization: Bearer eyJ...
   - API: JWTAuth validates token → extracts userID → GetByID(userID) → return user
   - Web handler renders profile_view.html with user data
```

### 7.2 Signup (Email/Password)

Same as login, except:
- Calls `APIClient.Signup` instead of `Login`
- API calls `UserStore.Create` (bcrypt hash + INSERT)
- Redirects to `/profile/edit` instead of `/profile` (new user fills profile first)

### 7.3 Google OAuth

```
1. Browser → GET /auth/google
   - Generate random state (32 bytes, crypto/rand)
   - Store state in session: session.Values["oauth_state"] = state
   - Redirect to Google authorization URL

2. Browser → Google consent → redirected to GET /auth/google/callback?code=X&state=Y

3. Browser → GET /auth/google/callback
   - Read saved state from session
   - Delete "oauth_state" from session
   - Verify state == query param state
   - Exchange code for Google access token
   - Fetch user info from Google
   - Call APIClient.GoogleAuth(googleID, email, name)
   - API: FindOrCreateGoogleUser → generate JWT
   - Web: SetToken(jwt) in session cookie
   - Redirect to /profile/edit (new) or /profile (returning)
```

### 7.4 Logout

```
1. Browser → POST /logout
   - RequireAuth: token exists → pass
   - Handler: SessionManager.Clear(w, r)
     → session.Values = empty map
     → session.Options.MaxAge = -1
     → session.Save(r, w)
     → Set-Cookie: session=<empty>; Max-Age=-1 (expires immediately)
   - Redirect → 302 /login

2. Browser → GET /login
   - Cookie is expired/deleted
   - RedirectIfAuth: no token → pass
   - Render login.html
```

### 7.5 Expired Session / Token

```
1. Browser → GET /profile
   - RequireAuth: token string exists in cookie → pass
   - Handler calls APIClient.GetProfile(token)
   - API: JWTAuth → token expired → 401

2. Web handler receives error from API client
   - Calls SessionManager.Clear(w, r)  (clears stale session)
   - Redirect → /login?error=Session+expired,+please+login+again
```

---

## 8. How Sessions Are Stored

### 8.1 Storage Mechanism

`gorilla/sessions.CookieStore` stores session data **inside the cookie itself**. The data flow:

```
Session data (Go map)
    │
    ▼
Gob encoding (Go's binary serialization)
    │
    ▼
HMAC-SHA256 signing (using SESSION_SECRET)
    │
    ▼
Base64 encoding
    │
    ▼
Set as cookie value in HTTP response
```

When reading:
```
Cookie value from HTTP request
    │
    ▼
Base64 decoding
    │
    ▼
HMAC-SHA256 verification (using SESSION_SECRET)
    │
    ▼
Gob decoding
    │
    ▼
Session data (Go map)
```

### 8.2 What This Means

- **No server-side state:** The web server can restart, scale horizontally, or lose all memory, and sessions survive because they live in the browser.
- **Cookie size limit:** Browser cookies are typically limited to ~4KB. The JWT is roughly 200-300 bytes after encoding, plus gorilla overhead — well within limits.
- **Tamper-proof but not encrypted:** The cookie is **signed** (HMAC) but not **encrypted** by default. The JWT string inside is visible if someone base64-decodes the cookie, but they cannot modify it without the `SESSION_SECRET`. The JWT itself is also just base64-encoded (not encrypted), but it only contains a user ID and timestamps.

### 8.3 There Is No Sessions Table

The database has only a `users` table. There is no `sessions` table, no `tokens` table, and no `refresh_tokens` table. Session persistence is entirely cookie-based.

---

## 9. How Tokens Flow Between Layers

```
┌──────────┐         ┌──────────┐         ┌──────────┐         ┌──────────┐
│ Browser  │         │   Web    │         │   API    │         │  MySQL   │
│          │         │  Server  │         │  Server  │         │          │
└────┬─────┘         └────┬─────┘         └────┬─────┘         └────┬─────┘
     │                    │                    │                    │
     │ POST /login        │                    │                    │
     │ (email,password)   │                    │                    │
     │───────────────────>│                    │                    │
     │                    │ POST /api/auth/login                   │
     │                    │ {"email","password"}                   │
     │                    │───────────────────>│                    │
     │                    │                    │ SELECT * WHERE     │
     │                    │                    │ email=?            │
     │                    │                    │───────────────────>│
     │                    │                    │    user row        │
     │                    │                    │<───────────────────│
     │                    │                    │                    │
     │                    │                    │ bcrypt.Compare()   │
     │                    │                    │ GenerateToken()    │
     │                    │                    │ → JWT signed with  │
     │                    │                    │   JWT_SECRET       │
     │                    │                    │                    │
     │                    │ {"token":"eyJ..."}  │                    │
     │                    │<───────────────────│                    │
     │                    │                    │                    │
     │                    │ SetToken("eyJ...")  │                    │
     │                    │ → serialize JWT     │                    │
     │                    │   into cookie using │                    │
     │                    │   SESSION_SECRET    │                    │
     │                    │                    │                    │
     │ Set-Cookie: session│                    │                    │
     │  =<signed+encoded> │                    │                    │
     │ 302 /profile       │                    │                    │
     │<───────────────────│                    │                    │
     │                    │                    │                    │
     │ GET /profile       │                    │                    │
     │ Cookie: session=...│                    │                    │
     │───────────────────>│                    │                    │
     │                    │ GetToken(r)        │                    │
     │                    │ → decode cookie    │                    │
     │                    │ → verify HMAC      │                    │
     │                    │ → extract "eyJ..." │                    │
     │                    │                    │                    │
     │                    │ GET /api/profile   │                    │
     │                    │ Authorization:     │                    │
     │                    │ Bearer eyJ...      │                    │
     │                    │───────────────────>│                    │
     │                    │                    │ JWTAuth:           │
     │                    │                    │ → verify HMAC      │
     │                    │                    │ → check expiry     │
     │                    │                    │ → extract userID   │
     │                    │                    │                    │
     │                    │                    │ GetByID(userID)    │
     │                    │                    │───────────────────>│
     │                    │                    │    user data       │
     │                    │                    │<───────────────────│
     │                    │ {"user":{...}}     │                    │
     │                    │<───────────────────│                    │
     │                    │                    │                    │
     │ <HTML page>        │                    │                    │
     │<───────────────────│                    │                    │
```

### Two Distinct Secrets

| Secret | Used by | Purpose |
|---|---|---|
| `JWT_SECRET` | API server | Signs/verifies JWT tokens (HMAC-SHA256) |
| `SESSION_SECRET` | Web server | Signs/verifies the cookie containing the JWT (gorilla securecookie HMAC) |

These are **independent keys** with **independent purposes**. The web server never validates the JWT — it only signs the cookie envelope. The API server never reads cookies — it only validates JWTs from the `Authorization` header.

---

## 10. Route Protection Matrix

### API Server Routes

| Route | Method | Middleware | Auth Required | Handler |
|---|---|---|---|---|
| `/api/auth/signup` | POST | CORS only | No | `AuthHandler.Signup` |
| `/api/auth/login` | POST | CORS only | No | `AuthHandler.Login` |
| `/api/auth/google` | POST | CORS only | No | `AuthHandler.GoogleAuth` |
| `/api/profile` | GET | CORS + JWTAuth | Yes (Bearer token) | `ProfileHandler.GetProfile` |
| `/api/profile` | PUT | CORS + JWTAuth | Yes (Bearer token) | `ProfileHandler.UpdateProfile` |

### Web Server Routes

| Route | Method | Middleware | Auth Required | Handler |
|---|---|---|---|---|
| `/static/*` | GET | None | No | File server |
| `/` | GET | None | No | Redirect → `/login` |
| `/login` | GET | RedirectIfAuth | No (redirects if yes) | `AuthWebHandler.LoginPage` |
| `/signup` | GET | RedirectIfAuth | No (redirects if yes) | `AuthWebHandler.SignupPage` |
| `/login` | POST | None | No | `AuthWebHandler.LoginSubmit` |
| `/signup` | POST | None | No | `AuthWebHandler.SignupSubmit` |
| `/auth/google` | GET | None | No | `AuthWebHandler.GoogleLogin` |
| `/auth/google/callback` | GET | None | No | `AuthWebHandler.GoogleCallback` |
| `/profile` | GET | RequireAuth | Yes (session cookie) | `ProfileWebHandler.ViewProfile` |
| `/profile/edit` | GET | RequireAuth | Yes (session cookie) | `ProfileWebHandler.EditProfile` |
| `/profile/edit` | POST | RequireAuth | Yes (session cookie) | `ProfileWebHandler.EditProfileSubmit` |
| `/logout` | POST | RequireAuth | Yes (session cookie) | `AuthWebHandler.Logout` |

---

## 11. Security Audit Findings

### 11.1 Strengths

| # | Finding | Details |
|---|---|---|
| S1 | **HttpOnly cookies** | `HttpOnly=true` prevents JavaScript from accessing the session cookie, mitigating XSS-based session theft (`session.go:25`) |
| S2 | **SameSite=Lax** | Provides CSRF protection for state-changing POST requests (`session.go:26`) |
| S3 | **Algorithm pinning** | `ValidateToken` verifies the signing method is `*jwt.SigningMethodHMAC` before accepting, preventing algorithm-switching attacks (`auth.go:39`) |
| S4 | **Password hashing** | bcrypt with default cost (10 rounds) for password storage (`user.go:45`) |
| S5 | **Password hash never serialized** | `PasswordHash` field has `json:"-"` tag, preventing accidental exposure in API responses (`user.go:24`) |
| S6 | **OAuth state parameter** | 32 bytes from `crypto/rand` for OAuth CSRF protection (`auth.go:177-180`) |
| S7 | **OAuth state single-use** | State is deleted from session immediately after reading in callback (`auth.go:114`) |
| S8 | **No-cache headers** | All protected pages set `Cache-Control: no-cache, no-store, must-revalidate` to prevent back-button exposure after logout (`session.go:94-97`) |
| S9 | **Logout uses POST** | Prevents CSRF logout via image tags or link preloading (`profile_view.html:20`) |
| S10 | **Auth provider enforcement** | Server-side checks prevent Google users from using password login and vice versa (`user.go:92-94`, `user.go:114`) |
| S11 | **Google email immutability** | Both UI (disabled field, `profile_edit.html:22`) and API (`profile.go:63-64`) enforce that Google users cannot change their email |
| S12 | **Parameterized queries** | All SQL uses `?` placeholders, preventing SQL injection (`user.go` throughout) |

### 11.2 Concerns

| # | Severity | Finding | Details | Location |
|---|---|---|---|---|
| C1 | **High** | Cookie `Secure` flag not set | The cookie is transmitted over HTTP as well as HTTPS. In production, the session cookie (containing the JWT) could be intercepted over unencrypted connections. | `session.go:22-27` — `Secure` not set in `store.Options` |
| C2 | **High** | Hardcoded default secrets | `JWT_SECRET` defaults to `"change-me-to-a-secure-secret"` and `SESSION_SECRET` to `"change-me-to-a-session-secret"`. If deployed without setting env vars, all tokens are signed with known keys. | `config.go:38-39` |
| C3 | **Medium** | No token revocation | The API is stateless with no token blacklist. After logout, the JWT is valid for up to 24 hours if intercepted. An attacker who captures a JWT can use it until expiry. | Architecture-level |
| C4 | **Medium** | CORS allows all origins | `Access-Control-Allow-Origin: *` allows any website to make authenticated requests to the API if they somehow obtain a JWT. | `router.go:38` |
| C5 | **Medium** | Session cookie not encrypted | gorilla `CookieStore` signs but does not encrypt by default. The JWT (containing user ID and expiry) is readable by anyone who inspects the cookie. While not directly exploitable (the JWT itself is not secret from the cookie holder), it leaks information. | `session.go:21` — only one key passed to `NewCookieStore` (for signing). A second key would enable encryption. |
| C6 | **Low** | No rate limiting | Login, signup, and Google auth endpoints have no rate limiting. Brute-force attacks against login are possible. | `router.go` — no rate-limit middleware |
| C7 | **Low** | RequireAuth doesn't validate JWT | The web middleware only checks if the token string is non-empty. An expired or corrupted JWT will pass `RequireAuth` but fail at the API level. This creates unnecessary round-trips and slightly confusing error flows. | `session.go:71-73` |
| C8 | **Low** | No CSRF tokens on forms | The signup, login, and profile-edit forms do not include CSRF tokens. While `SameSite=Lax` mitigates most CSRF, it doesn't protect against all scenarios (e.g., same-site subdomain attacks). | `login.html`, `signup.html`, `profile_edit.html` |
| C9 | **Info** | JWT contains minimal claims | The JWT only contains `user_id`, `exp`, and `iat`. No issuer (`iss`), audience (`aud`), or JWT ID (`jti`). While not a vulnerability, standard claims improve token validation rigor. | `auth.go:25-31` |

---

## 12. Configuration & Secrets Management

**Source:** `internal/config/config.go`

### 12.1 Secret Loading

Secrets are loaded via environment variables, with `.env` file as fallback:

```
1. config.LoadEnvFile(".env")    — parse .env, set vars only if not already in env
2. config.Load()                 — read env vars into Config struct
```

Environment variables always override `.env` file values (enforced at `config.go:71`).

### 12.2 Security-Sensitive Configuration

| Variable | Default | Risk if unchanged | Used by |
|---|---|---|---|
| `JWT_SECRET` | `change-me-to-a-secure-secret` | Anyone can forge valid JWTs | API server |
| `SESSION_SECRET` | `change-me-to-a-session-secret` | Anyone can forge session cookies | Web server |
| `DB_PASSWORD` | `""` (empty) | No database auth | API server |
| `GOOGLE_CLIENT_ID` | `""` (empty) | Google OAuth won't work (fails gracefully) | Web server |
| `GOOGLE_CLIENT_SECRET` | `""` (empty) | Google OAuth won't work (fails gracefully) | Web server |

### 12.3 Docker Compose Secrets

The `docker-compose.yml` passes secrets via environment variables. Default passwords are set:

- `DB_PASSWORD` → `rootpassword`
- `JWT_SECRET` → `change-me-to-a-secure-secret`
- `SESSION_SECRET` → `change-me-to-a-session-secret`

These are intended for development only. Production deployments must override all three.

---

## 13. Key File Reference

| File | Role in Session/Auth |
|---|---|
| `internal/api/middleware/auth.go` | JWT generation, validation, JWTAuth middleware, context extraction |
| `internal/web/middleware/session.go` | CookieStore setup, GetToken/SetToken/Clear, RequireAuth, RedirectIfAuth, no-cache headers |
| `internal/api/handler/auth.go` | API-side login/signup/google endpoints — issues JWTs |
| `internal/web/handler/auth.go` | Web-side login/signup/google/logout — manages cookie sessions, orchestrates OAuth flow |
| `internal/web/handler/profile.go` | Reads JWT from session, sends to API, handles expired-token cleanup |
| `internal/web/client/client.go` | HTTP client that attaches `Authorization: Bearer <JWT>` header to API calls |
| `internal/api/router.go` | Wires JWTAuth middleware to protected API routes |
| `internal/web/router.go` | Wires RequireAuth/RedirectIfAuth middleware to web routes, creates SessionManager |
| `internal/config/config.go` | Loads JWT_SECRET and SESSION_SECRET from environment |
| `internal/models/user.go` | Password hashing (bcrypt), auth provider checks, account collision handling |
| `internal/api/middleware/auth_test.go` | Unit tests for JWT generation, validation, and middleware behavior |

---

## 14. Summary of Concerns and Observations

### Architecture Observations

1. **The web server is a BFF (Backend-for-Frontend).** It owns the browser relationship (cookies, HTML, redirects) but delegates all data operations and token validation to the API server. This is a clean separation.

2. **The JWT is treated as an opaque token by the web server.** The web server never decodes or validates it — it stores it, retrieves it, and passes it to the API. The API is the sole authority on token validity.

3. **Both "session" and "token" expire in 24 hours, but there is no refresh mechanism.** When the JWT expires, the user must re-authenticate. There are no refresh tokens.

4. **Session = cookie = JWT.** There's a direct 1:1 mapping. Destroying the cookie (logout) destroys the session. There is no dangling server-side state to clean up.

5. **The Google OAuth flow temporarily stores additional state in the session cookie** (`oauth_state`), but this is cleaned up in the callback handler.

### Production Readiness Gaps

| Gap | Recommendation |
|---|---|
| `Secure` flag not set on cookie | Set `Secure: true` in production (requires HTTPS) |
| Default secrets are predictable | Enforce non-default secrets at startup (refuse to start if defaults detected) |
| No token revocation | Consider a short-lived JWT (15 min) + refresh token pattern, or a lightweight token blacklist |
| CORS allows all origins | Restrict to the actual web server origin in production |
| No rate limiting | Add rate limiting on auth endpoints |
| Cookie not encrypted | Pass a second key to `NewCookieStore` for AES encryption |

### What Works Well

- Clean separation between cookie session (web) and JWT auth (API)
- Stateless API server is horizontally scalable
- No server-side session state means no session-store bottleneck
- OAuth state parameter is properly generated and validated
- Password hashes never leave the API
- Back-button protection via no-cache headers
- Auth provider collision detection prevents account takeover scenarios