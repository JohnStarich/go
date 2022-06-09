package scutil

// Flag is an scutil flag value. See possible Flag enum values for examples.
type Flag string

// Common flag values for convenient comparisons
const (
	// Scoped requires a resolver to only send queries on the specified interface
	Scoped Flag = "Scoped"

	RequestARecords    Flag = "Request A records"
	RequestAAAARecords Flag = "Request AAAA records"
)
