package init

import (
	"net/http"
	"sort"

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
		jobs.GET("", di.jobsController.ListQueues())
		jobs.GET("/", di.jobsController.ListQueues())
		jobs.GET("/data/pending", di.jobsController.PendingJobsPieChartData())                // todo better htmx fruednly data URL
		jobs.GET("/data/processed/:interval", di.jobsController.ProcessedJobsLineChartData()) // todo better htmx fruednly data URL
		jobs.GET("/:queue", di.jobsController.ShowQueue()).Name = "admin.jobs.queue"          // todo move route(s) to /queue/:queue_name (or similar)
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
		jobs.GET("/job/:job_id", di.jobsController.ShowJob()).Name = "admin.jobs.job"
	}
}
