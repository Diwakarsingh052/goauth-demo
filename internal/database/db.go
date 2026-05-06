package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"challange-go-cyaz/internal/config"
)

// Connect establishes a connection to the MySQL database.
// It retries up to 30 times with a 2-second delay to handle Docker startup ordering.
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

// Migrate creates the required database tables if they do not exist.
func Migrate(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INT AUTO_INCREMENT PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) DEFAULT '',
		google_id VARCHAR(255) DEFAULT '',
		auth_provider ENUM('local', 'google') NOT NULL DEFAULT 'local',
		full_name VARCHAR(255) DEFAULT '',
		telephone VARCHAR(50) DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	_, err := db.Exec(query)
	return err
}