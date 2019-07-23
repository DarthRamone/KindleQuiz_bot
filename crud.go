package main

import (
	"fmt"
	"log"
)

func getRandomWord(userId int) (*word, error) {
	var wordId int

	row := db.QueryRow("SELECT word_id FROM user_words WHERE user_id=$1 OFFSET floor(random() * (SELECT COUNT(*) FROM words)) LIMIT 1", userId)
	err := row.Scan(&wordId)
	if err != nil {
		fmt.Printf("shit")
		return nil, err
	}
	fmt.Printf("word_id: %d\n", wordId)

	wordRow := db.QueryRow("SELECT word, stem, lang FROM words WHERE id=$1", wordId)

	w := word{}
	var langId int
	err = wordRow.Scan(&w.word, &w.stem, &langId)
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
	err := userRow.Scan(&u.id, &langId)
	if err != nil {
		return nil, fmt.Errorf("Get user: %v", err.Error())
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
