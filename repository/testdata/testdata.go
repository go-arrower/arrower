package testdata

import (
	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
)

type EntityID string

type Entity struct {
	ID   EntityID
	Name string
}

type EntityWithoutID struct {
	Name string
}

type (
	EntityIDInt     int
	EntityIDUint    uint
	EntityWithIntPK struct {
		ID     EntityIDInt
		UintID EntityIDUint // todo check or remove
		Name   string
	}
)

var DefaultEntity = TestEntity()

func TestEntity() Entity {
	return Entity{
		ID:   EntityID(uuid.New().String()),
		Name: gofakeit.Name(),
	}
}
