package kindle_quiz_bot

import (
	"log"

	tg "github.com/go-telegram-bot-api/telegram-bot-api"
)

type QuizTelegramBot interface {
	Start() error
	Stop()
}

type quizTelegramBot struct {
	*tg.BotAPI
	q Quiz
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

	bot.q = NewQuiz(&bot)

	quizBot = &bot

	return quizBot, nil
}

func (bot *quizTelegramBot) SendMessage(userId int, text string) error {
	msg := tg.NewMessage(int64(userId), text)
	_, err := bot.Send(msg)
	if err != nil {
		return err
	}

	return nil
}

func (bot quizTelegramBot) Start() error {
	u := tg.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	defer bot.StopReceivingUpdates()

	if err != nil {
		return err
	}

	q := bot.q

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		go func(upd tg.Update) {
			log.Printf("[%s] %s", upd.Message.From.UserName, upd.Message.Text)
			bot.processUpdate(upd, q)
		}(update)
	}

	return nil
}

func (bot quizTelegramBot) processUpdate(update tg.Update, q Quiz) {
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
	}
}

func (bot quizTelegramBot) Stop() {
	bot.StopReceivingUpdates()
}
