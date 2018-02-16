// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package flare

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"

	subscriptionWorker "github.com/diegobernardes/flare/domain/subscription/worker"
	"github.com/diegobernardes/flare/infra/config"
	infraWorker "github.com/diegobernardes/flare/infra/worker"
)

type worker struct {
	base                  []infraWorker.Client
	subscriptionPartition *subscriptionWorker.Partition
	subscriptionSpread    *subscriptionWorker.Spread
	subscriptionDelivery  *subscriptionWorker.Delivery
	logger                log.Logger
	cfg                   *config.Client
	repository            *repository
	queue                 *queue
}

func (w *worker) init() error {
	key := "worker.enable"
	if !w.cfg.GetBool(key) {
		return nil
	}

	if err := w.initSubscriptionDelivery(); err != nil {
		return errors.Wrap(err, "error during subscription.delivery worker initialization")
	}

	if err := w.initSubscriptionSpread(); err != nil {
		return errors.Wrap(err, "error during subscription.spread worker initialization")
	}

	if err := w.initSubscriptionPartition(); err != nil {
		return errors.Wrap(err, "error during subscription.partition worker initialization")
	}

	for i := 0; i < len(w.base); i++ {
		w.base[i].Start()
	}

	return nil
}

func (w *worker) processor(
	name string, processor infraWorker.Processor,
) (*infraWorker.Client, error) {
	timeout, err := w.cfg.GetDuration(fmt.Sprintf("worker.%s.timeout", name))
	if err != nil {
		return nil, err
	}

	queuer, err := w.queue.fetch(name)
	if err != nil {
		return nil, err
	}

	client, err := infraWorker.NewClient(
		infraWorker.WorkerGoroutines(w.cfg.GetInt(fmt.Sprintf("worker.%s.concurrency", name))),
		infraWorker.WorkerLogger(w.logger),
		infraWorker.WorkerProcessor(processor),
		infraWorker.WorkerPuller(queuer),
		infraWorker.WorkerPusher(queuer),
		infraWorker.WorkerTimeout(timeout),
	)
	if err != nil {
		return nil, errors.Wrap(err, "error during worker initialization")
	}

	return client, nil
}

func (w *worker) stop() error {
	for i := 0; i < len(w.base); i++ {
		w.base[i].Stop()
	}
	return nil
}

func (w *worker) initSubscriptionPartition() error {
	unitOfWork := &subscriptionWorker.Partition{}
	client, err := w.processor("subscription.partition", unitOfWork)
	if err != nil {
		return err
	}

	err = unitOfWork.Init(
		subscriptionWorker.PartitionResourceRepository(w.repository.base.Resource()),
		subscriptionWorker.PartitionPusher(client),
		subscriptionWorker.PartitionOutput(w.subscriptionSpread),
		subscriptionWorker.PartitionConcurrency(
			w.cfg.GetInt("worker.subscription.partition.concurrency-output"),
		),
	)
	if err != nil {
		return errors.Wrap(err, "error during client initialization")
	}

	client.Start()
	w.subscriptionPartition = unitOfWork

	return nil
}

func (w *worker) initSubscriptionDelivery() error {
	unitOfWork := &subscriptionWorker.Delivery{}
	client, err := w.processor("subscription.delivery", unitOfWork)
	if err != nil {
		return err
	}

	maxIdleConnections := w.cfg.GetInt("http.client.max-idle-connections")
	maxIdleConnectionsPerHost := w.cfg.GetInt("http.client.max-idle-connections-per-host")
	idleConnectionTimeout, err := w.cfg.GetDuration("http.client.idle-connection-timeout")
	if err != nil {
		return err
	}

	hc := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          maxIdleConnections,
			MaxIdleConnsPerHost:   maxIdleConnectionsPerHost,
			IdleConnTimeout:       idleConnectionTimeout,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	err = unitOfWork.Init(
		subscriptionWorker.DeliveryPusher(client),
		subscriptionWorker.DeliveryResourceRepository(w.repository.base.Resource()),
		subscriptionWorker.DeliverySubscriptionRepository(w.repository.base.Subscription()),
		subscriptionWorker.DeliveryHTTPClient(hc),
	)
	if err != nil {
		return errors.Wrap(err, "error during client initialization")
	}

	client.Start()
	w.subscriptionDelivery = unitOfWork

	return nil
}

func (w *worker) initSubscriptionSpread() error {
	unitOfWork := &subscriptionWorker.Spread{}
	client, err := w.processor("subscription.spread", unitOfWork)
	if err != nil {
		return err
	}

	err = unitOfWork.Init(
		subscriptionWorker.SpreadSubscriptionRepository(w.repository.base.Subscription()),
		subscriptionWorker.SpreadPusher(client),
		subscriptionWorker.SpreadOutput(w.subscriptionDelivery),
		subscriptionWorker.SpreadConcurrency(
			w.cfg.GetInt("worker.subscription.spread.concurrency-output"),
		),
	)
	if err != nil {
		return errors.Wrap(err, "error during worker processor initialization")
	}

	client.Start()
	w.subscriptionSpread = unitOfWork

	return nil
}
