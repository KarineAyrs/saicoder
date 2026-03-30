package scheduler

type Task struct {
	Status string `json:"status"`
	Result string `json:"result"`
	ID     string `json:"id"`
}
