package consumer

import (
	"time"
)

type Consumer struct {
	ID        string
	Source    ConsumerSource
	Payload   ConsumerPayload
	CreatedAt time.Time
}

type ConsumerSource struct {
	AWSSQS     *ConsumerSourceAWSSQS
	AWSKinesis *ConsumerSourceAWSKinesis
}

type ConsumerSourceAWSSQS struct {
	ARN         string
	Concurrency int
}

type ConsumerSourceAWSKinesis struct {
	Stream string
}

type ConsumerPayload struct {
	ID       string
	Revision ConsumerPayloadRevision
}

type ConsumerPayloadRevision struct {
	Field  string
	Format string
}
