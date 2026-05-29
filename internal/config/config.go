package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken           string
	RedisAddress           string
	RedisPassword          string
	RedisDB                int
	VerifiedRoleName       string
	UnverifiedRoleName     string
	CleanupEnabled         bool
	CleanupIntervalMinutes int
	CleanupMaxAgeHours     int
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	redisDB, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	cleanupInterval, _ := strconv.Atoi(getEnv("CLEANUP_INTERVAL_MINUTES", "30"))
	cleanupMaxAge, _ := strconv.Atoi(getEnv("CLEANUP_MAX_AGE_HOURS", "48"))
	cleanupEnabled := getEnv("CLEANUP_ENABLED", "true") == "true"

	return &Config{
		DiscordToken:           os.Getenv("DISCORD_TOKEN"),
		RedisAddress:           os.Getenv("REDIS_ADDRESS"),
		RedisPassword:          os.Getenv("REDIS_PASSWORD"),
		RedisDB:                redisDB,
		VerifiedRoleName:       getEnv("VERIFIED_ROLE_NAME", "Verified"),
		UnverifiedRoleName:     getEnv("UNVERIFIED_ROLE_NAME", "Unverified"),
		CleanupEnabled:         cleanupEnabled,
		CleanupIntervalMinutes: cleanupInterval,
		CleanupMaxAgeHours:     cleanupMaxAge,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}


