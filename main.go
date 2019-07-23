package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"github.com/DarthRamone/gtranslate"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
	"strings"
	"time"
)

var db *sql.DB

func main() {

	var err error

	connStr := "user=postgres dbname=vocab port=32770 sslmode=disable"

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Couldn't connect to database: %v", err.Error())
	}
	defer db.Close()

	words := make(chan word)
	go func() {
		for i := 0; i < 20; i++ {
			word, err := getRandomWord(0)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			words <- *word
		}
	}()

	user, err := getUser(0)
	if err != nil {
		log.Fatalf(err.Error())
	}

	reader := bufio.NewScanner(os.Stdin)
	for w := range words {
		fmt.Printf("Word is: %s; Lang: %s\n", w.word, w.lang.english_name)
		if reader.Scan() {
			text := reader.Text()

			log.Printf(user.currentLanguage.code)

			param := &guessParam{w, text, user}

			res, err := guessWord(param)
			if err != nil {
				log.Printf("Translation error: %v\n", err.Error())
			}

			if res.correct() {
				fmt.Println("Your answer is correct")
			} else {
				fmt.Printf("Your answer is incorrect; Correct answer is: %s\n", res.translation)
			}

			err = writeAnswer(param, res)

			if err != nil {
				log.Printf(err.Error())
			}
		}
	}
}

type guessResult struct {
	guess       string
	translation string
}

func (t *guessResult) correct() bool {
	return compareWords(t.guess, t.translation)
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

type guessParam struct {
	word  word
	guess string
	user  *user
}

func guessWord(p *guessParam) (*guessResult, error) {

	translated, err := translateWord(p.word, p.user.currentLanguage)
	if err != nil {
		return nil, err
	}

	res := guessResult{p.guess, translated}

	return &res, nil
}

func compareWords(w1, w2 string) bool {

	s1 := strings.ToLower(strings.Trim(w1, " "))
	s2 := strings.ToLower(strings.Trim(w2, " "))

	return s1 == s2
}
