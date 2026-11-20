package commands

import (
	"fmt"
	"time"

	"orion-event-bot/bot/core"
	"orion-event-bot/bot/handlers"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func init() {
	handlers.RegisterCommand(handlers.Command{
		Name:        "hello",
		Description: "hello",
		Create: discord.SlashCommandCreate{
			Name:        "ping",
			Description: "Replies with pong",
		},
		Execute: func(b *core.Bot, event *events.ApplicationCommandInteractionCreate) {
			loc := core.GetLocalizer(event.Locale().String())
			wsPing := b.Client.Gateway.Latency().Milliseconds()
			dbPing := 0

			botPing := time.Since(event.ID().Time()).Milliseconds()
			avatarURL := event.User().EffectiveAvatarURL()

			pingStats, _ := loc.Localize(&i18n.LocalizeConfig{
				MessageID: "PingStats",
				TemplateData: map[string]interface{}{
					"BotPing": botPing,
					"WSPing":  wsPing,
					"DBPing":  dbPing,
				},
			})

			msg := discord.NewMessageCreateV2(
				discord.NewContainer(
					discord.NewSection(
						discord.NewTextDisplay("# Orion Hosting"),
						discord.NewTextDisplay(pingStats),
					).WithAccessory(discord.NewThumbnail(avatarURL)),
				).WithAccentColor(0x5865F2),
			)

			err := event.CreateMessage(msg)
			if err != nil {
				fmt.Println("Error sending ping message:", err)
			}
		},
	})
}
