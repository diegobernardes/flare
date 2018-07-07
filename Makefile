DOCKER_RUN_VERSION ?= v0.1.0-alpha
DOCKER_RUN_IMAGE   ?= diegobernardes/flare
DOCKER_CI_VERSION  ?= 0.2
DOCKER_CI_IMAGE    ?= diegobernardes/flare-ci
PROJECT_PATH       ?= github.com/diegobernardes/flare
FLARE_VERSION       = $(shell git describe --tags --always --dirty="-dev")
FLARE_DATE          = $(shell date -u '+%Y-%m-%d %H:%M UTC')
FLARE_COMMIT        = $(shell git rev-parse --short HEAD)
VERSION_FLAGS       = -ldflags='-X "github.com/diegobernardes/flare/service/flare.Version=$(FLARE_VERSION)" \
                                -X "github.com/diegobernardes/flare/service/flare.BuildTime=$(FLARE_DATE)" \
                                -X "github.com/diegobernardes/flare/service/flare.Commit=$(FLARE_COMMIT)"'

run:
	@go run service/flare/cmd/flare.go start

configure:
	@git config pull.rebase true
	@git config branch.master.mergeoptions "--ff-only"

coveralls:
	@docker run \
		-t \
		--rm \
		-v "$(PWD)":/go/src/$(PROJECT_PATH) \
		-w /go/src/$(PROJECT_PATH) \
		-e "TERM=xterm-256color" \
		-e TRAVIS_BRANCH=$(TRAVIS_BRANCH) \
		-e COVERALLS_TOKEN=$(COVERALLS_TOKEN) \
		$(DOCKER_CI_IMAGE):$(DOCKER_CI_VERSION) \
		/bin/bash -c "gotest -race -failfast -covermode=atomic -coverprofile=profile.cov ./...; goveralls -coverprofile=profile.cov"

pre-pr: test lint-fast lint-slow

test:
	@docker run \
		-t \
		--rm \
		-v "$(PWD)":/go/src/$(PROJECT_PATH) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /go/src/$(PROJECT_PATH) \
		-e "TERM=xterm-256color" \
		$(DOCKER_CI_IMAGE):$(DOCKER_CI_VERSION) \
		gotest -v -race -failfast ./...

lint-fast:
	@docker run \
		-t \
		--rm \
		-v "$(PWD)":/go/src/$(PROJECT_PATH) \
		-w /go/src/$(PROJECT_PATH) \
		-e "TERM=xterm-256color" \
		$(DOCKER_CI_IMAGE):$(DOCKER_CI_VERSION) \
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
		$(DOCKER_CI_IMAGE):$(DOCKER_CI_VERSION) \
		gometalinter ./... \
			--disable-all \
			--enable=megacheck \
			--enable=deadcode \
			--enable=structcheck \
			--enable=test \
			--enable=testify \
			--enable=unconvert \
			--enable=varcheck \
			--enable=nakedret \
			--enable=unparam \
			--deadline=20m \
			--aggregate \
			--line-length=100 \
			--min-confidence=.9 \
			--enable-gc \
			--tests \
			--vendor

docker-ci-build:
	@docker build --network=host -t $(DOCKER_CI_IMAGE):$(DOCKER_CI_VERSION) misc/docker/ci

docker-ci-push:
	@docker push $(DOCKER_CI_IMAGE):$(DOCKER_CI_VERSION)

docker-run-build:
	@($(MAKE) flare-build)
	@docker build --network=host -t $(DOCKER_RUN_IMAGE):$(DOCKER_RUN_VERSION) misc/docker/run
	@rm -Rf flare

docker-run-push:
	@docker push $(DOCKER_RUN_IMAGE):$(DOCKER_RUN_VERSION)

flare-build:
	@docker run \
		-t \
		--rm \
		-v "$(PWD)":/go/src/$(PROJECT_PATH) \
		-w /go/src/$(PROJECT_PATH) \
		-e "TERM=xterm-256color" \
		$(DOCKER_CI_IMAGE):$(DOCKER_CI_VERSION) \
		go build $(VERSION_FLAGS) service/flare/cmd/flare.go

git-clean:
	@git remote prune origin
	@git gc --auto

release:
	@(FLARE_VERSION='$(FLARE_VERSION)' FLARE_DATE='$(FLARE_DATE)' FLARE_COMMIT='$(FLARE_COMMIT)' goreleaser --rm-dist)
	@rm -Rf dist