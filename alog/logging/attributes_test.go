package logging_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/alog/logging"
)

func TestError(t *testing.T) {
	t.Parallel()

	got := logging.Error(errors.New("my-error")) //nolint:err113
	assert.Equal(t, "error", got.Key)
	assert.Equal(t, "my-error", got.Value.String())
	assert.Equal(t, "error=my-error", got.String())
}
