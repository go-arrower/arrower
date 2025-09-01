package arepo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/arepo"
)

func TestWithIDField(t *testing.T) {
	t.Parallel()

	err := arepo.WithIDField("ID")(&invalidRepositoryMissingIDFieldName{})
	assert.Error(t, err)

	err = arepo.WithIDField("ID")(&invalidRepositoryWrongIDFieldNameType{})
	assert.Error(t, err)
}

type invalidRepositoryMissingIDFieldName struct{}

type invalidRepositoryWrongIDFieldNameType struct {
	IDFieldName int
}
