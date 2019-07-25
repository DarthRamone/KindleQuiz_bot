package main

import (
	"flag"
	quiz "github.com/DarthRamone/KindleQuiz_bot/internal/app/kindle_quiz_bot"
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"os"
)

type botAPI struct {
	*tg.BotAPI
}

var bot = botAPI{}

var token = flag.String("token", "", "telegram API bot token")

func main() {

	flag.Parse()

	var tgToken = *token
	if tgToken == "" {
		tgToken = os.Getenv("TG_TOKEN")
	}

	//Initialize telegram bot
	var err error
	bot.BotAPI, err = tg.NewBotAPI(tgToken)

	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	//Initialize quiz
	quiz.StartListen(bot)
	defer quiz.StopListen()

	u := tg.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		if quiz.Stopped() {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		userId := update.Message.From.ID

		switch update.Message.Text {
		case "/start":
			quiz.Greetings(userId)
		case "/quiz":
			quiz.RequestWord(userId)
		case "/help":
			quiz.ShowHelp(userId)
		case "/set_lang":
			quiz.SelectLang(userId)
		case "/upload":
			quiz.AwaitUpload(userId)
		case "/cancel":
			quiz.CancelOperation(userId)
		default:
			go func(upd tg.Update) {
				userId := update.Message.From.ID
				if update.Message.Document != nil {

					url, err := bot.GetFileDirectURL(update.Message.Document.FileID)
					if err != nil {
						log.Fatalf(err.Error())
						return //TODO: Error handling
					}

					quiz.ProcessMessage(userId, update.Message.Text, url)

				} else {
					quiz.ProcessMessage(userId, update.Message.Text, "")
				}
			}(update)
		}
	}
}

func (s botAPI) SendMessage(userId int, text string) error {
	msg := tg.NewMessage(int64(userId), text)
	_, err := bot.Send(msg)
	if err != nil {
		return err
	}

	return nil
}
