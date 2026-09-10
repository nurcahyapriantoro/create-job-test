package constant

type contextKey string

const DataloaderContextKey contextKey = "dataloader"

const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusFailed    = "failed"
	StatusCompleted = "completed"
)
