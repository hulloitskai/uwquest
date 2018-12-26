package uwquest_test

import (
	"os"
	"testing"

	"github.com/joho/godotenv"

	"github.com/stevenxie/uwquest"
	ess "github.com/unixpickle/essentials"
)

var client *uwquest.Client

func TestMain(m *testing.M) {
	if os.Getenv("TRAVIS") != "true" { // only load from .env if not CI
		if err := godotenv.Load(); err != nil {
			ess.Die("Failed to load environment variabels:", err)
		}
	}

	user, ok := os.LookupEnv("QUEST_USER")
	if !ok {
		ess.Die("No such env var 'QUEST_USER'")
	}
	pass, ok := os.LookupEnv("QUEST_PASS")
	if !ok {
		ess.Die("No such env var 'QUEST_PASS'")
	}

	var err error
	if client, err = uwquest.NewClient(); err != nil {
		ess.Die("Error creating Quest client:", err)
	}
	if err = client.Login(user, pass); err != nil {
		ess.Die("Error while logging into Quest:", err)
	}
	os.Exit(m.Run())
}
