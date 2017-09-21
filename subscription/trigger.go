package subscription

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/pkg/errors"

	"github.com/diegobernardes/flare"
)

// Kinds of trigger on a document.
const (
	TriggerActionDelete = "delete"
	TriggerActionUpdate = "update"
	TriggerActionCreate = "create"
)

// Trigger is used to process the signals on documents change.
type Trigger struct {
	repository flare.SubscriptionRepositorier
	httpClient *http.Client
}

// Update the document change signal.
func (t *Trigger) Update(ctx context.Context, document *flare.Document) error {
	if err := t.repository.Trigger(ctx, TriggerActionUpdate, document, t.exec(document)); err != nil {
		return errors.Wrap(err, "error during trigger")
	}
	return nil
}

// Delete the document change signal.
func (t *Trigger) Delete(ctx context.Context, document *flare.Document) error {
	if err := t.repository.Trigger(ctx, TriggerActionDelete, document, t.exec(document)); err != nil {
		return errors.Wrap(err, "error during trigger")
	}
	return nil
}

func (t *Trigger) exec(
	document *flare.Document,
) func(context.Context, flare.Subscription, string) error {
	return func(ctx context.Context, sub flare.Subscription, kind string) error {
		content, err := json.Marshal(map[string]interface{}{
			"id":               document.Id,
			"changeFieldValue": document.ChangeFieldValue,
			"updatedAt":        document.UpdatedAt.String(),
			"action":           kind,
		})
		if err != nil {
			return errors.Wrap(err, "error during response generate")
		}

		buf := bytes.NewBuffer(content)
		req, err := http.NewRequest(sub.Endpoint.Method, sub.Endpoint.URL.String(), buf)
		if err != nil {
			return errors.Wrap(err, "error during http request create")
		}
		req = req.WithContext(ctx)

		for key, values := range sub.Endpoint.Headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		resp, err := t.httpClient.Do(req)
		if err != nil {
			return errors.Wrap(err, "error during http request")
		}

		for _, status := range sub.Delivery.Success {
			if status == resp.StatusCode {
				return nil
			}
		}

		for _, status := range sub.Delivery.Discard {
			if status == resp.StatusCode {
				return nil
			}
		}

		return errors.Errorf(
			"success and discard status don't match with the response value '%d'", resp.StatusCode,
		)
	}
}

// NewTrigger initialize the Trigger.
func NewTrigger(options ...func(*Trigger)) (*Trigger, error) {
	trigger := &Trigger{}

	for _, option := range options {
		option(trigger)
	}

	if trigger.repository == nil {
		return nil, errors.New("repository not found")
	}

	if trigger.httpClient == nil {
		return nil, errors.New("httpClient not found")
	}

	return trigger, nil
}

// TriggerRepository set the repository on Trigger.
func TriggerRepository(repository flare.SubscriptionRepositorier) func(*Trigger) {
	return func(t *Trigger) {
		t.repository = repository
	}
}

// TriggerHTTPClient set the httpClient on Trigger.
func TriggerHTTPClient(httpClient *http.Client) func(*Trigger) {
	return func(t *Trigger) {
		t.httpClient = httpClient
	}
}
