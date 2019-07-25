package kindle_quiz_bot

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
)

const (
	defaultLanguageId = 2 //English
)

const (
	awaitingUpload = iota
	waitingAnswer
	readyForQuestion
	migrationInProgress
	awaitingLanguage
)

type crud struct {
	db *sql.DB
}

type connectionParams struct {
	user     string
	password string
	dbName   string
	port     int
	sslMode  string
}

func (crud *crud) connect(p connectionParams) error {

	connStr := fmt.Sprintf("user=%s dbname=%s port=%d sslmode=%s", p.user, p.dbName, p.port, p.sslMode)

	var err error
	crud.db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	return nil
}

func (crud *crud) close() {
	crud.db.Close()
}

func (crud *crud) getUserLanguage(userId int) (*lang, error) {
	l := lang{}
	err := crud.db.QueryRow(""+
		"SELECT * FROM languages "+
		"WHERE id="+
		"(SELECT current_lang FROM users WHERE id=$1)", userId).Scan(&l.id, &l.code, &l.english_name, &l.localized_name)
	if err != nil {
		return nil, err
	}

	return &l, nil
}

func (crud *crud) getAllUserIds() ([]int, error) {
	var count int
	err := crud.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return nil, err
	}

	rows, err := crud.db.Query("SELECT id FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]int, 0, count)
	for rows.Next() {
		var id int
		err := rows.Scan(&id)
		if err != nil {
			continue
		}

		res = append(res, id)
	}

	return res, nil
}

func (crud *crud) updateUserState(userId, state int) error {
	_, err := crud.db.Exec("UPDATE users SET current_state=$1 WHERE id=$2", state, userId)
	if err != nil {
		return err
	}
	return nil
}

func (crud *crud) updateUserLang(userId, langId int) error {
	_, err := crud.db.Exec("UPDATE users SET current_lang=$1 WHERE id=$2", langId, userId)
	if err != nil {
		return err
	}
	return nil
}

func (crud *crud) createUser(userId int) (*user, error) {

	lang, err := crud.getLang(defaultLanguageId)
	if err != nil {
		return nil, err
	}

	u := user{userId, readyForQuestion, lang}

	_, err = crud.db.Exec("INSERT INTO users (id, current_lang) VALUES ($1, $2) ON CONFLICT DO NOTHING", userId, defaultLanguageId)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (crud *crud) deleteLastWord(userId int) error {
	_, err := crud.db.Exec("DELETE FROM questions WHERE user_id=$1", userId)
	if err != nil {
		return err
	}
	return nil
}

func (crud *crud) setLastWord(userId int, w word) error {
	_, err := crud.db.Exec("INSERT INTO questions (user_id, word_id) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET word_id=$2", userId, w.id)
	if err != nil {
		return err
	}
	return nil
}

func (crud *crud) getLastWord(userId int) (*word, error) {
	row := crud.db.QueryRow("SELECT word_id FROM questions WHERE user_id=$1", userId)
	var wordId int
	err := row.Scan(&wordId)
	if err != nil {
		return nil, err
	}

	return crud.getWord(wordId)
}

func (crud *crud) getRandomWord(userId int) (*word, error) {
	var wordId int

	row := crud.db.QueryRow("SELECT word_id FROM user_words WHERE user_id=$1 OFFSET floor(random() * (SELECT COUNT(*) FROM words)) LIMIT 1", userId)
	err := row.Scan(&wordId)
	if err != nil {
		fmt.Printf("shit")
		return nil, err
	}

	return crud.getWord(wordId)
}

func (crud *crud) getWord(wordId int) (*word, error) {
	wordRow := crud.db.QueryRow("SELECT word, stem, lang, id FROM words WHERE id=$1", wordId)

	w := word{}
	var langId int
	err := wordRow.Scan(&w.word, &w.stem, &langId, &w.id)
	if err != nil {
		return nil, fmt.Errorf("Random word row scan: %v", err.Error())
	}

	l, err := crud.getLang(langId)
	if err != nil {
		return nil, err
	}
	w.lang = l

	return &w, nil
}

func (crud *crud) getUser(id int) (*user, error) {
	userRow := crud.db.QueryRow("SELECT * FROM users WHERE id=$1", id)
	u := user{}
	var langId int
	err := userRow.Scan(&u.id, &langId, &u.currentState)

	if err != nil {
		return nil, err
	}

	l, err := crud.getLang(langId)
	if err != nil {
		return nil, err
	}

	u.currentLanguage = l

	return &u, nil
}

func (crud *crud) getLang(id int) (*lang, error) {
	langRow := crud.db.QueryRow("SELECT * FROM languages WHERE id=$1", id)
	l := lang{}
	err := langRow.Scan(&l.id, &l.code, &l.english_name, &l.localized_name)
	if err != nil {
		return nil, fmt.Errorf("Get lang: %v", err.Error())
	}
	return &l, nil
}

func (crud *crud) getLanguages() ([]lang, error) {

	log.Println("query langs count")
	row := crud.db.QueryRow("SELECT COUNT(*) FROM languages")
	var count int
	err := row.Scan(&count)
	if err != nil {
		return nil, err
	}

	langs := make([]lang, 0, count)

	log.Println("query languages")
	rows, err := crud.db.Query("SELECT * FROM languages")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		log.Println("lang iter")
		l := lang{}
		err := rows.Scan(&l.id, &l.code, &l.english_name, &l.localized_name)
		if err != nil {
			fmt.Printf("Get lang: %v", err.Error())
			continue
		}
		langs = append(langs, l)
	}

	return langs, nil
}

func (crud *crud) getLanguageWithCode(code string) (*lang, error) {
	l := lang{}
	err := crud.db.QueryRow("SELECT * FROM languages WHERE code=$1", code).Scan(&l.id, &l.code, &l.english_name, &l.localized_name)
	if err != nil {
		return nil, fmt.Errorf("lang with code: %v", err.Error())
	}
	return &l, nil
}

func (crud *crud) persistAnswer(r guessResult) error {

	p := r.params

	tx, err := crud.db.Begin()
	if err != nil {
		return err
	}

	lang, err := crud.getUserLanguage(r.params.userId)
	if err != nil {
		//TODO: error handling
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec(""+
		"INSERT INTO answers (word_id, user_id, correct, user_lang, guess)"+
		"VALUES ($1, $2, $3, $4, $5)", p.word.id, p.userId, r.correct(), lang.id, p.guess)

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
		return nil
	}

	return nil
}
