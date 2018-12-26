package uwquest

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	gq "github.com/PuerkitoBio/goquery"
	ess "github.com/unixpickle/essentials"
)

// A CourseGrade represents the grades for a particular course.
type CourseGrade struct {
	Index        int
	Name         string
	Description  string
	GradingBasis string
	Units        *float32 // may be nil
	Grade        string
	GradePoints  *float32 // may be nil
}

func (cg *CourseGrade) String() string {
	sb := new(strings.Builder)
	fmt.Fprintf(sb, "CourseGrade{Index: %d, Name: %s, Description: %s, "+
		"GradingBasis: %s, Units: ", cg.Index, cg.Name, cg.Description,
		cg.GradingBasis)

	if cg.Units == nil {
		sb.WriteString("<nil>")
	} else {
		fmt.Fprintf(sb, "%f", *cg.Units)
	}
	fmt.Fprintf(sb, ", Grade: %s, GradePoints: ", cg.Grade)

	if cg.GradePoints == nil {
		sb.WriteString("<nil>")
	} else {
		fmt.Fprintf(sb, "%f", *cg.GradePoints)
	}
	sb.WriteByte('}')
	return sb.String()
}

// Grades fetches the grades for a particular term.
func (c *Client) Grades(termIndex int) ([]*CourseGrade, error) {
	// Scrape hidden fields from Quest grades page.
	res, err := c.Session.Get(GradesURL)
	if err != nil {
		return nil, ess.AddCtx("uwquest: fetching grades page", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("uwquest: got non-200 status code while fetching "+
			"grades page: got code %d", res.StatusCode)
	}
	defer res.Body.Close()

	// Make request form.
	form, err := scrapeHiddenFields(res.Body)
	if err != nil {
		return nil, ess.AddCtx("uwquest: scraping hidden fields on grades page",
			err)
	}
	if err = res.Body.Close(); err != nil {
		return nil, ess.AddCtx("uwquest: closing response body", err)
	}

	// Set custom form fields.
	form.Set("ICAJAX", "1")
	form.Set("ICNAVTYPEDROPDOWN", "0")
	form.Set("ICAction", "UW_DRVD_SSS_SCT_SSR_PB_GO")
	form.Set("DERIVED_SSTSNAV_SSTS_MAIN_GOTO$27$", "9999")
	form.Set("SSR_DUMMY_RECV1$sels$1$$0", strconv.Itoa(termIndex))
	body := strings.NewReader(form.Encode())

	// Create and send request.
	req, err := http.NewRequest("POST", GradesURL, body)
	if err != nil {
		return nil, ess.AddCtx("uwquest: creating grades request", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err = c.Session.Do(req)
	if err != nil {
		return nil, ess.AddCtx("uwquest: fetching grades", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("uwquest: got non-200 status code while fetching "+
			"grades: got code %d", res.StatusCode)
	}
	defer res.Body.Close()

	// Scrape response for grades table.
	doc, err := gq.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, ess.AddCtx("uwquest: parsing response body with goquery", err)
	}
	sel := doc.Find(`#TERM_CLASSES\$scroll\$0`).Find("table.PSLEVEL1GRID")
	if sel.Length() != 1 {
		return nil, errors.New("uwquest: could not locate grades table")
	}
	sel = sel.Children()

	var grades []*CourseGrade
	sel.Children().EachWithBreak(func(i int, row *gq.Selection) bool {
		if _, ok := row.Attr("id"); !ok {
			return true // continue
		}

		var grade *CourseGrade
		if grade, err = parseGradeRow(row); err != nil {
			ess.AddCtxTo(fmt.Sprintf("row %d", i), &err)
			return false
		}

		grades = append(grades, grade)
		return true
	})
	if err != nil {
		return nil, ess.AddCtx("uwquest: parsing grades table", err)
	}

	err = res.Body.Close()
	return grades, ess.AddCtx("uwquest: closing response body", err)
}

func parseGradeRow(row *gq.Selection) (*CourseGrade, error) {
	const nbsp = "\u00a0"
	var (
		cg     = new(CourseGrade)
		id, ok = row.Attr("id")
	)
	if !ok {
		return nil, errors.New("row does not contain an 'id' attribute")
	}
	cg.Index = int(id[len(id)-1]-'0') - 1

	var (
		scraper  = indexedScraper{Index: cg.Index, Sel: row}
		sel, err = scraper.Find(`CLS_LINK\$span`, "course name")
	)
	if err != nil {
		return nil, err
	}
	cg.Name = sel.Text()

	sel, err = scraper.Find("CLASS_TBL_VW_DESCR", "course description")
	if err != nil {
		return nil, err
	}
	cg.Description = sel.Text()

	sel, err = scraper.Find("STDNT_ENRL_SSV1_UNT_TAKEN", "course units")
	if err != nil {
		return nil, err
	}
	if text := sel.Text(); text != nbsp {
		u64, err := strconv.ParseFloat(text, 32)
		if err != nil {
			return nil, ess.AddCtx("parsing course units string into float", err)
		}
		u32 := float32(u64)
		cg.Units = &u32
	}

	if sel, err = scraper.Find("GRADING_BASIS", "grading basis"); err != nil {
		return nil, err
	}
	cg.GradingBasis = sel.Text()

	sel, err = scraper.Find("STDNT_ENRL_SSV1_CRSE_GRADE_OFF", "grade")
	if err != nil {
		return nil, err
	}
	if text := sel.Text(); text != nbsp {
		cg.Grade = sel.Text()
	}

	sel, err = scraper.Find("STDNT_ENRL_SSV1_GRADE_POINTS", "grade points")
	if err != nil {
		return nil, err
	}
	if text := sel.Text(); text != nbsp {
		p64, err := strconv.ParseFloat(text, 32)
		if err != nil {
			return nil, ess.AddCtx("parsing grade points string into float", err)
		}
		p32 := float32(p64)
		cg.GradePoints = &p32
	}

	return cg, nil
}
