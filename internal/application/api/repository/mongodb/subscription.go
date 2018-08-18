package mongodb

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/countopt"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/diegobernardes/flare/internal"
)

type Subscription struct {
	Database mongo.Database
	Timeout  Timeout

	collection mongo.Collection
}

func (s *Subscription) Init() error {
	if err := s.Timeout.Init(); err != nil {
		return errors.Wrap(err, "error during Timeout initialization")
	}

	s.collection = *s.Database.Collection("subscriptions")
	return nil
}

func (s Subscription) Find(
	ctx context.Context, pagination internal.Pagination,
) ([]internal.Subscription, internal.Pagination, error) {
	g, gCtx := errgroup.WithContext(ctx)

	var count uint
	g.Go(func() error {
		var err error
		count, err = s.findCount(gCtx, pagination)
		return err
	})

	var subscriptions []internal.Subscription
	g.Go(func() error {
		var err error
		subscriptions, err = s.findEntries(ctx, pagination)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, pagination, err
	}

	np := internal.Pagination{
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
		Total:  count,
	}
	return subscriptions, np, nil
}

func (s Subscription) FindByID(ctx context.Context, subscriptionID string) (*internal.Subscription, error) {
	result := s.collection.FindOne(ctx, bson.NewDocument(bson.EC.String("_id", subscriptionID)))
	subscription, err := s.unmarshal(result)
	return subscription, errors.Wrap(err, "error during parse result")

	return nil, nil
}

func (s Subscription) Create(ctx context.Context, subscription internal.Subscription) (string, error) {
	_, err := s.collection.InsertOne(ctx, s.marshal(subscription))
	return subscription.ID, errors.Wrap(err, "error during insert")
}

func (s Subscription) Delete(ctx context.Context, subscriptionID string) error {
	result, err := s.collection.DeleteOne(ctx, bson.NewDocument(bson.EC.String("_id", subscriptionID)))
	if err != nil {
		return errors.Wrap(err, "error during delete")
	}

	if result.DeletedCount == 0 {
		return customError{cause: errors.New("subscription not found"), notFound: true}
	}
	return nil
}

func (s Subscription) findCount(ctx context.Context, pagination internal.Pagination) (uint, error) {
	timeout := countopt.MaxTimeMs((int32)(timeout(ctx, s.Timeout.Count) / time.Millisecond))
	count, err := s.collection.Count(ctx, nil, timeout)
	return uint(count), errors.Wrap(err, "error during count")
}

func (s Subscription) findEntries(
	ctx context.Context,
	pagination internal.Pagination,
) ([]internal.Subscription, error) {
	opts := []findopt.Find{
		findopt.Limit(int64(pagination.Limit)),
		findopt.Skip(int64(pagination.Offset)),
		findopt.MaxTime(timeout(ctx, s.Timeout.Find)),
	}
	cursor, err := s.collection.Find(ctx, nil, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "error during find")
	}
	defer cursor.Close(ctx) // nolint

	var subscriptions []internal.Subscription
	for cursor.Next(ctx) {
		subscription, err := s.unmarshal(cursor)
		if err != nil {
			return nil, errors.Wrap(err, "error during unmarshal")
		}
		subscriptions = append(subscriptions, *subscription)
	}

	return subscriptions, errors.Wrap(cursor.Err(), "error while processing the cursor")
}

type subscriptionView struct {
	ID         string                   `bson:"_id"`
	Endpoint   subscriptionViewEndpoint `bson:"endpoint"`
	Delivery   subscriptionViewDelivery `bson:"delivery"`
	ResourceID string                   `bson:"resourceID"`
	Partition  string                   `bson:"partition"`
	Data       map[string]interface{}   `bson:"data"`
	Content    subscriptionViewContent  `bson:"content"`
	Mode       string                   `bson:"mode"`
	Status     string                   `bson:"status"`
	Revision   string                   `bson:"revision"`
}

type subscriptionViewEndpoint struct {
	URL     *urlView                            `bson:"url"`
	Method  string                              `bson:"method"`
	Headers http.Header                         `bson:"headers"`
	Action  map[string]subscriptionViewEndpoint `bson:"action"`
}

type subscriptionViewDelivery struct {
	Success []int `bson:"success"`
	Discard []int `bson:"discard"`
}

type subscriptionViewContent struct {
	Document bool `bson:"document"`
	Envelope bool `bson:"envelope"`
}

func (s Subscription) marshal(subscription internal.Subscription) subscriptionView {
	return subscriptionView{
		ID:         subscription.ID,
		ResourceID: subscription.Resource.ID,
		Partition:  subscription.Partition,
		Mode:       subscription.Mode,
		Status:     subscription.Status,
		Revision:   subscription.Revision,
		Data:       subscription.Data,
		Endpoint:   s.marshalEndpoint(subscription.Endpoint),
		Delivery: subscriptionViewDelivery{
			Success: subscription.Delivery.Success,
			Discard: subscription.Delivery.Discard,
		},
		Content: subscriptionViewContent{
			Document: subscription.Content.Document,
			Envelope: subscription.Content.Envelope,
		},
	}
}

func (s Subscription) unmarshal(d decoder) (*internal.Subscription, error) {
	var sv subscriptionView
	if err := d.Decode(&sv); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, errors.Wrap(err, "error during response decode")
	}

	return &internal.Subscription{
		ID: sv.ID,
		Resource: internal.Resource{
			ID: sv.ResourceID,
		},
		Partition: sv.Partition,
		Mode:      sv.Mode,
		Status:    sv.Status,
		Revision:  sv.Revision,
		Data:      sv.Data,
		Endpoint:  s.unmarshalEndpoint(sv.Endpoint),
		Delivery: internal.SubscriptionDelivery{
			Success: sv.Delivery.Success,
			Discard: sv.Delivery.Discard,
		},
		Content: internal.SubscriptionContent{
			Document: sv.Content.Document,
			Envelope: sv.Content.Envelope,
		},
	}, nil
}

func (s Subscription) marshalEndpoint(
	endpoint internal.SubscriptionEndpoint,
) subscriptionViewEndpoint {
	viewEndpoint := subscriptionViewEndpoint{
		Method:  endpoint.Method,
		Headers: endpoint.Headers,
		Action:  make(map[string]subscriptionViewEndpoint),
	}

	if endpoint.URL != nil {
		viewEndpoint.URL = &urlView{
			Scheme: endpoint.URL.Scheme,
			Host:   endpoint.URL.Host,
			Path:   endpoint.URL.Path,
		}
	}

	for key, value := range endpoint.Action {
		viewEndpoint.Action[key] = s.marshalEndpoint(value)
	}

	return viewEndpoint
}

func (s Subscription) unmarshalEndpoint(
	viewEndpoint subscriptionViewEndpoint,
) internal.SubscriptionEndpoint {
	endpoint := internal.SubscriptionEndpoint{
		Method:  viewEndpoint.Method,
		Headers: viewEndpoint.Headers,
		Action:  make(map[string]internal.SubscriptionEndpoint),
	}

	if endpoint.URL != nil {
		endpoint.URL = &url.URL{
			Scheme: viewEndpoint.URL.Scheme,
			Host:   viewEndpoint.URL.Host,
			Path:   viewEndpoint.URL.Path,
		}
	}

	for key, value := range viewEndpoint.Action {
		endpoint.Action[key] = s.unmarshalEndpoint(value)
	}

	return endpoint
}
