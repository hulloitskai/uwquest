package main

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/stevenxie/uwquest"
	ess "github.com/unixpickle/essentials"
)

func init() {
	godotenv.Load()
}

func main() {
	creds, err := ReadCreds()
	if err != nil {
		ess.Die("Reading Quest credentials:", err)
	}

	quest, err := uwquest.NewClient()
	if err != nil {
		ess.Die("Creating Quest client:", err)
	}

	fmt.Println("Logging into Quest...")
	if err = quest.Login(creds.User, creds.Pass); err != nil {
		ess.Die("Error logging into Quest:", err)
	}

	fmt.Println("Fetching terms...")
	terms, err := quest.Terms()
	if err != nil {
		ess.Die("Error fetching terms data:", err)
	}

	fmt.Println("Terms found:")
	for _, term := range terms {
		fmt.Printf("Term %d: %s\n", term.Index, term.Name)
		grades, err := quest.Grades(term.Index)
		if err != nil {
			ess.Die(fmt.Sprintf("Error fetching grades data for term %d:",
				term.Index), err)
		}
		for _, courseGrade := range grades {
			if courseGrade.Grade == "" {
				continue
			}
			fmt.Printf("\t- %s: %s\n", courseGrade.Name, courseGrade.Grade)
		}
	}

	fmt.Print("\nPress enter to exit...")
	fmt.Scanln() // wait for newline
}
