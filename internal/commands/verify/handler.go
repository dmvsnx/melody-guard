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
	if i.Type != discordgo.InteractionApplicationCommand || i.ApplicationCommandData().Name != "verify" {
		return
	}

	verifiedRoleID, _, err := h.service.GetOrSetupRoles(i.GuildID)
	if err != nil {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ " + err.Error(),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	hasVerified := false
	for _, roleID := range i.Member.Roles {
		if roleID == verifiedRoleID {
			hasVerified = true
			break
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔐 Verification",
		Color:       0x5865F2,
		Description: "Click the button below to get verified and access all channels.",
	}
	if hasVerified {
		embed.Description = "✅ You are already verified!"
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Verify",
							Style:    discordgo.SuccessButton,
							CustomID: "verify_button",
							Disabled: hasVerified,
							Emoji: &discordgo.ComponentEmoji{
								Name: "✅",
							},
						},
					},
				},
			},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Failed to respond to verify command: %v", err)
	}
}

func (h *Handler) HandlerVerifyButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent || i.MessageComponentData().CustomID != "verify_button" {
		return
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
	if err != nil {
		log.Printf("Failed to defer button interaction: %v", err)
		return
	}

	ctx := context.Background()
	guildID := i.GuildID
	userID := i.Member.User.ID

	verifiedRoleID, unverifiedRoleID, err := h.service.GetOrSetupRoles(guildID)
	if err != nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ " + err.Error(),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	for _, roleID := range i.Member.Roles {
		if roleID == verifiedRoleID {
			h.service.VerifyUser(ctx, guildID, userID)
			_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "✅ You are already verified!",
				Flags:   discordgo.MessageFlagsEphemeral,
			})
			return
		}
	}

	if err := h.service.VerifyUser(ctx, guildID, userID); err != nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ " + err.Error(),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	if unverifiedRoleID != "" {
		if err := s.GuildMemberRoleRemove(guildID, userID, unverifiedRoleID); err != nil {
			log.Printf("Warning: failed to remove unverified role: %v", err)
		}
	}

	if err := s.GuildMemberRoleAdd(guildID, userID, verifiedRoleID); err != nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ Failed to add verified role: " + err.Error(),
			Flags:   discordgo.MessageFlagsEphemeral,
		})
		return
	}

	log.Printf("✅ Verified via button - Guild: %s, User: %s", guildID, userID)

	h.service.RemoveJoinTime(ctx, guildID, userID)

	if i.Interaction.Message != nil {
		if i.Interaction.Message.Flags&discordgo.MessageFlagsEphemeral != 0 {
			err := s.InteractionResponseDelete(i.Interaction)
			if err != nil {
				log.Printf("Warning: failed to delete ephemeral verify message: %v", err)
			}
		} else {
			err := s.ChannelMessageDelete(i.ChannelID, i.Interaction.Message.ID)
			if err != nil {
				log.Printf("Warning: failed to delete verify message: %v", err)
			}
		}
	}

	_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "✅ You are now verified! Welcome to the server!",
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}
