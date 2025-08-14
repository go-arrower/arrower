package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/repository"
)

func TestWithIDField(t *testing.T) {
	t.Parallel()

	err := repository.WithIDField("ID")(&invalidRepositoryMissingIDFieldName{})
	assert.Error(t, err)

	err = repository.WithIDField("ID")(&invalidRepositoryWrongIDFieldNameType{})
	assert.Error(t, err)
}

type invalidRepositoryMissingIDFieldName struct{}

type invalidRepositoryWrongIDFieldNameType struct {
	IDFieldName int
}
