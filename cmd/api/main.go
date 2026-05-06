package main

import (
	"log"
	"net/http"

	"challange-go-cyaz/internal/api"
	"challange-go-cyaz/internal/config"
	"challange-go-cyaz/internal/database"
	"challange-go-cyaz/internal/users"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file %f", err)
	}

	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed successfully")

	users := users.NewUserStore(db)
	router := api.NewRouter(users, cfg.JWTSecret)

	log.Printf("REST API server starting on :%s", cfg.APIPort)
	if err := http.ListenAndServe(":"+cfg.APIPort, router); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}
