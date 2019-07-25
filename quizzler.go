package main

import (
	"database/sql"
	"fmt"
	"github.com/DarthRamone/gtranslate"
	"log"
	"strings"
	"time"
)

type messageSender interface {
	sendMessage(userId int, text string) error
}

func tellResult(r guessResult) {
	if r.correct() {
		_ = sender.sendMessage(r.params.userId, "Your answer is correct")
	} else {
		_ = sender.sendMessage(r.params.userId, fmt.Sprintf("Your answer is incorrect. Correct answer: %s\n", r.translation))
	}
}

func ask(r guessRequest) {
	w := r.word
	q := fmt.Sprintf("Word is: %s; Stem: %s; Lang: %s\n", w.word, w.stem, w.lang.english_name)
	_ = sender.sendMessage(r.userId, q) //TODO: error handle
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

func (t *guessResult) correct() bool {
	return compareWords(t.params.guess, t.translation)
}

var sender messageSender

var results = make(chan guessResult)
var requests = make(chan guessRequest)

func StartListen(s messageSender) {

	sender = s

	go func() {
		for {
			select {
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

func RequestWord(userId int) {
	go func(id int) {

		log.Println("request word")

		w, err := getRandomWord(userId)

		if err == sql.ErrNoRows {
			//TODO: error handle
			_ = sender.sendMessage(id, "No words found. Please run /upload and follow instructions")
			return
		}

		if err != nil {
			log.Println("report error: random word")
			//TODO Error handle
			_ = sender.sendMessage(id, err.Error())
			return
		}

		err = setLastWord(id, *w)
		if err != nil {
			log.Println("report error: set last word")
			//TODO Error handle
			_ = sender.sendMessage(id, err.Error())
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

func GuessWord(userId int, guess string) {
	go func(id int) {

		word, err := getLastWord(id)
		if err != nil {
			//TODO Error handle
			_ = sender.sendMessage(id, err.Error())
			return
		}

		lang, err := getUserLanguage(id)
		if err != nil {
			return //TODO: error handle
		}

		translated, err := translateWord(*word, lang)
		if err != nil {
			//TODO Error handle
			_ = sender.sendMessage(id, err.Error())
			return
		}

		err = deleteLastWord(id)

		p := guessParams{*word, guess, id}
		r := guessResult{p, translated}
		results <- r

		err = writeAnswer(r)
		if err != nil {
			log.Printf("Failed to write answer: %v\n", err.Error())
		}

		err = updateUserState(id, readyForQuestion)
		if err != nil {
			//TODO: what?
		}
	}(userId)
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
