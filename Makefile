LICENSE := ASL2-Short
VERSION ?= local

check-fmt: goimports go-licenser
	@go-licenser -d -license ${LICENSE}
	@goimports -l -e -local github.com/elastic . | read && echo "Code differs from gofmt's style. Run 'gofmt -w .'" 1>&2 && exit 1 || true

docker:
	docker build -t docker.elastic.co/observability/stream:${VERSION} .

fmt: goimports go-licenser
	go-licenser -license ${LICENSE}
	goimports -l -w -local github.com/elastic .

goimports:
	GO111MODULE=off go get golang.org/x/tools/cmd/goimports

go-licenser:
	GO111MODULE=off go get github.com/elastic/go-licenser

.PHONY: check-fmt docker fmt goimports go-licenser
