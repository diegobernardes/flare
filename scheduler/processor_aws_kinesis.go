package scheduler

import (
	"sync"

	"github.com/diegobernardes/flare/domain/consumer"
	"github.com/diegobernardes/flare/provider/aws/kinesis"
)

// temos esses 4 estados, no deletando, vamos sair.
// no creating e updating, vamos esperar chegar no active.
// CREATING, DELETING, ACTIVE, UPDATING

/*
	temos que persistir o estado do sqs, precisamos do stream, do shard id e do sequence.
	se um shard for deletado, vamos deletar os dados do banco.
*/

type processorAWSKinesis struct {
	consumer consumer.Consumer
	kinesis  kinesis.Client

	mutex            sync.Mutex
	processingShards map[string]bool
}

func (p *processorAWSKinesis) checkShards() {
	shards, err := p.kinesis.FetchShards()
	if err != nil {
		panic(err)
	}

	for _, shard := range shards {
		if _, ok := p.processingShards[*shard.ShardId]; !ok {
			// disparar o processamento do shard, achamos outro!
		}
	}

}

func (p *processorAWSKinesis) process(content []byte) error {
	return nil
}

/*
	toda vez que um nó subir, ele vai ter que ir se registrar no cluter.

	toda vez que um processo subir, vamos cadastrar ele no banco em um nó de nodes.

	os nodes vão ter um ttl, o processo precisa ficar mandando keep alives para o o banco resetando
	o tempo, toda vez.

	se um nó novo entrar, devemos recalcular a merda toda.

	no cassandra, posso lockar o consumer para um node.

*/
