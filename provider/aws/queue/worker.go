// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue

import (
	"context"
	"fmt"
)

type Worker struct {
	kv kv
}

func (*Worker) unmarshal(rawContent []byte) (action, subscriptionID string, err error) {
	return "", "", nil
}

func (*Worker) Process(ctx context.Context, rawContent []byte) error {
	return nil
}

func (*Worker) Delete(ctx context.Context, id string) error {
	return nil
}

/*
	toda vez que chamo o create, temos que verificar se ja existe, se existir, vamos seguir em frente.


	o delete eh o mais facil, qnd recebermos a mensagem, vamos verificar se existe algum subscription
	com a id que temos, se nao tiver, deletamos, se der erro falando que a fila nao existe, ignoramos.
	se der outro erro retentamos e se o subscription existir, vamos ignorar a mensagem.

	o problema vai ser qnd a gente criar o subscription e o prox comando que seria colocar na fila,
	falhar ou der timeout, nesse momento o sistema vai parar de funcionar.
*/
func (w *Worker) Create(ctx context.Context, id string) error {
	exists, err := w.kv.Exists(ctx, fmt.Sprintf("queue-%s", id))
	if err != nil {
		panic(err)
	}
	if exists {
		return nil
	}

	// o problema que pode dar erro no meio do caminho....
	// mas que merda eim...

	/*
		imagina que eu veja que nao existe, blz, nao existe
		dai 2 coisas:

		envio a mensagem e marco no banco ou marco no bacno e envio a mensagem, em qualquer um dos dois
		na segunda operacao pode haver um erro
	*/

	return nil
}

// o kv tem que ser atomico e safe, tipo etcd.
type kv interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, content []byte) error
	Exists(ctx context.Context, key string) (bool, error)
}

/*
	worker dispatcher, recebe a task e passa para alguem.

	{"id": "queue-create", "subscriptionID": "123"}
	{"id": "queue-delete", "subscriptionID": "123"}

	o banco talvez tenha uma interface KV, essa interface vai ser usada para a gente fazer os nossos
	locks e controles.

	Get
	Set
	Exists
*/
