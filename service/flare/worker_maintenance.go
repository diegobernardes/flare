package flare

import "context"

type workerMaintenance struct{}

/*
preciso ter um worker dispatcher que vai ser um worker para trabalhar com tarefas do flare.
tipo, qnd algo for deletado, ele vai cuidar de limpar a bagunca.

flare-maintenance

tudo vai cair aqui

{"action": "delete-resource", "id": "123"}
{"action": "delete-subscription", "id": "456", "resourceID": "123"}
{"action": "clean-document-state", "documentID": "789"}

esse worker não faz parte do dominio, logo, onde vamos colocar? talvez no service/flare.
temos que injetar ele
*/

// isso aqui serve só para colocar na fila.
func (wm *workerMaintenance) DeleteResource(ctx context.Context, id string) error {
	return nil
}

func (wm *workerMaintenance) DeleteSubscription(ctx context.Context, id string) error {
	return nil
}

/*
	- qnd um subscription é deletado, devemos tirar todos os subscriptionTriggers. o problema que essa
	abstração é do mongodb e não do flare. logo, como vamos fazer isso? só se eu chamar callbacks
	que vão estar definidos nos repositórios. se não retornar erro, continuo com o processamento.

	- qnd um resource é deletado, devemos tirar todos os documentos associados a esse resource.

	- qnd um documento é atualizado/deletado devemos procurar o passado e deletar as as revisões
	passadas. no caso do delete, não tem body.
*/
