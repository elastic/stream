LICENSE := ASL2-Short
VERSION ?= latest
GOBIN   := $(shell go env GOPATH)/bin

check-fmt: goimports go-licenser
	@${GOBIN}/go-licenser -d -license ${LICENSE}
	@${GOBIN}/goimports -l -e -local github.com/andrewkroh . | read && echo "Code differs from gofmt's style. Run 'gofmt -w .'" 1>&2 && exit 1 || true

docker:
	docker build -t akroh/stream:${VERSION} .

publish: docker
	docker push akroh/stream:${VERSION}

fmt: goimports go-licenser
	${GOBIN}/go-licenser -license ${LICENSE}
	${GOBIN}/goimports -l -w -local github.com/andrewkroh .

goimports:
	GO111MODULE=off go get golang.org/x/tools/cmd/goimports

go-licenser:
	GO111MODULE=off go get github.com/elastic/go-licenser

.PHONY: check-fmt docker fmt goimports go-licenser
