package configs

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DbConfig DbConfig
    ImagesPath         string `yaml:"imagesPath"`
    ProductLineArtsPath string `yaml:"productLineArtsPath"`
}

type DbConfig struct {
	Server   string
	Port     int
	User     string
	Password string
	Database string
}

type RedisConfig struct {
	Dsn string
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

	return &Config{
		DbConfig: DbConfig{
			Server:   os.Getenv("SERVER"),
			Port:     port,
			User:     os.Getenv("USER"),
			Password: os.Getenv("PASSWORD"),
			Database: os.Getenv("DATABASE"),
		},
	}
}
