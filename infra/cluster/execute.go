package cluster

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/kr/pretty"
	"github.com/pkg/errors"

	"github.com/diegobernardes/flare/domain/consumer"
	"github.com/diegobernardes/flare/infra/cluster/execute"
)

type ExecuteRunner interface {
	Start() error
	Stop() error
}

/*
	mt parecido com o schedule, sendo que aqui eh para rodar.
	a interface vai passar o nodeid, e vamos pegar os consumers que temos para processar.

	vamos ficar ouvindo pela criacao de novos consumers ou atualizacao, se ja estiver rodando,
	restartar, se nao, soh rodar.

	temos que fazer o processador do sqs e do kinesis. no caso do kinesis, temos que guardar o estado
	de onde esta o processamento por shard.

	/aws-kinesis/:id/shard/:number value

	criar o processador do sqs e do kinesis, vamos mandar para o domain processar. la vamos apenas
	printar e vamos colocar uns rand dando erro para poder ver como funcionamos em cadso de erro.
*/
type ExecuteConsumerFetcher interface {
	Load(ctx context.Context, id string) (consumers []consumer.Consumer, err error)
	Watch(
		ctx context.Context, fn func(consumer consumer.Consumer, action string) error, id string,
	) context.Context
}

// nao pode ser o consumer fetcher, tem que ser o execute pq precisamos passar a id.
type Execute struct {
	ConsumerFetcher ExecuteConsumerFetcher
	Logger          log.Logger
	NodeID          string
	runners         map[string]ExecuteRunner
	mutex           sync.Mutex
	wg              sync.WaitGroup
	loaded          sync.WaitGroup
	ctx             context.Context
	ctxCancel       func()
}

func (e *Execute) Init() error {
	if e.ConsumerFetcher == nil {
		return errors.New("missing ConsumerFetcher")
	}

	if e.Logger == nil {
		return errors.New("missing Logger")
	}

	if e.NodeID == "" {
		return errors.New("invalid NodeID")
	}

	return nil
}

func (e *Execute) Start() {
	defer func() {
		if err := recover(); err != nil {
			level.Error(e.Logger).Log(
				"message", "panic recovered", "error", err, "stacktrace", string(debug.Stack()),
			)

			if e.ctx.Err() == nil {
				e.mutex = sync.Mutex{}
				go e.Start()
			}
		}
		e.wg.Done()
	}()

	e.ctx, e.ctxCancel = context.WithCancel(context.Background())
	e.runners = make(map[string]ExecuteRunner)
	for {
		ctx, ctxCancel := context.WithCancel(e.ctx)
		defer ctxCancel()

		e.loaded = sync.WaitGroup{}
		e.loaded.Add(1)
		cctx := e.ConsumerFetcher.Watch(ctx, e.handleConsumerUpdates, e.NodeID)

		if err := e.load(ctx); err != nil {
			level.Error(e.Logger).Log("message", "error during initial load", "error", err)
			ctxCancel()
			continue
		}

		select {
		case <-ctx.Done():
			break
		case <-cctx.Done():
		}

		ctxCancel()
	}
}
func (e *Execute) load(ctx context.Context) error {
	e.mutex.Lock()
	defer func() {
		e.mutex.Unlock()
		e.loaded.Done()
	}()

	consumers, err := e.ConsumerFetcher.Load(ctx, e.NodeID)
	if err != nil {
		return errors.Wrap(err, "error during consumers load")
	}

	// iniciar os consumers
	pretty.Println(consumers)

	return nil
}

func (e *Execute) handleConsumerUpdates(consumer consumer.Consumer, action string) error {
	if consumer.Source.AWSSQS != nil {
		sqs := execute.AWSSQS{}
		pretty.Println(sqs.Start())
	}

	pretty.Println("consumer: ", consumer, " - action: ", action)
	return nil
}

func (e *Execute) Stop() {
	e.ctxCancel()
	e.wg.Wait()
}

// inciar o consumo. como vamos fazer isso?
