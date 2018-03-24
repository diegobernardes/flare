package consumer

import "time"

var (
	ConsumerPayloadJSON = "json"
	ConsumerPayloadRAW  = "raw"
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
	ARN string
}

type ConsumerSourceAWSKinesis struct {
	Stream string
}

type ConsumerPayload struct {
	Format   string
	Revision ConsumerPayloadRevision
}

type ConsumerPayloadRevision struct {
	ID     string
	Field  string
	Format string
}
