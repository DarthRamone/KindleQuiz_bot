package kindle_quiz_bot

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

func downloadAndMigrateKindleSQLite(url string, userId int, crud *crud) error {
	path := strconv.Itoa(userId) + "_vocab.db"

	err := downloadFile(path, url)
	if err != nil {
		return fmt.Errorf("downloading document: %v", err.Error())
	}
	defer func() {
		err = os.Remove(path)
		if err != nil {
			log.Printf("downloading document: %v", err.Error())
		}
	}()

	err = migrateFromKindleSQLite(path, userId, crud)
	if err != nil {
		return fmt.Errorf("downloading document: %v", err.Error())
	}

	return nil
}

func migrateFromKindleSQLite(sqlitePath string, userId int, crud *crud) error {
	sqliteDB, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		return fmt.Errorf("db migration: %v", err.Error())
	}
	defer sqliteDB.Close()

	log.Println("get languages")
	langs, err := crud.getLanguages()
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
		err := rows.Err()
		if err != nil {
			return err
		}

		log.Println("Words iter")

		var lc string
		w := word{}
		err = rows.Scan(&w.word, &w.stem, &lc)
		if err != nil {
			return fmt.Errorf("migration: scan word: %v", err.Error())
		}

		err = crud.addWordForUser(userId, w, lc)
		if err != nil {
			return fmt.Errorf("migration: add word: %v", err.Error())
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
