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
		return nil, ess.AddCtx("uwquest: parsing response body with goquery", err)
	}
	sel := doc.Find(`#SSR_DUMMY_RECV1\$scroll\$0`).Children()
	if sel.Length() != 1 {
		return nil, errors.New("uwquest: could not locate terms table")
	}

	var terms []*Term
	sel.Children().EachWithBreak(func(i int, row *gq.Selection) bool {
		if _, ok := row.Attr("id"); !ok {
			return true // continue
		}

		var term *Term
		if term, err = parseTermRow(row); err != nil {
			err = ess.AddCtx(fmt.Sprintf("row %d", i), err)
			return false
		}

		terms = append(terms, term)
		return true
	})
	if err != nil {
		return nil, ess.AddCtx("uwquest: parsing term table", err)
	}

	err = res.Body.Close()
	return terms, ess.AddCtx("uwquest: closing response body", err)
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

	sel := row.Find(fmt.Sprintf(`#TERM_CAR\$%d`, t.Index))
	if sel.Length() == 0 {
		return nil, errors.New("could not find term name")
	}
	t.Name = sel.Text()

	if sel = row.Find(fmt.Sprintf(`#CAREER\$%d`, t.Index)); sel.Length() == 0 {
		return nil, errors.New("could not find career info")
	}
	t.Career = sel.Text()

	if sel = row.Find(fmt.Sprintf(`#INSTITUTION\$%d`, t.Index)); sel.Length() == 0 {
		return nil, errors.New("could not find institution name")
	}
	t.Institution = sel.Text()

	return t, nil
}
