package kindle_quiz_bot

import (
	"fmt"
	tg "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
)

type QuizTelegramBot interface {
	Start() error
	Stop() error
}

type quizTelegramBot struct {
	*tg.BotAPI
	q *Quiz
}

func NewQuizTelegramBot(token string) (QuizTelegramBot, error) {

	var quizBot QuizTelegramBot

	bot := quizTelegramBot{}

	var err error
	bot.BotAPI, err = tg.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	q := newQuiz(&bot)
	bot.q = &q

	quizBot = &bot

	return quizBot, nil
}

func (b *quizTelegramBot) SendMessage(userId int, text string) error {
	msg := tg.NewMessage(int64(userId), text)
	_, err := b.Send(msg)
	if err != nil {
		return err
	}

	return nil
}

func (bot quizTelegramBot) Start() error {

	u := tg.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	if err != nil {
		return err
	}

	q := *bot.q

	q.StartListen()

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

	return nil
}

func (bot quizTelegramBot) Stop() error {

	quiz := *bot.q

	quiz.StopListen()

	err := bot.Stop()
	if err != nil {
		return fmt.Errorf("tg bot: stop receiving updates: %v", err.Error())
	}

	return nil
}
