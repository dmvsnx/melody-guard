package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/savanyv/melody-guard/internal/config"
	"github.com/savanyv/melody-guard/internal/store"
)

// Service handles the business logic for user verification
type Service interface {
	VerifyUser(ctx context.Context, guildID, userID string) error
	hasRole(guildID, userID, roleID string) (bool, error)
	UnverifyUser(ctx context.Context, guildID, userID string) error
	CheckVerification(ctx context.Context, guildID, userID string) (bool, error)
	GetOrSetupRoles(guildID string) (verifiedRoleID, unverifiedRoleID string, err error)
	HandleUserLeave(ctx context.Context, guildID, userID string) error
}
type service struct {
	repo       store.VerifyRepository
	roleConfig *config.RolesConfig
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

	// Mark user as unverified in storage
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
	// Remove the user's verification status for this guild
	err := s.repo.RemoveUserFromGuild(ctx, guildID, userID)
	if err != nil {
		return fmt.Errorf("failed to handle user leave: %w", err)
	}

	return nil
}
