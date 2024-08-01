package jobs_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/jobs"
)

func TestTest(t *testing.T) {
	t.Parallel()

	t.Run("test queue", func(t *testing.T) {
		t.Parallel()

		jq := jobs.Test(t)
		assert.NotNil(t, jq)
	})

	t.Run("nil does panic", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			jobs.Test(nil)
		})
	})
}
