# Go Authentication Challenge

A full-stack Go web application featuring user authentication with email/password and Google OAuth 2.0, profile management, and a clean separation between a REST API backend and a server-rendered frontend using Go templates.

## Architecture

```
                       Docker Compose
 ┌──────────────────────────────────────────────────────────┐
 │                                                          │
 │  ┌──────────────┐       ┌──────────────┐                │
 │  │  Web Frontend │ HTTP  │  REST API    │                │
 │  │  (Go Templ.)  │──────>│  (JSON)      │                │
 │  │  :8081        │       │  :8080       │                │
 │  │               │       │              │                │
 │  │ - HTML render │       │ - Business   │  ┌───────────┐ │
 │  │ - Sessions    │       │   logic      │  │  MySQL    │ │
 │  │ - OAuth flow  │       │ - JWT auth   │─>│  :3306    │ │
 │  └──────────────┘       └──────────────┘  └───────────┘ │
 │                                                          │
 └──────────────────────────────────────────────────────────┘
```

The application is split into **two independent servers**:

1. **REST API** (`cmd/api`) -- Handles all business logic, database operations, and authentication via JWT tokens. Fully independent and can be consumed by any frontend technology.

2. **Web Frontend** (`cmd/web`) -- Server-rendered Go templates that communicate with the REST API via HTTP. Manages browser sessions using cookies.

## Project Structure

```
.
├── cmd/
│   ├── api/
│   │   └── main.go                 # REST API server entry point
│   └── web/
│       └── main.go                 # Web frontend server entry point
├── internal/
│   ├── config/
│   │   └── config.go               # Configuration management
│   ├── database/
│   │   └── db.go                   # MySQL connection and migrations
│   ├── models/
│   │   └── user.go                 # User model and data access layer
│   ├── api/
│   │   ├── handler/
│   │   │   ├── auth.go             # Auth API handlers (signup, login, google)
│   │   │   ├── auth_test.go        # Auth handler integration tests
│   │   │   ├── profile.go          # Profile API handlers (get, update)
│   │   │   └── profile_test.go     # Profile handler integration tests
│   │   ├── middleware/
│   │   │   ├── auth.go             # JWT authentication middleware
│   │   │   └── auth_test.go        # JWT middleware unit tests
│   │   └── router.go               # API route definitions
│   └── web/
│       ├── client/
│       │   └── client.go           # HTTP client for API communication
│       ├── handler/
│       │   ├── auth.go             # Web auth handlers (pages + forms)
│       │   └── profile.go          # Web profile handlers
│       ├── middleware/
│       │   └── session.go          # Session management + auth middleware
│       └── router.go               # Web route definitions + template loading
├── templates/
│   ├── base.html                   # Base layout template
│   ├── login.html                  # Login page (2-A)
│   ├── signup.html                 # Sign-up page (2-B)
│   ├── profile_edit.html           # Profile edit page (2-C)
│   └── profile_view.html           # Main profile page (2-D)
├── static/
│   └── css/
│       └── style.css               # Application styles
├── migrations/
│   └── 001_init.sql                # Database schema
├── Dockerfile                      # Multi-stage Docker build
├── docker-compose.yml              # Full stack orchestration
├── .env.example                    # Environment variable template
├── .gitignore
├── Makefile                        # Build, run, test, Docker commands
├── go.mod
├── go.sum
└── README.md
```

## Prerequisites

### Option A: Docker (Recommended)

- **Docker** and **Docker Compose**
- **Google OAuth 2.0 credentials** (for Google sign-in)

### Option B: Local Development

- **Go** 1.21 or later
- **MySQL** 5.7 or later
- **Google OAuth 2.0 credentials** (for Google sign-in)

---

## Quick Start with Docker

The fastest way to get the full stack running:

```bash
# 1. Clone and enter the project
git clone <repository-url>
cd challange-go-cyaz

# 2. Create environment file
cp .env.example .env
# Edit .env and add your Google OAuth credentials (see section below)

# 3. Start everything (MySQL + API + Web)
make docker-up

# 4. Open in browser
open http://localhost:8081
```

This starts three containers:
- **MySQL** on port 3306 (with automatic schema migration)
- **REST API** on port 8080
- **Web Frontend** on port 8081

### Docker Commands

| Command | Description |
|---|---|
| `make docker-build` | Build Docker images |
| `make docker-up` | Start all services in background |
| `make docker-down` | Stop all services |
| `make docker-logs` | Tail logs from all containers |
| `make docker-restart` | Restart all services |
| `make docker-clean` | Stop, remove volumes and images |

---

## Local Development Setup

### 1. Clone the Repository

```bash
git clone <repository-url>
cd challange-go-cyaz
```

### 2. Configure Environment

```bash
cp .env.example .env
```

Edit `.env` and configure:

