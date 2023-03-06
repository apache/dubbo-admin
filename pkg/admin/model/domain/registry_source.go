package domain

type RegistrySource int

const (
	ALL RegistrySource = iota
	INTERFACE
	INSTANCE
)
