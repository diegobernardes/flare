package scheduler

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Node struct {
	ID        string
	CreatedAt time.Time
}

func (n *Node) init() {
	n.ID = uuid.NewV4().String()
}
