vou criar uma abstracao do repository.
essa abstracao qnd der um create, ele vai criar uma fila no sqs e depois vai gravar o cara no banco.
vamos ter que ter uma tarefa para deletar filas criadas erradamente. tipo, procurar as filas criadas e ver se estao
associadas no banco, se noa estiverem, deletar. pode ser uma tarefa recorrente do flare fazer essa limpeza, podemos configurar 
no arquivo de configuracao.

essa regra tem que existir para os providers individualmente, tipo, soh para o aws sqs por exemplo.

bom, salvei a fila no banco? como tudo vai funcionar agora? precisamos ligar o worker, como vamos ligar?
dps de criar, ligamos ele?

// qnd ligar
procura todos os subscriptions, dai pega as ids e liga os workers pelas ids, se por acaso algum nao existir, tem que dar erro 
de inicializacao e sair.

periodicamente, o worker tem que monitorar por novaas filas para processar. esse processo hj no mongodb vai ser um pooling, mas 
se estivessemos usando alguma ferramenta de comunicaao, poderiamos enviar na hora avisando que tem mais filas para processar. 
tenho que fazer o watch da v4, exatamente igual.


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