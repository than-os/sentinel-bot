package main

import (
	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/than-os/sentinel-bot/constants"
	"github.com/than-os/sentinel-bot/dbo"
	"github.com/than-os/sentinel-bot/handlers"
	"github.com/than-os/sentinel-bot/helpers"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"os"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("error while reading ENV config. shutting down bot now.")
	}
}

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_API_KEY"))
	if err != nil {
		log.Fatalf("error in instantiating the bot: %v", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		color.Red("error while receiving messages: %s", err)
	}
	color.Green("started %s successfully", bot.Self.UserName)

	db, nodes, err := dbo.NewDB()
	if err != nil {
		log.Fatal(err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.From.IsBot {
			return
		}

		// handle the commands for the bot
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "mynode":
				handlers.ShowMyNode(bot, update, db)
			case "start":
				handlers.Greet(bot, update, db)
			case "restart":
				handlers.Restart(bot, update, db)
			case "info":
				handlers.ShowMyInfo(bot, update, db)
			case "eth":
				handlers.ShowEthWallet(bot, update, db)
			case "about":
				handlers.AboutSentinel(bot, update)
			default:
				return
			}
		}
		// handle the app flow for bot
		handlers.MainHandler(bot, update, db, *nodes)
		TMState := helpers.GetState(bot, update, constants.TMState, db)
		color.Green("******* APP STATE = %d *******", TMState)

	}

}
