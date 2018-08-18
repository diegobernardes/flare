package mongodb

import (
	"context"
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

// Resource has the logic to persist the resources.
type Resource struct {
	Database mongo.Database
	Timeout  Timeout

	collection mongo.Collection
}

// Init check if the struct has everything needed to execute.
func (r *Resource) Init() error {
	if err := r.Timeout.Init(); err != nil {
		return errors.Wrap(err, "error during Timeout initialization")
	}

	r.collection = *r.Database.Collection("resources")
	return nil
}

// EnsureIndex create the indexes on mongodb.
func (r Resource) EnsureIndex(ctx context.Context) error {
	index := r.collection.Indexes()
	opts := mongo.NewIndexOptionsBuilder().Background(false).Unique(true).Build()

	model := mongo.IndexModel{
		Keys: bson.NewDocument(
			bson.EC.Int32("endpoint.schema", 1),
			bson.EC.Int32("endpoint.host", 1),
			bson.EC.Int32("endpoint.path", 1),
		),
		Options: opts,
	}

	_, err := index.CreateOne(ctx, model)
	return errors.Wrap(err, "error during index creation")
}

// Create a resource at the database.
func (r Resource) Create(ctx context.Context, resource internal.Resource) (string, error) {
	_, err := r.collection.InsertOne(ctx, r.marshal(resource))
	return resource.ID, errors.Wrap(err, "error during insert")
}

// FindByID fetch a resource by id.
func (r Resource) FindByID(ctx context.Context, resourceID string) (*internal.Resource, error) {
	result := r.collection.FindOne(ctx, bson.NewDocument(bson.EC.String("_id", resourceID)))
	resource, err := r.unmarshal(result)
	return resource, errors.Wrap(err, "error during parse result")
}

// Find all the resources respecting the pagination.
func (r Resource) Find(
	ctx context.Context, pagination internal.Pagination,
) ([]internal.Resource, internal.Pagination, error) {
	g, gCtx := errgroup.WithContext(ctx)

	var count uint
	g.Go(func() error {
		var err error
		count, err = r.findCount(gCtx, pagination)
		return err
	})

	var resources []internal.Resource
	g.Go(func() error {
		var err error
		resources, err = r.findEntries(ctx, pagination)
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
	return resources, np, nil
}

// Delete a resource by id.
func (r Resource) Delete(ctx context.Context, resourceID string) error {
	result, err := r.collection.DeleteOne(ctx, bson.NewDocument(bson.EC.String("_id", resourceID)))
	if err != nil {
		return errors.Wrap(err, "error during delete")
	}

	if result.DeletedCount == 0 {
		return customError{cause: errors.New("resource not found"), notFound: true}
	}
	return nil
}

func (r Resource) findCount(ctx context.Context, pagination internal.Pagination) (uint, error) {
	timeout := countopt.MaxTimeMs((int32)(timeout(ctx, r.Timeout.Count) / time.Millisecond))
	count, err := r.collection.Count(ctx, nil, timeout)
	return uint(count), errors.Wrap(err, "error during count")
}

func (r Resource) findEntries(
	ctx context.Context,
	pagination internal.Pagination,
) ([]internal.Resource, error) {
	opts := []findopt.Find{
		findopt.Limit(int64(pagination.Limit)),
		findopt.Skip(int64(pagination.Offset)),
		findopt.MaxTime(timeout(ctx, r.Timeout.Find)),
	}
	cursor, err := r.collection.Find(ctx, nil, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "error during find")
	}
	defer cursor.Close(ctx) // nolint

	var resources []internal.Resource
	for cursor.Next(ctx) {
		resource, err := r.unmarshal(cursor)
		if err != nil {
			return nil, errors.Wrap(err, "error during unmarshal")
		}
		resources = append(resources, *resource)
	}

	return resources, errors.Wrap(cursor.Err(), "error while processing the cursor")
}

func (Resource) marshal(resource internal.Resource) resourceView {
	return resourceView{
		ID: resource.ID,
		Change: resourceViewChange{
			Field:  resource.Change.Field,
			Format: resource.Change.Format,
		},
		Endpoint: urlView{
			Scheme: resource.Endpoint.Scheme,
			Host:   resource.Endpoint.Host,
			Path:   resource.Endpoint.Path,
		},
	}
}

func (Resource) unmarshal(d decoder) (*internal.Resource, error) {
	var rv resourceView
	if err := d.Decode(&rv); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, errors.Wrap(err, "error during response decode")
	}

	return &internal.Resource{
		ID: rv.ID,
		Endpoint: url.URL{
			Scheme: rv.Endpoint.Scheme,
			Host:   rv.Endpoint.Host,
			Path:   rv.Endpoint.Path,
		},
		Change: internal.ResourceChange{
			Field:  rv.Change.Field,
			Format: rv.Change.Format,
		},
	}, nil
}

type resourceView struct {
	ID       string             `bson:"_id"`
	Endpoint urlView            `bson:"endpoint"`
	Change   resourceViewChange `bson:"change"`
}

type resourceViewChange struct {
	Field  string `bson:"field"`
	Format string `bson:"format"`
}
