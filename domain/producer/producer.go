package producer

import "time"

type Producer struct {
	ID        string
	Source    ProducerSource
	Metadata  map[string]interface{}
	CreatedAt time.Time
}

type ProducerSource struct {
	AWSSQS     *ProducerSourceAWSSQS
	AWSKinesis *ProducerSourceAWSKinesis
}

type ProducerSourceAWSSQS struct {
	ARN string
}

type ProducerSourceAWSKinesis struct {
	Stream string
}

/*
{
  "source": {
    "type": "aws.kinesis",
    "stream": "some kinesis stream"
  },
  "metadata": {
    "service": "product",
  },
  "createdAt": "2018-02-20T23:15:38-03:00"
}

pensar em como vamos armazenar isso no etcd...
*/
