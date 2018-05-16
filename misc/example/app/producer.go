package app

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type Producer struct{}

func (p *Producer) Start() {
	// Wait the Flare server to be ready to accept requests.
	p.wait()

	// Create the resource at Flare.
	p.createResource()

	// Send the updates to Flare.
	p.sendUpdates()
}

func (*Producer) wait() {
	for {
		<-time.After(100 * time.Millisecond)

		resp, err := http.Get("http://server:8080")
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			break
		}
	}
}

func (*Producer) createResource() {
	buf := bytes.NewBufferString(`
		{
      "endpoint": "http://product/{id}",
      "change": {
        "field": "updatedAt",
        "format": "2006-01-02T15:04:05.999-07:00"
      }
    }
	`)

	resp, err := http.Post("http://server:8080/resources", "application./json", buf)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		panic("invalid resource status")
	}
}

func (*Producer) sendUpdates() {
	for {
		<-time.After(5 * time.Second)

		var buf io.Reader
		method := http.MethodDelete

		if rand.Intn(10) > 3 {
			method = http.MethodPut
			buf = bytes.NewBufferString(fmt.Sprintf(`
				{
					"id": "123",
					"title": "Smartphone",
					"updatedAt": "%s"
				}
			`, time.Now().Format("2006-01-02T15:04:05.999-07:00")))
		}

		fmt.Printf("[%s] http://product/123\n", method)
		req, err := http.NewRequest(method, "http://server:8080/documents/http://product/123", buf)
		if err != nil {
			panic(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
	}
}
