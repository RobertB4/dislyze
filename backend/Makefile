dev:
	$(HOME)/go/bin/air

test-unit:
	go test `go list ./... | grep -v test`

test-integration:
	cd test && sh test.sh

lint:
	golangci-lint run --timeout 5m

build:
	go build -o main ./main.go

sqlc:
	sqlc generate
