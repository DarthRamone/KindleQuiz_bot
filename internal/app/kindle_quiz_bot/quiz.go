package kindle_quiz_bot

import (
	"fmt"
	"github.com/bregydoc/gtranslate"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	maxMigrationWorkersCount = 3
	maxDownloadJobsCount     = 3
)

type Quiz interface {
	Close()
	Greetings(userId int)
	ShowHelp(userId int)
	RequestWord(userId int)
	SelectLang(userId int)
	AwaitUpload(userId int)
	CancelOperation(userId int)
	ProcessMessage(userId int, text, documentUrl string)
}

type quiz struct {
	repo          *repository
	sender        MessageSender
	downloadJobs  chan downloadJob
	migrationJobs chan migrationJob
}

type guessRequest struct {
	userId int
	word   word
}

type guessParams struct {
	word   word
	guess  string
	userID int
}

type guessResult struct {
	params      guessParams
	translation string
}

type word struct {
	id     int
	word   string
	stem   string
	langId int
}

type user struct {
	id           int
	currentState userState
}

type lang struct {
	id            int
	code          string
	englishName   string
	localizedName string
}

type downloadJob struct {
	userId      int
	documentUrl string
}

type migrationJob struct {
	downloadJob
	documentPath string
}

type MessageSender interface {
	SendMessage(userId int, text string) error
}

func (q *quiz) Close() {
	q.repo.close()
	close(q.migrationJobs)
}

func NewQuiz(s MessageSender) Quiz {
	q := quiz{sender: s}

	err := q.connectToDB()
	if err != nil {
		log.Fatalf("db connect: %v", err.Error())
	}

	q.downloadJobs = make(chan downloadJob, 20)
	q.migrationJobs = make(chan migrationJob, 20)

	for i := 0; i < maxDownloadJobsCount; i++ {
		go q.downloadWorker(q.downloadJobs)
	}

	for i := 0; i < maxMigrationWorkersCount; i++ {
		go q.migrationWorker(q.migrationJobs)
	}

	return &q
}

func (q *quiz) RequestWord(userId int) {
	log.Println("request word")

	w, err := q.repo.getRandomWord(userId)

	if err == errNoWordsFound {
		q.sendMessage(userId, "No words found. Please run /upload and follow instructions")
		return
	}

	if err != nil {
		log.Println("report error: random word")
		q.sendMessage(userId, err.Error())
		return
	}

	log.Println("send request")
	r := guessRequest{userId, *w}
	q.ask(r)
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

	_, err := q.repo.createUser(userId)
	if err != nil {
		//TODO: error handle
	} else {
		msg := `Yo. Firstly you have to run /upload and upload your vocab.db file. 
Next run /quiz and have some fun, idk. You can ask me for /help also.`
		q.sendMessage(userId, msg)
	}
}

func (q *quiz) SelectLang(userId int) {
	langs, err := q.repo.getLanguages()
	if err != nil {
		log.Printf("Couldn't update user state: %v", err)
		return
	}

	msg := "Select language code:\n\n"
	for _, l := range langs {
		msg += fmt.Sprintf("[%s] %s\n", l.code, l.englishName)
	}

	err = q.repo.updateUserState(userId, awaitingLanguage)
	if err != nil {
		log.Printf("Couldn't update user state: %v", err)
	}

	q.sendMessage(userId, msg)
}

func (q *quiz) AwaitUpload(userId int) {
	err := q.repo.updateUserState(userId, awaitingUpload)
	if err != nil {
		log.Printf("await upload: %v", err)
		return //TODO: Error handle
	}

	q.sendMessage(userId, "Now send vocab.db file exported from your kindle")
}

func (q *quiz) CancelOperation(userId int) {
	user, err := q.repo.getUser(userId)
	if err != nil {
		log.Printf("await upload: %v", err)
		return //TODO: error handle
	}

	if user.currentState == readyForQuestion {
		q.sendMessage(userId, "Nothing to cancel")
		return
	}

	err = q.repo.updateUserState(userId, readyForQuestion)
	if err != nil {
		log.Printf("await upload: %v", err)
		return //TODO: Error handle
	}

	q.sendMessage(userId, "Done")
}

