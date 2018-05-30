procurar pelo document id ordenado por revisao, pegar o menor.
deletar tudo que for menor que essa revisão.

db.getCollection('subscriptionTriggers').find({"document.id": "http://user.com/users/141"}).sort({"document.revision": 1}).limit(1);
db.getCollection('subscriptionTriggers').remove({"document.id": "http://user.com/users/141", "document.revision": { $ls: 123}});

db.products.remove(
    { qty: { $gt: 20 } },
    { writeConcern: { w: "majority", wtimeout: 5000 } }
)

isso vai acabar sendo pesado, a melhor coisa a se fazer é colocar isso em um worker. sempre que uma mensagem processar, vamos criar um debounce worker.
tipo, colocamos na fila e gravamos no banco o processamento, dai, se algo tentar rodar antes do debounce, ignoramos a mensagem. caso contrario, colocamos no banco.


estudar sobre a questao dos tempos e colisao, atomico, vector, cassandra e riak.


no futuro para evitar problemas, se um cliente ficar mt atras, podemos deletar em partes o historico dos  documentos. tipo
vamos pensar que temos 5 estados, 1 cliente esta  no primeiro e todos os outros no ultimo. nesse caso, hj, nao deletariamos  nada.
no caso otimizado, podemos deletar os estados do meio, dai qnd o cliente for pular, ele  ja pula para a ultima versao.


---
(system worker)

We need to create a system worker. This worker gonna do actions to maintain the server running well. It gonna process multiples kinds of jobs, one of then is the document/subscription cleanup.

After a document is processed by the delivery queue we should check if we should delete the old revisions of the given document. Doing this at the delivery flow is kinda bad because it envolves some queries and logics that don't belong to the delivery and may even cause a fail which disrupts the main purpose of the delivery worker.

So, after every time the deliver run, a message gonna be send to the system worker for that document to check if it has revisions to be deleted.

---
(Debounce on document cleanup worker)

The document cleanup worker should have a debounce to decrease database overhead. If we send 1000 updates for a given document, this should generate alot of unnecessary database access. 

To solve this, we should add a cleanup message everytime a delivery is processed. Then, should check if there is already something on queue waiting to be processed, in this case, the message deleted. In case of a delete, where the subscriptionTrigger, in case of MongoDB, is deleted, another worker should be generate to handle this cases, if there is any subscriptionTriger using that document version.

isso ainda eh factivel de falha, certo? por causa do debounce.

---
podemos criar um worker genérico que fica rodando outros workers. tipo, podemos colocar  os workers  de criacao d e fila, e manutencao rodando no mesmo pool.
o worker de delecao de documentos, pode  rodar em outro pool.

---

um documento chega sem subscription, descartamos.
um documento chega, atualizamos os clientes e rodamos o clean.
um outro documento chega, atualizamos os clientes e rodamos o clean.
o documento é deletado, atualizamos os clientes e rodamos o clean.

o delete é problemático porque temos uma verificacao, se o documento estiver deletado, saimos do worker.
acho que podemos tirar isso e tratar o delete de uma forma diferente.
oq podemos verificar é se tem o subscriptionTrigger, pq se nao tiver, tempos que sair mesmo.

---
qnd as filas forem ser criadas, precisamos validar mais coisas. tipo se o prefixo do sqs for muito grande, podemos estourar o limite criando as filas dos subscriptions.
entao temos que definir o tamanho maximo do prefix no caso do sqs, e isso tem que estar encapsulado dentro do sqs.

---
acho que o delete tem que ser feito no controller. o request chega de delete e ele coloca na fila.

oq pode acontecer com esse delete? oq acontece se deletar e em seguida o cliente criar?
vamos colocar na fila o delete, e o cliente vai criar, qnd o delete rodar, o documento vai ser deletado.
talvez o interessante seja descobrir qual eh a ultima revisao e mandar um delete daquela revisao para trás.
ai toda a reponsabilidade está no cliente para gerar as ids corretamente.

---
qnd um subscription for deletado, em tese é isso, sendo que pode existir casos, como no caso do mongodb que outras coisas tenham que ser deletadas também.
tipo o subscriptionTrigger, nesse caso o provider do mongo deveria ter workers e se registrar e se registrar com um nome para receber a carga.

