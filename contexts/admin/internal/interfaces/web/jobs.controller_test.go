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
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/web"
)

func TestJobsController_Index(t *testing.T) { //nolint:dupl
	t.Parallel()

	echoRouter := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		handler := web.NewJobsController(nil, application.App{ListAllQueues: app.TestSuccessQueryHandler[application.ListAllQueuesQuery, application.ListAllQueuesResponse]()}, nil, nil)

		if assert.NoError(t, handler.Index()(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		handler := web.NewJobsController(nil, application.App{ListAllQueues: app.TestFailureQueryHandler[application.ListAllQueuesQuery, application.ListAllQueuesResponse]()}, nil, nil)

		assert.Error(t, handler.Index()(c))
	})
}

func TestJobsController_ShowQueue(t *testing.T) { //nolint:dupl
	t.Parallel()

	echoRouter := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		handler := web.NewJobsController(nil, application.App{GetQueue: app.TestSuccessQueryHandler[application.GetQueueQuery, application.GetQueueResponse]()}, nil, nil)

		if assert.NoError(t, handler.ShowQueue()(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		handler := web.NewJobsController(nil, application.App{GetQueue: app.TestFailureQueryHandler[application.GetQueueQuery, application.GetQueueResponse]()}, nil, nil)

		assert.Error(t, handler.ShowQueue()(c))
	})
}

func TestJobsController_ListWorkers(t *testing.T) { //nolint:dupl
	t.Parallel()

	echoRouter := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		handler := web.NewJobsController(nil, application.App{GetWorkers: app.TestSuccessQueryHandler[application.GetWorkersQuery, application.GetWorkersResponse]()}, nil, nil)

		if assert.NoError(t, handler.ListWorkers()(c)) {
			assert.Equal(t, http.StatusOK, rec.Code)
		}
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		handler := web.NewJobsController(nil, application.App{GetWorkers: app.TestFailureQueryHandler[application.GetWorkersQuery, application.GetWorkersResponse]()}, nil, nil)

		assert.Error(t, handler.ListWorkers()(c))
	})
}

func TestJobsController_DeleteJob(t *testing.T) {
	t.Parallel()

	echoRouter := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(req, rec)

		c.SetPath("/:queue/delete/:job_id")
		c.SetParamNames("queue", "job_id")
		c.SetParamValues("Default", "1337")

		handler := web.NewJobsController(nil, application.App{DeleteJob: app.TestSuccessCommandHandler[application.DeleteJobCommand]()}, nil, nil)

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

		handler := web.NewJobsController(nil, application.App{DeleteJob: app.TestFailureCommandHandler[application.DeleteJobCommand]()}, nil, nil)

		if assert.NoError(t, handler.DeleteJob()(c)) {
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Empty(t, rec.Body)
		}
	})
}

func TestJobsController_DeleteHistory(t *testing.T) {
	t.Parallel()

	echoRouter := newTestRouter()

	// set http POST payload
	reqBody := url.Values{}
	reqBody.Set("days", "all")

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		validRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody.Encode()))
		validRequest.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(validRequest, rec)

		handler := web.NewJobsController(nil, application.App{
			PruneJobHistory: app.TestSuccessRequestHandler[application.PruneJobHistoryRequest, application.PruneJobHistoryResponse](),
		}, nil, nil)

		assert.NoError(t, handler.DeleteHistory()(c))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "arrower:admin.jobs.history.deleted", rec.Header().Get("HX-Trigger"))
	})

	t.Run("usecase failure", func(t *testing.T) {
		t.Parallel()

		validRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody.Encode()))
		validRequest.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)

		rec := httptest.NewRecorder()
		c := echoRouter.NewContext(validRequest, rec)

		handler := web.NewJobsController(nil, application.App{
			PruneJobHistory: app.TestFailureRequestHandler[application.PruneJobHistoryRequest, application.PruneJobHistoryResponse](),
		}, nil, nil)

		assert.NoError(t, handler.DeleteHistory()(c))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "", rec.Header().Get("HX-Trigger"))
	})
}
