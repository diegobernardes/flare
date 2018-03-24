package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"

	baseConsumer "github.com/diegobernardes/flare/domain/consumer"
	"github.com/diegobernardes/flare/infra/cluster"
	"github.com/diegobernardes/flare/infra/pagination"
)

type Consumer struct {
	Client *Client
	Node   *Node
	Logger log.Logger
}

func (c *Consumer) Init() error {
	if c.Client == nil {
		return errors.New("missing Client")
	}

	if c.Node == nil {
		return errors.New("missing Node")
	}

	if c.Logger == nil {
		return errors.New("missing Logger")
	}

	return nil
}

// TODO: nao esta funcionando mt bem, acho que vamos precisar mais informacoes no offset para ficar
// bom.
func (c *Consumer) Find(
	ctx context.Context, page *pagination.Pagination,
) ([]baseConsumer.Consumer, *pagination.Pagination, error) {
	var (
		results []baseConsumer.Consumer
		kv      = clientv3.NewKV(c.Client.base)
	)

	options := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
		clientv3.WithLimit(int64(page.Limit)),
	}

	key := "/consumer/"
	if page.Offset != "" {
		options = append(options, clientv3.WithFromKey())
		key = fmt.Sprintf("/consumer/%s", page.Offset)
	}
	options = append(options)

	rawResp, err := kv.Get(ctx, key, options...)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error during fetch consumer")
	}

	var lastKey string
	kvs := rawResp.Kvs
	if page.Offset != "" && len(kvs) > 0 {
		kvs = rawResp.Kvs[1:]
	}
	for _, resp := range kvs {
		consumer, err := c.unmarshal(resp.Value)
		if err != nil {
			return nil, nil, err
		}

		results = append(results, *consumer)
		lastKey = consumer.ID
	}

	rpage := &pagination.Pagination{
		Total: int(rawResp.Count),
		Limit: page.Limit,
	}
	if rpage.Total > rpage.Limit {
		rpage.Offset = lastKey
	}
	return results, rpage, nil
}

func (c *Consumer) FindByID(ctx context.Context, id string) (*baseConsumer.Consumer, error) {
	kv := clientv3.NewKV(c.Client.base)
	resp, err := kv.Get(ctx, fmt.Sprintf("/consumer/%s", id))
	if err != nil {
		return nil, err
	}

	if resp.Count == 0 {
		return nil, nil
	}

	consumer, err := c.unmarshal(resp.Kvs[0].Value)
	if err != nil {
		return nil, err
	}

	return consumer, nil
}

func (c *Consumer) Create(ctx context.Context, consumer *baseConsumer.Consumer) error {
	return c.Update(ctx, consumer)
}

func (c *Consumer) Update(ctx context.Context, consumer *baseConsumer.Consumer) error {
	kv := clientv3.NewKV(c.Client.base)

	content, err := c.marshal(consumer)
	if err != nil {
		return errors.Wrap(err, "error during marshal consumer to json")
	}
	if _, err := kv.Put(ctx, fmt.Sprintf("/consumer/%s", consumer.ID), content); err != nil {
		return errors.Wrap(err, "error during update consumer")
	}
	return nil
}

// Delete a given consumer.
func (c *Consumer) Delete(ctx context.Context, id string) error {
	kv := clientv3.NewKV(c.Client.base)
	txn := kv.Txn(ctx)
	txn = txn.Then(
		clientv3.OpDelete(fmt.Sprintf("/consumer/%s", id)),
		clientv3.OpDelete(fmt.Sprintf("/consumer-assign/%s", id)),
	)

	resp, err := txn.Commit()
	if err != nil {
		return errors.Wrap(err, "error during consumer delete")
	}
	if !resp.Succeeded {
		return errors.New("failed to delete consumer")
	}
	return nil
}

// Load all consumers.
func (c *Consumer) Load(ctx context.Context) ([]cluster.Consumer, error) {
	var (
		results    []cluster.Consumer
		kv         = clientv3.NewKV(c.Client.base)
		pagination = 100
		lastKey    string
	)

	for {
		options := []clientv3.OpOption{
			clientv3.WithPrefix(),
			clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
			clientv3.WithLimit(int64(pagination)),
		}

		key := "/consumer/"
		if lastKey != "" {
			key = lastKey
			options = append(options, clientv3.WithFromKey())
		}

		rawResp, err := kv.Get(ctx, key, options...)
		if err != nil {
			return nil, errors.Wrap(err, "error during fetch consumer")
		}

		for _, resp := range rawResp.Kvs {
			fragments := strings.Split(string(resp.Key), "/")
			consumer := cluster.Consumer{ID: fragments[2]}
			nodeID, err := c.fetchNodeID(ctx, consumer.ID)
			if err != nil {
				return nil, errors.Wrapf(err, "error during fetch nodeID from consumer '%s'", consumer.ID)
			}
			consumer.NodeID = nodeID

			results = append(results, consumer)
			lastKey = string(resp.Key)
		}

		if len(rawResp.Kvs) < pagination {
			break
		}
	}

	return results, nil
}

// Watch any change on consumers.
func (c *Consumer) Watch(
	ctx context.Context, fn func(cluster.Consumer, string) error,
) context.Context {
	return c.watch(ctx, "/consumer/", fn)
}

// WatchAssign watch for any changes on a consumer assign.
func (c *Consumer) WatchAssign(
	ctx context.Context, fn func(cluster.Consumer, string) error,
) context.Context {
	return c.watch(ctx, "/consumer-assign/", fn)
}

