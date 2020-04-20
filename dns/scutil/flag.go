package scutil

type Flag string

const (
	// Scoped requires a resolver to only send queries on the specified interface
	Scoped Flag = "Scoped"

	RequestARecords Flag = "Request A records"
)
