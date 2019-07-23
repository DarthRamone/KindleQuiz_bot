package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

var db *sql.DB

func main() {

	var err error

	connStr := "user=postgres dbname=vocab port=32768 sslmode=disable"

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Couldn't connect to database: %v", err.Error())
	}
	defer db.Close()

	err = migrateToLocalDB("vocab.db", 0)
	if err != nil {
		log.Fatalf(err.Error())
	}
}
