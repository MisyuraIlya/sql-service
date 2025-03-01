package configs

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Redis RedisConfig
}

type RedisConfig struct {
	Dsn string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, using default config")
	}
	return &Config{
		Redis: RedisConfig{
			Dsn: os.Getenv("REDIS_ADDR"),
		},
	}
}
