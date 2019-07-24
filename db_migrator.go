package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

func downloadAndMigrateKindleSQLite(url string, userId int) error {
	path := strconv.Itoa(userId) + "_vocab.db"

	err := updateUserState(userId, migrationInProgress)
	if err != nil {
		return fmt.Errorf("downloading document: %v", err.Error())
	}

	err = downloadFile(path, url)
	if err != nil {
		return fmt.Errorf("downloading document: %v", err.Error())
	}
	defer func() {
		err = os.Remove(path)
		if err != nil {
			log.Printf("downloading document: %v", err.Error())
		}
	}()

	err = migrateFromKindleSQLite(path, userId)
	if err != nil {
		return fmt.Errorf("downloading document: %v", err.Error())
	}

	err = updateUserState(userId, readyForQuestion)
	if err != nil {
		return fmt.Errorf("downloading document: %v", err.Error())
	}

	return nil
}

func migrateFromKindleSQLite(sqlitePath string, userId int) error {
	sqliteDB, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		return fmt.Errorf("db migration: %v", err.Error())
	}
	defer sqliteDB.Close()

	log.Println("get languages")
	langs, err := getLanguages()
	if err != nil {
		return fmt.Errorf("get languages: %v", err.Error())
	}
	langMap := make(map[string]int, len(langs))
	for _, l := range langs {
		langMap[l.code] = l.id
	}

	log.Println("Query words")
	rows, err := sqliteDB.Query("SELECT word, stem, lang FROM WORDS")
	if err != nil {
		return fmt.Errorf("sqlite: querying words: %v", err.Error())
	}
	defer rows.Close()

	for rows.Next() {

		log.Println("Words iter")

		var lc string
		w := word{}
		err = rows.Scan(&w.word, &w.stem, &lc)
		if err != nil {
			fmt.Printf("sqlite: scanning word row: %v\n", err.Error())
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("postgre tx begin: %v\n", err.Error())
		}

		//Trying to insert word with ignoring duplicate keys

		langId, ok := langMap[lc]
		if !ok {
			fmt.Printf("No such lang code found: %s", lc)
			continue
		}

		log.Println("Insert word")

		var wordId int
		err = tx.QueryRow(""+
			"INSERT INTO words (word, stem, lang) "+
			"VALUES ($1, $2, $3) "+
			"ON CONFLICT (word, stem, lang) DO UPDATE SET word=$1 RETURNING id", w.word, w.stem, langId).Scan(&wordId)

		if err != nil {
			log.Printf("insertion word: %v\n", err.Error())
			log.Println("Query existing word id")
			//If error happened, probably word with same key was already added, trying to get id
			err = tx.QueryRow(""+
				"SELECT id "+
				"FROM words "+
				"WHERE word=$1 AND stem=$2 AND lang=$3", w.word, w.stem, w.lang).Scan(&wordId)

			if err != nil {
				fmt.Printf(err.Error())
				continue
			}
		}

		log.Println("Insert user")
		_, err = tx.Exec(""+
			"INSERT INTO users (id) "+
			"VALUES ($1) "+
			"ON CONFLICT DO NOTHING", userId)
		if err != nil {
			return fmt.Errorf("postgre: inserting user: %v", err.Error())
		}

		log.Println("Insert user word")
		_, err = tx.Exec(""+
			"INSERT INTO user_words (user_id, word_id) "+
			"VALUES ($1, $2) "+
			"ON CONFLICT DO NOTHING", userId, wordId)
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

func downloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
