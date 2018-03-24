package cluster

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

// se o for reiniciar, tenho que reiniciar os estados, mutex, loaded, etc.. etc..
// tratar os casos de erro aqui, tem muito caso... sempre tem que verificar se o cara existe no
// estado, nao podemos assumir que ele sempre vai existir.

// sempre que um nó novo entrar, devemos verificar todos os consumers que estão sem nó para
// processar e dar assign em todos eles.

// NodeFetcher represent the oring of node changes.
type NodeFetcher interface {
	Load(ctx context.Context) (nodeIDS []string, err error)
	Watch(ctx context.Context, fn func(nodeID, action string) error) context.Context
}

// SchedulerConsumerFetcher represent the origin of consumer changes.
type SchedulerConsumerFetcher interface {
	Load(ctx context.Context) (consumers []Consumer, err error)
	Watch(ctx context.Context, fn func(consumer Consumer, action string) error) context.Context
}

// ConsumerAssigner represent the actions to assign and unassign consumers to nodes.
type ConsumerAssigner interface {
	Assign(ctx context.Context, consumerID, nodeID string) error
	Unassign(ctx context.Context, nodeID string) error
}

// Schedule is used to keep track of nodes and consumers to assign the then to nodes.
type Schedule struct {
	Node            NodeFetcher
	ConsumerFetcher SchedulerConsumerFetcher
	Assigner        ConsumerAssigner
	Logger          log.Logger
	ctx             context.Context
	ctxCancel       func()
	wg              sync.WaitGroup
	mutex           sync.Mutex
	state           map[string][]Consumer
	loaded          sync.WaitGroup
}

// Init check if all the requirements are fulfilled.
func (s *Schedule) Init() error {
	if s.Node == nil {
		return errors.New("missing Node")
	}

	if s.ConsumerFetcher == nil {
		return errors.New("missing Consumer")
	}

	if s.Assigner == nil {
		return errors.New("missing Assigner")
	}

	if s.Logger == nil {
		return errors.New("missing Logger")
	}

	s.state = make(map[string][]Consumer, 0)
	return nil
}

// Start the service.
func (s *Schedule) Start() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				level.Error(s.Logger).Log(
					"message", "panic recovered", "error", err, "stacktrace", string(debug.Stack()),
				)

				if s.ctx.Err() == nil {
					s.mutex = sync.Mutex{}
					go s.Start()
				}
			}
			s.wg.Done()
		}()

		s.ctx, s.ctxCancel = context.WithCancel(context.Background())
		for {
			ctx, ctxCancel := context.WithCancel(s.ctx)
			defer ctxCancel()

			s.loaded = sync.WaitGroup{}
			s.loaded.Add(1)
			nctx := s.Node.Watch(ctx, s.handleNodeUpdates)
			cctx := s.ConsumerFetcher.Watch(ctx, s.handleConsumerUpdates)

			if err := s.load(ctx); err != nil {
				level.Error(s.Logger).Log("message", "error during initial load", "error", err)
				ctxCancel()
				continue
			}

			select {
			case <-ctx.Done():
				break
			case <-nctx.Done():
			case <-cctx.Done():
			}

			ctxCancel()
		}
	}()
}

// Stop the service.
func (s *Schedule) Stop() {
	s.ctxCancel()
	s.wg.Wait()
}

func (s *Schedule) load(ctx context.Context) error {
	s.mutex.Lock()
	defer func() {
		s.mutex.Unlock()
		s.loaded.Done()
	}()

	nodeIDS, err := s.Node.Load(ctx)
	if err != nil {
		return errors.Wrap(err, "error during nodes load")
	}

	consumers, err := s.ConsumerFetcher.Load(ctx)
	if err != nil {
		return errors.Wrap(err, "error during consumers load")
	}

	for _, nodeID := range nodeIDS {
		s.state[nodeID] = make([]Consumer, 0)
	}

	for _, consumer := range consumers {
		if consumer.NodeID == "" {
			consumer.NodeID = s.selectNode()
			if err := s.Assigner.Assign(s.ctx, consumer.ID, consumer.NodeID); err != nil {
				return errors.Wrapf(
					err, "error during assign consumer '%s' to node '%s'", consumer.ID, consumer.NodeID,
				)
			}
		}

		s.state[consumer.NodeID] = append(s.state[consumer.NodeID], consumer)
	}

	return nil
}

func (s *Schedule) handleNodeUpdates(nodeID, action string) error {
	s.loaded.Wait()
	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch action {
	case ActionCreate:
		s.state[nodeID] = make([]Consumer, 0)
	case ActionDelete:
		consumers := s.state[nodeID]
		delete(s.state, nodeID)

		for _, consumer := range consumers {
			consumer.NodeID = s.selectNode()
			s.state[consumer.NodeID] = append(s.state[consumer.NodeID], consumer)
			if err := s.Assigner.Assign(s.ctx, consumer.ID, consumer.NodeID); err != nil {
				return errors.Wrapf(
					err, "error during assign consumer '%s' to node '%s'", consumer.ID, consumer.NodeID,
				)
			}
		}
	}

	return nil
}

func (s *Schedule) handleConsumerUpdates(consumer Consumer, action string) error {
	s.loaded.Wait()
	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch action {
	case ActionCreate:
		consumer.NodeID = s.selectNode()
		if err := s.Assigner.Assign(s.ctx, consumer.ID, consumer.NodeID); err != nil {
			return errors.Wrapf(
				err, "error during assign consumer '%s' to node '%s'", consumer.ID, consumer.NodeID,
			)
		}
		s.state[consumer.NodeID] = append(s.state[consumer.NodeID], consumer)
	case ActionDelete:
		consumers := s.state[consumer.NodeID]
		for i, c := range consumers {
			if c.ID == consumer.ID {
				consumers = append(consumers[:i], consumers[i+1:]...)
				break
			}
		}
		s.state[consumer.NodeID] = consumers
	}

	return nil
}
func (s *Schedule) selectNode() string {
	var (
		id  string
		qtd int
	)

	for nodeID, consumers := range s.state {
		consumerQtd := len(consumers)
		if id == "" {
			id = nodeID
			qtd = consumerQtd
			continue
		}

		if consumerQtd < qtd {
			id = nodeID
			qtd = consumerQtd
		}
	}

	return id
}
