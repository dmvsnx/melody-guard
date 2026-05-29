package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/savanyv/melody-guard/internal/bot"
	"github.com/savanyv/melody-guard/internal/config"
	"github.com/savanyv/melody-guard/internal/services"
	"github.com/savanyv/melody-guard/internal/store"
)

func main() {
	cfg := config.LoadConfig()

	if cfg.DiscordToken == "" {
		log.Fatal("DISCORD_TOKEN is required")
	}

	rdb := store.NewRedisClient(cfg.RedisAddress, cfg.RedisPassword, cfg.RedisDB)
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	verifyRepo := store.NewRedisRepository(rdb)

	rolesConfig := config.NewRoleConfig(nil, cfg.VerifiedRoleName, cfg.UnverifiedRoleName)
	verifyService := services.NewService(verifyRepo, rolesConfig)

	discordBot, err := bot.NewBot(cfg.DiscordToken, verifyService)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	rolesConfig.Session = discordBot.GetSession()

	if err := discordBot.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}
	defer discordBot.Stop()

	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()

	if cfg.CleanupEnabled {
		verifyService.StartCleanupJob(
			cleanupCtx,
			time.Duration(cfg.CleanupIntervalMinutes)*time.Minute,
			time.Duration(cfg.CleanupMaxAgeHours)*time.Hour,
		)
	}

	log.Println("✅ MelodyGuard is now running. Press CTRL+C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	cleanupCancel()
	log.Println("🛑 Shutting down MelodyGuard.")
}
