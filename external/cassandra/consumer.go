package cassandra

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gocql/gocql"
	"github.com/minio/blake2b-simd"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"

	"github.com/diegobernardes/flare/infra/cluster"
	"github.com/diegobernardes/flare/infra/pagination"
	baseConsumer "github.com/diegobernardes/flare/service/consumer"
)

type Consumer struct {
	Base     *Client
	Logger   log.Logger
	Interval time.Duration
	state    []string
}

func (c *Consumer) Init() error {
	if c.Base == nil {
		return errors.New("missing cassandra client")
	}

	if c.Interval <= 0 {
		return errors.New("invalid Interval")
	}

	return nil
}

func (c *Consumer) fetchNodeIDS(ctx context.Context) ([]string, error) {
	iter := c.Base.Session.Query("SELECT node_id FROM consumers").WithContext(ctx).Iter()

	var (
		id  string
		ids []string
	)
	for iter.Scan(&id) {
		if len(ids) != 0 && len(ids) == sort.SearchStrings(ids, id) {
			continue
		}
		ids = append(ids, id)
		sort.Strings(ids)
	}

	if err := iter.Close(); err != nil {
		panic(err)
	}

	return ids, nil
}

func (c *Consumer) Fetch(ctx context.Context, fn func(cluster.ConsumerStatus) error) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				level.Error(c.Logger).Log("message", "panic at Fetch", "reason", err)
				go c.Fetch(ctx, fn)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.Interval):
			}

			iter := c.Base.Session.Query("SELECT id, node_id FROM consumers").WithContext(ctx).Iter()
			var (
				statusList []cluster.ConsumerStatus
				status     cluster.ConsumerStatus
			)
			for iter.Scan(&status.ID, &status.NodeID) {
				statusList = append(statusList, status)
			}

			if err := iter.Close(); err != nil {
				panic(err)
			}

			for _, status := range statusList {
				if member(status.ID, c.state) {
					continue
				}

				status.Status = cluster.ConsumerStatusCreate
				if err := fn(status); err != nil {
					level.Error(c.Logger).Log("message", "error during process create", "error", err)
					continue
				}
				c.state = append(c.state, status.ID)
			}

			for i := 0; i < len(c.state); i++ {
				id := c.state[i]

				if memberClusterStatus(id, statusList) {
					continue
				}

				status = cluster.ConsumerStatus{ID: id, Status: cluster.ConsumerStatusDelete}
				if err := fn(status); err != nil {
					level.Error(c.Logger).Log("message", "error during process create", "error", err)
					continue
				}
				c.state = append(c.state[:i], c.state[i+1:]...)
			}

		}
	}()
}

func (c *Consumer) Assign(ctx context.Context, id, nodeID string) error {
	var hash string
	err := c.Base.Session.Query(`SELECT hash FROM consumers WHERE id = ?`, id).Scan(&hash)
	if err != nil {
		panic(err)
	}

	query := c.Base.Session.Query(
		"UPDATE consumers SET node_id = ? WHERE hash = ?", nodeID, hash,
	).WithContext(ctx)

	if err := query.Exec(); err != nil {
		panic(err)
	}
	return nil
}

func (c *Consumer) Find(
	ctx context.Context, pagination *pagination.Pagination,
) ([]baseConsumer.Consumer, *pagination.Pagination, error) {
	countIter := c.Base.Session.Query("SELECT count(id) FROM consumers").WithContext(ctx).Iter()
	countIter.Scan(&pagination.Total)

	query := c.Base.Session.Query(
		`SELECT id, created_at, payload, source, source_type FROM consumers`,
	).WithContext(ctx)
	query = query.PageSize(pagination.Limit).Consistency(gocql.All)

	if pagination.Offset != "" {
		offset, err := hex.DecodeString(pagination.Offset)
		if err != nil {
			return nil, nil, errors.Wrapf(
				err, "error during pagination.Offset decode '%s'", pagination.Offset,
			)
		}
		query = query.PageState(offset)
	}

	iter := query.Iter()

	var consumers []baseConsumer.Consumer
	for {
		consumer, err := unmarshalConsumer(iter)
		if err != nil {
			panic(err)
		}
		if consumer == nil {
			break
		}
		consumers = append(consumers, *consumer)

		if iter.WillSwitchPage() {
			break
		}
	}

	state := iter.PageState()
	content := make([]byte, hex.EncodedLen(len(state)))
	hex.Encode(content, state)
	pagination.Offset = string(content)

	if err := iter.Close(); err != nil {
		return nil, nil, errors.Wrap(err, "error during cassandra iter close")
	}

	return consumers, pagination, nil
}

func unmarshalConsumer(iter *gocql.Iter) (*baseConsumer.Consumer, error) {
	type payload struct {
		ID             string `json:"id"`
		RevisionField  string `json:"revisionField"`
		RevisionFormat string `json:"revisionFormat"`
	}

	var (
		c             baseConsumer.Consumer
		rawPayload    string
		rawSource     string
		rawSourceType string
	)

	if ok := iter.Scan(&c.ID, &c.CreatedAt, &rawPayload, &rawSource, &rawSourceType); !ok {
		return nil, nil
	}

	var p payload
	if err := json.Unmarshal([]byte(rawPayload), &p); err != nil {
		return nil, nil
	}

	c.Payload.ID = p.ID
	c.Payload.Revision.Field = p.RevisionField
	c.Payload.Revision.Format = p.RevisionFormat

	var err error
	switch rawSourceType {
	case "aws.sqs":
		c.Source.AWSSQS, err = unmarshalConsumerSourceAWSSQS([]byte(rawSource))
		if err != nil {
			panic(err)
		}
	case "aws.kinesis":
		c.Source.AWSKinesis, err = unmarshalConsumerSourceAWSKinesis([]byte(rawSource))
		if err != nil {
			panic(err)
		}
	}

	return &c, nil
}

