configure:
	git config pull.rebase true

lint-fast:
	gometalinter \
		--disable-all \
		--enable=gas \
		--enable=goconst \
		--enable=gocyclo \
		--enable=gofmt \
		--enable=goimports \
		--enable=golint \
		--enable=ineffassign \
		--enable=lll \
		--enable=misspell \
		--enable=vet \
		--enable=vetshadow \
		--deadline=30s \
		--aggregate \
		--line-length=100 \
		--min-confidence=.9 \
		--linter='gas:gas -exclude=G104 -fmt=csv {path}/*.go:^(?P<path>.*?\.go),(?P<line>\d+),(?P<message>[^,]+,[^,]+,[^,]+)' \
		--tests \
		--vendor ./...

lint-slow:
	gometalinter \
		--disable-all \
		--enable=megacheck \
		--enable=aligncheck \
		--enable=deadcode \
		--enable=interfacer \
		--enable=structcheck \
		--enable=test \
		--enable=testify \
		--enable=unconvert \
		--enable=varcheck \
		--deadline=20m \
		--aggregate \
		--line-length=100 \
		--min-confidence=.9 \
		--enable-gc \
		--tests \
		--vendor ./...

test:
	go test -race ./...

flare-build:
	go build services/flare/cmd/flare.go

docker-lint-fast:
	docker run --rm -v "$$PWD":/go/src/github.com/diegobernardes/flare -w /go/src/github.com/diegobernardes/flare diegobernardes/flare:0.1 make lint-fast

docker-lint-slow:
	docker run --rm -v "$$PWD":/go/src/github.com/diegobernardes/flare -w /go/src/github.com/diegobernardes/flare diegobernardes/flare:0.1 make lint-slow

docker-test:
	docker run --rm -v "$$PWD":/go/src/github.com/diegobernardes/flare -w /go/src/github.com/diegobernardes/flare diegobernardes/flare:0.1 make test

docker-flare-build:
	docker run --rm -v "$$PWD":/go/src/github.com/diegobernardes/flare -w /go/src/github.com/diegobernardes/flare diegobernardes/flare:0.1 make flare-build

docker-build:
	docker build -t diegobernardes/flare:latest -t diegobernardes/flare:0.1 devstuff/docker

docker-push:
	docker push diegobernardes/flare:latest
	docker push diegobernardes/flare:0.1