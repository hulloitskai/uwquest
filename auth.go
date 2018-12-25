package uwquest

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	gq "github.com/PuerkitoBio/goquery"
	ess "github.com/unixpickle/essentials"
)

// Login authenticats the Client session with the Quest API backend.
//
// Requires a username (WatIAM ID) and password.
func (c *Client) Login(user, pass string) error {
	const questSAMLAuthURL = "https://quest.pecs.uwaterloo.ca/psp/SS/ACADEMIC/" +
		"SA/h/?tab=DEFAULT"

	loginURL, err := c.prelogin()
	if err != nil {
		return ess.AddCtx("uwquest: performing prelogin sequence", err)
	}

	// Create URL-encoded login form.
	form := make(url.Values)
	form.Add("j_username", user)
	form.Add("j_password", pass)
	form.Add("_eventId_proceed", "Login")
	body := strings.NewReader(form.Encode())

	// Create and perform IDP login request.
	req, err := http.NewRequest("POST", loginURL, body)
	if err != nil {
		return ess.AddCtx("uwquest: creating IDP login request", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.Session.Do(req)
	if err != nil {
		return ess.AddCtx("uwquest: performing IDP login", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("uwquest: got non-200 status code from IDP login: got "+
			"code: %d", res.StatusCode)
	}
	defer res.Body.Close()

	// Scrape SAML response from IDP login response.
	samlResp, err := parseSAMLResponse(res.Body)
	if err != nil {
		return ess.AddCtx("uwquest: parsing login response body for SAML response",
			err)
	}
	if err = res.Body.Close(); err != nil {
		return ess.AddCtx("uwquest: closing response body", err)
	}

	// Create and perform Quest auth request.
	form = make(url.Values)
	form.Add("SAMLResponse", samlResp)
	body = strings.NewReader(form.Encode())

	if req, err = http.NewRequest("POST", questSAMLAuthURL, body); err != nil {
		return ess.AddCtx("uwquest: creating Quest auth request", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	if res, err = c.Session.Do(req); err != nil {
		return ess.AddCtx("uwquest: authenticating with Quest", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("uwquest: got non-200 status code from Quest "+
			"authentication: got code %d", res.StatusCode)
	}
	return nil
}

// prelogin prepares c.Session for a login attempt by fetching pre-login cookies
// and querying for the dynamic login link.
func (c *Client) prelogin() (loginURL string, err error) {
	const (
		idpCookieURL = "https://quest.pecs.uwaterloo.ca/psp/SS/ACADEMIC/SA/" +
			"?cmd=login&languageCd=ENG"
		idpLinkURL = "https://idp.uwaterloo.ca/idp/profile/SAML2/Unsolicited/SSO" +
			"?providerId=quest.ss.apps.uwaterloo.ca"
	)

	// Set cookies required for IDP login.
	res, err := c.Session.Get(idpCookieURL)
	if err != nil {
		return "", ess.AddCtx("fetching IDP prelogin cookies", err)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got non-200 status code when fetching IDP prelogin "+
			"cookies: got code %d", res.StatusCode)
	}

	// Fetch IDP login page to begin server-side authentication procedure.
	res, err = c.Session.Get(idpLinkURL)
	if err != nil {
		return "", ess.AddCtx("fetching dynamic IDP login URL", err)
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("got non-200 status code while fetching IDP login "+
			"page: got code %d", res.StatusCode)
	}

	rawQuery := res.Request.URL.RawQuery
	if rawQuery == "" {
		return "", errors.New("could not determine dynamic IDP login URL")
	}
	loginURL = "https://idp.uwaterloo.ca/idp/profile/SAML2/Unsolicited/SSO?" +
		rawQuery
	return loginURL, nil
}

func parseSAMLResponse(body io.Reader) (string, error) {
	doc, err := gq.NewDocumentFromReader(body)
	if err != nil {
		return "", err
	}

	sel := doc.Find("input[type=\"hidden\"]")
	if slen := sel.Length(); slen != 1 {
		return "", fmt.Errorf("expected 1 hidden input tag, got %d", slen)
	}
	samlResp, ok := sel.Attr("value")
	if !ok {
		return "", errors.New("form input has no 'value' attribute")
	}
	return samlResp, nil
}
