install:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.1.6

dev:
	$(HOME)/go/bin/air

test-unit:
	go test `go list ./... | grep -v test` -json -v 2>&1 | gotestfmt -hide=empty-packages

test-integration:
	cd test && bash test.sh

lint:
	golangci-lint run --timeout 5m

build:
	go build -o tmp/main main.go

sqlc:
	sqlc generate
