package pages

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/go-arrower/arrower/contexts/admin/internal/domain/jobs"
)

type JobWorker struct { // todo embed jobs.WorkerPool struct
	ID                      string
	Queue                   string
	NotSeenSince            string
	Version                 string
	JobTypes                []string
	Workers                 int
	LastSeenAtColourSuccess bool
}

func PresentWorkers(pool []jobs.WorkerPool) []JobWorker {
	jobWorkers := make([]JobWorker, len(pool))

	for i := range pool {
		jt := []string{}

		for _, t := range pool[i].JobTypes {
			jt = append(jt, string(t))
		}

		jobWorkers[i].ID = pool[i].InstanceName
		jobWorkers[i].Queue = string(pool[i].Queue)
		jobWorkers[i].Workers = pool[i].Workers
		jobWorkers[i].Version = pool[i].Version
		jobWorkers[i].JobTypes = jt

		sort.Slice(jobWorkers[i].JobTypes, func(ii, ij int) bool {
			return jobWorkers[i].JobTypes[ii] <= jobWorkers[i].JobTypes[ij]
		})

		var warningSecondsWorkerPoolNotSeenSince time.Duration = 30

		jobWorkers[i].LastSeenAtColourSuccess = true
		if time.Since(pool[i].LastSeenAt)/time.Second >= warningSecondsWorkerPoolNotSeenSince {
			jobWorkers[i].LastSeenAtColourSuccess = false
		}

		jobWorkers[i].NotSeenSince = notSeenSinceTimeString(pool[i].LastSeenAt, warningSecondsWorkerPoolNotSeenSince)
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
