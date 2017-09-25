configure:
	git config pull.rebase true

lint-fast:
	docker run --rm -t -v "$$PWD":/go/src/github.com/diegobernardes/flare -w /go/src/github.com/diegobernardes/flare diegobernardes/flare:0.3 gometalinter \
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
	docker run --rm -t -v "$$PWD":/go/src/github.com/diegobernardes/flare -w /go/src/github.com/diegobernardes/flare diegobernardes/flare:0.3 gometalinter \
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
	docker run --rm -t -v "$$PWD":/go/src/github.com/diegobernardes/flare -w /go/src/github.com/diegobernardes/flare diegobernardes/flare:0.3 gotest -v -race ./...

coveralls:
	docker run --rm -t -v "$$PWD":/go/src/github.com/diegobernardes/flare -w /go/src/github.com/diegobernardes/flare -e TRAVIS_BRANCH=$$TRAVIS_BRANCH -e COVERALLS_TOKEN=$$COVERALLS_TOKEN diegobernardes/flare:0.3 goveralls -race

flare-build:
	docker run --rm -t -v "$$PWD":/go/src/github.com/diegobernardes/flare -w /go/src/github.com/diegobernardes/flare diegobernardes/flare:0.3 go build services/flare/cmd/flare.go

docker-build:
	docker build -t diegobernardes/flare:latest -t diegobernardes/flare:0.3 devstuff/docker

docker-push:
	docker push diegobernardes/flare:latest
	docker push diegobernardes/flare:0.3

git-clean:
	git gc
	git fetch --all --prune