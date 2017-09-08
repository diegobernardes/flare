configure:
	git config pull.rebase true
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

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
