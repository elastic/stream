LICENSE := ASL2-Short
VERSION ?= local

GOIMPORTS := go run golang.org/x/tools/cmd/goimports@latest
GOLICENSER := go run github.com/elastic/go-licenser@latest

check-fmt:
	@${GOLICENSER} -d -license ${LICENSE}
	@${GOIMPORTS} -l -e -local github.com/elastic . | read && echo "Code differs from gofmt's style. Run 'gofmt -w .'" 1>&2 && exit 1 || true

docker:
	docker build -t docker.elastic.co/observability/stream:${VERSION} .

fmt:
	${GOLICENSER} -license ${LICENSE}
	${GOIMPORTS} -l -w -local github.com/elastic .

.PHONY: check-fmt docker fmt
