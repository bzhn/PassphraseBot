package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// This variable changes in the generate.go file
var IKBWordlistChooser tgbotapi.InlineKeyboardMarkup

var IKBCancelAction = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Cancel", "system$$cancelaction"),
	),
)
