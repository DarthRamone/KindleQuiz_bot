package kindle_quiz_bot

import (
	"testing"
)

func TestCreateUser(t *testing.T) {
	user, err := repo.createUser(0)
	if err != nil {
		t.Fatalf("Couldn't create user: %v", err)
	}

	if user == nil {
		t.Fatalf("User is nil")
	}
}

func TestDeleteLastWord(t *testing.T) {
	err := repo.deleteLastWord(testUserId)
	if err != nil {
		t.Fatalf("Couldn't delete last word: %v", err)
	}
}

func TestGetLang(t *testing.T) {
	lang, err := repo.getLang(langId)
	if err != nil {
		t.Fatalf("Couldn't get lang: %v", err)
	}

	if lang == nil {
		t.Fatalf("Lang is nil")
	}
}

func TestGetRandomWord(t *testing.T) {
	testWord, err := repo.getRandomWord(testUserId)
	if err != nil {
		t.Fatalf("Couldn't get random word: %v", err)
	}

	if testWord == nil {
		t.Fatalf("Word is nil")
	}
}

func TestSetLastWord(t *testing.T) {
	word, err := repo.getRandomWord(testUserId)
	if err != nil {
		t.Fatalf("Couldn't get random word: %v", err)
	}

	err = repo.setLastWord(testUserId, *word)
	if err != nil {
		t.Fatalf("Couldn't set last word: %v", err)
	}
}

func TestGetLastWord(t *testing.T) {
	word, err := repo.getRandomWord(testUserId)
	if err != nil {
		t.Fatalf("Couldn't get random word: %v", err)
	}

	err = repo.setLastWord(testUserId, *word)
	if err != nil {
		t.Fatalf("Couldn't set last word: %v", err)
	}

	lastWord, err := repo.getLastWord(testUserId)
	if err != nil {
		t.Fatalf("Couldn't get last word: %v", err)
	}

	//TODO: better equality comparing
	if lastWord.id != word.id {
		t.Fatalf("Last word id isn't same")
	}
}

func TestGetWord(t *testing.T) {
	word, err := repo.getRandomWord(testUserId)
	if err != nil {
		t.Fatalf("Couldn't get random word: %v", err)
	}

	newWord, err := repo.getWord(word.id)
	if err != nil {
		t.Fatalf("Couldn't get word: %v", err)
	}

	//TODO: better equality comparing
	if newWord.id != word.id {
		t.Fatalf("Word id isn't same")
	}
}

func TestGetUser(t *testing.T) {
	user, err := repo.getUser(testUserId)
	if err != nil {
		t.Fatalf("Couldn't get user: %v", err)
	}

	//TODO: better equality comparing
	if user.id != testUserId {
		t.Fatalf("User id isn't same")
	}
}
