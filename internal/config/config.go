package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken string
	RedisAddress string
	RedisPassword string
	RedisDB int
	VerifiedRoleName string
	UnverifiedRoleName string
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))

	return &Config{
		DiscordToken: os.Getenv("DISCORD_TOKEN"),
		RedisAddress: os.Getenv("REDIS_ADDRESS"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB: redisDB,
		VerifiedRoleName: getEnv("VERIFIED_ROLE_NAME", "Verified"),
		UnverifiedRoleName: getEnv("UNVERIFIED_ROLE_NAME", "Unverified"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}
