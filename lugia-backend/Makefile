install:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.1.6

dev:
	$(HOME)/go/bin/air

test-unit:
	go test `go list ./... | grep -v test` -json -v 2>&1 | gotestfmt -hide=empty-packages,successful-tests

test-integration:
	cd test && bash test.sh

test-integration-single:
	@if [ -z "$(TEST)" ]; then \
		echo "Usage: make test-integration-single TEST=TestCreateRole_Integration"; \
		exit 1; \
	fi
	cd test && docker compose -f docker-compose.integration.yml down
	cd test && docker compose -f docker-compose.integration.yml build lugia-backend
	cd test && docker compose -f docker-compose.integration.yml up -d
	cd test && docker compose -f docker-compose.integration.yml exec lugia-backend go test ./test/integration/... -run "$(TEST)" -json -v -p 1 -parallel 1 2>&1 | gotestfmt -hide=empty-packages

lint:
	golangci-lint run --timeout 5m

build:
	go build -o tmp/main main.go

sqlc:
	sqlc generate

deadcode:
	@output=$$(deadcode `go list ./... | grep -v test`); \
	if [ -n "$$output" ]; then \
		echo "Dead code found:"; \
		echo "$$output"; \
		exit 1; \
	else \
		echo "No dead code found"; \
	fi