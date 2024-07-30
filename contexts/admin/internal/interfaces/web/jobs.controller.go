package web

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/contexts/admin/internal/application"
	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
	"github.com/go-arrower/arrower/contexts/admin/internal/interfaces/repository/models"
	"github.com/go-arrower/arrower/contexts/admin/internal/views/pages"
)

const (
	historyTableSizeChangedJSEvent   = "arrower:admin.jobs.history.deleted"
	finishedJobsFilterChangedJSEvent = "arrower:admin.jobs.filter.changed"

	htmlDatetimeLayout = "2006-01-02T15:04" // format used by the HTML datetime-local input element
)

const (
	// timeDay is a helper to represent time.Day, which the std lib does not define
	timeDay = time.Hour * 24
)

func NewJobsController(
	logger alog.Logger,
	appDI application.App,
	repo jobs.Repository,
	queries *models.Queries,
) *JobsController {
	return &JobsController{
		logger:  logger,
		appDI:   appDI,
		repo:    repo,
		queries: queries,
	}
}

type JobsController struct {
	logger alog.Logger

	appDI   application.App
	repo    jobs.Repository
	queries *models.Queries
}

func (jc *JobsController) Index() func(c echo.Context) error {
	return func(c echo.Context) error {
		res, err := jc.appDI.ListAllQueues.H(c.Request().Context(), application.ListAllQueuesQuery{})
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"could not list all queues").
				WithInternal(err)
		}

		return c.Render(http.StatusOK, "jobs.home", echo.Map{
			"Title":  "Queues",
			"Queues": res.QueueStats,
		})
	}
}

func (jc *JobsController) PendingJobsPieChartData() func(echo.Context) error {
	type pieData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	return func(c echo.Context) error {
		res, err := jc.appDI.ListAllQueues.H(c.Request().Context(), application.ListAllQueuesQuery{})
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"could not list all queues").
				WithInternal(err)
		}

		keys := []string{}

		for k := range res.QueueStats {
			keys = append(keys, string(k))
		}

		sort.Sort(sort.StringSlice(keys))

		var json []pieData
		for _, k := range keys {
			json = append(json, pieData{
				Name:  string(res.QueueStats[jobs.QueueName(k)].QueueName),
				Value: res.QueueStats[jobs.QueueName(k)].PendingJobs,
			})
		}

		return c.JSON(http.StatusOK, json)
	}
}

func (jc *JobsController) ProcessedJobsLineChartData() func(echo.Context) error {
	type lineData struct {
		XAxis  []string `json:"xAxis"`
		Series []int    `json:"series"`
	}

	return func(c echo.Context) error {
		interval := c.Param("interval")

		const (
			bucketsPerHour = 12
			bucketsPerWeek = 7
		)

		var (
			jobData []models.PendingJobsRow
			err     error
		)

		if interval == "hour" { // show last 60 minutes
			jobData, err = jc.queries.PendingJobs(c.Request().Context(), models.PendingJobsParams{
				DateBin: pgtype.Interval{Valid: true, Microseconds: int64(time.Minute * 5 / time.Microsecond)},
				FinishedAt: pgtype.Timestamptz{
					Valid:            true,
					Time:             time.Now().UTC().Add(-time.Hour),
					InfinityModifier: pgtype.Finite,
				},
				Limit: bucketsPerHour,
			})
		} else {
			jobData, err = jc.queries.PendingJobs(c.Request().Context(), models.PendingJobsParams{ // show whole week
				DateBin: pgtype.Interval{Valid: true, Days: 1},
				FinishedAt: pgtype.Timestamptz{
					Valid:            true,
					Time:             time.Now().UTC().Add(-timeDay * bucketsPerWeek),
					InfinityModifier: pgtype.Finite,
				},
				Limit: bucketsPerWeek,
			})
		}

		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"could not get pending jobs").
				WithInternal(err)
		}

		var (
			xaxis  []string
			series []int
		)

		for _, data := range jobData {
			if interval == "hour" {
				xaxis = append([]string{data.T.Time.Format("15:04")}, xaxis...)
			} else {
				xaxis = append([]string{data.T.Time.Format("01.02")}, xaxis...)
			}

			series = append([]int{int(data.Count)}, series...)
		}

		return c.JSON(http.StatusOK, lineData{
			XAxis:  xaxis,
			Series: series,
		})
	}
}

