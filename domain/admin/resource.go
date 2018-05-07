package admin

import (
	"io/ioutil"
	"net/http"

	"github.com/alecthomas/template"
)

type Resource struct{}

func (re *Resource) Index(w http.ResponseWriter, r *http.Request) {
	rawTemplate, err := ioutil.ReadFile("/home/diego/projects/go/1.10.0/src/github.com/diegobernardes/flare/domain/admin/assets/views/index.html")
	if err != nil {
		panic(err)
	}

	t, err := template.New("webpage").Parse(string(rawTemplate))
	if err != nil {
		panic(err)
	}

	if err := t.Execute(w, nil); err != nil {
		panic(err)
	}
}

func (re *Resource) New(w http.ResponseWriter, r *http.Request) {}

func (re *Resource) Show(w http.ResponseWriter, r *http.Request) {}

func (re *Resource) Create(w http.ResponseWriter, r *http.Request) {}

func (re *Resource) Delete(w http.ResponseWriter, r *http.Request) {}
