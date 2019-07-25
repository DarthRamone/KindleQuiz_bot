package kindle_quiz_bot

import (
	"database/sql"
	"fmt"
	"github.com/DarthRamone/gtranslate"
	"log"
	"strings"
	"time"
)

var db *sql.DB
var sender MessageSender
var results = make(chan guessResult)
var requests = make(chan guessRequest)
var stop = make(chan struct{})

type guessRequest struct {
	userId int
	word   word
}

type guessParams struct {
	word   word
	guess  string
	userId int
}

type guessResult struct {
	params      guessParams
	translation string
}

type word struct {
	id   int
	word string
	stem string
	lang *lang
}

type user struct {
	id              int
	currentState    int
	currentLanguage *lang
}

type lang struct {
	id             int
	code           string
	english_name   string
	localized_name string
}

type MessageSender interface {
	SendMessage(userId int, text string) error
}


func tellResult(r guessResult) {
	if r.correct() {
		_ = sender.SendMessage(r.params.userId, "Your answer is correct")
	} else {
		_ = sender.SendMessage(r.params.userId, fmt.Sprintf("Your answer is incorrect. Correct answer: %s\n", r.translation))
	}
}

func ask(r guessRequest) {
	w := r.word
	q := fmt.Sprintf("Word is: %s; Stem: %s; Lang: %s\n", w.word, w.stem, w.lang.english_name)
	_ = sender.SendMessage(r.userId, q) //TODO: error handle
}

func (t *guessResult) correct() bool {
	return compareWords(t.params.guess, t.translation)
}

func StartListen(s MessageSender) {

	if db == nil {
		err := connectToDB()
		if err != nil {
			log.Fatalf("db connect: %v", err.Error())
		}
	}

	sender = s

	go func() {
		for {
			select {
			case <-stop:
				for range results {
				}
				for range requests {
				}
			case req := <-requests:
				go func() {
					ask(req)
				}()
			case res := <-results:
				go func() {
					tellResult(res)
				}()
			}
		}
	}()
}

func StopListen() {
	db.Close()
	close(stop)
}

func RequestWord(userId int) {
	go func(id int) {

		log.Println("request word")

		w, err := getRandomWord(userId)

		if err == sql.ErrNoRows {
			//TODO: error handle
			_ = sender.SendMessage(id, "No words found. Please run /upload and follow instructions")
			return
		}

		if err != nil {
			log.Println("report error: random word")
			//TODO Error handle
			_ = sender.SendMessage(id, err.Error())
			return
		}

		err = setLastWord(id, *w)
		if err != nil {
			log.Println("report error: set last word")
			//TODO Error handle
			_ = sender.SendMessage(id, err.Error())
			return
		}

		log.Println("send request")
		r := guessRequest{id, *w}
		requests <- r

		err = updateUserState(id, waitingAnswer)
		if err != nil {
			//TODO: what?
		}
	}(userId)
}

func ShowHelp(userId int) {
	msg := "" +
		"/quiz - ask a random word\n" +
		"/help - show this help\n" +
		"/set_lang - change language\n" +
		"/upload - uploading mode\n" +
		"/cancel - cancel current operation\n"

	//TODO: error handling
	_ = sender.SendMessage(userId, msg)
}

func Greetings(userId int) {
	msg := "Yo. Firstly you have to run /upload and upload your vocab.db file.\n" +
		"Next run /quiz and have some fun, idk. You can ask me for /help also."

	//TODO: error handling
	_ = sender.SendMessage(userId, msg)
}

func SelectLang(userId int) {
	langs, err := getLanguages()
	if err != nil {
		return //TODO Error hanling
	}

	msg := "Select language code:\n\n"
	for _, l := range langs {
		msg += fmt.Sprintf("[%s] %s\n", l.code, l.english_name)
	}

	err = updateUserState(userId, awaitingLanguage)
	if err != nil {
		//TODO: what?
	}

	//TODO: error handling
	_ = sender.SendMessage(userId, msg)
}

