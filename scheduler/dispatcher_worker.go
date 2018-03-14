package scheduler

import (
	"context"
	"time"

	"github.com/diegobernardes/flare/domain/consumer"
)

// varrer o banco procurando por coisas para rodar, qnd pegar, guardar um etado e comecar a rodar.
// na prox vez, se o cara nao estiver no estado, paro de programar.
// qnd for pegar para processar, eu preciso dar um lock primeiro.

type DispatcherWorkerStorager interface {
	FindByNodeID(ctx context.Context, id string) ([]consumer.Consumer, error)
}

type DispatcherWorker struct {
	NodeID   string
	Interval time.Duration

	currentConsumers []consumer.Consumer
}

func (dw *DispatcherWorker) Start() {
	defer func() {
		if err := recover(); err != nil {
			go dw.Start()
		}
	}()

	for {

		<-time.After(dw.Interval)
	}

}

func (dw *DispatcherWorker) Stop() {

}
