package commands

import (
	"fmt"
	"math/rand"

	"orion-event-bot/bot/core"
	"orion-event-bot/bot/handlers"
	"orion-event-bot/bot/utils"
	"orion-event-bot/bot/vars"
	"orion-event-bot/db"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func init() {
	handlers.RegisterCommand(handlers.Command{
		Name:        "ride",
		Description: "Choose a ski slope to descend",
		Create: discord.SlashCommandCreate{
			Name:        "ride",
			Description: "Choose a ski slope to descend",
			DescriptionLocalizations: map[discord.Locale]string{
				discord.LocaleFrench: "Choisissez une piste de ski à descendre (Verte, Rouge ou Noire)",
			},
		},
		Execute: func(b *core.Bot, event *events.ApplicationCommandInteractionCreate) {
			var user db.User
			if err := b.DB.GormDB.Where("id = ?", event.User().ID.String()).First(&user).Error; err != nil {
				fmt.Println("DB error:", err)
				return
			}

			if user.Energy <= 0 {
				loc := core.GetLocalizer(event.Locale().String())
				if err := event.CreateMessage(discord.NewMessageCreateV2(
					discord.NewContainer(
						discord.NewTextDisplay(localizeText(loc, "RideNoEnergy", nil, "You have no energy left.")),
					),
				).WithFlags(discord.MessageFlagEphemeral)); err != nil {
					return
				}
				return
			}

			loc := core.GetLocalizer(event.Locale().Code())

			title := localizeText(loc, "RideTitle", nil, "# ⛷️ Ride")
			prompt := localizeText(loc, "RidePrompt", nil, "Choose which slope to descend: Green, Red, or Black:")
			btnGreen := localizeText(loc, "RideButtonGreen", nil, "Green Slope")
			btnRed := localizeText(loc, "RideButtonRed", nil, "Red Slope")
			btnBlack := localizeText(loc, "RideButtonBlack", nil, "Black Slope")

			msg := discord.NewMessageCreateV2(
				discord.NewContainer(
					discord.NewSection(
						discord.NewTextDisplay(title),
						discord.NewTextDisplay(prompt),
					).WithAccessory(discord.NewThumbnail(event.User().EffectiveAvatarURL())),
					discord.NewActionRow(
						discord.NewSuccessButton(btnGreen, "ride-buttonGreen-"+event.User().ID.String()).WithDisabled(!(user.Energy >= 1)),
						discord.NewDangerButton(btnRed, "ride-buttonRed-"+event.User().ID.String()).WithDisabled(!(user.Energy >= 1)),
						discord.NewSecondaryButton(btnBlack, "ride-buttonBlack-"+event.User().ID.String()).WithDisabled(!(user.Energy >= 2)),
					),
				),
			)

			err := event.CreateMessage(msg)
			if err != nil {
				fmt.Println("Error sending ride message:", err)
			}
		},
		ExecuteButton: func(b *core.Bot, event *events.ComponentInteractionCreate) {
			customId := event.Data.CustomID()
			var user db.User
			if err := b.DB.GormDB.Where("id = ?", event.User().ID.String()).First(&user).Error; err != nil {
				fmt.Println("DB error:", err)
				return
			}

			var e db.Event
			if err := b.DB.GormDB.Where("id = ?", 1).First(&e).Error; err != nil {
				fmt.Println("DB error:", err)
				return
			}

			currentTile := utils.GetTile(user.Position, vars.Plate)

			var tileToMove int = 0
			var isImmune bool = false
			var isAvalanche bool = false
			switch customId {
			case "ride-buttonGreen-" + event.User().ID.String():
				user.Energy -= 1
				tileToMove = rand.Intn(2)
				isImmune = true
			case "ride-buttonRed-" + event.User().ID.String():
				user.Energy -= 1
				tileToMove = rand.Intn(4)
			case "ride-buttonBlack-" + event.User().ID.String():
				user.Energy -= 2
				if rand.Intn(10) <= 5 {
					tileToMove = rand.Intn(3) + 3
				} else {
					tileToMove = -3
					isAvalanche = true
				}
			default:
				loc := core.GetLocalizer(event.Locale().String())
				if err := event.CreateMessage(discord.NewMessageCreateV2(
					discord.NewContainer(
						discord.NewTextDisplay(localizeText(loc, "NotYourComponent", nil, "You cannot interact with someone else's menu.")),
					),
				).WithFlags(discord.MessageFlagEphemeral)); err != nil {
					return
				}
				return
			}

			if user.Energy < 0 {
				if err := event.CreateMessage(discord.NewMessageCreateV2(
					discord.NewContainer(
						discord.NewTextDisplay(localizeText(core.GetLocalizer(event.Locale().String()), "RideNoEnergy", nil, "You have no energy left.")),
					),
				).WithFlags(discord.MessageFlagEphemeral)); err != nil {
					return
				}
				return
			}

			if !isImmune {
				switch currentTile.SpecialCase {
				case vars.SpecialCaseSlush:
				case vars.SpecialCaseChairlift:
					tileToMove = int(tileToMove / 2)
				}
			}

			if isAvalanche && user.BuffHelmet {
				tileToMove = 0
				isAvalanche = false
				user.BuffHelmet = false
			}

			if user.BuffWax {
				if tileToMove > 0 {
					tileToMove *= 2
				}
				user.BuffWax = false
			}

			newTilePos := user.Position + tileToMove
			if newTilePos < 0 {
				newTilePos = 0
			}
			user.Position = newTilePos

			newTile := utils.GetTile(newTilePos, vars.Plate)
			var gamepos db.GamePosition
			if err := b.DB.GormDB.Where("posid = ?", newTile.Number).FirstOrCreate(&gamepos).Error; err != nil {
				fmt.Println("DB error:", err)
				return
			}

			user, gamepos, e = utils.UpdateUserWithSpecialTile(b.DB.GormDB, user, newTile.SpecialCase, gamepos, e)
			if user.Position < 0 {
				user.Position = 0
			}
			if user.Snowflakes < 0 {
				user.Snowflakes = 0
			}

			if err := b.DB.GormDB.Save(&user).Error; err != nil {
				fmt.Println("DB error:", err)
				return
			}
			if err := b.DB.GormDB.Save(&gamepos).Error; err != nil {
				fmt.Println("DB error:", err)
				return
			}
			if err := b.DB.GormDB.Save(&e).Error; err != nil {
				fmt.Println("DB error:", err)
				return
			}

			var msg discord.MessageUpdate
			if isAvalanche {
				loc := core.GetLocalizer(event.Locale().Code())
				title := localizeText(loc, "RideTitle", nil, "# ⛷️ Ride")
				body := localizeText(loc, "RideAvalanche", map[string]interface{}{
					"newPos": user.Position,
				}, "")
				msg = discord.NewMessageUpdateV2(
					discord.NewContainer(
						discord.NewSection(
							discord.NewTextDisplay(title),
							discord.NewTextDisplay(body),
						).WithAccessory(discord.NewThumbnail(event.User().EffectiveAvatarURL())),
					),
				)
			} else {
				msg = utils.GetMessageByTileType(newTile, event.User(), user, event.Locale().Code())
			}

			if err := event.UpdateMessage(msg); err != nil {
				fmt.Println(err)
			}
		},
	})
}
