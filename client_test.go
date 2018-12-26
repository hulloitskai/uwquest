package uwquest_test

import (
	"os"
	"testing"

	"github.com/stevenxie/uwquest"
	ess "github.com/unixpickle/essentials"
)

var client *uwquest.Client

func TestMain(m *testing.M) {
	var err error
	if client, err = uwquest.NewClient(); err != nil {
		ess.Die("Error creating Quest client:", err)
	}
	if err = client.Login(user, pass); err != nil {
		ess.Die("Error while logging into Quest:", err)
	}
	os.Exit(m.Run())
}
