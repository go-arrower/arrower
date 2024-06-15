package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-arrower/arrower/alog"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

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

func NewJobsController(
	logger alog.Logger,
	queries *models.Queries,
	repo jobs.Repository,
	app application.JobsApplication,
	appDI application.App,
) *JobsController {
	return &JobsController{
		logger:  logger,
		queries: queries,
		repo:    repo,
		app:     app,
		appDI:   appDI,
	}
}

type JobsController struct {
	logger alog.Logger

	queries *models.Queries
	repo    jobs.Repository
	app     application.JobsApplication
	appDI   application.App
}

func (jc *JobsController) ListQueues() func(c echo.Context) error {
	return func(c echo.Context) error {
		res, err := jc.appDI.ListAllQueues.H(c.Request().Context(), application.ListAllQueuesQuery{})
		if err != nil {
			return fmt.Errorf("%w", err)
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
			return fmt.Errorf("%w", err)
		}

		var json []pieData
		for _, q := range res.QueueStats {
			json = append(json, pieData{Name: string(q.QueueName), Value: q.PendingJobs})
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
				DateBin:    pgtype.Interval{Valid: true, Microseconds: int64(time.Minute * 5 / time.Microsecond)},
				FinishedAt: pgtype.Timestamptz{Valid: true, Time: time.Now().UTC().Add(-time.Hour), InfinityModifier: pgtype.Finite},
				Limit:      bucketsPerHour,
			})
		} else {
			jobData, err = jc.queries.PendingJobs(c.Request().Context(), models.PendingJobsParams{ // show whole week
				DateBin:    pgtype.Interval{Valid: true, Days: 1},
				FinishedAt: pgtype.Timestamptz{Valid: true, Time: time.Now().UTC().Add(-time.Hour * 24 * bucketsPerWeek), InfinityModifier: pgtype.Finite},
				Limit:      bucketsPerWeek,
			})
		}

		if err != nil {
			return fmt.Errorf("%w", err)
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
			return fmt.Errorf("%w", err)
		}

		page := buildQueuePage(queue, res.Jobs, res.Kpis)

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
			return fmt.Errorf("%w", err)
		}

		return c.Render(http.StatusOK, "jobs.workers", echo.Map{
			"Title":   "Workers",
			"workers": presentWorkers(res.Pool),
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
		q := c.Param("queue")
		jobID := c.Param("job_id")

		_ = jc.app.RescheduleJob(c.Request().Context(), application.RescheduleJobRequest{JobID: jobID})

		return c.Redirect(http.StatusSeeOther, "/admin/jobs/"+q)
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
				queue = "Default"
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
	const timeDay = time.Hour * 24 // todo const is defined multiple times => make global

	return func(c echo.Context) error {
		days, _ := strconv.Atoi(c.FormValue("days"))
		estimateBefore := time.Now().Add(-1 * time.Duration(days) * timeDay)

		queue := c.FormValue("queue")
		if queue == "Default" {
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
	const timeDay = time.Hour * 24

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
	const timeDay = time.Hour * 24

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
		queues, _ := jc.app.Queues(c.Request().Context())

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
			return fmt.Errorf("%w", err)
		}
		priority *= -1

		count, err := strconv.Atoi(num)
		if err != nil {
			return fmt.Errorf("%w", err)
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
			return fmt.Errorf("%w", err)
		}

		return c.Redirect(http.StatusSeeOther, "/admin/jobs/"+queue)
	}
}

func presentWorkers(pool []jobs.WorkerPool) []pages.JobWorker {
	jobWorkers := make([]pages.JobWorker, len(pool))

	for i, _ := range pool {
		jobWorkers[i].ID = pool[i].ID
		jobWorkers[i].Queue = pool[i].Queue
		jobWorkers[i].Workers = pool[i].Workers
		jobWorkers[i].Version = pool[i].Version
		jobWorkers[i].JobTypes = pool[i].JobTypes

		sort.Slice(jobWorkers[i].JobTypes, func(ii, ij int) bool {
			return jobWorkers[i].JobTypes[ii] <= jobWorkers[i].JobTypes[ij]
		})

		var warningSecondsWorkerPoolNotSeenSince time.Duration = 30

		jobWorkers[i].LastSeenAtColourSuccess = true
		if time.Since(pool[i].LastSeen)/time.Second >= warningSecondsWorkerPoolNotSeenSince {
			jobWorkers[i].LastSeenAtColourSuccess = false
		}

		jobWorkers[i].NotSeenSince = notSeenSinceTimeString(pool[i].LastSeen, warningSecondsWorkerPoolNotSeenSince)
	}

	sort.Slice(jobWorkers, func(i, j int) bool {
		return jobWorkers[i].ID <= jobWorkers[j].ID
	})

	return jobWorkers
}

func notSeenSinceTimeString(t time.Time, warningSecondsWorkerPoolNotSeenSince time.Duration) string {
	seconds := time.Since(t).Seconds()

	if time.Duration(seconds) >= warningSecondsWorkerPoolNotSeenSince && seconds < 60 {
		return "recently"
	}

	secondsPerMinute := 60.0
	if seconds > secondsPerMinute {
		minutes := int(math.Round(seconds / secondsPerMinute))
		if minutes == 1 {
			return fmt.Sprintf("%d minute ago", minutes)
		}

		return fmt.Sprintf("%d minutes ago", minutes)
	}

	return "now"
}

type (
	QueueStats struct {
		PendingJobsPerType   map[string]int
		QueueName            string
		PendingJobs          int
		FailedJobs           int
		ProcessedJobs        int
		AvailableWorkers     int
		PendingJobsErrorRate float64 // can be calculated: FailedJobs * 100 / PendingJobs
		AverageTimePerJob    time.Duration
		EstimateUntilEmpty   time.Duration // can be calculated
	}

	ListQueuesPage struct {
		Queues map[jobs.QueueName]jobs.QueueStats
	}

	QueuePage struct {
		Jobs      []jobs.PendingJob
		QueueName string
		Stats     QueueStats
	}
)

func buildQueuePage(queue string, jobs []jobs.PendingJob, kpis jobs.QueueKPIs) QueuePage {
	jobs = prettyFormatPayload(jobs)

	return QueuePage{
		QueueName: queue,
		Stats:     queueKpiToStats(queue, kpis),

		Jobs: jobs,
	}
}

func prettyFormatPayload(pJobs []jobs.PendingJob) []jobs.PendingJob {
	for i := 0; i < len(pJobs); i++ { //nolint:varnamelen
		var m application.JobPayload

		_ = json.Unmarshal([]byte(pJobs[i].Payload), &m)
		data, _ := json.Marshal(m.JobData)

		var prettyJSON bytes.Buffer

		if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		}

		if pJobs[i].Queue == "" {
			pJobs[i].Queue = string(jobs.DefaultQueueName)
		}
		pJobs[i].Payload = prettyJSON.String()
		pJobs[i].RunAtFmt = fmtRunAtTime(pJobs[i].RunAt)
	}

	return pJobs
}

