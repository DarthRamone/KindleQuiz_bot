package kindle_quiz_bot

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"log"
)

const (
	defaultLanguageId = 2 //English
)

var (
	noWordsFound = errors.New("No words found for user")
	langMap      map[string]int
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
	user     string
	password string
	dbName   string
	port     int
	sslMode  string
	url      string
}

func (repo *repository) connect(p connectionParams) error {

	connStr := fmt.Sprintf("user=%s dbname=%s port=%d sslmode=%s host=%s", p.user, p.dbName, p.port, p.sslMode, p.url)

	log.Printf("DB connection string: %s", connStr)

	var err error
	repo.db, err = sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	log.Println("get languages")
	langs, err := repo.getLanguages()
	if err != nil {
		return fmt.Errorf("get languages: %v", err.Error())
	}
	langMap = make(map[string]int, len(langs))
	for _, l := range langs {
		langMap[l.code] = l.id
	}

	return nil
}

func (repo *repository) close() {
	repo.db.Close()
}

func (repo *repository) getUserLanguage(userId int) (*lang, error) {
	l := lang{}
	err := repo.db.QueryRow(`
		SELECT * FROM languages 
		WHERE id=(SELECT current_lang FROM users WHERE id=$1)`, userId).Scan(&l.id, &l.code, &l.englishName, &l.localizedName)
	if err != nil {
		return nil, err
	}

	return &l, nil
}

func (repo *repository) getAllUserIds() ([]int, error) {
	var count int
	err := repo.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return nil, err
	}

	rows, err := repo.db.Query("SELECT id FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := make([]int, 0, count)
	for rows.Next() {

		err := rows.Err()
		if err != nil {
			return nil, err
		}

		var id int
		err = rows.Scan(&id)
		if err != nil {
			continue
		}

		res = append(res, id)
	}

	return res, nil
}

func (repo *repository) updateUserState(userId int, state userState) error {
	_, err := repo.db.Exec("UPDATE users SET current_state=$1 WHERE id=$2", state, userId)
	if err != nil {
		return err
	}
	return nil
}

func (repo *repository) updateUserLang(userId, langId int) error {
	_, err := repo.db.Exec("UPDATE users SET current_lang=$1 WHERE id=$2", langId, userId)
	if err != nil {
		return err
	}
	return nil
}

func (repo *repository) createUser(userId int) (*user, error) {

	lang, err := repo.getLang(defaultLanguageId)
	if err != nil {
		return nil, err
	}

	u := user{userId, readyForQuestion, lang}

	_, err = repo.db.Exec(`
		INSERT INTO users (id, current_lang) 
		VALUES ($1, $2) 
		ON CONFLICT DO NOTHING`, userId, defaultLanguageId)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func (repo *repository) deleteLastWord(userId int) error {
	_, err := repo.db.Exec("DELETE FROM questions WHERE user_id=$1", userId)
	if err != nil {
		return err
	}
	return nil
}

func (repo *repository) setLastWord(userId int, w word) error {
	_, err := repo.db.Exec(`
		INSERT INTO questions (user_id, word_id) 
		VALUES ($1, $2) 
		ON CONFLICT (user_id) 
		    DO UPDATE SET word_id=$2`, userId, w.id)
	if err != nil {
		return err
	}
	return nil
}

func (repo *repository) getLastWord(userId int) (*word, error) {
	row := repo.db.QueryRow("SELECT word_id FROM questions WHERE user_id=$1", userId)
	var wordId int
	err := row.Scan(&wordId)
	if err != nil {
		return nil, err
	}

	return repo.getWord(wordId)
}

func (repo *repository) getRandomWord(userId int) (*word, error) {
	var wordId int

	tx, err := repo.db.Begin()

	if err != nil {
		return nil, err
	}

	row := tx.QueryRow(`
		SELECT word_id 
		FROM user_words 
		WHERE user_id=$1 
		OFFSET floor(random() * (SELECT COUNT(*) FROM words)) LIMIT 1`, userId)

	err = row.Scan(&wordId)
	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		return nil, noWordsFound
	}

	_, err = tx.Exec(`
		INSERT INTO questions (user_id, word_id) 
		VALUES ($1, $2) 
		ON CONFLICT (user_id, word_id) DO UPDATE SET word_id=$2`, userId, wordId)

	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	_, err = tx.Exec("UPDATE users SET current_state=$1 WHERE id=$2", waitingAnswer, userId)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}

	return repo.getWord(wordId)
}

