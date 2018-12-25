package uwquest

import (
	"net/http"
	"net/http/cookiejar"

	ess "github.com/unixpickle/essentials"
)

// Client is capable of interacting with the UW Quest API.
type Client struct {
	// Session refers the HTTP client session that performs Client's underlying
	// requests.
	//
	// This Session is authorized with the Quest API backend upon login.
	Session *http.Client

	// Jar is a cookiejar that contains Session's cookies.
	Jar *cookiejar.Jar
}

// NewClient returns a new Client.
//
// It needs to be authenticated with the Quest backend using Login, before it
// can fetch other data from Quest.
func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, ess.AddCtx("client: creating cookiejar", err)
	}

	return &Client{
		Session: &http.Client{Jar: jar},
		Jar:     jar,
	}, nil
}