---
o mongodb qnd ligar tem que poder se registrar em um worker. agora, quem detem essa lógica? hoje isso está sendo feito no lado de fora, deveria?
ou deveria ficar dentro do mongodb? outra coisa, deveria ter uma fila individual?

---
de qualquer forma, acho que temos que ter um outro sistema para rodar um tipo, cleanup, isso vai rodar tudo e procurar por coisas a serem deletadas.
tipo, algo que passou e nao devia ter passado.

---
quais workers precisamos?

- document_clean
	qnd deletar o resource, temos que deletar todos os documentos

- document_revision_clean
	deletar versoes antigas do documento

- subscription_clean
	qnd um subscription for deletado, precisamos deletar os subscriptions triggers.
	preciamos tb deletar as filas.

- subscription_create
	qnd um subscription é criado, precisamos criar as filas.

o problema que os workers vao variar de provider para provider, como podemos fazer isso? precisa ser generico.
hooks? callbacks?




resource

subscription
	- precisa remover os subscriptionTriggers (aws.sqs)
	- precisa criar as filas
	- precisa remover as filas (aws.sqs)
	
document
	- precisa remover os documentos
	- precisa deletar as versoes antigas do documento

---
no caso do delete do subscription, eu poderia ter um status. esse status eh tipo, ativo, inativo e sendo deletado.
dai, todo dia denoite, um worker roda para pegar quem esta sendo deletado e colocar na fila.
a questao eh, ai vou ter que ter alguem que faca isso, eleger um master, ou entaouma cron, etc...?

o etcd ia cair muito bem nesse caso.

---
quem chama o document delete revisions?
o handler no delete ou cada subscription trigger?


---
suportar delay nos workers, tipo, o error, poderia ser um error tipado, dai eu poderia retornar um delay=true e o tempo que deveria ter delay.

---
podemos colocar triggers de callback? tipo, toda vez que um resource for criado/deletado.
vão existir workers que rodam nos providers também, como resolver isso?

podemos ligar os workers e assinar os callbacks, dai vamos receber esses callbacks.  

resource:
	* cron
		* check if there is orphan documents to be deleted (mongodb)

	* delete
		* trigger no delete dos documentos

subscription:
	* cron
		* check if there is queues that needed to create sqs queues (mongodb)

	* create
		* create the queues.

	* delete
		* trigger no delete do subscriptionTrigger (mongodb)
		* delete queues

document:
	* processed
		* delete old revisions
		* compact revisions

a questao eh que eu gostaria de rodar algumas rotinas de limpeza para garantir que tudo esta funcionando corretamente.
para isso, preciso gerar apenas 1x, nao sei como... 
vamos precisar entrar na questao do cluster para isso com talvz um etcd para eleger o master, etc...
quai tarefas deveriam ser rodadas para garantir que tudo esta funcionando:

subscription:
	* check if the resource exists

document:
	* check if the subscription exists

iriamos eleger um master, esse master vai ter uma cron que ele vai executar e vai colocar as mensagens na fila.
por ser algo mais de manutencao, rodar menos vezes ou até mesmo criar um endpoint para forçar esse comando.
só podemos focar nessa tasks de cleanup e nao nas primeiras.


usar o repository hook para injetar os processamentos sem ter que ficar metrificando em tudo qnt eh lugar. vai ser mais simples e elegante.

