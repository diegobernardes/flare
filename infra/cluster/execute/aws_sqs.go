package execute

import (
	"github.com/kr/pretty"

	"github.com/diegobernardes/flare/domain/consumer"
)

type AWSSQS struct {
	Consumer consumer.Consumer
}

func (s *AWSSQS) Start() error {
	pretty.Println("diego")
	// chamar alguma funcao aws para consumir e passar um metodo para processar.
	return nil
}

func (s *AWSSQS) Stop() error { return nil }
