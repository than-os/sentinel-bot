package buttons

import (
	"github.com/than-os/sentinel-bot/dbo/models"
	"gopkg.in/telegram-bot-api.v4"
)

func ReplyButtons(opts []string) [][]tgbotapi.KeyboardButton {
	var btns [][]tgbotapi.KeyboardButton

	for _, label := range opts {
		btns = append(btns, []tgbotapi.KeyboardButton{
			{
				Text: label,
			},
		})
	}

	return btns
}

func InlineButtons(opts []models.InlineButtonOptions) [][]tgbotapi.InlineKeyboardButton {
	var btns [][]tgbotapi.InlineKeyboardButton

	for _, opt := range opts {
		btns = append(btns, []tgbotapi.InlineKeyboardButton{
			{
				Text: opt.Label,
				URL:  &opt.URL,
			},
		})
	}

	return btns
}
