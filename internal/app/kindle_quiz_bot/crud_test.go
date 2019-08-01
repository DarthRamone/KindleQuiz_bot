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
	lang, err := repo.getLang(testLangId)
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

func TestGetLanguages(t *testing.T) {
	languages, err := repo.getLanguages()
	if err != nil {
		t.Fatalf("get languages failed: %v", err)
	}

	if languages == nil {
		t.Fatalf("Languages are nil")
	}
}

func TestGetLanguageWithCode(t *testing.T) {
	lang, err := repo.getLanguageWithCode("ru")
	if err != nil {
		t.Fatalf("Couldn't get language with code: %v", err)
	}

	if lang == nil {
		t.Fatalf("Language is nil")
	}

	if lang.code != "ru" {
		t.Fatalf("Language code incorrect")
	}
}

func TestPersistAnswer(t *testing.T) {
	word, err := repo.getRandomWord(testUserId)
	if err != nil {
		t.Fatalf("Couldn't get random word: %v", err)
	}

	params := guessParams{*word, "!@#$%", testUserId}
	result := guessResult{params, "foobar"}

	err = repo.persistAnswer(result)
	if err != nil {
		t.Fatalf("Couldn't persist answer: %v", err)
	}
}

func TestAddWordForUser(t *testing.T) {
	word := word{word: "проверил", stem: "проверить"}

	err := repo.addWordForUser(testUserId, word, "ru")
	if err != nil {
		t.Fatalf("Couldn't add word for user: %v", err)
	}
}

func TestGetUserLanguage(t *testing.T) {
	_, err := repo.getUserLanguage(-1)
	if err == nil {
		t.Fatalf("Lang should be nil")
	}

	lang, err := repo.getUserLanguage(testUserId)
	if err != nil {
		t.Fatalf("Couldn't get user language: %v", err)
	}

	if lang == nil {
		t.Fatalf("User language is nil")
	}
}

func TestUpdateUserState(t *testing.T) {
	err := repo.updateUserState(testUserId, awaitingUpload)
	if err != nil {
		t.Fatalf("Couldn't update user state: %v", err)
	}

	user, err := repo.getUser(testUserId)
	if err != nil {
		t.Fatalf("Couldn't get user: %v", err)
	}

	if user.currentState != awaitingUpload {
		t.Fatalf("Invalid user state")
	}

	//Second iteration, in case of initial user state was
	//already awaitingUpload, but for some reasons silent failed
	err = repo.updateUserState(testUserId, waitingAnswer)
	if err != nil {
		t.Fatalf("Couldn't update user state: %v", err)
	}

	user, err = repo.getUser(testUserId)
	if err != nil {
		t.Fatalf("Couldn't get user: %v", err)
	}

	if user.currentState != waitingAnswer {
		t.Fatalf("Invalid user state")
	}
}

func TestUpdateUserLan(t *testing.T) {
	err := repo.updateUserLang(testUserId, testLangId)
	if err != nil {
		t.Fatalf("Couldn't update user lang: %v", err)
	}

	user, err := repo.getUser(testUserId)
	if err != nil {
		t.Fatalf("Couldn't get user: %v", err)
	}

	if user.currentLanguage.id != testLangId {
		t.Fatalf("Invalid user state")
	}

	//Second iteration, in case of initial user state was
	//already testLangId, but for some reasons silent failed
	err = repo.updateUserLang(testUserId, 1)
	if err != nil {
		t.Fatalf("Couldn't update user state: %v", err)
	}

	user, err = repo.getUser(testUserId)
	if err != nil {
		t.Fatalf("Couldn't get user: %v", err)
	}

	if user.currentLanguage.id != 1 {
		t.Fatalf("Invalid user state")
	}
}
