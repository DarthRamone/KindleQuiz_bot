package kindle_quiz_bot

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

func migrateFromKindleSQLite(sqlitePath string, userId int, repo *repository) error {
	SQLiteDB, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		return fmt.Errorf("db migration: %v", err.Error())
	}
	defer SQLiteDB.Close()

	langs, err := repo.getLanguages()
	if err != nil {
		return fmt.Errorf("get languages: %v", err.Error())
	}
	langMap := make(map[string]int, len(langs))
	for _, l := range langs {
		langMap[l.code] = l.id
	}

	rows, err := SQLiteDB.Query("SELECT word, stem, lang FROM WORDS")
	if err != nil {
		return fmt.Errorf("sqlite: querying words: %v", err.Error())
	}
	defer func() {
		err = rows.Close()
		if err != nil {
			//TODO: error handle
		}
	}()

	for rows.Next() {
		err := rows.Err()
		if err != nil {
			return err
		}

		var lc string
		w := word{}
		err = rows.Scan(&w.word, &w.stem, &lc)
		if err != nil {
			return fmt.Errorf("migration: scan word: %v", err.Error())
		}

		err = repo.addWordForUser(userId, w, lc)
		if err != nil {
			return fmt.Errorf("migration: add word: %v", err.Error())
		}
	}

	return nil
}


