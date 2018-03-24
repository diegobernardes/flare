package cluster

// Tasker represent a work unit that can be started and stopped.
type Tasker interface {
	Start()
	Stop()
}

const (
	ActionCreate = "create"
	ActionDelete = "delete"
	ActionUpdate = "update"
)

// mudar a implementacao do consumer...
type Consumer struct {
	ID     string
	NodeID string
}
