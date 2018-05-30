// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package worker

// import (
// 	"context"
// 	"encoding/json"
// 	"net/url"

// 	"github.com/pkg/errors"

// 	"github.com/diegobernardes/flare"
// 	"github.com/diegobernardes/flare/infra/worker"
// )

// // CleanRevision is used to delete old revisions of a document. Everytime a document is processed,
// // a new job is enqueued.
// type CleanRevision struct {
// 	Repository flare.DocumentRepositorier
// 	Pusher     worker.Pusher
// }

// // Process handle the stream of messages to be processed.
// func (cr *CleanRevision) Process(ctx context.Context, content []byte) error {
// 	doc, err := cr.unmarshal(content)
// 	if err != nil {
// 		return errors.Wrap(err, "error during message unmarshal")
// 	}

// 	if err := cr.Repository.DeleteOldRevisions(ctx, doc); err != nil {
// 		return errors.Wrapf(
// 			err,
// 			"error during delete older revisions then '%d' for document '%s'",
// 			doc.Revision,
// 			doc.ID.String(),
// 		)
// 	}

// 	return nil
// }

// // Enqueue delivery the message to another stream to handle it.
// func (cr *CleanRevision) Enqueue(ctx context.Context, doc *flare.Document) error {
// 	content, err := cr.marshal(doc)
// 	if err != nil {
// 		return errors.Wrap(err, "error during message marshal")
// 	}

// 	if err := cr.Pusher.Push(ctx, content); err != nil {
// 		return errors.Wrap(err, "error during message delivery")
// 	}
// 	return nil
// }

// func (cr *CleanRevision) marshal(doc *flare.Document) ([]byte, error) {
// 	rawContent := map[string]interface{}{
// 		"id": map[string]interface{}{
// 			"scheme": doc.ID.Scheme,
// 			"host":   doc.ID.Host,
// 			"path":   doc.ID.Path,
// 		},
// 		"revision": doc.Revision,
// 	}

// 	content, err := json.Marshal(rawContent)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "error during message marshal to json")
// 	}

// 	return content, nil
// }

// func (cr *CleanRevision) unmarshal(rawContent []byte) (*flare.Document, error) {
// 	type message struct {
// 		ID struct {
// 			Scheme string `json:"scheme"`
// 			Host   string `json:"host"`
// 			Path   string `json:"path"`
// 		}
// 		Revision int64 `json:"revision"`
// 	}

// 	var m message
// 	if err := json.Unmarshal(rawContent, &m); err != nil {
// 		return nil, errors.Wrap(err, "error during message unmarshal from json")
// 	}

// 	return &flare.Document{
// 		ID: url.URL{
// 			Scheme: m.ID.Scheme,
// 			Host:   m.ID.Host,
// 			Path:   m.ID.Path,
// 		},
// 		Revision: m.Revision,
// 	}, nil
// }
