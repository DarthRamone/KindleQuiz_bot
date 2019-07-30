package kindle_quiz_bot

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func migrateFromKindleSQLite(sqlitePath string, userId int, crud *crud) error {
	SQLiteDB, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		return fmt.Errorf("db migration: %v", err.Error())
	}
	defer SQLiteDB.Close()

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
	rows, err := SQLiteDB.Query("SELECT word, stem, lang FROM WORDS")
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


