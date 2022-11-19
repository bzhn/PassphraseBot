package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type GeneratePasswordConfig struct {
	length    int
	separator string
	wordlist  WL
}

// Const number of one of several wordlists
type WL int

// bip39_en - English bip39 wordlist that is used as a human-readable private key for crypto wallet
//
// wordle_en - English list of 12000-ish number of 5-chars length words
//
// dice_long_en - List of words for use with five dice (6^5 = 7776 words)
//
// dice_short1_en - Featuring only short words, for use with four dice (6^4 = 1296 words)
//
// dice_short2_en - for use with four dice, featuring longer words that may be more memorable (6^4 = 1296 words)
//
// BIP39 wordlist - https://github.com/hatgit/BIP39-wordlist-printable-en/blob/master/BIP39-en-printable.txt
//
// All dice lists are here: https://www.eff.org/dice
//
// dice_long_en - https://www.eff.org/files/2016/07/18/eff_large_wordlist.txt
//
// dice_short1_en - https://www.eff.org/files/2016/09/08/eff_short_wordlist_1.txt
//
// dice_short2_en - https://www.eff.org/files/2016/09/08/eff_short_wordlist_2_0.txt
const (
	bip39_en WL = iota
	wordle_en
	dice_long_en
	dice_short1_en
	dice_short2_en
	endofwl
)

type Wordlist struct {
	size        int
	words       *[]string
	uri         string
	name        string
	description string
}

// A map with slices of words
var Wordlists = make(map[WL]*Wordlist)

func (wl WL) ShortName() string {
	return fmt.Sprint(wlNames[wl])
}

// Names of each wordlist
var wlNames = map[WL]string{
	bip39_en:       `BIP39`,
	wordle_en:      `Wordle`,
	dice_long_en:   `Dice Long`,
	dice_short1_en: `Dice Short 1`,
	dice_short2_en: `Dice Short 2`,
}

// Length of each wordlist (used to save memory when allocating it for lists)
var wlCapacities = map[WL]int{
	bip39_en:       2048,
	wordle_en:      12972,
	dice_long_en:   7776,
	dice_short1_en: 1296,
	dice_short2_en: 1296,
}

// Links where you can download wordlists in JSON format
var wlLink = map[WL]string{
	bip39_en:       `https://raw.githubusercontent.com/bzhn/passph/master/wordlists/bip39_dictionary.json`,
	wordle_en:      `https://raw.githubusercontent.com/bzhn/passph/master/wordlists/wordle-powerlanguage.json`,
	dice_long_en:   `https://raw.githubusercontent.com/bzhn/passph/master/wordlists/eff_large_wordlist.json`,
	dice_short1_en: `https://raw.githubusercontent.com/bzhn/passph/master/wordlists/eff_short_wordlist_1.json`,
	dice_short2_en: `https://raw.githubusercontent.com/bzhn/passph/master/wordlists/eff_short_wordlist_2_0.json`,
}

func panicIfEmpty(wl []string) {
	if len(wl) == 0 {
		log.Panic("Fetched wordlist is empty")
	}
}

func init() {
	var err error
	// Fill map with wordlists
	for wl := WL(0); wl < endofwl; wl++ {
		// wlSlice := make([]string, 0, wlCapacities[wl])
		// wlSlice, err = getWordList(wlLink[wl])
		Wordlists[wl] = &Wordlist{
			size:        wlCapacities[wl],
			uri:         wlLink[wl],
			name:        wlNames[wl],
			description: "",
		}

		Wordlists[wl].Fill()
		errPanic(err)
		// panicIfEmpty(wordlist[wl])
		// wordlist[wl] = wlSlice
	}

	// Keyboard on /list command
	IKBWordlistChooser = func() tgbotapi.InlineKeyboardMarkup {
		var ikb [][]tgbotapi.InlineKeyboardButton

		// Loop through all wordlists and add them to keyboard
		for n := WL(0); n < endofwl; n++ {
			var ikbrow []tgbotapi.InlineKeyboardButton
			ikbrow = append(ikbrow, tgbotapi.NewInlineKeyboardButtonData(Wordlists[n].Name(), fmt.Sprintf("setwl$$%d", n)))

			// Finally, add the cancel button
			if n+1 == endofwl {
				if n%2 == 0 {
					// Place Cancel near the last element
					ikbrow = append(ikbrow, tgbotapi.NewInlineKeyboardButtonData("Cancel", "system$$cancel"))
					ikb = append(ikb, ikbrow)
				} else {
					// Finish current row and create a new one with one cancel button
					ikb = append(ikb, ikbrow)
					ikb = append(ikb, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Cancel", "system$$cancel")))
				}
				continue
			}

			// Try to add one button next to first
			if n+1 < endofwl {
				n++
				ikbrow = append(ikbrow, tgbotapi.NewInlineKeyboardButtonData(Wordlists[n].Name(), fmt.Sprintf("setwl$$%d", n)))
			}

			ikb = append(ikb, ikbrow)
		}
		return tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: ikb,
		}
	}()
}

func NewGeneratePasswordConfig() *GeneratePasswordConfig {
	config := new(GeneratePasswordConfig)
	config.length = 3
	config.separator = " "
	config.wordlist = bip39_en
	return config
}

// Change amount of words in the future passphrase
func (gpc *GeneratePasswordConfig) Length(n int) *GeneratePasswordConfig {
	gpc.length = n
	return gpc
}

// Change separator in the future passphrase
func (gpc *GeneratePasswordConfig) Separator(s string) *GeneratePasswordConfig {
	gpc.separator = s
	return gpc
}

// Change wordlist of the future passphrase
func (gpc *GeneratePasswordConfig) Wordlist(wl WL) *GeneratePasswordConfig {
	gpc.wordlist = wl
	return gpc
}

// Change wordlist of the future passphrase
func (gpc *GeneratePasswordConfig) Valid() bool {
	if _, ok := Wordlists[gpc.wordlist]; !ok {
		return false
	}

	if len(*Wordlists[gpc.wordlist].words) == 0 {
		return false
	}

	return true
}

// Change wordlist of the future passphrase
func (gpc *GeneratePasswordConfig) Generate() (string, error) {
	if !gpc.Valid() {
		return " ", errors.New("Generate password config is not valid")
	}

	var parts []string

	for i := gpc.length; i > 0; i-- {
		rnd, _ := rand.Int(rand.Reader, big.NewInt(int64(Wordlists[gpc.wordlist].Size())))
		parts = append(parts, (*Wordlists[gpc.wordlist].words)[rnd.Int64()])
	}

	return strings.Join(parts, gpc.separator), nil
}

// Return size of the wordlist
func (wl *Wordlist) Size() int {
	return wl.size
}

func (wl *Wordlist) Words() *[]string {
	return wl.words
}

func (wl *Wordlist) URI() string {
	return wl.uri
}

func (wl *Wordlist) Name() string {
	return wl.name
}

func (wl *Wordlist) Description() string {
	return wl.description
}

// Download wordlist from the internet and insert words to the wordlist
func (wl *Wordlist) Fill() error {
	words := make([]string, 0, wl.size) // TODO: how to remove this assignment?
	wl.words = &words

	c := http.Client{
		Timeout: 3 * time.Second,
	}
	req, err := http.NewRequest(http.MethodGet, wl.uri, nil)
	if err != nil {
		return err
	}

	res, err := c.Do(req)
	if err != nil {
		return err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)

	json.Unmarshal(body, wl.words)
	if err != nil {
		return err
	}

	return nil
}
