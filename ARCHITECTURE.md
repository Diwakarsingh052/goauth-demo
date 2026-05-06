# Architecture Reference

This document explains the full architecture of the project: how the two servers are structured, how every request flows from the browser through the system, the role of every package, and the purpose of every exported type and method.

---

## Table of Contents

- [1. High-Level Architecture](#1-high-level-architecture)
- [2. Directory Map](#2-directory-map)
- [3. Dependency Graph](#3-dependency-graph)
- [4. Request Flows](#4-request-flows)
  - [4.1 Email/Password Login](#41-emailpassword-login)
  - [4.2 Email/Password Sign-Up](#42-emailpassword-sign-up)
  - [4.3 Google OAuth Login / Sign-Up](#43-google-oauth-login--sign-up)
  - [4.4 View Profile (2-D)](#44-view-profile-2-d)
  - [4.5 Edit Profile (2-C)](#45-edit-profile-2-c)
  - [4.6 Logout](#46-logout)
- [5. Package Reference](#5-package-reference)
  - [5.1 cmd/api](#51-cmdapi)
  - [5.2 cmd/web](#52-cmdweb)
  - [5.3 internal/config](#53-internalconfig)
  - [5.4 internal/database](#54-internaldatabase)
  - [5.5 internal/models](#55-internalmodels)
  - [5.6 internal/api/middleware](#56-internalapimiddleware)
  - [5.7 internal/api/handler](#57-internalapihandler)
  - [5.8 internal/api (router)](#58-internalapi-router)
  - [5.9 internal/web/client](#59-internalwebclient)
  - [5.10 internal/web/middleware](#510-internalwebmiddleware)
  - [5.11 internal/web/handler](#511-internalwebhandler)
  - [5.12 internal/web (router)](#512-internalweb-router)
- [6. Database Schema](#6-database-schema)
- [7. Authentication Architecture](#7-authentication-architecture)
- [8. Security Mechanisms](#8-security-mechanisms)
- [9. Template System](#9-template-system)

---

## 1. High-Level Architecture

The application is two independent Go servers that run side by side.

```
 Browser (user)
    |
    |  HTTP requests (port 8081)
    v
 ┌──────────────────────────────────────────┐
 │            WEB SERVER (cmd/web)          │
 │                                          │
 │  Responsibilities:                       │
 │  - Serve HTML pages via Go templates     │
 │  - Manage browser sessions (cookies)     │
 │  - Run the Google OAuth redirect flow    │
 │  - Forward every data operation to the   │
 │    API server over HTTP                  │
 │                                          │
 │  Does NOT talk to the database directly. │
 └─────────────────┬────────────────────────┘
                   │
                   │  HTTP/JSON (port 8080)
                   │  Authorization: Bearer <JWT>
                   v
 ┌──────────────────────────────────────────┐
 │            API SERVER (cmd/api)          │
 │                                          │
 │  Responsibilities:                       │
 │  - Expose a stateless REST API (JSON)   │
 │  - Validate JWT tokens                   │
 │  - Execute all business logic            │
 │  - Read/write the MySQL database         │
 │                                          │
 │  Knows nothing about HTML, templates,    │
 │  sessions, or Google OAuth redirects.    │
 └─────────────────┬────────────────────────┘
                   │
                   │  MySQL protocol (port 3306)
                   v
 ┌──────────────────────────────────────────┐
 │               MySQL                      │
 │          (users table)                   │
 └──────────────────────────────────────────┘
```

**Why two servers?**  
The specification requires the REST API to be independent, meaning you could replace the entire Go-template frontend with a React SPA, a mobile app, or a CLI tool and the API would require zero changes. The web server is just one possible consumer of the API.

---

## 2. Directory Map

```
.
├── cmd/
│   ├── api/main.go               # Entry point: starts the REST API server
│   └── web/main.go               # Entry point: starts the Web frontend server
│
├── internal/
│   ├── config/
│   │   └── config.go             # Reads env vars / .env file into a Config struct
│   │
│   ├── database/
│   │   └── db.go                 # Opens MySQL connection, runs table migration
│   │
│   ├── models/
│   │   └── user.go               # User struct, UserStore (all DB queries)
│   │
│   ├── api/                      # ── Everything below is API-server-only ──
│   │   ├── router.go             # Wires routes and middleware for the API
│   │   ├── middleware/
│   │   │   ├── auth.go           # JWT generation, validation, HTTP middleware
│   │   │   └── auth_test.go      # Unit tests for JWT logic
│   │   └── handler/
│   │       ├── auth.go           # Signup, Login, GoogleAuth handlers
│   │       ├── auth_test.go      # Integration tests (need MySQL)
│   │       ├── profile.go        # GetProfile, UpdateProfile handlers
│   │       └── profile_test.go   # Integration tests (need MySQL)
│   │
│   └── web/                      # ── Everything below is Web-server-only ──
│       ├── router.go             # Wires routes, middleware, loads templates
│       ├── client/
│       │   └── client.go         # HTTP client that calls the REST API
│       ├── middleware/
│       │   └── session.go        # Cookie session manager, auth guard
│       └── handler/
│           ├── auth.go           # Login/Signup pages, Google OAuth redirect
│           └── profile.go        # Profile view/edit pages
│
├── templates/                    # Go HTML templates (used by web server)
│   ├── base.html                 # Shared layout: <html>, <head>, <body>
│   ├── login.html                # Login page content block
│   ├── signup.html               # Sign-up page content block
│   ├── profile_edit.html         # Profile edit form content block
│   └── profile_view.html         # Profile display content block
│
├── static/css/style.css          # All application CSS
├── migrations/001_init.sql       # SQL schema (also auto-run by API on startup)
├── Dockerfile                    # Multi-stage Docker build (api + web targets)
├── docker-compose.yml            # MySQL + API + Web orchestration
├── Makefile                      # Build, run, test, Docker commands
├── .env.example                  # Environment variable template
└── README.md                     # Setup and usage instructions
```

---

## 3. Dependency Graph

This shows which package imports which. Arrows mean "depends on".

```
cmd/api/main
  ├── internal/config
  ├── internal/database       ──> internal/config
  ├── internal/models
  └── internal/api            ──> internal/api/handler   ──> internal/api/middleware
                                                          └──> internal/models

cmd/web/main
  ├── internal/config
  └── internal/web            ──> internal/web/handler   ──> internal/web/client
                               │                          └──> internal/web/middleware
                               ├── internal/web/client
                               ├── internal/web/middleware
                               └── internal/config
```

Key separation rules:
- `internal/api/*` never imports `internal/web/*` (and vice versa).
- `internal/web/*` never imports `internal/models` or `internal/database`. It only talks to the API via `internal/web/client`.
- `internal/models` and `internal/config` are shared foundation packages with no cross-dependencies.

---

## 4. Request Flows

Every user action is traced from browser to database and back.

### 4.1 Email/Password Login

```
Browser                  Web Server                      API Server                 MySQL
  │                         │                               │                        │
  │ GET /login              │                               │                        │
  │────────────────────────>│                               │                        │
  │                         │ RedirectIfAuth middleware:     │                        │
  │                         │ no session token found, pass  │                        │
  │                         │                               │                        │
  │     <login.html>        │                               │                        │
  │<────────────────────────│                               │                        │
  │                         │                               │                        │
  │ POST /login             │                               │                        │
  │ (email + password)      │                               │                        │
  │────────────────────────>│                               │                        │
  │                         │ POST /api/auth/login          │                        │
  │                         │ {"email":"…","password":"…"}  │                        │
  │                         │──────────────────────────────>│                        │
  │                         │                               │ SELECT … WHERE email=? │
  │                         │                               │───────────────────────>│
  │                         │                               │    user row            │
  │                         │                               │<───────────────────────│
  │                         │                               │ bcrypt.Compare(hash,pw)│
  │                         │                               │ GenerateToken(userID)  │
  │                         │  {"token":"jwt…","user":{…}}  │                        │
  │                         │<──────────────────────────────│                        │
  │                         │                               │                        │
  │                         │ SetToken(jwt) in session cookie                        │
  │  302 /profile           │                               │                        │
  │<────────────────────────│                               │                        │
```

If the email does not exist, the API returns `{"error":"username entered does not exist"}` (401).  
If the password is wrong, the API returns `{"error":"password is incorrect"}` (401).  
The web handler re-renders `login.html` with the error message.

### 4.2 Email/Password Sign-Up

```
Browser                  Web Server                      API Server                 MySQL
  │                         │                               │                        │
  │ GET /signup             │                               │                        │
  │────────────────────────>│                               │                        │
  │     <signup.html>       │                               │                        │
  │<────────────────────────│                               │                        │
  │                         │                               │                        │
  │ POST /signup            │                               │                        │
  │ (email + password)      │                               │                        │
  │────────────────────────>│                               │                        │
  │                         │ POST /api/auth/signup         │                        │
  │                         │ {"email":"…","password":"…"}  │                        │
  │                         │──────────────────────────────>│                        │
  │                         │                               │ bcrypt.Hash(password)  │
  │                         │                               │ INSERT INTO users …    │
  │                         │                               │───────────────────────>│
  │                         │                               │ GenerateToken(userID)  │
  │                         │  {"token":"jwt…","user":{…}}  │                        │
  │                         │<──────────────────────────────│                        │
  │                         │                               │                        │
  │                         │ SetToken(jwt) in session cookie                        │
  │  302 /profile/edit      │                               │                        │
  │<────────────────────────│  (new users go to 2-C first)  │                        │
```

After sign-up the redirect target is `/profile/edit` (page 2-C), not `/profile`. This ensures new users fill in their profile information before seeing the main profile page.

### 4.3 Google OAuth Login / Sign-Up

```
Browser                 Web Server                  Google                API Server        MySQL
  │                        │                          │                      │                │
  │ GET /auth/google       │                          │                      │                │
  │───────────────────────>│                          │                      │                │
  │                        │ generate random state    │                      │                │
  │                        │ save state in session    │                      │                │
  │  302 Google OAuth URL  │                          │                      │                │
  │<───────────────────────│                          │                      │                │
  │                        │                          │                      │                │
  │ GET accounts.google.com/o/oauth2/auth?…           │                      │                │
  │──────────────────────────────────────────────────>│                      │                │
  │        (user signs in with Google)                │                      │                │
  │  302 /auth/google/callback?code=…&state=…         │                      │                │
  │<─────────────────────────────────────────────────│                      │                │
  │                        │                          │                      │                │
  │ GET /auth/google/callback?code=…&state=…          │                      │                │
  │───────────────────────>│                          │                      │                │
  │                        │ verify state matches     │                      │                │
  │                        │                          │                      │                │
  │                        │ Exchange(code) for token │                      │                │
  │                        │─────────────────────────>│                      │                │
  │                        │  access_token            │                      │                │
  │                        │<─────────────────────────│                      │                │
  │                        │                          │                      │                │
  │                        │ GET /oauth2/v2/userinfo  │                      │                │
  │                        │─────────────────────────>│                      │                │
  │                        │  {id, email, name}       │                      │                │
  │                        │<─────────────────────────│                      │                │
  │                        │                                                 │                │
  │                        │ POST /api/auth/google                           │                │
  │                        │ {"google_id":"…","email":"…","name":"…"}        │                │
  │                        │────────────────────────────────────────────────>│                │
  │                        │                                                 │ FindOrCreate   │
  │                        │                                                 │───────────────>│
  │                        │  {"token":"jwt…","user":{…},"is_new":true/false}│                │
  │                        │<────────────────────────────────────────────────│                │
  │                        │                                                 │                │
  │                        │ SetToken(jwt) in session cookie                 │                │
  │                        │                                                 │                │
  │                        │ if is_new:  302 /profile/edit                   │                │
  │                        │ if !is_new: 302 /profile                        │                │
  │<───────────────────────│                                                 │                │
```

The `is_new` flag is how the system distinguishes a first-time Google user (who needs to fill in profile info on page 2-C) from a returning Google user (who goes straight to page 2-D).

### 4.4 View Profile (2-D)

```
Browser                  Web Server                      API Server                 MySQL
  │                         │                               │                        │
  │ GET /profile            │                               │                        │
  │────────────────────────>│                               │                        │
  │                         │ RequireAuth middleware:        │                        │
  │                         │   read JWT from session cookie │                        │
  │                         │   token found -> pass          │                        │
  │                         │   set Cache-Control: no-store  │                        │
  │                         │                               │                        │
  │                         │ GET /api/profile              │                        │
  │                         │ Authorization: Bearer jwt…    │                        │
  │                         │──────────────────────────────>│                        │
  │                         │                               │ JWTAuth middleware:     │
  │                         │                               │   validate token       │
  │                         │                               │   extract userID       │
  │                         │                               │ SELECT … WHERE id=?    │
  │                         │                               │───────────────────────>│
  │                         │  {"user":{…}}                 │                        │
  │                         │<──────────────────────────────│                        │
  │                         │                               │                        │
  │                         │ render profile_view.html      │                        │
  │   <profile_view.html>   │ with user data                │                        │
  │<────────────────────────│                               │                        │
```

If the session has no token, `RequireAuth` redirects to `/login` before the handler runs. If the JWT is expired, the API returns 401, the web handler clears the session and redirects to `/login`. In both cases the template is never rendered without valid authentication.

### 4.5 Edit Profile (2-C)

**Loading the form:**

```
Browser  ──GET /profile/edit──>  Web Server  ──GET /api/profile──>  API  ──SELECT──>  MySQL
                                              <── user JSON ──
                               render profile_edit.html with pre-filled fields
         <── HTML with populated form ──
```

**Submitting the form:**

```
Browser  ──POST /profile/edit──>  Web Server  ──PUT /api/profile──>  API  ──UPDATE──>  MySQL
           (full_name,                         {"full_name":"…",
            telephone,                          "telephone":"…",
            email)                              "email":"…"}
                                              <── updated user JSON ──
         <── 302 /profile ──
```

For Google-authenticated users, the email field is rendered as `disabled` in the HTML and a hidden input sends the original email value. The API also enforces this server-side: if `auth_provider` is `"google"`, the API ignores the submitted email and keeps the original.

### 4.6 Logout

```
Browser                  Web Server
  │                         │
  │ POST /logout            │
  │────────────────────────>│
  │                         │ RequireAuth middleware: token present, pass
  │                         │ Clear session (MaxAge = -1, delete all values)
  │  302 /login             │
  │<────────────────────────│
  │                         │
  │ GET /login              │
  │────────────────────────>│
  │                         │ (no session token -> normal login page)
```

The JWT still exists in memory until it expires (24 hours), but the web server has deleted it from the session cookie. The browser cannot present it on subsequent requests. The `Cache-Control: no-store` header on all protected pages ensures the browser does not show a cached version of `/profile` or `/profile/edit` when the user presses the back button.

---

## 5. Package Reference

### 5.1 `cmd/api`

**File:** `cmd/api/main.go`  
**Role:** Entry point for the REST API server. Loads configuration, connects to MySQL (with retries), runs migrations, creates a `UserStore`, builds the API router, and starts listening on `API_PORT`.

**Boot sequence:**
1. `config.LoadEnvFile(".env")` -- parse `.env` file
2. `config.Load()` -- read env vars into `Config` struct
3. `database.Connect(cfg)` -- open MySQL connection (retries 30x for Docker)
4. `database.Migrate(db)` -- `CREATE TABLE IF NOT EXISTS users`
5. `models.NewUserStore(db)` -- wrap `*sql.DB` in the data-access layer
6. `api.NewRouter(users, jwtSecret)` -- build the HTTP router
7. `http.ListenAndServe` -- start serving

### 5.2 `cmd/web`

**File:** `cmd/web/main.go`  
**Role:** Entry point for the Web frontend server. Loads configuration, builds the web router (which internally creates the API client, session manager, OAuth config, and loads templates), and starts listening on `WEB_PORT`.

**Boot sequence:**
1. `config.LoadEnvFile(".env")`
2. `config.Load()`
3. `web.NewRouter(cfg)` -- build the HTTP router with all dependencies
4. `http.ListenAndServe`

---

### 5.3 `internal/config`

**File:** `internal/config/config.go`  
**Role:** Centralized configuration management. Every tunable value (database credentials, ports, secrets, OAuth keys) flows through this package. Both servers import it.

#### Type: `Config`

Holds all configuration values as string fields.

| Field | Environment Variable | Default | Used By |
|---|---|---|---|
| `DBHost` | `DB_HOST` | `localhost` | API |
| `DBPort` | `DB_PORT` | `3306` | API |
| `DBUser` | `DB_USER` | `root` | API |
| `DBPassword` | `DB_PASSWORD` | (empty) | API |
| `DBName` | `DB_NAME` | `challange_go` | API |
| `APIPort` | `API_PORT` | `8080` | API |
| `WebPort` | `WEB_PORT` | `8081` | Web |
| `APIBaseURL` | `API_BASE_URL` | `http://localhost:8080` | Web |
| `JWTSecret` | `JWT_SECRET` | `change-me-to-a-secure-secret` | API |
| `SessionSecret` | `SESSION_SECRET` | `change-me-to-a-session-secret` | Web |
| `GoogleClientID` | `GOOGLE_CLIENT_ID` | (empty) | Web |
| `GoogleClientSecret` | `GOOGLE_CLIENT_SECRET` | (empty) | Web |
| `GoogleRedirectURL` | `GOOGLE_REDIRECT_URL` | `http://localhost:8081/auth/google/callback` | Web |

#### Functions

| Signature | Description |
|---|---|
| `Load() *Config` | Reads all environment variables and returns a populated `Config`. Called once at startup. |
| `(c *Config) DSN() string` | Builds the MySQL DSN string: `user:pass@tcp(host:port)/dbname?parseTime=true`. |
| `LoadEnvFile(path string)` | Parses a `.env` file and sets environment variables. Skips keys that are already set in the environment (env vars take precedence). Silently does nothing if the file does not exist. |

---

### 5.4 `internal/database`

**File:** `internal/database/db.go`  
**Role:** Opens the MySQL connection and creates the schema. Only imported by the API server.

#### Functions

| Signature | Description |
|---|---|
| `Connect(cfg *config.Config) (*sql.DB, error)` | Opens a MySQL connection using `cfg.DSN()`. Retries up to 30 times with a 2-second delay between attempts, logging each failure. This retry loop exists to handle Docker Compose startup ordering where the API container may start before MySQL is ready. Sets connection pool limits (25 open, 5 idle). |
| `Migrate(db *sql.DB) error` | Executes `CREATE TABLE IF NOT EXISTS users (…)`. Called once at API startup. Idempotent -- safe to run on every boot. |

---

### 5.5 `internal/models`

**File:** `internal/models/user.go`  
**Role:** Data access layer. Defines the `User` struct and `UserStore` which contains every database query the application needs. Only imported by the API server's handler package.

#### Sentinel Errors

These are package-level `error` values used for control flow. Callers check them with `errors.Is()`.

| Variable | Value | When Returned |
|---|---|---|
| `ErrUserNotFound` | `"username entered does not exist"` | `GetByID`, `GetByEmail`, `GetByGoogleID` when no row matches |
| `ErrDuplicateEmail` | `"email already exists"` | `Create`, `CreateWithGoogle`, `UpdateProfile` on unique constraint violation |
| `ErrInvalidPassword` | `"password is incorrect"` | `Authenticate` when bcrypt comparison fails |
| `ErrGoogleAccount` | `"this account uses Google sign-in, please use the Google button"` | `Authenticate` when a Google user tries to log in with a password |
| `ErrLocalAccount` | `"this email is registered with email/password, please use the login form"` | `FindOrCreateGoogleUser` when a Google login collides with an existing email/password account |

#### Type: `User`

Represents a single row in the `users` table.

| Field | Type | JSON Tag | Description |
|---|---|---|---|
| `ID` | `int` | `"id"` | Auto-increment primary key |
| `Email` | `string` | `"email"` | Unique email address (also the username) |
| `PasswordHash` | `string` | `"-"` (never serialized) | bcrypt hash, empty for Google users |
| `GoogleID` | `string` | `"google_id,omitempty"` | Google's user ID, empty for local users |
| `AuthProvider` | `string` | `"auth_provider"` | `"local"` or `"google"` |
| `FullName` | `string` | `"full_name"` | User's display name |
| `Telephone` | `string` | `"telephone"` | User's phone number |
| `CreatedAt` | `time.Time` | `"created_at"` | Row creation timestamp |
| `UpdatedAt` | `time.Time` | `"updated_at"` | Last modification timestamp |

#### Type: `UserStore`

Wraps a `*sql.DB` and provides all user-related database operations.

| Field | Type | Description |
|---|---|---|
| `DB` | `*sql.DB` | The MySQL connection pool |

**Constructor:**

| Signature | Description |
|---|---|
| `NewUserStore(db *sql.DB) *UserStore` | Wraps the database connection in a `UserStore`. |

**Methods:**

| Signature | Description |
|---|---|
| `(s *UserStore) Create(email, password string) (*User, error)` | Hashes the password with bcrypt, inserts a new row with `auth_provider='local'`, and returns the created user. Returns `ErrDuplicateEmail` if the email already exists. |
| `(s *UserStore) CreateWithGoogle(email, googleID, fullName string) (*User, error)` | Inserts a new row with `auth_provider='google'` and the Google ID and name pre-filled. Returns `ErrDuplicateEmail` if the email already exists. |
| `(s *UserStore) Authenticate(email, password string) (*User, error)` | Looks up the user by email, checks that `auth_provider` is not `"google"`, and compares the password against the stored bcrypt hash. Returns `ErrUserNotFound`, `ErrGoogleAccount`, or `ErrInvalidPassword` on failure. |
| `(s *UserStore) FindOrCreateGoogleUser(email, googleID, fullName string) (*User, bool, error)` | First tries to find an existing user by `google_id`. If found, returns `(user, false, nil)`. If not found, checks whether the email belongs to a local account (returns `ErrLocalAccount` if so). Otherwise creates a new Google user and returns `(user, true, nil)`. The `bool` return value (`isNew`) tells callers whether this is a brand-new registration. |
| `(s *UserStore) GetByID(id int) (*User, error)` | `SELECT … WHERE id = ?`. Returns `ErrUserNotFound` on no rows. |
| `(s *UserStore) GetByEmail(email string) (*User, error)` | `SELECT … WHERE email = ?`. Returns `ErrUserNotFound` on no rows. |
| `(s *UserStore) GetByGoogleID(googleID string) (*User, error)` | `SELECT … WHERE google_id = ?`. Returns `ErrUserNotFound` on no rows. |
| `(s *UserStore) UpdateProfile(id int, fullName, telephone, email string) error` | `UPDATE users SET full_name=?, telephone=?, email=? WHERE id=?`. Returns `ErrDuplicateEmail` if the new email collides with another account. |

---

### 5.6 `internal/api/middleware`

**File:** `internal/api/middleware/auth.go`  
**Role:** JWT token generation, validation, and an HTTP middleware that protects API routes. This package is the boundary between "public" and "protected" API endpoints.

#### Type: `contextKey`

A private `string` type used as context key to avoid collisions.

| Constant | Value | Description |
|---|---|---|
| `UserIDKey` | `contextKey("userID")` | The key under which the authenticated user's ID is stored in `context.Context` |

#### Type: `Claims`

Embeds `jwt.RegisteredClaims` and adds the application-specific `UserID` field.

| Field | Type | Description |
|---|---|---|
| `UserID` | `int` | The database ID of the authenticated user |
| `RegisteredClaims` | `jwt.RegisteredClaims` | Standard JWT fields (expiry, issued-at, etc.) |

#### Functions

| Signature | Description |
|---|---|
| `GenerateToken(userID int, secret string) (string, error)` | Creates a signed HS256 JWT containing the user ID. The token expires 24 hours after creation. Called by auth handlers after successful login/signup. |
| `ValidateToken(tokenString, secret string) (*Claims, error)` | Parses the JWT string, verifies the HMAC-SHA256 signature and expiry, and returns the claims. Returns an error if the token is malformed, expired, or signed with a different secret. |
| `JWTAuth(secret string) func(http.Handler) http.Handler` | Returns a Gorilla Mux-compatible middleware function. For each request it: (1) reads the `Authorization` header, (2) splits on `"Bearer "`, (3) calls `ValidateToken`, (4) stores `claims.UserID` in the request context, (5) passes to the next handler. Returns 401 JSON error on any failure. |
| `GetUserID(r *http.Request) int` | Extracts the user ID that `JWTAuth` stored in the request context. Returns 0 if not present (should never happen on protected routes). |

---

### 5.7 `internal/api/handler`

**Files:** `internal/api/handler/auth.go`, `internal/api/handler/profile.go`  
**Role:** HTTP handler functions for every API endpoint. Each handler decodes the JSON request, calls `UserStore` methods, and writes a JSON response.

#### Type: `AuthHandler`

Handles all authentication endpoints.

| Field | Type | Description |
|---|---|---|
| `Users` | `*models.UserStore` | Data access layer |
| `JWTSecret` | `string` | Secret key for signing JWTs |

**Constructor:**

| Signature | Description |
|---|---|
| `NewAuthHandler(users *models.UserStore, jwtSecret string) *AuthHandler` | Creates an `AuthHandler` with its dependencies. |

**Methods:**

| Method | Route | Description |
|---|---|---|
| `Signup(w, r)` | `POST /api/auth/signup` | Reads `{"email","password"}`, validates (non-empty, password >= 6 chars), calls `UserStore.Create`, generates JWT, returns `{"token","user"}` with status 201. Returns 400 for validation errors, 409 for duplicate email. |
| `Login(w, r)` | `POST /api/auth/login` | Reads `{"email","password"}`, calls `UserStore.Authenticate`, generates JWT, returns `{"token","user"}` with status 200. Returns 401 with `"username entered does not exist"` or `"password is incorrect"` as required by the spec. |
| `GoogleAuth(w, r)` | `POST /api/auth/google` | Reads `{"google_id","email","name"}`, calls `UserStore.FindOrCreateGoogleUser`, generates JWT, returns `{"token","user","is_new"}`. The `is_new` boolean tells the web frontend whether to redirect to profile-edit or profile-view. Returns 409 if the email belongs to a local account. |

**Request / response types (unexported, used internally):**

| Type | Fields | Used By |
|---|---|---|
| `signupRequest` | `Email`, `Password` | `Signup` |
| `loginRequest` | `Email`, `Password` | `Login` |
| `googleAuthRequest` | `GoogleID`, `Email`, `Name` | `GoogleAuth` |
| `authResponse` | `Token`, `User (*models.User)`, `IsNew` | All three handlers |

**Helper functions (unexported):**

| Function | Description |
|---|---|
| `jsonResponse(w, data, status)` | Sets `Content-Type: application/json`, writes status code, encodes `data` as JSON. |
| `jsonError(w, message, status)` | Writes `{"error":"message"}` with the given HTTP status. |

#### Type: `ProfileHandler`

Handles profile endpoints. Both methods run behind the `JWTAuth` middleware, so the user ID is always available via `middleware.GetUserID(r)`.

| Field | Type | Description |
|---|---|---|
| `Users` | `*models.UserStore` | Data access layer |

**Constructor:**

| Signature | Description |
|---|---|
| `NewProfileHandler(users *models.UserStore) *ProfileHandler` | Creates a `ProfileHandler`. |

**Methods:**

| Method | Route | Description |
|---|---|---|
| `GetProfile(w, r)` | `GET /api/profile` | Extracts user ID from context, calls `UserStore.GetByID`, returns `{"user":{…}}`. Returns 404 if user not found. |
| `UpdateProfile(w, r)` | `PUT /api/profile` | Reads `{"full_name","telephone","email"}`. If the user's `auth_provider` is `"google"`, the submitted email is ignored and the existing email is kept (server-side enforcement). Calls `UserStore.UpdateProfile`, then `GetByID` to return the updated user. Returns 409 if the new email collides with another account. |

---

### 5.8 `internal/api` (router)

**File:** `internal/api/router.go`  
**Role:** Wires together all API routes and middleware into a single `http.Handler`.

#### Function: `NewRouter`

```
NewRouter(users *models.UserStore, jwtSecret string) http.Handler
```

Creates the full API router. Internally:
1. Creates a `gorilla/mux.Router`
2. Applies `corsMiddleware` globally (allows all origins, needed for standalone API use)
3. Registers public endpoints under `/api`:
   - `POST /api/auth/signup`
   - `POST /api/auth/login`
   - `POST /api/auth/google`
4. Creates a protected sub-router under `/api` with `JWTAuth` middleware:
   - `GET /api/profile`
   - `PUT /api/profile`

**Middleware pipeline for a protected request:**

```
Request --> corsMiddleware --> JWTAuth --> ProfileHandler.GetProfile --> Response
```

**Middleware pipeline for a public request:**

```
Request --> corsMiddleware --> AuthHandler.Login --> Response
```

---

### 5.9 `internal/web/client`

**File:** `internal/web/client/client.go`  
**Role:** HTTP client that the web server uses to communicate with the REST API. This is the only point of contact between the web server and the API. It translates Go function calls into HTTP requests with JSON bodies and parses the JSON responses back into Go structs.

#### Type: `APIClient`

| Field | Type | Description |
|---|---|---|
| `BaseURL` | `string` | The API server URL, e.g. `"http://localhost:8080"` or `"http://api:8080"` in Docker |
| `HTTPClient` | `*http.Client` | Standard Go HTTP client used for all requests |

**Constructor:**

| Signature | Description |
|---|---|
| `New(baseURL string) *APIClient` | Creates an `APIClient` pointed at the given API base URL. |

**Methods:**

| Signature | API Call | Description |
|---|---|---|
| `Signup(email, password string) (*AuthResponse, error)` | `POST /api/auth/signup` | Sends credentials, returns token + user data. |
| `Login(email, password string) (*AuthResponse, error)` | `POST /api/auth/login` | Sends credentials, returns token + user data. |
| `GoogleAuth(googleID, email, name string) (*AuthResponse, error)` | `POST /api/auth/google` | Sends Google user info, returns token + user data + `IsNew` flag. |
| `GetProfile(token string) (*ProfileResponse, error)` | `GET /api/profile` | Sends JWT in `Authorization: Bearer` header, returns user data. |
| `UpdateProfile(token, fullName, telephone, email string) (*ProfileResponse, error)` | `PUT /api/profile` | Sends updated fields with JWT, returns updated user data. |

**Response types:**

| Type | Fields | Description |
|---|---|---|
| `AuthResponse` | `Token string`, `User UserData`, `IsNew bool`, `Error string` | Returned by `Signup`, `Login`, `GoogleAuth` |
| `ProfileResponse` | `User UserData`, `Error string` | Returned by `GetProfile`, `UpdateProfile` |
| `UserData` | `ID int`, `Email string`, `AuthProvider string`, `FullName string`, `Telephone string` | User fields needed by templates |

**Internal helpers (unexported):**

| Function | Description |
|---|---|
| `doAuthRequest(method, path, body, token)` | Calls `doRequest`, reads body, unmarshals into `AuthResponse`, returns error if status >= 400. |
| `doRequest(method, path, body, token)` | Builds an `http.Request`, sets `Content-Type: application/json` and optional `Authorization: Bearer` header, executes it, and returns the raw `*http.Response`. |

---

### 5.10 `internal/web/middleware`

**File:** `internal/web/middleware/session.go`  
**Role:** Manages browser session cookies and provides HTTP middleware that enforces authentication on protected pages.

#### Constants

| Name | Value | Description |
|---|---|---|
| `SessionName` | `"session"` | The cookie name used by gorilla/sessions |
| `TokenKey` | `"token"` | The session key under which the JWT is stored |

#### Type: `SessionManager`

Wraps a `gorilla/sessions.CookieStore`.

| Field | Type | Description |
|---|---|---|
| `Store` | `*sessions.CookieStore` | The underlying cookie store |

**Constructor:**

| Signature | Description |
|---|---|
| `NewSessionManager(secret string) *SessionManager` | Creates a `SessionManager`. Configures the cookie with: `Path=/`, `MaxAge=86400` (24h), `HttpOnly=true`, `SameSite=Lax`. |

**Methods:**

| Signature | Description |
|---|---|
| `GetToken(r *http.Request) string` | Reads the JWT string from the session cookie. Returns `""` if no session exists or the token key is absent. |
| `SetToken(w http.ResponseWriter, r *http.Request, token string) error` | Stores the JWT string in the session cookie and saves it. Called after successful login or signup. |
| `Clear(w http.ResponseWriter, r *http.Request) error` | Deletes all session values and sets `MaxAge=-1` to expire the cookie. Called on logout. |
| `RequireAuth(next http.Handler) http.Handler` | Middleware. Sets no-cache headers (see `SetNoCacheHeaders`), then checks if a JWT exists in the session. If not, redirects to `/login`. If yes, calls the next handler. |
| `RedirectIfAuth(next http.Handler) http.Handler` | Middleware. If a JWT exists in the session, redirects to `/profile`. Used on login and signup pages to prevent authenticated users from seeing those forms. |

#### Function: `SetNoCacheHeaders`

```
SetNoCacheHeaders(w http.ResponseWriter)
```

Sets three headers on the response:
- `Cache-Control: no-cache, no-store, must-revalidate`
- `Pragma: no-cache`
- `Expires: 0`

This prevents the browser from caching protected pages. After logout, pressing the back button triggers a fresh request (which `RequireAuth` will redirect to `/login`) instead of showing a cached copy of the profile page.

---

### 5.11 `internal/web/handler`

**Files:** `internal/web/handler/auth.go`, `internal/web/handler/profile.go`  
**Role:** HTTP handlers that render HTML templates and process form submissions. Each handler either renders a page or processes a form, calls the API client, manages the session, and redirects.

#### Type: `PageData`

The struct passed to every template.

| Field | Type | Description |
|---|---|---|
| `Title` | `string` | Page title (`"Login"`, `"Sign Up"`, `"Enter Profile Information"`, `"Main Profile"`) |
| `Error` | `string` | Error message to display (empty if none) |
| `Success` | `string` | Success message to display (empty if none) |
| `User` | `*client.UserData` | User data for pre-filling forms or displaying profile info |
| `IsGoogle` | `bool` | Whether the current user authenticated via Google (used to disable the email field) |

#### Type: `AuthWebHandler`

Handles login, signup, Google OAuth, and logout on the web frontend.

| Field | Type | Description |
|---|---|---|
| `Templates` | `map[string]*template.Template` | Parsed HTML templates keyed by page name |
| `APIClient` | `*client.APIClient` | HTTP client for calling the REST API |
| `Sessions` | `*middleware.SessionManager` | Browser session manager |
| `OAuth` | `*oauth2.Config` | Google OAuth2 configuration |

**Constructor:**

| Signature | Description |
|---|---|
| `NewAuthWebHandler(templates, apiClient, sessions, oauthCfg) *AuthWebHandler` | Creates an `AuthWebHandler` with all its dependencies. |

**Methods:**

| Method | Route | Verb | Description |
|---|---|---|---|
| `LoginPage(w, r)` | `/login` | GET | Renders `login.html`. Reads optional `?error=` query param to show error messages from redirects. |
| `LoginSubmit(w, r)` | `/login` | POST | Reads email/password from form, calls `APIClient.Login`. On success: stores JWT in session, redirects to `/profile`. On error: re-renders login page with error. |
| `SignupPage(w, r)` | `/signup` | GET | Renders `signup.html`. Reads optional `?error=` query param. |
| `SignupSubmit(w, r)` | `/signup` | POST | Reads email/password from form, calls `APIClient.Signup`. On success: stores JWT in session, redirects to `/profile/edit` (new users go to profile setup first). On error: re-renders signup page with error. |
| `GoogleLogin(w, r)` | `/auth/google` | GET | Generates a random OAuth state string, saves it in the session, and redirects the browser to Google's OAuth consent screen. |
| `GoogleCallback(w, r)` | `/auth/google/callback` | GET | Google redirects here after user consents. Verifies the state parameter matches the session value. Exchanges the authorization code for an access token. Fetches user info (id, email, name) from `googleapis.com/oauth2/v2/userinfo`. Calls `APIClient.GoogleAuth` to create or find the user. Stores the JWT in the session. Redirects to `/profile/edit` if `IsNew` is true (first-time user), or `/profile` if false (returning user). |
| `Logout(w, r)` | `/logout` | POST | Clears the session and redirects to `/login`. |

**Private helper:**

| Function | Description |
|---|---|
| `generateOAuthState() string` | Returns 32 bytes of `crypto/rand` data encoded as base64url. Used as the OAuth `state` parameter to prevent CSRF. |

#### Type: `ProfileWebHandler`

Handles profile viewing and editing.

| Field | Type | Description |
|---|---|---|
| `Templates` | `map[string]*template.Template` | Parsed HTML templates |
| `APIClient` | `*client.APIClient` | HTTP client for the REST API |
| `Sessions` | `*middleware.SessionManager` | Browser session manager |

**Constructor:**

| Signature | Description |
|---|---|
| `NewProfileWebHandler(templates, apiClient, sessions) *ProfileWebHandler` | Creates a `ProfileWebHandler`. |

**Methods:**

| Method | Route | Verb | Description |
|---|---|---|---|
| `ViewProfile(w, r)` | `/profile` | GET | Gets JWT from session, calls `APIClient.GetProfile`. If the API call fails (e.g. expired token), clears the session and redirects to `/login`. Otherwise renders `profile_view.html` showing the user's name, telephone, and email in read-only display. |
| `EditProfile(w, r)` | `/profile/edit` | GET | Same API call as above. Renders `profile_edit.html` with form fields pre-populated with the current values. Sets `IsGoogle=true` in `PageData` if the user's `AuthProvider` is `"google"`, which causes the template to render the email field as `disabled`. |
| `EditProfileSubmit(w, r)` | `/profile/edit` | POST | Reads full_name, telephone, email from the form. Calls `APIClient.UpdateProfile`. On success: redirects to `/profile`. On error: redirects back to `/profile/edit?error=…`. |

---

### 5.12 `internal/web` (router)

**File:** `internal/web/router.go`  
**Role:** Wires together all web routes, middleware, templates, and dependencies into a single `http.Handler`.

#### Function: `NewRouter`

```
NewRouter(cfg *config.Config) http.Handler
```

Creates the full web frontend router. Internally:

1. Creates a `SessionManager` with `cfg.SessionSecret`
2. Creates an `APIClient` pointed at `cfg.APIBaseURL`
3. Builds an `oauth2.Config` for Google (client ID, secret, redirect URL, scopes: `openid`, `email`, `profile`)
4. Calls `loadTemplates()` to parse all HTML templates
5. Creates `AuthWebHandler` and `ProfileWebHandler` with their dependencies
6. Builds a `gorilla/mux.Router` with three route groups:

**Route groups and their middleware:**

| Group | Middleware | Routes |
|---|---|---|
| Static | (none) | `/static/` -- serves CSS files from the `static/` directory |
| Public | `RedirectIfAuth` | `GET /login`, `GET /signup` -- if already logged in, redirects to `/profile` |
| Auth forms | (none) | `POST /login`, `POST /signup`, `GET /auth/google`, `GET /auth/google/callback` |
| Protected | `RequireAuth` | `GET /profile`, `GET /profile/edit`, `POST /profile/edit`, `POST /logout` |
| Root | (none) | `GET /` -- redirects to `/login` |

**Middleware pipeline for a protected page:**

```
Request --> RequireAuth (check session, set no-cache headers) --> ProfileWebHandler.ViewProfile --> Response
```

**Middleware pipeline for a public page:**

```
Request --> RedirectIfAuth (redirect if already logged in) --> AuthWebHandler.LoginPage --> Response
```

#### Function: `loadTemplates`

```
loadTemplates() map[string]*template.Template
```

Parses each page template together with the base layout using `template.ParseFiles(base, page)`. Returns a map keyed by page name:

| Key | Files Parsed | Description |
|---|---|---|
| `"login"` | `base.html` + `login.html` | Login page |
| `"signup"` | `base.html` + `signup.html` | Sign-up page |
| `"profile_edit"` | `base.html` + `profile_edit.html` | Profile edit form |
| `"profile_view"` | `base.html` + `profile_view.html` | Profile display |

---

## 6. Database Schema

Single table, auto-created by `database.Migrate()` on API startup:

```sql
CREATE TABLE users (
    id            INT AUTO_INCREMENT PRIMARY KEY,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) DEFAULT '',
    google_id     VARCHAR(255) DEFAULT '',
    auth_provider ENUM('local', 'google') NOT NULL DEFAULT 'local',
    full_name     VARCHAR(255) DEFAULT '',
    telephone     VARCHAR(50) DEFAULT '',
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

| Column | Purpose |
|---|---|
| `email` | The user's login identifier. Unique constraint prevents duplicates. Editable for local users, immutable for Google users. |
| `password_hash` | bcrypt hash of the password. Empty string for Google users (they never use a password). |
| `google_id` | Google's unique user ID. Empty string for local users. Used to find returning Google users via `GetByGoogleID`. |
| `auth_provider` | `'local'` for email/password users, `'google'` for Google OAuth users. Determines which login method is allowed and whether the email can be edited. |

---

## 7. Authentication Architecture

The system uses two separate but connected authentication mechanisms:

```
 ┌─────────────────────────────────────────────────────────────────┐
 │                     WEB SERVER                                  │
 │                                                                 │
 │  Browser  <──cookie──>  SessionManager  <──stores JWT──>  API   │
 │                         (gorilla/sessions)                      │
 │                                                                 │
 │  - Cookie is HttpOnly (no JS access)                            │
 │  - Cookie is SameSite=Lax (CSRF protection)                     │
 │  - Cookie MaxAge = 24h                                          │
 │  - JWT is stored as a session value, not directly in cookie     │
 └─────────────────────────────────────────────────────────────────┘

 ┌─────────────────────────────────────────────────────────────────┐
 │                     API SERVER                                  │
 │                                                                 │
 │  Request  ──>  JWTAuth middleware  ──>  Handler                 │
 │                    │                                             │
 │                    ├── Read "Authorization: Bearer <token>"      │
 │                    ├── Validate HMAC-SHA256 signature            │
 │                    ├── Check expiry                              │
 │                    ├── Extract userID from claims                │
 │                    └── Store userID in request context           │
 │                                                                 │
 │  - JWT expires 24h after creation                               │
 │  - JWT contains only the user ID (no sensitive data)            │
 │  - API is stateless: no session store, no token blacklist       │
 └─────────────────────────────────────────────────────────────────┘
```

**Login flow (condensed):**
1. User submits email + password to web server
2. Web server forwards to API (`POST /api/auth/login`)
3. API verifies credentials, returns JWT
4. Web server stores JWT in session cookie
5. On subsequent requests, web server reads JWT from cookie, sends it to API in `Authorization` header
6. API validates JWT, extracts user ID, serves the request

**Logout flow:**
1. Web server deletes the JWT from the session cookie
2. The JWT itself is not invalidated (stateless API), but the browser can no longer present it
3. No-cache headers prevent the browser from showing cached protected pages

---

## 8. Security Mechanisms

| Threat | Protection |
|---|---|
| Password theft from database | Passwords are hashed with bcrypt (`cost=10`) before storage. The `PasswordHash` field has `json:"-"` so it is never included in API responses. |
| Unauthorized access to protected pages | `RequireAuth` middleware checks for a valid session token. If missing, it returns a 302 redirect to `/login` before the handler executes. The handler also validates the token against the API before rendering any content. |
| Browser back button after logout | All protected pages are served with `Cache-Control: no-cache, no-store, must-revalidate`, `Pragma: no-cache`, and `Expires: 0`. The browser must re-request the page, triggering the auth check. |
| Direct URL access without login | Same as above: `RequireAuth` middleware runs on `/profile`, `/profile/edit`, and `/logout`. |
| Session cookie theft (XSS) | Cookies are set with `HttpOnly=true`, so JavaScript cannot read them. |
| Cross-site request forgery (CSRF) | Cookies use `SameSite=Lax`. The Google OAuth flow uses a random `state` parameter validated on callback. Logout uses `POST` (not `GET`) to prevent CSRF via image tags. |
| Google email spoofing by local user | `FindOrCreateGoogleUser` checks if the email belongs to a local account and returns `ErrLocalAccount` instead of creating a duplicate. |
| Local login by Google user | `Authenticate` checks `auth_provider` and returns `ErrGoogleAccount` if a Google user tries to login with a password. |
| JWT forgery | Tokens are signed with HMAC-SHA256. The secret is never exposed. `ValidateToken` verifies the signing method is HMAC before accepting the token. |
| Google OAuth state CSRF | `GoogleLogin` generates 32 random bytes, stores them in the session, and passes them as the OAuth `state` parameter. `GoogleCallback` verifies the returned state matches the stored value. |

---

## 9. Template System

Templates use Go's `html/template` package with a layout-based composition pattern.

**`base.html`** defines the `"base"` template with a placeholder:
```
{{define "base"}}
  <!DOCTYPE html>
  <html> ... <body>
    <div class="container">
      {{template "content" .}}
    </div>
  </body></html>
{{end}}
```

Each page template defines the `"content"` block:
```
{{define "content"}}
  <h1>Login</h1>
  ...
{{end}}
```

At startup, `loadTemplates()` parses each page template together with `base.html` using `template.ParseFiles("templates/base.html", "templates/login.html")`. The handler calls `tmpl.ExecuteTemplate(w, "base", data)` to render the full page.

**Data flow into templates:**

Every handler creates a `PageData` struct and passes it to the template. The template accesses fields like `{{.Title}}`, `{{.Error}}`, `{{.User.FullName}}`, and `{{.IsGoogle}}`.

| Template | PageData Fields Used |
|---|---|
| `login.html` | `Title`, `Error`, `Success` |
| `signup.html` | `Title`, `Error` |
| `profile_edit.html` | `Title`, `Error`, `User` (all fields), `IsGoogle` (to disable email input) |
| `profile_view.html` | `Title`, `User` (all fields) |