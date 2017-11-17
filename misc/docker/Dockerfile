FROM golang:1.9.0-stretch

RUN go get -u github.com/alecthomas/gometalinter && gometalinter --install \
  && go get github.com/mattn/goveralls \
  && go get -u github.com/rakyll/gotest
