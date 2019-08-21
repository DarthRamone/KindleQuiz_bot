package kindle_quiz_bot

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"log"
)

var (
	errNoWordsFound = errors.New("no words found for user")
)

type userState int

const (
	awaitingUpload userState = iota
	waitingAnswer
	readyForQuestion
	migrationInProgress
	awaitingLanguage
)

type repository struct {
	db *sql.DB
}

type connectionParams struct {
	user    string
	dbName  string
	port    int
	sslMode string
	url     string
}

func (repo *repository) connect(p connectionParams) error {
	connStr := fmt.Sprintf("user=%s dbname=%s port=%d sslmode=%s host=%s", p.user, p.dbName, p.port, p.sslMode, p.url)

	var err error
	repo.db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	return nil
}

func (repo *repository) close() {
	if repo.db == nil {
		return
	}

	err := repo.db.Close()

	if err != nil {
		log.Printf("DB close: %v", err)
	}
}

func (repo *repository) getUserLanguage(userID int) (*lang, error) {
	l := lang{}
	err := repo.db.QueryRow(`
		SELECT * FROM languages 
		WHERE id=(SELECT current_lang FROM users WHERE id=$1)`, userID).Scan(&l.id, &l.code, &l.englishName, &l.localizedName)

	if err != nil {
		return nil, err
	}

	return &l, nil
}

func (repo *repository) updateUserState(userID int, state userState) error {
	_, err := repo.db.Exec("UPDATE users SET current_state=$1 WHERE id=$2", state, userID)
	if err != nil {
		return err
	}
	return nil
}

func (repo *repository) updateUserLang(userID, langId int) error {
	_, err := repo.db.Exec("UPDATE users SET current_lang=$1 WHERE id=$2", langId, userID)
	if err != nil {
		return err
	}
	return nil
}

func (repo *repository) createUser(userID int) (*user, error) {
	u := user{userID, readyForQuestion}

	_, err := repo.db.Exec(`
		INSERT INTO users (id) 
		VALUES ($1)
		ON CONFLICT DO NOTHING`, userID)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (repo *repository) deleteLastWord(userID int) error {
	_, err := repo.db.Exec("DELETE FROM questions WHERE user_id=$1", userID)
	if err != nil {
		return err
	}
	return nil
}

func (repo *repository) setLastWord(userID int, w word) error {
	_, err := repo.db.Exec(`
		INSERT INTO questions (user_id, word_id) 
		VALUES ($1, $2) 
		ON CONFLICT (user_id) 
		    DO UPDATE SET word_id=$2`, userID, w.id)
	if err != nil {
		return err
	}
	return nil
}

func (repo *repository) getLastWord(userID int) (*word, error) {
	var wordID int
	err := repo.db.QueryRow("SELECT word_id FROM questions WHERE user_id=$1", userID).Scan(&wordID)
	if err != nil {
		return nil, err
	}

	return repo.getWord(wordID)
}

func (repo *repository) getRandomWord(userID int) (word *word, err error) {
	var wordID int

	tx, err := repo.db.Begin()

	if err != nil {
		return nil, err
	}

	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Printf("unable to rollback transaction: %v", rollbackErr)
			}
		}
	}()

	err = tx.QueryRow(`
		SELECT word_id 
		FROM user_words 
		WHERE user_id=$1 
		OFFSET floor(random() * (SELECT COUNT(*) FROM words)) LIMIT 1`, userID).Scan(&wordID)

	if err == sql.ErrNoRows {
		return nil, errNoWordsFound
	}

	_, err = tx.Exec(`
		INSERT INTO questions (user_id, word_id) 
		VALUES ($1, $2) 
		ON CONFLICT (user_id) 
		    DO UPDATE SET word_id=$2`, userID, wordID)

	if err != nil {
		return nil, err
	}

	_, err = tx.Exec("UPDATE users SET current_state=$1 WHERE id=$2", waitingAnswer, userID)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return repo.getWord(wordID)
}

func (repo *repository) getWord(wordID int) (*word, error) {
	w := word{}
	err := repo.db.QueryRow("SELECT word, stem, lang, id FROM words WHERE id=$1", wordID).Scan(&w.word, &w.stem, &w.langId, &w.id)

	if err != nil {
		return nil, fmt.Errorf("random word row scan: %v", err.Error())
	}

	return &w, nil
}

