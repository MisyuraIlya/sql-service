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
		ImagesPath:          getEnv("IMAGES_PATH", defaultBitmapsPath+`\ProductImages`),
		ProductLineArtsPath: getEnv("PRODUCT_LINEARTS_PATH", defaultBitmapsPath+`\Productlinearts`),
	}
}

// The SAP B1 share (b1_shr) lives on the same host as this service, so the
// bitmaps are reached on local disk rather than over SMB. Override either path
// via the environment if the share moves off this box - a UNC path such as
// \\SRV-DC-TLITE\b1_shr\Bitmaps\ProductImages works too.
const defaultBitmapsPath = `C:\Program Files (x86)\SAP\SAP Business One Server\B1_SHR\Bitmaps`

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
