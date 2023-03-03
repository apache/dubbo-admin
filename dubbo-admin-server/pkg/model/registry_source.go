package model

type RegistrySource int

const (
	All RegistrySource = iota

	Interface

	Instance
)
