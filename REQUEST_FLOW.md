# How Requests Flow from HTML Templates to the Backend

This document explains, step by step, how a user action in the browser (like submitting the profile edit form) travels through the entire system and reaches the database.

---

## The Two Servers

This project runs **two separate servers**:

| Server | Code | Port | Role |
|--------|------|------|------|
| **Web Server** | `cmd/web/main.go` | `WebPort` (e.g. 8080) | Serves HTML pages to the browser. Handles form submissions. Does NOT talk to the database directly. |
| **API Server** | `cmd/api/main.go` | `APIPort` (e.g. 8081) | A REST API. Receives JSON requests, talks to the database, returns JSON responses. |

The browser **only** talks to the Web Server. The Web Server then talks to the API Server behind the scenes. The browser never communicates with the API Server directly.

```
┌─────────┐        HTML/Forms        ┌────────────┐       JSON/HTTP        ┌────────────┐        SQL         ┌──────────┐
│ Browser │ ◄──────────────────────► │ Web Server │ ◄────────────────────► │ API Server │ ◄───────────────► │ Database │
│         │   (port 8080)            │            │   (port 8081)          │            │                    │ (MySQL)  │
└─────────┘                          └────────────┘                        └────────────┘                    └──────────┘
```

---

## Full Example: Editing Your Profile

Let's trace what happens when you change your phone number on the profile edit page.

### Step 1: Browser Shows the Form

When you visit `http://localhost:8080/profile/edit`, your browser sends:

```
GET /profile/edit HTTP/1.1
Host: localhost:8080
```

The Web Server receives this. The route is defined in `internal/web/router.go`:

```go
protected.HandleFunc("/profile/edit", profileHandler.EditProfile).Methods("GET")
```

This calls the `EditProfile` function in `internal/web/handler/profile.go`:

```go
func (h *ProfileWebHandler) EditProfile(w http.ResponseWriter, r *http.Request) {
    token := h.Sessions.GetToken(r)                    // 1. Get JWT token from cookie
    resp, err := h.APIClient.GetProfile(token)         // 2. Call API server to get current user data
    // ...
    h.Templates["profile_edit"].ExecuteTemplate(w, "base", data) // 3. Render HTML template with user data
}
```

Three things happen here:
1. It reads the JWT token from the user's session cookie
2. It calls the API Server to fetch the current user data (so the form can show existing values)
3. It renders `templates/profile_edit.html` and sends the HTML back to the browser

The browser now shows the form with your current name, phone, and email filled in.

---

### Step 2: You Submit the Form

You type a phone number and click "Save & Continue". The form in `templates/profile_edit.html` is:

```html
<form method="POST" action="/profile/edit">
```

This tells the browser: **send a POST request to `/profile/edit`** with the form data. The browser sends:

```
POST /profile/edit HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded

full_name=John+Doe&telephone=5551234567&email=john@example.com
```

The key things to notice:
- `method="POST"` → the browser uses the POST HTTP method
- `action="/profile/edit"` → the browser sends the request to this URL path
- The form data is sent as `key=value` pairs in the request body (from the `name` attribute on each `<input>`)

---

### Step 3: Web Server Receives the Form

The Web Server's router matches `POST /profile/edit` to this route (in `internal/web/router.go`):

```go
protected.HandleFunc("/profile/edit", profileHandler.EditProfileSubmit).Methods("POST")
```

This calls `EditProfileSubmit` in `internal/web/handler/profile.go`:

```go
func (h *ProfileWebHandler) EditProfileSubmit(w http.ResponseWriter, r *http.Request) {
    token := h.Sessions.GetToken(r)          // 1. Get JWT token from session cookie

    fullName := r.FormValue("full_name")      // 2. Extract form field values
    telephone := r.FormValue("telephone")     //    from the POST body
    email := r.FormValue("email")

    _, err := h.APIClient.UpdateProfile(token, fullName, telephone, email)  // 3. Forward to API
    if err != nil {
        http.Redirect(w, r, "/profile/edit?error="+err.Error(), http.StatusSeeOther)
        return
    }

    http.Redirect(w, r, "/profile", http.StatusSeeOther)  // 4. Redirect to profile view
}
```

What happens:
1. It reads the JWT token from the session cookie (this proves who the user is)
2. `r.FormValue("telephone")` extracts the value `5551234567` from the POST body — the key `"telephone"` matches the `name="telephone"` attribute on the HTML input
3. It calls `h.APIClient.UpdateProfile(...)` — this is where the Web Server talks to the API Server
4. If successful, it redirects the browser to `/profile` to see the updated data

**Important:** The Web Server does NOT touch the database. It only forwards the data to the API Server.

---

### Step 4: Web Server Calls the API Server (The API Client)

`h.APIClient` is an instance of `APIClient` defined in `internal/web/client/client.go`. When `UpdateProfile` is called:

```go
func (c *APIClient) UpdateProfile(token, fullName, telephone, email string) (*ProfileResponse, error) {
    body := map[string]string{
        "full_name": fullName,
        "telephone": telephone,
        "email":     email,
    }

    resp, err := c.doRequest("PUT", "/api/profile", body, token)
    // ... reads response JSON ...
}
```

