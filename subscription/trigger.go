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

// Process the document change signal.
func (t *Trigger) Process(ctx context.Context, kind string, document *flare.Document) error {
	switch kind {
	case TriggerActionDelete, TriggerActionUpdate, TriggerActionCreate:
	default:
		return errors.Errorf("invalid kind '%s'", kind)
	}

	if err := t.repository.Trigger(ctx, kind, document, t.exec(kind, document)); err != nil {
		return errors.Wrap(err, "error during trigger")
	}
	return nil
}

func (t *Trigger) exec(
	kind string, document *flare.Document,
) func(context.Context, flare.Subscription) error {
	return func(ctx context.Context, sub flare.Subscription) error {
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
