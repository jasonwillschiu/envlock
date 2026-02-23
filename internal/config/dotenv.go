package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// LoadDotenvIfPresent reads a local .env file for development and local CLI use.
// It does not override existing environment variables and is a no-op when the
// file is absent.
func LoadDotenvIfPresent() {
	if strings.EqualFold(os.Getenv("ENVLOCK_ENV"), "production") {
		return
	}

	if _, err := os.Stat(".env"); err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("dotenv stat error: %v", err)
		return
	}

	if err := godotenv.Load(".env"); err != nil {
		log.Printf("dotenv load error: %v", err)
	}
}
