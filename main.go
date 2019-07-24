package main

import (
	"database/sql"
	"fmt"
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
		case "/start":
			greetings(*user)
		case "/quiz":
			requestWord(*user)
		case "/help":
			showHelp(*user)
		case "/set_lang":
			selectLang(*user)
		case "/upload":
			awaitUpload(*user)
		case "/cancel":
			cancel(*user)
		default:
			processNonRouteUpdate(*user, update)
		}
	}
}

func processNonRouteUpdate(u user, update tgbotapi.Update) {

	log.Printf("process non route: curr state: %d\n", u.currentState)

	switch u.currentState {
	case awaitingUpload:
		tryToMigrate(u, update)
	case readyForQuestion:
		showHelp(u)
	case waitingAnswer:
		guessWord(u, update.Message.Text)
	case migrationInProgress:
		showMigrationInProgressWarn(u)
	case awaitingLanguage:
		setLanguage(u, update.Message.Text)
	}
}

func greetings(u user) {
	msg := "Yo. Firstly you have to run /upload and upload your vocab.db file.\n" +
		"Next run /quiz and have some fun, idk. You can ask me for /help also."
	sendMessageToUser(u, msg)
}

func cancel(u user) {
	if u.currentState == readyForQuestion {
		sendMessageToUser(u, "Nothing to cancel")
		return
	}

	err := updateUserState(u.id, readyForQuestion)
	if err != nil {
		return //TODO: Error handle
	}

	sendMessageToUser(u, "Done")
}

func awaitUpload(u user) {
	err := updateUserState(u.id, awaitingUpload)
	if err != nil {
		return //TODO: Error handle
	}

	sendMessageToUser(u, "Now send vocab.db file exported from your kindle")
}


func setLanguage(u user, lc string) {
	l, err := getLanguageWithCode(lc)
	if err != nil {
		sendMessageToUser(u, "Invalid language code")
		return
	}

	err = updateUserLang(u.id, l.id)
	if err != nil {
		return //TODO
	}

	sendMessageToUser(u, fmt.Sprintf("Language changed to: %s", l.localized_name))
}

func tryToMigrate(u user, update tgbotapi.Update) {
	if (update.Message.Document != nil) {
		go func(u user) {

			url, err := bot.GetFileDirectURL(update.Message.Document.FileID)
			if err != nil {
				log.Fatalf(err.Error())
				return //TODO: Error handling
			}

			err = downloadAndMigrateKindleSQLite(url, update.Message.From.ID)
			if err != nil {
				sendMessageToUser(u, "Looks like db file in incorrect format. Try again.")
				return
			}

			sendMessageToUser(u, "Migration completed. Press /quiz to start a game.")

		}(u)

		sendMessageToUser(u, "Processing...")
	}
}

func showHelp(u user) {
	msg := "" +
		"/quiz - ask a random word\n" +
		"/help - show this help\n" +
		"/set_lang - change language\n" +
		"/upload - uploading mode\n" +
		"/cancel - cancel current operation\n"
	sendMessageToUser(u, msg)
}

func showMigrationInProgressWarn(u user) {
	sendMessageToUser(u, "Migration still in progress.")
}

func selectLang(u user) {
	langs, err := getLanguages()
	if err != nil {
		return //TODO Error hanling
	}

	msg := "Select language code:\n\n"
	for _, l := range langs {
		msg += fmt.Sprintf("[%s] %s\n", l.code, l.english_name)
	}

	err = updateUserState(u.id, awaitingLanguage)
	if err != nil {
		//TODO: what?
	}

	sendMessageToUser(u, msg)
}

func sendMessageToUser(u user, text string) {
	_, _ = fmt.Fprintf(u, text)
}
