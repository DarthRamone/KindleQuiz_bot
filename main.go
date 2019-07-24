package main

import (
	"bufio"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"os"
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

	user, err := getUser(0)
	if err != nil {
		log.Fatalf(err.Error())
	}

	quizStartListen()

	requestWord(*user)

	time.Sleep(time.Second * 2)

	reader := bufio.NewScanner(os.Stdin)

	reader.Scan()

	text := reader.Text()

	guessWord(*user, text)

	time.Sleep(time.Second * 5)

	println(text)
}
