// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package worker

import "context"

// toda vez que um subscription for deletado, esse cara vai ser chamado.
// dai ele vai tentar deletar o subscription e em seguida apagar os triggers.

type SubscriptionDelete struct{}

func (sd *SubscriptionDelete) Process(ctx context.Context, content []byte) error {
	/*
		primeiro tentar deletar o subscription, consegui? se sim, ok
		se nao, foi pq ele nao existe? ok
		se for outro erro, tentar novamente

		tentar deletar todos os subscriptions, se sim, ok, se nao, dar erro.
		deletar com um cursor ou algo parecido para evitar sobrecargar o banco.

		toda vez que o delete for chamado, internamente ele vai chamar esse worker.

		talvez o retorno do delete seja o error e um status, esse status pode ser do tipo, deletei
		agora ou entao, esta na fila e vai ser deletado em breve.
	*/
	return nil
}