func (repo *repository) getUser(id int) (*user, error) {
	u := user{}
	var langId int
	err := repo.db.QueryRow("SELECT * FROM users WHERE id=$1", id).Scan(&u.id, &langId, &u.currentState)

	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (repo *repository) getLang(id int) (*lang, error) {
	l := lang{}
	err := repo.db.QueryRow("SELECT * FROM languages WHERE id=$1", id).Scan(&l.id, &l.code, &l.englishName, &l.localizedName)
	if err != nil {
		return nil, fmt.Errorf("get lang: %v", err.Error())
	}
	return &l, nil
}

func (repo *repository) getLanguages() ([]lang, error) {
	langs := make([]lang, 0)

	rows, err := repo.db.Query("SELECT * FROM languages")
	if err != nil {
		return nil, err
	}
	defer func() {
		//TODO: error handle
		_ = rows.Close()
	}()

	for rows.Next() {

		err := rows.Err()
		if err != nil {
			return nil, err
		}

		l := lang{}
		err = rows.Scan(&l.id, &l.code, &l.englishName, &l.localizedName)
		if err != nil {
			return nil, fmt.Errorf("get lang: %v", err.Error())
		}
		langs = append(langs, l)
	}

	return langs, nil
}

func (repo *repository) getLanguageWithCode(code string) (*lang, error) {
	l := lang{}
	err := repo.db.QueryRow("SELECT * FROM languages WHERE code=$1", code).Scan(&l.id, &l.code, &l.englishName, &l.localizedName)
	if err != nil {
		return nil, fmt.Errorf("lang with code: %v", err.Error())
	}
	return &l, nil
}

func (repo *repository) persistAnswer(r guessResult) error {
	p := r.params

	tx, err := repo.db.Begin()
	if err != nil {
		return err
	}

	lang, err := repo.getUserLanguage(r.params.userID)
	if err != nil {
		//TODO: error handling
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO answers (word_id, user_id, correct, user_lang, guess) 
		VALUES ($1, $2, $3, $4, $5)`, p.word.id, p.userID, r.correct(), lang.id, p.guess)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	var field string
	if r.correct() {
		field = "correct_answers"
	} else {
		field = "incorrect_answers"
	}

	var queryStr = fmt.Sprintf("UPDATE user_words SET %s = %[1]s + 1 WHERE word_id = %d", field, p.word.id)

	_, err = tx.Exec(queryStr)

	if err != nil {
		//TODO: error handling
		_ = tx.Rollback()
		return fmt.Errorf("write answer: %s", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (repo *repository) addWordForUser(userID int, word word, lc string) (err error){
	tx, err := repo.db.Begin()
	if err != nil {
		return fmt.Errorf("postgres tx begin: %v", err.Error())
	}

	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Printf("Unable to rollback tx: %v", rollbackErr)
			}
		}
	}()

	var langId int
	err = tx.QueryRow(`
			SELECT current_lang 
			FROM users
			WHERE id=$1`, userID).Scan(&langId)
	if err != nil {
		return err
	}

	var wordID int
	err = tx.QueryRow(`
		INSERT INTO words (word, stem, lang)
		VALUES ($1, $2, $3)
		ON CONFLICT (word, stem, lang) 
		    DO UPDATE SET word=$1 RETURNING id`, word.word, word.stem, langId).Scan(&wordID)

	if err != nil {
		log.Printf("insertion word: %v\n", err)
		log.Println("Query existing word id")
		//If error happened, probably word with same key was already added, trying to get id
		err = tx.QueryRow(`
			SELECT id
			FROM words
			WHERE word=$1 AND stem=$2 AND lang=$3`, word.word, word.stem, word.langId).Scan(&wordID)

		if err != nil {
			fmt.Printf("add words for user: %v", err)
			return err
		}
	}

	_, err = tx.Exec(`
		INSERT INTO user_words (user_id, word_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, word_id) 
		    DO NOTHING`, userID, wordID)

	if err != nil {
		return fmt.Errorf("postgre: inserting user_word: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("postgre: tx commit: %v", err.Error())
	}

	return nil
}
