package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bzhn/strkit"
	"github.com/gomodule/redigo/redis"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

type LastAction string

const (
	laSetSeparator LastAction = "setseparator"
	laSetNubmer    LastAction = "setnumberofwords"
	laSetEncPass   LastAction = "setencryptionpass"
)

var (
	words    []string
	wordsLen int
	bot      *tgbotapi.BotAPI
)

var (
	ErrCantParseCtx = errors.New("Can't parse context value")

	ErrSeparatorTooLong          = errors.New("Separator is too long")
	ErrNumberOfWordsTooBig       = errors.New("Number of words is too big")
	ErrNumberOfWordsLessThanZero = errors.New("Number of words is less than zero")
	ErrEncPassTooLong            = errors.New("Password for encryption is too long")
)

func init() {
	// Load environmental variables from file .env
	godotenv.Load("/secret/.env") // For docker
	godotenv.Load()               // For host

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

	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func ToJson(data interface{}) string {
	jsonData, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return fmt.Sprint(err)
	}
	return string(jsonData)
}

func main() {

	// # Test

	pool := NewRedisPool("redis:6379")
	conn := NewConn(pool)
	defer conn.Close()
	mainCtx := context.WithValue(context.Background(), "redis-conn", conn)

	sr := conn.NewRedisSetRequest()
	ttl, err := time.Parse("20060102-1504", "20221119-1428")
	log.Print("time parse ERROR: ", err, ttl)
	sr.Key("newvar").Value("I'm working").ExpireAt(ttl)
	log.Print(sr.SetPersonList(172035, 2))

	// * Test

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// check for new messages in a loop
	for upd := range updates {

		if upd.CallbackQuery != nil {
			updCtx := context.WithValue(mainCtx, "person", upd.CallbackQuery.From.ID)
			handleInlineButtonClick(updCtx, upd.CallbackQuery)
			continue
		}

		if upd.FromChat() == nil || upd.FromChat().Type != "private" {
			log.Printf("The message is not private:\n%s", ToJson(upd.FromChat()))
			continue
		}

		if upd.Message.Text == "" {
			log.Printf("Got non-text message from chat")
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
		if m.Text == "Generate" || m.Text == "gen" || m.Text == "generate" {
			deleteMessage(m.Chat.ID, m.MessageID)
			generatePassphrase(mainCtx, m.Chat.ID)
			continue
		}
		if m.IsCommand() {
			updCtx := context.WithValue(mainCtx, "person", m.Chat.ID)
			msg = handleCommand(updCtx, m)
			botSend(msg)
			continue
		}
		if m.Text != "" {
			updCtx := context.WithValue(mainCtx, "person", m.Chat.ID)
			updCtx = context.WithValue(updCtx, "msg", m.Text)
			err := handleLastActionText(updCtx)
			if err != nil {
				log.Println(err)
			}
		}

	}
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
// If there is an error
func botSend(msg tgbotapi.MessageConfig) {
	_, err := bot.Send(msg)
	if err != nil {
		fmt.Println("ERROR:", err)
	}
}

// handleCommand handles commands from users
// and returns a message that has to be sent
func handleCommand(ctx context.Context, m *tgbotapi.Message) (msg tgbotapi.MessageConfig) {
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
<code>3</code>

Currently syntax parsing is not supported. Type /number to change number of generated words and /sep to change the separator.`
		msg.ParseMode = tgbotapi.ModeHTML
		return

	case "number": // set number of words in generated passphrases
		setLastAction(ctx, laSetNubmer)
		msg.Text = "Choose number of words in the passphrases that will be generated. The value have to contain only numbers and nothing more."
		msg.ReplyMarkup = IKBCancelAction
	case "sep": // set separator in generated passphrases
		setLastAction(ctx, laSetSeparator)
		msg.Text = "Type separator of the passphrases that will be generated. It can be <code>-</code> or <code>_</code> or even newline, for instance. Separator has to be less than 10 bytes long.\nTo set space as a separator, type <code>\\</code> (just backslash). For newline, type <code>\\n</code>. Note that first backslash will be removed from any of your messages (if you use it), so for one backslash as a separator you have to specify two backslashes."
		msg.ParseMode = tgbotapi.ModeHTML
		msg.ReplyMarkup = IKBCancelAction

	case "list":
		msg.ReplyMarkup = IKBWordlistChooser
		msg.Text = `<b>Select desired wordlist</b>
		
Here are some examples of generated passphrases:
BIP39
<code>spider music exhibit</code>

<b>Wordle</b>
(only 5-chars words, 12000ish words in the list)
<code>spews livid airns</code>

<b>Dice Long</b>
(6^5 = 7776 words)
<code>freebee attendant empirical</code>

<b>Dice Short 1</b>
Featuring only short words (6^4 = 1296 words)
<code>stack lip visa</code>

<b>Dice Short 2</b>
Featuring longer words that may be more memorable (6^4 = 1296 words)
<code>liquid mapmaker shyness</code>
`
		msg.ParseMode = tgbotapi.ModeHTML

	case "addlist":
		msg.ReplyMarkup = genButton()
		msg.Text = "In development. Later it will be possible to add custom lists."
	case "vault":
		msg.ReplyMarkup = genButton()
		msg.Text = "In development. Later you'll have access to your vault, where passwords are stored"
	case "encryption":
		msg.ReplyMarkup = genButton()
		msg.Text = "In development. Setup your encryption settings. Disable/enable encryption and change password for encryption"
	case "search":
		msg.ReplyMarkup = genButton()
		msg.Text = "In development. Search your stored passphrases"

	default:
		msg.ReplyMarkup = genButton()
		msg.Text = "Unknown command, sorry. Type /help to get help."
		return
	}

	return
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
func handleInlineButtonClick(ctx context.Context, cq *tgbotapi.CallbackQuery) {
	log.Println("Inline button click:", cq.Data)

	if complexDataParts := strings.Split(cq.Data, "$$"); len(complexDataParts) == 2 {
		switch complexDataParts[0] {
		case "system":
			switch complexDataParts[1] {
			case "cancel":
				deleteMessage(cq.From.ID, cq.Message.MessageID)
			case "cancelaction":
				deleteMessage(cq.From.ID, cq.Message.MessageID)
				removeLastAction(ctx)
				callbackAnswer(cq.ID, "Last action successfully removed!")
			}
		case "setwl":
			if c, ok := ctx.Value("redis-conn").(RedisConn); ok {
				wl, err := strconv.Atoi(complexDataParts[1])
				if err != nil {
					log.Println("Can't convert to int second part of cq data", err)
					return
				}
				err = c.NewRedisSetRequest().SetPersonList(cq.From.ID, WL(wl))
				if err != nil {
					log.Println(err)
					return
				}
				callbackAnswer(cq.ID, fmt.Sprintf("%s is your new wordlist", WL(wl).ShortName()))

				return

			} else {
				log.Println("not ok")
			}
		}
		return
	}

	switch cq.Data {
	case "regenerate":
		err := regeneratePassword(ctx, cq)
		if err != nil {
			log.Println(err)
		}
	case "delete":
		deleteMessage(cq.Message.Chat.ID, cq.Message.MessageID)
	case "save":
		savePassword(cq.From.ID, cq.Message.Text)
		callbackAnswer(cq.ID, "Your password wasn't saved. This functionality is under maintenance.")

	}

}

// deleteMessage takes chatID and messageID and tries to delete it
func deleteMessage(chatID int64, msgID int) error {
	dl := tgbotapi.NewDeleteMessage(chatID, msgID)
	_, err := bot.Send(dl)
	return err
}

// Push a new callback message to the user
func callbackAnswer(cqID string, text string) {
	c := tgbotapi.NewCallback(cqID, text)
	bot.Request(c)
}

// regeneratePassword context with redis connection and callback query
// and tries to edit it with new generated password
func regeneratePassword(ctx context.Context, cq *tgbotapi.CallbackQuery) error {
	chatID := cq.From.ID
	msgID := cq.Message.MessageID

	// Get list of a user
	if rc, ok := ctx.Value("redis-conn").(RedisConn); ok {
		rg := rc.NewRedisGetRequest().ID(chatID)
		wl := rg.GetPersonList()
		n := func() int {
			if n, err := rg.GetWordsNumber(); err == nil {
				if n > 0 {
					return n
				} else {
					log.Println("Amount of words is less than 1")
					return 3
				}
			} else {
				log.Println(err)
				return 3
			}
		}()

		sep, err := rg.GetSeparator()
		if err != nil {
			log.Println(err)
			sep = "-"
		}
		gpc := NewGeneratePasswordConfig().Wordlist(wl).Length(n).Separator(sep)
		passphrase, err := gpc.Generate()
		if err != nil {
			return err
		}

		ec := tgbotapi.NewEditMessageText(chatID, msgID, fmt.Sprintf("<code>%s</code>", tgbotapi.EscapeText(tgbotapi.ModeHTML, passphrase)))
		ec.ParseMode = tgbotapi.ModeHTML
		ec.ReplyMarkup = inlPasswordOptions()
		_, err = bot.Request(ec)
		if err != nil {
			return err
		}

		callbackAnswer(cq.ID, fmt.Sprintf("You use %s wordlist", wl.ShortName()))

		return nil
	}

	return errors.New("Can't connect to Redis")
}

// generatePassphrase takes the slice of words,
// amount of words and a separator. Mnemonic password will be returned
func generatePassphrase(ctx context.Context, chatID int64) error {

	if rc, ok := ctx.Value("redis-conn").(RedisConn); ok {
		rg := rc.NewRedisGetRequest().ID(chatID)
		wl := rg.GetPersonList()
		n := func() int {
			if n, err := rg.GetWordsNumber(); err == nil {
				if n > 0 {
					return n
				} else {
					log.Println("Amount of words is less than 1")
					return 3
				}
			} else {
				log.Println(err)
				return 3
			}
		}()

		sep, err := rg.GetSeparator()
		if err != nil {
			log.Println(err)
			sep = "-"
		}
		gpc := NewGeneratePasswordConfig().Wordlist(wl).Length(n).Separator(sep)
		passphrase, err := gpc.Generate()
		if err != nil {
			return err
		}

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("<code>%s</code>", tgbotapi.EscapeText(tgbotapi.ModeHTML, passphrase)))
		msg.ParseMode = tgbotapi.ModeHTML
		msg.ReplyMarkup = inlPasswordOptions()
		_, err = bot.Send(msg)
		if err != nil {
			return err
		}

		return nil
	}

	return errors.New("Can't connect to Redis")
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

// inlPasswordOptions returns replyMarkup as an inline keyboard with the following options:
// delete password, regenerate password, save password, save password with note
func inlPasswordOptions() *tgbotapi.InlineKeyboardMarkup {
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🗑️ Delete passphrase", "delete"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔀 Regenerate passphrase", "regenerate"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💾 Save", "save"),
			tgbotapi.NewInlineKeyboardButtonData("🖊️ Save with note", "save_with_name"),
		),
	)

	return &inlineKeyboard
}

func NewRedisPool(address string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:   80,
		MaxActive: 12000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", address)
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}
}

// Try to parse integer
// 0 is returned if it's impossible to do so
func ParseInt(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func handleLastActionText(ctx context.Context) error {

	conn, ok := ctx.Value("redis-conn").(RedisConn)
	if !ok {
		log.Println(ErrCantParseCtx)
		return ErrCantParseCtx
	}

	la, err := getLastAction(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	value, ok := ctx.Value("msg").(string)
	if !ok {
		log.Println(ErrCantParseCtx)
		return ErrCantParseCtx
	}

	pid, ok := ctx.Value("person").(int64)
	if !ok {
		log.Println(ErrCantParseCtx)
		return ErrCantParseCtx
	}

	msg := tgbotapi.NewMessage(pid, "Error!")

	switch la {
	case laSetNubmer:
		if n := ParseInt(value); n > 0 {
			if n > 200 {
				msg.Text = "Number of words have to be less than 200"
				msg.ReplyMarkup = IKBCancelAction
				bot.Send(msg)
				log.Println(ErrNumberOfWordsTooBig)
				return ErrNumberOfWordsTooBig
			}
			err := conn.NewRedisSetRequest().SetNumberOfWords(pid, n)
			if err != nil {
				msg.Text = "Can't set number of words"
				msg.ReplyMarkup = IKBCancelAction
				bot.Send(msg)
				log.Println(err)
				return err
			}
			msg.Text = "Number of words successfully changed!"
			bot.Send(msg)
			return removeLastAction(ctx)
		} else {
			msg.Text = "Number of words have to be positive"
			msg.ReplyMarkup = IKBCancelAction
			bot.Send(msg)
			log.Println(ErrNumberOfWordsLessThanZero)
			return ErrNumberOfWordsLessThanZero
		}
	case laSetSeparator:
		if strkit.Fitsb(value, 8) {
			switch value {
			case `\`:
				value = " "
			case `\n`:
				value = "\n"
			default:
				if len(value) > 1 && value[0] == '\\' {
					value = value[1:]
				}
			}
			err := conn.NewRedisSetRequest().SetSeparator(pid, value)
			if err != nil {
				msg.Text = "Error on the server side. Sorry."
				bot.Send(msg)
				log.Println(err)
				return err
			}

			msg.Text = "Separator successfully changed"
			bot.Send(msg)
			return removeLastAction(ctx)
		} else {
			msg.Text = "Separator have to be less than 8 bytes long"
			msg.ReplyMarkup = IKBCancelAction
			bot.Send(msg)
			log.Println(ErrSeparatorTooLong)
			return ErrSeparatorTooLong
		}
	case laSetEncPass:
		return removeLastAction(ctx)

	}
	removeLastAction(ctx)

	return nil

}

func removeLastAction(ctx context.Context) error {
	if conn, ok := ctx.Value("redis-conn").(RedisConn); ok {
		pid, ok := ctx.Value("person").(int64)
		if !ok {
			log.Println(ErrCantParseCtx)
			return ErrCantParseCtx
		}
		return conn.NewRedisDelRequest().ID(pid).DeleteLastAction()
	}

	log.Println(ErrCantParseCtx)
	return ErrCantParseCtx
}

func setLastAction(ctx context.Context, lastAction LastAction) error {
	if conn, ok := ctx.Value("redis-conn").(RedisConn); ok {
		pid, ok := ctx.Value("person").(int64)
		if !ok {
			log.Println(ErrCantParseCtx)
			return ErrCantParseCtx
		}

		return conn.NewRedisSetRequest().SetLastAction(pid, lastAction)

	}
	log.Println(ErrCantParseCtx)
	return ErrCantParseCtx
}

func getLastAction(ctx context.Context) (LastAction, error) {
	if conn, ok := ctx.Value("redis-conn").(RedisConn); ok {
		pid, ok := ctx.Value("person").(int64)

		if !ok {
			log.Println(ErrCantParseCtx)
			return "", ErrCantParseCtx
		}

		return conn.NewRedisGetRequest().ID(pid).GetLastAction()

	}
	log.Println(ErrCantParseCtx)
	return "", ErrCantParseCtx
}
