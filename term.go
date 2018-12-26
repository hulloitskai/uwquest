package uwquest

import (
	"errors"
	"fmt"
	"net/http"

	gq "github.com/PuerkitoBio/goquery"
	ess "github.com/unixpickle/essentials"
)

// A Term represents a UW school term.
type Term struct {
	Index       int
	Name        string
	Career      string
	Institution string
}

func (t *Term) String() string {
	return fmt.Sprintf("Term{Index: %d, Name: %s, Career: %s, Institution: %s}",
		t.Index, t.Name, t.Career, t.Institution)
}

// Terms fetches all the terms that a student has been enrolled for.
func (c *Client) Terms() ([]*Term, error) {
	res, err := c.Session.Get(GradesURL)
	if err != nil {
		return nil, ess.AddCtx("uwquest: fetching grades page", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("uwquest: got non-200 status code while fetching "+
			"grades page: got code %d", res.StatusCode)
	}
	defer res.Body.Close()

	// Scrape response for data in the terms table.
	doc, err := gq.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, ess.AddCtx("parsing response body with goquery", err)
	}
	sel := doc.Find(`#SSR_DUMMY_RECV1\$scroll\$0`).Children()
	if sel.Length() != 1 {
		return nil, errors.New("could not locate terms table")
	}

	terms, err := parseTerms(sel)
	if err != nil {
		return nil, ess.AddCtx("uwquest: parsing terms", err)
	}

	err = res.Body.Close()
	return terms, ess.AddCtx("uwquest: closing response body", err)
}

// TermsWithSchedule fetches the study terms for which Quest has course
// schedules available.
func (c *Client) TermsWithSchedule() ([]*Term, error) {
	res, err := c.Session.Get(SchedulesURL)
	if err != nil {
		return nil, ess.AddCtx("uwquest: fetching course schedule page", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("uwquest: got non-200 status code while fetching "+
			"schedule page: got code %d", res.StatusCode)
	}
	defer res.Body.Close()

	// Scrape response for data in the terms table.
	doc, err := gq.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, ess.AddCtx("parsing response body with goquery", err)
	}
	sel := doc.Find(`#SSR_DUMMY_RECV1\$scroll\$0`).Children().Find("tbody")
	if sel.Length() != 1 {
		return nil, errors.New("could not locate terms table")
	}

	terms, err := parseTerms(sel)
	if err != nil {
		return nil, ess.AddCtx("uwquest: parsing terms", err)
	}

	err = res.Body.Close()
	return terms, ess.AddCtx("uwquest: closing response body", err)
}

func parseTerms(tableBody *gq.Selection) ([]*Term, error) {
	var (
		terms []*Term
		err   error
	)
	tableBody.Children().EachWithBreak(func(i int, row *gq.Selection) bool {
		if _, ok := row.Attr("id"); !ok {
			return true // continue
		}

		var term *Term
		if term, err = parseTermRow(row); err != nil {
			ess.AddCtxTo(fmt.Sprintf("row %d", i), &err)
			return false
		}

		terms = append(terms, term)
		return true
	})
	if err != nil {
		return nil, ess.AddCtx("parsing terms table", err)
	}
	return terms, nil
}

func parseTermRow(row *gq.Selection) (*Term, error) {
	var (
		t      = new(Term)
		id, ok = row.Attr("id")
	)
	if !ok {
		return nil, errors.New("row does not contain an 'id' attribute")
	}
	t.Index = int(id[len(id)-1]-'0') - 1

	var (
		scraper  = indexedScraper{Index: t.Index, Sel: row}
		sel, err = scraper.Find("TERM_CAR", "term name")
	)
	if err != nil {
		return nil, err
	}
	t.Name = sel.Text()

	if sel, err = scraper.Find("CAREER", "career info"); err != nil {
		return nil, err
	}
	t.Career = sel.Text()

	if sel, err = scraper.Find("INSTITUTION", "institution name"); err != nil {
		return nil, err
	}
	t.Institution = sel.Text()

	return t, nil
}
