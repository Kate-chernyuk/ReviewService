// config/database.go
package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"PR/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func NewDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "pai626p"),
		DBName:   getEnv("DB_NAME", "pr_review_service"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

func (c *DatabaseConfig) GetDSN(dbname string) string {
	if dbname == "" {
		dbname = c.DBName
	}
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		c.Host, c.User, c.Password, dbname, c.Port, c.SSLMode)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func InitDB() (*gorm.DB, error) {
	config := NewDatabaseConfig()

	db, err := gorm.Open(postgres.Open(config.GetDSN("")))
	if err != nil {
		log.Printf("База данных %s не существует, пытаемся создать...", config.DBName)

		tempDB, err := gorm.Open(postgres.Open(config.GetDSN("postgres")))
		if err != nil {
			return nil, fmt.Errorf("не удалось подключиться к postgres: %w", err)
		}

		createDBSQL := fmt.Sprintf("CREATE DATABASE %s", config.DBName)
		result := tempDB.Exec(createDBSQL)
		if result.Error != nil {
			if !isDatabaseExistsError(result.Error) {
				return nil, fmt.Errorf("не удалось создать базу данных %s: %w", config.DBName, result.Error)
			}
			log.Printf("База данных %s уже существует", config.DBName)
		} else {
			log.Printf("База данных %s создана успешно", config.DBName)
		}

		sqlTempDB, _ := tempDB.DB()
		sqlTempDB.Close()

		db, err = gorm.Open(postgres.Open(config.GetDSN("")))
		if err != nil {
			return nil, fmt.Errorf("не удалось подключиться к новой базе данных: %w", err)
		}
	} else {
		log.Printf("Успешно подключились к существующей базе данных %s", config.DBName)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := db.AutoMigrate(
		&models.Team{},
		&models.User{},
		&models.PullRequest{},
	); err != nil {
		return nil, fmt.Errorf("ошибка миграции базы данных: %w", err)
	}

	log.Println("Успешное подключение к БД и миграция")
	return db, nil
}

// Вспомогательная функция для проверки ошибки "база уже существует"
func isDatabaseExistsError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	// Проверяем различные варианты ошибки "база уже существует"
	return contains(errorStr, "already exists") ||
		contains(errorStr, "уже существует") ||
		contains(errorStr, "SQLSTATE 42P04")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > len(substr) && (s[:len(substr)] == substr ||
			contains(s[1:], substr)))
}
