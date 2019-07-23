package main

import (
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func main() {
	err := migrateToLocalDB("vocab.db", 0)
	if err != nil {
		log.Fatalf(err.Error())
	}
}
