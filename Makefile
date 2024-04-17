TOOL_BIN = bin/gotools/$(shell uname -s)-$(shell uname -m)

build: build-go

build-go:
	go build -tags osusergo,netgo -o viam-revolution-pi -v .

tool-install:
	GOBIN=`pwd`/$(TOOL_BIN) go install \
		github.com/edaniels/golinters/cmd/combined \
		github.com/golangci/golangci-lint/cmd/golangci-lint \
		github.com/AlekSi/gocov-xml \
		github.com/axw/gocov/gocov \
		gotest.tools/gotestsum \
		github.com/rhysd/actionlint/cmd/actionlint

lint: lint-go
	PATH=$(TOOL_BIN) actionlint

lint-go: tool-install
	go mod tidy
	export pkgs="`go list -f '{{.Dir}}' ./... | grep -v /proto/`" && echo "$$pkgs" | xargs go vet -vettool=$(TOOL_BIN)/combined
	GOGC=50 $(TOOL_BIN)/golangci-lint run -v --fix --config=./etc/golangci.yaml

test: test-go

test-go: tool-install
	go test -race ./...

clean-all:
	git clean -fxd