---
no mongodb talvez seja interessante fazer algo como two phase commit. (https://docs.mongodb.com/manual/tutorial/perform-two-phase-commits/)
vamos deletar um resource. primeiro, marco ele para soft delete. em seguida, gero uma mensagem pro worker que vai entao deletar os documentos e depois deletar de vez o resource.
a cada x tempo, um worker roda para pegar resources que estao em soft delete e ainda não foram deletados, pegando, ele gera a mensagem e repete o processamento.

talvez seja interessante marcar no documento quantas vezes ja estamos fazendo isso para detectar falha.

---
preciso criar um worker generico agora e dar a trigger no cara para colocar na fila.

------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
---
# <img src="misc/doc/logo.png" border="0" alt="flare" height="45">
<a href="https://travis-ci.org/diegobernardes/flare"><img src="https://img.shields.io/travis/diegobernardes/flare/master.svg?style=flat-square" alt="Build Status"></a>
<a href="https://coveralls.io/github/diegobernardes/flare"><img src="https://img.shields.io/coveralls/diegobernardes/flare/master.svg?style=flat-square" alt="Coveralls"></a>
<a href="https://godoc.org/github.com/diegobernardes/flare"><img src="https://img.shields.io/badge/api-reference-blue.svg?style=flat-square" alt="GoDoc"></a>

Flare is a service that listen to changes on HTTP endpoints and notify subscripted clients about the changes. It help reduce the pressure on APIs by avoiding the clients to do pooling requests to search for new/changed content and the need of the APIs to develop workers to notify the clients about the.

There is no need to the the service know anything about who is consuming it's updates, this is abstracted and lead to a simpler design on APIs. Problems like scaling the workers to notify the changes if the number of subscriptions increase, need to control the delivery success of the messages, include/update/delete the clients on your subscription list and so on are just solved with Flare.

## How to run
```bash
go get github.com/diegobernardes/flare
cd $GOPATH/src/github.com/diegobernardes/flare/service/flare/cmd
go run flare.go start
```

There is a example at `misc/example`, it's a `docker-compose` that starts a Flare server, a producer and a consumer. From times to times the producer create/update/delete a given document and the consumer receives this changes from Flare. You must have `docker-compose` and `docker` to run this example.

```bash
go get github.com/diegobernardes/flare
cd $GOPATH/src/github.com/diegobernardes/flare/misc/example
make run
```

There is also a Docker image:
```bash
docker run --rm -p 8080:8080 diegobernardes/flare:v0.1.0-alpha
```

## How it works
Flare has 3 basic entities: `Resource`, `Subscription` and `Document`. The origin of content is responsible for `Resource` and `Document` entities and the clients are responsible for `Subscription`.

### Resource
Resource represents the entity you want to track. It cannot be updated, only deleted, and to delete, first you need to remove all the associated subscriptions.


| Field  | Description |
| ------------- | ------------- |
| `endpoint` | Is the actual document that gonna be tracked. `wildcards` are required to track the collection and they can be later used at subscriptions. |
| `change.field` | The field that is used to track changes on a document. It can be a string containing a date or a auto incremented integer. |
| `change.format` | If the field is a date, this fields has the format to parse the document date. More info about the format [here](https://golang.org/pkg/time/#pkg-constants). |

Endpoint: `POST http://flare.com/resources`
```json
{
	"endpoint": "http://api.company.com/users/{id}",
	"change": {
		"field": "updatedAt",
		"format": "2006-01-02T15:04:05Z07:00"
	}
}
```

### Subscription
Subscription is the responsible to notify the clients when a document from a resource changes.

| Field  | Description |
| ------------- | ------------- |
| `endpoint.url` | The address of the client that gonna receive the notification. |
| `endpoint.method` | The method used on the notification request. |
| `endpoint.headers` | A list of headers to sent within the request. |
| `endpoint.actions.(create,update,delete).(url,method,headers)` | Override of attributes per action. |
| `delivery.success` | List of success status code. This is used to mark the notification as delivered for the respective client. |
| `delivery.discard` | List of status code to discard the notification. |
| `content.document` | Send the document. |
| `content.envelope` | Send the document, if marked, inside a envelope with some metadata. |
| `data` | Can only be set if `content.envelope` is true. Can be used to provide aditional information to the client that gonna receive the notification. It also can interpolate wildcards used at resource endpoint definition. |

Endpoint: `POST http://flare.com/resources/:resource-id/subscriptions`
```json
{
	"endpoint": {
		"url": "http://api.company.com/wishlist/{id}",
		"method": "post",
		"headers": {
			"Authorization": [
				"Basic YWxhZGRpbjpvcGVuc2VzYW1l"
			]
		},
		"actions": {
			"delete": {
				"method": "delete"
			}
		}
	},
	"delivery": {
		"success": [200],
		"discard": [500]
	},
	"content": {
		"envelope": true,
		"document": true
	},
	"data": {
		"service": "user",
		"id": "{id}"
	}
}
```

### Document
To update a document, a `PUT` should be done at `http://flare.com/documents/{endpoint}`, where the `{endpoint}` is the real document endpoint and it should match the information inserted at the resource creation. The body should contain the document.
If the origin send the same document or older documents more then one time, the service don't gonna notify the clients again because it know the document version each client has. The notification only happens when is really needed.

The delete should be sent with the delete method and no body.