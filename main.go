package main

import (
	"database/sql"
	"fmt"
	"github.com/bregydoc/gtranslate"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strings"
)

var db *sql.DB

func main() {

	var err error

	connStr := "user=postgres dbname=vocab port=32768 sslmode=disable"

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Couldn't connect to database: %v", err.Error())
	}

	word, err := getRandomWord(0)
	if err != nil {
		log.Fatalf(err.Error())
	}

	translated, err := translateWord(*word, "ru")
	if err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Printf("%s = %s \n", word.word, translated)
}

func getRandomWord(userId int) (*word, error) {
	var wordId int

	row := db.QueryRow("SELECT word_id FROM user_words WHERE user_id=$1 OFFSET floor(random() * (SELECT COUNT(*) FROM words)) LIMIT 1", userId)
	err := row.Scan(&wordId)
	if err != nil {
		fmt.Printf("shit")
		return nil, err
	}
	fmt.Printf("word_id: %d\n", wordId)

	wordRow := db.QueryRow("SELECT word, stem, lang FROM words WHERE id=$1", wordId)

	w := word{}
	err = wordRow.Scan(&w.word, &w.stem, &w.lang)
	if err != nil {
		return nil, fmt.Errorf("Random word row scan: %v", err.Error())
	}

	return &w, nil
}

func translateWord(w word, lc string) (translated string, err error) {

	translated, err = gtranslate.TranslateWithFromTo(
		w.word,
		gtranslate.FromTo{
			From: w.lang,
			To:   lc,
		},
	)
	if err != nil {
		return "", err
	}

	return
}

//func guessWord(w, answer word, sl, dl string) bool, error {
//
//}
