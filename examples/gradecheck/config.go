package main

import (
	"fmt"

	"github.com/caarlos0/env"
	"github.com/howeyc/gopass"
	ess "github.com/unixpickle/essentials"
)

// Credentials is a username / password set that is used to login to Quest.
type Credentials struct {
	User string `env:"QUEST_USER"`
	Pass string `env:"QUEST_PASS"`
}

// ReadCreds reads Quest credentials from os.Stdin.
func ReadCreds() (*Credentials, error) {
	creds := new(Credentials)
	if err := env.Parse(creds); err != nil {
		return nil, ess.AddCtx("gradecheck: parsing environment", err)
	}

	if creds.User == "" {
		fmt.Print("Enter your Quest ID: ")
		if _, err := fmt.Scanf("%s", &creds.User); err != nil {
			return nil, ess.AddCtx("gradecheck: reading username", err)
		}
	}
	if creds.Pass == "" {
		fmt.Print("Enter your Quest password: ")
		data, err := gopass.GetPasswdMasked()
		if (err != nil) && (err != gopass.ErrInterrupted) {
			return nil, ess.AddCtx("gradecheck: reading password", err)
		}
		creds.Pass = string(data)
	}
	return creds, nil
}
