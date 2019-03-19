package main

import (
	"github.com/fatih/color"
	"github.com/joho/godotenv"
	"github.com/than-os/sentinel-bot/dbo"
	"github.com/than-os/sentinel-bot/handlers"
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
		return
	}
	color.Green("%s", "started the bot successfully")

	db, nodes, err := dbo.NewDB()
	if err != nil {
		log.Fatal(err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}

		handlers.MainHandler(bot, update, db, *nodes)

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "mynode":
				handlers.ShowMyNode(bot, update, db)
			case "start":
				handlers.Greet(bot, update)
			case "restart":
				handlers.Restart(bot, update, db)
			case "info":
				handlers.ShowMyInfo(bot, update, db)
			case "eth":
				handlers.ShowEthWallet(bot, update, db)
			case "beta":
				//tendermint.FindTmTxnByHash("")
			default:
				return
			}
		}

	}

}
