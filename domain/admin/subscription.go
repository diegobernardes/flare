package admin

import "net/http"

type Subscription struct{}

func (s *Subscription) Index(w http.ResponseWriter, r *http.Request) {}

func (s *Subscription) New(w http.ResponseWriter, r *http.Request) {}

func (s *Subscription) Show(w http.ResponseWriter, r *http.Request) {}

func (s *Subscription) Create(w http.ResponseWriter, r *http.Request) {}

func (s *Subscription) Delete(w http.ResponseWriter, r *http.Request) {}
