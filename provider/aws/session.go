// Copyright 2018 Diego Bernardes. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
)

// Session store the AWS session access.
type Session struct {
	key    string
	secret string
	region string
	Base   *session.Session
}

// NewSession returns a configured session to access AWS services.
func NewSession(options ...func(*Session)) (*Session, error) {
	s := &Session{}

	for _, option := range options {
		option(s)
	}

	if s.key == "" {
		return nil, errors.New("missing key")
	}

	if s.secret == "" {
		return nil, errors.New("missing secret")
	}

	if s.region == "" {
		return nil, errors.New("missing region")
	}

	awsSession, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(s.key, s.secret, ""),
		Region:      aws.String(s.region),
	})
	if err != nil {
		return nil, errors.Wrap(err, "error during AWS session create")
	}
	s.Base = awsSession

	return s, nil
}

// SessionKey set the session key.
func SessionKey(key string) func(*Session) {
	return func(s *Session) {
		s.key = key
	}
}

// SessionSecret set the session secret.
func SessionSecret(secret string) func(*Session) {
	return func(s *Session) {
		s.secret = secret
	}
}

// SessionRegion set the session region.
func SessionRegion(region string) func(*Session) {
	return func(s *Session) {
		s.region = region
	}
}
