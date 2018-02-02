package flare

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/infra/config"
	memoryRepository "github.com/diegobernardes/flare/provider/memory/repository"
	"github.com/diegobernardes/flare/provider/mongodb"
	mongoDBRepository "github.com/diegobernardes/flare/provider/mongodb/repository"
)

type repositorier interface {
	Resource() flare.ResourceRepositorier
	Subscription() flare.SubscriptionRepositorier
	Document() flare.DocumentRepositorier
}

type repository struct {
	cfg  *config.Client
	base repositorier
}

func (r *repository) init() error {
	partition := r.cfg.GetInt("domain.resource.partition")
	provider := r.cfg.GetString("provider.repository")

	switch provider {
	case providerMemory:
		r.base = memoryRepository.NewClient(
			memoryRepository.ClientResourceOptions(memoryRepository.ResourcePartitionLimit(partition)),
		)
	case providerMongoDB:
		repository, err := r.initMongoDB(partition)
		if err != nil {
			return errors.Wrap(err, "error during MongoDB initialization")
		}
		r.base = repository
	default:
		return fmt.Errorf("invalid provider.repository config '%s'", provider)
	}

	return nil
}

func (r *repository) initMongoDB(partition int) (repositorier, error) {
	options, err := r.initMongoDBOptions()
	if err != nil {
		return nil, err
	}

	client, err := mongodb.NewClient(options...)
	if err != nil {
		return nil, errors.Wrap(err, "error during client initialization")
	}

	repository, err := mongoDBRepository.NewClient(
		mongoDBRepository.ClientConnection(client),
		mongoDBRepository.ClientResourceOptions(
			mongoDBRepository.ResourcePartitionLimit(partition),
		),
	)
	if err != nil {
		return nil, errors.Wrap(err, "error during repository initialization")
	}

	return repository, nil
}

func (r *repository) initMongoDBOptions() ([]func(*mongodb.Client), error) {
	var options []func(*mongodb.Client)

	options = append(options, mongodb.ClientAddrs(r.cfg.GetStringSlice("provider.mongodb.addrs")))
	options = append(options, mongodb.ClientDatabase(r.cfg.GetString("provider.mongodb.database")))
	options = append(options, mongodb.ClientUsername(r.cfg.GetString("provider.mongodb.username")))
	options = append(options, mongodb.ClientPassword(r.cfg.GetString("provider.mongodb.password")))
	options = append(options, mongodb.ClientPoolLimit(r.cfg.GetInt("provider.mongodb.pool-limit")))
	options = append(options, mongodb.ClientReplicaSet(
		r.cfg.GetString("provider.mongodb.replica-set")),
	)

	timeout, err := r.cfg.GetDuration("provider.mongodb.timeout")
	if err != nil {
		return nil, errors.Wrap(
			err,
			fmt.Sprintf(
				"invalid provider.mongodb.timeout '%s' config, error during parse",
				r.cfg.GetString("provider.mongodb.timeout"),
			),
		)
	}
	options = append(options, mongodb.ClientTimeout(timeout))

	return options, nil
}

func (r *repository) stop() error {
	type closer interface {
		Stop() error
	}

	group, ok := r.base.(closer)
	if !ok {
		return nil
	}

	return group.Stop()
}

func (r *repository) setup(ctx context.Context) error {
	type setup interface {
		Setup(context.Context) error
	}

	s, ok := r.base.(setup)
	if !ok {
		return nil
	}

	return s.Setup(ctx)
}
