package config

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type RolesConfig struct {
	Session *discordgo.Session
	verifiedName string
	unverifiedName string
}

func NewRoleConfig(s *discordgo.Session, verifiedName, unverifiedName string) *RolesConfig {
	return &RolesConfig{
		Session: s,
		verifiedName: verifiedName,
		unverifiedName: unverifiedName,
	}
}

func (rc *RolesConfig) SetupRolesForGuild(guildID string) (verifiedRoleID, unverifiedRoleID string, err error) {
	if rc.Session == nil {
		return "", "", fmt.Errorf("session is RolesConfig is nil")
	}

	verifiedRoleID, err = rc.getOrCreateRole(guildID, rc.verifiedName, 0x00FF00)
	if err != nil {
		return "", "", fmt.Errorf("failed to setup verified role: %w", err)
	}

	unverifiedRoleID, err = rc.getOrCreateRole(guildID, rc.unverifiedName, 0xFF0000)
	if err != nil {
		return "", "", fmt.Errorf("failed to setup unverified role: %w", err)
	}
	
	return verifiedRoleID, unverifiedRoleID, nil
}

func (rc *RolesConfig) getOrCreateRole(guildID, name string, color int) (string, error) {
	roles, err := rc.Session.GuildRoles(guildID)
	if err != nil {
		return "", err
	}

	for _, role := range roles {
		if role.Name == name {
			return role.ID, nil
		}
	}

	perm := int64(discordgo.PermissionViewChannel)
	mentionable := false

	newRole, err := rc.Session.GuildRoleCreate(guildID, &discordgo.RoleParams{
		Name: name,
		Color: &color,
		Permissions: &perm,
		Mentionable: &mentionable,
	})
	if err != nil {
		return "", err
	}

	return newRole.ID, nil
}
