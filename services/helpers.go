package services

import (
	"fmt"
	"github.com/than-os/sentinel-bot/buttons"
	"github.com/than-os/sentinel-bot/constants"
	"github.com/than-os/sentinel-bot/dbo/ldb"
	"github.com/than-os/sentinel-bot/dbo/models"
	"github.com/than-os/sentinel-bot/templates"
	"gopkg.in/telegram-bot-api.v4"
	"time"
)

func Send(b *tgbotapi.BotAPI, u tgbotapi.Update, msg string, opts ...models.ButtonHelper) {
	c := tgbotapi.NewMessage(u.Message.Chat.ID, msg)

	for _, o := range opts {
		if o.Type == "replyButton" {
			c.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
				Keyboard:        buttons.ReplyButtons(o.Labels),
				OneTimeKeyboard: true,
				ResizeKeyboard:  true,
			}
		}
		if o.Type == "inlineButton" {
			c.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
				InlineKeyboard: buttons.InlineButtons(o.InlineKeyboardOpts),
			}
		}
	}

	_, _ = b.Send(c)
}

func SubscriptionPeriod(b *tgbotapi.BotAPI, u tgbotapi.Update, db ldb.BotDB, t time.Duration, network, price, period string) {
	EthPairs := []models.KV{
		{
			Key: constants.Timestamp, Value: time.Now().Add(t).Format(time.RFC3339),
		},
		{
			Key: constants.NodePrice, Value: price,
		},
	}
	TMPairs := []models.KV{
		{
			Key: constants.TimestampTM, Value: time.Now().Add(t).Format(time.RFC3339),
		},
		{
			Key: constants.NodePriceTM, Value: price,
		},
	}
	if network == constants.EthNetwork {
		err := db.MultiWriter(EthPairs, u.Message.From.UserName)
		if err != nil {
			Send(b, u, templates.BWError)
		}
		msg := fmt.Sprintf(templates.BWPeriods, period)
		Send(b, u, msg)
		return
	}
	err := db.MultiWriter(TMPairs, u.Message.From.UserName)
	if err != nil {
		Send(b, u, templates.BWError)
	}
	msg := fmt.Sprintf(templates.BWPeriods, period)
	Send(b, u, msg)

}
