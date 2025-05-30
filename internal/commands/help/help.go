package help

import "github.com/bwmarrin/discordgo"

type HandlerHelp struct {}

func NewHandlerHelp() *HandlerHelp {
	return &HandlerHelp{}
}

func (h *HandlerHelp) HandlerHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.ApplicationCommandData().Name != "help" {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "📝 Bot Commands Help",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🔹 /help",
				Value:  "Show this help message",
				Inline: false,
			},
			{
				Name:   "🔹 /verify",
				Value:  "Get verified to access all channels",
				Inline: false,
			},
			{
				Name:   "🔹 /play [query]",
				Value:  "Play music from YouTube\nExample: `/play never gonna give you up`",
				Inline: false,
			},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
