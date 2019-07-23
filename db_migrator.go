package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func migrateToLocalDB(sqlitePath string, userId int) error {
	sqliteDB, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		return fmt.Errorf("db migration: %v", err.Error())
	}
	defer sqliteDB.Close()

	var wordsCount int
	count := sqliteDB.QueryRow("SELECT COUNT(*) FROM WORDS")
	err = count.Scan(&wordsCount)
	if err != nil {
		return fmt.Errorf("sqlite: words count: %v", err.Error())
	}

	rows, err := sqliteDB.Query("SELECT word, stem, lang FROM WORDS")
	if err != nil {
		return fmt.Errorf("sqlite: querying words: %v", err.Error())
	}

	connStr := "user=postgres dbname=vocab port=32768 sslmode=disable"
	postgreDB, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("open postgre connection: %v", err.Error())
	}
	defer postgreDB.Close()

	for rows.Next() {

		tx, err := postgreDB.Begin()
		if err != nil {
			return fmt.Errorf("postgre tx begin: %v", err.Error())
		}

		w := word{}
		err = rows.Scan(&w.word, &w.stem, &w.lang)
		if err != nil {
			fmt.Errorf("sqlite: scanning word row: %v", err.Error())
			continue
		}

		//Trying to insert word with ignoring duplicate keys
		row := tx.QueryRow("INSERT INTO words (word, stem, lang) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING RETURNING id", w.word, w.stem, w.lang)
		var wordId int
		err = row.Scan(&wordId)
		if err != nil {
			//If error happened, probably word with same key was already added, trying to get id
			row = tx.QueryRow("SELECT id FROM words WHERE word=$1 AND stem=$2 AND lang=$3", w.word, w.stem, w.lang)
			err = row.Scan(&wordId)
			if err != nil {
				fmt.Printf(err.Error())
				continue
			}
		}

		_, err = tx.Exec("INSERT INTO users (id) VALUES ($1) ON CONFLICT DO NOTHING", userId)
		if err != nil {
			return fmt.Errorf("postgre: inserting user: %v", err.Error())
		}

		_, err = tx.Exec("INSERT INTO user_words (user_id, word_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", userId, wordId)
		if err != nil {
			return fmt.Errorf("postgre: inserting user_word: %v", err.Error())
		}

		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("postgre: tx commit: %v", err.Error())
		}
	}

	return nil
}
