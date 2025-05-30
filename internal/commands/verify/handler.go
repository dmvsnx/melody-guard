package verify

import (
	"context"
	"log"

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

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Failed to respond to interaction: %v", err)
		return
	}

	ctx := context.Background()
	guildID := i.GuildID
	userID := i.Member.User.ID

	go func() {
		sendResponse := func(content string) {
			_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: content,
				Flags: discordgo.MessageFlagsEphemeral,
			})
			if err != nil {
				log.Printf("Failed to send followup message: %v", err)
			}
		}

		verifiedRoleID, unverifiedRoleID, err := h.service.GetOrSetupRoles(guildID)
		if err != nil {
			sendResponse("❌ Failed to setup roles: " + err.Error())
			return
		}

		if err := h.service.VerifyUser(ctx, guildID, userID); err != nil {
			sendResponse("❌ Verification failed: " + err.Error())
			return
		}

		if unverifiedRoleID != "" {
			if err := s.GuildMemberRoleRemove(guildID, userID, unverifiedRoleID); err != nil {
				log.Printf("Warning: Failed to remove unverified role: %v", err)
			}
		}

		if err := s.GuildMemberRoleAdd(guildID, userID, verifiedRoleID); err != nil {
			sendResponse("❌ Failed to add verified role: " + err.Error())
			return
		}

		log.Println("✅ Verified Successfully - Guild:", guildID, "User:", userID)
		sendResponse("✅ Verification successful!!!")
	}()
}

// func (h *Handler) HandlerUnverify(s *discordgo.Session, i *discordgo.InteractionCreate) {
// 	if i.ApplicationCommandData().Name != "unverify" {
// 		return
// 	}

// 	ctx := context.Background()
// 	guildID := i.GuildID
// 	userID := i.Member.User.ID

// 	verifiedRoleID, unverifiedRoleID, err := h.service.GetOrSetupRoles(guildID)
// 	if err != nil {
// 		h.respondError(s, i, "Failed to get verification roles: "+err.Error())
// 		return
// 	}

// 	err = h.service.UnverifyUser(ctx, guildID, userID)
// 	if err != nil {
// 		h.respondSuccess(s, i, "Unverification failed: "+err.Error())
// 		return
// 	}

// 	if verifiedRoleID != "" {
// 		err = s.GuildMemberRoleRemove(guildID, userID, verifiedRoleID)
// 		if err != nil {
// 			log.Printf("Warning: Failed to remove verified role: %v", err)
// 		}
// 	}

// 	if unverifiedRoleID != "" {
// 		err = s.GuildMemberRoleAdd(guildID, userID, unverifiedRoleID)
// 		if err != nil {
// 			log.Printf("Warning: Failed to add unverified role: %v", err)
// 		}
// 	}

// 	h.respondSuccess(s, i, "🔒 You have been unverified. Please verify again to access server features.")
// }
