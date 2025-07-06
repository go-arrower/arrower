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

type EntityWithoutID struct { // todo can this be removed? Since the third constructor is gone from the TestSuite, this one might not be required any longer
	Name string
}

type (
	EntityIDInt     int
	EntityIDUint    uint
	EntityWithIntPK struct {
		ID     EntityIDInt
		UintID EntityIDUint
		Name   string
	}
)

var DefaultEntity = RandomEntity()

func RandomEntity() Entity {
	return Entity{
		ID:   EntityID(uuid.New().String()),
		Name: gofakeit.Name(),
	}
}
