package arrower

import (
	"context"
	"fmt"
	"time"
)

func getSystemStatus(di *Container, serverStartedAt time.Time) interface{} {
	uptime := time.Since(serverStartedAt).Round(time.Second)

	dbOnline := "online"

	if di.PGx != nil {
		err := di.PGx.Ping(context.Background())
		if err != nil {
			dbOnline = fmt.Errorf("err: %w", err).Error()
		}
	}

	statusData := map[string]any{
		"status":           "online", // later: maintenance mode, degraded etc.
		"time":             time.Now(),
		"uptime":           uptime.String(),
		"gitCommit":        "", // todo what is the difference to the hash? Does it mean git tag instead?
		"gitHash":          "", // todo
		"organisationName": di.Config.OrganisationName,
		"applicationName":  di.Config.ApplicationName,
		"instanceName":     di.Config.InstanceName,
		"environment":      di.Config.Environment,

		"web":      di.Config.HTTP,
		"database": dbStatus{Postgres: di.Config.Postgres, Status: dbOnline},
		// s3
		// REST API
		// feature flags
		// queues
		// memory consumption

		"failures": map[string]any{},
	}

	return statusData
}

type dbStatus struct {
	Postgres
	Status string `json:"status"`
	// average response time (?)
}
