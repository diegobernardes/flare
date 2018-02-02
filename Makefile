DOCKER_VERSION ?= 0.4
DOCKER_IMAGE   ?= diegobernardes/flare
PROJECT_PATH   ?= github.com/diegobernardes/flare
VERSION        = $(shell git describe --tags --always --dirty="-dev")
DATE           = $(shell date -u '+%Y-%m-%d %H:%M UTC')
COMMIT         = $(shell git rev-parse --short HEAD)
VERSION_FLAGS  = -ldflags='-X "github.com/diegobernardes/flare/service/flare.Version=$(VERSION)" \
                           -X "github.com/diegobernardes/flare/service/flare.BuildTime=$(DATE)" \
                           -X "github.com/diegobernardes/flare/service/flare.Commit=$(COMMIT)"'

run:
	@echo $(VERSION)
	@echo $(DATE)

configure:
	@git config pull.rebase true
	@git config branch.master.mergeoptions "--ff-only"

coveralls:
	@docker run \
		-t \
		--rm \
		-v "$(PWD)":/go/src/$(PROJECT_PATH) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /go/src/$(PROJECT_PATH) \
		-e "TERM=xterm-256color" \
		-e TRAVIS_BRANCH=$(TRAVIS_BRANCH) \
		-e COVERALLS_TOKEN=$(COVERALLS_TOKEN) \
		$(DOCKER_IMAGE):$(DOCKER_VERSION) \
		goveralls

pre-pr: test lint-fast lint-slow

test:
	@docker run \
		-t \
		--rm \
		-v "$(PWD)":/go/src/$(PROJECT_PATH) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /go/src/$(PROJECT_PATH) \
		-e "TERM=xterm-256color" \
		$(DOCKER_IMAGE):$(DOCKER_VERSION) \
		gotest -v -race ./...

lint-fast:
	@docker run \
		-t \
		--rm \
		-v "$(PWD)":/go/src/$(PROJECT_PATH) \
		-w /go/src/$(PROJECT_PATH) \
		-e "TERM=xterm-256color" \
		$(DOCKER_IMAGE):$(DOCKER_VERSION) \
		gometalinter ./... \
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
			--enable=errcheck \
			--deadline=30s \
			--aggregate \
			--line-length=100 \
			--min-confidence=.9 \
			--linter='errcheck:errcheck -ignorepkg github.com/go-kit/kit/log -abspath {not_tests=-ignoretests}:PATH:LINE:COL:MESSAGE' \
			--linter='gas:gas -exclude=G104 -fmt=csv {path}/*.go:^(?P<path>.*?\.go),(?P<line>\d+),(?P<message>[^,]+,[^,]+,[^,]+)' \
			--tests \
			--vendor

lint-slow:
	@docker run \
		-t \
		--rm \
		-v "$(PWD)":/go/src/$(PROJECT_PATH) \
		-w /go/src/$(PROJECT_PATH) \
		-e "TERM=xterm-256color" \
		$(DOCKER_IMAGE):$(DOCKER_VERSION) \
		gometalinter ./... \
			--disable-all \
			--enable=megacheck \
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
			--vendor

docker-build:
	@docker build --network=host -t $(DOCKER_IMAGE):$(DOCKER_VERSION) misc/docker

docker-push:
	@docker push $(DOCKER_IMAGE):$(DOCKER_VERSION)

flare-build:
	@docker run \
		-t \
		--rm \
		-v "$(PWD)":/go/src/$(PROJECT_PATH) \
		-w /go/src/$(PROJECT_PATH) \
		-e "TERM=xterm-256color" \
		$(DOCKER_IMAGE):$(DOCKER_VERSION) \
		go build $(VERSION_FLAGS) service/flare/cmd/flare.go

git-clean:
	@git remote prune origin
	@git gc --auto