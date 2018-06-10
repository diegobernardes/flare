// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"context"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

type subscriptionHook interface {
	Create(context.Context, *flare.Subscription) error
}

// Subscription implements the hook for subscription repository.
type Subscription struct {
	Repository flare.SubscriptionRepositorier
	Hook       subscriptionHook
}

// Init check if everything needed is set.
func (s *Subscription) Init() error {
	if s.Repository == nil {
		return errors.New("missing repository")
	}

	if s.Hook == nil {
		return errors.New("missing hook")
	}
	return nil
}

func (s *Subscription) Find(
	ctx context.Context, pagination *flare.Pagination, id string,
) ([]flare.Subscription, *flare.Pagination, error) {
	return s.Repository.Find(ctx, pagination, id)
}

func (s *Subscription) FindByID(
	ctx context.Context, resourceID, id string,
) (*flare.Subscription, error) {
	return s.Repository.FindByID(ctx, resourceID, id)
}

func (s *Subscription) FindByPartition(
	ctx context.Context, resourceID, partition string,
) (<-chan flare.Subscription, <-chan error, error) {
	return s.Repository.FindByPartition(ctx, resourceID, partition)
}

func (s *Subscription) Create(ctx context.Context, sub *flare.Subscription) error {
	if err := s.Repository.Create(ctx, sub); err != nil {
		return err
	}

	if err := s.Hook.Create(ctx, sub); err != nil {
		return errors.Wrap(err, "error during subscription hook create")
	}
	return nil
}

func (s *Subscription) Delete(ctx context.Context, resourceID, id string) error {
	return s.Repository.Delete(ctx, resourceID, id)
}

func (s *Subscription) Trigger(
	ctx context.Context,
	action string,
	document *flare.Document,
	subscription *flare.Subscription,
	fn func(context.Context, *flare.Document, *flare.Subscription, string) error,
) error {
	return s.Repository.Trigger(ctx, action, document, subscription, fn)
}
