package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	quiz "github.com/DarthRamone/KindleQuiz_bot/internal/app/kindle_quiz_bot"
)

var token = flag.String("token", "", "telegram API bot token")

func main() {
	flag.Parse()

	tgToken, err := getTgToken()
	if err != nil {
		log.Fatal(err)
	}

	bot, err := quiz.NewQuizTelegramBot(tgToken)
	if err != nil {
		log.Fatal(err)
	}

	err = bot.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer bot.Stop()
}

func getTgToken() (string, error) {
	var tgToken = *token
	if tgToken == "" {
		tgToken = os.Getenv("TG_TOKEN")
	}

	if tgToken == "" {
		return "", fmt.Errorf("Unable get telegram bot api token. Pass token argument to command line, or set TG_TOKEN environment variable.")
	}

	return tgToken, nil
}
