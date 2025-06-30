package repository_test

import (
	"testing"

	"github.com/go-arrower/arrower/repository"
	"github.com/stretchr/testify/assert"
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
