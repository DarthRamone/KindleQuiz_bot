package kindle_quiz_bot

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/DarthRamone/gtranslate"
	"log"
	"strings"
	"time"
)

type Quiz interface {
	StartListen()
	StopListen()
	Stopped() bool
	Greetings(userId int)
	ShowHelp(userId int)
	RequestWord(userId int)
	SelectLang(userId int)
	AwaitUpload(userId int)
	CancelOperation(userId int)
	ProcessMessage(userId int, text, documentUrl string)
}

type quiz struct {
	crud     *crud
	sender   messageSender
	cancel   context.CancelFunc
	ctx      context.Context
}

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
	currentState    userState
	currentLanguage *lang
}

type lang struct {
	id             int
	code           string
	english_name   string
	localized_name string
}

type messageSender interface {
	SendMessage(userId int, text string) error
}

func (q *quiz) StartListen() {
	if q.crud == nil {
		err := q.connectToDB()
		if err != nil {
			log.Fatalf("db connect: %v", err.Error())
		}
	}

	q.ctx = context.Background()
	q.ctx, q.cancel = context.WithCancel(q.ctx)
}

func (q *quiz) StopListen() {
	q.crud.close()
	q.cancel()
}

func (q *quiz) RequestWord(userId int) {
	go func(id int) {

		log.Println("request word")

		w, err := q.crud.getRandomWord(userId)

		if err == sql.ErrNoRows {
			//TODO: error handle
			_ = q.sender.SendMessage(id, "No words found. Please run /upload and follow instructions")
			return
		}

		if err != nil {
			log.Println("report error: random word")
			q.sendMessage(id, err.Error())
			return
		}

		err = q.crud.setLastWord(id, *w)
		if err != nil {
			log.Println("report error: set last word")
			q.sendMessage(id, err.Error())
			return
		}

		log.Println("send request")
		r := guessRequest{id, *w}
		q.ask(r)

		err = q.crud.updateUserState(id, waitingAnswer)
		if err != nil {
			//TODO: what?
		}
	}(userId)
}

func (q *quiz) ShowHelp(userId int) {
	msg := `
/quiz - ask a random word
/help - show this help
/set_lang - change language
/upload - uploading mode
/cancel - cancel current operation
`
	q.sendMessage(userId, msg)
}

func (q *quiz) Greetings(userId int) {
	msg := `Yo. Firstly you have to run /upload and upload your vocab.db file. 
Next run /quiz and have some fun, idk. You can ask me for /help also.`
	q.sendMessage(userId, msg)
}

func (q *quiz) SelectLang(userId int) {
	langs, err := q.crud.getLanguages()
	if err != nil {
		return //TODO Error hanling
	}

	msg := "Select language code:\n\n"
	for _, l := range langs {
		msg += fmt.Sprintf("[%s] %s\n", l.code, l.english_name)
	}

	err = q.crud.updateUserState(userId, awaitingLanguage)
	if err != nil {
		//TODO: what?
	}

	q.sendMessage(userId, msg)
}

func (q *quiz) AwaitUpload(userId int) {
	err := q.crud.updateUserState(userId, awaitingUpload)
	if err != nil {
		return //TODO: Error handle
	}

	q.sendMessage(userId, "Now send vocab.db file exported from your kindle")
}

func (q *quiz) CancelOperation(userId int) {

	user, err := q.crud.getUser(userId)
	if err != nil {
		return //TODO: error handle
	}

	if user.currentState == readyForQuestion {
		q.sendMessage(userId, "Nothing to cancel")
		return
	}

	err = q.crud.updateUserState(userId, readyForQuestion)
	if err != nil {
		return //TODO: Error handle
	}

	q.sendMessage(userId, "Done")
}

func (q *quiz) ProcessMessage(userId int, text, documentUrl string) {

	u, err := q.crud.getUser(userId)
	if err != nil {
		return //TODO: error handle
	}

	log.Printf("process non route: curr state: %d\n", u.currentState)

	switch u.currentState {
	case awaitingUpload:
		q.tryToMigrate(userId, documentUrl)
	case readyForQuestion:
		q.ShowHelp(u.id)
	case waitingAnswer:
		q.guessWord(*u, text)
	case migrationInProgress:
		q.showMigrationInProgressWarn(userId)
	case awaitingLanguage:
		q.setLanguage(*u, text)
	}
}

func (q *quiz) Stopped() bool {
	select {
	case <-q.ctx.Done():
		return true
	default:
		return false
	}
}

func newQuiz(s messageSender) Quiz {
	var q Quiz = &quiz{sender: s}
	return q
}

func (q *quiz) guessWord(usr user, guess string) {

	go func(u user) {

		word, err := q.crud.getLastWord(u.id)
		if err != nil {
			q.sendMessage(u.id, err.Error())
			return
		}

		translated, err := translateWord(*word, u.currentLanguage)
		if err != nil {
			q.sendMessage(u.id, err.Error())
			return
		}

		err = q.crud.deleteLastWord(u.id)

		p := guessParams{*word, guess, u.id}
		r := guessResult{p, translated}

		q.tellResult(r)

		err = q.crud.persistAnswer(r)
		if err != nil {
			log.Printf("Failed to write answer: %v\n", err.Error())
		}

		err = q.crud.updateUserState(u.id, readyForQuestion)
		if err != nil {
			//TODO: what?
		}
	}(usr)

}

func (q *quiz) tryToMigrate(userId int, url string) {
	err := q.crud.downloadAndMigrateKindleSQLite(url, userId)
	if err != nil {
		q.sendMessage(userId, "Looks like db file in incorrect format. Try again.")
		return
	}

	q.sendMessage(userId, "Migration completed. Press /quiz to start a game.")
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

func (q *quiz) setLanguage(u user, lc string) {
	l, err := q.crud.getLanguageWithCode(lc)
	if err != nil {
		q.sendMessage(u.id, "Invalid language code")
		return
	}

	err = q.crud.updateUserLang(u.id, l.id)
	if err != nil {
		return //TODO
	}

	q.sendMessage(u.id, fmt.Sprintf("Language changed to: %s", l.localized_name))
}

func (q *quiz) showMigrationInProgressWarn(userId int) {
	q.sendMessage(userId, "Migration still in progress.")
}

func (q *quiz) connectToDB() error {

	c := crud{}
	p := connectionParams{user: "postgres", dbName: "vocab", port: 32770, sslMode: "disable"}
	err := c.connect(p)
	if err != nil {
		return err
	}

	q.crud = &c

	return nil
}

func (q *quiz) tellResult(r guessResult) {
	if r.correct() {
		q.sendMessage(r.params.userId, "Your answer is correct")
	} else {
		q.sendMessage(r.params.userId, fmt.Sprintf("Your answer is incorrect. Correct answer: %s\n", r.translation))
	}
}

func (q *quiz) ask(r guessRequest) {
	w := r.word
	question := fmt.Sprintf("Word is: %s; Stem: %s; Lang: %s\n", w.word, w.stem, w.lang.english_name)
	q.sendMessage(r.userId, question)
}

func (t *guessResult) correct() bool {
	return compareWords(t.params.guess, t.translation)
}

func (q *quiz) sendMessage(userId int, text string) {
	err := q.sender.SendMessage(userId, text)
	if err != nil {
		//TODO: Error handle
	}
}
