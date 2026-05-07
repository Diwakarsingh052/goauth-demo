package handler

import (
	"html/template"
	"net/http"

	"challenge-go-cyaz/internal/web/client"
	"challenge-go-cyaz/internal/web/middleware"
)

// ProfileWebHandler handles web profile pages.
type ProfileWebHandler struct {
	Templates map[string]*template.Template
	APIClient *client.APIClient
	Sessions  *middleware.SessionManager
}

// NewProfileWebHandler creates a new ProfileWebHandler.
func NewProfileWebHandler(
	templates map[string]*template.Template,
	apiClient *client.APIClient,
	sessions *middleware.SessionManager,
) *ProfileWebHandler {
	return &ProfileWebHandler{
		Templates: templates,
		APIClient: apiClient,
		Sessions:  sessions,
	}
}

// ViewProfile renders the main profile page (2-D).
func (h *ProfileWebHandler) ViewProfile(w http.ResponseWriter, r *http.Request) {
	token := h.Sessions.GetToken(r)

	resp, err := h.APIClient.GetProfile(token)
	if err != nil {
		h.Sessions.Clear(w, r)
		http.Redirect(w, r, "/login?error=Session+expired,+please+login+again", http.StatusSeeOther)
		return
	}

	data := PageData{
		Title: "Main Profile",
		User:  &resp.User,
	}
	h.Templates["profile_view"].ExecuteTemplate(w, "base", data)
}

// EditProfile renders the edit profile page (2-C).
func (h *ProfileWebHandler) EditProfile(w http.ResponseWriter, r *http.Request) {
	token := h.Sessions.GetToken(r)

	resp, err := h.APIClient.GetProfile(token)
	if err != nil {
		h.Sessions.Clear(w, r)
		http.Redirect(w, r, "/login?error=Session+expired,+please+login+again", http.StatusSeeOther)
		return
	}

	data := PageData{
		Title:    "Enter Profile Information",
		User:     &resp.User,
		IsGoogle: resp.User.AuthProvider == "google",
	}

	if msg := r.URL.Query().Get("error"); msg != "" {
		data.Error = msg
	}

	h.Templates["profile_edit"].ExecuteTemplate(w, "base", data)
}

// EditProfileSubmit processes the profile edit form.
func (h *ProfileWebHandler) EditProfileSubmit(w http.ResponseWriter, r *http.Request) {
	token := h.Sessions.GetToken(r)

	fullName := r.FormValue("full_name")
	telephone := r.FormValue("telephone")
	email := r.FormValue("email")

	_, err := h.APIClient.UpdateProfile(token, fullName, telephone, email)
	if err != nil {
		http.Redirect(w, r, "/profile/edit?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}