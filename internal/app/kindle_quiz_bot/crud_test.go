package kindle_quiz_bot

import (
	"fmt"
	"github.com/DarthRamone/dockertest"
	_ "github.com/lib/pq"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

const (
	userId = 0
	langId = 2
)

var (
	repo = repository{}
	dbParams = connectionParams{user: "postgres", dbName: "vocab", sslMode: "disable", url:"localhost"}
	testWord word
	testLang lang
)

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	postgresOptions := dockertest.RunOptions{
		Name:"postgres",
		Hostname:"postgres_test",
		Repository:"postgres",
		Tag:"11.4",
		Env:[]string{"POSTGRES_DB=vocab"},
	}

	// pulls an image, creates a container based on it and runs it
	postgres, err := pool.RunWithOptions(&postgresOptions)
	if err != nil {
		log.Fatalf("Could not start postgres: %s", err)
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		var err error
		port, err := strconv.Atoi(postgres.GetPort("5432/tcp"))
		if err != nil {
			return nil
		}

		dbParams.port = port

		err = repo.connect(dbParams)
		if err != nil {
			return err
		}
		return repo.db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	contextPath, err := filepath.Abs("../../../")
	fmt.Printf("Path: %s\n", contextPath)

	dockerfilePath := contextPath + "/build/migrations/Dockerfile"
	fmt.Printf("Dockerfile path: %s\n", dockerfilePath)

	pgPortEnv := fmt.Sprintf("PG_PORT=%d", 5432)
	pgHostEnv := fmt.Sprintf("PG_HOST=%s", postgres.Container.NetworkSettings.IPAddress)

	gooseOptions := dockertest.RunOptions{
		Name:"goose",
		Env:[]string{pgPortEnv, pgHostEnv},
	}

	goose, err := pool.BuildAndRunWithOptions("build/migrations/Dockerfile", contextPath, &gooseOptions)
	if err != nil {
		log.Fatalf(err.Error())
	}

	//TODO: figure out when migration is completed 
	time.Sleep(time.Second * 2)

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(postgres); err != nil {
		log.Fatalf("Could not purge postgres: %s", err)
	}

	if err := pool.Purge(goose); err != nil {
		log.Fatalf("Could not purge postgres: %s", err)
	}

	os.Exit(code)
}

func TestBuildLangMap(t *testing.T) {
	err := repo.buildLangMap()
	if err != nil {
	log.Fatalf("Couldn't build lang map: %v", err)
	}
}

func TestCreateUser(t *testing.T) {
	user, err := repo.createUser(0)
	if err != nil {
		log.Fatalf("Couldn't create user: %v", err)
	}

	if user == nil {
		t.Fatalf("User is nil")
	}
}

func TestDeleteLastWord(t *testing.T) {
	err := repo.deleteLastWord(userId)
	if err != nil {
		log.Fatalf("Couldn't delete last word: %v", err)
	}
}

func TestGetLang(t *testing.T) {
	lang, err := repo.getLang(langId)
	if err != nil {
		log.Fatalf("Couldn't get lang: %v", err)
	}

	if lang == nil {
		t.Fatalf("Lang is nil")
	}

	testLang = *lang
}

func TestGetRandomWord(t *testing.T) {
	testWord, err := repo.getRandomWord(userId)
	if err != nil {
		log.Fatalf("Couldn't get random word: %v", err)
	}

	if testWord == nil {
		t.Fatalf("Word is nil")
	}
}

func TestSetLastWord(t *testing.T) {
	err := repo.setLastWord(userId, testWord)
	if err != nil {
		log.Fatalf("Couldn't set last word: %v", err)
	}
}
