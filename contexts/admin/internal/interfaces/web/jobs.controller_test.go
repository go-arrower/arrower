package web_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/app"
	"github.com/go-arrower/arrower/contexts/admin/internal/application"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/web"
)

func TestJobsController_Index(t *testing.T) { //nolint:dupl
	t.Parallel()

	echoRouter := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		handler := web.NewJobsController(nil, nil, application.App{ListAllQueues: app.TestSuccessQueryHandler[application.ListAllQueuesQuery, application.ListAllQueuesResponse]()}, nil, nil)

		if assert.NoError(t, handler.Index()(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		handler := web.NewJobsController(nil, nil, application.App{ListAllQueues: app.TestFailureQueryHandler[application.ListAllQueuesQuery, application.ListAllQueuesResponse]()}, nil, nil)

		assert.Error(t, handler.Index()(c))
	})
}

func TestJobsController_ShowQueue(t *testing.T) { //nolint:dupl
	t.Parallel()

	echoRouter := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		handler := web.NewJobsController(nil, nil, application.App{GetQueue: app.TestSuccessQueryHandler[application.GetQueueQuery, application.GetQueueResponse]()}, nil, nil)

		if assert.NoError(t, handler.ShowQueue()(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		handler := web.NewJobsController(nil, nil, application.App{GetQueue: app.TestFailureQueryHandler[application.GetQueueQuery, application.GetQueueResponse]()}, nil, nil)

		assert.Error(t, handler.ShowQueue()(c))
	})
}

func TestJobsController_ListWorkers(t *testing.T) { //nolint:dupl
	t.Parallel()

	echoRouter := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		handler := web.NewJobsController(nil, nil, application.App{GetWorkers: app.TestSuccessQueryHandler[application.GetWorkersQuery, application.GetWorkersResponse]()}, nil, nil)

		if assert.NoError(t, handler.ListWorkers()(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		handler := web.NewJobsController(nil, nil, application.App{GetWorkers: app.TestFailureQueryHandler[application.GetWorkersQuery, application.GetWorkersResponse]()}, nil, nil)

		assert.Error(t, handler.ListWorkers()(c))
	})
}

func TestJobsController_DeleteJob(t *testing.T) {
	t.Parallel()

	echoRouter := newTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		c.SetPath("/:queue/delete/:job_id")
		c.SetParamNames("queue", "job_id")
		c.SetParamValues("Default", "1337")

		handler := web.NewJobsController(nil, nil, application.App{DeleteJob: app.TestSuccessCommandHandler[application.DeleteJobCommand]()}, nil, nil)

		if assert.NoError(t, handler.DeleteJob()(c)) {
			assert.Equal(t, http.StatusSeeOther, rec.Code)
			assert.Equal(t, "/admin/jobs/Default", rec.Header().Get(echo.HeaderLocation))
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		c.SetPath("/:queue/delete/:job_id")
		c.SetParamNames("queue", "job_id")
		c.SetParamValues("Default", "1337")

		handler := web.NewJobsController(nil, nil, application.App{DeleteJob: app.TestFailureCommandHandler[application.DeleteJobCommand]()}, nil, nil)

		if assert.NoError(t, handler.DeleteJob()(c)) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Empty(t, rec.Body)
		}
	})
}

func TestJobsController_DeleteHistory(t *testing.T) {
	t.Parallel()

	echoRouter := newTestRouter(t)

	// set http POST payload
	reqBody := url.Values{}
	reqBody.Set("days", "all")

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		validRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody.Encode()))
		validRequest.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(validRequest, rec)

		handler := web.NewJobsController(nil, nil, application.App{
			PruneJobHistory: app.TestSuccessRequestHandler[application.PruneJobHistoryRequest, application.PruneJobHistoryResponse](),
		}, nil, nil)

		assert.NoError(t, handler.DeleteHistory()(c))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "arrower:admin.jobs.history.deleted", rec.Header().Get("Hx-Trigger"))
	})

	t.Run("usecase failure", func(t *testing.T) {
		t.Parallel()

		validRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody.Encode()))
		validRequest.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(validRequest, rec)

		handler := web.NewJobsController(nil, nil, application.App{
			PruneJobHistory: app.TestFailureRequestHandler[application.PruneJobHistoryRequest, application.PruneJobHistoryResponse](),
		}, nil, nil)

		assert.NoError(t, handler.DeleteHistory()(c))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Empty(t, rec.Header().Get("Hx-Trigger"))
	})
}

func TestJobsController_ScheduleJobs(t *testing.T) {
	t.Parallel()

	echoRouter := newTestRouter(t)

	t.Run("validate", func(t *testing.T) {
		t.Parallel()

		//nolint:nlreturn,wsl_v5
		tests := map[string]struct {
			body        url.Values
			errContains string
		}{
			"priority too high": {
				body: func() url.Values {
					reqBody := url.Values{}
					reqBody.Set("runAt-time", "2026-04-01T09:00")
					reqBody.Set("job-type", "my-job")
					reqBody.Set("payload", "{}")
					reqBody.Set("count", "1")
					reqBody.Set("priority", "-32768") // max int16 + 1; controller transforms *=-1 into linux priorities
					return reqBody
				}(),
				errContains: `'Priority' failed on the 'max' tag`,
			},
			"runAt date format wrong 0": {
				body: func() url.Values {
					reqBody := url.Values{}
					reqBody.Set("runAt-time", "1337")
					return reqBody
				}(),
				errContains: "cannot parse",
			},
			"runAt date format wrong 1": {
				body: func() url.Values {
					reqBody := url.Values{}
					reqBody.Set("runAt-time", "2006-01-02T15:04:05Z07:00")
					return reqBody
				}(),
				errContains: "parsing time",
			},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body.Encode()))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

				rec := httptest.NewRecorder()
				c := echoRouter.NewContext(req, rec)

				var called bool

				handler := web.NewJobsController(nil, nil, application.App{
					ScheduleJobs: application.NewScheduleJobsCommandHandler(models.New(nil)),
				}, nil, nil)

				err := handler.ScheduleJobs()(c)
				assert.Error(t, err)
				t.Log(err.Error())
				t.Log(err)
				t.Log(err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.False(t, called)
				t.Log(err)
			})
		}
	})
}
