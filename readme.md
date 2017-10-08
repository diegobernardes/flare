## Flare 

[![Build Status](https://travis-ci.org/diegobernardes/flare.svg?branch=master)](https://travis-ci.org/diegobernardes/flare) [![Coverage Status](https://coveralls.io/repos/github/diegobernardes/flare/badge.svg?branch=master)](https://coveralls.io/github/diegobernardes/flare?branch=master) [![GoDoc](https://godoc.org/github.com/diegobernardes/flare?status.svg)](https://godoc.org/github.com/diegobernardes/flare)  

Flare is a service that notify changes of REST APIs. Everytime a resource change (create, update or delete), anyone with a subscription gets a notification.

## Features
* Work with any HTTP endpoint.
* Deliver only new changes, the clients never gonna receive the same notification twice.
* Play nice with your current infrastructure.

## How to run

```bash
go get github.com/diegobernardes/flare/services/flare/cmd
cd github.com/diegobernardes/flare/services/flare/cmd
go run flare.go start
```

## How it works

Flare has 3 basic entities: `Resource`, `Subscription` and `Document`.

### Resource
Resource represents the entity you want to monitor.

```bash
curl -H "Content-Type: application/json" -XPOST http://localhost:8080/resources -d @- << EOF
{
	"addresses": [
		"http://app.io",
		"https://app.com"
	],
	"path": "/users/{*}",
	"change": {
		"kind": "date",
		"field": "updatedAt",
		"dateFormat": "2006-01-02T15:04:05Z07:00"
	}
}
EOF
```

### Subscription
Subscriptions track the document changes on resources and notify clients.

```bash
curl -H "Content-Type: application/json" -XPOST http://localhost:8080/resources/{id}/subscriptions -d @- << EOF
{
	"endpoint": {
		"url": "http://localhost:8000/update",
		"method": "post"
	},
	"delivery": {
		"success": [200],
		"discard": [500]
	}
}
EOF
```

### Document
Update a given document at Flare.

```bash
curl -H "Content-Type: application/json" -XPOST http://localhost:8080/documents/http://app.io/users/123 -d @- << EOF
{
	"updatedAt": "2017-09-23T07:08:08.008Z",
	"name": "Diego Bernardes",
}
EOF
```