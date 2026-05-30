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

	verifiedRoleID, _, err := h.service.GetOrSetupRoles(i.GuildID)
	if err != nil {
		if _, ferr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: strPtr("❌ " + err.Error()),
		}); ferr != nil {
			log.Printf("Failed to respond: %v", ferr)
		}
		return
	}

	for _, roleID := range i.Member.Roles {
		if roleID == verifiedRoleID {
			if _, ferr := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: strPtr("✅ You are already verified!"),
			}); ferr != nil {
				log.Printf("Failed to respond: %v", ferr)
			}
			return
		}
	}

	msg := i.Interaction.Message
	channelID := i.ChannelID

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "verify_modal",
			Title:    "📜 Accept Server Rules",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "rules_accept",
							Label:       "Type ACCEPT to verify you agree to the rules",
							Style:       discordgo.TextInputShort,
							Placeholder: "ACCEPT",
							MinLength:   6,
							MaxLength:   6,
							Required:    true,
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Failed to show modal: %v", err)
		return
	}

	if msg != nil {
		if err := s.ChannelMessageDelete(channelID, msg.ID); err != nil {
			log.Printf("Warning: failed to delete verify message: %v", err)
		}
	}
}

func strPtr(s string) *string {
	return &s
}

func (h *Handler) HandlerVerifyModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionModalSubmit || i.ModalSubmitData().CustomID != "verify_modal" {
		return
	}

	data := i.ModalSubmitData()
	acceptValue := ""
	for _, row := range data.Components {
		actionsRow, ok := row.(*discordgo.ActionsRow)
		if !ok {
			continue
		}
		for _, comp := range actionsRow.Components {
			input, ok := comp.(*discordgo.TextInput)
			if !ok || input.CustomID != "rules_accept" {
				continue
			}
			acceptValue = input.Value
		}
	}

	if acceptValue != "ACCEPT" {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ You must type **ACCEPT** to verify. Please try again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Printf("Failed to respond to modal: %v", err)
		}
		return
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Failed to defer modal response: %v", err)
		return
	}

	ctx := context.Background()
	guildID := i.GuildID
	userID := i.Member.User.ID

	verifiedRoleID, unverifiedRoleID, err := h.service.GetOrSetupRoles(guildID)
	if err != nil {
		if _, ferr := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ " + err.Error(),
			Flags:   discordgo.MessageFlagsEphemeral,
		}); ferr != nil {
			log.Printf("Failed to send followup: %v", ferr)
		}
		return
	}

	for _, roleID := range i.Member.Roles {
		if roleID == verifiedRoleID {
			if _, ferr := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
				Content: "✅ You are already verified!",
				Flags:   discordgo.MessageFlagsEphemeral,
			}); ferr != nil {
				log.Printf("Failed to send followup: %v", ferr)
			}
			return
		}
	}

	if err := h.service.VerifyUser(ctx, guildID, userID); err != nil {
		if _, ferr := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ " + err.Error(),
			Flags:   discordgo.MessageFlagsEphemeral,
		}); ferr != nil {
			log.Printf("Failed to send followup: %v", ferr)
		}
		return
	}

	if unverifiedRoleID != "" {
		if err := s.GuildMemberRoleRemove(guildID, userID, unverifiedRoleID); err != nil {
			log.Printf("Warning: failed to remove unverified role: %v", err)
		}
	}

	if err := s.GuildMemberRoleAdd(guildID, userID, verifiedRoleID); err != nil {
		if _, ferr := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: "❌ Failed to add verified role: " + err.Error(),
			Flags:   discordgo.MessageFlagsEphemeral,
		}); ferr != nil {
			log.Printf("Failed to send followup: %v", ferr)
		}
		return
	}

	log.Printf("✅ Verified via modal - Guild: %s, User: %s", guildID, userID)

	h.service.RemoveJoinTime(ctx, guildID, userID)

	if _, ferr := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: "✅ You are now verified! Welcome to the server!",
		Flags:   discordgo.MessageFlagsEphemeral,
	}); ferr != nil {
		log.Printf("Failed to send followup: %v", ferr)
	}
}