func AwaitUpload(userId int) {
	err := updateUserState(userId, awaitingUpload)
	if err != nil {
		return //TODO: Error handle
	}

	//TODO: error handling
	_ = sender.SendMessage(userId, "Now send vocab.db file exported from your kindle")
}

func CancelOperation(userId int) {

	user, err := getUser(userId)
	if err != nil {
		return //TODO: error handle
	}

	if user.currentState == readyForQuestion {
		//TODO error handle
		_ = sender.SendMessage(userId, "Nothing to cancel")
		return
	}

	err = updateUserState(userId, readyForQuestion)
	if err != nil {
		return //TODO: Error handle
	}

	//TODO: error handle
	_ = sender.SendMessage(userId, "Done")
}

func ProcessMessage(userId int, text, documentUrl string) {

	u, err := getUser(userId)
	if err != nil {
		return //TODO: error handle
	}

	log.Printf("process non route: curr state: %d\n", u.currentState)

	switch u.currentState {
	case awaitingUpload:
		tryToMigrate(userId, documentUrl)
	case readyForQuestion:
		ShowHelp(u.id)
	case waitingAnswer:
		guessWord(*u, text)
	case migrationInProgress:
		showMigrationInProgressWarn(userId)
	case awaitingLanguage:
		setLanguage(*u, text)
	}
}

func guessWord(usr user, guess string) {

	go func(u user) {

		word, err := getLastWord(u.id)
		if err != nil {
			//TODO Error handle
			_ = sender.SendMessage(u.id, err.Error())
			return
		}

		translated, err := translateWord(*word, u.currentLanguage)
		if err != nil {
			//TODO Error handle
			_ = sender.SendMessage(u.id, err.Error())
			return
		}

		err = deleteLastWord(u.id)

		p := guessParams{*word, guess, u.id}
		r := guessResult{p, translated}
		results <- r

		err = writeAnswer(r)
		if err != nil {
			log.Printf("Failed to write answer: %v\n", err.Error())
		}

		err = updateUserState(u.id, readyForQuestion)
		if err != nil {
			//TODO: what?
		}
	}(usr)

}

func tryToMigrate(userId int, url string) {
	err := downloadAndMigrateKindleSQLite(url, userId)
	if err != nil {
		//TODO: error handle
		_ = sender.SendMessage(userId, "Looks like db file in incorrect format. Try again.")
		return
	}

	//TODO: error handle
	_ = sender.SendMessage(userId, "Migration completed. Press /quiz to start a game.")
}

func translateWord(w word, dst *lang) (string, error) {

	translated, err := gtranslate.TranslateWithParams(
		w.word,
		gtranslate.TranslationParams{
			From:  w.lang.code,
			To:    dst.code,
			Delay: time.Second,
			Tries: 5,
		},
	)

	if err != nil {
		return "", err
	}

	return translated, nil
}

func compareWords(w1, w2 string) bool {

	s1 := strings.ToLower(strings.Trim(w1, " "))
	s2 := strings.ToLower(strings.Trim(w2, " "))

	return s1 == s2
}

func setLanguage(u user, lc string) {
	l, err := getLanguageWithCode(lc)
	if err != nil {
		//TODO: error handle
		_ = sender.SendMessage(u.id, "Invalid language code")
		return
	}

	err = updateUserLang(u.id, l.id)
	if err != nil {
		return //TODO
	}

	//TODO: error handle
	_ = sender.SendMessage(u.id, fmt.Sprintf("Language changed to: %s", l.localized_name))
}

func showMigrationInProgressWarn(userId int) {
	//TODO: error handle
	_ = sender.SendMessage(userId, "Migration still in progress.")
}

func Stopped() bool {
	select {
	case <- stop:
		return true
	default:
		return false
	}
}

func connectToDB() error {
	connStr := "user=postgres dbname=vocab port=32770 sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	return nil
}