func (jc *JobsController) ShowQueue() func(c echo.Context) error {
	return func(c echo.Context) error {
		queue := c.Param("queue")

		res, err := jc.appDI.GetQueue.H(c.Request().Context(), application.GetQueueQuery{QueueName: jobs.QueueName(queue)})
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"could not get queue").
				WithInternal(err)
		}

		page := pages.BuildQueuePage(queue, res.Jobs, res.Kpis)

		return c.Render(http.StatusOK, "jobs.queue",
			echo.Map{
				"Title":     page.QueueName + " queue",
				"QueueName": page.QueueName,
				"Jobs":      page.Jobs,
				"Stats":     page.Stats,
			})
	}
}

func (jc *JobsController) ListWorkers() func(c echo.Context) error {
	return func(c echo.Context) error {
		res, err := jc.appDI.GetWorkers.H(c.Request().Context(), application.GetWorkersQuery{})
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"could not get workers").
				WithInternal(err)
		}

		return c.Render(http.StatusOK, "jobs.workers", echo.Map{
			"Title":     "Workers",
			"workers":   pages.PresentWorkers(res.Pool),
			"schedules": res.Schedules,
		})
	}
}

func (jc *JobsController) DeleteJob() func(c echo.Context) error {
	return func(c echo.Context) error {
		queue := c.Param("queue")
		jobID := c.Param("job_id")

		err := jc.appDI.DeleteJob.H(c.Request().Context(), application.DeleteJobCommand{JobID: jobID})
		if err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		return c.Redirect(http.StatusSeeOther, "/admin/jobs/"+queue)
	}
}

func (jc *JobsController) RescheduleJob() func(c echo.Context) error {
	return func(c echo.Context) error {
		queue := c.Param("queue")
		jobID := c.Param("job_id")

		err := jc.repo.RunJobAt(c.Request().Context(), jobID, time.Now())
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"could not schedule job").
				WithInternal(err)
		}

		return c.Redirect(http.StatusSeeOther, "/admin/jobs/"+queue)
	}
}

// todo clean into proper application usecase.
func (jc *JobsController) ShowMaintenance() func(c echo.Context) error {
	return func(c echo.Context) error {
		size, _ := jc.queries.JobTableSize(c.Request().Context())

		res, _ := jc.appDI.ListAllQueues.H(c.Request().Context(), application.ListAllQueuesQuery{}) // fixme: don't call existing use case, create own or call domain model

		var queues []string

		for q, _ := range res.QueueStats {
			queue := string(q)
			if queue == "" {
				queue = string(jobs.DefaultQueueName)
			}

			queues = append(queues, queue)
		}

		return c.Render(http.StatusOK, "jobs.maintenance",
			echo.Map{
				"Title":   "Maintenance",
				"Jobs":    size.Jobs,
				"History": size.History,
				"Queues":  queues,
			},
		)
	}
}

func (jc *JobsController) VacuumJobTables() func(echo.Context) error {
	return func(c echo.Context) error {
		table := c.Param("table")

		size, err := jc.appDI.VacuumJobTable.H(c.Request().Context(), application.VacuumJobTableRequest{Table: table})
		if err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		// reload the dashboard badges with the size, by using htmx's oob technique
		return c.Render(http.StatusOK, "jobs.maintenance#table-size",
			echo.Map{
				"Jobs":    size.Jobs,
				"History": size.History,
			},
		)
	}
}

func (jc *JobsController) DeleteHistory() func(echo.Context) error {
	return func(c echo.Context) error {
		// valid days values: any number or "all", with "all" mapping to 0
		days, err := strconv.Atoi(c.FormValue("days"))
		if errors.Is(err, strconv.ErrSyntax) && c.FormValue("days") != "all" {
			return c.NoContent(http.StatusBadRequest)
		}

		size, err := jc.appDI.PruneJobHistory.H(
			c.Request().Context(),
			application.PruneJobHistoryRequest{Days: days},
		)
		if err != nil {
			return c.NoContent(http.StatusBadRequest)
		}

		// trigger size change of history table, for other size-estimation widgets to reload.
		c.Response().Header().Set("HX-Trigger", historyTableSizeChangedJSEvent)

		// reload the dashboard badges with the new size, by using htmx's oob technique.
		return c.Render(http.StatusOK, "jobs.maintenance#table-size",
			echo.Map{
				"Jobs":    size.Jobs,
				"History": size.History,
			},
		)
	}
}

