package main

import (
	"github.com/DarthRamone/gtranslate"
	"strings"
	"time"
)

type guessParams struct {
	word  word
	guess string
	user  *user
}

type guessResult struct {
	params      *guessParams
	translation string
}

func (t *guessResult) correct() bool {
	return compareWords(t.params.guess, t.translation)
}

//var guesses = make(chan guessParams)
//var results = make(chan guessResult)
//var stopListen = make(chan struct{})
//
//func startListen() {
//	go func() {
//		for {
//			select {
//			case g := <- guesses:
//
//			case r := <- results:
//			}
//		}
//	}()
//}

func guessWord(p *guessParams) (*guessResult, error) {

	translated, err := translateWord(p.word, p.user.currentLanguage)
	if err != nil {
		return nil, err
	}

	res := guessResult{p, translated}

	return &res, nil
}

func translateWord(w word, dst *lang) (string, error) {

	translated, err := gtranslate.TranslateWithParams(
		w.word,
		gtranslate.TranslationParams{
			From:  w.lang.code,
			To:    dst.code,
			Delay: time.Second,
			Tries: 5,
		},
	)

	if err != nil {
		return "", err
	}

	return translated, nil
}

func compareWords(w1, w2 string) bool {

	s1 := strings.ToLower(strings.Trim(w1, " "))
	s2 := strings.ToLower(strings.Trim(w2, " "))

	return s1 == s2
}
