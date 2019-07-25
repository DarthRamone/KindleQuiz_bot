package main

import (
	"fmt"
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

func getUserLanguage(userId int) (*lang, error) {
	l := lang{}
	err := db.QueryRow("" +
		"SELECT * FROM languages " +
		"WHERE id=" +
			"(SELECT current_lang FROM users WHERE id=$1)", userId).Scan(&l.id, &l.code, &l.english_name, &l.localized_name)
	if err != nil {
		return nil, err
	}

	return &l, nil
}

func getAllUserIds() ([]int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query("SELECT id FROM users")
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

func updateUserState(userId, state int) error {
	_, err := db.Exec("UPDATE users SET current_state=$1 WHERE id=$2", state, userId)
	if err != nil {
		return err
	}
	return nil
}

func updateUserLang(userId, langId int) error {
	_, err := db.Exec("UPDATE users SET current_lang=$1 WHERE id=$2", langId, userId)
	if err != nil {
		return err
	}
	return nil
}

func createUser(userId int) (*user, error) {

	lang, err := getLang(defaultLanguageId)
	if err != nil {
		return nil, err
	}

	u := user{userId, readyForQuestion, lang}

	_, err = db.Exec("INSERT INTO users (id, current_lang) VALUES ($1, $2) ON CONFLICT DO NOTHING", userId, defaultLanguageId)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

func deleteLastWord(userId int) error {
	_, err := db.Exec("DELETE FROM questions WHERE user_id=$1", userId)
	if err != nil {
		return err
	}
	return nil
}

func setLastWord(userId int, w word) error {
	_, err := db.Exec("INSERT INTO questions (user_id, word_id) VALUES ($1, $2) ON CONFLICT (user_id) DO UPDATE SET word_id=$2", userId, w.id)
	if err != nil {
		return err
	}
	return nil
}

func getLastWord(userId int) (*word, error) {
	row := db.QueryRow("SELECT word_id FROM questions WHERE user_id=$1", userId)
	var wordId int
	err := row.Scan(&wordId)
	if err != nil {
		return nil, err
	}

	return getWord(wordId)
}

func getRandomWord(userId int) (*word, error) {
	var wordId int

	row := db.QueryRow("SELECT word_id FROM user_words WHERE user_id=$1 OFFSET floor(random() * (SELECT COUNT(*) FROM words)) LIMIT 1", userId)
	err := row.Scan(&wordId)
	if err != nil {
		fmt.Printf("shit")
		return nil, err
	}

	return getWord(wordId)
}

func getWord(wordId int) (*word, error) {
	wordRow := db.QueryRow("SELECT word, stem, lang, id FROM words WHERE id=$1", wordId)

	w := word{}
	var langId int
	err := wordRow.Scan(&w.word, &w.stem, &langId, &w.id)
	if err != nil {
		return nil, fmt.Errorf("Random word row scan: %v", err.Error())
	}

	l, err := getLang(langId)
	if err != nil {
		return nil, err
	}
	w.lang = l

	return &w, nil
}

func getUser(id int) (*user, error) {
	userRow := db.QueryRow("SELECT * FROM users WHERE id=$1", id)
	u := user{}
	var langId int
	err := userRow.Scan(&u.id, &langId, &u.currentState)

	if err != nil {
		return nil, err
	}

	l, err := getLang(langId)
	if err != nil {
		return nil, err
	}

	u.currentLanguage = l

	return &u, nil
}

func getLang(id int) (*lang, error) {
	langRow := db.QueryRow("SELECT * FROM languages WHERE id=$1", id)
	l := lang{}
	err := langRow.Scan(&l.id, &l.code, &l.english_name, &l.localized_name)
	if err != nil {
		return nil, fmt.Errorf("Get lang: %v", err.Error())
	}
	return &l, nil
}

func getLanguages() ([]lang, error) {

	log.Println("query langs count")
	row := db.QueryRow("SELECT COUNT(*) FROM languages")
	var count int
	err := row.Scan(&count)
	if err != nil {
		return nil, err
	}

	langs := make([]lang, 0, count)

	log.Println("query languages")
	rows, err := db.Query("SELECT * FROM languages")
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

func getLanguageWithCode(code string) (*lang, error) {
	l := lang{}
	err := db.QueryRow("SELECT * FROM languages WHERE code=$1", code).Scan(&l.id, &l.code, &l.english_name, &l.localized_name)
	if err != nil {
		return nil, fmt.Errorf("lang with code: %v", err.Error())
	}
	return &l, nil
}

func writeAnswer(r guessResult) error {

	p := r.params

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	lang, err := getUserLanguage(r.params.userId)
	if err != nil {
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

	_, err = tx.Exec(queryStr, p.word.id)

	if err != nil {
		return fmt.Errorf("write answer: %s", err.Error())
	}

	err = tx.Commit()
	if err != nil {
		return nil
	}

	return nil
}
