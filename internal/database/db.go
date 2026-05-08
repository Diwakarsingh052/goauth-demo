package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"small-app/internal/config"

	_ "github.com/go-sql-driver/mysql"
)

// Connect establishes a connection to the MySQL database with retry logic.
func Connect(cfg *config.Config) (*sql.DB, error) {
	var db *sql.DB
	var err error

	for i := 1; i <= 30; i++ {
		db, err = sql.Open("mysql", cfg.DSN())
		if err != nil {
			log.Printf("Database open error (attempt %d/30): %v", i, err)
			time.Sleep(2 * time.Second)
			continue
		}

		if err = db.Ping(); err != nil {
			db.Close()
			log.Printf("Waiting for database... (attempt %d/30): %v", i, err)
			time.Sleep(2 * time.Second)
			continue
		}

		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		return db, nil
	}

	return nil, fmt.Errorf("failed to connect to database after 30 attempts: %w", err)
}
