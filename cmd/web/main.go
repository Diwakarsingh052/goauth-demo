package main

import (
	"challenge-go-cyaz/internal/web"
	"log"
	"net/http"

	"challenge-go-cyaz/internal/config"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file %v", err)
	}
	cfg := config.Load()

	router := web.NewRouter(cfg)

	log.Printf("Web server starting on :%s", cfg.WebPort)
	log.Printf("Open http://localhost:%s in your browser", cfg.WebPort)
	if err := http.ListenAndServe(":"+cfg.WebPort, router); err != nil {
		log.Fatalf("Web server failed: %v", err)
	}
}
