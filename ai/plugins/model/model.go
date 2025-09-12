package model

import (
	"github.com/firebase/genkit/go/ai"
)

type Model struct {
	provider string
	label    string // Label is the internal model representation of different providers.
	info     ai.ModelSupports
}

func New(provider string, label string, info ai.ModelSupports) Model {
	return Model{
		provider: provider,
		label:    label,
		info:     info,
	}
}

// Key is the model query string of genkit registry.
func (m Model) Key() string {
	return m.provider + "/" + m.label
}

func (m Model) Info() ai.ModelOptions {
	return ai.ModelOptions{
		Label:    m.label,
		Supports: &m.info,
		Versions: []string{m.label},
	}
}
