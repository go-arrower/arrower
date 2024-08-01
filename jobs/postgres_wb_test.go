package jobs

import (
	"log"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// white box test. if it fails, feel free to delete it.
func TestGetJobTypeFromType(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		rType    reflect.Type
		basePath string
		jobType  string
		fullPath string
	}{
		"normal base path": {
			reflect.TypeOf(simpleJob{}),
			"github.com/go-arrower/arrower",
			"jobs.simpleJob",
			"github.com/go-arrower/arrower/jobs.simpleJob",
		},
		"base path ending /": {
			reflect.TypeOf(simpleJob{}),
			"github.com/go-arrower/arrower/",
			"jobs.simpleJob",
			"github.com/go-arrower/arrower/jobs.simpleJob",
		},
		"job struct from other module": {
			reflect.TypeOf(log.Logger{}),
			"github.com/go-arrower/arrower/",
			"log.Logger",
			"log.Logger",
		},
		"no base path": {
			reflect.TypeOf(simpleJob{}),
			"",
			"github.com/go-arrower/arrower/jobs.simpleJob",
			"github.com/go-arrower/arrower/jobs.simpleJob",
		},
		"custom overwrite": {
			reflect.TypeOf(jobWithJobType{}),
			"github.com/go-arrower/arrower/",
			"custom.job.type",
			"github.com/go-arrower/arrower/jobs.jobWithJobType",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			j, p, _ := getJobTypeFromType(tt.rType, tt.basePath)
			assert.Equal(t, tt.jobType, j)
			assert.Equal(t, tt.fullPath, p)
		})
	}
}

type simpleJob struct{}

type jobWithJobType struct {
	Name string
}

func (j jobWithJobType) JobType() string {
	return "custom.job.type"
}
