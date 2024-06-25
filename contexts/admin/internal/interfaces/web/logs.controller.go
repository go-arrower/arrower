package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"github.com/go-arrower/arrower/alog"
	"github.com/go-arrower/arrower/alog/models"
	"github.com/go-arrower/arrower/setting"
)

func NewLogsController(
	logger alog.Logger,
	settings setting.Settings,
	queries *models.Queries,
	routes *echo.Group,
) *LogsController {
	return &LogsController{
		logger:   logger,
		settings: settings,
		queries:  queries,
		r:        routes,
	}
}

type LogsController struct {
	logger   alog.Logger
	settings setting.Settings
	queries  *models.Queries
	r        *echo.Group
}

// ShowLogs pattern issue: how to add route with and without trailing slash?
func (lc *LogsController) ShowLogs() {
	const defaultLogs = 1000

	type logFilter struct {
		Level string `query:"level"`
		K0    string `query:"k0"`
		F0    string `query:"f0"`
		K1    string `query:"k1"`
		F1    string `query:"f1"`
		K2    string `query:"k2"`
		F2    string `query:"f2"`
		Range int    `query:"range"`
	}

	type logLine struct {
		Time   time.Time
		Log    map[string]any
		UserID string
	}

	lc.r.GET("/", func(c echo.Context) error {
		timeParam := c.QueryParam("time")
		searchMsgParam := c.QueryParam("msg")

		var filter logFilter
		err := c.Bind(&filter)
		if err != nil {
			return c.String(http.StatusBadRequest, "bad request")
		}

		if filter.Range == 0 {
			filter.Range = 15
		}

		filterTime := time.Now().UTC().Add(-time.Duration(filter.Range) * time.Minute)
		if t, err := time.Parse("2006-01-02T15:04:05.999999999", timeParam); err == nil { //nolint:govet // shadow err is ok
			filterTime = t
		}

		filterTime = filterTime.Add(-1 * time.Hour)

		level := []string{"INFO"}
		if filter.Level == "DEBUG" {
			level = []string{"INFO", "DEBUG"}
		}

		queryParams := models.GetRecentLogsParams{
			Time:  pgtype.Timestamptz{Time: filterTime, Valid: true, InfinityModifier: pgtype.Finite},
			Msg:   "%" + searchMsgParam + "%",
			Level: level,
			Limit: defaultLogs,
		}

		if filter.K0 != "" {
			queryParams.F0 = fmt.Sprintf(`$.**.%s ? (@ like_regex "^.*%s.*" flag "i")`, filter.K0, filter.F0)
		}
		if filter.K1 != "" {
			queryParams.F1 = fmt.Sprintf(`$.**.%s ? (@ like_regex "^.*%s.*" flag "i")`, filter.K1, filter.F1)
		}
		if filter.K2 != "" {
			queryParams.F2 = fmt.Sprintf(`$.**.%s ? (@ like_regex "^.*%s.*" flag "i")`, filter.K2, filter.F2)
		}

		rawLogs, _ := lc.queries.GetRecentLogs(c.Request().Context(), queryParams)

		var logs []logLine
		for _, l := range rawLogs {
			var log logLine

			_ = json.Unmarshal(l.Log, &log.Log)
			log.Time = l.Time.Time
			log.UserID = l.UserID.UUID.String()
			if (uuid.NullUUID{}) == l.UserID { //nolint:exhaustruct // checking if user ID is empty
				log.UserID = ""
			}

			logs = append(logs, log)
		}

		settingLevel, err := lc.settings.Setting(c.Request().Context(), alog.SettingLogLevel)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		vals := echo.Map{
			"Title":     "Logs",
			"SearchMsg": searchMsgParam,
			"Logs":      logs,
			"Filter":    filter,
			"Settings": map[string]any{
				// !!! assumes all loggers in all replicas are configured the same
				"Enabled": alog.Unwrap(lc.logger).UsesSettings(),
				"Level":   getLevelName(slog.Level(settingLevel.MustInt())),
			},
		}

		if len(logs) == 0 {
			vals["LastLogTime"] = time.Now()
		}

		return c.Render(http.StatusOK, "logs.show", vals)
	}).Name = "admin.logs"
}

func (lc *LogsController) SettingLogs() {
	lc.r.GET("/setting", func(c echo.Context) error {
		levelParam := c.QueryParam("level")

		level, err := strconv.Atoi(levelParam)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		err = lc.settings.Save(c.Request().Context(), alog.SettingLogLevel, setting.NewValue(level))
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		return c.Render(http.StatusOK, "logs.show#level-setting", echo.Map{
			"Level": getLevelName(slog.Level(level)),
		})
	})
}

//nolint:misspell // leveller is wrong in this case
func getLevelName(leveler slog.Leveler) string {
	return map[slog.Leveler]string{
		slog.LevelInfo:  "INFO",
		slog.LevelDebug: "DEBUG",
		alog.LevelInfo:  "ARROWER:INFO",
		alog.LevelDebug: "ARROWER:DEBUG",
	}[leveler]
}
