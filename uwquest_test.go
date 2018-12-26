package uwquest_test

import (
	"os"

	"github.com/joho/godotenv"
	ess "github.com/unixpickle/essentials"
)

var user, pass string

// Load environment variables upon initializing.
func init() {
	if err := godotenv.Load(); err != nil {
		ess.Die("Failed to load environment variabels:", err)
	}

	var ok bool
	if user, ok = os.LookupEnv("QUEST_USER"); !ok {
		ess.Die("No such env var 'QUEST_USER'")
	}
	if pass, ok = os.LookupEnv("QUEST_PASS"); !ok {
		ess.Die("No such env var 'QUEST_PASS'")
	}
}
