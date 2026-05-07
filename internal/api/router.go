package api

import (
	"net/http"

	"github.com/gorilla/mux"

	"challenge-go-cyaz/internal/api/handler"
	"challenge-go-cyaz/internal/api/middleware"
	"challenge-go-cyaz/internal/users"
)

// NewRouter creates and configures the REST API router.
func NewRouter(users *users.UserStore, jwtSecret string) http.Handler {
	r := mux.NewRouter()
	r.Use(corsMiddleware)

	authHandler := handler.NewAuthHandler(users, jwtSecret)
	profileHandler := handler.NewProfileHandler(users)

	// Public auth endpoints
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/auth/signup", authHandler.Signup).Methods("POST")
	api.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
	api.HandleFunc("/auth/google/signup", authHandler.GoogleSignup).Methods("POST")
	api.HandleFunc("/auth/google/login", authHandler.GoogleLogin).Methods("POST")

	// Protected endpoints
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.JWTAuth(jwtSecret))
	protected.HandleFunc("/profile", profileHandler.GetProfile).Methods("GET")
	protected.HandleFunc("/profile", profileHandler.UpdateProfile).Methods("PUT")

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
