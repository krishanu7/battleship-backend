package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl     string
	JWTSecret string
}

func LoadConfig() Config {
	err := godotenv.Load()

	if err != nil {
		log.Println("No .env file found. Using environment variables.")
	}

	return Config{
		DBUrl:     os.Getenv("DB_URL"),
		JWTSecret: os.Getenv("JWT_SECRET"),
	}
}
