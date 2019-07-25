package main

import (
	"database/sql"
	"fmt"
	"github.com/DarthRamone/gtranslate"
	"io"
	"log"
	"strings"
	"time"
)

type messageSender interface {
	sendMessage(userId int, text string) error
}

type quizPlayer interface {
	io.Writer
	ask(w word)
	tellResult(r guessResult)
	reportError(err error)
}

func (u user) ask(w word) {
	_, _ = fmt.Fprintf(u, "Word is: %s; Stem: %s; Lang: %s\n", w.word, w.stem, w.lang.english_name)
}

func (u user) tellResult(r guessResult) {

	if r.correct() {
		_, _ = fmt.Fprintf(u, "Your answer is correct;\n")
	} else {
		_, _ = fmt.Fprintf(u, "Your answer is incorrect. Correct answer: %s\n", r.translation)
	}

}

func (u user) reportError(err error) {
	_, _ = fmt.Fprintf(u, err.Error())
}

func (u user) Write(p []byte) (int, error) {
	chatId := int64(u.id)
	log.Printf("chatId: %d\n", chatId)

	err := sender.sendMessage(u.id, string(p))

	if err != nil {
		return 0, err
	}

	return len(p), nil
}

type asker interface {
	ask()
}

type guessRequest struct {
	user quizPlayer
	word word
}

func (r guessRequest) ask() {
	r.user.ask(r.word)
}

type guessParams struct {
	word  word
	guess string
	user  user
}

type resultTeller interface {
	tellResult()
}

type guessResult struct {
	params      guessParams
	translation string
}

func (g guessResult) tellResult() {
	g.params.user.tellResult(g)
}

func (t *guessResult) correct() bool {
	return compareWords(t.params.guess, t.translation)
}

var sender messageSender

var results = make(chan resultTeller)
var requests = make(chan asker)

func quizStartListen(s messageSender) {

	sender = s

	go func() {
		for {
			select {
			case req := <-requests:
				go func() {
					req.ask()
				}()
			case res := <-results:
				go func() {
					res.tellResult()
				}()
			}
		}
	}()
}

func requestWord(u user) {
	go func(p quizPlayer) {

		log.Println("request word")

		w, err := getRandomWord(u.id)

		if err == sql.ErrNoRows {
			u.reportError(fmt.Errorf("No words found. Please run /upload and follow instructions"))
			return
		}

		if err != nil {
			log.Println("report error: random word")
			u.reportError(err)
			return
		}

		err = setLastWord(u.id, *w)
		if err != nil {
			log.Println("report error: set last word")
			u.reportError(err)
			return
		}

		log.Println("send request")
		r := guessRequest{p, *w}
		requests <- r

		err = updateUserState(u.id, waitingAnswer)
		if err != nil {
			//TODO: what?
		}
	}(u)
}

func guessWord(u user, guess string) {
	go func() {

		word, err := getLastWord(u.id)
		if err != nil {
			u.reportError(err)
			return
		}

		translated, err := translateWord(*word, u.currentLanguage)
		if err != nil {
			u.reportError(err)
			return
		}

		err = deleteLastWord(u.id)

		p := guessParams{*word, guess, u}
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
	}()
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
