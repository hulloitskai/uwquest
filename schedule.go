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

// CourseSchedule represents the course schedule for a particular course.
type CourseSchedule struct {
	Index        int
	Name         string
	Status       string
	Units        float32
	GradingBasis string
	Classes      []*Class
}

func (cs *CourseSchedule) String() string {
	return fmt.Sprintf("CourseSchedule{Index: %d, Name: %s, Status: %s, "+
		"Units: %f, GradingBasis: %s, Classes: %v}", cs.Index, cs.Name, cs.Status,
		cs.Units, cs.GradingBasis, cs.Classes)
}

// Class represents a class within a particular course.
type Class struct {
	Index           int
	Number, Section int
	Component       string
	Schedule        string
	Location        string
	Instructor      string
	StartEndDate    string
}

func (c *Class) String() string {
	return fmt.Sprintf("Class{Index: %d, Number: %d, Section: %d, "+
		"Component: %s, Schedule: %s, Location: %s, Instructor: %s, "+
		"StartEndDate: %s", c.Index, c.Number, c.Section, c.Component, c.Schedule,
		c.Location, c.Instructor, c.StartEndDate)
}

// Schedules fetches course schedules for a particular term.
func (c *Client) Schedules(termIndex int) ([]*CourseSchedule, error) {
	// Scrape hidden fields from Quest grades page.
	res, err := c.Session.Get(SchedulesURL)
	if err != nil {
		return nil, ess.AddCtx("uwquest: fetching course schedule page", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("uwquest: got non-200 status code while fetching "+
			"course schedule page: got code %d", res.StatusCode)
	}
	defer res.Body.Close()

	// Make request form.
	form, err := scrapeHiddenFields(res.Body)
	if err != nil {
		return nil, ess.AddCtx("uwquest: scraping hidden fields on course "+
			"schedule page", err)
	}
	if err = res.Body.Close(); err != nil {
		return nil, ess.AddCtx("uwquest: closing response body", err)
	}

	// Set custom form fields.
	form.Set("ICAJAX", "1")
	form.Set("ICNAVTYPEDROPDOWN", "1")
	form.Set("ICAction", "DERIVED_SSS_SCT_SSR_PB_GO")
	form.Set("DERIVED_SSTSNAV_SSTS_MAIN_GOTO$27$", "9999")
	form.Set("SSR_DUMMY_RECV1$sels$0$$0", strconv.Itoa(termIndex))
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

	// Scrape schedule data from response body.
	doc, err := gq.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, ess.AddCtx("uwquest: parsing response body with goquery", err)
	}

	schedules, err := parseSchedules(doc.Selection)
	if err != nil {
		return nil, ess.AddCtx("uwquest: parsing schedule", err)
	}

	err = res.Body.Close()
	return schedules, ess.AddCtx("uwquest: closing response body", err)
}

// parseSchedules parses the schedules section of the course schedules page
// into a set of CourseSchedules.
func parseSchedules(sel *gq.Selection) ([]*CourseSchedule, error) {
	sel = sel.Find(`#ACE_STDNT_ENRL_SSV2\$0`).Children()
	if sel.Length() != 1 {
		return nil, errors.New("could not find schedule container table")
	}

	sel = sel.Children().Find("table.PSGROUPBOXWBO")
	if sel.Length() == 0 {
		return nil, errors.New("could not find schedule tables")
	}

	var (
		schedules   []*CourseSchedule
		classOffset int
		err         error
	)
	sel.EachWithBreak(func(i int, table *gq.Selection) bool {
		var cs *CourseSchedule
		if cs, err = parseScheduleTable(table, classOffset); err != nil {
			ess.AddCtxTo(fmt.Sprintf("table %d", i), &err)
			return false
		}

		schedules = append(schedules, cs)
		classOffset += len(cs.Classes)
		return true
	})

	return schedules, ess.AddCtx("parsing schedules table", err)
}