This function:
1. Packages the form data into a JSON object: `{"full_name":"John Doe","telephone":"5551234567","email":"john@example.com"}`
2. Calls `doRequest` which builds an actual HTTP request

```go
func (c *APIClient) doRequest(method, path string, body interface{}, token string) (*http.Response, error) {
    // ... marshals body to JSON ...
    req, err := http.NewRequest(method, c.BaseURL+path, reqBody)   // e.g. PUT http://localhost:8081/api/profile
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)               // attach JWT token
    return c.HTTPClient.Do(req)                                     // send the HTTP request
}
```

So this sends an HTTP request **from the Web Server process to the API Server process**:

```
PUT /api/profile HTTP/1.1
Host: localhost:8081
Content-Type: application/json
Authorization: Bearer eyJhbGciOi...

{"full_name":"John Doe","telephone":"5551234567","email":"john@example.com"}
```

This is a **server-to-server** call. The browser does not see this — it only talks to the Web Server.

---

### Step 5: API Server Receives the Request

The API Server's router (in `internal/api/router.go`) matches `PUT /api/profile`:

```go
protected.HandleFunc("/profile", profileHandler.UpdateProfile).Methods("PUT")
```

But first, the `protected` subrouter runs the JWT middleware (`internal/api/middleware/auth.go`). The middleware:
1. Reads the `Authorization: Bearer <token>` header
2. Decodes and validates the JWT token
3. Extracts the user ID from the token
4. Stores the user ID in the request context so the handler can access it

Then the `UpdateProfile` handler in `internal/api/handler/profile.go` runs:

```go
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
    userID := middleware.GetUserID(r)                     // 1. Get user ID from JWT (set by middleware)

    var req updateProfileRequest
    json.NewDecoder(r.Body).Decode(&req)                  // 2. Parse the JSON body

    telephone, err := validatePhone(req.Telephone)        // 3. Validate phone is 10 digits

    h.usr.UpdateProfile(userID, req.FullName, telephone, email)  // 4. Save to database

    jsonResponse(w, map[string]*users.User{"user": user}, http.StatusOK)  // 5. Return JSON response
}
```

---

### Step 6: Database Update

`h.usr.UpdateProfile` is in `internal/users/user.go`:

```go
func (s *UserStore) UpdateProfile(id int, fullName, telephone, email string) error {
    _, err := s.db.Exec(
        "UPDATE users SET full_name = ?, telephone = ?, email = ?, updated_at = NOW() WHERE id = ?",
        fullName, telephone, email, id,
    )
    return err
}
```

This executes a SQL query against the MySQL database: `UPDATE users SET telephone = '5551234567' ... WHERE id = 42`

---

### Step 7: Response Travels Back

The response flows back through the same chain in reverse:

```
Database → UserStore → API Handler → JSON Response → API Client → Web Handler → Browser Redirect
```

1. **Database** returns success to `UserStore`
2. **API Handler** sends back JSON: `{"user": {"telephone": "5551234567", ...}}`
3. **API Client** (in Web Server) reads this JSON response
4. **Web Handler** sees no error, so it redirects the browser: `HTTP 303 See Other → /profile`
5. **Browser** follows the redirect, loads `/profile`, and you see your updated phone number

---

## Summary: The Complete Chain

```
Browser                 Web Server                          API Server               Database
  │                         │                                    │                       │
  │  POST /profile/edit     │                                    │                       │
  │  (form data)            │                                    │                       │
  │────────────────────────►│                                    │                       │
  │                         │                                    │                       │
  │                         │  PUT /api/profile                  │                       │
  │                         │  (JSON + JWT token)                │                       │
  │                         │───────────────────────────────────►│                       │
  │                         │                                    │                       │
  │                         │                                    │  UPDATE users SET...  │
  │                         │                                    │─────────────────────► │
  │                         │                                    │                       │
  │                         │                                    │  ◄── OK               │
  │                         │                                    │                       │
  │                         │  ◄── {"user": {...}}               │                       │
  │                         │                                    │                       │
  │  ◄── Redirect /profile  │                                    │                       │
  │                         │                                    │                       │
```

## Key Takeaways

1. **The `name` attribute on `<input>` elements is what connects HTML to Go.** When the form has `<input name="telephone">`, the Go handler reads it with `r.FormValue("telephone")` — the string must match exactly.

2. **The Web Server is a middleman.** It receives HTML form submissions from the browser, converts them to JSON API calls, and forwards them to the API Server. It never touches the database.

3. **The API Client (`internal/web/client/client.go`) is the bridge.** It converts Go function calls into HTTP requests that the API Server understands.

4. **JWT tokens link everything together.** When you log in, you get a token stored in a session cookie. Every request sends this token so the API Server knows who you are without needing your password again.

5. **Two data formats are involved.** Browser → Web Server uses HTML form encoding (`key=value&key=value`). Web Server → API Server uses JSON (`{"key": "value"}`). The Web Handler converts between the two.