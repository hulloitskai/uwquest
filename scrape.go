package uwquest

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"reflect"

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

// unmarshallHTML unmarshalls sel into the struct that v points to, according to
// v's struct tags.
func unmarshallHTML(sel *gq.Selection, v interface{}) error {
	// Validate v's type.
	t := reflect.TypeOf(v)
	if (t.Kind() != reflect.Ptr) || (t.Elem().Kind() != reflect.Struct) {
		return errors.New("can only unmarshal HTML into a struct pointer")
	}

	var (
		elem = t.Elem()
		val  = reflect.ValueOf(v)
	)
	for i := 0; i < elem.NumField(); i++ {
		var (
			ft  = elem.Field(i)
			tag = ft.Tag.Get("selector")
		)
		if tag == "" { // skip fields without a selector tag
			continue
		}
		if ft.Type.Kind() != reflect.String {
			return errors.New("can only unmarshal HTML text into a string field")
		}

		res := sel.Find(tag)
		if res.Length() == 0 {
			return fmt.Errorf("could not find element by the selection '%s'", tag)
		}

		fv := val.Field(i)
		fv.SetString(res.First().Text())
	}

	return nil
}