func (jc *JobsController) PruneHistory() func(echo.Context) error {
	return func(c echo.Context) error {
		days, _ := strconv.Atoi(c.FormValue("days"))
		estimateBefore := time.Now().Add(-1 * time.Duration(days) * timeDay)

		queue := c.FormValue("queue")
		if queue == string(jobs.DefaultQueueName) {
			queue = ""
		}

		// todo check if pruneJobHistoryRequestHandler can be used here
		_ = jc.queries.PruneHistoryPayload(c.Request().Context(), models.PruneHistoryPayloadParams{
			Queue:     queue,
			CreatedAt: pgtype.Timestamptz{Time: estimateBefore, Valid: true, InfinityModifier: pgtype.Finite},
		})

		c.Response().Header().Set("HX-Trigger", historyTableSizeChangedJSEvent)

		return c.NoContent(http.StatusOK)
	}
}

func (jc *JobsController) EstimateHistorySize() func(echo.Context) error {
	return func(c echo.Context) error {
		days, _ := strconv.Atoi(c.QueryParam("days"))

		estimateBefore := time.Now().Add(-1 * time.Duration(days) * timeDay)

		size, _ := jc.queries.JobHistorySize(c.Request().Context(), pgtype.Timestamptz{Time: estimateBefore, Valid: true})

		var fmtSize string
		if size != "" {
			fmtSize = fmt.Sprintf("~ %s", size)
		}

		return c.String(http.StatusOK, fmtSize)
	}
}

func (jc *JobsController) EstimateHistoryPayloadSize() func(echo.Context) error {
	return func(c echo.Context) error {
		queue := c.QueryParam("queue")
		if queue == "Default" {
			queue = ""
		}

		days, _ := strconv.Atoi(c.QueryParam("days"))
		estimateBefore := time.Now().Add(-1 * time.Duration(days) * timeDay)

		size, _ := jc.queries.JobHistoryPayloadSize(c.Request().Context(), models.JobHistoryPayloadSizeParams{
			Queue:     queue,
			CreatedAt: pgtype.Timestamptz{Time: estimateBefore, Valid: true, InfinityModifier: pgtype.Finite},
		})

		var fmtSize string
		if size != "" {
			fmtSize = fmt.Sprintf("~ %s", size)
		}

		return c.String(http.StatusOK, fmtSize)
	}
}

func (jc *JobsController) CreateJobs() func(c echo.Context) error {
	return func(c echo.Context) error {
		queues, _ := jc.repo.FindAllQueueNames(c.Request().Context())

		jobType, _ := jc.appDI.JobTypesForQueue.H(
			c.Request().Context(),
			application.JobTypesForQueueQuery{Queue: jobs.DefaultQueueName},
		)

		year, month, day := time.Now().Date()

		return c.Render(http.StatusOK, "jobs.schedule",
			echo.Map{
				"Title":    "Schedule a Job",
				"Queues":   queues,
				"JobTypes": jobType,
				"RunAt":    time.Now().Format(htmlDatetimeLayout),
				"RunAtMin": fmt.Sprintf("%d-%02d-%02dT00:00", year, month, day),
			},
		)
	}
}

func (jc *JobsController) ShowJobTypes() func(_ echo.Context) error {
	return func(c echo.Context) error {
		queue := c.QueryParam("queue")

		jobType, _ := jc.appDI.JobTypesForQueue.H(
			c.Request().Context(),
			application.JobTypesForQueueQuery{Queue: jobs.QueueName(queue)},
		)

		return c.Render(http.StatusOK, "jobs.schedule#known-job-types", echo.Map{
			"JobTypes": jobType,
		})
	}
}

func (jc *JobsController) PayloadExamples() func(_ echo.Context) error {
	return func(c echo.Context) error {
		queue := c.QueryParam("queue")
		jobType := c.QueryParam("job-type")

		if jobs.QueueName(queue) == jobs.DefaultQueueName {
			queue = ""
		}

		payloads, _ := jc.queries.LastHistoryPayloads(c.Request().Context(), models.LastHistoryPayloadsParams{
			Queue:   queue,
			JobType: jobType,
		})

		return c.Render(http.StatusOK, "jobs.schedule#payload-examples",
			pages.PresentJobsExamplePayloads(queue, jobType, payloads))
	}
}

