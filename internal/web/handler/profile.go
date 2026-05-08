package handler

import (
	"log"
	"net/http"
)

// ViewProfile renders the main profile page with user data.
func (h *Handler) ViewProfile(w http.ResponseWriter, r *http.Request) {
	resp, err := h.api.GetProfile(h.sessions.GetToken(r))
	if err != nil {
		if clearErr := h.sessions.Clear(w, r); clearErr != nil {
			log.Printf("failed to clear session after profile fetch error: %v", clearErr)
		}
		http.Redirect(w, r, "/login?error=Session+expired,+please+login+again", http.StatusSeeOther)
		return
	}

	h.render(w, "profile_view", PageData{Title: "Main Profile", User: &resp.User})
}

// EditProfile renders the profile edit form with current user data.
func (h *Handler) EditProfile(w http.ResponseWriter, r *http.Request) {
	resp, err := h.api.GetProfile(h.sessions.GetToken(r))
	if err != nil {
		if clearErr := h.sessions.Clear(w, r); clearErr != nil {
			log.Printf("failed to clear session after profile fetch error: %v", clearErr)
		}
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

	h.render(w, "profile_edit", data)
}

// EditProfileSubmit saves the updated profile fields.
func (h *Handler) EditProfileSubmit(w http.ResponseWriter, r *http.Request) {
	_, err := h.api.UpdateProfile(h.sessions.GetToken(r), r.FormValue("full_name"), r.FormValue("telephone"), r.FormValue("email"))
	if err != nil {
		http.Redirect(w, r, "/profile/edit?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}