package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/go-arrower/arrower/repository"
)

func TestWithIDField(t *testing.T) {
	t.Parallel()

	assert.PanicsWithValue(t, "cannot set IDField: repository MUST have a field `IDFieldName` of type string", func() {
		newInvalidRepositoryMissingIDFieldName(repository.WithIDField("ID"))
	})

	assert.PanicsWithValue(t, "cannot set IDField: repository MUST have a field `IDFieldName` of type string", func() {
		newInvalidRepositoryWrongIDFieldNameType(repository.WithIDField("ID"))
	})
}

func newInvalidRepositoryMissingIDFieldName(opts ...repository.Option) *invalidRepositoryMissingIDFieldName {
	repo := &invalidRepositoryMissingIDFieldName{}

	for _, opt := range opts {
		opt(repo)
	}

	return repo
}

type invalidRepositoryMissingIDFieldName struct {
	IDFieldName int
}

func newInvalidRepositoryWrongIDFieldNameType(opts ...repository.Option) *invalidRepositoryWrongIDFieldNameType {
	repo := &invalidRepositoryWrongIDFieldNameType{}

	for _, opt := range opts {
		opt(repo)
	}

	return repo
}

type invalidRepositoryWrongIDFieldNameType struct {
	IDFieldName int
}
