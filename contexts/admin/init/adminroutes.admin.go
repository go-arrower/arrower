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

func registerAdminRoutes(di *AdminContext) {
	di.shared.WebRouter.StaticFS("/static/admin/", echo.MustSubFS(views.PublicAssets, "static"))

	di.shared.AdminRouter.GET("", func(c echo.Context) error {
		return c.Render(http.StatusOK, "admin.home", nil)
	}).Name = "admin.home"

	di.shared.AdminRouter.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "admin.home", nil)
	})

	jobs := di.shared.AdminRouter.Group("/jobs")
	jobs.GET("", di.jobsController.Index()).Name = "admin.jobs"
	jobs.GET("/", di.jobsController.Index())
	jobs.GET("/data/pending", di.jobsController.PendingJobsPieChartData())                // todo better htmx fruednly data URL
	jobs.GET("/data/processed/:interval", di.jobsController.ProcessedJobsLineChartData()) // todo better htmx fruednly data URL
	jobs.GET("/:queue", di.jobsController.ShowQueue()).Name = "admin.jobs.queue"
	jobs.GET("/:queue/delete/:job_id", di.jobsController.DeleteJob()).Name = "admin.jobs.delete"
	jobs.GET("/:queue/reschedule/:job_id", di.jobsController.RescheduleJob()).Name = "admin.jobs.reschedule"
	jobs.GET("/schedule", di.jobsController.CreateJobs()).Name = "admin.jobs.schedule"
	jobs.POST("/schedule", di.jobsController.ScheduleJobs()).Name = "admin.jobs.new"
	jobs.GET("/jobTypes", di.jobsController.ShowJobTypes()).Name = "admin.jobs.jobTypes"
	jobs.GET("/payloads", di.jobsController.PayloadExamples()).Name = "admin.jobs.payloads"
	jobs.GET("/workers", di.jobsController.ListWorkers()).Name = "admin.jobs.workers"
	jobs.GET("/maintenance", di.jobsController.ShowMaintenance()).Name = "admin.jobs.maintenance"
	jobs.POST("/vacuum/:table", di.jobsController.VacuumJobTables()).Name = "admin.jobs.vacuum"
	jobs.POST("/history", di.jobsController.DeleteHistory()).Name = "admin.jobs.history"
	jobs.POST("/history/prune", di.jobsController.PruneHistory()).Name = "admin.jobs.history.prune"
	jobs.GET("/history/size/", di.jobsController.EstimateHistorySize()).Name = "admin.jobs.history.size"
	jobs.GET("/history/payload/size/", di.jobsController.EstimateHistoryPayloadSize()).Name = "admin.jobs.history.payload-size"
	jobs.GET("/finished", di.jobsController.FinishedJobs()).Name = "admin.jobs.finished"
	jobs.GET("/finished/total", di.jobsController.FinishedJobsTotal()).Name = "admin.jobs.finished_total"
	jobs.GET("/job/:job_id", di.jobsController.JobShow()).Name = "admin.jobs.job.show"

	routes := di.shared.AdminRouter.Group("/routes")
	routes.GET("", di.routesController.Index()).Name = "admin.routes"
	routes.GET("/", di.routesController.Index()).Name = "admin.routes"

	settings := di.shared.AdminRouter.Group("/settings")
	settings.GET("", di.settingsController.Index()).Name = "admin.settings"
	settings.GET("/", di.settingsController.Index()).Name = "admin.settings"

	logs := di.shared.AdminRouter.Group("/logs")
	logs.GET("", di.logsController.Index()).Name = "admin.logs"
	logs.GET("/", di.logsController.Index())
	logs.POST("/setting", di.logsController.Update()).Name = "admin.logs.setting"

	di.shared.AdminRouter.GET("/charts/users", func(c echo.Context) error {
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
