package scheduler

import (
	"context"
	"time"

	"github.com/kr/pretty"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/domain/consumer"
)

type DispatcherWorkerStorager interface {
	FindByNodeID(ctx context.Context, id string) ([]consumer.Consumer, error)
}

type DispatcherWorker struct {
	NodeID           string
	Interval         time.Duration
	Storager         DispatcherWorkerStorager
	currentConsumers []consumer.Consumer
}

func (dw *DispatcherWorker) Init() error {
	if dw.NodeID == "" {
		return errors.New("missing NodeID")
	}

	if dw.Interval <= 0 {
		return errors.New("invalid Interval")
	}

	if dw.Storager == nil {
		return errors.New("invalid Storager")
	}

	return nil
}

func (dw *DispatcherWorker) Start() {
	defer func() {
		if err := recover(); err != nil {
			go dw.Start()
		}
	}()

	for {
		consumers, err := dw.Storager.FindByNodeID(context.Background(), dw.NodeID)
		if err != nil {
			panic(err)
		}

		dw.startWorkers(dw.shouldStart(consumers))
		dw.stopWorkers(dw.shouldStop(consumers))

		dw.currentConsumers = consumers
		<-time.After(dw.Interval)
	}

}

func (dw *DispatcherWorker) Stop() {

}

func (dw *DispatcherWorker) startWorkers(consumers []consumer.Consumer) {
	pretty.Println("start: ", consumers)
}

func (dw *DispatcherWorker) stopWorkers(consumers []consumer.Consumer) {
	pretty.Println("stop: ", consumers)
}

func (dw *DispatcherWorker) shouldStart(newConsumers []consumer.Consumer) []consumer.Consumer {
	var result []consumer.Consumer

	for _, nc := range newConsumers {
		var found bool
		for _, cc := range dw.currentConsumers {
			if nc.ID == cc.ID {
				found = true
				break
			}
		}

		if !found {
			result = append(result, nc)
		}
	}

	return result
}

func (dw *DispatcherWorker) shouldStop(newConsumers []consumer.Consumer) []consumer.Consumer {
	var result []consumer.Consumer

	for _, cc := range dw.currentConsumers {
		var found bool
		for _, nc := range newConsumers {
			if nc.ID == cc.ID {
				found = true
				break
			}
		}

		if !found {
			result = append(result, cc)
		}
	}

	return result
}
