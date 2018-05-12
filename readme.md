acho que podemos criar um sistema de backup. tipo, esse sistema roda de tempos em tempos para verificar se todas as filas estao criadas.
se nao estiverem, ele joga em um worker para poder criar. como o endpoint de status funcionaria no modelo de sass? vamos exibir informacoes para todos!?

entao vale a pena voltar para o modelo antigo!?

---
bom nesse branch vamos individualizar as filas, cada subscription vai ter uma fila separada e o worker vai ficar processando individualmente as filas.
vamos checar se existe e se nao existir, criamos.

como vamos colocar na fila? qnd um documento for processado pelo worker ele vai chegar no spread, do spread para o delivery como vamos fazer?
como o spread sabe a fila que ele tem que colocar a mensagem para processar? vamos criar uma funcao getQueue que ela vai retornar e o spread vai prosseguir.

essa funcao vai estar aonde? vai cachear algo?

onde que eu defino essa regra? toda vez que um subscription for criado, devo criar também uma fila no sqs? dentro do sqs?
eu to fazendo isso pq nao consigo dar um get filtrado, se desse, nao teria feito isso. logo, isso nao faz parte do dominio.

---
por ser algo que eu preciso garantir, talvez a criacao/delecao da fila tenha que acontecer por um worker.
se tentar pegar um queue e nao tiver, vamos retornar nulo dai quem for usar, se ver que nao tem nada ainda, soh avanca sem fazer nada.

---
o worker pode ser generico, tipo, vamos dar assign de um worker para algo que vai ser processado, tipo, `queue-create` vai para um lugar e o `queue-delete` para outro.

---
no caso do sqs podemos fazer um sistema de auto fixup, tipo, se tentar dar um pull e a fila nao existir, logo, precisamos criar, ou seja, chamar o worker de criar.
mas oq podemos fazer para nao ter que colocar a mensagem na fila trocentas vezes a mesma mensagem? podemos guardar no banco uma flag para isso?
tipo, `create-queue-subscription-:uuid` , se isso existir, nao vamos colocar na fila novamente. o worker vai criar e depois vai tirar essa trava.
mas essa trava fica a nivel do worker e nao espalhado por tudo o codigo. quase um lock e vamos usar isso como? vamos ter um repositorio para isso!?

---
oq acontece se a fila for deletada acidentalmente? o flare recria? em que momento?
vamos fazer uma rotina de fixup? cleanup? ai se for o caso podemos escolher oq queremos fazer e em qual recurso.
tipo, fixup no aws.sqs, vou pegar no banco todos os subscriptions e vou ver se as filas estao criadas, se nao estiverem, deleto.
e o cleanup, vou fazer o contrario, tudo que comecar com o nosso formato e nao tiver no subscriptions, deleto.
pedir confirmacoes, tipo o terraform.

oq eu posso fazer eh, no `/subscription/:id/status`, posso retornar algo lá falando que a fila está zuada. seria legal isso?
podemos colocar outras metricas como rate limit, etc...

---
acho que temos que ter um changelog das acoes no subscriptionTrigger, tipo, se por acaso deletarmos algo, ao invés de remover do subscriptionTrigger, vamos apenas marcar
como deletado e a revision. se chegar um revision com a data menor doq o delete, devemos ignorar. hoje não está assim, mesmo essa situação sendo rara, ela ainda pode acontecer.

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