| Variable | Description | Default |
|---|---|---|
| `DB_HOST` | MySQL host | `localhost` |
| `DB_PORT` | MySQL port | `3306` |
| `DB_USER` | MySQL user | `root` |
| `DB_PASSWORD` | MySQL password | (empty) |
| `DB_NAME` | Database name | `challange_go` |
| `API_PORT` | REST API server port | `8080` |
| `WEB_PORT` | Web frontend port | `8081` |
| `API_BASE_URL` | Full URL of the API | `http://localhost:8080` |
| `JWT_SECRET` | Secret key for JWT tokens | (change this!) |
| `SESSION_SECRET` | Secret key for session cookies | (change this!) |
| `GOOGLE_CLIENT_ID` | Google OAuth client ID | |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret | |
| `GOOGLE_REDIRECT_URL` | Google OAuth callback URL | `http://localhost:8081/auth/google/callback` |

### 3. Create the Database

```bash
make setup
```

Or manually:

```sql
CREATE DATABASE challange_go;
CREATE DATABASE challange_go_test;  -- for running tests
```

### 4. Install Dependencies

```bash
make deps
```

### 5. Run the Application

Open **two terminal windows**:

**Terminal 1 -- Start the REST API:**
```bash
make run-api
```

**Terminal 2 -- Start the Web Frontend:**
```bash
make run-web
```

The application will be available at **http://localhost:8081**

---

## Setting Up Google OAuth 2.0

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Navigate to **APIs & Services** > **Credentials**
4. Click **Create Credentials** > **OAuth client ID**
5. Select **Web application**
6. Add `http://localhost:8081/auth/google/callback` as an **Authorized redirect URI**
7. Copy the **Client ID** and **Client Secret** into your `.env` file

## REST API Endpoints

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/auth/signup` | No | Register with email and password |
| `POST` | `/api/auth/login` | No | Login with email and password |
| `POST` | `/api/auth/google` | No | Login/register with Google account |
| `GET` | `/api/profile` | JWT | Get current user's profile |
| `PUT` | `/api/profile` | JWT | Update current user's profile |

### Example API Calls

**Sign Up:**
```bash
curl -X POST http://localhost:8080/api/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "mypassword"}'
```

**Login:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "mypassword"}'
```

**Get Profile (with JWT):**
```bash
curl http://localhost:8080/api/profile \
  -H "Authorization: Bearer <your-jwt-token>"
```

**Update Profile:**
```bash
curl -X PUT http://localhost:8080/api/profile \
  -H "Authorization: Bearer <your-jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{"full_name": "John Doe", "telephone": "555-1234", "email": "john@example.com"}'
```

## Web Pages

| Path | Page | Description |
|------|------|-------------|
| `/login` | Login (2-A) | Home page with email/password login and Google sign-in |
| `/signup` | Sign Up (2-B) | Registration with email/password or Google |
| `/profile/edit` | Enter Profile Info (2-C) | Create or edit profile (full name, telephone, email) |
| `/profile` | Main Profile (2-D) | View profile with Edit and Logout buttons |

## Running Tests

**Unit tests** (no database required):
```bash
make test-unit
```

**Integration tests** (requires MySQL `challange_go_test` database):
```bash
make test-integration
```

**All tests:**
```bash
make test
```

## Security Features

- **JWT Authentication**: API endpoints are protected with JWT tokens (24-hour expiry)
- **Bcrypt Password Hashing**: Passwords are hashed using bcrypt before storage
- **Session Security**: Browser sessions use HttpOnly, SameSite cookies
- **No-Cache Headers**: Protected pages include `Cache-Control: no-store` to prevent viewing via browser back button after logout
- **Auth Middleware**: All protected routes enforce authentication; unauthenticated access redirects to login
- **CSRF via SameSite**: Cookie SameSite attribute provides CSRF protection
- **OAuth State Parameter**: Google OAuth flow uses random state parameter to prevent CSRF

## User Flow

```
  (2-A) Login Page       (2-B) Sign-Up Page
  ┌──────────────┐       ┌──────────────┐
  │ Email + Pass │       │ Email + Pass │
  │ Google Login │       │ Google SignUp │
  │              │──────>│              │
  │ [Sign Up ->] │       │ [<- Login]   │
  └──────┬───────┘       └──────┬───────┘
         │ login success         │ signup success
         │                       │
         v                       v
  (2-D) Main Profile     (2-C) Edit Profile
  ┌──────────────┐       ┌──────────────┐
  │ View-only    │       │ Full Name    │
  │ Name / Phone │       │ Telephone    │
  │ Email        │<──────│ Email        │
  │              │ save  │              │
  │ [Edit] [Out] │──────>│ [Save] [Cancel]
  └──────────────┘ edit  └──────────────┘
```

- New users (email or Google) always go to **2-C** first to fill in profile info
- Existing users logging in go directly to **2-D**
- Google users cannot edit their email in **2-C**
- Logout expires the session and returns to **2-A**
- Protected pages (**2-C**, **2-D**) cannot be accessed without authentication, even via back button or direct URL

## Technologies Used

- **Backend**: Go standard library + Gorilla toolkit (mux, sessions)
- **Database**: MySQL with `go-sql-driver/mysql`
- **Authentication**: JWT (`golang-jwt/jwt`), bcrypt (`golang.org/x/crypto`), Google OAuth2 (`golang.org/x/oauth2`)
- **Frontend**: Go `html/template`, vanilla CSS
- **Containerization**: Docker multi-stage builds, Docker Compose
- **No JavaScript frameworks**: Pure server-rendered pages with minimal vanilla JS