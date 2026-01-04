package configs

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DbConfig            DbConfig
	ImagesPath          string
	ProductLineArtsPath string
}

type DbConfig struct {
	Dialect  string
	DSN      string
	Server   string
	Port     int
	User     string
	Password string
	Database string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, using default config")
	}

	portStr := os.Getenv("PORT")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Printf("Invalid PORT value: %v. Using default port 3306.", portStr)
		port = 3306
	}

	dialect := strings.TrimSpace(strings.ToLower(os.Getenv("DB_DIALECT")))
	if dialect == "" {
		dialect = "mssql"
	}

	return &Config{
		DbConfig: DbConfig{
			Dialect:  dialect,
			DSN:      strings.TrimSpace(os.Getenv("DB_DSN")),
			Server:   os.Getenv("SERVER"),
			Port:     port,
			User:     os.Getenv("USER"),
			Password: os.Getenv("PASSWORD"),
			Database: os.Getenv("DATABASE"),
		},
		ImagesPath:          `\\192.168.2.41\b1_shr\Bitmaps\ProductImages`,
		ProductLineArtsPath: `\\192.168.2.41\b1_shr\Bitmaps\Productlinearts`,
	}
}
