package main

import (
	"os"

	"github.com/diegobernardes/flare/misc/example/app"
)

func main() {
	switch os.Getenv("TYPE") {
	case "consumer":
		var consumer app.Consumer
		consumer.Start()
	case "producer":
		var producer app.Producer
		producer.Start()
	default:
		panic("unknow type")
	}
}
