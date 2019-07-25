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

	tgToken := getTgToken()

	//Initialize telegram bot
	var err error
	bot.BotAPI, err = tg.NewBotAPI(tgToken)

	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	q := *quiz.NewQuiz(bot)
	q.StartListen()
	defer q.StopListen()

	u := tg.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	defer bot.StopReceivingUpdates()

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		if q.Stopped() {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		userId := update.Message.From.ID

		switch update.Message.Text {
		case "/start":
			q.Greetings(userId)
		case "/quiz":
			q.RequestWord(userId)
		case "/help":
			q.ShowHelp(userId)
		case "/set_lang":
			q.SelectLang(userId)
		case "/upload":
			q.AwaitUpload(userId)
		case "/cancel":
			q.CancelOperation(userId)
		default:
			go func(upd tg.Update) {
				userId := update.Message.From.ID
				if update.Message.Document != nil {

					url, err := bot.GetFileDirectURL(update.Message.Document.FileID)
					if err != nil {
						log.Fatalf(err.Error())
						return //TODO: Error handling
					}

					q.ProcessMessage(userId, update.Message.Text, url)

				} else {
					q.ProcessMessage(userId, update.Message.Text, "")
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

func getTgToken() string {
	var tgToken = *token
	if tgToken == "" {
		tgToken = os.Getenv("TG_TOKEN")
	}

	if tgToken == "" {
		log.Fatal("unable get telegram bot api token")
	}

	return tgToken
}
