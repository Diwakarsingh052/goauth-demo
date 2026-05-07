package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIClient provides methods to communicate with the REST API.
type APIClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// New creates a new APIClient.
func New(baseURL string) *APIClient {
	return &APIClient{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{},
	}
}

// AuthResponse represents the response from auth endpoints.
type AuthResponse struct {
	Token string   `json:"token"`
	User  UserData `json:"user"`

	Error string   `json:"error,omitempty"`
}

// ProfileResponse represents the response from profile endpoints.
type ProfileResponse struct {
	User  UserData `json:"user"`
	Error string   `json:"error,omitempty"`
}

// UserData holds user information returned by the API.
type UserData struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	AuthProvider string `json:"auth_provider"`
	FullName     string `json:"full_name"`
	Telephone    string `json:"telephone"`
}

// Signup calls POST /api/auth/signup.
func (c *APIClient) Signup(email, password string) (*AuthResponse, error) {
	body := map[string]string{"email": email, "password": password}
	return c.doAuthRequest("POST", "/api/auth/signup", body, "")
}

// Login calls POST /api/auth/login.
func (c *APIClient) Login(email, password string) (*AuthResponse, error) {
	body := map[string]string{"email": email, "password": password}
	return c.doAuthRequest("POST", "/api/auth/login", body, "")
}

// GoogleSignup calls POST /api/auth/google/signup.
func (c *APIClient) GoogleSignup(googleID, email, name string) (*AuthResponse, error) {
	body := map[string]string{"google_id": googleID, "email": email, "name": name}
	return c.doAuthRequest("POST", "/api/auth/google/signup", body, "")
}

// GoogleLogin calls POST /api/auth/google/login.
func (c *APIClient) GoogleLogin(googleID, email string) (*AuthResponse, error) {
	body := map[string]string{"google_id": googleID, "email": email}
	return c.doAuthRequest("POST", "/api/auth/google/login", body, "")
}

// GetProfile calls GET /api/profile.
func (c *APIClient) GetProfile(token string) (*ProfileResponse, error) {
	resp, err := c.doRequest("GET", "/api/profile", nil, token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result ProfileResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("API error: %s", result.Error)
	}
	return &result, nil
}

// UpdateProfile calls PUT /api/profile.
func (c *APIClient) UpdateProfile(token, fullName, telephone, email string) (*ProfileResponse, error) {
	body := map[string]string{
		"full_name": fullName,
		"telephone": telephone,
		"email":     email,
	}

	resp, err := c.doRequest("PUT", "/api/profile", body, token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result ProfileResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("%s", result.Error)
	}
	return &result, nil
}

func (c *APIClient) doAuthRequest(method, path string, body interface{}, token string) (*AuthResponse, error) {
	resp, err := c.doRequest(method, path, body, token)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result AuthResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return &result, fmt.Errorf("%s", result.Error)
	}
	return &result, nil
}

func (c *APIClient) doRequest(method, path string, body interface{}, token string) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return c.HTTPClient.Do(req)
}