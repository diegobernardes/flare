package scheduler

import (
	"context"
	"time"

	"github.com/diegobernardes/flare/domain/consumer"
	"github.com/diegobernardes/flare/scheduler/node"
)

type Dispatcher interface {
	Fetch(ctx context.Context, time *time.Time) ([]consumer.Consumer, error)
	Assign(ctx context.Context, consumerID, nodeID string) error
	// Unassign(ctx context.Context, consumerID string) error
}

type consumerProcessor interface {
	Process(consumer consumer.Consumer, payload []byte) error
}

// Locker is used to lock a given key within the cluster.
type Locker interface {
	Lock(ctx context.Context, key, nodeID string, ttl time.Duration) (bool, error)
	Refresh(ctx context.Context, key, nodeID string, ttl time.Duration) (bool, error)
	Release(ctx context.Context, key, nodeID string) error
}

// Cluster control the information about the cluster.
type Cluster interface {
	Join(ctx context.Context, id string, ttl time.Duration) error
	KeepAlive(ctx context.Context, id string, ttl time.Duration) error
	Leave(ctx context.Context, id string) error
	Nodes(ctx context.Context, time *time.Time) ([]node.Node, error)
}

/*
	pegamos 1 consumer e attribuimos ele para um node.
	esse node vai verificar na tabela, vai ver que ele tem algo novo para processar evai comecar.

	oq acontece se o nó morrer?

	vamos pegar os consumers que o nó estava processando e vamos atribuir para outros nós. mas e se ele
	ainda estiver processando!?

	o consumer vai ter que ter um state: processing ou algo do tipo, dai quem estiver consumindo vai
	ter que ficar marcando esse cara como processando.
	o novo nó sò vai comecar a processar qnd o state estiver nulo.

	podemos usar o mesmo lock? vamos atribuir as ids dos nos nos consumers.
	o consumer qnd rodar, vai tentar dar um lock, ele soh vai conseguir processar qnd tiver o lock.
	o oturo no, qnd perceber que nao esta mais com ele, vai liberar o lock.

	futuramente levar outras informacoes em consideracao como cpu, memoria, goroutines, shards do
	kinesis, etc...
*/