func unmarshalConsumerSourceAWSSQS(data []byte) (*baseConsumer.ConsumerSourceAWSSQS, error) {
	type source struct {
		ARN         string `json:"arn"`
		Concurrency int    `json:"concurrency"`
	}
	var rawSource source
	if err := json.Unmarshal(data, &rawSource); err != nil {
		panic(err)
	}

	return &baseConsumer.ConsumerSourceAWSSQS{
		ARN:         rawSource.ARN,
		Concurrency: rawSource.Concurrency,
	}, nil
}

func unmarshalConsumerSourceAWSKinesis(data []byte) (*baseConsumer.ConsumerSourceAWSKinesis, error) {
	type source struct {
		Stream string `json:"stream"`
	}
	var rawSource source
	if err := json.Unmarshal(data, &rawSource); err != nil {
		panic(err)
	}

	return &baseConsumer.ConsumerSourceAWSKinesis{Stream: rawSource.Stream}, nil
}

func genHash(source baseConsumer.ConsumerSource) string {
	var data []byte
	if source.AWSKinesis != nil {
		data = append(data, []byte("aws.kinesis")...)
		data = append(data, []byte(source.AWSKinesis.Stream)...)
	} else if source.AWSSQS != nil {
		data = append(data, []byte("aws.sqs")...)
		data = append(data, []byte(source.AWSSQS.ARN)...)
	}
	return fmt.Sprintf("%x", blake2b.Sum256(data))
}

func (c *Consumer) FindByID(context.Context, string) (*baseConsumer.Consumer, error) {
	return nil, nil
}

func (c *Consumer) Create(ctx context.Context, consumer *baseConsumer.Consumer) error {
	sourceType, err := c.fetchSourceType(consumer.Source)
	if err != nil {
		return errors.Wrap(err, "error during source type extraction")
	}

	var source map[string]interface{}
	switch sourceType {
	case "aws.sqs":
		source = c.fetchSourceAWSSQS(consumer.Source.AWSSQS)
	case "aws.kinesis":
		source = c.fetchSourceAWSKinesis(consumer.Source.AWSKinesis)
	}

	payload := c.fetchPayload(consumer.Payload)

	nsource, err := json.Marshal(source)
	if err != nil {
		panic(err)
	}

	npayload, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	query := c.Base.Session.Query(`
		INSERT INTO consumers (id, hash, source_type, source, payload, created_at)
		       VALUES (?, ?, ?, ?, ?, ?) IF NOT EXISTS`,
		uuid.NewV4().String(),
		genHash(consumer.Source),
		sourceType,
		string(nsource),
		string(npayload),
		consumer.CreatedAt,
	)

	if applied, err := query.MapScanCAS(map[string]interface{}{}); err != nil {
		return errors.Wrap(err, "error during insert")
	} else if !applied {
		return errors.New("consumer already exists")
	}
	return nil
}

func (c *Consumer) fetchSourceType(source baseConsumer.ConsumerSource) (string, error) {
	if source.AWSKinesis != nil {
		return "aws.kinesis", nil
	}

	if source.AWSSQS != nil {
		return "aws.sqs", nil
	}

	return "", errors.New("consumer.Source not found")
}

func (c *Consumer) fetchSourceAWSSQS(source *baseConsumer.ConsumerSourceAWSSQS) map[string]interface{} {
	return map[string]interface{}{
		"arn":         source.ARN,
		"concurrency": source.Concurrency,
	}
}

func (c *Consumer) fetchSourceAWSKinesis(
	source *baseConsumer.ConsumerSourceAWSKinesis,
) map[string]interface{} {
	return map[string]interface{}{"stream": source.Stream}
}

func (c *Consumer) fetchPayload(payload baseConsumer.ConsumerPayload) map[string]interface{} {
	result := map[string]interface{}{
		"revisionField": payload.Revision.Field,
	}

	if payload.Revision.Format != "" {
		result["revisionFormat"] = payload.Revision.Format
	}

	if payload.ID != "" {
		result["id"] = payload.ID
	}

	return result
}

func (c *Consumer) Delete(ctx context.Context, id string) error {
	var hash string
	err := c.Base.Session.Query(`SELECT hash FROM consumers WHERE id = ?`, id).Scan(&hash)
	if err != nil {
		panic(err)
	}

	query := c.Base.Session.Query(`DELETE FROM consumers WHERE hash = ?`, hash).WithContext(ctx)
	if err := query.Exec(); err != nil {
		return errors.Wrap(err, "error during delete consumer")
	}
	return nil
}

func (c *Consumer) Update(ctx context.Context, consumer *baseConsumer.Consumer) error {
	return nil
}
