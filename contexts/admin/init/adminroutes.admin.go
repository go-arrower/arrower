package init

import (
	"net/http"
	"sort"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/labstack/echo/v4"
)

func registerAdminRoutes(di *AdminContext) {
	di.globalContainer.AdminRouter.GET("", func(c echo.Context) error {
		return c.Render(http.StatusOK, "admin.home", nil)
	})

	di.globalContainer.AdminRouter.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "admin.home", nil)
	})

	di.globalContainer.AdminRouter.GET("/routes", func(c echo.Context) error {
		routes := di.globalContainer.WebRouter.Routes()

		// sort routes by path and then by method
		sort.Slice(routes, func(i, j int) bool {
			if routes[i].Path < routes[j].Path {
				return true
			}

			if routes[i].Path == routes[j].Path {
				return routes[i].Method < routes[j].Method
			}

			return false
		})

		return c.Render(http.StatusOK, "admin.routes", echo.Map{
			"Flashes": nil,
			"Routes":  routes,
		})
	})

	di.settingsController.List()

	di.logsController.ShowLogs()
	di.logsController.SettingLogs()

	{
		jobs := di.globalContainer.AdminRouter.Group("/jobs")
		jobs.GET("", di.jobsController.Index()).Name = "admin.jobs"
		jobs.GET("/", di.jobsController.Index())
		jobs.GET("/data/pending", di.jobsController.PendingJobsPieChartData())                // todo better htmx fruednly data URL
		jobs.GET("/data/processed/:interval", di.jobsController.ProcessedJobsLineChartData()) // todo better htmx fruednly data URL
		jobs.GET("/:queue", di.jobsController.ShowQueue()).Name = "admin.jobs.queue"
		jobs.GET("/:queue/delete/:job_id", di.jobsController.DeleteJob())
		jobs.GET("/:queue/reschedule/:job_id", di.jobsController.RescheduleJob())
		jobs.GET("/schedule", di.jobsController.CreateJobs()).Name = "admin.jobs.schedule"
		jobs.POST("/schedule", di.jobsController.ScheduleJobs()).Name = "admin.jobs.new"
		jobs.GET("/jobTypes", di.jobsController.ShowJobTypes())
		jobs.GET("/payloads", di.jobsController.PayloadExamples())
		jobs.GET("/workers", di.jobsController.ListWorkers())
		jobs.GET("/maintenance", di.jobsController.ShowMaintenance()).Name = "admin.jobs.maintenance"
		jobs.POST("/vacuum/:table", di.jobsController.VacuumJobTables())
		jobs.POST("/history", di.jobsController.DeleteHistory())
		jobs.POST("/history/prune", di.jobsController.PruneHistory())
		jobs.GET("/history/size/", di.jobsController.EstimateHistorySize())
		jobs.GET("/history/payload/size/", di.jobsController.EstimateHistoryPayloadSize())
		jobs.GET("/finished", di.jobsController.FinishedJobs()).Name = "admin.jobs.finished"
		jobs.GET("/finished/total", di.jobsController.FinishedJobsTotal()).Name = "admin.jobs.finished_total"
		jobs.GET("/job/:job_id", di.jobsController.JobShow()).Name = "admin.jobs.job.show"
	}

	di.globalContainer.AdminRouter.GET("/charts/users", func(c echo.Context) error {
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
				s.Max = int(value * 1.6)
			})}...)

		page := components.NewPage()
		page.AddCharts(chart)
		page.SetPageTitle("Users Chart - Arrower")
		page.SetLayout(components.PageNoneLayout)
		//page.SetAssetsHost() //todo

		return page.Render(c.Response().Writer)
	})
}
