package repository_test

import (
	"context"
	"fmt"

	"github.com/go-arrower/arrower/tests"
)

func Example_overwriteRepositoryMethodWithOwnBehaviour() {
	ctx := context.Background()

	repo := NewElementMemoryRepository()
	repo.Add(ctx, Element{ID: 1})

	c, _ := repo.Count(ctx)
	fmt.Println(c)

	// Output: -1
}

type Element struct {
	ID int
}

func NewElementMemoryRepository() *ElementMemoryRepository {
	return &ElementMemoryRepository{
		MemoryRepository: tests.NewMemoryRepository[Element, int](),
	}
}

type ElementMemoryRepository struct {
	*tests.MemoryRepository[Element, int]
}

// Count overwrites the existing Count method with your own implementation.
func (repo *ElementMemoryRepository) Count(_ context.Context) (int, error) {
	return -1, nil
}