func (jc *JobsController) ScheduleJobs() func(c echo.Context) error {
	return func(c echo.Context) error {
		queue := c.FormValue("queue")
		jt := c.FormValue("job-type")
		prio := c.FormValue("priority")
		payload := c.FormValue("payload")
		num := c.FormValue("count")
		runAtParam := c.FormValue("runAt-time")

		jq := queue
		if queue == "Default" {
			jq = ""
		}

		priority, err := strconv.Atoi(prio)
		if err != nil {
			return echo.NewHTTPError(
				http.StatusBadRequest,
				"could not parse priority").
				WithInternal(err)
		}

		priority *= -1

		count, err := strconv.Atoi(num)
		if err != nil {
			return echo.NewHTTPError(
				http.StatusBadRequest,
				"could not parse count").
				WithInternal(err)
		}

		runAt, _ := time.Parse(htmlDatetimeLayout, runAtParam)

		err = jc.appDI.ScheduleJobs.H(c.Request().Context(), application.ScheduleJobsCommand{
			Queue:    jq,
			JobType:  jt,
			Priority: int16(priority),
			Payload:  payload,
			Count:    count,
			RunAt:    runAt.Add(-1 * time.Hour), // todo needs to apply read tz, to prevent dirty hack, to overcome client and server times
		})
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"could not schedule job").
				WithInternal(err)
		}

		return c.Redirect(http.StatusSeeOther, "/admin/jobs/"+queue)
	}
}

func (jc *JobsController) FinishedJobs() func(echo.Context) error {
	return func(c echo.Context) error {
		if updateJobTypeSelectOptions := c.QueryParam("updateJobTypes"); updateJobTypeSelectOptions == "true" {
			q := c.QueryParam("queue")
			if q == string(jobs.DefaultQueueName) {
				q = ""
			}

			jobTypes, err := jc.queries.JobTypes(c.Request().Context(), q)
			if err != nil {
				return echo.NewHTTPError(
					http.StatusInternalServerError,
					"could not get job types").
					WithInternal(err)
			}

			return c.Render(http.StatusOK, "jobs.finished#known-job-types", echo.Map{
				"JobType":  jobTypes,
				"Selected": c.QueryParam("job-type"),
			})
		}

		filter := jobs.Filter{ // todo, see if echo can autobind to this; same for total count controller
			Queue:   jobs.QueueName(c.QueryParam("queue")),
			JobType: jobs.JobType(c.QueryParam("job-type")),
		}

		finishedJobs, err := jc.repo.FinishedJobs(c.Request().Context(), filter)
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"could not get finished jobs").
				WithInternal(err)
		}

		if filter != (jobs.Filter{}) {
			c.Response().Header().Set("HX-TRIGGER", finishedJobsFilterChangedJSEvent)

			return c.Render(http.StatusOK, "jobs.finished#jobs.list", pages.NewFinishedJobs(finishedJobs, nil))
		}

		queues, err := jc.repo.FindAllQueueNames(c.Request().Context())
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"could not get queues").
				WithInternal(err)
		}

		return c.Render(http.StatusOK, "jobs.finished", pages.NewFinishedJobs(finishedJobs, queues))
	}
}

func (jc *JobsController) FinishedJobsTotal() func(ctx echo.Context) error {
	return func(c echo.Context) error {
		filter := jobs.Filter{
			Queue:   jobs.QueueName(c.QueryParam("queue")),
			JobType: jobs.JobType(c.QueryParam("job-type")),
		}

		total, err := jc.repo.FinishedJobsTotal(c.Request().Context(), filter)
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"could not get finished jobs count").
				WithInternal(err)
		}

		return c.String(http.StatusOK, strconv.FormatInt(total, 10))
	}
}

func (jc *JobsController) JobShow() func(ctx echo.Context) error {
	return func(c echo.Context) error {
		job, err := jc.queries.GetJobHistory(c.Request().Context(), c.Param("job_id"))
		if err != nil {
			return echo.NewHTTPError(
				http.StatusInternalServerError,
				"could not get job").
				WithInternal(err)
		}

		return c.Render(http.StatusOK, "jobs.job", echo.Map{
			"Title": "Job",
			"Jobs":  pages.ConvertFinishedJobsForShow(job),
		})
	}
}
