package pgerr

const (
	UniqueViolation       = "23505"
	ForeignKeyViolation   = "23503"
	CheckViolation        = "23514"
	NotNullViolation      = "23502"
	NotFound              = "P0002"
	InvalidTextRepresent  = "22P02" // invalid input syntax for type
	StringTooLong         = "22001"
	DivisionByZero        = "22012"
	DeadlockDetected      = "40P01"
	SerializationFailure  = "40001"
	InsufficientPrivilege = "42501"
	UndefinedTable        = "42P01"
	UndefinedColumn       = "42703"
)
