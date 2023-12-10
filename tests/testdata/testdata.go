//nolint:gochecknoglobals // this is testdata and global variables are a feature
package testdata

import (
	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
)

var DefaultEntity = NewEntity()

func NewEntity() Entity {
	return Entity{
		ID:   EntityID(uuid.New().String()),
		Name: gofakeit.Name(),
	}
}

type (
	EntityID string
	Entity   struct {
		ID   EntityID
		Name string
	}
)

type EntityWithoutID struct {
	Name string
}
