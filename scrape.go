package uwquest

import (
	"errors"
	"fmt"
	"io"
	"net/url"

	gq "github.com/PuerkitoBio/goquery"
	ess "github.com/unixpickle/essentials"
)

// scrapeHiddenFields scrapes the HTML data from body for hidden fields, and
// returns the fields as a url.Values.
func scrapeHiddenFields(body io.Reader) (url.Values, error) {
	doc, err := gq.NewDocumentFromReader(body)
	if err != nil {
		return nil, ess.AddCtx("parsing Quest homepage with goquery", err)
	}
	sel := doc.Find("#win0divPSHIDDENFIELDS")
	if sel.Length() != 1 {
		return nil, errors.New("could not find hidden fields on Quest homepage")
	}

	fields := make(url.Values)
	sel.Children().Each(func(_ int, field *gq.Selection) {
		name, ok := field.Attr("name")
		if !ok {
			return
		}
		value, _ := field.Attr("value")
		fields.Set(name, value)
	})
	return fields, nil
}

type indexedScraper struct {
	Index int
	Sel   *gq.Selection
}

func (is indexedScraper) Find(id, desc string) (*gq.Selection, error) {
	sel := is.Sel.Find(fmt.Sprintf(`#%s\$%d`, id, is.Index))
	if sel.Length() != 1 {
		return nil, fmt.Errorf("could not find %s", desc)
	}
	return sel, nil
}
