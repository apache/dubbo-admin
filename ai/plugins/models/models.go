package models

import (
	"github.com/firebase/genkit/go/ai"
)

type Model struct {
	provider string
	label    string
	info     ai.ModelSupports
}

func New(provider string, label string, info ai.ModelSupports) Model {
	return Model{
		provider: provider,
		label:    label,
		info:     info,
	}
}

func (m Model) Key() string {
	return m.provider + "/" + m.label
}

func (m Model) Info() ai.ModelInfo {
	return ai.ModelInfo{
		Label:    m.label,
		Supports: &m.info,
		Versions: []string{m.label},
	}
}
