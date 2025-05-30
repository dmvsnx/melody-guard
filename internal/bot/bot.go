package bot

import (
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
	commands []*discordgo.ApplicationCommand
}

func NewBot(token string, verifyServices services.Service) (*Bot, error) {
	session, err := discordgo.New("Bot "+ token)
	if err != nil {
		return nil, err
	}

	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildMessages

	commands := []*discordgo.ApplicationCommand{
		{
			Name: "help",
			Description: "Show this help message",
		},
		{
			Name: "verify",
			Description: "Verify yourself",
		},
		{
			Name: "play",
			Description: "Play a song",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type: discordgo.ApplicationCommandOptionString,
					Name: "query",
					Description: "Song name or url",
					Required: true,
				},
			},
		},
	}

	return &Bot{
		session: session,
		verifyHandler: verify.NewHandler(verifyServices),
		helphandler: help.NewHandlerHelp(),
		verifyService: verifyServices,
		commands: commands,

	}, nil
}

func (b *Bot) Start() error {
	b.session.AddHandler(b.readyHandler)
	b.session.AddHandler(b.handlerMemberJoin)
	b.session.AddHandler(b.handlerGuildCreate)
	b.session.AddHandler(b.verifyHandler.HandlerVerify)
	b.session.AddHandler(b.helphandler.HandlerHelp)
	b.session.AddHandler(b.handlerGuildMemberRemove)
	// b.session.AddHandler(b.verifyHandler.HandlerUnverify)

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
}

func (b *Bot) handlerGuildMemberRemove(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	log.Printf("👋 User left: %s from guild %s", m.User.Username, m.GuildID)
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
		{
			Name: "play",
			Description: "Play a song",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type: discordgo.ApplicationCommandOptionString,
					Name: "query",
					Description: "Song name or url",
					Required: true,
				},
			},
		},
	}

	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", cmd)
		if err != nil {
			log.Printf("Cannot create '%s' command: %v", cmd.Name, err)
		}
	}
}
