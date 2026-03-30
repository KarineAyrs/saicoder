package domain

// Task from scheduler.
type Task struct {
	ID        string `json:"id"`
	Statement string `json:"statement"`
}

type Status string

const (
	Queued     Status = "QUEUED"
	Processing Status = "PROCESSING"
	Done       Status = "DONE"
	Failed     Status = "FAILED"
	Discarded  Status = "DISCARDED"
)

type Result struct {
	Status Status `json:"status"`
	Base64 string `json:"base64"`
	ID     string `json:"id"`
}