func (q *quiz) ProcessMessage(userId int, text, documentUrl string) {
	u, err := q.repo.getUser(userId)
	if err != nil {
		log.Printf("await upload: %v", err)
		return //TODO: error handle
	}

	log.Printf("process non route: curr state: %d\n", u.currentState)

	switch u.currentState {
	case awaitingUpload:
		q.downloadJobs <- downloadJob{userId, documentUrl}
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

func (q *quiz) guessWord(u user, guess string) {
	word, err := q.repo.getLastWord(u.id)
	if err != nil {
		q.sendMessage(u.id, err.Error())
		return
	}

	lang, err := q.repo.getUserLanguage(u.id)
	if err != nil {
		q.sendMessage(u.id, err.Error())
		return
	}

	translated, err := q.translateWord(*word, lang)
	if err != nil {
		q.sendMessage(u.id, err.Error())
		return
	}

	err = q.repo.deleteLastWord(u.id)
	if err != nil {
		q.sendMessage(u.id, err.Error())
		return
	}

	p := guessParams{*word, guess, u.id}
	r := guessResult{p, translated}

	q.tellResult(r)

	err = q.repo.persistAnswer(r)
	if err != nil {
		log.Printf("Failed to write answer: %v\n", err.Error())
	}

	err = q.repo.updateUserState(u.id, readyForQuestion)
	if err != nil {
		log.Printf("Couldn't update user state: %v", err)
	}
}

func (q *quiz) tryToMigrate(userId int, path string) error {
	err := q.repo.updateUserState(userId, migrationInProgress)
	if err != nil {
		return fmt.Errorf("migrate: update state: %v", err.Error())
	}

	err = migrateFromKindleSQLite(path, userId, q.repo)
	if err != nil {
		q.sendMessage(userId, "Looks like db file in incorrect format. Try again.")
		return nil
	}

	err = q.repo.updateUserState(userId, readyForQuestion)
	if err != nil {
		return fmt.Errorf("downloading document: %v", err.Error())
	}

	return nil
}

func(q *quiz) translateWord(w word, dst *lang) (string, error) {
	lang, err := q.repo.getLang(w.langId)
	if err != nil {
		return "", err
	}

	translated, err := gtranslate.TranslateWithParams(
		w.word,
		gtranslate.TranslationParams{
			From:  lang.code,
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
	l, err := q.repo.getLanguageWithCode(lc)
	if err != nil {
		q.sendMessage(u.id, "Invalid language code")
		return
	}

	err = q.repo.updateUserLang(u.id, l.id)
	if err != nil {
		return //TODO
	}

	q.sendMessage(u.id, fmt.Sprintf("Language changed to: %s", l.localizedName))
}

func (q *quiz) showMigrationInProgressWarn(userId int) {
	q.sendMessage(userId, "Migration still in progress.")
}

func (q *quiz) connectToDB() error {
	c := repository{}
	p := connectionParams{user: "postgres", dbName: "vocab", port: 5432, sslMode: "disable", url: "postgres"}
	err := c.connect(p)
	if err != nil {
		return err
	}

	q.repo = &c

	return nil
}

func (q *quiz) tellResult(r guessResult) {
	if r.correct() {
		q.sendMessage(r.params.userID, "Your answer is correct")
	} else {
		q.sendMessage(r.params.userID, fmt.Sprintf("Your answer is incorrect. Correct answer: %s\n", r.translation))
	}
}

func (q *quiz) ask(r guessRequest) {

	lang, err := q.repo.getLang(r.word.langId)
	if err != nil {
		q.sendMessage(r.userId, err.Error())
		return
	}

	w := r.word
	question := fmt.Sprintf("Word is: %s; Stem: %s; Lang: %s\n", w.word, w.stem, lang.englishName)
	q.sendMessage(r.userId, question)
}

func (t *guessResult) correct() bool {
	return compareWords(t.params.guess, t.translation)
}

func (q *quiz) sendMessage(userId int, text string) {
	err := q.sender.SendMessage(userId, text)
	if err != nil {
		log.Printf("Couldn't send message: %v", err)
	}
}

func (q *quiz) migrationWorker(jobs <-chan migrationJob) {
	for downloadJob := range jobs {
		func(job migrationJob) {
			userId := job.userId
			path := job.documentPath

			defer func() {
				err := os.Remove(path)
				if err != nil {
					log.Printf("downloading document: %v", err.Error())
				}
			}()

			q.sendMessage(userId, "Processing...")

			err := q.tryToMigrate(userId, path)
			if err != nil {
				q.sendMessage(userId, "migration failed")
				return
			}

			q.sendMessage(job.userId, "Migration completed. Press /quiz to start a game.")
		}(downloadJob)
	}
}

func (q *quiz) downloadWorker(jobs <-chan downloadJob) {
	for job := range jobs {
		userId := job.userId

		path := strconv.Itoa(userId) + "_vocab.db"

		err := downloadFile(path, job.documentUrl)
		if err != nil {
			//TODO: add retry policy maybe
			q.sendMessage(userId, "Document couldn't be downloaded")
		}

		q.migrationJobs <- migrationJob{job, path}
	}
}

func downloadFile(filepath string, url string) (err error) {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		//TODO: error handle
		_ = resp.Body.Close()
	}()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}

	defer func() {
		_ = out.Close()
	}()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