// parseScheduleTable parses a course schedule table into a CourseSchedule.
func parseScheduleTable(table *gq.Selection, classOffset int) (
	*CourseSchedule, error) {
	var (
		cs  = new(CourseSchedule)
		sel = table.Find("table.PSGROUPBOX")
	)
	if sel.Length() != 1 {
		return nil, errors.New("could not find inner table")
	}
	id, ok := sel.Attr("id")
	if !ok {
		return nil, errors.New("inner table does not have an 'id' attribute")
	}
	cs.Index = int(id[len(id)-1] - '0')

	// Parse course name from table divider.
	if sel = table.Find("td.PAGROUPDIVIDER"); sel.Length() != 1 {
		return nil, errors.New("could not find course name")
	}
	cs.Name = sel.Text()

	// Parse course info from header row.
	row := table.Find(fmt.Sprintf(`#trSSR_DUMMY_RECVW\$%d_row1`, cs.Index))
	if row.Length() != 1 {
		return nil, errors.New("could not find header info row")
	}

	scraper := indexedScraper{Index: cs.Index, Sel: row}
	sel, err := scraper.Find("STATUS", "course status")
	if err != nil {
		return nil, err
	}
	cs.Status = sel.Text()

	sel, err = scraper.Find("DERIVED_REGFRM1_UNT_TAKEN", "units taken")
	if err != nil {
		return nil, err
	}
	units64, err := strconv.ParseFloat(sel.Text(), 32)
	if err != nil {
		return nil, ess.AddCtx("couldn't parse units text into float", err)
	}
	cs.Units = float32(units64)

	if sel, err = scraper.Find("GB_DESCR", "grading basis"); err != nil {
		return nil, err
	}
	cs.GradingBasis = sel.Text()

	// Parse data from classes table.
	ctable := table.Find(fmt.Sprintf(`#CLASS_MTG_VW\$scroll\$%d`, cs.Index)).
		Find("table.PSLEVEL3GRID").Children()
	if ctable.Length() != 1 {
		return nil, errors.New("could not locate classes table")
	}

	ctable.Children().EachWithBreak(func(i int, row *gq.Selection) bool {
		if _, ok := row.Attr("id"); !ok {
			return true // continue
		}

		var class *Class
		if class, err = parseClassRow(row, classOffset); err != nil {
			ess.AddCtxTo(fmt.Sprintf("row %d (offset %d)", i, classOffset), &err)
			return false
		}

		cs.Classes = append(cs.Classes, class)
		return true
	})

	return cs, ess.AddCtx("uwquest: parsing classes table", err)
}

// parseClassRow parses a class row within a course schedule table into a
// Class.
func parseClassRow(row *gq.Selection, offset int) (*Class, error) {
	const nbsp = "\u00a0"
	var (
		class  = new(Class)
		id, ok = row.Attr("id")
	)
	if !ok {
		return nil, errors.New("row does not have an 'id' attribute")
	}
	class.Index = int(id[len(id)-1]-'0') - 1 + offset

	var (
		scraper  = indexedScraper{Index: class.Index, Sel: row}
		sel, err = scraper.Find("DERIVED_CLS_DTL_CLASS_NBR", "class number")
	)
	if err != nil {
		return nil, err
	}
	if class.Number, err = strconv.Atoi(sel.Text()); err != nil {
		return nil, ess.AddCtx("could not parse class number into int", err)
	}

	if sel, err = scraper.Find("MTG_SECTION", "class section"); err != nil {
		return nil, err
	}
	if class.Section, err = strconv.Atoi(sel.Text()); err != nil {
		return nil, ess.AddCtx("could not parse class section into int", err)
	}

	if sel, err = scraper.Find("MTG_COMP", "course component"); err != nil {
		return nil, err
	}
	class.Component = sel.Text()

	if sel, err = scraper.Find("MTG_SCHED", "class schedule"); err != nil {
		return nil, err
	}
	if text := sel.Text(); text != nbsp {
		class.Schedule = text
	}

	if sel, err = scraper.Find("MTG_LOC", "class location"); err != nil {
		return nil, err
	}
	class.Location = strings.Replace(sel.Text(), nbsp, "", -1)

	sel, err = scraper.Find("DERIVED_CLS_DTL_SSR_INSTR_LONG", "instructor")
	if err != nil {
		return nil, err
	}
	class.Instructor = sel.Text()

	if sel, err = scraper.Find("MTG_DATES", "'Start/End Date'"); err != nil {
		return nil, err
	}
	class.StartEndDate = sel.Text()

	return class, nil
}
