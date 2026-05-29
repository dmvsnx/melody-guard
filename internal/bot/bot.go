package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/savanyv/melody-guard/internal/commands/help"
	"github.com/savanyv/melody-guard/internal/commands/verify"
	"github.com/savanyv/melody-guard/internal/services"
)

type Bot struct {
	session *discordgo.Session
	verifyHandler *verify.Handler
	verifyService services.Service
	helphandler *help.HandlerHelp
}

func NewBot(token string, verifyServices services.Service) (*Bot, error) {
	session, err := discordgo.New("Bot "+ token)
	if err != nil {
		return nil, err
	}

	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildMessages

	return &Bot{
		session: session,
		verifyHandler: verify.NewHandler(verifyServices),
		helphandler: help.NewHandlerHelp(),
		verifyService: verifyServices,
	}, nil
}

func (b *Bot) Start() error {
	b.session.AddHandler(b.readyHandler)
	b.session.AddHandler(b.handlerMemberJoin)
	b.session.AddHandler(b.handlerGuildCreate)
	b.session.AddHandler(b.verifyHandler.HandlerVerify)
	b.session.AddHandler(b.verifyHandler.HandlerVerifyButton)
	b.session.AddHandler(b.helphandler.HandlerHelp)
	b.session.AddHandler(b.handlerGuildMemberRemove)

	err := b.session.Open()
	if err != nil {
		return err
	}

	b.registerCommands()

	return nil
}

func (b *Bot) Stop() error {
	return b.session.Close()
}

func (b *Bot) GetSession() *discordgo.Session {
	return b.session
}

func (b *Bot) readyHandler(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("🤖 Logged in as %s", r.User.Username)
	s.UpdateGameStatus(0, "🎮 Type /help for commands")
}

func (b *Bot) handlerGuildCreate(s *discordgo.Session, g *discordgo.GuildCreate) {
	log.Printf("📥 Joined new guild: %s (%s)", g.Name, g.ID)

	verifiedID, unverifiedID, err := b.verifyService.GetOrSetupRoles(g.ID)
	if err != nil {
		log.Printf("Failed to setup roles: %v", err)
		return
	}

	log.Printf("✅ Created roles - Verified: %s, Unverified: %s", verifiedID, unverifiedID)
}

func (b *Bot) handlerMemberJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	guildID := m.GuildID
	userID := m.User.ID

	_, unverifiedRoleID, err := b.verifyService.GetOrSetupRoles(guildID)
	if err != nil {
		log.Printf("Failed to setup roles: %v", err)
		return
	}

	err = s.GuildMemberRoleAdd(guildID, userID, unverifiedRoleID)
	if err != nil {
		log.Printf("Failed to add unverified role: %v", err)
		return
	}

	log.Printf("🔐 Assigned unverified role to new member %s", m.User.Username)

	if err := b.verifyService.UnverifyUser(context.Background(), guildID, userID); err != nil {
		log.Printf("Warning: failed to reset verification state: %v", err)
	}

	if err := b.verifyService.RecordJoinTime(context.Background(), guildID, userID); err != nil {
		log.Printf("Warning: failed to record join time: %v", err)
	}

	guild, err := s.Guild(guildID)
	if err != nil {
		log.Printf("Failed to fetch guild info: %v", err)
		return
	}

	if guild.SystemChannelID == "" {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "👋 Welcome!",
		Description: fmt.Sprintf("Welcome %s! Click the button below to get verified and access all channels.", m.Member.Mention()),
		Color:       0x5865F2,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: m.User.AvatarURL(""),
		},
	}

	_, err = s.ChannelMessageSendComplex(guild.SystemChannelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Verify",
						Style:    discordgo.SuccessButton,
						CustomID: "verify_button",
						Emoji: &discordgo.ComponentEmoji{
							Name: "✅",
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Failed to send welcome message: %v", err)
	}
}

func (b *Bot) handlerGuildMemberRemove(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	log.Printf("👋 User left: %s from guild %s", m.User.Username, m.GuildID)

	if err := b.verifyService.HandleUserLeave(context.Background(), m.GuildID, m.User.ID); err != nil {
		log.Printf("Warning: failed to cleanup data for leaving user: %v", err)
	}
	if err := b.verifyService.RemoveJoinTime(context.Background(), m.GuildID, m.User.ID); err != nil {
		log.Printf("Warning: failed to remove join time for leaving user: %v", err)
	}
}

func (b *Bot) registerCommands() {
	commands := []*discordgo.ApplicationCommand{
		{
			Name: "help",
			Description: "Show all available commands",
		},
		{
			Name: "verify",
			Description: "Get verified to access all channels",
		},
	}

	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", cmd)
		if err != nil {
			log.Printf("Cannot create '%s' command: %v", cmd.Name, err)
		}
	}
}
