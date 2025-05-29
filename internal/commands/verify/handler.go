package verify

import (
	"context"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/savanyv/melody-guard/internal/services"
)

type Handler struct {
	service services.Service
}

func NewHandler(service services.Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) HandlerVerify(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name != "verify" {
		return
	}

	ctx := context.Background()
	guildID := i.GuildID
	userID := i.Member.User.ID

	verifiedRoleID, unverifiedRoleID, err := h.service.GetOrSetupRoles(guildID)
	if err != nil {
		h.respondError(s, i, "Failed to setup verification roles: "+err.Error())
		return
	}

	err = h.service.VerifyUser(ctx, guildID, userID)
	if err != nil {
		errorMsg := "Verification failed"
		if strings.Contains(err.Error(), "don't have unverified role") {
			errorMsg = "❌ You're already verified or don't need verification"
		}
		h.respondError(s, i, errorMsg)
		return
	}

	if unverifiedRoleID != "" {
		err = s.GuildMemberRoleRemove(guildID, userID, unverifiedRoleID)
		if err != nil {
			log.Printf("Warning: Failed to remove unverified role: %v", err)
		}
	}

	err = s.GuildMemberRoleAdd(guildID, userID, verifiedRoleID)
	if err != nil {
		h.respondError(s, i, "Verification succeeded but failed to assign role")
		return
	}

	h.respondSuccess(s, i, "✅ You have been verified!")
}

func (h *Handler) HandlerUnverify(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name != "unverify" {
		return
	}

	ctx := context.Background()
	guildID := i.GuildID
	userID := i.Member.User.ID

	verifiedRoleID, unverifiedRoleID, err := h.service.GetOrSetupRoles(guildID)
	if err != nil {
		h.respondError(s, i, "Failed to get verification roles: "+err.Error())
		return
	}

	err = h.service.UnverifyUser(ctx, guildID, userID)
	if err != nil {
		h.respondSuccess(s, i, "Unverification failed: "+err.Error())
		return
	}

	if verifiedRoleID != "" {
		err = s.GuildMemberRoleRemove(guildID, userID, verifiedRoleID)
		if err != nil {
			log.Printf("Warning: Failed to remove verified role: %v", err)
		}
	}

	if unverifiedRoleID != "" {
		err = s.GuildMemberRoleAdd(guildID, userID, unverifiedRoleID)
		if err != nil {
			log.Printf("Warning: Failed to add unverified role: %v", err)
		}
	}

	h.respondSuccess(s, i, "🔒 You have been unverified. Please verify again to access server features.")
}

func (h *Handler) respondSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (h *Handler) respondError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "❌ " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
