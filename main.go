package main

import (
	"flag"
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
)

type botAPI struct {
	*tg.BotAPI
}


var bot = botAPI{}

var token = flag.String("token", "", "telegram API bot token")

func main() {

	flag.Parse()

	//Initialize quiz
	StartListen(_messageSender(sendMessageToUser))
	defer StopListen()

	//Initialize telegram bot
	var err error
	bot.BotAPI, err = tg.NewBotAPI(*token)

	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tg.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		if Stopped() {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		userId := update.Message.From.ID

		switch update.Message.Text {
		case "/start":
			Greetings(userId)
		case "/quiz":
			RequestWord(userId)
		case "/help":
			ShowHelp(userId)
		case "/set_lang":
			SelectLang(userId)
		case "/upload":
			AwaitUpload(userId)
		case "/cancel":
			CancelOperation(userId)
		default:
			if update.Message.Document != nil {
				go func(id int) {

					url, err := bot.GetFileDirectURL(update.Message.Document.FileID)
					if err != nil {
						log.Fatalf(err.Error())
						return //TODO: Error handling
					}

					ProcessMessage(id, update.Message.Text, url)
				}(userId)

			} else {
				ProcessMessage(userId, update.Message.Text, "")
			}
		}
	}
}

func sendMessageToUser(userId int, text string) error {
	msg := tg.NewMessage(int64(userId), text)
	_, err := bot.Send(msg)
	if err != nil {
		return err
	}

	return nil
}

type _messageSender func(int, string) error

func (s _messageSender) sendMessage(userId int, text string) error {
	return s(userId, text)
}
