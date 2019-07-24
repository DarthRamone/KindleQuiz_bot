package main

import (
	"bufio"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
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

			param := &guessParams{w, text, user}

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
