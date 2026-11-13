package init

import (
	"net/http"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/labstack/echo/v5"

	"github.com/go-arrower/arrower/contexts/admin/internal/views"
)

func (c *AdminContext) registerAdminRoutes() {
	c.shared.WebRouter.StaticFS("/static/admin/", echo.MustSubFS(views.PublicAssets, "static"))

	addNamedRoute(c.shared.AdminRouter, "GET", "", "admin.home", func(c *echo.Context) error {
		return c.Render(http.StatusOK, "admin.home", nil)
	})

	c.shared.AdminRouter.GET("/", func(c *echo.Context) error {
		return c.Render(http.StatusOK, "admin.home", nil)
	})

	jobs := c.shared.AdminRouter.Group("/jobs")
	addNamedRoute(jobs, "GET", "", "admin.jobs", c.jobsController.Index())
	jobs.GET("/", c.jobsController.Index())
	jobs.GET("/data/pending", c.jobsController.PendingJobsPieChartData())
	jobs.GET("/data/processed/:interval", c.jobsController.ProcessedJobsLineChartData())
	addNamedRoute(jobs, "GET", "/:queue", "admin.jobs.queue", c.jobsController.ShowQueue())
	addNamedRoute(jobs, "GET", "/:queue/delete/:job_id", "admin.jobs.delete", c.jobsController.DeleteJob())
	addNamedRoute(jobs, "GET", "/:queue/reschedule/:job_id", "admin.jobs.reschedule", c.jobsController.RescheduleJob())
	addNamedRoute(jobs, "GET", "/schedule", "admin.jobs.schedule", c.jobsController.CreateJobs())
	addNamedRoute(jobs, "POST", "/schedule", "admin.jobs.new", c.jobsController.ScheduleJobs())
	addNamedRoute(jobs, "GET", "/jobTypes", "admin.jobs.jobTypes", c.jobsController.ShowJobTypes())
	addNamedRoute(jobs, "GET", "/payloads", "admin.jobs.payloads", c.jobsController.PayloadExamples())
	addNamedRoute(jobs, "GET", "/workers", "admin.jobs.workers", c.jobsController.ListWorkers())
	addNamedRoute(jobs, "GET", "/maintenance", "admin.jobs.maintenance", c.jobsController.ShowMaintenance())
	addNamedRoute(jobs, "POST", "/vacuum/:table", "admin.jobs.vacuum", c.jobsController.VacuumJobTables())
	addNamedRoute(jobs, "POST", "/history", "admin.jobs.history", c.jobsController.DeleteHistory())
	addNamedRoute(jobs, "POST", "/history/prune", "admin.jobs.history.prune", c.jobsController.PruneHistory())
	addNamedRoute(jobs, "GET", "/history/size/", "admin.jobs.history.size", c.jobsController.EstimateHistorySize())
	addNamedRoute(jobs, "GET", "/history/payload/size/", "admin.jobs.history.payload-size", c.jobsController.EstimateHistoryPayloadSize())
	addNamedRoute(jobs, "POST", "/history/cron/", "admin.jobs.cron", c.jobsController.UpdateCron())
	addNamedRoute(jobs, "GET", "/finished", "admin.jobs.finished", c.jobsController.FinishedJobs())
	addNamedRoute(jobs, "GET", "/finished/total", "admin.jobs.finished_total", c.jobsController.FinishedJobsTotal())
	addNamedRoute(jobs, "GET", "/job/:job_id", "admin.jobs.job.show", c.jobsController.JobShow())

	routes := c.shared.AdminRouter.Group("/routes")
	addNamedRoute(routes, "GET", "", "admin.routes", c.routesController.Index())
	addNamedRoute(routes, "GET", "/", "admin.routes", c.routesController.Index())

	settings := c.shared.AdminRouter.Group("/settings")
	addNamedRoute(settings, "GET", "", "admin.settings", c.settingsController.Index())
	addNamedRoute(settings, "GET", "/", "admin.settings", c.settingsController.Index())

	logs := c.shared.AdminRouter.Group("/logs")
	addNamedRoute(logs, "GET", "", "admin.logs", c.logsController.Index())
	logs.GET("/", c.logsController.Index())
	addNamedRoute(logs, "POST", "/setting", "admin.logs.setting", c.logsController.Update())

	//nolint:exhaustruct
	addNamedRoute(c.shared.AdminRouter, "GET", "/charts/users", "admin.charts.users", func(c *echo.Context) error {
		chart := charts.NewGauge()
		chart.SetGlobalOptions(
			charts.WithTooltipOpts(opts.Tooltip{Formatter: "{b}: {c}"}),
			charts.WithInitializationOpts(opts.Initialization{
				AssetsHost: "https://go-echarts.github.io/go-echarts-assets/assets/", // todo
				PageTitle:  "User Count - Arrower",
				Theme:      types.ThemeWalden,
				Width:      "500px",
			}),
		)

		ani := true
		value := 40.0

		chart.AddSeries("",
			[]opts.GaugeData{{Name: "Users", Value: value}},
			[]charts.SeriesOpts{charts.WithSeriesOpts(func(s *charts.SingleSeries) {
				s.Progress = &opts.Progress{Show: &ani}
				s.Detail = &opts.Detail{Formatter: "{value}"}
				s.Max = int(value * 1.6) //nolint:mnd
			})}...)

		page := components.NewPage()
		page.AddCharts(chart)
		page.SetPageTitle("Users Chart - Arrower")
		page.SetLayout(components.PageNoneLayout)

		return page.Render(c.Response())
	})
}

func addNamedRoute(group *echo.Group, method, path, name string, handler echo.HandlerFunc, middleware ...echo.MiddlewareFunc) {
	_, err := group.AddRoute(echo.Route{
		Method:      method,
		Path:        path,
		Name:        name,
		Handler:     handler,
		Middlewares: middleware,
	})
	if err != nil {
		panic(err)
	}
}
