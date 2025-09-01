package arepo

import "errors"

var (
	ErrStore = errors.New("could not store repository data")
	ErrLoad  = errors.New("could not load repository data")
)

// Store is an interface to access the data of a MemoryRepository as a whole,
// so it can be persisted easily.
type Store interface {
	Store(fileName string, data any) error
	Load(fileName string, data any) error
}

var NoopStore Store = &noopStore{} //nolint:gochecknoglobals // pattern from std lib slog.DiscardHandler

type noopStore struct{}

func (n noopStore) Store(_ string, _ any) error {
	return nil
}

func (n noopStore) Load(_ string, _ any) error {
	return nil
}