func fmtRunAtTime(tme time.Time) string {
	now := time.Now()

	isToday := tme.Year() == now.Year() && tme.Month() == now.Month() && tme.Day() == now.Day()
	if isToday {
		return fmt.Sprintf("%02d:%02d", tme.Hour(), tme.Minute())
	}

	return tme.Format("2006.01.02 15:04")
}

func queueKpiToStats(queue string, kpis jobs.QueueKPIs) QueueStats {
	var errorRate float64

	if kpis.FailedJobs != 0 {
		errorRate = float64(kpis.FailedJobs * 100 / kpis.PendingJobs)
	}

	var duration time.Duration
	if kpis.AvailableWorkers != 0 {
		duration = time.Duration(kpis.PendingJobs/kpis.AvailableWorkers) * kpis.AverageTimePerJob
	}

	return QueueStats{
		QueueName:            queue,
		PendingJobs:          kpis.PendingJobs,
		PendingJobsPerType:   kpis.PendingJobsPerType,
		FailedJobs:           kpis.FailedJobs,
		ProcessedJobs:        kpis.ProcessedJobs,
		AvailableWorkers:     kpis.AvailableWorkers,
		PendingJobsErrorRate: errorRate,
		AverageTimePerJob:    kpis.AverageTimePerJob.Truncate(time.Second),
		EstimateUntilEmpty:   duration.Truncate(time.Second),
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
				return fmt.Errorf("%w", err)
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
			return fmt.Errorf("%w", err)
		}

		if filter != (jobs.Filter{}) {
			c.Response().Header().Set("HX-TRIGGER", finishedJobsFilterChangedJSEvent)

			return c.Render(http.StatusOK, "jobs.finished#jobs.list", pages.NewFinishedJobs(finishedJobs, nil))
		}

		queues, err := jc.repo.Queues(c.Request().Context())
		if err != nil {
			return fmt.Errorf("%w", err)
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
			return fmt.Errorf("%w", err)
		}

		return c.String(http.StatusOK, strconv.FormatInt(total, 10))
	}
}

func (jc *JobsController) ShowJob() func(ctx echo.Context) error {
	return func(c echo.Context) error {
		jobs, err := jc.queries.GetJobHistory(c.Request().Context(), c.Param("job_id"))
		if err != nil {
			return fmt.Errorf("%v", err)
		}

		return c.Render(http.StatusOK, "jobs.job", echo.Map{
			"Title": "Job",
			"Jobs":  pages.ConvertFinishedJobsForShow(jobs),
		})
	}
}