func (repo *repository) getWord(wordId int) (*word, error) {
	wordRow := repo.db.QueryRow("SELECT word, stem, lang, id FROM words WHERE id=$1", wordId)

	w := word{}
	var langId int
	err := wordRow.Scan(&w.word, &w.stem, &langId, &w.id)
	if err != nil {
		return nil, fmt.Errorf("random word row scan: %v", err.Error())
	}

	l, err := repo.getLang(langId)
	if err != nil {
		return nil, err
	}
	w.lang = l

	return &w, nil
}

func (repo *repository) getUser(id int) (*user, error) {
	userRow := repo.db.QueryRow("SELECT * FROM users WHERE id=$1", id)
	u := user{}
	var langId int
	err := userRow.Scan(&u.id, &langId, &u.currentState)

	if err != nil {
		return nil, err
	}

	l, err := repo.getLang(langId)
	if err != nil {
		return nil, err
	}

	u.currentLanguage = l

	return &u, nil
}

func (repo *repository) getLang(id int) (*lang, error) {
	langRow := repo.db.QueryRow("SELECT * FROM languages WHERE id=$1", id)
	l := lang{}
	err := langRow.Scan(&l.id, &l.code, &l.englishName, &l.localizedName)
	if err != nil {
		return nil, fmt.Errorf("Get lang: %v", err.Error())
	}
	return &l, nil
}

func (repo *repository) getLanguages() ([]lang, error) {

	log.Println("query langs count")
	row := repo.db.QueryRow("SELECT COUNT(*) FROM languages")
	var count int
	err := row.Scan(&count)
	if err != nil {
		return nil, err
	}

	langs := make([]lang, 0, count)

	log.Println("query languages")
	rows, err := repo.db.Query("SELECT * FROM languages")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {

		err := rows.Err()
		if err != nil {
			return nil, err
		}

		log.Println("lang iter")
		l := lang{}
		err = rows.Scan(&l.id, &l.code, &l.englishName, &l.localizedName)
		if err != nil {
			fmt.Printf("Get lang: %v", err.Error())
			continue
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

	lang, err := repo.getUserLanguage(r.params.userId)
	if err != nil {
		//TODO: error handling
		_ = tx.Rollback()
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO answers (word_id, user_id, correct, user_lang, guess) 
		VALUES ($1, $2, $3, $4, $5)`, p.word.id, p.userId, r.correct(), lang.id, p.guess)

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

func (repo *repository) addWordForUser(userId int, word word, lc string) error {

	tx, err := repo.db.Begin()
	if err != nil {
		return fmt.Errorf("postgre tx begin: %v\n", err.Error())
	}

	//Trying to insert word with ignoring duplicate keys
	langId, ok := langMap[lc]
	if !ok {
		//TODO: error handling?
		_ = tx.Rollback()
		fmt.Printf("No such lang code found: %s", lc)
		return err
	}

	log.Println("Insert word")

	var wordId int
	err = tx.QueryRow(`
		INSERT INTO words (word, stem, lang)
		VALUES ($1, $2, $3)
		ON CONFLICT (word, stem, lang) 
		    DO UPDATE SET word=$1 RETURNING id`, word.word, word.stem, langId).Scan(&wordId)

	if err != nil {
		log.Printf("insertion word: %v\n", err.Error())
		log.Println("Query existing word id")
		//If error happened, probably word with same key was already added, trying to get id
		err = tx.QueryRow(`
			SELECT id
			FROM words
			WHERE word=$1 AND stem=$2 AND lang=$3`, word.word, word.stem, word.lang).Scan(&wordId)

		if err != nil {
			//TODO: error handling?
			_ = tx.Rollback()
			fmt.Printf(err.Error())
			return err
		}
	}

	log.Println("Insert user word")
	_, err = tx.Exec(`
		INSERT INTO user_words (user_id, word_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, userId, wordId)

	if err != nil {
		//TODO: error handling?
		_ = tx.Rollback()
		return fmt.Errorf("postgre: inserting user_word: %v", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("postgre: tx commit: %v", err.Error())
	}

	return nil
}
