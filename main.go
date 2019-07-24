package main

import (
	"database/sql"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

var db *sql.DB
var bot *tgbotapi.BotAPI

func main() {

	var err error

	//Initialize PostgreSQL connection
	connStr := "user=postgres dbname=vocab port=32770 sslmode=disable"

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Couldn't connect to database: %v", err.Error())
	}
	defer db.Close()

	//Initialize users map
	ids, err := getAllUserIds()
	//Ignoring ErrNoRows, it will create 0-len map anyway
	if err == sql.ErrNoRows {
		err = nil
	}

	users := make(map[int]bool, len(ids))
	for u := range users {
		users[u] = true
	}

	//Initialize quiz
	quizStartListen()

	//Initialize telegram bot
	bot, err = tgbotapi.NewBotAPI("***REMOVED***")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		var user *user
		_, ok := users[update.Message.From.ID]
		if ok {
			user, err = getUser(update.Message.From.ID)
		} else {
			user, err = createUser(update.Message.From.ID)
			users[user.id] = true
		}

		if err != nil {
			//TODO: log error
			log.Fatalf(err.Error())
			continue
		}

		switch update.Message.Text {
		case "/quiz":

			log.Println("quiz case")

			if err != nil {
				log.Fatalf(err.Error())
				break
			}

			requestWord(*user)

		default:
			if user.currentState == awaitingUpload && update.Message.Document != nil {

				go func(u tgbotapi.Update) {

					url, err := bot.GetFileDirectURL(update.Message.Document.FileID)
					if err != nil {
						log.Fatalf(err.Error())
						return //TODO: Error handling
					}

					err = downloadAndMigrateKindleSQLite(url, update.Message.From.ID)
					if err != nil {
						log.Fatalf(err.Error())
						return //TODO: Error handling
					}

					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Migration completed. Press /quiz to start.")
					_, _ = bot.Send(msg) //TODO: error handling

				}(update)

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Processing...")
				_, _ = bot.Send(msg) //TODO: error handling
			}

		}
	}
}
