package cluster

import (
	"context"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

const (
	ConsumerStatusCreate    = "create"
	ConsumerStatusDelete    = "delete"
	ConsumerStatusUnchanged = "unchanged"
	consumerStatusAssign    = "assign"
)

const (
	NodeStatusCreate    = "create"
	NodeStatusDelete    = "delete"
	NodeStatusUnchanged = "unchanged"
)

type ConsumerStatus struct {
	ID     string
	NodeID string
	Status string
}

type NodeStatus struct {
	ID     string
	Status string
}

type ConsumerFetcher interface {
	Fetch(ctx context.Context, fn func(ConsumerStatus) error)
	Assign(ctx context.Context, consumerID, nodeID string) error
	Unassign(ctx context.Context, consumerID string) error
}

type NodeFetcher interface {
	Fetch(ctx context.Context, fn func(NodeStatus) error)
}

type Schedule struct {
	Consumer ConsumerFetcher
	Node     NodeFetcher
	Logger   log.Logger

	ctx       context.Context
	ctxCancel func()
	nodes     []string
	consumers []string
	mutex     sync.Mutex
	wg        sync.WaitGroup
	state     map[string][]string
}

func (s *Schedule) Init() error {
	if s.Consumer == nil {
		return errors.New("missing Consumer")
	}

	if s.Node == nil {
		return errors.New("missing Node")
	}

	if s.Logger == nil {
		return errors.New("missing Logger")
	}

	s.state = make(map[string][]string)
	s.ctx, s.ctxCancel = context.WithCancel(context.Background())
	return nil
}

func (s *Schedule) Start() {
	go func() {
		s.wg.Add(1)
		defer s.wg.Done()

		defer func() {
			if err := recover(); err != nil {
				level.Error(s.Logger).Log("message", "panic during schedule", "reason", err)
				go s.Start()
			}
		}()

		s.Node.Fetch(s.ctx, s.handleNodeStatus)
		s.Consumer.Fetch(s.ctx, s.handleConsumerStatus)
		<-s.ctx.Done()
	}()
}

func (s *Schedule) Stop() {
	s.ctxCancel()
	s.wg.Wait()
}

func (s *Schedule) handleConsumerStatus(status ConsumerStatus) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.consumeUpdateConsumer(status)
}

func (s *Schedule) handleNodeStatus(status NodeStatus) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.consumeUpdateNode(status)
}

// se por acaso o node nao existir, chamar o unassign.
func (s *Schedule) consumeUpdateNode(status NodeStatus) error {
	switch status.Status {
	case NodeStatusUnchanged, NodeStatusCreate:
		s.state[status.ID] = make([]string, 0)
	case NodeStatusDelete:
		if len(s.state[status.ID]) != 0 {
			for _, id := range s.state[status.ID] {
				cstatus := ConsumerStatus{
					ID:     id,
					NodeID: status.ID,
					Status: consumerStatusAssign,
				}

				if err := s.consumeUpdateConsumer(cstatus); err != nil {
					return errors.Wrap(err, "error during consumer update")
				}
			}
		}

		delete(s.state, status.ID)
	}

	return nil
}

func (s *Schedule) consumeUpdateConsumer(status ConsumerStatus) error {
	switch status.Status {
	case ConsumerStatusUnchanged:
		s.state[status.NodeID] = append(s.state[status.NodeID], status.ID)
	case consumerStatusAssign:
		status.Status = ConsumerStatusCreate
		if err := s.consumeUpdateConsumer(status); err != nil {
			return err
		}

		ids := s.state[status.NodeID]
		for i := 0; i < len(ids); i++ {
			if ids[i] == status.NodeID {
				ids = append(ids[:i], ids[i+1:]...)
			}
		}
	case ConsumerStatusCreate:
		var (
			lowerCount int
			lowerID    string
		)

		for nodeID, consumerIDS := range s.state {
			if lowerID == "" {
				lowerID = nodeID
				continue
			}

			if len(consumerIDS) > lowerCount {
				lowerID = nodeID
			}
		}

		if lowerID == "" {
			return errors.New("could not find a node to process the consumer")
		}

		if err := s.Consumer.Assign(s.ctx, status.ID, lowerID); err != nil {
			return errors.Wrapf(err, "error during assign consumer '%s' to node '%s'", status.ID, lowerID)
		}

		s.state[status.NodeID] = append(s.state[status.NodeID], status.ID)
	case ConsumerStatusDelete:
		ids := s.state[status.NodeID]
		for i := 0; i < len(ids); i++ {
			if ids[i] == status.ID {
				ids = append(ids[:i], ids[i+1:]...)
				break
			}
		}
	}

	return nil
}
