package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/go-arrower/arrower/repository"
)

type TestCasesController struct {
	repo *repository.MemoryRepository[testcase, string]
}

// showTestCase is a fat controller rendering a specific testcase.
// If no testcase is selected it shows the first one.
// If no run of a testcase is selected it shows the most recent one.
func (cont TestCasesController) showTestCase() func(c echo.Context) error { //nolint:funlen,gocognit,gocyclo,cyclop,lll // fat controller
	return func(c echo.Context) error {
		tcName := strings.TrimSpace(c.QueryParam("tc"))
		vParam := strings.TrimSpace(c.QueryParam("run"))

		all, err := cont.repo.All(c.Request().Context())
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		if len(all) == 0 {
			return c.String(http.StatusOK, "No tests yet. Run your unit tests and they will show up here.")
		}

		slices.SortFunc(all, func(a, b testcase) int { // sort alphabetically
			if a.Name < b.Name {
				return -1
			}

			return 1
		})

		if tcName == "" { // if no case is loaded, choose the first one
			tcName = all[0].Name
		}

		test, err := cont.repo.Read(c.Request().Context(), tcName)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		var (
			tcNames  []string
			next     string
			previous string
		)

		for i, tc := range all {
			tcNames = append(tcNames, tc.Name)

			if tc.Name == test.Name {
				length := len(all)
				if i+1 < length {
					next = all[i+1].Name
				}

				if i != 0 {
					previous = all[i-1].Name
				}
			}
		}

		var (
			runNames   []string
			curRunName string
			curRun     run
		)

		for name, run := range test.Runs {
			runNames = append(runNames, name)

			if name == vParam {
				curRun = run
				curRunName = name
			}
		}

		sort.Sort(sort.Reverse(sort.StringSlice(runNames))) // most recent first

		if reflect.DeepEqual(curRun, run{}) { //nolint:exhaustruct // if no run is loaded, choose the first one
			curRun = test.Runs[runNames[0]]
			curRunName = runNames[0]
		}

		return c.Render(http.StatusOK, "testcases", echo.Map{
			"TestcaseNames":       tcNames,
			"CurrentTestcaseName": test.Name,
			"RunNames":            runNames,
			"CurrentRunName":      curRunName,
			"CurrentRun":          curRun,
			"Next":                next,
			"Previous":            previous,
		})
	}
}

// storeTestcase is a fat controller receiving new runs.
// It implements an upsert semantic: creating a new testcase whenever it first sees a new run for it.
func (cont TestCasesController) storeTestcase() func(c echo.Context) error {
	type testRunBody struct {
		TestName string

		RunName  string
		Template template.HTML
		Data     string
		HTML     template.HTML
	}

	return func(c echo.Context) error {
		var testRun testRunBody

		err := c.Bind(&testRun)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		test, err := cont.repo.Read(c.Request().Context(), testRun.TestName)
		if errors.Is(err, repository.ErrNotFound) { // first time this test case is seen => create it
			test = testcase{Name: testRun.TestName, Runs: make(map[string]run)}
		} else if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		test.Runs[testRun.RunName] = run{
			Template:   testRun.Template,
			Data:       prettyJSON([]byte(testRun.Data)),
			HTML:       testRun.HTML,
			Assertions: make([]assertion, 0),
		}

		err = cont.repo.Save(c.Request().Context(), test)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		return nil
	}
}

// storeAssertion stores individual assertions for an existing testcase and run.
func (cont TestCasesController) storeAssertion() func(c echo.Context) error {
	type testAssertion struct {
		TestName string
		RunName  string

		Name string
		Args []any
		Pass bool
	}

	return func(c echo.Context) error {
		var ass testAssertion

		err := c.Bind(&ass)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		test, err := cont.repo.Read(c.Request().Context(), ass.TestName)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		runs := test.Runs[ass.RunName]
		runs.Assertions = append(runs.Assertions, assertion{
			Name: ass.Name,
			Args: ass.Args,
			Pass: ass.Pass,
		})
		test.Runs[ass.RunName] = runs

		err = cont.repo.Save(c.Request().Context(), test)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		return nil
	}
}

func prettyJSON(str []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, str, "", "  "); err != nil {
		return ""
	}

	return prettyJSON.String()
}

type (
	// testcase is the model of a test executed multiple times by the developer.
	testcase struct {
		Runs map[string]run
		Name string
	}

	run struct {
		Template   template.HTML
		Data       string
		HTML       template.HTML
		Assertions []assertion
	}

	assertion struct {
		Name string
		Args []any // TODO separate the Args / msgAndArgs from the failure message, to make an assertion more speaking in the UI
		Pass bool
	}
)
