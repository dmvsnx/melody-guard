package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/savanyv/melody-guard/internal/config"
	"github.com/savanyv/melody-guard/internal/store"
)

type Service interface {
	VerifyUser(ctx context.Context, guildID, userID string) error
	hasRole(guildID, userID, roleID string) (bool, error)
	UnverifyUser(ctx context.Context, guildID, userID string) error
	CheckVerification(ctx context.Context, guildID, userID string) (bool, error)
	GetOrSetupRoles(guildID string) (verifiedRoleID, unverifiedRoleID string, err error)
	HandleUserLeave(ctx context.Context, guildID, userID string) error
	StartCleanupJob(ctx context.Context, interval, maxAge time.Duration)
	RecordJoinTime(ctx context.Context, guildID, userID string) error
	RemoveJoinTime(ctx context.Context, guildID, userID string) error
}

type service struct {
	repo       store.VerifyRepository
	roleConfig *config.RolesConfig
	cleanupMu  sync.Mutex
}

func NewService(repo store.VerifyRepository, roleConfig *config.RolesConfig) Service {
	return &service{
		repo:       repo,
		roleConfig: roleConfig,
	}
}

func (s *service) VerifyUser(ctx context.Context, guildID, userID string) error {
	_, unverifiedRoleID, err := s.GetOrSetupRoles(guildID)
	if err != nil {
		return fmt.Errorf("failed to get roles: %v", err)
	}

	hasUnverified, err := s.hasRole(guildID, userID, unverifiedRoleID)
	if err != nil {
		return fmt.Errorf("failed to check roles: %v", err)
	}

	if !hasUnverified {
		return errors.New("you don't have unverified role")
	}

	err = s.repo.SetVerified(ctx, guildID, userID)
	if err != nil {
		return fmt.Errorf("failed to set user as verified: %w", err)
	}

	return nil
}

func (s *service) hasRole(guildID, userID, roleID string) (bool, error) {
	member, err := s.roleConfig.Session.GuildMember(guildID, userID)
	if err != nil {
		return false, err
	}

	for _, r := range member.Roles {
		if r == roleID {
			return true, nil
		}
	}

	return false, nil
}

func (s *service) UnverifyUser(ctx context.Context, guildID, userID string) error {
	verified, err := s.repo.IsVerified(ctx, guildID, userID)
	if err != nil {
		return fmt.Errorf("failed to check verification status: %w", err)
	}
	if !verified {
		return errors.New("user is not verified")
	}

	err = s.repo.SetUnverified(ctx, guildID, userID)
	if err != nil {
		return fmt.Errorf("failed to set user as unverified: %w", err)
	}

	return nil
}

func (s *service) CheckVerification(ctx context.Context, guildID, userID string) (bool, error) {
	return s.repo.IsVerified(ctx, guildID, userID)
}

func (s *service) GetOrSetupRoles(guildID string) (verifiedRoleID, unverifiedRoleID string, err error) {
	verifiedRoleID, unverifiedRoleID, err = s.roleConfig.SetupRolesForGuild(guildID)
	if err != nil {
		return "", "", fmt.Errorf("failed to setup roles: %w", err)
	}

	return verifiedRoleID, unverifiedRoleID, nil
}

func (s *service) HandleUserLeave(ctx context.Context, guildID, userID string) error {
	err := s.repo.RemoveUserFromGuild(ctx, guildID, userID)
	if err != nil {
		return fmt.Errorf("failed to handle user leave: %w", err)
	}

	return nil
}

func (s *service) RecordJoinTime(ctx context.Context, guildID, userID string) error {
	return s.repo.SetJoinTime(ctx, guildID, userID, time.Now())
}

func (s *service) RemoveJoinTime(ctx context.Context, guildID, userID string) error {
	return s.repo.RemoveJoinTime(ctx, guildID, userID)
}

func (s *service) StartCleanupJob(ctx context.Context, interval, maxAge time.Duration) {
	go func() {
		log.Printf("🧹 Cleanup job started (interval=%s, maxAge=%s)", interval, maxAge)
		s.cleanupUnverified(ctx, maxAge)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.cleanupUnverified(ctx, maxAge)
			case <-ctx.Done():
				log.Println("🧹 Cleanup job stopped")
				return
			}
		}
	}()
}

func (s *service) cleanupUnverified(ctx context.Context, maxAge time.Duration) {
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()

	keys, err := s.repo.GetAllUnverifiedKeys(ctx)
	if err != nil {
		log.Printf("🧹 Cleanup: failed to scan keys: %v", err)
		return
	}

	now := time.Now()
	kicked := 0

	for _, key := range keys {
		parts := strings.SplitN(key, ":", 4)
		if len(parts) < 4 {
			continue
		}
		guildID := parts[1]
		userID := parts[3]

		joinTime, err := s.repo.GetJoinTime(ctx, guildID, userID)
		if err != nil || joinTime == nil {
			continue
		}

		if now.Sub(*joinTime) <= maxAge {
			continue
		}

		member, err := s.roleConfig.Session.GuildMember(guildID, userID)
		if err != nil {
			s.repo.RemoveUserFromGuild(ctx, guildID, userID)
			s.repo.RemoveJoinTime(ctx, guildID, userID)
			continue
		}

		_, unverifiedRoleID, _ := s.GetOrSetupRoles(guildID)

		hasUnverified := false
		for _, roleID := range member.Roles {
			if roleID == unverifiedRoleID {
				hasUnverified = true
				break
			}
		}

		if hasUnverified {
			err := s.roleConfig.Session.GuildMemberDeleteWithReason(guildID, userID, "Auto-kicked: unverified for too long")
			if err != nil {
				log.Printf("🧹 Cleanup: failed to kick %s in guild %s: %v", userID, guildID, err)
			} else {
				log.Printf("🧹 Cleanup: kicked unverified user %s from guild %s", userID, guildID)
				kicked++
			}
		}

		s.repo.RemoveUserFromGuild(ctx, guildID, userID)
		s.repo.RemoveJoinTime(ctx, guildID, userID)
	}

	if len(keys) > 0 {
		log.Printf("🧹 Cleanup: scanned %d, kicked %d unverified users", len(keys), kicked)
	}
}
