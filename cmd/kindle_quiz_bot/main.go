package main

import (
	"flag"
	quiz "github.com/DarthRamone/KindleQuiz_bot/internal/app/kindle_quiz_bot"
	"log"
	"os"
)

var token = flag.String("token", "", "telegram API bot token")

func main() {

	flag.Parse()

	tgToken := getTgToken()


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



func getTgToken() string {
	var tgToken = *token
	if tgToken == "" {
		tgToken = os.Getenv("TG_TOKEN")
	}

	if tgToken == "" {
		log.Fatal("unable get telegram bot api token")
	}

	return tgToken
}