// Assign the consumer for a given node.
func (c *Consumer) Assign(ctx context.Context, consumerID, nodeID string) error {
	lease, err := c.Node.lease(ctx, nodeID)
	if err != nil {
		return errors.Wrapf(err, "error during fetch lease from node '%s'", nodeID)
	}

	key := fmt.Sprintf("/consumer-assign/%s", consumerID)
	kv := clientv3.NewKV(c.Client.base)
	if _, err := kv.Put(ctx, key, nodeID, clientv3.WithLease(lease)); err != nil {
		return errors.Wrapf(err, "error during assign consumer '%s' to node '%s'", consumerID, nodeID)
	}
	return nil
}

// Unassign the consumer from a node.
func (c *Consumer) Unassign(ctx context.Context, consumerID string) error {
	key := fmt.Sprintf("/consumer-assign/%s", consumerID)
	kv := clientv3.NewKV(c.Client.base)

	if _, err := kv.Delete(ctx, key); err != nil {
		return errors.Wrapf(err, "error during unassign consumer '%s'", consumerID)
	}
	return nil
}

func (c *Consumer) marshal(consumer *baseConsumer.Consumer) (string, error) {
	revision := map[string]interface{}{
		"field": consumer.Payload.Revision.Field,
	}
	if consumer.Payload.Revision.Format != "" {
		revision["format"] = consumer.Payload.Revision.Format
	}
	if consumer.Payload.Revision.ID != "" {
		revision["id"] = consumer.Payload.Revision.ID
	}

	content := map[string]interface{}{
		"id":        consumer.ID,
		"createdAt": consumer.CreatedAt,
		"payload": map[string]interface{}{
			"format":   consumer.Payload.Format,
			"revision": revision,
		},
	}

	var source map[string]interface{}
	if consumer.Source.AWSKinesis != nil {
		source = map[string]interface{}{
			"aws.kinesis": map[string]interface{}{
				"stream": consumer.Source.AWSKinesis.Stream,
			},
		}
	} else if consumer.Source.AWSSQS != nil {
		source = map[string]interface{}{
			"aws.sqs": map[string]interface{}{
				"arn": consumer.Source.AWSSQS.ARN,
			},
		}
	}

	content["source"] = source
	result, err := json.Marshal(content)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func (c *Consumer) unmarshal(content []byte) (*baseConsumer.Consumer, error) {
	type raw struct {
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"createdAt"`
		Source    struct {
			AWSSQS struct {
				ARN string `json:"arn"`
			} `json:"aws.sqs"`
			AWSKinesis struct {
				Stream string `json:"stream"`
			} `json:"aws.kinesis"`
		} `json:"source"`
		Payload struct {
			Format   string `json:"format"`
			Revision struct {
				ID     string `json:"id"`
				Field  string `json:"field"`
				Format string `json:"format"`
			} `json:"revision"`
		} `json:"payload"`
	}

	var r raw
	if err := json.Unmarshal(content, &r); err != nil {
		return nil, errors.Wrap(err, "")
	}

	result := &baseConsumer.Consumer{
		ID:        r.ID,
		CreatedAt: r.CreatedAt,
		Payload: baseConsumer.ConsumerPayload{
			Format: r.Payload.Format,
			Revision: baseConsumer.ConsumerPayloadRevision{
				ID:     r.Payload.Revision.ID,
				Field:  r.Payload.Revision.Field,
				Format: r.Payload.Revision.Format,
			},
		},
	}

	if r.Source.AWSKinesis.Stream != "" {
		result.Source.AWSKinesis = &baseConsumer.ConsumerSourceAWSKinesis{
			Stream: r.Source.AWSKinesis.Stream,
		}
	} else if r.Source.AWSSQS.ARN != "" {
		result.Source.AWSSQS = &baseConsumer.ConsumerSourceAWSSQS{
			ARN: r.Source.AWSSQS.ARN,
		}
	}

	return result, nil
}

func (c *Consumer) fetchNodeID(ctx context.Context, id string) (string, error) {
	kv := clientv3.NewKV(c.Client.base)
	resp, err := kv.Get(ctx, fmt.Sprintf("/consumer-assign/%s", id))
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", nil
	}
	return string(resp.Kvs[0].Value), nil
}

func (c *Consumer) watch(
	ctx context.Context, key string, fn func(cluster.Consumer, string) error,
) context.Context {
	watch := clientv3.NewWatcher(c.Client.base)

	ch := watch.Watch(ctx, key, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	ctx, ctxCancel := context.WithCancel(ctx)

	go func() {
		defer ctxCancel()

		for change := range ch {
			if change.Canceled {
				return
			}

			for _, event := range change.Events {
				action := cluster.ActionCreate
				if event.IsModify() {
					action = cluster.ActionUpdate
				} else if event.Type == mvccpb.DELETE {
					action = cluster.ActionDelete
				}

				key := string(event.Kv.Key)
				fragments := strings.Split(key, "/")
				if len(fragments) < 3 {
					level.Error(c.Logger).Log("message", fmt.Sprintf("invalid key '%s'", key))
					continue
				}
				consumer := cluster.Consumer{ID: fragments[2]}
				nodeID, err := c.fetchNodeID(ctx, consumer.ID)
				if err != nil {
					level.Error(c.Logger).Log(
						"message", fmt.Sprintf("error during fetch nodeID from consumer '%s'", consumer.ID),
						"error", err,
					)
				}
				consumer.NodeID = nodeID

				if err := fn(consumer, action); err != nil {
					level.Error(c.Logger).Log("message", "error during consumer process", "error", err)
					return
				}
			}
		}
	}()

	return ctx
}
