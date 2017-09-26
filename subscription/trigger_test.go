// Copyright 2017 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package subscription

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/pkg/errors"
	gock "gopkg.in/h2non/gock.v1"

	"github.com/diegobernardes/flare"
	"github.com/diegobernardes/flare/repository/test"
)

func TestTriggerAction(t *testing.T) {
	tests := []struct {
		name       string
		repository flare.SubscriptionRepositorier
		httpClient *http.Client
		document   *flare.Document
		hasErr     bool
		action     string
	}{
		{
			"Error",
			test.NewSubscription(
				test.SubscriptionError(errors.New("error at repository")),
			),
			http.DefaultClient,
			nil,
			true,
			"Update",
		},
		{
			"Error",
			test.NewSubscription(
				test.SubscriptionError(errors.New("error at repository")),
			),
			http.DefaultClient,
			nil,
			true,
			"Delete",
		},
		{
			"Success",
			test.NewSubscription(),
			http.DefaultClient,
			&flare.Document{Resource: flare.Resource{Id: "123"}},
			false,
			"Update",
		},
		{
			"Success",
			test.NewSubscription(),
			http.DefaultClient,
			&flare.Document{Resource: flare.Resource{Id: "123"}},
			false,
			"Delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger, err := NewTrigger(
				TriggerHTTPClient(tt.httpClient),
				TriggerRepository((tt.repository)),
			)
			if err != nil {
				t.Error(errors.Wrap(err, "error during trigger initialization"))
				t.FailNow()
			}

			switch tt.action {
			case "Update":
				err = trigger.Update(context.Background(), tt.document)
			case "Delete":
				err = trigger.Delete(context.Background(), tt.document)
			default:
				t.Errorf("unexpected action '%s'", tt.action)
				t.FailNow()
			}

			if (err != nil) != tt.hasErr {
				t.Errorf(fmt.Sprintf(
					"Trigger.%s invalid result, want '%v', got '%v'", tt.action, tt.hasErr, err != nil,
				))
			}
		})
	}
}

func TestTriggerExec(t *testing.T) {
	tests := []struct {
		name          string
		hasErr        bool
		document      *flare.Document
		subscription  flare.Subscription
		kind          string
		cancelContext bool
		mock          func()
	}{
		{
			"Fail",
			true,
			&flare.Document{
				Id:               "123",
				ChangeFieldValue: "version",
				UpdatedAt:        time.Now(),
			},
			flare.Subscription{
				Endpoint: flare.SubscriptionEndpoint{
					Method: "bad method",
					URL:    url.URL{Scheme: "HTTP", Host: "app.com"},
				},
			},
			flare.SubscriptionTriggerUpdate,
			false,
			func() {},
		},
		{
			"Fail",
			true,
			&flare.Document{
				Id:               "123",
				ChangeFieldValue: "version",
				UpdatedAt:        time.Now(),
			},
			flare.Subscription{
				Endpoint: flare.SubscriptionEndpoint{
					Method: http.MethodGet,
					URL:    url.URL{Scheme: "HTTP", Host: "app.com"},
				},
			},
			flare.SubscriptionTriggerUpdate,
			true,
			func() {},
		},
		{
			"Invalid status code",
			true,
			&flare.Document{
				Id:               "123",
				ChangeFieldValue: "version",
				UpdatedAt:        time.Now(),
			},
			flare.Subscription{
				Endpoint: flare.SubscriptionEndpoint{
					Method:  http.MethodGet,
					URL:     url.URL{Scheme: "HTTP", Host: "app.com"},
					Headers: http.Header{"Key": []string{"value"}},
				},
				Delivery: flare.SubscriptionDelivery{
					Success: []int{200},
					Discard: []int{500},
				},
			},
			flare.SubscriptionTriggerUpdate,
			false,
			func() { gock.New("http://app.com").Reply(400) },
		},
		{
			"Success status code",
			false,
			&flare.Document{
				Id:               "123",
				ChangeFieldValue: "version",
				UpdatedAt:        time.Now(),
			},
			flare.Subscription{
				Endpoint: flare.SubscriptionEndpoint{
					Method:  http.MethodGet,
					URL:     url.URL{Scheme: "HTTP", Host: "app.com"},
					Headers: http.Header{"Key": []string{"value"}},
				},
				Delivery: flare.SubscriptionDelivery{
					Success: []int{200},
				},
			},
			flare.SubscriptionTriggerUpdate,
			false,
			func() { gock.New("http://app.com").Reply(200) },
		},
		{
			"Discard status code",
			false,
			&flare.Document{
				Id:               "123",
				ChangeFieldValue: "version",
				UpdatedAt:        time.Now(),
			},
			flare.Subscription{
				Endpoint: flare.SubscriptionEndpoint{
					Method:  http.MethodGet,
					URL:     url.URL{Scheme: "HTTP", Host: "app.com"},
					Headers: http.Header{"Key": []string{"value"}},
				},
				Delivery: flare.SubscriptionDelivery{
					Discard: []int{500},
				},
			},
			flare.SubscriptionTriggerUpdate,
			false,
			func() { gock.New("http://app.com").Reply(500) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger, err := NewTrigger(
				TriggerRepository(test.NewSubscription()),
				TriggerHTTPClient(http.DefaultClient),
			)
			if err != nil {
				t.Error("NewTrigger invalid result, want 'false', got 'true'")
			}

			ctx := context.Background()
			if tt.cancelContext {
				newCtx, newCtxCancel := context.WithCancel(ctx)
				newCtxCancel()
				ctx = newCtx
			}

			tt.mock()
			defer gock.Off()

			err = trigger.exec(tt.document)(ctx, tt.subscription, tt.kind)
			if (err != nil) != tt.hasErr {
				t.Errorf("Trigger.exec invalid result, want '%v', got '%v'", tt.hasErr, (err != nil))
			}
		})
	}
}

func TestNewTrigger(t *testing.T) {
	tests := []struct {
		name    string
		options []func(*Trigger)
		hasErr  bool
	}{
		{
			"Missing repository",
			nil,
			true,
		},
		{
			"Missing httpClient",
			[]func(*Trigger){TriggerRepository(test.NewSubscription())},
			true,
		},
		{
			"Success",
			[]func(*Trigger){
				TriggerRepository(test.NewSubscription()),
				TriggerHTTPClient(http.DefaultClient),
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTrigger(tt.options...)
			if (err != nil) != tt.hasErr {
				t.Errorf("NewTrigger invalid result, want '%v', got '%v'", tt.hasErr, (err != nil))
			}
		})
	}
}
