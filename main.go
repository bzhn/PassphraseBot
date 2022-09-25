package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

var (
	words    []string
	wordsLen int
	bot      *tgbotapi.BotAPI
)

func init() {
	// Load environmental variables from file .env
	godotenv.Load()

	var err error
	// Get list of words and check if it's not empty
	words, err = getWordList("https://raw.githubusercontent.com/bzhn/passph/master/wordlists/bip39_dictionary.json")
	errPanic(err)
	wordsLen = len(words)
	if wordsLen == 0 {
		panic("Length of words is 0")
	}

	// Use telegram bot
	bot, err = tgbotapi.NewBotAPI(os.Getenv("PASSPHRASEBOT_TOKEN"))
	errPanic(err)
	fmt.Printf("Authorized on account %s\n", bot.Self.UserName)
}

func main() {

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// check for new messages in a loop
	for upd := range updates {

		if upd.CallbackQuery != nil {
			handleInlineButtonClick(upd.CallbackQuery)
			continue
		}

		// msg is a message to be sent
		var msg tgbotapi.MessageConfig
		m := upd.Message
		msg.ChatID = m.Chat.ID
		if m.Text == "" {
			msg = handleUnknowMessage(m)
			botSend(msg)
			continue
		}
		if m.Text == "Generate" {
			deleteMessage(m.Chat.ID, m.MessageID)
			msg.Text = fmt.Sprintf("<code>%s</code>", generatePassphrase(words, 3, " "))
			msg.ParseMode = tgbotapi.ModeHTML
			msg.ReplyMarkup = inlPasswordOptions()
			botSend(msg)
			continue
		}
		if m.IsCommand() {
			msg = handleCommand(m)
			botSend(msg)
			continue
		}
	}
}

// delete
func prints() {
	fmt.Println(words)
}

// getWordList gets list of words from the providen url
// and returns the slice of string with words
// Error will be returned if the link is invalid, or if
// it's impossible to parse list in the url
func getWordList(url string) (words []string, err error) {
	c := http.Client{
		Timeout: 3 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}

	res, err := c.Do(req)
	if err != nil {
		return
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, &words)
	if err != nil {
		return
	}

	return
}

// errPanic receive error and if it's not nil, make panic
// It's a shortener of error handling
func errPanic(err error) {
	if err != nil {
		panic(err)
	}
}

// botSend receive MessageConfig and tries to send it
// If there is an error, the panic will be called
func botSend(msg tgbotapi.MessageConfig) {
	_, err := bot.Send(msg)
	errPanic(err)
}

// handleCommand handles commands from users
// and returns a message that has to be sent
func handleCommand(m *tgbotapi.Message) (msg tgbotapi.MessageConfig) {
	msg.ChatID = m.Chat.ID
	switch m.Command() {
	case "start":
		msg.ReplyMarkup = genButton()
		msg.Text = "Hello. Use this bot to generate strong mnemonic passwords which, however, easy to memorise!"
		return

	case "help":
		msg.ReplyMarkup = genButton()
		msg.Text = `<b>Syntax</b>
There are several examples below of how you can use this bot to generate passwords.

Generate 5-words password where separator is equals sign:
<code>5=</code>
Generate 3-words password with space as a separator:
<code>3</code>`
		msg.ParseMode = tgbotapi.ModeHTML
		return

	default:
		msg.ReplyMarkup = genButton()
		msg.Text = "Unknown command, sorry. Type /help to get help."
		return
	}
}

// handleUnknowMessage handles messages which have no text deletes them
// and returns a message that has to be sent
func handleUnknowMessage(m *tgbotapi.Message) (msg tgbotapi.MessageConfig) {
	msg.ChatID = m.Chat.ID
	msg.Text = "Sorry, I don't understand. Send me /help to get help."
	msg.ReplyMarkup = genButton()
	deleteMessage(m.Chat.ID, m.MessageID)
	return
}

// handleInlineButtonClick is called when user clicked the button on inline keyboard
func handleInlineButtonClick(cq *tgbotapi.CallbackQuery) {
	switch cq.Data {
	case "delete":
		deleteMessage(cq.Message.Chat.ID, cq.Message.MessageID)
	case "save":
		savePassword(cq.From.ID, cq.Message.Text)
	}

}

// deleteMessage takes chatID and messageID and tries to delete it
func deleteMessage(chatID int64, msgID int) error {
	dl := tgbotapi.NewDeleteMessage(chatID, msgID)
	_, err := bot.Send(dl)
	return err
}

// generatePassphrase takes the slice of words,
// amount of words and a separator. Mnemonic password will be returned
func generatePassphrase(wl []string, n uint8, s string) string {
	var passphraseWords []string

	for i := n; i > 0; i-- {
		rnd, _ := rand.Int(rand.Reader, big.NewInt(int64(len(wl))))
		passphraseWords = append(passphraseWords, (words)[rnd.Int64()])
	}

	return strings.Join(passphraseWords, s)
}

// savePassword encrypts and saves user's password in the database
func savePassword(userID int64, password string) {
	// TODO
	fmt.Println("Imagine like I'm encrypting and saving password", password, "in the database. UserID =", userID)
}

// savePassword encrypts and saves user's password in the database
func savePasswordNote(userID int64, password string, note string) {
	// TODO
	fmt.Println("Imagine like I'm encrypting and saving password", password, "in the database. UserID =", userID)
}

// genButton returns replyMarkup keyboard with one word Generate
func genButton() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Generate")))
}

// inlPasswordOptions returns replyMarkup as an inline keyboard with three options:
// delete password, save password, save password with note
func inlPasswordOptions() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete passphrase", "delete"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💾 Save", "save"),
			tgbotapi.NewInlineKeyboardButtonData("🖊️ Save with note", "save_with_name"),
		),
	)
}
