package node

import (
	"time"

	uuid "github.com/satori/go.uuid"
)

type Node struct {
	ID        string
	CreatedAt time.Time
}

func (n *Node) Init() {
	n.ID = uuid.NewV4().String()
}
