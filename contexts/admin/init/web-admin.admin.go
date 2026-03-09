package init

import (
	"net/http"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/labstack/echo/v4"

	"github.com/go-arrower/arrower/contexts/admin/internal/views"
)

func (c *AdminContext) registerAdminRoutes() {
	c.shared.WebRouter.StaticFS("/static/admin/", echo.MustSubFS(views.PublicAssets, "static"))

	c.shared.AdminRouter.GET("", func(c echo.Context) error {
		return c.Render(http.StatusOK, "admin.home", nil)
	}).Name = "admin.home"

	c.shared.AdminRouter.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "admin.home", nil)
	})

	jobs := c.shared.AdminRouter.Group("/jobs")
	jobs.GET("", c.jobsController.Index()).Name = "admin.jobs"
	jobs.GET("/", c.jobsController.Index())
	jobs.GET("/data/pending", c.jobsController.PendingJobsPieChartData())                // todo better htmx fruednly data URL
	jobs.GET("/data/processed/:interval", c.jobsController.ProcessedJobsLineChartData()) // todo better htmx fruednly data URL
	jobs.GET("/:queue", c.jobsController.ShowQueue()).Name = "admin.jobs.queue"
	jobs.GET("/:queue/delete/:job_id", c.jobsController.DeleteJob()).Name = "admin.jobs.delete"
	jobs.GET("/:queue/reschedule/:job_id", c.jobsController.RescheduleJob()).Name = "admin.jobs.reschedule"
	jobs.GET("/schedule", c.jobsController.CreateJobs()).Name = "admin.jobs.schedule"
	jobs.POST("/schedule", c.jobsController.ScheduleJobs()).Name = "admin.jobs.new"
	jobs.GET("/jobTypes", c.jobsController.ShowJobTypes()).Name = "admin.jobs.jobTypes"
	jobs.GET("/payloads", c.jobsController.PayloadExamples()).Name = "admin.jobs.payloads"
	jobs.GET("/workers", c.jobsController.ListWorkers()).Name = "admin.jobs.workers"
	jobs.GET("/maintenance", c.jobsController.ShowMaintenance()).Name = "admin.jobs.maintenance"
	jobs.POST("/vacuum/:table", c.jobsController.VacuumJobTables()).Name = "admin.jobs.vacuum"
	jobs.POST("/history", c.jobsController.DeleteHistory()).Name = "admin.jobs.history"
	jobs.POST("/history/prune", c.jobsController.PruneHistory()).Name = "admin.jobs.history.prune"
	jobs.GET("/history/size/", c.jobsController.EstimateHistorySize()).Name = "admin.jobs.history.size"
	jobs.GET("/history/payload/size/", c.jobsController.EstimateHistoryPayloadSize()).Name = "admin.jobs.history.payload-size"
	jobs.POST("/history/cron/", c.jobsController.UpdateCron()).Name = "admin.jobs.cron"
	jobs.GET("/finished", c.jobsController.FinishedJobs()).Name = "admin.jobs.finished"
	jobs.GET("/finished/total", c.jobsController.FinishedJobsTotal()).Name = "admin.jobs.finished_total"
	jobs.GET("/job/:job_id", c.jobsController.JobShow()).Name = "admin.jobs.job.show"

	routes := c.shared.AdminRouter.Group("/routes")
	routes.GET("", c.routesController.Index()).Name = "admin.routes"
	routes.GET("/", c.routesController.Index()).Name = "admin.routes"

	settings := c.shared.AdminRouter.Group("/settings")
	settings.GET("", c.settingsController.Index()).Name = "admin.settings"
	settings.GET("/", c.settingsController.Index()).Name = "admin.settings"

	logs := c.shared.AdminRouter.Group("/logs")
	logs.GET("", c.logsController.Index()).Name = "admin.logs"
	logs.GET("/", c.logsController.Index())
	logs.POST("/setting", c.logsController.Update()).Name = "admin.logs.setting"

	c.shared.AdminRouter.GET("/charts/users", func(c echo.Context) error {
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
		// page.SetAssetsHost() //todo

		return page.Render(c.Response().Writer)
	}).Name = "admin.charts.users"
}
