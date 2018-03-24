// consumer
{
  "id": "77bd86d6-8b7c-4efb-91d3-ac9bdfd75e87",
  "source": {
    "arn": "some arn",
    "type": "aws.sqs",
    "concurrency": "auto" // default 
  },
  "payload": {
    "format": "json",  
    "revision": {
      "id": "id",
      "field": "updatedAt",
      "format": "2006-01-02T15:04:05Z07:00"
    }
  },
  "createdAt": "0001-01-01T00:00:00Z"
}

// consumer
{
  "id": "77bd86d6-8b7c-4efb-91d3-ac9bdfd75e87",
  "source": {
    "arn": "some arn",
    "type": "aws.sqs",
    "concurrency": "auto" // default 
  },
  "payload": {
    "format": "raw"
  },
  "createdAt": "0001-01-01T00:00:00Z"
}

// producer
// pode usar o filter, se for json. se tiver revision, pode acessar o state, se nao, dar erro no cadastro.
{
  "source": {
    "type": "aws.kinesis",
    "stream": "some kinesis stream"
  },
  "filter": "current.quantity != state.quantity && 'state'.name = 'sample'",
  "filter": "sample > 0", 
  "createdAt": "2018-02-20T23:15:38-03:00"
}

// message
{
  "id": "xyz",
  "updatedAt": "2006-01-02T15:04:05Z07:00",
  "action": "delete"
}
