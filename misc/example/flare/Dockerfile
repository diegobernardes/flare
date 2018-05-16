FROM golang:1.10.0-stretch

VOLUME ["/opt/go/src/github.com/diegobernardes/flare"]

WORKDIR /opt/go/src/github.com/diegobernardes/flare/service/flare/cmd

CMD go build -o /opt/flare/flare flare.go \
  && cd /opt/flare \
  && ./flare start