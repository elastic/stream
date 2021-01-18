fmt:
	GO111MODULE=off go get golang.org/x/tools/cmd/goimports
	goimports -l -w -local github.com/andrewkroh .

.PHONY: fmt
