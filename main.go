package main

import (
	"database/sql"
	"github.com/bregydoc/gtranslate"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strings"
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

	//words := make(chan word)
	//go func() {
	//	for i := 0; i < 20; i++ {
	//		word, err := getRandomWord(0)
	//		if err != nil {
	//			log.Println(err.Error())
	//			continue
	//		}
	//		words <- *word
	//	}
	//}()
	//
	//user, err := getUser(0)
	//if err != nil {
	//	log.Fatalf(err.Error())
	//}
	//
	//reader := bufio.NewScanner(os.Stdin)
	//for w := range words {
	//	fmt.Printf("Word is: %s; Lang: %s\nAnswer:", w.word, w.lang)
	//	if reader.Scan() {
	//		text := reader.Text()
	//
	//		ok, err := guessWord(w, text, *user.currentLanguage)
	//		if err != nil {
	//			log.Printf("Translation error: %v\n", err.Error())
	//		}
	//
	//		if ok {
	//			fmt.Println("Your answer is correct")
	//		} else {
	//			fmt.Println("Your answer is incorrect")
	//		}
	//	}
	//}
}

func translateWord(w word, dst lang) (translated string, err error) {

	translated, err = gtranslate.TranslateWithFromTo(
		w.word,
		gtranslate.FromTo{
			From: w.lang.code,
			To:   dst.code,
		},
	)
	if err != nil {
		return "", err
	}

	return
}

func guessWord(w word, guess string, dst lang) (bool, error) {

	translated, err := translateWord(w, dst)
	if err != nil {
		return false, err
	}

	return compareWords(guess, translated), nil
}

func compareWords(w1, w2 string) bool {
	s1 := strings.ToLower(strings.Trim(w1, " "))
	s2 := strings.ToLower(strings.Trim(w2, " "))

	return s1 == s2
}
