package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds all application configuration.
type Config struct {
	DBHost             string
	DBPort             string
	DBUser             string
	DBPassword         string
	DBName             string
	APIPort            string
	WebPort            string
	APIBaseURL         string
	JWTSecret          string
	SessionSecret      string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		DBHost:             getEnv("DB_HOST", "localhost"),
		DBPort:             getEnv("DB_PORT", "3306"),
		DBUser:             getEnv("DB_USER", "root"),
		DBPassword:         getEnv("DB_PASSWORD", ""),
		DBName:             getEnv("DB_NAME", "challange_go"),
		APIPort:            getEnv("API_PORT", "8080"),
		WebPort:            getEnv("WEB_PORT", "8081"),
		APIBaseURL:         getEnv("API_BASE_URL", "http://localhost:8080"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-to-a-secure-secret"),
		SessionSecret:      getEnv("SESSION_SECRET", "change-me-to-a-session-secret"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8081/auth/google/callback"),
	}
}

// DSN returns the MySQL Data Source Name connection string.
func (c *Config) DSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName + "?parseTime=true"
}

// LoadEnvFile loads key=value pairs from a .env file into the process environment.
func LoadEnvFile(path string) error {
	err := godotenv.Load(path)
	if err != nil {
		return err
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
