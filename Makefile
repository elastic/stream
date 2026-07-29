LICENSE := ASL2-Short
VERSION ?= local

GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
GOLICENSER := go run github.com/elastic/go-licenser@v0.4.2

check-fmt:
	@${GOLICENSER} -d -license ${LICENSE}
	@${GOLANGCI_LINT} fmt --diff > /dev/null || (echo "Please run 'make fmt' to fix the formatting issues" 1>&2 && exit 1)

docker:
	docker build -t docker.elastic.co/observability/stream:${VERSION} .

fmt:
	${GOLICENSER} -license ${LICENSE}
	${GOLANGCI_LINT} fmt ./...

lint:
	@${GOLANGCI_LINT} run ./...

.PHONY: check-fmt docker fmt lint
