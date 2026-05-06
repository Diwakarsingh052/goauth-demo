-- Migration: 001_init.sql
-- Creates the users table for the authentication system.

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;