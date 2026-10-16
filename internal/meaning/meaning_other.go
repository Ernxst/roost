//go:build !darwin

package meaning

import "errors"

// Model is a sentence embedding, available only on macOS.
type Model struct{}

func New() (*Model, error) { return nil, errors.New("sentence embeddings need macOS") }

func (m *Model) Name() string { return "" }

func (m *Model) Vector(string) ([]float32, bool) { return nil, false }
