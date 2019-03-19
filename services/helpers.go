package services

import (
	"github.com/than-os/sentinel-bot/buttons"
	"github.com/than-os/sentinel-bot/dbo/models"
	"gopkg.in/telegram-bot-api.v4"
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
