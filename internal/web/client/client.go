package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// APIClient communicates with the REST API server over HTTP.
type APIClient struct {
	baseURL string
	http    *http.Client
}

// New creates an APIClient pointing at the given base URL.
func New(baseURL string) *APIClient {
	return &APIClient{baseURL: baseURL, http: &http.Client{}}
}

// APIResponse is the unified response structure from all API endpoints.
type APIResponse struct {
	Token string   `json:"token,omitempty"`
	User  UserData `json:"user"`
	Error string   `json:"error,omitempty"`
}

// UserData holds user fields returned by the API.
type UserData struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	AuthProvider string `json:"auth_provider"`
	FullName     string `json:"full_name"`
	Telephone    string `json:"telephone"`
}

// Signup creates a new user account with email and password.
func (c *APIClient) Signup(email, password string) (*APIResponse, error) {
	resp, err := c.do("POST", "/api/auth/signup", map[string]string{"email": email, "password": password}, "")
	return resp, err
}

// Login authenticates an existing user with email and password.
func (c *APIClient) Login(email, password string) (*APIResponse, error) {
	resp, err := c.do("POST", "/api/auth/login", map[string]string{"email": email, "password": password}, "")
	return resp, err
}

// GoogleSignup creates or finds a user account via Google OAuth.
func (c *APIClient) GoogleSignup(googleID, email, name string) (*APIResponse, error) {
	resp, err := c.do("POST", "/api/auth/google/signup", map[string]string{"google_id": googleID, "email": email, "name": name}, "")
	return resp, err
}

// GoogleLogin finds an existing Google user account.
func (c *APIClient) GoogleLogin(googleID, email string) (*APIResponse, error) {
	resp, err := c.do("POST", "/api/auth/google/login", map[string]string{"google_id": googleID, "email": email}, "")
	return resp, err
}

// GetProfile fetches the authenticated user's profile.
func (c *APIClient) GetProfile(token string) (*APIResponse, error) {
	resp, err := c.do("GET", "/api/profile", nil, token)
	return resp, err
}

// UpdateProfile saves updated profile fields for the authenticated user.
func (c *APIClient) UpdateProfile(token, fullName, telephone, email string) (*APIResponse, error) {
	resp, err := c.do("PUT", "/api/profile", map[string]string{"full_name": fullName, "telephone": telephone, "email": email}, token)
	return resp, err
}

// do sends an HTTP request to the API and decodes the JSON response.
func (c *APIClient) do(method, path string, body any, token string) (*APIResponse, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, c.baseURL+path, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s", result.Error)
	}
	return &result, nil
}
