package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var _ Store = (*JSONStore)(nil)

// JSONStore is a naive implementation of a Store. It persists the data as a human-readable JSON file on disc.
type JSONStore struct {
	dir string

	mu sync.Mutex
}

func NewJSONStore(path string) *JSONStore {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		panic("could not create path: " + path + ": " + err.Error())
	}

	return &JSONStore{dir: path, mu: sync.Mutex{}}
}

func (s *JSONStore) Store(fileName string, data any) error {
	if data == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Create(filepath.Join(s.dir, fileName))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStore, err) //nolint:errorlint // prevent err in api
	}
	defer file.Close()

	b, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStore, err) //nolint:errorlint // prevent err in api
	}

	_, err = io.Copy(file, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrStore, err) //nolint:errorlint // prevent err in api
	}

	return nil
}

func (s *JSONStore) Load(fileName string, data any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(filepath.Join(s.dir, fileName))
	if err != nil {
		return fmt.Errorf("%w: %w", ErrLoad, err)
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrLoad, err) //nolint:errorlint // prevent err in api
	}

	return nil
}
