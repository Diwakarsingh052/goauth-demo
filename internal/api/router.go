package api

import (
	"net/http"

	"github.com/gorilla/mux"

	"challenge-go-cyaz/internal/api/handler"
	"challenge-go-cyaz/internal/api/middleware"
	"challenge-go-cyaz/internal/auth"
	"challenge-go-cyaz/internal/users"
)

// NewRouter creates and configures the REST API router.
func NewRouter(users *users.UserStore, jwtSecret, corsOrigin string) http.Handler {
	r := mux.NewRouter()
	r.Use(corsMiddleware(corsOrigin))

	tokens := auth.NewTokenService(jwtSecret)
	authHandler := handler.NewAuthHandler(users, tokens)
	profileHandler := handler.NewProfileHandler(users)

	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/auth/signup", authHandler.Signup).Methods("POST")
	api.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
	api.HandleFunc("/auth/google/signup", authHandler.GoogleSignup).Methods("POST")
	api.HandleFunc("/auth/google/login", authHandler.GoogleLogin).Methods("POST")

	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.JWTAuth(tokens))
	protected.HandleFunc("/profile", profileHandler.GetProfile).Methods("GET")
	protected.HandleFunc("/profile", profileHandler.UpdateProfile).Methods("PUT")

	return r
}

func corsMiddleware(allowedOrigin string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
