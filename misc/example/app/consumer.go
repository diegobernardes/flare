package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Consumer holds the logic to consume the notifications from Flare.
type Consumer struct{}

// Start the consumer.
func (c *Consumer) Start() {
	// Wait the Flare server to be ready to accept requests.
	c.wait()

	// Create the subscription on the first resource at Flare.
	c.createSubscription()

	// Listen for changes from Flare.
	c.server()
}

func (c *Consumer) wait() {
	type result struct {
		Pagination struct {
			Total int `json:"total"`
		} `json:"pagination"`
	}

	for {
		<-time.After(100 * time.Millisecond)

		resp, err := http.Get("http://server:8080/resources")
		if err != nil {
			continue
		}

		var r result
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(&r); err != nil {
			if err := resp.Body.Close(); err != nil {
				panic(err)
			}
			continue
		}
		if err := resp.Body.Close(); err != nil {
			panic(err)
		}

		if r.Pagination.Total > 0 {
			break
		}
	}
}

func (*Consumer) resourceID() (string, error) {
	resp, err := http.Get("http://server:8080/resources")
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", errors.New("invalid status code")
	}

	type resourceList struct {
		Resources []struct {
			ID string `json:"id,omitempty"`
		} `json:"resources"`
	}

	var resources resourceList
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&resources); err != nil {
		return "", err
	}

	if len(resources.Resources) == 0 {
		return "", errors.New("missing resource")
	}

	return resources.Resources[0].ID, nil
}

func (c *Consumer) createSubscription() {
	resourceID, err := c.resourceID()
	if err != nil {
		panic(err)
	}

	buf := bytes.NewBufferString(`
		{
			"endpoint": {
				"url": "http://consumer:8080/feed/product",
				"method": "POST",
				"actions": {
					"update": {
						"method": "PUT",
						"url": "http://consumer:8080/feed/product/{id}"
					},
					"delete": {
						"method": "DELETE",
					  "url": "http://consumer:8080/feed/product/{id}"
					}
				}
			},
			"delivery": {
				"success": [200],
				"discard": [500]
			},
			"content": {
				"envelope": true,
				"document": true
			},
			"data": {
				"service": "product",
				"id": "{id}"
			}
		}
	`)
	resp, err := http.Post(
		fmt.Sprintf("http://server:8080/resources/%s/subscriptions", resourceID),
		"application/json",
		buf,
	)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			panic(err)
		}
	}()

	if resp.StatusCode != http.StatusCreated {
		panic(errors.New("invalid status code"))
	}
}

func (c *Consumer) server() {
	http.HandleFunc("/", c.handler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func (c *Consumer) handler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[%s] %s\n", r.Method, r.URL.String())
	w.WriteHeader(http.StatusOK)
}
